# Antigravity Adapter

## Purpose

This adapter integrates `x-harness` with the **Antigravity** agent platform. It provides the constraints, rules, and workflows to govern task execution and verification for Antigravity-driven agents.

## Install

To install this adapter in your project, copy the configuration files and workflows to your repository:

```bash
cp -r adapters/antigravity/* .
```

This will place the rules under `rules/` and execution workflows under `workflows/` in your workspace, allowing your Antigravity agent system to discover and follow them.

## Files included

- `rules/x-harness.md`: Core rules for Antigravity agents defining handoff tiers, completion invariants, and verifier requirements.
- `workflows/x-harness-implementation.md`: Workflow detailing the implementation stage and completion card generation.
- `workflows/x-harness-verify.md`: Workflow for running the read-only verify gate.
- `workflows/x-harness-recover.md`: Detailed guidance on handling task failures and routing blocked actions.

## Beginner-Friendly Actions

| Action        | Alias for               | Description                                            |
| :------------ | :---------------------- | :----------------------------------------------------- |
| **`prepare`** | `handoff readiness`     | Check if workspace is ready for agent task handoff     |
| **`check`**   | `verify`                | Run read-only verification against a completion card   |
| **`recover`** | `recovery suggest`      | Get recovery playbook suggestions from errors or trace |
| **`doctor`**  | (standalone)            | Validate workspace health and configuration            |
| **`actions`** | (standalone)            | List all beginner-friendly actions                     |
| **`status`**  | `report` (no --metrics) | Show trace summary or card metrics                     |
| **`reset`**   | `clean --tmp --force`   | Clean generated harness state (requires --confirm)     |

**Slash commands for agent adapters:** `/xh-check`, `/xh-prepare`, `/xh-recover`, `/xh-doctor`, `/xh-actions`, `/xh-status`, `/xh-reset`

## Workflow

1. **Initiate Task**: Dispatch a task utilizing one of the tiered handoff templates.
2. **Execute & Test**: The implementation agent performs changes, tests execution, and writes a `completion-card.yaml`.
3. **Trigger Verify Gate**: Run the verification utility in read-only mode:
   ```bash
   node packages/cli/dist/index.js check --card completion-card.yaml --strict
   # or: node packages/cli/dist/index.js verify --card completion-card.yaml --strict
   ```
4. **Resolution**: If verification succeeds, mark as `accepted`. If it fails, routing guidelines from `x-harness-recover.md` are executed based on the blocking predicates.

## Constraints

- **Strict Read-Only Verification**: The verifier must not write files or attempt to fix compilation/logical errors during the check phase.
- **Strict Tiers**: Antigravity runs must strictly use `light`, `standard`, and `deep` tiers.
- **Fail-Closed**: Non-success outcomes are always withheld (`withheld`).
- **No heavy runtime required**: Fully local, offline-first.

## When to use

Use this adapter when running the **Antigravity** agent framework to ensure that its development loops, verification checks, and recovery routines align seamlessly with the repository's x-harness policies.

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

<!-- BEGIN X-HARNESS MANAGED CONTRACT: antigravity-readme-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: antigravity-readme-contract -->
