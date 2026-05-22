# CI Integration Guide

> How to run x-harness verification in continuous integration.

## GitHub Actions (recommended)

The repository includes a reference workflow at `.github/workflows/x-harness-verify.yml`:

```yaml
name: x-harness Verify

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-node@v6
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: npm run typecheck
      - run: npm run build
      - run: npm run lint
      - run: npm run format:check
      - run: npm test
      - run: node packages/cli/dist/index.js doctor --root .
```

### What the workflow does

1. Installs dependencies
2. Type-checks the CLI
3. Builds the CLI
4. Runs lint and format checks
5. Runs unit tests
6. Runs `doctor` to validate workspace health

## Local-build fallback

x-harness is **not yet published to npm**. Consumers must build from source.

### Option A: build in CI

```yaml
- run: npm ci
- run: npm run build
- run: node packages/cli/dist/index.js verify --card completion-card.yaml
```

### Option B: composite action (local-build)

See `examples/actions/x-harness-verify/action.yml` for a reusable composite action that:
- Checks out the x-harness repository
- Builds the CLI
- Runs verification against a provided completion card

### Option C: vendor the dist/

For faster CI, you can commit `packages/cli/dist/` to your repository. The CLI has no runtime dependencies beyond Node.js 20+.

## CI behavior of interactive commands

Some x-harness commands support `--interactive` prompts. In CI (non-TTY), these commands:

- Skip interactive prompts
- Exit with appropriate code (`0` for ready, `1` for not ready)
- Output JSON with `--json` for machine parsing

Example:

```bash
node packages/cli/dist/index.js handoff readiness --json
```

## Verification in CI

To verify a completion card in CI:

```bash
node packages/cli/dist/index.js verify --card completion-card.yaml --trace
```

The `--trace` flag appends the verify event to `.x-harness/traces/events.jsonl`.

## Trace verification in CI

To check trace integrity:

```bash
node packages/cli/dist/index.js trace verify-chain
```

This exits `0` if the chain is valid, `1` if tampering is detected.

## Non-goals for CI

- Publishing to npm (not yet)
- Docker image (not required)
- Self-hosted runner requirements (none)
- Dashboard or webhook notifications (out of scope)
