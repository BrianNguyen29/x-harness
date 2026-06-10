# x-harness Adapter Guide

x-harness adapters are thin conventions that make the harness usable inside specific agent platforms. Adapters are optional: the CLI is the source of truth.

## Beginner-friendly actions (primary interface)

| Action        | Alias for               | Description                                            |
| :------------ | :---------------------- | :----------------------------------------------------- |
| **`start`**   | (standalone)            | Guided onboarding: doctor, examples verify, init wizard, next steps |
| **`learn`**   | (standalone)            | Read-only concept tour for beginners                   |
| **`quick`**   | (standalone)            | Read-only next-action recommender for newcomers        |
| **`check`**   | `verify`                | Run read-only verification against a completion card   |
| **`prepare`** | `handoff readiness`     | Check if workspace is ready for agent task handoff     |
| **`recover`** | `recovery suggest`      | Get recovery playbook suggestions from errors or trace |
| **`doctor`**  | (standalone)            | Validate workspace health and configuration            |
| **`actions`** | (standalone)            | List all beginner-friendly actions                     |
| **`status`**  | `report` (no --metrics) | Show trace summary or card metrics                     |
| **`reset`**   | `clean --tmp --force`   | Clean generated harness state (requires --confirm)     |
| **`init`**    | (standalone)            | Install core harness assets, schemas, and policies. Adapters require `--full` or `--adapters` |
| **`add`**     | (standalone)            | Add a metadata helper file for compatibility modes     |
| **`run`**     | (standalone)            | Run a built-in workflow recipe                         |
| **`ci`**      | (standalone)            | Run the built-in CI workflow                           |

**Slash commands for agent adapters:** Use `/xh:<command>` in agent chat (for example, `/xh:check`, `/xh:doctor`). The space-delimited `/xh <action>` and legacy `/xh-check`, `/xh-prepare`, `/xh-recover`, `/xh-doctor`, `/xh-actions`, `/xh-status`, `/xh-reset` styles remain supported for compatibility. Beta commands such as `packet` are available via the CLI but are not part of the beginner stable adapter interface.

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
- Key files: `README.md`, `verify-agent.md`, `agents/x-harness-verify.md`, `agents/x-harness-recover.md`, `orchestrator_append.example.md`, `opencode.example.json`, `opencode.verify.example.json`
- Workflow: orchestrator dispatch -> worker completion -> verify-agent read-only gate.

### Antigravity

- Location: `adapters/antigravity/`
- Use when: You use Antigravity as the agent platform.
- Key files: `rules/x-harness.md`, `workflows/x-harness-implementation.md`, `workflows/x-harness-verify.md`, `workflows/x-harness-recover.md`

## Tier selection quick reference

| Tier     | When to use                                         | Evidence required                                                                                             |
| -------- | --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| light    | Narrow, low-ceremony work                           | `files_changed` + (`command_evidence` or `manual_rationale`)                                                    |
| standard | Bounded multi-step work                             | `files_changed` + `command_evidence` + `done_checklist` + `prediction`                                          |
| deep     | High-stakes, multi-source, high-cost-of-being-wrong | `files_changed` + `command_evidence` + `evidence_scope` + `untested_regions` + `remaining_risks` + `execution_controls` + `rollback_policy` + `state.read_set` + `state.write_set` |

## Agent roles

- **Worker / Implementation agent**: Performs the task, writes the completion card, claims a fix status.
- **Verifier / Admission agent**: Read-only inspector. Validates schema, checks canonical consistency, recommends admission outcome. Does not edit the work product.
- **Orchestrator**: Dispatches tasks, routes handoffs, tracks trace events.

## Exit codes (CLI)

- `0`: Accepted
- `1`: Withheld / invalid / verification failed
- `2`: Usage or config error
