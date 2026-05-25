# Predictions

Predictions make a completion candidate falsifiable. They are advisory observability records until the verify gate evaluates the matching episode outcome.

## Card Field

```yaml
prediction:
  claim: "The changed behavior is fixed."
  expected_effect: "The declared verify command passes."
  measurable_signal: "npm run typecheck exits 0."
  falsification_method: "Run the same verify command from the completion card."
  horizon: same_verify
  confidence: high
```

`standard` and `deep` cards must include a prediction. Weak predictions are withheld by admission. `light` cards may include one.

## Commands

```bash
node packages/cli/dist/index.js prediction check --card completion-card.yaml
node packages/cli/dist/index.js prediction verify --episode .x-harness/episodes/<episode-id>
node packages/cli/dist/index.js prediction report --episodes-dir .x-harness/episodes --since 30d
```

`prediction verify --episode` supports `same_verify` deterministically:

- `confirmed`: the episode has `admission.outcome: success` and `acceptance_status: accepted`.
- `falsified`: the episode is withheld.
- `inconclusive`: the prediction is missing, invalid, or uses a horizon that requires external signal.

Prediction status does not grant admission authority.
