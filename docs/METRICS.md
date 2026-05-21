# Metrics

`npx x-harness report --metrics` computes deterministic local metrics for a completion card without external services.

## Metric categories

### verification_strength

- `command_evidence_count`: number of verification artifacts.
- `oracle_kinds`: distinct artifact kinds (e.g., unit_test, typecheck).
- `untested_regions_count`: declared untested regions.
- `remaining_risks_count`: declared remaining risks.

### state_consistency

- `owner_present`: owner field is non-empty.
- `accountable_present`: accountable field is non-empty.
- `files_changed_present`: files_changed or claim.evidence is non-empty.
- `admission_mapping_valid`: admission outcome aligns with acceptance_status.

### recovery_ability

- `blocked_has_next_action`: blocked/failed cards have handoff.next_action.
- `blocked_has_owner`: blocked/failed cards have handoff.owner.
- `recovery_route_present`: both next_action and owner are present for blocked/failed.

### replayability

- `completion_card_present`: task_id is present.
- `input_card_hash_present`: SHA-256 of the card was computed.
- `policy_hash_present`: SHA-256 of policies/admission.yaml was computed.

### cost

- `default_context_class`: `low` for light, `medium` for standard, `high` for deep.
- `verify_runtime_ms`: local wall-clock time of verify/metrics computation.

## Classes

Metrics use simple classes rather than complex scoring:

- `low | medium | high`
- `weak | adequate | strong`
- `present | missing`

## Denominator warning

Every metrics report includes:

> Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.
