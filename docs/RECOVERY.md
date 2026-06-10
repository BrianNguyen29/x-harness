# Recovery Routing

When verification is blocked or failed, x-harness suggests a recovery route based on the blocking predicate.

## Default routes

| Predicate                         | Next action                                                          | Owner                 |
| --------------------------------- | -------------------------------------------------------------------- | --------------------- |
| evidence_missing                  | Attach validation evidence or explain why unavailable.               | implementation-worker |
| evidence_floor_not_met            | Attach the tier-required evidence floor and rerun verification.      | implementation-worker |
| evidence_scope_missing            | Declare what each validation artifact verifies and does not verify.  | implementation-worker |
| typecheck_failed                  | Return to implementation-worker for type repair.                     | implementation-worker |
| test_failed                       | Diagnose failing behavior and update implementation or tests.       | implementation-worker |
| lint_failed                       | Fix lint issues or justify why the lint rule is not applicable.      | implementation-worker |
| build_failed                      | Fix build failure before requesting admission.                        | implementation-worker |
| approval_missing                  | Request human approval before admission.                             | user                  |
| classifier_approval_required      | Request approval for high-risk commands and attach an approval_receipt. | user              |
| conflicting_scope                 | Ask user to clarify task scope.                                      | user                  |
| verifier_not_read_only            | Rerun verification with a read-only verifier.                       | admission-verifier    |
| state_read_write_missing          | Declare state.read_set and state.write_set for the task.             | implementation-worker |
| done_checklist_missing            | Declare the done_checklist required for standard or deep admission.   | implementation-worker |
| done_checklist_mismatch           | Align done_checklist claims with state, evidence, artifacts, and prediction. | implementation-worker |
| prediction_missing               | Declare the falsifiable prediction required for standard or deep admission. | implementation-worker |
| prediction_invalid                | Complete the required prediction fields and rerun verification.      | implementation-worker |
| done_checklist_prediction_mismatch | Align done_checklist.prediction_declared with the prediction block. | implementation-worker |
| stale_ground                      | Refresh stale context or rule it out before requesting admission.    | implementation-worker |
| context_floor_blocked             | Add context_alignment with stale_ground_checked, at least one ref, and resolve context questions. | implementation-worker |
| admission_failed                  | Resolve admission validation errors and rerun verification.           | implementation-worker |
| Fpermission                       | Request human approval for this protected path change before admission. | user              |
| Fintervention                     | Review intervention artifact for authority boundary violation and resolve. | implementation-worker |
| contract_oracle_blocked           | Fix contract oracle violations or update the contract oracle policy.       | implementation-worker |

## Recovery playbook

Generate a deterministic, review-required recovery playbook candidate:

```bash
./x-harness recovery suggest \
  --errors "tests failed; lint errors" \
  --outcome failed
```

> The TypeScript compatibility CLI is also available in source checkouts via `node packages/cli/dist/index.js <command>`.

The playbook:
- Maps each error to a recovery predicate deterministically
- Dedupes multiple errors that map to the same predicate
- Marks every suggestion as `review_required: true`
- Does **not** mutate policies or completion cards

## Usage

Recovery routes are included in:

- `./x-harness verify --json` under the `recovery` field.
- `./x-harness report --metrics` as part of recovery ability metrics.
- `./x-harness recovery suggest` generates a review-required playbook.

The CLI does not mutate `completion-card.yaml` by default.

## Patch card

`xh recover --patch-card <path>` generates a deterministic, conservative patch for a completion card. By default it runs in **dry-run/preview mode** and never mutates the card file.

```bash
# Preview what would change
xh recover --patch-card completion-card.yaml

# Apply the patch in-place (requires --confirm)
xh recover --patch-card completion-card.yaml --confirm

# Write the patched card to a new file instead of overwriting
xh recover --patch-card completion-card.yaml --out patched-card.yaml

# Append evidence to empty claim.evidence and evidence.files_changed
xh recover --patch-card completion-card.yaml --confirm --evidence src/main.go
```

Behavior:
- `--confirm` is required to actually mutate the card. Without it, the command prints a preview of the planned operations.
- `--out <path>` writes the patched card to a new file. In dry-run mode, `--out` writes the would-be patched card for review.
- `--evidence <id-or-path>` appends the value to `claim.evidence` and `evidence.files_changed` only when those fields are empty.
- When confirming, a timestamped backup is written next to the original card before mutation.
- The patcher never overwrites user-provided scalar fields; existing values are reported as `skipped`.
- V1 patch categories are intentionally narrow: missing `handoff.next_action` and `handoff.owner` are filled using deterministic recovery routes when safe.
