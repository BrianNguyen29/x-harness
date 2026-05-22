# Runtime Contract

Canonical runtime labels: `light`, `standard`, `deep`.

Forbidden active aliases: `small`, `medium`, `large`.

Candidate completion:

```yaml
result.fix_status: fixed
verification.status: passed
```

Accepted completion:

```yaml
verify_gate.outcome: success
acceptance_status: accepted
```

Withheld outcomes: `failed`, `blocked`, `skipped`, `timeout`, `error`.

## Authoritative artifact hierarchy

In multi-agent or long-running sessions, the following artifact precedence applies:

1. Source files and git diff are authoritative for implementation state.
2. `completion-card.yaml` is authoritative for completion claim state.
3. `policies/admission.yaml` is authoritative for admission policy.
4. `node packages/cli/dist/index.js verify` output is authoritative for accepted/withheld mapping in this repository.
5. Chat summaries are non-authoritative.

### Adapter rule

If chat says done but `completion-card.yaml` says withheld, treat completion as withheld.
If `completion-card.yaml` claims accepted but verify output disagrees, verify output wins.
