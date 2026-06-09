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

The verify workflow is the main pull-request gate. See `.github/workflows/x-harness-verify.yml` for the current pinned YAML. At a high level it runs four jobs:

- `quality` — Node 22 matrix for `typecheck`, `build`, `lint`, `format:check`, and `test`.
- `go-quality` — Go 1.22 matrix for `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/x-harness`, and `npm run parity:check-go`.
- `go-fuzz-smoke` — bounded fuzz target (`FuzzValidate` in `./internal/schema`).
- `verify-gates` — builds both CLIs and runs Go-native primary gates plus TypeScript compatibility parity gates.

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
