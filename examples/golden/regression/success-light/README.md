# Golden Example: Success (Light Tier)

A minimal, valid completion card for a light-tier task that passes verification and is accepted.

## Scenario

A single agent completes a small task with minimal evidence. The completion card meets all requirements for the `light` tier.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The agent's completion claim.
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
node packages/cli/dist/index.js verify --card examples/golden/regression/success-light/completion-card.yaml
# -> ACCEPTED
```
