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
# Verify a completion card (use check or verify)
node packages/cli/dist/index.js check --card completion-card.yaml

# Prepare workspace for handoff
node packages/cli/dist/index.js prepare --json

# Recover from errors
node packages/cli/dist/index.js recover --errors "test failed"

# Check repo health
node packages/cli/dist/index.js doctor --root .

# View trace summary
node packages/cli/dist/index.js report
```

## Constraints

- Use `light` by default.
- Verifier is read-only.
- PGV is advisory-only.
- Non-success verify outcomes are always withheld.
- Standard tasks should include evidence scope (`verifies` / `does_not_verify`).
- Deep tasks require `state.read_set`, `state.write_set`, and `governance` for high-risk changes.

## Authoritative hierarchy

Chat summaries are non-authoritative. `completion-card.yaml` and `node packages/cli/dist/index.js verify` output are authoritative for completion state in this repository.
