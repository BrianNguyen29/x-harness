export type VerifyOutcome = "success" | "failed" | "blocked" | "skipped" | "timeout" | "error";
export type AcceptanceStatus = "accepted" | "withheld";
export type Tier = "light" | "standard" | "deep";

export function acceptanceStatus(outcome: VerifyOutcome): AcceptanceStatus {
  return outcome === "success" ? "accepted" : "withheld";
}

export interface AdmissionInput {
  claim?: Record<string, unknown>;
  evidence?: Record<string, unknown>;
  subagentReturn?: Record<string, unknown>;
  tier?: Tier;
  staleGround?: boolean;
}

export interface AdmissionResult {
  outcome: VerifyOutcome;
  acceptance_status: AcceptanceStatus;
  errors: string[];
  notes: string[];
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

  // Canonical contradiction: verification.status passed must imply fix_status fixed
  if (input.subagentReturn) {
    const result = input.subagentReturn.result as Record<string, unknown> | undefined;
    const verification = input.subagentReturn.verification as Record<string, unknown> | undefined;

    if (result && verification) {
      const fixStatus = result.fix_status as string | undefined;
      const verifyStatus = verification.status as string | undefined;

      if (verifyStatus === "passed" && fixStatus !== "fixed") {
        errors.push(`canonical contradiction: verification.status is "passed" but result.fix_status is "${fixStatus}" (must be "fixed")`);
        return {
          outcome: "failed",
          acceptance_status: "withheld",
          errors,
          notes: ["canonical rule: verification.status passed implies fix_status fixed"],
        };
      }
    }
  }

  // Evidence/owner/accountable required
  if (input.evidence) {
    const ev = input.evidence;
    if (!ev.owner && !ev.accountable) {
      notes.push("evidence packet lacks owner/accountable fields");
    }
  }

  // Evidence floor check based on tier
  if (input.tier && input.tier !== "light") {
    if (!input.evidence) {
      errors.push(`tier "${input.tier}" requires evidence packet`);
    }
  }

  if (errors.length > 0) {
    return {
      outcome: "failed",
      acceptance_status: "withheld",
      errors,
      notes,
    };
  }

  // PGV is advisory-only, so we don't block based on pgv_advice

  return {
    outcome: "success",
    acceptance_status: "accepted",
    errors: [],
    notes: ["admission checks passed"],
  };
}
