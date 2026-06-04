import * as path from "node:path";
import fs from "node:fs";
import {
  hasApprovedTierDowngradeIntervention,
  isRuntimeTier,
  isTierDowngrade,
} from "./intake.js";
import {
  getAdmissionOutcome,
  getGovernance,
  getIntake,
  getPgvAdvice,
  isCompletionCardShape,
} from "./admission-accessors.js";
import {
  evaluateEvidenceRules,
  evaluateEscalation,
  evaluateOperationEscalation,
  evaluateTierGuard,
} from "./admission-evidence.js";
import { evaluateDoneChecklistAndPrediction } from "./admission-prediction.js";
import { evaluateApprovalReceipt } from "./admission-approval.js";
import {
  collectCanonicalStatusContradictions,
  collectFixStatusContradictions,
} from "./admission-contradictions.js";

export type VerifyOutcome =
  | "success"
  | "failed"
  | "blocked"
  | "skipped"
  | "timeout"
  | "error";
export type AcceptanceStatus = "accepted" | "withheld";
export type Tier = "light" | "standard" | "deep";
export type FixStatus = "fixed" | "not_fixed" | "partial";
export type VerificationStatus =
  | "passed"
  | "failed"
  | "skipped"
  | "blocked"
  | "timeout"
  | "error";

export function acceptanceStatus(outcome: VerifyOutcome): AcceptanceStatus {
  return outcome === "success" ? "accepted" : "withheld";
}

export interface Claim {
  fix_status: FixStatus;
  summary: string;
  evidence: unknown[];
}

export interface Verification {
  status: VerificationStatus;
  checks: unknown[];
}

export interface Admission {
  outcome: VerifyOutcome;
}

export interface Handoff {
  next_action: string;
  owner: string;
}

export interface AdmissionInput {
  schema_version?: string;
  task_id?: string;
  tier?: Tier;
  owner?: string;
  accountable?: string;
  claim?: Claim | Record<string, unknown>;
  verification?: Verification | Record<string, unknown>;
  admission?: Admission | Record<string, unknown>;
  acceptance_status?: AcceptanceStatus;
  handoff?: Handoff | Record<string, unknown>;
  staleGround?: boolean;
  pgv_risk?: "LOW" | "MED" | "HIGH";
  // Path to the completion card on disk. Used by context-floor file
  // existence checks to resolve relative paths authored next to the card.
  cardPath?: string;
  // New optional fields
  evidence?: Record<string, unknown>;
  state?: Record<string, unknown>;
  governance?: Record<string, unknown>;
  intake?: Record<string, unknown>;
  context_acknowledged?: boolean;
  done_checklist?: Record<string, unknown>;
  prediction?: Record<string, unknown>;
  pgv_advice?: Record<string, unknown>;
  context_alignment?: Record<string, unknown>;
  // Flag to indicate card mode (full completion card) vs legacy mode
  // When true, done_checklist and prediction are required for standard/deep tiers
  // When false/undefined, legacy mode is assumed and new requirements are not enforced
  isCardMode?: boolean;
  // Strict verify mode strengthens evidence provenance checks for standard/deep.
  strict?: boolean;
  // Approval receipt for high-risk commands
  approval_receipt?: Record<string, unknown>;
  // Context floor enforcement flag
  contextFloor?: boolean;
  // Backward compatibility: subagent-return shape
  subagentReturn?: Record<string, unknown>;
  // Optional product intent status. Advisory-only; admission acceptance is
  // not product correctness. When present, the engine emits advisory notes
  // for missing or "unknown" status on standard/deep tiers. The light tier
  // remains quiet. aligned/unreviewed/disputed/not_applicable do not block.
  product_intent?: Record<string, unknown>;
  // Optional test adequacy declaration. Advisory-only; admission acceptance
  // is not test adequacy. On standard/deep the engine emits advisory notes
  // for a missing object or for missing/empty impacted_behaviors,
  // tests_selected, why_sufficient. On deep, known_gaps must be present
  // (an explicit [] is accepted and produces no note). The light tier
  // remains quiet. Mirrors the Go implementation in
  // internal/admission/test_adequacy.go and the policy documentation in
  // policies/admission.yaml.
  test_adequacy?: Record<string, unknown>;
  // Optional evidence adequacy declaration. Advisory-only; admission
  // acceptance is not evidence adequacy. On standard/deep the engine emits
  // a top-level missing note when evidence_adequacy is absent and a
  // summary note when summary is missing/blank. The light tier remains
  // quiet. Never blocks admission. Mirrors the Go implementation in
  // internal/admission/evidence_adequacy.go and the policy documentation
  // in policies/admission.yaml.
  evidence_adequacy?: Record<string, unknown>;
  // Optional intent contract declaration. Advisory-only; admission
  // acceptance is not intent correctness. On standard/deep the engine
  // emits a top-level missing note when intent_contract is absent, a
  // product_goal note when product_goal is missing/blank, and a
  // user_visible_change note when the key is absent. An explicit
  // user_visible_change == false is accepted and produces no uvchange
  // note. The light tier remains quiet. Never blocks admission. Mirrors
  // the Go implementation in internal/admission/intent_contract.go and
  // the policy documentation in policies/admission.yaml.
  intent_contract?: Record<string, unknown>;
  // Optional advisory reference to a product intent record (id or path)
  // described by product-intent.schema.json. The engine emits a
  // top-level missing note for standard/deep when intent_ref is absent
  // or blank and stays quiet otherwise. The light tier remains quiet.
  // Never blocks admission. Mirrors the Go implementation in
  // internal/admission/intent_ref.go and the policy documentation in
  // policies/admission.yaml.
  intent_ref?: string;
  // Optional advisory references to decision records (ADR-lite) described
  // by decision-record.schema.json. The references live under
  // context_alignment.decision_refs (a string array) per
  // schemas/context-alignment.schema.json. The engine emits a top-level
  // note for standard/deep when the array is missing or contains no
  // non-blank string entries, and stays quiet otherwise. The light tier
  // remains quiet. Never blocks admission. Mirrors the Go implementation
  // in internal/admission/decision_refs.go and the policy documentation
  // in policies/admission.yaml.
  decision_refs?: string[];
}

export interface AdmissionResult {
  outcome: VerifyOutcome;
  acceptance_status: AcceptanceStatus;
  errors: string[];
  notes: string[];
  blocking_predicate?: string | null;
}

interface ContextFloorResult {
  errors: string[];
  notes: string[];
}

function evaluateContextFloor(input: AdmissionInput): ContextFloorResult {
  const result: ContextFloorResult = { errors: [], notes: [] };
  const tier = input.tier;

  // Context floor is only enforced for standard and deep tiers
  if (tier !== "standard" && tier !== "deep") {
    result.notes.push("context floor advisory only for light tier");
    return result;
  }

  const ctxAlign = input.context_alignment as
    | Record<string, unknown>
    | undefined;
  if (ctxAlign == null) {
    result.errors.push(
      "context_alignment is required for standard/deep tier when context floor is enabled"
    );
    return result;
  }

  if (ctxAlign.stale_ground_checked !== true) {
    result.errors.push(
      "context_alignment.stale_ground_checked must be true for standard/deep tier"
    );
    return result;
  }

  const refArrays = [
    "product_contract_refs",
    "architecture_refs",
    "decision_refs",
    "test_matrix_refs",
  ];
  let hasOneRef = false;
  for (const refKey of refArrays) {
    const refs = ctxAlign[refKey] as unknown[] | undefined;
    if (refs != null && refs.length > 0) {
      hasOneRef = true;
      break;
    }
  }
  if (!hasOneRef) {
    result.errors.push(
      "context_alignment must have at least one non-empty ref array (product_contract_refs, architecture_refs, decision_refs, or test_matrix_refs)"
    );
    return result;
  }

  if (tier === "deep") {
    const contextPackID = ctxAlign.context_pack_id as string | undefined;
    if (contextPackID == null || String(contextPackID).trim() === "") {
      result.errors.push(
        "context_alignment.context_pack_id is required for deep tier"
      );
      return result;
    }

    const unresolvedQuestions = ctxAlign.unresolved_context_questions as
      | unknown[]
      | undefined;
    if (unresolvedQuestions != null && unresolvedQuestions.length > 0) {
      result.errors.push(
        "context_alignment.unresolved_context_questions must be empty for deep tier"
      );
      return result;
    }
  }

  // All referenced files must exist. Resolution mirrors Go: cwd first, then
  // path.dirname(cardPath); absolute paths are used literally.
  const cardDir = input.cardPath ? path.dirname(input.cardPath) : "";
  for (const fileErr of validateContextFiles(ctxAlign, cardDir)) {
    result.errors.push(fileErr);
  }

  return result;
}

// stripAnchor removes any #anchor suffix from a path reference.
function stripAnchor(ref: string): string {
  const idx = ref.indexOf("#");
  return idx >= 0 ? ref.slice(0, idx) : ref;
}

// fileExists resolves a relative path against (1) the current working
// directory and (2) cardDir, in that order. Absolute paths skip both lookups
// and use the literal path. Mirrors `admission.FileExists` in Go.
function fileExists(refPath: string, cardDir: string): boolean {
  if (path.isAbsolute(refPath)) {
    return fs.existsSync(refPath);
  }
  const cwd = process.cwd();
  for (const base of [cwd, cardDir]) {
    if (!base) continue;
    const candidate =
      base === cwd ? path.resolve(cwd, refPath) : path.join(base, refPath);
    if (fs.existsSync(candidate)) return true;
  }
  return false;
}

// validateContextFiles checks that all file references in context_alignment
// exist. Emits "referenced file does not exist: <path>" for ref array entries
// and "context_evidence ref file does not exist: <path>" for context_evidence
// entries, mirroring Go.
function validateContextFiles(
  ctxAlign: Record<string, unknown>,
  cardDir: string
): string[] {
  const errors: string[] = [];

  const refArrays = [
    "product_contract_refs",
    "architecture_refs",
    "decision_refs",
    "test_matrix_refs",
  ];
  for (const refKey of refArrays) {
    const refs = ctxAlign[refKey] as unknown[] | undefined;
    if (!Array.isArray(refs)) continue;
    for (const ref of refs) {
      if (typeof ref !== "string") continue;
      const refPath = stripAnchor(ref);
      if (!fileExists(refPath, cardDir)) {
        errors.push(`referenced file does not exist: ${refPath}`);
      }
    }
  }

  const contextEvidence = ctxAlign.context_evidence as unknown[] | undefined;
  if (Array.isArray(contextEvidence)) {
    for (const evidence of contextEvidence) {
      if (evidence == null || typeof evidence !== "object") continue;
      const evMap = evidence as Record<string, unknown>;
      const ref = evMap.ref;
      if (typeof ref !== "string" || ref === "") continue;
      const refPath = stripAnchor(ref);
      if (!fileExists(refPath, cardDir)) {
        errors.push(`context_evidence ref file does not exist: ${refPath}`);
      }
    }
  }

  return errors;
}

const PRODUCT_INTENT_MISSING_NOTE =
  "product_intent.status not declared (advisory-only; admission acceptance is not product correctness)";
const PRODUCT_INTENT_UNKNOWN_NOTE =
  "product_intent.status is unknown (advisory-only; admission acceptance is not product correctness)";

const TEST_ADEQUACY_MISSING_NOTE =
  "test_adequacy not declared (advisory-only; admission acceptance is not test adequacy)";
const TEST_ADEQUACY_BEHAVIORS_MISSING_NOTE =
  "test_adequacy.impacted_behaviors not declared (advisory-only; consider listing behavior covered by tests)";
const TEST_ADEQUACY_TESTS_MISSING_NOTE =
  "test_adequacy.tests_selected not declared (advisory-only; consider listing selected tests)";
const TEST_ADEQUACY_WHY_MISSING_NOTE =
  "test_adequacy.why_sufficient not declared (advisory-only; consider explaining why tests are sufficient)";
const TEST_ADEQUACY_GAPS_MISSING_NOTE =
  "test_adequacy.known_gaps not declared (advisory-only; deep should list gaps or set [])";

const EVIDENCE_ADEQUACY_MISSING_NOTE =
  "evidence_adequacy not declared (advisory-only; admission acceptance is not evidence adequacy)";
const EVIDENCE_ADEQUACY_SUMMARY_MISSING_NOTE =
  "evidence_adequacy.summary not declared (advisory-only; consider explaining how evidence covers the change)";

const INTENT_CONTRACT_MISSING_NOTE =
  "intent_contract not declared (advisory-only; admission acceptance is not intent correctness)";
const INTENT_CONTRACT_GOAL_MISSING_NOTE =
  "intent_contract.product_goal not declared (advisory-only; consider documenting the intended change goal)";
const INTENT_CONTRACT_UVCHANGE_MISSING_NOTE =
  "intent_contract.user_visible_change not declared (advisory-only; consider declaring whether the change is user-visible)";

const INTENT_REF_MISSING_NOTE =
  "intent_ref not declared (advisory-only; admission acceptance is not intent correctness)";

const DECISION_REFS_EMPTY_NOTE =
  "context_alignment.decision_refs is empty (advisory-only; admission acceptance is not decision correctness)";

// Advisory note constants are exported so that the drift guard in
// packages/cli/tests/admission.test.ts can assert parity between the
// runtime values and the policy documentation in policies/admission.yaml.
// No behavior change: the constants were already module-internal; exporting
// them only adds a public name binding.
export {
  PRODUCT_INTENT_MISSING_NOTE,
  PRODUCT_INTENT_UNKNOWN_NOTE,
  TEST_ADEQUACY_MISSING_NOTE,
  TEST_ADEQUACY_BEHAVIORS_MISSING_NOTE,
  TEST_ADEQUACY_TESTS_MISSING_NOTE,
  TEST_ADEQUACY_WHY_MISSING_NOTE,
  TEST_ADEQUACY_GAPS_MISSING_NOTE,
  EVIDENCE_ADEQUACY_MISSING_NOTE,
  EVIDENCE_ADEQUACY_SUMMARY_MISSING_NOTE,
  INTENT_CONTRACT_MISSING_NOTE,
  INTENT_CONTRACT_GOAL_MISSING_NOTE,
  INTENT_CONTRACT_UVCHANGE_MISSING_NOTE,
  INTENT_REF_MISSING_NOTE,
  DECISION_REFS_EMPTY_NOTE,
};

// evaluateProductIntent emits advisory notes (never errors) for standard and
// deep tier cards when product_intent.status is missing or set to "unknown".
// The light tier remains quiet. aligned/unreviewed/disputed/not_applicable do
// not produce any advisory note. This is the first vertical slice; it never
// blocks admission. Wording is parity-safe with the Go implementation in
// internal/admission/product_intent.go and the policy documentation in
// policies/admission.yaml.
function evaluateProductIntent(input: AdmissionInput): string[] {
  const notes: string[] = [];
  if (input.tier !== "standard" && input.tier !== "deep") {
    return notes;
  }

  const productIntent = input.product_intent;
  if (productIntent == null) {
    notes.push(PRODUCT_INTENT_MISSING_NOTE);
    return notes;
  }

  const statusRaw = productIntent.status;
  const status = typeof statusRaw === "string" ? statusRaw.trim() : "";
  if (status === "") {
    notes.push(PRODUCT_INTENT_MISSING_NOTE);
    return notes;
  }
  if (status === "unknown") {
    notes.push(PRODUCT_INTENT_UNKNOWN_NOTE);
  }
  return notes;
}

// isNonEmptyStringArray reports whether value is a non-empty array whose
// entries are all non-blank strings. Used by evaluateTestAdequacy.
function isNonEmptyStringArray(value: unknown): boolean {
  if (!Array.isArray(value) || value.length === 0) return false;
  for (const entry of value) {
    if (typeof entry !== "string" || entry.trim() === "") return false;
  }
  return true;
}

// evaluateTestAdequacy emits advisory notes (never errors) for standard and
// deep tier cards when test_adequacy or its sub-properties are missing or
// blank. The light tier remains quiet. known_gaps == [] (explicit empty
// array) is accepted for deep without emitting a note. This is the first
// vertical slice; it never blocks admission. Wording is parity-safe with
// the Go implementation in internal/admission/test_adequacy.go and the
// policy documentation in policies/admission.yaml.
function evaluateTestAdequacy(input: AdmissionInput): string[] {
  const notes: string[] = [];
  if (input.tier !== "standard" && input.tier !== "deep") {
    return notes;
  }

  const testAdequacy = input.test_adequacy;
  if (testAdequacy == null) {
    notes.push(TEST_ADEQUACY_MISSING_NOTE);
    return notes;
  }

  if (!isNonEmptyStringArray(testAdequacy.impacted_behaviors)) {
    notes.push(TEST_ADEQUACY_BEHAVIORS_MISSING_NOTE);
  }
  if (!isNonEmptyStringArray(testAdequacy.tests_selected)) {
    notes.push(TEST_ADEQUACY_TESTS_MISSING_NOTE);
  }
  const whyRaw = testAdequacy.why_sufficient;
  const why = typeof whyRaw === "string" ? whyRaw.trim() : "";
  if (why === "") {
    notes.push(TEST_ADEQUACY_WHY_MISSING_NOTE);
  }

  if (input.tier === "deep") {
    // known_gaps must be present; explicit [] is accepted and produces no
    // note. Other shapes (missing key, null, non-array) emit a note.
    if (!Object.prototype.hasOwnProperty.call(testAdequacy, "known_gaps")) {
      notes.push(TEST_ADEQUACY_GAPS_MISSING_NOTE);
    } else if (!isNonEmptyStringArray(testAdequacy.known_gaps)) {
      // Field is present but not a non-empty string array. Accept
      // [] as "explicit empty" (isNonEmptyStringArray returns false for
      // []; detect that case and skip the note).
      const gaps = testAdequacy.known_gaps;
      const isExplicitEmpty = Array.isArray(gaps) && gaps.length === 0;
      if (!isExplicitEmpty) {
        notes.push(TEST_ADEQUACY_GAPS_MISSING_NOTE);
      }
    }
  }

  return notes;
}

// evaluateEvidenceAdequacy emits advisory notes (never errors) for standard
// and deep tier cards when evidence_adequacy is missing or when summary is
// missing/blank. The light tier remains quiet. A non-blank summary
// suppresses the summary note but does not gate the missing-object note.
// This is the first vertical slice; it never blocks admission. Wording is
// parity-safe with the Go implementation in
// internal/admission/evidence_adequacy.go and the policy documentation in
// policies/admission.yaml.
function evaluateEvidenceAdequacy(input: AdmissionInput): string[] {
  const notes: string[] = [];
  if (input.tier !== "standard" && input.tier !== "deep") {
    return notes;
  }

  const evidenceAdequacy = input.evidence_adequacy;
  if (evidenceAdequacy == null) {
    notes.push(EVIDENCE_ADEQUACY_MISSING_NOTE);
    return notes;
  }

  const summaryRaw = evidenceAdequacy.summary;
  const summary = typeof summaryRaw === "string" ? summaryRaw.trim() : "";
  if (summary === "") {
    notes.push(EVIDENCE_ADEQUACY_SUMMARY_MISSING_NOTE);
  }

  return notes;
}

// evaluateIntentContract emits advisory notes (never errors) for standard
// and deep tier cards when intent_contract is missing, when its
// product_goal is missing/blank, or when its user_visible_change key is
// absent. The light tier remains quiet. user_visible_change == false (an
// explicit non-user-visible declaration) is accepted and produces no
// uvchange note. This is the first vertical slice; it never blocks
// admission. Wording is parity-safe with the Go implementation in
// internal/admission/intent_contract.go and the policy documentation in
// policies/admission.yaml.
function evaluateIntentContract(input: AdmissionInput): string[] {
  const notes: string[] = [];
  if (input.tier !== "standard" && input.tier !== "deep") {
    return notes;
  }

  const intentContract = input.intent_contract;
  if (intentContract == null) {
    notes.push(INTENT_CONTRACT_MISSING_NOTE);
    return notes;
  }

  const goalRaw = intentContract.product_goal;
  const goal = typeof goalRaw === "string" ? goalRaw.trim() : "";
  if (goal === "") {
    notes.push(INTENT_CONTRACT_GOAL_MISSING_NOTE);
  }

  if (
    !Object.prototype.hasOwnProperty.call(intentContract, "user_visible_change")
  ) {
    notes.push(INTENT_CONTRACT_UVCHANGE_MISSING_NOTE);
  }

  return notes;
}

// evaluateIntentRef emits advisory notes (never errors) for standard and
// deep tier cards when the optional top-level intent_ref field is missing
// or blank. The light tier remains quiet. A non-blank intent_ref (slug id,
// path, or URI fragment) suppresses the note. This is the first safe-V1
// vertical slice; it never blocks admission. Wording is parity-safe with
// the Go implementation in internal/admission/intent_ref.go and the
// policy documentation in policies/admission.yaml.
function evaluateIntentRef(input: AdmissionInput): string[] {
  const notes: string[] = [];
  if (input.tier !== "standard" && input.tier !== "deep") {
    return notes;
  }

  const refRaw = input.intent_ref;
  const ref = typeof refRaw === "string" ? refRaw.trim() : "";
  if (ref === "") {
    notes.push(INTENT_REF_MISSING_NOTE);
  }
  return notes;
}

// hasAnyDecisionRef reports whether doc carries a context_alignment
// block with a decision_refs array containing at least one non-blank
// string entry. Mirrors the Go helper `internal/admission/decision_refs.go`
// `HasAnyDecisionRef` and is reused by the verify-layer decision_enforce
// block path.
export function hasAnyDecisionRef(
  doc: Record<string, unknown> | null | undefined
): boolean {
  if (doc == null) return false;
  const ctx = doc.context_alignment;
  if (ctx == null || typeof ctx !== "object" || Array.isArray(ctx)) {
    return false;
  }
  const refs = (ctx as Record<string, unknown>).decision_refs;
  if (!Array.isArray(refs)) return false;
  for (const entry of refs) {
    if (typeof entry === "string" && entry.trim() !== "") {
      return true;
    }
  }
  return false;
}

// evaluateDecisionRefs emits advisory notes (never errors) for standard
// and deep tier cards when the optional context_alignment.decision_refs
// array is missing or contains no non-blank string entries. A non-blank
// entry (slug id, path, or URI fragment) suppresses the note. The light
// tier remains quiet. This is the first safe-V1 vertical slice; it never
// blocks admission. Wording is parity-safe with the Go implementation
// in internal/admission/decision_refs.go and the policy documentation
// in policies/admission.yaml.
function evaluateDecisionRefs(input: AdmissionInput): string[] {
  const notes: string[] = [];
  if (input.tier !== "standard" && input.tier !== "deep") {
    return notes;
  }

  const ctxAlign = input.context_alignment as
    | Record<string, unknown>
    | undefined;
  if (ctxAlign == null) {
    notes.push(DECISION_REFS_EMPTY_NOTE);
    return notes;
  }

  const refsRaw = ctxAlign.decision_refs;
  if (!Array.isArray(refsRaw)) {
    notes.push(DECISION_REFS_EMPTY_NOTE);
    return notes;
  }

  for (const entry of refsRaw) {
    if (typeof entry === "string" && entry.trim() !== "") {
      return notes;
    }
  }
  notes.push(DECISION_REFS_EMPTY_NOTE);
  return notes;
}

export function runAdmission(input: AdmissionInput): AdmissionResult {
  const errors: string[] = [];
  const notes: string[] = [];
  let blockingPredicate: string | null = null;
  const completionCardShape = isCompletionCardShape(input);

  // Stale-ground fail-closed
  if (input.staleGround) {
    errors.push(
      "stale_ground detected: withholding pending refresh or ruling out"
    );
    return {
      outcome: "blocked",
      acceptance_status: "withheld",
      errors,
      notes: ["stale-ground policy: if_detected = withhold"],
      blocking_predicate: "stale_ground",
    };
  }

  // Required fields check (only for completion card shape)
  if (completionCardShape) {
    if (!input.owner || input.owner.trim().length === 0) {
      errors.push("missing owner: owner is required");
    }
    if (!input.accountable || input.accountable.trim().length === 0) {
      errors.push("missing accountable: accountable is required");
    }
    if (!input.task_id || input.task_id.trim().length === 0) {
      errors.push("missing task_id: task_id is required");
    }
  }

  // Tier validation
  if (input.tier && !["light", "standard", "deep"].includes(input.tier)) {
    errors.push(
      `invalid tier: "${input.tier}" must be one of light, standard, deep`
    );
  }

  const admissionOutcome = getAdmissionOutcome(input);
  const governance = getGovernance(input);
  const intake = getIntake(input);
  const pgvAdvice = getPgvAdvice(input);

  const applyFinding = (item: {
    message: string;
    predicate?: string;
    forcePredicate?: boolean;
  }): void => {
    errors.push(item.message);
    if (item.forcePredicate || (!blockingPredicate && item.predicate)) {
      blockingPredicate = item.predicate ?? null;
    }
  };

  for (const item of collectFixStatusContradictions(input)) {
    applyFinding(item);
  }

  if (intake) {
    const mappedTier = intake.mapped_tier;
    if (isRuntimeTier(input.tier) && isRuntimeTier(mappedTier)) {
      if (isTierDowngrade(input.tier, mappedTier)) {
        if (!hasApprovedTierDowngradeIntervention(governance)) {
          errors.push(
            `intake tier downgrade requires governance intervention approval: declared ${input.tier}, mapped ${mappedTier}`
          );
          if (!blockingPredicate) blockingPredicate = "Fintervention";
        } else {
          notes.push(
            `intake tier downgrade approved by governance intervention: declared ${input.tier}, mapped ${mappedTier}`
          );
        }
      }
    } else if (mappedTier !== undefined && !isRuntimeTier(mappedTier)) {
      errors.push(
        `intake.mapped_tier "${String(mappedTier)}" must be one of light, standard, deep`
      );
      if (!blockingPredicate) blockingPredicate = "admission_failed";
    }
  }

  const escalationResult = evaluateEscalation(input);
  notes.push(...escalationResult.notes);
  for (const item of escalationResult.errors) {
    applyFinding(item);
  }

  const operationEscalationResult = evaluateOperationEscalation(input);
  notes.push(...operationEscalationResult.notes);
  for (const item of operationEscalationResult.errors) {
    applyFinding(item);
  }

  const tierGuardResult = evaluateTierGuard(input);
  notes.push(...tierGuardResult.notes);
  for (const item of tierGuardResult.errors) {
    applyFinding(item);
  }

  const evidenceResult = evaluateEvidenceRules(input);
  notes.push(...evidenceResult.notes);
  for (const item of evidenceResult.errors) {
    applyFinding(item);
  }

  const approvalResult = evaluateApprovalReceipt(input, input.tier);
  notes.push(...approvalResult.notes);
  for (const item of approvalResult.errors) {
    applyFinding(item);
  }

  for (const item of evaluateDoneChecklistAndPrediction(input).errors) {
    applyFinding(item);
  }

  // Context floor (default for standard/deep; opt-in for light)
  if (input.contextFloor) {
    const cfResult = evaluateContextFloor(input);
    notes.push(...cfResult.notes);
    for (const errMsg of cfResult.errors) {
      applyFinding({ message: errMsg, predicate: "context_floor_blocked" });
    }
  }

  // Product intent advisory (optional; never blocks admission)
  notes.push(...evaluateProductIntent(input));

  // Test adequacy advisory (optional; never blocks admission). Mirrors
  // product_intent: emits notes for missing or incomplete test_adequacy
  // on standard/deep tiers; light tier stays quiet.
  notes.push(...evaluateTestAdequacy(input));

  // Evidence adequacy advisory (optional; never blocks admission).
  // Mirrors test_adequacy: emits a top-level missing note when
  // evidence_adequacy is absent on standard/deep tiers and a summary
  // note when summary is missing/blank. Light tier stays quiet.
  notes.push(...evaluateEvidenceAdequacy(input));

  // Intent contract advisory (optional; never blocks admission). Mirrors
  // evidence_adequacy: emits a top-level missing note when intent_contract
  // is absent on standard/deep tiers, a product_goal note when
  // product_goal is missing/blank, and a user_visible_change note when
  // the key is absent. An explicit user_visible_change == false is
  // accepted and produces no uvchange note. Light tier stays quiet.
  notes.push(...evaluateIntentContract(input));

  // Intent ref advisory (optional; never blocks admission). Mirrors
  // intent_contract: emits a top-level missing note when the optional
  // top-level intent_ref field is absent or blank on standard/deep
  // tiers. Light tier stays quiet. Safe V1 wording is parity-safe with
  // the Go implementation in internal/admission/intent_ref.go and the
  // policy documentation in policies/admission.yaml.
  notes.push(...evaluateIntentRef(input));

  // Decision refs advisory (optional; never blocks admission). Mirrors
  // intent_ref: emits a top-level note on standard/deep when the
  // optional context_alignment.decision_refs array is missing or
  // contains no non-blank string entries. Light tier stays quiet. Safe
  // V1 wording is parity-safe with the Go implementation in
  // internal/admission/decision_refs.go and the policy documentation
  // in policies/admission.yaml.
  notes.push(...evaluateDecisionRefs(input));

  // Governance / human approval check for deep
  if (input.tier === "deep" && governance) {
    if (governance.requires_human_approval === true) {
      const approvalStatus = governance.approval_status as string | undefined;
      if (approvalStatus !== "approved") {
        errors.push("deep task requires human approval before admission");
        if (!blockingPredicate) blockingPredicate = "approval_missing";
      }
    }
  }

  for (const item of collectCanonicalStatusContradictions(input)) {
    applyFinding(item);
  }

  // context_acknowledged is advisory-only; never blocks admission
  if (isCompletionCardShape(input) && input.context_acknowledged !== true) {
    notes.push(
      "context_acknowledged is missing or false (advisory-only; does not block admission)"
    );
  }

  // PGV is advisory-only; never blocks by default
  if (input.pgv_risk) {
    notes.push(`PGV risk level: ${input.pgv_risk} (advisory-only)`);
  }

  if (pgvAdvice?.admission_authority === true) {
    errors.push(
      "pgv_advice cannot grant admission authority; PGV is advisory-only"
    );
    if (!blockingPredicate) blockingPredicate = "admission_failed";
  }

  if (errors.length > 0) {
    return {
      outcome: "failed",
      acceptance_status: "withheld",
      errors,
      notes,
      blocking_predicate: blockingPredicate ?? "admission_failed",
    };
  }

  // Respect input admission outcome if present and valid
  const finalOutcome: VerifyOutcome =
    admissionOutcome &&
    ["success", "failed", "blocked", "skipped", "timeout", "error"].includes(
      admissionOutcome
    )
      ? (admissionOutcome as VerifyOutcome)
      : "success";

  return {
    outcome: finalOutcome,
    acceptance_status: acceptanceStatus(finalOutcome),
    errors: [],
    notes:
      notes.length > 0
        ? [...notes, "admission checks passed"]
        : ["admission checks passed"],
    blocking_predicate: null,
  };
}
