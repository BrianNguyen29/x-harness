# Golden Fixture: blocked-weak-prediction

## Purpose

This fixture demonstrates a `standard`-tier completion card that is **withheld** because its `prediction` is too weak to be falsifiable.

## Expected Behavior

When this card is verified (e.g., via `x-harness verify --card completion-card.yaml` or through the conformance suite), the outcome must be:

- `outcome: failed`
- `acceptance_status: withheld`
- Multiple verification checks fail due to the weak prediction signal.

## Why It Is Withheld

The card declares:

```yaml
prediction:
  measurable_signal: "some signal"
  confidence: high
```

A valid `standard`-tier prediction must provide a concrete, measurable signal that can be checked against evidence. The phrase `"some signal"` is vague and non-falsifiable, so the verify gate rejects the claim even though all other fields (schema, evidence, checklist) appear complete.

## Files

- `completion-card.yaml` — The completion card under test.
- `expected-verify-output.txt` — The expected verify gate output.

## Related

- `docs/ADMISSION_POLICY.md` — Evidence floor requirements by tier.
- `examples/golden/regression/` — Other regression-blocking fixtures.
