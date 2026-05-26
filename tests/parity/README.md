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

Check Go CLI parity against the committed TypeScript baseline with:

```bash
node scripts/check-go-parity.mjs
```

This script first verifies that the committed TypeScript baseline is still
current, then builds the Go CLI and compares the supported Go command surface
against the committed TypeScript baseline semantically.

The capture script builds the current TypeScript CLI, runs the canonical
verification, doctor, examples, context, and benchmark commands, then writes
normalized outputs under `tests/parity/baseline/typescript/`.

Normalization replaces volatile runtime fields such as durations, timestamps,
and event identifiers. Stable contract fields such as admission outcome,
acceptance status, recovery route, policy hash, card hash, checks, and command
exit code remain part of the baseline.

The Go rewrite compares against these fixtures semantically:
- `ok`, `admission_outcome`, `acceptance_status`, and exit code for verify cases
- `healthy` and exit code for doctor cases
- presence of core contract facts for context cases

Unsupported cases are explicitly skipped in the Go parity report rather than
silently ignored. The Go rewrite must preserve parity on supported cases before
it can replace the TypeScript implementation.

For machine-readable output:

```bash
node scripts/check-go-parity.mjs --json
```
