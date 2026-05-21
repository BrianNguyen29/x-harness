# Recovery Routing

When verification is blocked or failed, x-harness suggests a recovery route based on the blocking predicate.

## Default routes

| Predicate | Next action | Owner |
|-----------|-------------|-------|
| evidence_missing | Attach validation evidence or explain why unavailable. | implementation-worker |
| evidence_scope_missing | Declare what each validation artifact verifies and does not verify. | implementation-worker |
| typecheck_failed | Return to implementation-worker for type repair. | implementation-worker |
| test_failed | Diagnose failing behavior and update implementation or tests. | implementation-worker |
| lint_failed | Fix lint issues or justify why the lint rule is not applicable. | implementation-worker |
| build_failed | Fix build failure before requesting admission. | implementation-worker |
| approval_missing | Request human approval before admission. | user |
| conflicting_scope | Ask user to clarify task scope. | user |
| verifier_not_read_only | Rerun verification with a read-only verifier. | admission-verifier |
| admission_failed | Resolve admission validation errors and rerun verification. | implementation-worker |

## Usage

Recovery routes are included in:

- `npx x-harness verify --json` under the `recovery` field.
- `npx x-harness report --metrics` as part of recovery ability metrics.

The CLI does not mutate `completion-card.yaml` by default.
