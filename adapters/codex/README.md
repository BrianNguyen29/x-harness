# Codex Adapter

## Purpose

This adapter integrates `x-harness` with **Codex**, the agentic coding feature in OpenAI's ChatGPT and related platforms. Codex reads repo-root `AGENTS.md` as its primary instruction source and has no third-party custom slash command API.

## Setup

1. Copy [AGENTS.md](AGENTS.md) into your project root.
2. No additional configuration files are required. Codex discovers `AGENTS.md` automatically when it loads the repository.

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

**Agent-chat notation:**

`/xh:<command>` is agent-chat instruction notation; it is not a shell binary or filesystem path, and it is not a Codex slash command. When you need to invoke x-harness from within Codex, use the shell command `xh <command>` instead.

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

1. **Prepare**: Ensure `AGENTS.md` is present in the repository root.
2. **Dispatch**: Choose a tier (`light`, `standard`, `deep`) and generate a handoff template.
3. **Execute**: The worker performs the task and writes a completion card.
4. **Verify**: A read-only verifier checks the card against schemas and admission policy.
5. **Decide**: Only `admission.outcome: success` + `acceptance_status: accepted` counts as admitted.

## Constraints

- **Verifier is read-only**: The verifier inspects files and evidence; it does not edit source files to fix findings while verifying.
- **Strict Tiers**: Ensure dispatches strictly declare `light`, `standard`, or `deep` tiers.
- **Fail-Closed**: Non-success outcomes are always withheld (`withheld`).
- **No heavy runtime required**: Fully local, offline-first.

## When to use

Use this adapter when your primary agent platform is **Codex** and you want x-harness conventions delivered through the standard repo-root `AGENTS.md` mechanism.

<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- generated-at: 2026-05-25T03:35:44.539Z -->
<!-- context-hash: 8817d535c4e04a79 -->

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
The verifier may inspect files, evidence, diffs, and trace events. It must not edit source files or repair the work product while verifying.

### Success is the only accepted outcome
`admission.outcome: success` and `acceptance_status: accepted` are required for admission. All other outcomes are withheld.

### Canonical tiers
Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

### PGV is advisory-only
Pre-gate validation (PGV) advice never overrides the verify gate and never grants admission authority by default.

<!-- END X-HARNESS MANAGED CONTEXT -->

<!-- BEGIN X-HARNESS MANAGED CONTRACT: codex-readme-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: codex-readme-contract -->
