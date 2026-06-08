# Admission Policy

x-harness admission is fail-closed.

Success requires claim or completion card, evidence, owner, accountable, mapped success criteria, evidence floor, no unresolved blocker, stale ground resolved, verify invoked, and read-only verifier.

Reject success if canonical `claim.fix_status` is `partial` or `not_fixed`, if verification failed/skipped/blocked, if evidence is missing or weak, if stale ground remains, or if timeout/error occurred.

**Note:** `no_active_recovery` is an advisory invariant in the policy manifest. Because x-harness uses a stateless, file-first architecture, it is trivially satisfied and is not independently runtime-enforced in the admission engines. Recovery routing is handled by `policies/recovery.yaml`.

## Evidence floor

The canonical evidence floor is defined in the generated contract block below. In summary:

- **light**: `files_changed` and one of `command_evidence` or `manual_rationale`.
- **standard**: `files_changed` + `command_evidence` + `done_checklist` + `prediction`.
- **deep**: `files_changed` + `command_evidence` + `evidence_scope_declared` + `untested_regions_declared` + `remaining_risks_declared` + `execution_controls_present` + `rollback_policy_present` + `done_checklist` + `prediction`. Runtime-enforced: `verification_artifacts`, `state.read_set`, `state.write_set`.

`done_checklist` is cross-checked against declared evidence, prediction, verification artifacts, and state where applicable. Optional checklist blocks on `light` cards are also checked for honesty if present. In strict or deep mode, `read_write_sets_declared: true` must be backed by `state.read_set` and `state.write_set`.

> The exact floor definitions below are managed by x-harness and take precedence over any human-written summary.

## Rejection conditions

- `approval_required_but_missing`: true for deep tasks with pending/missing human approval.
- `timeout`: true.
- `error`: true.

## Advisory metadata

- **`context_acknowledged`**: Optional boolean on the completion card. Missing or `false` triggers an advisory note but does **not** block admission.

## Source of truth

The Go admission engine (`internal/admission`) is the native implementation, and the TypeScript engine (`packages/cli/src/core/admission.ts`) remains the compatibility baseline during the dual-run window. `policies/admission.yaml` is a synchronized manifest and human-readable documentation of the policy. Changes to admission behavior must keep Go, TypeScript compatibility, fixtures, and the YAML manifest aligned. The `doctor` and parity checks validate schema/policy health and Go-vs-TypeScript drift.

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
