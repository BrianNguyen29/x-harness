# Claude Code x-harness Instructions

This adapter integrates x-harness with Claude Code.

## Workflow

1. **Orchestrator** (you or a dispatcher) selects a tier and writes a task.
2. **Implementation worker** performs the task and produces a completion card.
3. **Admission verifier** runs read-only verification against the card.
4. **Outcome**: accepted or withheld. If withheld, worker receives handoff guidance.

## Agent roles

- `agents/implementation-worker.md`: Performs edits, writes completion card.
- `agents/admission-verifier.md`: Read-only inspector; runs `x-harness verify`.

## Skills

- `skills/verify/SKILL.md`: Read-only verification skill.
- `skills/handoff/SKILL.md`: Tiered handoff creation.
- `skills/recovery/SKILL.md`: Recovery packet preparation.

## Quick commands

```bash
# Verify a completion card
npx x-harness verify --card completion-card.yaml

# Check repo health
npx x-harness doctor --root .

# View trace summary
npx x-harness report
```

## Constraints

- Use `light` by default.
- Verifier is read-only.
- PGV is advisory-only.
- Non-success verify outcomes are always withheld.
