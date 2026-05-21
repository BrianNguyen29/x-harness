# Verify Report Template

## Scope

- Task ID: `TASK-___`
- Tier: `light|standard|deep`
- Claim summary: ___

## Files inspected

- `<file path>`: `<purpose>`
- `<file path>`: `<purpose>`

## Checks

- [ ] Schema validation: completion card is structurally valid.
- [ ] Canonical consistency: `verification.status: passed` implies `claim.fix_status: fixed`.
- [ ] Admission alignment: `acceptance_status: accepted` only when `admission.outcome: success`.
- [ ] Handoff completeness: non-success outcomes include `next_action` and `owner`.
- [ ] Evidence presence: non-light tiers require evidence.
- [ ] PGV review: PGV risk noted as advisory-only; does not block by default.

## Outcome

- `success|failed|blocked|skipped|timeout|error`
- Acceptance: `accepted|withheld`

## Blockers

List any blockers found:

- ___

## Handoff

- Next action: ___
- Owner: ___

## Rules

- The verifier is read-only; it does not edit source files to repair findings.
- Only `admission.outcome: success` + `acceptance_status: accepted` counts as accepted completion.
- All other outcomes are withheld.
- PGV advice is advisory-only and never grants admission authority by default.
