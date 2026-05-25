# Parity Fixtures

This directory stores deterministic baseline outputs used to migrate x-harness
from the TypeScript CLI to the Go CLI without contract drift.

Generate the TypeScript baseline with:

```bash
node scripts/capture-ts-baseline.mjs
```

Check that the committed baseline still matches the current TypeScript CLI with:

```bash
node scripts/check-ts-baseline.mjs
```

The capture script builds the current TypeScript CLI, runs the canonical
verification, doctor, examples, context, and benchmark commands, then writes
normalized outputs under `tests/parity/baseline/typescript/`.

Normalization replaces volatile runtime fields such as durations, timestamps,
and event identifiers. Stable contract fields such as admission outcome,
acceptance status, recovery route, policy hash, card hash, checks, and command
exit code remain part of the baseline.

The Go rewrite must compare against these fixtures semantically before it can
replace the TypeScript implementation.
