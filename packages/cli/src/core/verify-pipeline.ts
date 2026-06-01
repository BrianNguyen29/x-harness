import * as path from "node:path";
import fs from "fs-extra";
import { runAdmission, acceptanceStatus } from "./admission.js";
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

export { VerifyInputError } from "./verify-source-loader.js";

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
  const finalOutcome = guardBlocked || traceDirBlocked ? "blocked" : outcome;
  const finalAcceptance = acceptanceStatus(finalOutcome);
  const finalBlockingPredicate =
    guardBlocked || traceDirBlocked
      ? "verifier_not_read_only"
      : blockingPredicate;
  const finalRecoveryRoute =
    guardBlocked || traceDirBlocked
      ? getRecoveryRoute("verifier_not_read_only")
      : (getRecoveryRoute(finalBlockingPredicate) ?? recoveryRoute);

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
