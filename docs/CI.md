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
      - uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6
      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0
        with:
          go-version: "1.22"
      - uses: actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: ${{ matrix.command }}

  go-fuzz-smoke:
    name: go-fuzz-smoke
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6
      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0
        with:
          go-version: "1.22"
      - run: go test -fuzz=FuzzValidate -fuzztime=15s ./internal/schema

  verify-gates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6
      - uses: actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6
        with:
          node-version: 22
          cache: npm
      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0
        with:
          go-version: "1.22"
      - run: npm ci
      - run: npm run build
      - run: go build -o ./x-harness ./cmd/x-harness
      - name: Go-native policy matrix
        run: ./x-harness policy matrix --json
      - name: Go-native strict verify gate
        run: ./x-harness verify --card examples/ci/strict-verify/completion-card.yaml --strict --json
      - name: Go-native verify profile gate (ci-standard)
        run: ./x-harness verify --profile ci-standard --card examples/ci/strict-verify/completion-card.yaml --json
      - name: Go-native policy explain gate
        run: ./x-harness policy explain admission.evidence_floor --json
      - name: Go-native explain card gate
        run: ./x-harness explain --card examples/ci/strict-verify/completion-card.yaml --json
      - name: Go-native evidence run gate
        run: ./x-harness evidence run -- echo phase1-gate
      - name: Go-native docs drift gate
        run: ./x-harness doctor --docs-drift --root . --json
      - name: Go-native release verify-docs gate
        run: ./x-harness release verify-docs --root .
      - name: Go-native workspace doctor
        run: ./x-harness doctor --root . --json
      - name: Go-native golden examples
        run: ./x-harness examples verify --json
      - name: Go-native regression suite gate
        run: ./x-harness examples verify --suite=regression --json
      - name: Go-native adversarial benchmark gate
        run: ./x-harness benchmark --filter adversarial --gate --json
      - name: Go-native conformance minimal gate
        run: ./x-harness conformance run --profile minimal --json
      - name: "TypeScript compatibility: strict verify"
        run: node packages/cli/dist/index.js verify --card examples/ci/strict-verify/completion-card.yaml --strict --json
      - name: "TypeScript compatibility: workspace doctor"
        run: node packages/cli/dist/index.js doctor --root .
      - name: "TypeScript compatibility: golden examples"
        run: node packages/cli/dist/index.js examples verify
      - name: "TypeScript compatibility: adversarial benchmark"
        run: node packages/cli/dist/index.js benchmark --filter adversarial --gate --json
```

### What the workflow does

1. Installs Node and Go dependencies/toolchains
2. Type-checks, builds, lints, formats, and tests the TypeScript CLI
3. Runs Go tests, race detector, `go vet`, and `go build ./cmd/x-harness`
4. Runs `npm run parity:check-go` to validate that Go CLI behavior matches the committed TypeScript baseline
5. Runs a bounded Go fuzz smoke target (`FuzzValidate`)
6. Runs Go-native primary gates: policy matrix, strict verify, verify profile (`ci-standard`), policy explain, explain card, evidence run, docs drift, release verify-docs, doctor, examples verify, regression suite, adversarial benchmark, and conformance minimal
   - Note: `conformance run` supports only `minimal` and `strict` profiles. `ci-standard` is a `xh verify` profile, not a conformance profile.
7. Runs TypeScript compatibility gates as a secondary validation layer

The release workflow also builds native Go binaries for Linux, macOS, and
Windows, generates SHA256 checksums, signs tag-release binaries with cosign,
runs smoke tests on Linux/macOS/Windows amd64 artifacts, and attaches all
release artifacts to the GitHub Release for tagged builds.

## Parity checks

`npm run parity:check-go` validates that the Go CLI output matches the committed TypeScript baseline for key commands. This ensures the Go implementation does not drift from the canonical behavior established by the TypeScript CLI during the compatibility window. If parity fails, update the baseline with `npm run parity:capture-ts` after confirming the Go behavior is correct.

## Local-build fallback

Source checkouts can run either the native Go binary or the TypeScript compatibility CLI. The Go CLI is the canonical primary runtime; TypeScript remains a compatibility baseline.

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

For faster CI, you can commit `packages/cli/dist/` to your repository. The published CLI declares `engines.node: ">=20"` in `packages/cli/package.json`; the bundled verify workflow runs on Node 22 to match the latest LTS.

## CI behavior of interactive commands

Some x-harness commands support `--interactive` prompts. In CI (non-TTY), these commands:

- Skip interactive prompts
- Exit with appropriate code (`0` for ready, `1` for not ready)
- Output JSON with `--json` for machine parsing

Example:

```bash
node packages/cli/dist/index.js handoff readiness --json
```

Some interactive helper flows remain TypeScript-compatibility only during the Go parity window.

## Verification in CI

To verify a completion card in CI:

```bash
./x-harness verify --card completion-card.yaml --trace
# compatibility: node packages/cli/dist/index.js verify --card completion-card.yaml --trace
```

The `--trace` flag appends the verify event to `.x-harness/traces/events.jsonl`.

## Trace verification in CI

To check trace integrity:

```bash
./x-harness trace verify-chain
# compatibility: node packages/cli/dist/index.js trace verify-chain
```

This exits `0` if the chain is valid, `1` if tampering is detected.

## Non-goals for CI

- Docker image (not required)
- Self-hosted runner requirements (none)
- Dashboard or webhook notifications (out of scope)

Publishing remains restricted to the release workflow and tagged/provenance
path. Pull-request CI does not publish npm packages or Go binaries.
