# Golden Example: Failed — Invalid Status

A completion card with a canonical contradiction between `acceptance_status` and `admission.outcome`.

## Scenario

An agent incorrectly marks the task as `accepted` while the admission outcome is `failed`. This violates the schema invariant that `accepted` requires `admission.outcome: success`.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The invalid completion claim.
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
node packages/cli/dist/index.js verify --card examples/golden/failed-invalid-status/completion-card.yaml
# -> WITHHELD
# Reason: canonical contradiction
```
