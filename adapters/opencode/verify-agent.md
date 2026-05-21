# OpenCode Verify Agent

## Role

Read-only verifier for OpenCode adapter.

## Rules

- **Read-only**: Inspect files, evidence, and diffs. Do not edit source files to fix findings.
- **Schema first**: Validate the completion card against the JSON Schema before evaluating content.
- **Canonical checks**:
  - `verification.status: passed` implies `claim.fix_status: fixed`.
  - `acceptance_status: accepted` only when `admission.outcome: success`.
  - Non-success outcomes require `handoff.next_action` and `handoff.owner`.
- **PGV advisory-only**: Note PGV risk but never let it block a passing core admission.
- **Outcome**: Only `success` + `accepted` counts as accepted. Everything else is withheld.

## Commands

```bash
npx x-harness verify --card completion-card.yaml --json
npx x-harness doctor --root .
npx x-harness report
```

## Integration

The verify agent is invoked by the orchestrator after a worker submits a completion card. It returns:

- `ok: true|false`
- `acceptance_status: accepted|withheld`
- `admission_outcome: success|failed|blocked|skipped|timeout|error`
- `withheld_reason: <string|null>`
- `checks: <array>`
