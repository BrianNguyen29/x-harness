# Generic x-harness Adapter

This adapter provides system-agnostic conventions for integrating x-harness into any agent environment.

## Setup

1. Copy [AGENTS.md](AGENTS.md) into your project root.
2. Ensure completion cards are written as `completion-card.yaml` before claiming completion.
3. Run the verify gate read-only using `check` (alias for `verify`):
   ```bash
   xh check --card completion-card.yaml --strict
   # or: node packages/cli/dist/index.js check --card completion-card.yaml --strict
   ```

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

1. **Dispatch**: Choose a tier (`light`, `standard`, `deep`) and generate a handoff template.
2. **Execute**: The worker performs the task and writes a completion card.
3. **Verify**: A read-only verifier checks the card against schemas and admission policy.
4. **Decide**: Only `outcome: success` + `acceptance_status: accepted` counts as admitted.

## Evidence floor

- **light**: `files_changed` + (`command_evidence` or `manual_rationale`).
- **standard**: `files_changed` + `command_evidence` + `done_checklist` + `prediction`.
- **deep**: `files_changed` + `command_evidence` + `evidence_scope_declared` + `untested_regions_declared` + `remaining_risks_declared` + `execution_controls_present` + `rollback_policy_present` + `done_checklist` + `prediction`. Runtime-enforced: `verification_artifacts`, `state.read_set`, `state.write_set`.

## Policy file status

Policy files under `policies/` have the following runtime status:

- **`admission.yaml`**: Synchronized manifest; validated by `doctor` and parity checks. Admission behavior is enforced by the Go engine with TypeScript compatibility during the dual-run window.
- **`recovery.yaml`**: Runtime-enforced. Determines recovery routing for withheld outcomes.
- **Other policy files**: Advisory or reserved for future enforcement. Do not assume they are active unless documented here.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: generic-adapter-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: generic-adapter-contract -->
