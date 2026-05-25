# x-harness Implementation Workflow

## Goal

Produce a completion card for a task.

## Steps

1. **Receive task** from orchestrator with tier and scope.
2. **Execute work** within scope. Do not expand scope without escalation.
3. **Gather evidence**:
   - Commands ran
   - Files changed
   - Key outputs
4. **Write completion card** as `completion-card.yaml`:
   - `schema_version`, `task_id`, `tier`
   - `owner`, `accountable`
   - `claim.fix_status`, `claim.summary`, `claim.evidence`
   - `verification.status`, `verification.checks`
   - `admission.outcome`
   - `acceptance_status`
   - `handoff.next_action`, `handoff.owner`
5. **Self-check**:
   - Treat `claim.fix_status: fixed` as a candidate only; accepted success also requires `verification.status: passed`, `admission.outcome: success`, and `acceptance_status: accepted`.
   - If blocked, ensure `handoff` is populated.
6. **Submit** for verification.

## Constraints

- Use `light` by default.
- PGV advice is advisory-only.
- Do not claim completion without a completion card.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: antigravity-implementation-workflow-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: antigravity-implementation-workflow-contract -->
