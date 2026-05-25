export type RecoveryPredicate =
  | "evidence_missing"
  | "evidence_scope_missing"
  | "typecheck_failed"
  | "test_failed"
  | "lint_failed"
  | "build_failed"
  | "approval_missing"
  | "conflicting_scope"
  | "verifier_not_read_only"
  | "admission_failed"
  | "evidence_floor_not_met"
  | "evidence_provenance_missing"
  | "state_read_write_missing"
  | "done_checklist_missing"
  | "prediction_missing"
  | "prediction_invalid"
  | "done_checklist_prediction_mismatch"
  | "stale_ground"
  | "Fpermission"
  | "Fintervention";

export interface RecoveryRoute {
  next_action: string;
  owner: string;
}

const DEFAULT_ROUTES: Record<string, RecoveryRoute> = {
  evidence_missing: {
    next_action: "Attach validation evidence or explain why unavailable.",
    owner: "implementation-worker",
  },
  evidence_floor_not_met: {
    next_action:
      "Attach the tier-required evidence floor and rerun verification.",
    owner: "implementation-worker",
  },
  evidence_scope_missing: {
    next_action:
      "Declare what each validation artifact verifies and does not verify.",
    owner: "implementation-worker",
  },
  evidence_provenance_missing: {
    next_action:
      "Attach strict evidence provenance fields and rerun verification.",
    owner: "implementation-worker",
  },
  typecheck_failed: {
    next_action: "Return to implementation-worker for type repair.",
    owner: "implementation-worker",
  },
  test_failed: {
    next_action:
      "Diagnose failing behavior and update implementation or tests.",
    owner: "implementation-worker",
  },
  lint_failed: {
    next_action:
      "Fix lint issues or justify why the lint rule is not applicable.",
    owner: "implementation-worker",
  },
  build_failed: {
    next_action: "Fix build failure before requesting admission.",
    owner: "implementation-worker",
  },
  approval_missing: {
    next_action: "Request human approval before admission.",
    owner: "user",
  },
  conflicting_scope: {
    next_action: "Ask user to clarify task scope.",
    owner: "user",
  },
  verifier_not_read_only: {
    next_action: "Rerun verification with a read-only verifier.",
    owner: "admission-verifier",
  },
  admission_failed: {
    next_action: "Resolve admission validation errors and rerun verification.",
    owner: "implementation-worker",
  },
  state_read_write_missing: {
    next_action: "Declare state.read_set and state.write_set for the task.",
    owner: "implementation-worker",
  },
  done_checklist_missing: {
    next_action:
      "Declare the done_checklist required for standard or deep admission.",
    owner: "implementation-worker",
  },
  prediction_missing: {
    next_action:
      "Declare the falsifiable prediction required for standard or deep admission.",
    owner: "implementation-worker",
  },
  prediction_invalid: {
    next_action:
      "Complete the required prediction fields and rerun verification.",
    owner: "implementation-worker",
  },
  done_checklist_prediction_mismatch: {
    next_action:
      "Align done_checklist.prediction_declared with the prediction block.",
    owner: "implementation-worker",
  },
  stale_ground: {
    next_action:
      "Refresh stale context or rule it out before requesting admission.",
    owner: "implementation-worker",
  },
  Fpermission: {
    next_action:
      "Request human approval for this protected path change before admission.",
    owner: "user",
  },
  Fintervention: {
    next_action:
      "Review intervention artifact for authority boundary violation and resolve.",
    owner: "implementation-worker",
  },
};

export function getRecoveryRoute(
  predicate: string | null | undefined
): RecoveryRoute | null {
  if (!predicate) return null;
  return DEFAULT_ROUTES[predicate] ?? null;
}

export function suggestRecovery(
  errors: string[],
  outcome: string
): { predicate: RecoveryPredicate | null; route: RecoveryRoute | null } {
  if (outcome !== "blocked" && outcome !== "failed") {
    return { predicate: null, route: null };
  }

  // Heuristic: map error text to predicate
  const errorText = errors.join("; ").toLowerCase();
  if (errorText.includes("stale_ground"))
    return {
      predicate: "stale_ground",
      route: getRecoveryRoute("stale_ground"),
    };
  if (errorText.includes("done_checklist.prediction_declared"))
    return {
      predicate: "done_checklist_prediction_mismatch",
      route: getRecoveryRoute("done_checklist_prediction_mismatch"),
    };
  if (errorText.includes("done_checklist"))
    return {
      predicate: "done_checklist_missing",
      route: getRecoveryRoute("done_checklist_missing"),
    };
  if (errorText.includes("prediction.")) {
    return {
      predicate: "prediction_invalid",
      route: getRecoveryRoute("prediction_invalid"),
    };
  }
  if (errorText.includes("prediction"))
    return {
      predicate: "prediction_missing",
      route: getRecoveryRoute("prediction_missing"),
    };
  if (errorText.includes("governance") && errorText.includes("permission"))
    return {
      predicate: "Fpermission",
      route: getRecoveryRoute("Fpermission"),
    };
  if (errorText.includes("governance") && errorText.includes("intervention"))
    return {
      predicate: "Fintervention",
      route: getRecoveryRoute("Fintervention"),
    };
  if (errorText.includes("approval"))
    return {
      predicate: "approval_missing",
      route: getRecoveryRoute("approval_missing"),
    };
  if (errorText.includes("typecheck") || errorText.includes("type check"))
    return {
      predicate: "typecheck_failed",
      route: getRecoveryRoute("typecheck_failed"),
    };
  if (errorText.includes("test") && !errorText.includes("typecheck"))
    return { predicate: "test_failed", route: getRecoveryRoute("test_failed") };
  if (errorText.includes("lint"))
    return { predicate: "lint_failed", route: getRecoveryRoute("lint_failed") };
  if (errorText.includes("build"))
    return {
      predicate: "build_failed",
      route: getRecoveryRoute("build_failed"),
    };
  if (
    errorText.includes("state") ||
    errorText.includes("read_set") ||
    errorText.includes("write_set")
  )
    return {
      predicate: "state_read_write_missing",
      route: getRecoveryRoute("state_read_write_missing"),
    };
  if (
    errorText.includes("scope") ||
    errorText.includes("untested") ||
    errorText.includes("does_not_verify")
  )
    return {
      predicate: "evidence_scope_missing",
      route: getRecoveryRoute("evidence_scope_missing"),
    };
  if (errorText.includes("evidence floor"))
    return {
      predicate: "evidence_floor_not_met",
      route: getRecoveryRoute("evidence_floor_not_met"),
    };
  if (errorText.includes("evidence provenance"))
    return {
      predicate: "evidence_provenance_missing",
      route: getRecoveryRoute("evidence_provenance_missing"),
    };
  if (errorText.includes("evidence"))
    return {
      predicate: "evidence_missing",
      route: getRecoveryRoute("evidence_missing"),
    };
  return {
    predicate: "admission_failed",
    route: getRecoveryRoute("admission_failed"),
  };
}

export interface PlaybookSuggestion {
  predicate: RecoveryPredicate;
  route: RecoveryRoute;
  review_required: boolean;
  rationale: string;
  observed_count?: number;
  confidence?: "low" | "medium" | "high";
  source_trace_events?: number;
}

export interface TraceEventLike {
  outcome?: string;
  blocking_predicate?: string | null;
  errors?: string[];
}

/**
 * Generate a deterministic recovery playbook candidate from trace events.
 */
export function generatePlaybookFromTrace(
  events: TraceEventLike[]
): PlaybookSuggestion[] {
  const failedOrBlocked = events.filter(
    (e) => e.outcome === "failed" || e.outcome === "blocked"
  );
  if (failedOrBlocked.length === 0) return [];

  // Group by blocking predicate
  const groups = new Map<string, TraceEventLike[]>();
  for (const event of failedOrBlocked) {
    const predicate = event.blocking_predicate ?? "admission_failed";
    const list = groups.get(predicate) ?? [];
    list.push(event);
    groups.set(predicate, list);
  }

  const suggestions: PlaybookSuggestion[] = [];
  for (const [predicate, groupEvents] of groups) {
    const route = getRecoveryRoute(predicate);
    if (!route) continue;

    const count = groupEvents.length;
    const confidence: "low" | "medium" | "high" =
      count >= 5 ? "high" : count >= 2 ? "medium" : "low";

    suggestions.push({
      predicate: predicate as RecoveryPredicate,
      route,
      review_required: true,
      rationale: `Observed in ${count} trace event(s) with predicate "${predicate}"`,
      observed_count: count,
      confidence,
      source_trace_events: count,
    });
  }

  // Sort by observed_count descending
  suggestions.sort((a, b) => (b.observed_count ?? 0) - (a.observed_count ?? 0));
  return suggestions;
}

/**
 * Generate a deterministic recovery playbook candidate from errors.
 * Does NOT mutate policies or completion cards.
 */
export function generatePlaybook(
  errors: string[],
  outcome: string
): PlaybookSuggestion[] {
  if (outcome !== "blocked" && outcome !== "failed") {
    return [];
  }

  const suggestions: PlaybookSuggestion[] = [];
  const seen = new Set<string>();

  for (const error of errors) {
    const suggestion = suggestRecovery([error], outcome);
    if (
      suggestion.predicate &&
      suggestion.route &&
      !seen.has(suggestion.predicate)
    ) {
      seen.add(suggestion.predicate);
      suggestions.push({
        predicate: suggestion.predicate,
        route: suggestion.route,
        review_required: true,
        rationale: `Detected from error: "${error}"`,
      });
    }
  }

  return suggestions;
}

export function renderPlaybookMarkdown(
  suggestions: PlaybookSuggestion[]
): string {
  const lines: string[] = [
    "# Recovery Playbook (Review Required)",
    "",
    "> This playbook is a candidate generated from verification failures. Review before applying.",
    "> It does NOT modify policies or completion cards.",
    "",
  ];

  for (const s of suggestions) {
    lines.push(`## ${s.predicate}`);
    lines.push("");
    lines.push(`- **Next action:** ${s.route.next_action}`);
    lines.push(`- **Owner:** ${s.route.owner}`);
    lines.push(`- **Review required:** ${s.review_required ? "yes" : "no"}`);
    lines.push(`- **Rationale:** ${s.rationale}`);
    if (s.observed_count !== undefined) {
      lines.push(`- **Observed count:** ${s.observed_count}`);
    }
    if (s.confidence) {
      lines.push(`- **Confidence:** ${s.confidence}`);
    }
    if (s.source_trace_events !== undefined) {
      lines.push(`- **Source trace events:** ${s.source_trace_events}`);
    }
    lines.push("");
  }

  if (suggestions.length === 0) {
    lines.push("No recovery actions suggested.");
    lines.push("");
  }

  lines.push("---");
  lines.push("Generated by x-harness recovery playbook generator.");
  lines.push("");

  return lines.join("\n");
}
