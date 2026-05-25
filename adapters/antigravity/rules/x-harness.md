# Antigravity x-harness Rules

## Default behavior

- Use `light` tier by default.
- Use `standard` for multi-step work.
- Use `deep` only for risk/control decisions.

## Completion rules

- Write a completion card before claiming completion.
- `claim.fix_status: fixed` is only a completion candidate; accepted success also requires `verification.status: passed`, `admission.outcome: success`, and `acceptance_status: accepted`.
- `acceptance_status: accepted` only when `admission.outcome: success`.
- Non-success outcomes are always `withheld`.
- Blocked/failed/skipped outcomes must include `handoff.next_action` and `handoff.owner`.

## Verifier rules

- The verifier is read-only.
- The verifier does not edit source files while verifying.
- PGV advice is advisory-only; it never overrides verify.

## Tiers

Use only `light`, `standard`, `deep`. Do not use `small`, `medium`, `large`.

## Workflows

- `workflows/x-harness-implementation.md`: Worker produces claim/evidence/card.
- `workflows/x-harness-verify.md`: Verifier runs read-only admission checks.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: antigravity-rules-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: antigravity-rules-contract -->
