import type { AdmissionInput } from "./admission.js";

export function isString(value: unknown): value is string {
  return typeof value === "string";
}

export function isNonSuccessStatus(value: string): boolean {
  return ["failed", "blocked", "skipped", "timeout", "error"].includes(value);
}

export function getFixStatus(input: AdmissionInput): string | undefined {
  return getClaimFixStatus(input) ?? getSubagentFixStatus(input);
}

export function getClaimFixStatus(input: AdmissionInput): string | undefined {
  if (!input.claim) return undefined;
  const claim = input.claim as Record<string, unknown>;
  return isString(claim.fix_status) ? claim.fix_status : undefined;
}

export function getSubagentFixStatus(
  input: AdmissionInput
): string | undefined {
  if (!input.subagentReturn) return undefined;
  const result = input.subagentReturn.result as
    | Record<string, unknown>
    | undefined;
  if (result && isString(result.fix_status)) return result.fix_status;
  return undefined;
}

export function getVerifyStatus(input: AdmissionInput): string | undefined {
  if (input.verification) {
    const v = input.verification as Record<string, unknown>;
    if (isString(v.status)) return v.status;
  }
  if (input.subagentReturn) {
    const v = input.subagentReturn.verification as
      | Record<string, unknown>
      | undefined;
    if (v && isString(v.status)) return v.status;
  }
  return undefined;
}

export function getAdmissionOutcome(input: AdmissionInput): string | undefined {
  if (input.admission) {
    const a = input.admission as Record<string, unknown>;
    if (isString(a.outcome)) return a.outcome;
  }
  return undefined;
}

export function getEvidenceArray(input: AdmissionInput): unknown[] | undefined {
  if (input.claim) {
    const claim = input.claim as Record<string, unknown>;
    if (Array.isArray(claim.evidence)) return claim.evidence;
  }
  return undefined;
}

export function getHandoff(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  if (input.handoff) return input.handoff as Record<string, unknown>;
  if (input.subagentReturn) {
    const handoff = input.subagentReturn.handoff as
      | Record<string, unknown>
      | undefined;
    if (handoff) return handoff;
  }
  return undefined;
}

export function getEvidenceRecord(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  if (input.evidence) return input.evidence;
  if (input.subagentReturn) {
    const evidence = input.subagentReturn.evidence as
      | Record<string, unknown>
      | undefined;
    if (evidence) return evidence;
  }
  return undefined;
}

export function isCompletionCardShape(input: AdmissionInput): boolean {
  return (
    input.schema_version !== undefined ||
    input.task_id !== undefined ||
    input.verification !== undefined ||
    input.admission !== undefined ||
    input.acceptance_status !== undefined ||
    input.handoff !== undefined ||
    (input.claim !== undefined &&
      "fix_status" in (input.claim as Record<string, unknown>) &&
      "summary" in (input.claim as Record<string, unknown>))
  );
}

export function getVerificationArtifacts(
  input: AdmissionInput
): unknown[] | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    Array.isArray((evidence as Record<string, unknown>).verification_artifacts)
  ) {
    return (evidence as Record<string, unknown>)
      .verification_artifacts as unknown[];
  }
  return undefined;
}

export function getUntestedRegions(
  input: AdmissionInput
): unknown[] | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    Array.isArray((evidence as Record<string, unknown>).untested_regions)
  ) {
    return (evidence as Record<string, unknown>).untested_regions as unknown[];
  }
  return undefined;
}

export function getRemainingRisks(
  input: AdmissionInput
): unknown[] | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    Array.isArray((evidence as Record<string, unknown>).remaining_risks)
  ) {
    return (evidence as Record<string, unknown>).remaining_risks as unknown[];
  }
  return undefined;
}

export function getFilesChanged(input: AdmissionInput): unknown[] | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    Array.isArray((evidence as Record<string, unknown>).files_changed)
  ) {
    return (evidence as Record<string, unknown>).files_changed as unknown[];
  }
  return undefined;
}

export function getCommandEvidence(
  input: AdmissionInput
): unknown[] | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    Array.isArray((evidence as Record<string, unknown>).command_evidence)
  ) {
    return (evidence as Record<string, unknown>).command_evidence as unknown[];
  }
  return undefined;
}

export function getManualRationale(input: AdmissionInput): string | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    typeof (evidence as Record<string, unknown>).manual_rationale === "string"
  ) {
    return (evidence as Record<string, unknown>).manual_rationale as string;
  }
  return undefined;
}

export function getRollbackPolicy(
  input: AdmissionInput
): unknown[] | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    Array.isArray((evidence as Record<string, unknown>).rollback_policy)
  ) {
    return (evidence as Record<string, unknown>).rollback_policy as unknown[];
  }
  return undefined;
}

export function getExecutionControls(
  input: AdmissionInput
): unknown[] | undefined {
  const evidence = getEvidenceRecord(input);
  if (
    evidence &&
    Array.isArray((evidence as Record<string, unknown>).execution_controls)
  ) {
    return (evidence as Record<string, unknown>)
      .execution_controls as unknown[];
  }
  return undefined;
}

export function getState(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  return input.state;
}

export function getGovernance(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  return input.governance;
}

export function getIntake(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  return input.intake;
}

export function getDoneChecklist(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  return input.done_checklist;
}

export function getPrediction(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  return input.prediction;
}

export function getPgvAdvice(
  input: AdmissionInput
): Record<string, unknown> | undefined {
  return input.pgv_advice;
}
