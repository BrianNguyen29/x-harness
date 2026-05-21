# Golden Example: Failed — Typecheck Recovery Route

A standard-tier completion card that is withheld due to build/typecheck errors and yields a recovery routing guide.

## Scenario

An agent implements changes but the verify gate registers type compilation failures. The verification outcome fails, and the harness suggestions route recovery back to the `implementation-worker` for code repair.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The agent's completion claim containing typecheck errors.
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
npx x-harness verify --card examples/golden/failed-typecheck-recovery-route/completion-card.yaml
# -> WITHHELD
# Reason: typecheck failed (Suggested recovery owner: implementation-worker)
```

## Try it

Run the verification gate:
```bash
node packages/cli/dist/index.js verify --card examples/golden/failed-typecheck-recovery-route/completion-card.yaml
```
