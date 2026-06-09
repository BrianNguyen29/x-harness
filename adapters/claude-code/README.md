# Claude Code Adapter

## Purpose

This adapter integrates `x-harness` with the **Claude Code** agent platform. It allows Claude Code to act as an implementation worker (drafting changes and producing a completion card) or as an admission verifier (performing read-only checks to authorize completion).

## Install

To install this adapter into your repository, run the following command:

```bash
cp -r adapters/claude-code/* .
```

This copies the `CLAUDE.md` instructions, the `agents/` role definitions, and the custom verification `skills/` directly to your project root where Claude Code can discover and execute them.

## Files included

- `CLAUDE.md`: Authoritative instructions read by Claude Code on project initialization.
- `agents/implementation-worker.md`: Guide describing the worker role's responsibilities for implementation and completion card creation.
- `agents/admission-verifier.md`: Guide describing the read-only inspector role's verification instructions.
- `skills/verify/SKILL.md`: Verification skill instruction.
- `skills/handoff/SKILL.md`: Skills instruction for handoff creation.
- `skills/recovery/SKILL.md`: Skills instruction for task recovery.

## Beginner-Friendly Actions

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
| **`init`**    | (standalone)            | Install core harness assets, schemas, policies, and adapters |
| **`add`**     | (standalone)            | Add a metadata helper file for compatibility modes     |
| **`run`**     | (standalone)            | Run a built-in workflow recipe                         |
| **`ci`**      | (standalone)            | Run the built-in CI workflow                           |

**Slash commands for agent adapters:**

`/xh:<command>` is agent-chat slash notation; it is not a shell binary or filesystem path.

| Namespaced       | Maps to CLI    |
| :--------------- | :------------- |
| `/xh:start`      | `xh start`     |
| `/xh:learn`      | `xh learn`     |
| `/xh:quick`      | `xh quick`     |
| `/xh:check`      | `xh check`     |
| `/xh:prepare`    | `xh prepare`   |
| `/xh:recover`    | `xh recover`   |
| `/xh:doctor`     | `xh doctor`    |
| `/xh:actions`    | `xh actions`   |
| `/xh:status`     | `xh status`    |
| `/xh:reset`      | `xh reset`     |
| `/xh:init`       | `xh init`      |
| `/xh:add`        | `xh add`       |
| `/xh:run`        | `xh run`       |
| `/xh:ci`         | `xh ci`        |
| `/xh:verify`     | `xh verify`    |
| `/xh:intake`     | `xh intake`    |
| `/xh:handoff`    | `xh handoff`   |
| `/xh:decision`   | `xh decision`  |
| `/xh:boundary`   | `xh boundary`  |
| `/xh:context`    | `xh context`   |
| `/xh:packet`     | `xh packet`    |
| `/xh:examples`   | `xh examples`  |
| `/xh:trace`      | `xh trace`     |
| `/xh:report`     | `xh report`    |

Examples with args and subcommands:
- `/xh:verify --card completion-card.yaml --json`
- `/xh:intake contract --from issue.md`
- `/xh:context manifest check --manifest .x-harness/context-manifest.yaml --json`

Use `/xh:<command>` as the preferred shortcut notation in agent chat. The space-delimited `/xh <action>` and legacy `/xh-check`, `/xh-prepare`, `/xh-recover`, `/xh-doctor`, `/xh-actions`, `/xh-status`, `/xh-reset` styles remain supported for compatibility.

## Workflow

1. **Orchestrator** (user or parent agent) writes a task using a handoff template.
2. **Implementation Worker** (Claude Code) performs edits, tests the changes, and generates the `completion-card.yaml`.
3. **Admission Verifier** (Claude Code) runs read-only verification. In this repository, use:
   ```bash
   xh check --card completion-card.yaml --strict
   # or: node packages/cli/dist/index.js check --card completion-card.yaml --strict
   ```
4. **Outcome**: Accepted (status code `0`) or Withheld (status code `1`). If withheld, the next actions are routed based on recovery rules.

## Constraints

- **Verifier is read-only**: The verifier must not write or edit files to fix validation issues during the verification stage.
- **Fail-closed**: Any outcome other than success is withheld (`withheld`).
- **PGV is advisory-only**: Policy Governance recommendations never override verify gate outcomes.
- **No heavy runtime required**: No background daemon, server, database, or MCP is needed.

## When to use

Use this adapter when you are running **Claude Code** as your primary developer or verification assistant. It instructs the LLM on how to follow `x-harness` completion rules and equips it with context-sensitive verification skills.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: claude-readme-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: ec6438371a039c93 -->

## Generated Adapter Contract

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

## Evidence Floor

- **light**: files_changed + (command_evidence or manual_rationale).
- **standard**: files_changed + command_evidence + done_checklist + prediction.
- **deep**: files_changed + command_evidence + evidence_scope_declared + untested_regions_declared + remaining_risks_declared + execution_controls_present + rollback_policy_present + done_checklist + prediction. Runtime-enforced: verification_artifacts, state.read_set, state.write_set.

## Strict Evidence Provenance

- verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.
- verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.

<!-- END X-HARNESS MANAGED CONTRACT: claude-readme-contract -->
