# Metrics

`node packages/cli/dist/index.js report --metrics` computes deterministic local metrics for a completion card without external services in this repository.

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

## Report accounting sections

Reports include denominator-safe accounting sections. These are intentionally scoped to the analyzed events or card and do not imply completeness when the full denominator is unknown.

### verify_event_accounting

- `total_trace_events` (trace report) or `cards_analyzed` (metrics report): count of events/cards analyzed.
- Note: explicitly states the scope is limited to traced events or the single card.

### task_lifecycle_accounting

- `admitted`: count of accepted events/cards.
- `withheld`: count of withheld events/cards.
- Note: covers only events present in the trace log or the analyzed card.

### admission_accounting

- `accepted`: count of accepted events/cards.
- `total_trace_events` or `total_analyzed`: denominator.
- Note: admission requires `outcome=success`; non-success outcomes are withheld.

### withheld_accounting

- Breakdown by `failed`, `blocked`, `skipped`, `timeout`, `error`.
- Note: breakdown is only as complete as the trace event set or the single card.

### unknown_or_unlinked_events

- `count`: events with missing or unrecognized `outcome`/`acceptance_status`.
- Note: for single-card metrics analysis this is marked as not applicable.

## Denominator warning

Every metrics report includes:

> Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.
