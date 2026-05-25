# Failure Attribution

Failure attribution explains why an episode was withheld. It is deterministic, local, and advisory-only.

## Taxonomy

| Code             | Meaning                                              |
| :--------------- | :--------------------------------------------------- |
| `Ftask_spec`     | Task specification incomplete or contradictory.      |
| `Fcontext`       | Context missing, stale, or ignored.                  |
| `Ftool`          | Tool unavailable, unsafe, or failed.                 |
| `Fmemory`        | Memory lesson missing, stale, or misleading.         |
| `Fstate`         | Read/write set, conflict policy, or task state bad.  |
| `Fobservability` | Trace, evidence, or episode artifact malformed.      |
| `Fattribution`   | Attribution system could not classify the cause.     |
| `Fverification`  | Evidence missing, weak, or non-replayable.           |
| `Fpermission`    | Permission or read-only boundary violation.          |
| `Fentropy`       | Drift, orphan policy, or unregistered component.     |
| `Fintervention`  | Approval missing, expired, invalid, or out of scope. |
| `Fmodel`         | Likely model execution error despite adequate guard. |
| `Funknown`       | Insufficient data.                                   |

## Episode Artifact

Every verify episode writes:

```text
failure-attribution.json
```

The artifact validates against `schemas/attribution.schema.json` and always carries:

```json
{
  "admission_authority": false
}
```

Accepted episodes may have no primary failure. Withheld episodes must have a primary candidate.

## Commands

```bash
node packages/cli/dist/index.js attribution explain --episode .x-harness/episodes/<episode-id>
node packages/cli/dist/index.js attribution report --episodes-dir .x-harness/episodes --group-by predicate
node packages/cli/dist/index.js attribution report --episodes-dir .x-harness/episodes --group-by component
```

Reports track repeated predicates, repeated components, and `Funknown` rate. A high unknown rate is an entropy warning, not an admission decision.
