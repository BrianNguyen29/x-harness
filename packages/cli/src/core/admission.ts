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
  // Backward compatibility: subagent-return shape
  evidence?: Record<string, unknown>;
  subagentReturn?: Record<string, unknown>;
}

export interface AdmissionResult {
  outcome: VerifyOutcome;
  acceptance_status: AcceptanceStatus;
  errors: string[];
  notes: string[];
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

export function runAdmission(input: AdmissionInput): AdmissionResult {
  const errors: string[] = [];
  const notes: string[] = [];

  // Stale-ground fail-closed
  if (input.staleGround) {
    errors.push("stale_ground detected: withholding pending refresh or ruling out");
    return {
      outcome: "blocked",
      acceptance_status: "withheld",
      errors,
      notes: ["stale-ground policy: if_detected = withhold"],
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
  };
}
