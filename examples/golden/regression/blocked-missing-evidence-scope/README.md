# Golden Example: Blocked — Missing Evidence Scope

A deep-tier completion card that is withheld because it lacks evidence scope definitions.

## Scenario

An agent completes a deep-tier task. The card includes evidence files and even some `verification_artifacts`, but those artifacts are missing the required `verifies`/`does_not_verify` declarations that constitute the evidence **scope**. Deep-tier policies require each verification artifact to declare what it verifies or does not verify; without these scope declarations, verification is blocked.

This is not about total absence of evidence fields — it is specifically about artifact-level scope declarations being absent or empty.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The agent's completion claim (missing scope declarations).
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
node packages/cli/dist/index.js verify --card examples/golden/blocked-missing-evidence-scope/completion-card.yaml
# -> WITHHELD
# Reason: tier "deep" requires evidence scope declarations
```

## Try it

Run the verification gate:

```bash
node packages/cli/dist/index.js verify --card examples/golden/blocked-missing-evidence-scope/completion-card.yaml
```
