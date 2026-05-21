# x-harness Verify Workflow

## Goal

Run read-only verification against a completion card and emit an admission outcome.

## Steps

1. **Receive completion card** from worker or orchestrator.
2. **Validate schema** against `schemas/completion-card.schema.json`.
3. **Check canonical consistency**:
   - `verification.status: passed` -> `claim.fix_status: fixed`
   - `acceptance_status: accepted` -> `admission.outcome: success`
   - Non-success outcomes -> `handoff.next_action` and `handoff.owner` present
4. **Check evidence**:
   - Non-light tiers require evidence.
   - Evidence quality is sufficient for the tier.
5. **Note PGV advice** as advisory-only. Do not let PGV override a passing core admission.
6. **Emit outcome**:
   - `success` + `accepted` if all checks pass.
   - `failed`, `blocked`, or `skipped` + `withheld` if any check fails.
7. **Write verify event** to trace if `--trace` is enabled.

## Constraints

- Read-only: do not edit source files.
- PGV is advisory-only.
- Only `success` + `accepted` counts as accepted completion.
