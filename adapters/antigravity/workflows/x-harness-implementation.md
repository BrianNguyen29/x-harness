# x-harness Implementation Workflow

## Goal

Produce a completion card for a task.

## Steps

1. **Receive task** from orchestrator with tier and scope.
2. **Execute work** within scope. Do not expand scope without escalation.
3. **Gather evidence**:
   - Commands ran
   - Files changed
   - Key outputs
4. **Write completion card** as `completion-card.yaml`:
   - `schema_version`, `task_id`, `tier`
   - `owner`, `accountable`
   - `claim.fix_status`, `claim.summary`, `claim.evidence`
   - `verification.status`, `verification.checks`
   - `admission.outcome`
   - `acceptance_status`
   - `handoff.next_action`, `handoff.owner`
5. **Self-check**:
   - If `fix_status: fixed`, ensure `verification.status: passed`.
   - If blocked, ensure `handoff` is populated.
6. **Submit** for verification.

## Constraints

- Use `light` by default.
- PGV advice is advisory-only.
- Do not claim completion without a completion card.
