export type VerifyOutcome = "success" | "failed" | "blocked" | "skipped" | "timeout" | "error";
export type AcceptanceStatus = "accepted" | "withheld";
export type Tier = "light" | "standard" | "deep";
export type FixStatus = "fixed" | "not_fixed" | "partial";
export type VerificationStatus = "passed" | "failed" | "skipped" | "blocked";

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

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function getFixStatus(input: AdmissionInput): string | undefined {
  if (input.claim) {
    const claim = input.claim as Record<string, unknown>;
    if (isString(claim.fix_status)) return claim.fix_status;
  }
  if (input.subagentReturn) {
    const result = input.subagentReturn.result as Record<string, unknown> | undefined;
    if (result && isString(result.fix_status)) return result.fix_status;
  }
  return undefined;
}

function getVerifyStatus(input: AdmissionInput): string | undefined {
  if (input.verification) {
    const v = input.verification as Record<string, unknown>;
    if (isString(v.status)) return v.status;
  }
  if (input.subagentReturn) {
    const v = input.subagentReturn.verification as Record<string, unknown> | undefined;
    if (v && isString(v.status)) return v.status;
  }
  return undefined;
}

function getAdmissionOutcome(input: AdmissionInput): string | undefined {
  if (input.admission) {
    const a = input.admission as Record<string, unknown>;
    if (isString(a.outcome)) return a.outcome;
  }
  return undefined;
}

function getEvidenceArray(input: AdmissionInput): unknown[] | undefined {
  if (input.claim) {
    const claim = input.claim as Record<string, unknown>;
    if (Array.isArray(claim.evidence)) return claim.evidence;
  }
  return undefined;
}

function getHandoff(input: AdmissionInput): Record<string, unknown> | undefined {
  if (input.handoff) return input.handoff as Record<string, unknown>;
  return undefined;
}

function isCompletionCardShape(input: AdmissionInput): boolean {
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

function getVerificationArtifacts(input: AdmissionInput): unknown[] | undefined {
  if (input.evidence && Array.isArray((input.evidence as Record<string, unknown>).verification_artifacts)) {
    return (input.evidence as Record<string, unknown>).verification_artifacts as unknown[];
  }
  return undefined;
}

function getUntestedRegions(input: AdmissionInput): unknown[] | undefined {
  if (input.evidence && Array.isArray((input.evidence as Record<string, unknown>).untested_regions)) {
    return (input.evidence as Record<string, unknown>).untested_regions as unknown[];
  }
  return undefined;
}

function getRemainingRisks(input: AdmissionInput): unknown[] | undefined {
  if (input.evidence && Array.isArray((input.evidence as Record<string, unknown>).remaining_risks)) {
    return (input.evidence as Record<string, unknown>).remaining_risks as unknown[];
  }
  return undefined;
}

function getState(input: AdmissionInput): Record<string, unknown> | undefined {
  return input.state;
}

function getGovernance(input: AdmissionInput): Record<string, unknown> | undefined {
  return input.governance;
}

function hasScopeDeclared(artifacts: unknown[] | undefined): boolean {
  if (!artifacts || artifacts.length === 0) return false;
  for (const artifact of artifacts) {
    const a = artifact as Record<string, unknown> | undefined;
    if (!a) continue;
    const verifies = a.verifies as unknown[] | undefined;
    const doesNotVerify = a.does_not_verify as unknown[] | undefined;
    if (verifies && verifies.length > 0) return true;
    if (doesNotVerify && doesNotVerify.length > 0) return true;
  }
  return false;
}

export function runAdmission(input: AdmissionInput): AdmissionResult {
  const errors: string[] = [];
  const notes: string[] = [];
  let blockingPredicate: string | null = null;

  // Stale-ground fail-closed
  if (input.staleGround) {
    errors.push("stale_ground detected: withholding pending refresh or ruling out");
    return {
      outcome: "blocked",
      acceptance_status: "withheld",
      errors,
      notes: ["stale-ground policy: if_detected = withhold"],
      blocking_predicate: "stale_ground",
    };
  }

  // Required fields check (only for completion card shape)
  if (isCompletionCardShape(input)) {
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
    errors.push(`invalid tier: "${input.tier}" must be one of light, standard, deep`);
  }

  const fixStatus = getFixStatus(input);
  const verifyStatus = getVerifyStatus(input);
  const admissionOutcome = getAdmissionOutcome(input);
  const evidenceArray = getEvidenceArray(input);
  const handoff = getHandoff(input);
  const verificationArtifacts = getVerificationArtifacts(input);
  const untestedRegions = getUntestedRegions(input);
  const remainingRisks = getRemainingRisks(input);
  const state = getState(input);
  const governance = getGovernance(input);

  // Evidence presence check
  if (evidenceArray !== undefined) {
    if (evidenceArray.length === 0) {
      if (input.tier && input.tier !== "light") {
        errors.push(`tier "${input.tier}" requires evidence packet`);
      } else {
        notes.push("claim.evidence is empty");
      }
    }
  } else {
    // Backward compatibility: check old evidence shape
    if (input.evidence) {
      const ev = input.evidence;
      if (!ev.owner && !ev.accountable) {
        notes.push("evidence packet lacks owner/accountable fields");
      }
    }
    if (input.tier && input.tier !== "light") {
      if (!input.evidence) {
        errors.push(`tier "${input.tier}" requires evidence packet`);
      }
    }
  }

  // Tier evidence floor
  if (input.tier === "deep") {
    if (!verificationArtifacts || verificationArtifacts.length === 0) {
      errors.push('tier "deep" requires verification_artifacts');
      if (!blockingPredicate) blockingPredicate = "evidence_scope_missing";
    }
    if (!hasScopeDeclared(verificationArtifacts)) {
      errors.push('tier "deep" requires evidence scope declared (verifies/does_not_verify)');
      if (!blockingPredicate) blockingPredicate = "evidence_scope_missing";
    }
    if (!untestedRegions || untestedRegions.length === 0) {
      errors.push('tier "deep" requires untested_regions');
      if (!blockingPredicate) blockingPredicate = "evidence_scope_missing";
    }
    if (!remainingRisks || remainingRisks.length === 0) {
      errors.push('tier "deep" requires remaining_risks');
      if (!blockingPredicate) blockingPredicate = "evidence_scope_missing";
    }
    if (!state || !Array.isArray(state.write_set) || (state.write_set as unknown[]).length === 0) {
      errors.push('tier "deep" requires state.write_set');
      if (!blockingPredicate) blockingPredicate = "evidence_scope_missing";
    }
    if (!state || !Array.isArray(state.read_set) || (state.read_set as unknown[]).length === 0) {
      errors.push('tier "deep" requires state.read_set');
      if (!blockingPredicate) blockingPredicate = "evidence_scope_missing";
    }
  }

  if (input.tier === "standard") {
    if (!verificationArtifacts || verificationArtifacts.length === 0) {
      notes.push('tier "standard" recommends verification_artifacts');
    }
    if (!hasScopeDeclared(verificationArtifacts)) {
      notes.push('tier "standard" recommends evidence scope (verifies/does_not_verify)');
    }
    if (!untestedRegions || untestedRegions.length === 0) {
      notes.push('tier "standard" recommends untested_regions');
    }
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

  // Canonical contradiction: verification.status passed must imply fix_status fixed
  if (verifyStatus !== undefined && fixStatus !== undefined) {
    if (verifyStatus === "passed" && fixStatus !== "fixed") {
      errors.push(`canonical contradiction: verification.status is "passed" but claim.fix_status is "${fixStatus}" (must be "fixed")`);
    }

    // Reject accepted when verification is non-success
    if (input.acceptance_status === "accepted" && ["failed", "blocked", "skipped"].includes(verifyStatus)) {
      errors.push(`canonical contradiction: acceptance_status is "accepted" but verification.status is "${verifyStatus}"`);
    }
  }

  // Admission outcome and acceptance status alignment
  if (admissionOutcome !== undefined) {
    // acceptance_status=accepted only when admission.outcome=success
    if (input.acceptance_status === "accepted" && admissionOutcome !== "success") {
      errors.push(`canonical contradiction: acceptance_status is "accepted" but admission.outcome is "${admissionOutcome}" (must be "success")`);
    }

    // non-success outcome -> acceptance_status=withheld
    if (admissionOutcome !== "success" && input.acceptance_status === "accepted") {
      errors.push(`canonical contradiction: non-success outcome "${admissionOutcome}" cannot have acceptance_status "accepted"`);
    }

    // blocked/failed/skipped requires handoff.next_action + handoff.owner
    if (["blocked", "failed", "skipped"].includes(admissionOutcome)) {
      if (!handoff || !handoff.next_action || String(handoff.next_action).trim().length === 0) {
        errors.push(`admission.outcome "${admissionOutcome}" requires handoff.next_action`);
      }
      if (!handoff || !handoff.owner || String(handoff.owner).trim().length === 0) {
        errors.push(`admission.outcome "${admissionOutcome}" requires handoff.owner`);
      }
    }
  }

  // Also check handoff required for verification blocked/failed/skipped
  if (verifyStatus !== undefined && ["blocked", "failed", "skipped"].includes(verifyStatus)) {
    if (!handoff || !handoff.next_action || String(handoff.next_action).trim().length === 0) {
      errors.push(`verification.status "${verifyStatus}" requires handoff.next_action`);
    }
    if (!handoff || !handoff.owner || String(handoff.owner).trim().length === 0) {
      errors.push(`verification.status "${verifyStatus}" requires handoff.owner`);
    }
  }

  // PGV is advisory-only; never blocks by default
  if (input.pgv_risk) {
    notes.push(`PGV risk level: ${input.pgv_risk} (advisory-only)`);
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
  const finalOutcome: VerifyOutcome = admissionOutcome && ["success", "failed", "blocked", "skipped", "timeout", "error"].includes(admissionOutcome)
    ? (admissionOutcome as VerifyOutcome)
    : "success";

  return {
    outcome: finalOutcome,
    acceptance_status: acceptanceStatus(finalOutcome),
    errors: [],
    notes: notes.length > 0 ? [...notes, "admission checks passed"] : ["admission checks passed"],
    blocking_predicate: null,
  };
}
