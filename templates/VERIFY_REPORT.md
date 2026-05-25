# Verify Report Template

## Scope

- Task ID: `TASK-___`
- Tier: `light|standard|deep`
- Claim summary: \_\_\_

## Files inspected

- `<file path>`: `<purpose>`
- `<file path>`: `<purpose>`

## Checks

- [ ] Schema validation: completion card is structurally valid.
- [ ] Canonical consistency: `verification.status: passed` implies `claim.fix_status: fixed`.
- [ ] Admission alignment: `acceptance_status: accepted` only when `admission.outcome: success`.
- [ ] Handoff completeness: non-success outcomes include `next_action` and `owner`.
- [ ] Evidence floor: all tiers require `evidence.files_changed`; light requires `command_evidence` or `manual_rationale`; standard/deep require `command_evidence`; standard/deep completion cards require `done_checklist` and `prediction`; deep requires scoped artifacts, risk/rollback/execution controls, and `state.read_set/write_set`.
- [ ] PGV review: PGV risk noted as advisory-only; does not block by default.

## Outcome

- `success|failed|blocked|skipped|timeout|error`
- Acceptance: `accepted|withheld`

## Blockers

List any blockers found:

- ***

## Handoff

- Next action: \_\_\_
- Owner: \_\_\_

## Rules

- The verifier is read-only; it does not edit source files to repair findings.
- Only `admission.outcome: success` + `acceptance_status: accepted` counts as accepted completion.
- All other outcomes are withheld.
- PGV advice is advisory-only and never grants admission authority by default.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: verify-report-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: 52e5863e26fa46c0 -->

## Generated Verify Report Contract

- Evidence floor light: files_changed + (command_evidence or manual_rationale).
- Evidence floor standard: files_changed + command_evidence + done_checklist + prediction.
- Evidence floor deep: files_changed + command_evidence + evidence_scope_declared + untested_regions_declared + remaining_risks_declared + execution_controls_present + rollback_policy_present + done_checklist + prediction; runtime-enforced: verification_artifacts, state.read_set, state.write_set.
- Strict provenance: verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.
- Strict provenance: verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.
- Accepted completion requires `admission.outcome: success` and `acceptance_status: accepted`.

<!-- END X-HARNESS MANAGED CONTRACT: verify-report-contract -->
