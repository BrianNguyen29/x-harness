# Generic x-harness Adapter

This adapter provides system-agnostic conventions for integrating x-harness into any agent environment.

## Setup

1. Copy [AGENTS.md](AGENTS.md) into your project root.
2. Ensure completion cards are written as `completion-card.yaml` before claiming completion.
3. Run the verify gate read-only using `check` (alias for `verify`):
   ```bash
   node packages/cli/dist/index.js check --card completion-card.yaml
   ```

## Beginner-Friendly Actions

| Action       | Alias for              | Description                                              |
| :----------- | :--------------------- | :------------------------------------------------------- |
| **`check`**  | `verify`               | Run read-only verification against a completion card      |
| **`prepare`** | `handoff readiness`   | Check if workspace is ready for agent task handoff        |
| **`recover`** | `recovery suggest`    | Get recovery playbook suggestions from errors or trace     |
| **`doctor`** | (standalone)           | Validate workspace health and configuration               |

## Workflow

1. **Dispatch**: Choose a tier (`light`, `standard`, `deep`) and generate a handoff template.
2. **Execute**: The worker performs the task and writes a completion card.
3. **Verify**: A read-only verifier checks the card against schemas and admission policy.
4. **Decide**: Only `outcome: success` + `acceptance_status: accepted` counts as admitted.

## Evidence floor

- **light**: `files_changed` + command evidence or manual rationale.
- **standard**: `files_changed` + command evidence.
- **deep**: `files_changed` + command evidence + scope + untested regions + remaining risks + rollback policy + execution controls.

## Policy file status

Policy files under `policies/` have the following runtime status:

- **`admission.yaml`**: Runtime-enforced. The verify gate reads this file and applies its rules.
- **`recovery.yaml`**: Runtime-enforced. Determines recovery routing for withheld outcomes.
- **Other policy files**: Advisory or reserved for future enforcement. Do not assume they are active unless documented here.
