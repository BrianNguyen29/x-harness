# Admission Policy

x-harness admission is fail-closed.

Success requires claim or completion card, evidence, owner, accountable, mapped success criteria, evidence floor, no unresolved blocker, stale ground resolved, no active recovery, no active veto, verify invoked, and read-only verifier.

Reject success if `fix_status` is `partial` or `not_fixed`, if verification failed/skipped/blocked, if evidence is missing or weak, if stale ground remains, if active recovery remains, if unresolved questions remain, or if timeout/error occurred.

## Evidence floor

The evidence floor depends on tier:

### light

Required: `files_changed` and one of `command_evidence` or `manual_rationale`.

### standard

Required: `files_changed`, `command_evidence`.
Recommended: `evidence_scope_declared`, `untested_regions_declared`.

### deep

Required: `files_changed`, `command_evidence`, `evidence_scope_declared`, `untested_regions_declared`, `remaining_risks_declared`.
Also required: `execution_controls_present`, `rollback_policy_present`.

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
