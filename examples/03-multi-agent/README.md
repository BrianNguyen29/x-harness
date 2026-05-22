# Multi-Agent Example

This example demonstrates a worker/verifier split for a standard-tier task.

## Flow

1. **Worker** (`worker-output.md`) performs implementation and writes the completion card.
2. **Verifier** (`verifier-output.md`) runs read-only verification against the card.
3. **Outcome**: accepted (all checks pass).

## Files

- `completion-card.yaml`: The verified completion card.
- `verify-report.md`: The verifier's read-only report.
- `worker-output.md`: Summary of worker's claim and evidence.
- `verifier-output.md`: Summary of verifier's checks and admission decision.

## Expected verify outcome

```bash
node packages/cli/dist/index.js verify --card examples/03-multi-agent/completion-card.yaml
# -> ACCEPTED
```
