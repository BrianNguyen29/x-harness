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
  // New optional fields
  evidence?: Record<string, unknown>;
  state?: Record<string, unknown>;
  governance?: Record<string, unknown>;
  intake?: Record<string, unknown>;
  context_acknowledged?: boolean;
  done_checklist?: Record<string, unknown>;
  prediction?: Record<string, unknown>;
  pgv_advice?: Record<string, unknown>;
  // Flag to indicate card mode (full completion card) vs legacy mode
  // When true, done_checklist and prediction are required for standard/deep tiers
  // When false/undefined, legacy mode is assumed and new requirements are not enforced
  isCardMode?: boolean;
  // Strict verify mode strengthens evidence provenance checks for standard/deep.
  strict?: boolean;
  // Approval receipt for high-risk commands
  approval_receipt?: Record<string, unknown>;
  // Backward compatibility: subagent-return shape
  subagentReturn?: Record<string, unknown>;
}

export interface AdmissionResult {
  outcome: VerifyOutcome;
  acceptance_status: AcceptanceStatus;
  errors: string[];
  notes: string[];
  blocking_predicate?: string | null;
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
