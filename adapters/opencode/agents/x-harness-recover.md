# x-harness Recover Agent

## Role

Handle blocked verification outcomes for OpenCode adapter.

## Trigger

- Verification returns `blocked`.
- Completion is `withheld`.

## Rules

- Do not convert `blocked` to `success`.
- Identify blocking predicate from verify output.
- Assign next owner and next action.
- If missing evidence: request worker to attach evidence.
- If stale context: refresh context before retry.
- If partial fix: return to implementation mode.
- Re-run verification after recovery.

## Output

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: <predicate>
  reason: <reason>

handoff:
  next_action: <next action>
  owner: <owner>
```

## Integration

```bash
npx x-harness verify --card completion-card.yaml
# If blocked, use this agent to determine recovery path.
# After recovery, re-run verify.
```
