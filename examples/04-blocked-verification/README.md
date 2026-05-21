# Blocked Verification Example

A task is blocked due to missing dependency. The completion card is withheld with a handoff.

## Files

- `completion-card.yaml`: Blocked completion card with handoff.
- `rollback-policy.yaml`: Rollback guidance.
- `deep-handoff.md`: Deep-tier handoff notes.
- `audit-report.md`: Audit notes.

## Expected verify outcome

```bash
npx x-harness verify --card examples/04-blocked-verification/completion-card.yaml
# -> WITHHELD
# Handoff: resolve dependency and re-verify -> charlie
```
