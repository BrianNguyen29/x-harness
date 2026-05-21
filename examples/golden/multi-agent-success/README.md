# Golden Example: Multi-Agent Success

A standard-tier task completed collaboratively by multiple agents, with full evidence and passing verification.

## Scenario

Three agents collaborate:
1. **worker-1** implements the core feature.
2. **worker-2** writes tests and documentation.
3. **verifier** runs the verify gate and confirms acceptance.

The completion card includes evidence from all contributors and passes admission.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The collaborative completion claim.
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
npx x-harness verify --card examples/golden/multi-agent-success/completion-card.yaml
# -> ACCEPTED
```
