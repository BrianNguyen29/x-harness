# Cursor Adapter

## Purpose

This adapter integrates `x-harness` with the **Cursor** IDE agent platform. It provides Cursor with rules (`.cursorrules` or `.mdc` formats) to guide Cursor's composer or chat agent when working inside the repository.

## Install

To install this adapter in your project, copy the MDC rules file to the `.cursor/rules/` directory in your workspace:

```bash
mkdir -p .cursor/rules && cp adapters/cursor/rules/x-harness.mdc .cursor/rules/
```

Once copied, Cursor will automatically detect and apply these rules to all agent sessions inside the workspace.

## Files included

- `rules/x-harness.mdc`: The main Cursor rule file. It uses frontmatter configurations (`alwaysApply: true`) to ensure Cursor always follows `x-harness` protocols.

## Beginner-Friendly Actions

| Action       | Alias for              | Description                                              |
| :----------- | :--------------------- | :------------------------------------------------------- |
| **`start`**  | (standalone)           | Guided onboarding: doctor, examples verify, init wizard, next steps |
| **`learn`**  | (standalone)           | Read-only concept tour for beginners                     |
| **`quick`**  | (standalone)           | Read-only next-action recommender for newcomers          |
| **`check`**  | `verify`               | Run read-only verification against a completion card      |
| **`prepare`** | `handoff readiness`   | Check if workspace is ready for agent task handoff        |
| **`recover`** | `recovery suggest`    | Get recovery playbook suggestions from errors or trace     |
| **`doctor`** | (standalone)           | Validate workspace health and configuration               |
| **`actions`** | (standalone)           | List all beginner-friendly actions                        |
| **`status`** | `report` (no --metrics) | Show trace summary or card metrics                      |
| **`reset`**  | `clean --tmp --force` | Clean generated harness state (requires --confirm)        |
| **`init`**   | (standalone)           | Install core harness assets, schemas, policies, and adapters |
| **`add`**    | (standalone)           | Add a metadata helper file for compatibility modes       |
| **`run`**    | (standalone)           | Run a built-in workflow recipe                           |
| **`ci`**     | (standalone)           | Run the built-in CI workflow                             |

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

1. **Developer Handoff**: A task is dispatched to the Cursor Composer or agent.
2. **Work Phase**: The agent performs the required changes, implements tests, and documents findings.
3. **Card Creation**: The agent creates or updates the `completion-card.yaml` detailing claims and evidence.
4. **Verification**: In this repository, the developer or agent triggers `xh check` (or `node packages/cli/dist/index.js check`) to check the card.
5. **Admission**: Completion is only accepted when verification succeeds.

## Constraints

- **Verifier is read-only**: The verification agent must not write or edit files to fix validation issues during the verification stage. Run `xh check` to perform read-only verification.
- **Advisory-only rule**: Cursor rules act as a guide for the agent's behavior; they do not automatically execute verification or enforce policies.
- **Strict Tier Labels**: The rules instruct Cursor to use only the canonical tiers (`light`, `standard`, `deep`).
- **No heavy runtime required**: Fully local, offline-first.

## When to use

Use this adapter if you or your team use the **Cursor** editor as the primary IDE or agent interface. It ensures Cursor agents adhere to the read-only verification model, fill out completion cards correctly, and use the correct handoff tiers.

<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- context-hash: b40c242bdf8b2bdb -->

# x-harness Canonical Context

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

## Source-of-Truth Reading Order

The managed context block in AGENTS.md is authoritative. Files are read in this order:

1. AGENTS.md (managed block)
1. X_HARNESS.md
1. policies/admission.yaml
1. policies/recovery.yaml
1. policies/intake.yaml
1. schemas/completion-card.schema.json

## Rules

### Completion is admitted, not claimed
Agents may propose completion but cannot self-admit. A completion card with `claim.fix_status: fixed` is only a completion candidate. Compatibility subagent returns may use `result.fix_status`.

### Verifier is read-only
The verifier may inspect files, tasks, stories, templates, returns, evidence, diffs, command output, and trace events. It must not edit source files or repair the work product while verifying.

### Success is the only accepted outcome
`admission.outcome: success` and `acceptance_status: accepted` are required for admission. All other outcomes are withheld.

### Canonical tiers
Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

### PGV is advisory-only
Pre-gate validation (PGV) advice never overrides the verify gate and never grants admission authority by default.

<!-- END X-HARNESS MANAGED CONTEXT -->

<!-- BEGIN X-HARNESS MANAGED CONTRACT: cursor-readme-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: cursor-readme-contract -->
