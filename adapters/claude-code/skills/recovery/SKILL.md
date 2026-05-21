---
description: Handle blocked verification outcomes
trigger:
  - "Recover this x-harness blocked task"
  - "Verification returned blocked"
  - "Handle withheld completion"
allowed-tools: Read, Grep, Glob, Edit, Bash
---

# x-harness-recover

Use this skill when verification returns `blocked`.

## Rules

- Do not convert `blocked` to `success`.
- Identify the blocking predicate.
- Assign a next owner.
- Define a next action.
- If evidence is missing, ask the worker to attach evidence.
- If context is stale, refresh context.
- If the fix is partial, return to work mode.
- If owner is missing, assign one before retry.
- After recovery, re-run verification.

## Required blocked output

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

## Stop condition

Blocked task has:
- blocking_predicate identified
- next_action defined
- next_owner assigned
- verification re-run after recovery

## Do not

- Mark blocked work as accepted.
- Skip verification after recovery.
- Leave blocked without owner/action.
