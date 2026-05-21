# Golden Example: Blocked — Missing Evidence Scope

A deep-tier completion card that is withheld because it lacks evidence scope definitions.

## Scenario

An agent completes a deep-tier task but fails to declare the required evidence scopes (`evidence.verification_artifacts`, `evidence.untested_regions`, `evidence.remaining_risks`). Since deep-tier policies require these declarations, verification is blocked.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The agent's completion claim (missing scope declarations).
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
npx x-harness verify --card examples/golden/blocked-missing-evidence-scope/completion-card.yaml
# -> WITHHELD
# Reason: tier "deep" requires evidence scope declarations
```

## Try it

Run the verification gate:
```bash
node packages/cli/dist/index.js verify --card examples/golden/blocked-missing-evidence-scope/completion-card.yaml
```
