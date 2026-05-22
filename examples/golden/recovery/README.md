# Recovery Golden Test Suite

This directory holds recovery-route fixtures used by `packages/cli/tests/recovery.test.ts`.

## Scope

These fixtures assert that `packages/cli/src/core/recovery.ts` DEFAULT_ROUTES stay in sync with:
- `policies/recovery.yaml` (policy source of truth)
- The completion-card golden examples that surface recovery routes end-to-end

## Unsupported predicates

The following predicates are requested by the roadmap but do **not** yet have dedicated recovery routes:

- `schema_invalid` — schema validation failures surface as `admission_failed`.
- `stale_ground` — handled as a fail-closed admission check in `admission.ts`, not recovery routing.
- `policy_drift` — planned for future policy-code drift guard; not implemented.
- `unknown_failure` — falls back to `admission_failed` via `suggestRecovery`.

## Mapping notes

| Requested predicate | Actual predicate in code | Notes |
|---|---|---|
| `tests_failed` | `test_failed` | Name mismatch; same semantics. |
| `unknown_failure` | `admission_failed` | Fallback catch-all. |
