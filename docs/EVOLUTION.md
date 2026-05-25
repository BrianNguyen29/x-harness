# Evolution

The evolution loop is experimental, disabled by default, and cannot mutate production harness files. It exists to turn observed failures into reviewable change requests while preserving the core contract: completion is admitted by verify, not by an evolution candidate.

## Boundaries

- Evolution has no admission authority.
- `xh evolve promote` writes a promotion request only. It does not merge, edit policy, edit schema, or run git.
- `xh evolve rollback` writes a rollback request only. It does not run git.
- PGV remains advisory-only.
- The verify gate must remain read-only.

## Constitution

The constitution lives at `tools/experimental/evolve/constitution.yaml` and is validated by `schemas/evolution-constitution.schema.json`. It defines invariants that candidates must not violate, including read-only verification, fail-closed admission, human approval boundaries, advisory PGV, and zero false accepts.

Run a constitution check:

```bash
node packages/cli/dist/index.js evolve constitution-check --candidate tools/experimental/evolve/fixtures/pass-candidate.yaml --json
```

Unsafe candidates fail closed:

```bash
node packages/cli/dist/index.js evolve constitution-check --candidate tools/experimental/evolve/fixtures/violating-candidate.yaml --json
```

## Budget

The budget lives at `tools/experimental/evolve/evolution-budget.yaml`. The default is:

```yaml
evolution_budget:
  enabled: false
```

When disabled, `xh evolve evaluate` reports that no local candidate loop will run.

## Change Request Loop

The local MVP supports request generation only:

```bash
node packages/cli/dist/index.js evolve evaluate --json
node packages/cli/dist/index.js evolve analyze --run run_001 --out .x-harness/evolution/analysis.md
node packages/cli/dist/index.js evolve propose --component admission_policy --write
node packages/cli/dist/index.js evolve compare --candidate tools/experimental/evolve/fixtures/pass-candidate.yaml --json
node packages/cli/dist/index.js evolve promote --candidate tools/experimental/evolve/fixtures/pass-candidate.yaml --out .x-harness/evolution/promotion.md
node packages/cli/dist/index.js evolve rollback --candidate tools/experimental/evolve/fixtures/pass-candidate.yaml --out .x-harness/evolution/rollback.md
```

Every generated request includes `admission_authority: false`.
