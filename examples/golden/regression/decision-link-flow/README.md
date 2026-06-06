# Golden Example: Decision Link Flow

A standard-tier completion card that mirrors the on-disk shape of a
real decision record + linked card workflow. The card carries a
slug id (`adr-001`) in `context_alignment.decision_refs`; the
corresponding decision record lives next to it in
`decisions/adr-001.yaml` so the regression check exercises both
halves of the safe-V1 decision memory flow.

## Scenario

An agent runs `xh decision record --id adr-001 --decision ...
--rationale ...` to persist a decision record under
`decisions/adr-001.yaml`, then runs
`xh decision link --card <card> --decision adr-001` to append the
slug to `context_alignment.decision_refs`. This fixture is a
hand-authored snapshot of the post-link card so the regression
suite can verify admission without invoking the live CLI commands.

## Files

- `completion-card.yaml` — standard-tier card with
  `context_alignment.decision_refs: [adr-001]`. This is the
  post-link shape; `xh decision link` appends the slug in-place
  using exact-string deduplication.
- `expected-verify-output.txt` — expected quiet summary from
  `xh examples verify` (outcome: success, accepted, 2 passed).
- `decisions/adr-001.yaml` — the on-disk decision record. Shape
  matches `schemas/decision-record.schema.json` and the output of
  `xh decision record` for the same id.
- `README.md` — this file.

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

The regression suite must admit this fixture under
`regression/decision-link-flow`. Admission only inspects
`context_alignment.decision_refs`; the linked record file is
documentation for the reader and is not validated by
`xh examples verify`.

## What this fixture intentionally does NOT cover

- The fixture does not invoke `xh decision record` or
  `xh decision link` at runtime. The card and the record are
  static artifacts authored by hand. A future slice may add a
  fixture that runs the live CLI to write a real record, but that
  would require a scratch directory and would be more expensive
  to maintain than this static snapshot.
- The fixture does not exercise the `xh decision list`,
  `xh decision query`, or `xh decision affected` commands. Those
  are covered by unit tests in
  `internal/cli/decision_test.go` and
  `packages/cli/tests/decision.test.ts`.
- The fixture does not exercise the verify-layer
  `decision_refs_missing` blocking predicate; see
  `decision-refs-empty-advisory` for the advisory path and the
  proposal notes for the follow-up profile-driven blocked
  fixture.
