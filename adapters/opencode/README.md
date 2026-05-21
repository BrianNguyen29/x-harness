# OpenCode Adapter

This adapter integrates x-harness with the OpenCode agent platform.

## Workflow

1. **Orchestrator** dispatches a task to a worker agent.
2. **Worker** performs the task and writes `completion-card.yaml`.
3. **Verify agent** (`verify-agent.md`) runs read-only verification.
4. **Outcome**: accepted or withheld. Handoff routed back to orchestrator.

## Files

- `README.md`: This file.
- `verify-agent.md`: Read-only verifier rules and commands.
- `opencode.example.json`: Example OpenCode configuration.
- `opencode.verify.example.json`: Example verify-agent configuration.
- `orchestrator_append.example.md`: Example orchestrator handoff snippet.

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

- Use `light` tier by default.
- Verifier is read-only.
- PGV is advisory-only.
- Non-success verify outcomes are always withheld.
- No heavy runtime required (no daemon, database, server, or MCP).
