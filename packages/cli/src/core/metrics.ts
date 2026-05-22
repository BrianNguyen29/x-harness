import type { AdmissionInput } from "./admission.js";

export interface MetricsReport {
  verification_strength: {
    command_evidence_count: number;
    oracle_kinds: string[];
    untested_regions_count: number;
    remaining_risks_count: number;
  };
  state_consistency: {
    owner_present: boolean;
    accountable_present: boolean;
    files_changed_present: boolean;
    admission_mapping_valid: boolean;
  };
  recovery_ability: {
    blocked_has_next_action: boolean;
    blocked_has_owner: boolean;
    recovery_route_present: boolean;
  };
  replayability: {
    completion_card_present: boolean;
    input_card_hash_present: boolean;
    policy_hash_present: boolean;
  };
  cost: {
    default_context_class: "low" | "medium" | "high";
    verify_runtime_ms: number;
  };
}

export function computeMetrics(
  input: AdmissionInput,
  options: {
    inputCardHash?: string | null;
    policyHash?: string | null;
    verifyRuntimeMs?: number;
  }
): MetricsReport {
  const evidence = input.evidence as Record<string, unknown> | undefined;
  const claim = input.claim as Record<string, unknown> | undefined;
  const cardEvidence = claim?.evidence as unknown[] | undefined;

  const verificationArtifacts =
    (evidence?.verification_artifacts as unknown[] | undefined) ?? [];
  const untestedRegions =
    (evidence?.untested_regions as unknown[] | undefined) ?? [];
  const remainingRisks =
    (evidence?.remaining_risks as unknown[] | undefined) ?? [];

  const oracleKinds = new Set<string>();
  for (const artifact of verificationArtifacts) {
    const a = artifact as Record<string, unknown> | undefined;
    if (a?.kind && typeof a.kind === "string") oracleKinds.add(a.kind);
  }

  const filesChanged =
    (evidence?.files_changed as unknown[] | undefined) ?? cardEvidence ?? [];

  const outcome = (input.admission as Record<string, unknown> | undefined)
    ?.outcome as string | undefined;
  const blocked = outcome === "blocked" || outcome === "failed";

  const handoff = input.handoff as Record<string, unknown> | undefined;
  const hasNextAction =
    !!handoff?.next_action &&
    String(handoff.next_action).trim().length > 0 &&
    String(handoff.next_action) !== "none";
  const hasOwner = !!handoff?.owner && String(handoff.owner).trim().length > 0;

  const tier = input.tier ?? "standard";
  const contextClass: "low" | "medium" | "high" =
    tier === "light" ? "low" : tier === "deep" ? "high" : "medium";

  return {
    verification_strength: {
      command_evidence_count: verificationArtifacts.length,
      oracle_kinds: Array.from(oracleKinds),
      untested_regions_count: untestedRegions.length,
      remaining_risks_count: remainingRisks.length,
    },
    state_consistency: {
      owner_present: !!input.owner && input.owner.trim().length > 0,
      accountable_present:
        !!input.accountable && input.accountable.trim().length > 0,
      files_changed_present: filesChanged.length > 0,
      admission_mapping_valid:
        outcome === "success"
          ? input.acceptance_status === "accepted"
          : input.acceptance_status !== "accepted",
    },
    recovery_ability: {
      blocked_has_next_action: blocked ? hasNextAction : true,
      blocked_has_owner: blocked ? hasOwner : true,
      recovery_route_present: blocked ? hasNextAction && hasOwner : true,
    },
    replayability: {
      completion_card_present:
        !!input.task_id && input.task_id.trim().length > 0,
      input_card_hash_present: !!options.inputCardHash,
      policy_hash_present: !!options.policyHash,
    },
    cost: {
      default_context_class: contextClass,
      verify_runtime_ms: options.verifyRuntimeMs ?? 0,
    },
  };
}
