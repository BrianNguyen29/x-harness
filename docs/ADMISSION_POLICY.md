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
