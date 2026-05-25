# Harness Change Contract

Use this template for PRs that modify admission policy, schemas, templates, CLI verify, adapters, or skills.

## Component modified

- <cli|policy|schema|template|adapter|docs>

## Target failure mode

- <premature_done|weak_evidence|blocked_without_owner|token_bloat|adapter_drift>

## Predicted improvement

- <what should improve>

## Must preserve

- Verification is read-only.
- success is the only accepted outcome.
- failed/blocked/skipped/timeout/error are withheld.
- PGV is advisory-only.
- minimal mode remains lightweight.
- deep remains opt-in.

## Falsifying evaluation

- <test/example/doctor check that would prove this change harmful>

## Rollback plan

- <how to revert>

## Cost impact

default_token_impact: <none|low|medium|high>
runtime_impact: <none|low|medium|high>

<!-- BEGIN X-HARNESS MANAGED CONTRACT: harness-change-template-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: c101e078ad9bdfb5 -->

## Generated Handoff Contract

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

Required completion card fields:

- schema_version
- task_id
- tier
- owner
- accountable
- claim
- verification
- admission
- acceptance_status
- handoff

## Evidence Floor

- **light**: files_changed + (command_evidence or manual_rationale).
- **standard**: files_changed + command_evidence + done_checklist + prediction.
- **deep**: files_changed + command_evidence + evidence_scope_declared + untested_regions_declared + remaining_risks_declared + execution_controls_present + rollback_policy_present + done_checklist + prediction. Runtime-enforced: verification_artifacts, state.read_set, state.write_set.

## Strict Evidence Provenance

- verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.
- verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.

Completion cards use claim.fix_status as the canonical fix-status field. Subagent returns may use result.fix_status only in compatibility return payloads.

<!-- END X-HARNESS MANAGED CONTRACT: harness-change-template-contract -->
