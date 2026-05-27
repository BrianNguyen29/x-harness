# CI Integration Guide

> How to run x-harness verification in continuous integration.

## GitHub Actions (recommended)

The repository includes these public CI workflows:

- `.github/workflows/x-harness-verify.yml` for TypeScript build, lint,
  typecheck, tests, Go build/test/vet/race/fuzz/parity, strict verify,
  doctor, examples, and adversarial benchmark gates.
- `.github/workflows/codeql.yml` for GitHub CodeQL JavaScript/TypeScript
  scanning.
- `.github/workflows/scorecard.yml` for OpenSSF Scorecard supply-chain checks.
- `.github/workflows/sbom.yml` for CycloneDX SBOM generation.
- `.github/workflows/release.yml` for tag-based package verification, Go binary
  release candidate builds/checksums/signing/cross-platform smoke tests, and npm
  provenance publishing.

The verify workflow is the main pull-request gate:

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
  quality:
    name: quality / ${{ matrix.name }}
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        include:
          - name: typecheck
            command: npm run typecheck
          - name: build
            command: npm run build
          - name: lint
            command: npm run lint
          - name: format
            command: npm run format:check
          - name: test
            command: npm run build && npm run test
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-node@v6
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: ${{ matrix.command }}

  go-quality:
    name: go-quality / ${{ matrix.name }}
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        include:
          - name: test
            command: go test ./...
          - name: race
            command: go test -race ./...
          - name: vet
            command: go vet ./...
          - name: build
            command: go build ./cmd/x-harness
          - name: parity
            command: npm run parity:check-go
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - uses: actions/setup-node@v6
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: ${{ matrix.command }}

  go-fuzz-smoke:
    name: go-fuzz-smoke
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - run: go test -fuzz=FuzzValidate -fuzztime=15s ./internal/schema

  verify-gates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-node@v6
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: npm run build
      - run: node packages/cli/dist/index.js examples verify
      - run: node packages/cli/dist/index.js doctor --root .
```

### What the workflow does

1. Installs Node and Go dependencies/toolchains
2. Type-checks, builds, lints, formats, and tests the TypeScript CLI
3. Runs Go tests, race detector, `go vet`, and `go build ./cmd/x-harness`
4. Runs `npm run parity:check-go` against the committed TypeScript baseline
5. Runs a bounded Go fuzz smoke target (`FuzzValidate`)
6. Runs strict verify, examples, doctor, and adversarial benchmark gates

The release workflow also builds native Go binaries for Linux, macOS, and
Windows, generates SHA256 checksums, signs tag-release binaries with cosign,
and runs smoke tests on Linux/macOS/Windows amd64 artifacts.

## Local-build fallback

During the Go rewrite migration, source checkouts can run either the native Go
binary or the TypeScript compatibility CLI.

### Option A: build in CI

```yaml
- run: go build ./cmd/x-harness
- run: ./x-harness verify --card completion-card.yaml
```

### Option B: TypeScript compatibility path

```yaml
- run: npm ci
- run: npm run build
- run: node packages/cli/dist/index.js verify --card completion-card.yaml
```

### Option C: composite action (local-build)

See `examples/actions/x-harness-verify/action.yml` for a reusable composite action that:
- Checks out the x-harness repository
- Builds the CLI
- Runs verification against a provided completion card

### Option D: vendor the dist/

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

- Docker image (not required)
- Self-hosted runner requirements (none)
- Dashboard or webhook notifications (out of scope)

Publishing remains restricted to the release workflow and tagged/provenance
path. Pull-request CI does not publish npm packages or Go binaries.
