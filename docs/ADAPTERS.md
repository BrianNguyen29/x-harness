# x-harness Adapter Guide

x-harness adapters are thin conventions that make the harness usable inside specific agent platforms. Adapters are optional: the CLI (`x-harness verify`, `x-harness doctor`, `x-harness report`) is the source of truth.

## Default constraints (all adapters)

- **Lightweight first**: `light` tier is the default. Use `standard` for multi-step work. Use `deep` only for risk/control decisions.
- **Completion card required**: Write a completion card before claiming completion.
- **Verifier is read-only**: The verifier inspects files and evidence; it does not edit source files to fix findings while verifying.
- **Non-success -> withheld**: Any outcome other than `success` results in `acceptance_status: withheld`.
- **PGV advisory-only**: PGV risk levels are advisory. They never override verify and never grant admission authority by default.
- **No heavy runtime required**: No daemon, database, server, or MCP is required.

## Adapters

### Generic

- Location: `adapters/generic/`
- Use when: You want plain markdown conventions without platform-specific files.
- Key files: `AGENTS.md`

### Claude Code

- Location: `adapters/claude-code/`
- Use when: You use Claude Code as the agent platform.
- Key files: `CLAUDE.md`, `agents/implementation-worker.md`, `agents/admission-verifier.md`, skills under `skills/`
- Workflow: worker produces claim/evidence/card -> verifier runs read-only verify -> admission gate accepts or withholds.

### Cursor

- Location: `adapters/cursor/rules/`
- Use when: You use Cursor as the agent platform.
- Key files: `x-harness.mdc` (primary)
- Rules are applied via Cursor's `.cursor/rules/` directory.

### OpenCode

- Location: `adapters/opencode/`
- Use when: You use OpenCode as the agent platform.
- Key files: `README.md`, `verify-agent.md`
- Workflow: orchestrator dispatch -> worker completion -> verify-agent read-only gate.

### Antigravity

- Location: `adapters/antigravity/`
- Use when: You use Antigravity as the agent platform.
- Key files: `rules/x-harness.md`, `workflows/x-harness-implementation.md`, `workflows/x-harness-verify.md`

## Tier selection quick reference

| Tier     | When to use                                         | Evidence required                                                                                             |
| -------- | --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| light    | Narrow, low-ceremony work                           | `files_changed` + (`command_evidence` or `manual_rationale`)                                                    |
| standard | Bounded multi-step work                             | `files_changed` + `command_evidence` (verification artifacts)                                                   |
| deep     | High-stakes, multi-source, high-cost-of-being-wrong | `files_changed` + `command_evidence` + `evidence_scope` + `untested_regions` + `remaining_risks` + `execution_controls` + `rollback_policy` + `state.read_set` + `state.write_set` |

## Agent roles

- **Worker / Implementation agent**: Performs the task, writes the completion card, claims a fix status.
- **Verifier / Admission agent**: Read-only inspector. Validates schema, checks canonical consistency, recommends admission outcome. Does not edit the work product.
- **Orchestrator**: Dispatches tasks, routes handoffs, tracks trace events.

## Exit codes (CLI)

- `0`: Accepted
- `1`: Withheld / invalid / verification failed
- `2`: Usage or config error
