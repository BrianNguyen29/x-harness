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
