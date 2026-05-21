# Solo Agent Example

A single agent performs a narrow, low-ceremony task and produces a completion card.

## Files

- `completion-card.yaml`: Light-tier completion card.
- `task.md`: Brief task description.

## Expected verify outcome

```bash
npx x-harness verify --card examples/01-solo-agent/completion-card.yaml
# -> ACCEPTED
```
