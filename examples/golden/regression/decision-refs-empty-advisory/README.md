# Golden Example: Decision Refs Empty (Advisory)

A standard-tier completion card whose
`context_alignment.decision_refs` is present but empty. The
fixture exercises the safe-V1 advisory path: the admission layer
emits a top-level note ("context_alignment.decision_refs is empty")
but the note is advisory-only and never blocks admission. The
verify-layer `decision_refs_missing` blocking predicate is wired
to `--decision-enforce block` and the `ci-strict` /
`governed-deep` profiles, which `xh examples verify` does not
apply; the regression suite therefore exercises the advisory
branch, not the blocking branch.

## Scenario

A standard-tier agent submits a completion card that is otherwise
valid but has not yet been linked to any decision record. The
admission layer should accept the card and surface the empty-array
advisory note so the reader knows the gate is silent at the
admission layer. The note wording makes it explicit that
admission acceptance is not decision correctness.

## Files

- `completion-card.yaml` — standard-tier card with
  `context_alignment.decision_refs: []`. The
  `product_contract_refs` field carries a real file reference so
  the context_alignment block is not treated as fully empty by
  future readers; this is documentation hygiene and does not
  affect the admission result.
- `expected-verify-output.txt` — expected quiet summary from
  `xh examples verify` (outcome: success, accepted, 2 passed).
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
`regression/decision-refs-empty-advisory`. The Go and TypeScript
admission layers both emit the empty advisory note in the notes
section; the regression compare step only checks
`outcome` and `acceptance_status` from the expected snapshot, so
the note does not affect the comparison.

## What this fixture intentionally does NOT cover

- The verify-layer `decision_refs_missing` blocking predicate
  (predicate id `decision_refs_missing`,
  `withheld_reason.class == "decision_refs_missing"`) is wired to
  `xh verify --decision-enforce block` and the `ci-strict` /
  `governed-deep` profiles. `xh examples verify` does not apply
  those flags, so a blocked-by-decision_refs fixture would not be
  exercised deterministically by the regression engine. This
  fixture therefore exercises only the advisory branch.
- The light-tier path is silent: a light-tier card with an empty
  `decision_refs` emits no advisory note. The light-tier
  behavior is covered by the safe-V1 unit tests in
  `internal/admission/admission_test.go` (see
  `TestDecisionRefsLightSilent` and the surrounding cases) and the
  parity test in
  `packages/cli/tests/admission.test.ts`.
- The fixture does not exercise the
  `xh decision record` / `xh decision link` flows; see
  `decision-link-flow` for that coverage.
