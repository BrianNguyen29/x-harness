# Admission Policy

x-harness admission is fail-closed.

Success requires claim or completion card, evidence, owner, accountable, mapped success criteria, evidence floor, no unresolved blocker, stale ground resolved, verify invoked, and read-only verifier.

Reject success if canonical `claim.fix_status` is `partial` or `not_fixed`, if verification failed/skipped/blocked, if evidence is missing or weak, if stale ground remains, or if timeout/error occurred.

**Note:** `no_active_recovery` and `no_active_veto` are policy manifest entries; they are advisory documentation and not independently runtime-enforced predicates in admission.ts. Recovery routing is handled by `policies/recovery.yaml`.

## Evidence floor

The evidence floor depends on tier:

### light

Required: `files_changed` and one of `command_evidence` or `manual_rationale`.

### standard

Required: `files_changed`, `command_evidence`.
Recommended: `evidence_scope_declared`, `untested_regions_declared`.
Completion-card mode also requires `done_checklist` and a falsifiable `prediction`.

### deep

Required: `files_changed`, `command_evidence`, `evidence_scope_declared`, `untested_regions_declared`, `remaining_risks_declared`.
Also required: `execution_controls_present`, `rollback_policy_present`.
Runtime-enforced artifacts: `verification_artifacts`, `state.read_set`, and `state.write_set`.
Completion-card mode also requires `done_checklist` and a falsifiable `prediction`.

## Rejection conditions

- `approval_required_but_missing`: true for deep tasks with pending/missing human approval.
- `timeout`: true.
- `error`: true.

## Advisory metadata

- **`context_acknowledged`**: Optional boolean on the completion card. Missing or `false` triggers an advisory note but does **not** block admission.

## Source of truth

The TypeScript admission engine (`packages/cli/src/core/admission.ts`) is the runtime source of truth for admission decisions. `policies/admission.yaml` is a synchronized manifest and human-readable documentation of the policy. Changes to admission behavior must be made in the TypeScript engine first; the YAML should then be updated to match. The `doctor` command includes a `policy_drift` check that validates the YAML remains synchronized with the code.

## Policy file status

- **`admission.yaml`**: Synchronized manifest; validated by `doctor --policy-drift`. Not runtime-enforced.
- **`recovery.yaml`**: Runtime-enforced for recovery routing.
- **Other files in `policies/`**: Advisory or reserved unless explicitly documented as runtime-enforced.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: admission-evidence-floor -->
<!-- generated-by: x-harness -->
<!-- contract-hash: 8abd1e3360597776 -->

## Generated Admission Evidence Floor

## Evidence Floor

- **light**: files_changed + (command_evidence or manual_rationale).
- **standard**: files_changed + command_evidence + done_checklist + prediction.
- **deep**: files_changed + command_evidence + evidence_scope_declared + untested_regions_declared + remaining_risks_declared + execution_controls_present + rollback_policy_present + done_checklist + prediction. Runtime-enforced: verification_artifacts, state.read_set, state.write_set.

## Strict Evidence Provenance

- verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.
- verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.

Generated fix-status guidance:

Completion cards use claim.fix_status as the canonical fix-status field. Subagent returns may use result.fix_status only in compatibility return payloads.

<!-- END X-HARNESS MANAGED CONTRACT: admission-evidence-floor -->
