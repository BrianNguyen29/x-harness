# x-harness Recover Workflow

## Goal

Handle blocked verification outcomes without converting blocked to success.

## Trigger

- Verification returns `blocked`.
- `acceptance_status: withheld`.

## Steps

1. **Read verify output**: Identify blocking_predicate and reason.
2. **Determine cause**:
   - Missing evidence?
   - Stale context?
   - Partial fix?
   - Missing owner?
3. **Assign recovery**:
   - next_action: specific step to resolve blocker.
   - owner: who is responsible for the next step.
4. **Execute recovery**:
   - If missing evidence: worker attaches evidence.
   - If stale context: refresh context.
   - If partial fix: return to implementation.
5. **Re-verify**: Run `npx x-harness verify` again after recovery.

## Rules

- Never convert blocked directly to success.
- Always assign next_action and owner.
- Always re-run verification after recovery.
- PGV advice remains advisory-only during recovery.

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
