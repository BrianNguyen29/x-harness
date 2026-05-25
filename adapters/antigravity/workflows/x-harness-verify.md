# x-harness Verify Workflow

## Goal

Run read-only verification against a completion card and emit an admission outcome.

## Steps

1. **Receive completion card** from worker or orchestrator.
2. **Validate schema** against `schemas/completion-card.schema.json`.
3. **Check canonical consistency**:
   - `verification.status: passed` -> `claim.fix_status: fixed`
   - `acceptance_status: accepted` -> `admission.outcome: success`
   - Non-success outcomes -> `handoff.next_action` and `handoff.owner` present
4. **Check evidence**:
   - All tiers require non-empty `evidence.files_changed`.
   - Light requires `command_evidence` or `manual_rationale`.
   - Standard/deep require `command_evidence`; standard/deep completion cards also require `done_checklist` and `prediction`.
   - Deep requires scoped `verification_artifacts`, `untested_regions`, `remaining_risks`, `rollback_policy`, `execution_controls`, and `state.read_set/write_set`.
5. **Note PGV advice** as advisory-only. Do not let PGV override a passing core admission.
6. **Emit outcome**:
   - `success` + `accepted` if all checks pass.
   - `failed`, `blocked`, or `skipped` + `withheld` if any check fails.
7. **Write verify event** to trace if `--trace` is enabled.

## Constraints

- Read-only: do not edit source files.
- PGV is advisory-only.
- Only `success` + `accepted` counts as accepted completion.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: antigravity-verify-workflow-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: antigravity-verify-workflow-contract -->
