# Golden Example: Intent Ref Present

A standard-tier completion card whose top-level `intent_ref` field
carries a non-blank slug id. The fixture exercises the positive
branch of the safe-V1 `intent_ref` gate: when the field is present
and non-blank, the admission layer emits no `intent_ref not declared`
advisory note and the card is admitted. The linked product intent
record lives at `src/intent/intake-lite.md` so the slug resolves
to a real on-disk path under the fixture directory.

## Scenario

An agent declares a product intent record (per
`schemas/product-intent.schema.json`) and links it to a completion
card by setting `intent_ref: doc/intake-lite.md`. The admission
layer inspects the field, finds a non-blank value, and admits the
card without surfacing the `intent_ref not declared` advisory note.

## Files

- `completion-card.yaml` — standard-tier card with a non-blank
  top-level `intent_ref`. The
  `context_alignment.decision_refs` array also references the
  same on-disk file so the path-resolution branches in the engine
  agree on a single fixture artifact.
- `expected-verify-output.txt` — expected quiet summary from
  `xh examples verify` (outcome: success, accepted, 2 passed).
- `src/intent/intake-lite.md` — placeholder product intent
  record referenced by the top-level `intent_ref` slug. The body
  is intentionally sparse; the regression check is the
  admission-layer advisory suppression, not the product intent
  content.

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
`regression/intent-ref-present`. The admission layer treats a
non-blank `intent_ref` as sufficient; the linked product intent
file is documentation for the reader and is not validated by
`xh examples verify`.

## What this fixture intentionally does NOT cover

- The verify-layer `intent_ref_missing` blocking predicate is
  wired to `--intent-enforce block` and the `governed-deep`
  profile (with `ci-strict` defaulting to advisory). `xh examples
  verify` does not apply those flags, so a blocked-by-intent_ref
  fixture would not be exercised deterministically by this
  engine. Future slices may add a profile-driven blocked fixture.
- The fixture does not exercise `xh intent record` or
  `xh intent link` flows; the on-disk artifact is a static
  placeholder rather than the output of a live CLI command.
- The fixture does not exercise the
  `product_intent.status` or `intent_contract` gates; the card
  leaves both fields absent on purpose so the empty-state
  advisories for those gates are emitted, isolating the
  `intent_ref` path from those other gates.
