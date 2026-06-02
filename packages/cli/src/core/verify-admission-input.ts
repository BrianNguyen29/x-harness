import type { runAdmission } from "./admission.js";
import type { VerifyLoadedSources } from "./verify-source-loader.js";

export interface VerifyAdmissionOptions {
  tier?: string;
  taskId?: string;
  staleGround?: boolean;
  strict?: boolean;
  contextFloor?: boolean;
}

export function buildAdmissionInput(
  loaded: VerifyLoadedSources,
  opts: VerifyAdmissionOptions
): Parameters<typeof runAdmission>[0] {
  const tier = (opts.tier as "light" | "standard" | "deep") ?? "standard";
  const staleGroundFromCard =
    loaded.card &&
    typeof (loaded.card as Record<string, unknown>).stale_ground === "boolean"
      ? ((loaded.card as Record<string, unknown>).stale_ground as boolean)
      : false;
  const effectiveStaleGround =
    opts.staleGround === true ? true : staleGroundFromCard;

  if (loaded.card) {
    const card = loaded.card;
    return {
      schema_version: String(card.schema_version ?? ""),
      task_id: String(card.task_id ?? opts.taskId ?? ""),
      tier: (card.tier as "light" | "standard" | "deep") ?? tier,
      owner: String(card.owner ?? ""),
      accountable: String(card.accountable ?? ""),
      claim: card.claim as Record<string, unknown>,
      verification: card.verification as Record<string, unknown>,
      admission: card.admission as Record<string, unknown>,
      acceptance_status: card.acceptance_status as "accepted" | "withheld",
      handoff: card.handoff as Record<string, unknown>,
      evidence: card.evidence as Record<string, unknown> | undefined,
      state: card.state as Record<string, unknown> | undefined,
      governance: card.governance as Record<string, unknown> | undefined,
      intake: card.intake as Record<string, unknown> | undefined,
      context_acknowledged:
        typeof card.context_acknowledged === "boolean"
          ? card.context_acknowledged
          : undefined,
      done_checklist: card.done_checklist as
        | Record<string, unknown>
        | undefined,
      prediction: card.prediction as Record<string, unknown> | undefined,
      approval_receipt: card.approval_receipt as
        | Record<string, unknown>
        | undefined,
      pgv_advice: card.pgv_advice as Record<string, unknown> | undefined,
      context_alignment: card.context_alignment as
        | Record<string, unknown>
        | undefined,
      product_intent: card.product_intent as
        | Record<string, unknown>
        | undefined,
      test_adequacy: card.test_adequacy as Record<string, unknown> | undefined,
      evidence_adequacy: card.evidence_adequacy as
        | Record<string, unknown>
        | undefined,
      isCardMode: true,
      staleGround: effectiveStaleGround,
      strict: opts.strict === true,
      contextFloor: opts.contextFloor === true,
      cardPath: loaded.cardPath,
    };
  }

  return {
    claim: loaded.claim,
    evidence: loaded.evidence,
    subagentReturn: loaded.subagentReturn,
    tier,
    done_checklist: loaded.subagentReturn?.done_checklist as
      | Record<string, unknown>
      | undefined,
    prediction: loaded.subagentReturn?.prediction as
      | Record<string, unknown>
      | undefined,
    pgv_advice: loaded.subagentReturn?.pgv_advice as
      | Record<string, unknown>
      | undefined,
    context_alignment: loaded.subagentReturn?.context_alignment as
      | Record<string, unknown>
      | undefined,
    product_intent: loaded.subagentReturn?.product_intent as
      | Record<string, unknown>
      | undefined,
    test_adequacy: loaded.subagentReturn?.test_adequacy as
      | Record<string, unknown>
      | undefined,
    evidence_adequacy: loaded.subagentReturn?.evidence_adequacy as
      | Record<string, unknown>
      | undefined,
    staleGround: effectiveStaleGround,
    strict: opts.strict === true,
    contextFloor: opts.contextFloor === true,
  };
}
