# Golden Example: Intent Ref Empty (Advisory)

A standard-tier completion card whose top-level `intent_ref` field
is absent. The fixture exercises the safe-V1 advisory path: the
admission layer emits a top-level note (`intent_ref not declared`)
but the note is advisory-only and never blocks admission. The
verify-layer `intent_ref_missing` blocking predicate is wired to
`--intent-enforce block` and the `governed-deep` profile (with
`ci-strict` defaulting to advisory), which `xh examples verify`
does not apply; the regression suite therefore exercises the
advisory branch, not the blocking branch.

## Scenario

A standard-tier agent submits a completion card that is otherwise
valid but has not yet been linked to any product intent record.
The admission layer should accept the card and surface the
missing-intent_ref advisory note so the reader knows the gate is
silent at the admission layer. The note wording makes it explicit
that admission acceptance is not intent correctness.

## Files

- `completion-card.yaml` — standard-tier card with no top-level
  `intent_ref` field. The
  `context_alignment.decision_refs` array is also empty; the
  focus of the fixture is the `intent_ref` gate, but the empty
  `decision_refs` keeps the fixture profile consistent with
  the existing `decision-refs-empty-advisory` regression case.
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
`regression/intent-ref-empty-advisory`. The Go and TypeScript
admission layers both emit the `intent_ref not declared`
advisory note; the regression compare step only checks
`outcome` and `acceptance_status` from the expected snapshot, so
the note does not affect the comparison.

## What this fixture intentionally does NOT cover

- The verify-layer `intent_ref_missing` blocking predicate
  (predicate id `intent_ref_missing`,
  `withheld_reason.class == "intent_ref_missing"`) is wired to
  `xh verify --intent-enforce block` and the `governed-deep`
  profile (with `ci-strict` defaulting to advisory). `xh examples
  verify` does not apply those flags, so a blocked-by-intent_ref
  fixture would not be exercised deterministically by the
  regression engine. This fixture therefore exercises only the
  advisory branch.
- The light-tier path is silent: a light-tier card with no
  top-level `intent_ref` emits no advisory note. The light-tier
  behavior is covered by the safe-V1 unit tests in
  `internal/admission/admission_test.go` (see
  `TestIntentRefLightSilent` and the surrounding cases) and the
  parity test in
  `packages/cli/tests/admission.test.ts`.
- The fixture does not exercise the
  `xh intent record` or `xh intent link` flows; see
  `intent-ref-present` for a fixture that links a static
  product intent record via a non-blank `intent_ref` slug.
