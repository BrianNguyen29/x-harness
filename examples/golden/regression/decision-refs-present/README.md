# Golden Example: Decision Refs Present

A standard-tier completion card that lists at least one non-blank
entry in `context_alignment.decision_refs`. The fixture exercises the
positive branch of the safe-V1 decision_refs gate: when the array
contains at least one non-blank string, the admission layer emits no
advisory note and the card is admitted.

## Scenario

An agent records a small ADR-lite next to a task and references the
file from the completion card. The on-disk placeholder file lives at
`src/decisions/adr-001.md` so the entry resolves to a real path
under the fixture directory. The completion card is otherwise
minimal and meets the standard-tier evidence floor.

## Files

- `completion-card.yaml` — standard-tier card with
  `context_alignment.decision_refs` populated.
- `expected-verify-output.txt` — expected quiet summary from
  `xh examples verify` (outcome: success, accepted, 2 passed).
- `src/decisions/adr-001.md` — placeholder file referenced by
  `decision_refs`. Empty content is intentional; only path
  resolution is exercised.

## Expected outcome

```yaml
outcome: success
acceptance_status: accepted
checks: 2 passed, 0 failed
```

## Try it

```bash
xh examples verify --suite=regression --json
```

The regression suite must admit this fixture under the
`regression/decision-refs-present` name. The admission layer does
not block on the empty/non-empty state of `decision_refs`; the
advisory note is the only signal. The verify-layer
`--decision-enforce block` profile is not exercised by
`xh examples verify`, so this fixture intentionally stays in the
advisory-only happy path.

## What this fixture intentionally does NOT cover

- The verify-layer `decision_refs_missing` blocking predicate is
  wired to the `--decision-enforce block` flag and the
  `ci-strict` / `governed-deep` profiles. `xh examples verify` does
  not apply those flags, so a blocked-by-decision_refs fixture would
  not be exercised deterministically by this engine. Future slices
  may add a profile-driven blocked fixture.
- The card does not exercise the `xh decision record` or
  `xh decision link` flows; see `decision-link-flow` for that
  coverage.
