# Golden Example: Success — Standard Scoped Evidence

A standard-tier completion card that successfully passes verification with rich, well-scoped evidence definitions.

## Scenario

An agent completes a task under the `standard` tier. Unlike the basic `light` tier, the `standard` tier requires the agent to not just state they finished, but to present concrete, granular evidence. In this scenario, the agent declares their read/write file sets and details specific **verification artifacts** (such as unit test runs and typecheck scripts), specifies precisely what they verify/do not verify, and documents any **untested regions** (e.g. omitting E2E browser tests).

Because the card is well-formed, contains no contradictions, and lists valid evidence scopes matching the admission policy, the verify gate admits the completion successfully.

## Files

- [input-task.md](./input-task.md) — The original task description requesting form validation.
- [completion-card.yaml](./completion-card.yaml) — The agent's completion claim containing the scoped evidence, test runs, and untested regions list.
- [expected-verify-output.txt](./expected-verify-output.txt) — Expected quiet output from `x-harness verify`.
- [expected-final-response.md](./expected-final-response.md) — Expected agent final response referencing the card.

## Expected Outcome

Running verification on this card yields a success admission status since all necessary fields for the standard tier are populated correctly:

```yaml
outcome: success
acceptance_status: accepted
```

## Try It Command

Run the verification gate locally:

```bash
xh verify --card examples/golden/regression/success-standard-scoped-evidence/completion-card.yaml
```

## Why It Matters

This example demonstrates the core power of `x-harness` in standard software development workflows:

1. **Evidence Accountability**: The agent must explicitly declare what their tests verified and what they did not verify, reducing false assertions of completeness.
2. **Untested Region Transparency**: By declaring untested regions, the agent flags potential regression risks or manual QA requirements clearly to reviewers.
3. **No Lock-in**: The verification is run locally in a fraction of a second via the CLI without complex integrations.
