import * as path from "node:path";
import fs from "fs-extra";
import {
  runAdmission,
  acceptanceStatus,
  hasAnyDecisionRef,
  hasAnyIntentRef,
} from "./admission.js";
import { appendTrace } from "./trace.js";
import type { TraceEvent } from "./trace.js";
import { suggestRecovery, getRecoveryRoute } from "./recovery.js";
import type { RecoveryRoute } from "./recovery.js";
import {
  getRepoRoot,
  isMutationGuardAllowlistedPath,
  runMutationGuard,
} from "./mutation-guard.js";
import type { ChangedFilesResolution } from "./changed-files.js";
import type { GuardResult } from "./mutation-guard.js";
import type { EpisodeCreateResult } from "./episode.js";
import { loadVerifySources } from "./verify-source-loader.js";
import { buildAdmissionInput } from "./verify-admission-input.js";
import { runVerifyGovernance } from "./verify-governance.js";
import { evaluateApprovalRisk } from "./approval-risk.js";
import { isValidDecisionEnforce } from "./decision.js";

export { VerifyInputError } from "./verify-source-loader.js";

// buildApprovalRiskAdvisoryNote evaluates the approval-risk engine against a
// completion card and produces the canonical advisory note string. Strictly
// read-only; never alters ok, admission.outcome, acceptance_status, errors,
// blocking_predicate, or admission_authority. Returns null when the policy is
// disabled, the card cannot be evaluated, or the engine is unavailable. Note
// wording is parity-safe with the Go implementation in
// internal/cli/verify.go.
async function buildApprovalRiskAdvisoryNote(
  root: string,
  cardPath: string
): Promise<string | null> {
  if (cardPath === "") return null;
  let report;
  try {
    report = await evaluateApprovalRisk({ root, cardPath });
  } catch {
    // Advisory-only: skip silently on evaluation errors so the verify
    // pipeline never fails because approval-risk is unavailable.
    return null;
  }
  if (!report.policy_enabled) return null;
  return `approval-risk advisory: score=${report.score} risk_class=${report.risk_class} signals=[${report.signals.join(",")}] required_approvals=${report.required_approvals}`;
}

export interface VerifyPipelineOptions {
  card?: string;
  claim?: string;
  evidence?: string;
  subagentReturn?: string;
  tier?: string;
  taskId?: string;
  storyId?: string;
  trace?: boolean;
  traceDir?: string;
  mutationGuard?: boolean;
  contextFloor?: boolean;
  strict?: boolean;
  governanceEnforced?: boolean;
  diff?: string;
  changedFilesSource?: string;
  staleGround?: boolean;
  // Profile-keyed default for `xh verify`. Mirrors the Go canonical
  // VerifyProfile.DecisionEnforce semantics: an explicit value
  // (`off`|`advisory`|`block`) always wins; an empty string means
  // "use the profile default". Profile defaults:
  //   light-local  -> advisory
  //   ci-standard  -> advisory
  //   ci-strict    -> block
  //   governed-deep -> block
  profile?: string;
  decisionEnforce?: string;
  // Profile-keyed default for the verify-layer intent_ref gate. Mirrors
  // the Go canonical VerifyProfile.IntentEnforce semantics: an explicit
  // value (`off`|`advisory`|`block`) always wins; an empty string means
  // "use the profile default". Conservative profile defaults:
  //   light-local   -> advisory
  //   ci-standard   -> advisory
  //   ci-strict     -> advisory (not block)
  //   governed-deep -> block
  intentEnforce?: string;
}

export interface VerifyCheck {
  name: string;
  status: string;
  severity: string;
  note?: string;
}

export interface VerifyEvent extends TraceEvent {
  event_id: string;
  event_type: "verify_completed";
  task_id: string;
  story_id: string | null;
  tier: unknown;
  claim_id: string | null;
  evidence_id: string | null;
  verifier: "x-harness";
  verifier_mode: "read_only";
  outcome: string;
  acceptance_status: string;
  blocking_predicate: string | null;
  blocked_reason_class: string | null;
  next_owner: string | null;
  next_action: string | null;
  created_at: string;
  notes: string[];
  errors: string[];
}

export interface VerifyPipelineResult {
  errors: string[];
  notes: string[];
  checks: VerifyCheck[];
  event: VerifyEvent;
  finalOutcome: string;
  finalAcceptance: string;
  finalRecoveryRoute: RecoveryRoute | null;
  finalBlockingPredicate: string | null;
  cardId: string | null;
  taskId: string;
  cardPath?: string;
  card?: Record<string, unknown>;
  claim?: Record<string, unknown>;
  evidence?: Record<string, unknown>;
  inputCardHash: string | null;
  policyHash: string | null;
  tier: string;
  accepted: boolean;
  strict: boolean;
  changedFiles?: ChangedFilesResolution;
  mutationGuardResult: GuardResult;
  episode?: EpisodeCreateResult;
}

async function injectTestMutation(cwd: string): Promise<void> {
  if (
    process.env.X_HARNESS_ENABLE_TEST_HOOKS !== "1" ||
    !process.env.X_HARNESS_TEST_INJECT_MUTATION
  ) {
    return;
  }

  const injectPath = path.resolve(
    cwd,
    process.env.X_HARNESS_TEST_INJECT_MUTATION
  );
  if (injectPath.startsWith(cwd + path.sep)) {
    fs.writeFileSync(injectPath, "test-mutation");
  } else {
    console.error(
      `test hook: rejected injection path ${injectPath} outside cwd (${cwd})`
    );
  }
}

async function getTraceDirGuardViolation(
  cwd: string,
  traceDir?: string
): Promise<string | null> {
  const repoRoot = (await getRepoRoot(cwd)) ?? path.resolve(cwd);

  const requestedDir = path.resolve(cwd, traceDir ?? ".x-harness/traces");
  const repoPrefix = repoRoot.endsWith(path.sep)
    ? repoRoot
    : `${repoRoot}${path.sep}`;
  if (requestedDir !== repoRoot && !requestedDir.startsWith(repoPrefix)) {
    return `mutation guard blocked: trace directory is outside repository: ${requestedDir}`;
  }

  const traceFile = path.relative(
    repoRoot,
    path.join(requestedDir, "events.jsonl")
  );
  if (!isMutationGuardAllowlistedPath(traceFile)) {
    return `mutation guard blocked: trace directory is not allowlisted: ${traceFile}`;
  }
  return null;
}

function buildChecks(notes: string[], errors: string[]): VerifyCheck[] {
  const checks: VerifyCheck[] = [];
  for (const note of notes) {
    const isWarning = note.includes("recommends") || note.includes("warning");
    checks.push({
      name: note.split(":")[0] || "note",
      status: isWarning ? "warning" : "passed",
      severity: isWarning ? "warning" : "info",
      note,
    });
  }
  for (const err of errors) {
    checks.push({
      name: err.split(":")[0] || "error",
      status: "failed",
      severity: "error",
      note: err,
    });
  }
  return checks;
}

export async function runVerifyPipeline(
  opts: VerifyPipelineOptions,
  cwd = process.cwd()
): Promise<VerifyPipelineResult> {
  const strict = opts.strict === true;

  const loaded = await loadVerifySources(cwd, opts);
  const tier =
    String(loaded.card?.tier ?? opts.tier ?? "standard") || "standard";
  const guardEnabled =
    strict ||
    (opts.mutationGuard ?? false) ||
    tier === "standard" ||
    tier === "deep";
  const guard = await runMutationGuard(guardEnabled, cwd);
  await guard.takeSnapshot();
  await injectTestMutation(cwd);
  const errors = [...loaded.errors];
  const notes = [...loaded.notes];
  const traceDirGuardViolation =
    guardEnabled && opts.trace
      ? await getTraceDirGuardViolation(cwd, opts.traceDir)
      : null;
  if (traceDirGuardViolation) {
    errors.push(traceDirGuardViolation);
  }
  if (strict) {
    notes.push("strict mode enabled");
  }

  const admissionInput = buildAdmissionInput(loaded, opts);
  const admission = runAdmission(admissionInput);
  errors.push(...admission.errors);
  notes.push(...admission.notes);

  const governance = await runVerifyGovernance({
    card: loaded.card,
    cwd,
    opts: {
      strict,
      governanceEnforced: opts.governanceEnforced,
      diff: opts.diff,
      changedFilesSource: opts.changedFilesSource,
    },
  });
  const changedFiles: ChangedFilesResolution | undefined =
    governance.changedFiles;
  errors.push(...governance.errors);
  notes.push(...governance.notes);

  // Advisory approval-risk note. Emitted only when the policy is enabled
  // and the engine evaluates successfully. Never alters ok,
  // admission.outcome, acceptance_status, errors, blocking_predicate, or
  // admission_authority. Skipped silently on evaluation errors.
  if (loaded.cardPath) {
    const arNote = await buildApprovalRiskAdvisoryNote(cwd, loaded.cardPath);
    if (arNote) {
      notes.push(arNote);
    }
  }

  const outcome =
    admission.outcome !== "success"
      ? admission.outcome
      : errors.length > 0
        ? "failed"
        : admission.outcome;

  const recovery = suggestRecovery(errors, outcome);
  const blockingPredicate =
    admission.blocking_predicate ??
    recovery.predicate ??
    (outcome === "blocked" || outcome === "failed" ? "admission_failed" : null);
  const recoveryRoute = recovery.route;

  const guardResult = await guard.evaluate();
  if (guardResult.skippedReason) {
    notes.push(`mutation guard skipped: ${guardResult.skippedReason}`);
    if (guardResult.enabled) {
      errors.push(
        `mutation guard blocked: ${guardResult.skippedReason}; read-only verification cannot be proven`
      );
    }
  } else if (guardResult.enabled && !guardResult.violated) {
    notes.push("mutation guard passed");
  }
  if (guardResult.violated && guardResult.unexpectedDeltas) {
    const paths = guardResult.unexpectedDeltas.map((d) => d.path);
    errors.push(
      `mutation guard blocked: unexpected changes detected: ${paths.join(", ")}`
    );
  }

  const guardBlocked =
    (guardResult.violated && (guardResult.unexpectedDeltas?.length ?? 0) > 0) ||
    (guardResult.enabled && Boolean(guardResult.skippedReason));
  const traceDirBlocked = traceDirGuardViolation !== null;
  // Decision-refs gate (profile-controlled). Mirrors the Go canonical
  // VerifyProfile.DecisionEnforce: an explicit value (off|advisory|block)
  // always wins; an empty/unset value means "use the profile default".
  // The off and advisory modes never block at the verify layer; the
  // advisory note from admission.Run is preserved as-is. The block mode
  // withholds standard/deep cards whose
  // context_alignment.decision_refs array is missing or contains no
  // non-blank string entries, with the explicit
  // `decision_refs_missing` blocking predicate so the failure remains
  // traceable. Light tier is always non-blocking, regardless of mode.
  const decisionEnforceMode = resolveDecisionEnforceMode(
    opts.profile,
    opts.decisionEnforce
  );
  const decisionEnforceBlockReason = applyDecisionEnforceGate({
    mode: decisionEnforceMode,
    tier,
    doc: loaded.card ?? null,
  });
  // Intent-ref gate (profile-controlled). Mirrors the decision-refs
  // gate but with conservative per-oracle profile defaults: only
  // governed-deep defaults to block; ci-strict, ci-standard, and
  // light-local stay advisory by default. An explicit
  // --intent-enforce block can still block under any profile. The
  // predicate is process/governance-only and never implies product
  // correctness.
  const intentEnforceMode = resolveIntentEnforceMode(
    opts.profile,
    opts.intentEnforce
  );
  const intentEnforceBlockReason = applyIntentEnforceGate({
    mode: intentEnforceMode,
    tier,
    doc: loaded.card ?? null,
  });
  const finalOutcome =
    guardBlocked || traceDirBlocked
      ? "blocked"
      : decisionEnforceBlockReason != null
        ? "blocked"
        : intentEnforceBlockReason != null
          ? "blocked"
          : outcome;
  const finalAcceptance = acceptanceStatus(finalOutcome);
  const finalBlockingPredicate =
    guardBlocked || traceDirBlocked
      ? "verifier_not_read_only"
      : decisionEnforceBlockReason != null
        ? "decision_refs_missing"
        : intentEnforceBlockReason != null
          ? "intent_ref_missing"
          : blockingPredicate;
  const finalRecoveryRoute =
    guardBlocked || traceDirBlocked
      ? getRecoveryRoute("verifier_not_read_only")
      : decisionEnforceBlockReason != null
        ? getRecoveryRoute("decision_refs_missing")
        : intentEnforceBlockReason != null
          ? getRecoveryRoute("intent_ref_missing")
          : (getRecoveryRoute(finalBlockingPredicate) ?? recoveryRoute);
  if (decisionEnforceBlockReason != null) {
    errors.push(decisionEnforceBlockReason);
  }
  if (intentEnforceBlockReason != null) {
    errors.push(intentEnforceBlockReason);
  }

  const cardId = (loaded.card?.id as string | undefined) ?? null;
  const taskId =
    opts.taskId ??
    (loaded.card?.task_id as string | undefined) ??
    "TASK-UNKNOWN";
  const event: VerifyEvent = {
    event_id: `VE-${Date.now()}`,
    event_type: "verify_completed",
    task_id: taskId,
    story_id: opts.storyId ?? null,
    tier,
    claim_id:
      (loaded.claim?.id as string | undefined) ??
      (loaded.card?.task_id as string | undefined) ??
      null,
    evidence_id: (loaded.evidence?.id as string | undefined) ?? null,
    verifier: "x-harness",
    verifier_mode: "read_only",
    outcome: finalOutcome,
    acceptance_status: finalAcceptance,
    blocking_predicate: finalBlockingPredicate,
    blocked_reason_class:
      finalOutcome === "blocked" ? "policy_violation" : null,
    next_owner:
      finalRecoveryRoute?.owner ??
      ((loaded.card?.handoff as Record<string, unknown> | undefined)?.owner as
        | string
        | null) ??
      null,
    next_action:
      finalRecoveryRoute?.next_action ??
      ((loaded.card?.handoff as Record<string, unknown> | undefined)
        ?.next_action as string | null) ??
      (errors.length > 0 ? "resolve validation errors" : null),
    created_at: new Date().toISOString(),
    notes,
    errors,
  };

  if (opts.trace && !traceDirBlocked) {
    await appendTrace(event, opts.traceDir);
  }

  const accepted = finalOutcome === "success" && finalAcceptance === "accepted";
  return {
    errors,
    notes,
    checks: buildChecks(notes, errors),
    event,
    finalOutcome,
    finalAcceptance,
    finalRecoveryRoute,
    finalBlockingPredicate,
    cardId,
    taskId,
    cardPath: loaded.cardPath,
    card: loaded.card,
    claim: loaded.claim,
    evidence: loaded.evidence,
    inputCardHash: loaded.inputCardHash,
    policyHash: loaded.policyHash,
    tier,
    accepted,
    strict,
    changedFiles,
    mutationGuardResult: guardResult,
  };
}

// Decision-enforce profile defaults mirror the Go canonical profiles in
// `internal/cli/verify.go` (VerifyProfile.DecisionEnforce). Empty /
// unset profile falls back to the off-by-default behavior: explicit
// --decision-enforce flag still wins when set.
const DECISION_ENFORCE_PROFILE_DEFAULTS: Record<string, string> = {
  "light-local": "advisory",
  "ci-standard": "advisory",
  "ci-strict": "block",
  "governed-deep": "block",
};

export function resolveDecisionEnforceMode(
  profile: string | undefined,
  explicit: string | undefined
): string {
  const explicitTrimmed = (explicit ?? "").trim();
  if (explicitTrimmed !== "") {
    if (!isValidDecisionEnforce(explicitTrimmed)) {
      return "off";
    }
    return explicitTrimmed;
  }
  const profileTrimmed = (profile ?? "").trim();
  if (profileTrimmed === "") return "off";
  const profileDefault = DECISION_ENFORCE_PROFILE_DEFAULTS[profileTrimmed];
  if (profileDefault === undefined) return "off";
  return profileDefault;
}

// applyDecisionEnforceGate mirrors the verify-layer decision_refs gate
// in `internal/cli/verify.go`. The block mode withholds standard/deep
// cards whose context_alignment.decision_refs is missing or contains no
// non-blank string entries; advisory and off return null (no block).
// The light tier is always non-blocking regardless of mode.
export function applyDecisionEnforceGate(args: {
  mode: string;
  tier: string;
  doc: Record<string, unknown> | null;
}): string | null {
  if (args.mode !== "block") return null;
  if (args.tier !== "standard" && args.tier !== "deep") return null;
  if (hasAnyDecisionRef(args.doc)) return null;
  return "context_alignment.decision_refs is empty (verify-stage block; admission acceptance is not decision correctness)";
}

// isValidIntentEnforce reports whether value is one of the supported
// enforcement modes for the verify-stage intent_ref gate. Mirrors the
// closed-enum style of isValidDecisionEnforce (in
// `packages/cli/src/core/decision.ts`).
export function isValidIntentEnforce(value: string): boolean {
  return value === "off" || value === "advisory" || value === "block";
}

// Intent-enforce profile defaults mirror the Go canonical profiles in
// `internal/cli/verify.go` (VerifyProfile.IntentEnforce). Conservative
// per-oracle defaults: only governed-deep blocks by default; ci-strict,
// ci-standard, and light-local stay advisory. An explicit
// --intent-enforce block can still block under any profile.
const INTENT_ENFORCE_PROFILE_DEFAULTS: Record<string, string> = {
  "light-local": "advisory",
  "ci-standard": "advisory",
  "ci-strict": "advisory",
  "governed-deep": "block",
};

export function resolveIntentEnforceMode(
  profile: string | undefined,
  explicit: string | undefined
): string {
  const explicitTrimmed = (explicit ?? "").trim();
  if (explicitTrimmed !== "") {
    if (!isValidIntentEnforce(explicitTrimmed)) {
      return "off";
    }
    return explicitTrimmed;
  }
  const profileTrimmed = (profile ?? "").trim();
  if (profileTrimmed === "") return "off";
  const profileDefault = INTENT_ENFORCE_PROFILE_DEFAULTS[profileTrimmed];
  if (profileDefault === undefined) return "off";
  return profileDefault;
}

// applyIntentEnforceGate mirrors the verify-layer intent_ref gate in
// `internal/cli/verify.go`. The block mode withholds standard/deep
// cards whose top-level intent_ref is missing or blank; advisory and
// off return null (no block). The light tier is always non-blocking
// regardless of mode. The predicate is process/governance-only and
// never implies product correctness.
export function applyIntentEnforceGate(args: {
  mode: string;
  tier: string;
  doc: Record<string, unknown> | null;
}): string | null {
  if (args.mode !== "block") return null;
  if (args.tier !== "standard" && args.tier !== "deep") return null;
  if (hasAnyIntentRef(args.doc)) return null;
  return "intent_ref not declared (verify-stage block; admission acceptance is not intent correctness)";
}
