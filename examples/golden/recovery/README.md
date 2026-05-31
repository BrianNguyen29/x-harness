# Recovery Golden Test Suite

This directory holds recovery-route fixtures used by `packages/cli/tests/recovery.test.ts`.

## Scope

These fixtures assert that `packages/cli/src/core/recovery.ts` DEFAULT_ROUTES stay in sync with:
- `policies/recovery.yaml` (policy source of truth)
- The completion-card golden examples that surface recovery routes end-to-end

## Unsupported predicates

The following predicates do not have dedicated recovery routes in the current implementation:

- `schema_invalid` — schema validation failures surface as `admission_failed`.
- `stale_ground` — handled as a fail-closed admission check in the admission engines, not recovery routing.
- `policy_drift` — falls back to existing admission checks; no dedicated recovery route.
- `unknown_failure` — falls back to `admission_failed` via `suggestRecovery`.

## Mapping notes

| Requested predicate | Actual predicate in code | Notes |
|---|---|---|
| `tests_failed` | `test_failed` | Name mismatch; same semantics. |
| `unknown_failure` | `admission_failed` | Fallback catch-all. |
