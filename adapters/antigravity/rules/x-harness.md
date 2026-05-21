# Antigravity x-harness Rules

## Default behavior

- Use `light` tier by default.
- Use `standard` for multi-step work.
- Use `deep` only for risk/control decisions.

## Completion rules

- Write a completion card before claiming completion.
- `fix_status: fixed` requires `verification.status: passed`.
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

## Legacy

Missions under `missions/` are preserved for compatibility.
