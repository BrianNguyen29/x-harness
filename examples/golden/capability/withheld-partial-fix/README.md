# Golden Example: Withheld — Partial Fix

A completion card where the agent admits the fix is only partial and verification has failed.

## Scenario

An agent completes part of a task but cannot fully resolve it. The completion card correctly reflects the partial state, and the verify gate withholds admission.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The honest but incomplete completion claim.
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
xh verify --card examples/golden/capability/withheld-partial-fix/completion-card.yaml
# -> WITHHELD
# Admission outcome: failed
```
