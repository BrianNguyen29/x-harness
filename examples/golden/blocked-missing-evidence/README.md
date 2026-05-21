# Golden Example: Blocked — Missing Evidence

A standard-tier completion card that is withheld because the evidence packet is empty.

## Scenario

An agent claims completion for a standard-tier task but provides no evidence. The verify gate withholds admission because the evidence floor is not met.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The agent's completion claim (missing evidence).
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
npx x-harness verify --card examples/golden/blocked-missing-evidence/completion-card.yaml
# -> WITHHELD
# Reason: tier "standard" requires evidence packet
```
