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
  | "admission_failed";

export interface RecoveryRoute {
  next_action: string;
  owner: string;
}

const DEFAULT_ROUTES: Record<string, RecoveryRoute> = {
  evidence_missing: {
    next_action: "Attach validation evidence or explain why unavailable.",
    owner: "implementation-worker",
  },
  evidence_scope_missing: {
    next_action: "Declare what each validation artifact verifies and does not verify.",
    owner: "implementation-worker",
  },
  typecheck_failed: {
    next_action: "Return to implementation-worker for type repair.",
    owner: "implementation-worker",
  },
  test_failed: {
    next_action: "Diagnose failing behavior and update implementation or tests.",
    owner: "implementation-worker",
  },
  lint_failed: {
    next_action: "Fix lint issues or justify why the lint rule is not applicable.",
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
};

export function getRecoveryRoute(predicate: string | null | undefined): RecoveryRoute | null {
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
  if (errorText.includes("approval")) return { predicate: "approval_missing", route: getRecoveryRoute("approval_missing") };
  if (errorText.includes("typecheck") || errorText.includes("type check")) return { predicate: "typecheck_failed", route: getRecoveryRoute("typecheck_failed") };
  if (errorText.includes("test") && !errorText.includes("typecheck")) return { predicate: "test_failed", route: getRecoveryRoute("test_failed") };
  if (errorText.includes("lint")) return { predicate: "lint_failed", route: getRecoveryRoute("lint_failed") };
  if (errorText.includes("build")) return { predicate: "build_failed", route: getRecoveryRoute("build_failed") };
  if (errorText.includes("scope") || errorText.includes("untested") || errorText.includes("does_not_verify")) return { predicate: "evidence_scope_missing", route: getRecoveryRoute("evidence_scope_missing") };
  if (errorText.includes("evidence")) return { predicate: "evidence_missing", route: getRecoveryRoute("evidence_missing") };

  return { predicate: "admission_failed", route: getRecoveryRoute("admission_failed") };
}
