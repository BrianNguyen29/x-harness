# CI Integration Guide

> How to run x-harness verification in continuous integration.

## GitHub Actions (recommended)

The repository includes these public CI workflows:

- `.github/workflows/x-harness-verify.yml` for TypeScript build, lint,
  typecheck, tests, Go build/test/vet/race/fuzz/parity, strict verify,
  doctor, examples, and adversarial benchmark gates.
- `.github/workflows/security-audit.yml` for dependency vulnerability scanning
  (`npm audit`, `govulncheck`) and secret-scanning requirement documentation.
- `.github/workflows/codeql.yml` for GitHub CodeQL JavaScript/TypeScript
  scanning.
- `.github/workflows/scorecard.yml` for OpenSSF Scorecard supply-chain checks.
- `.github/workflows/sbom.yml` for CycloneDX SBOM generation.
- `.github/workflows/release.yml` for tag-based package verification, Go binary
  release candidate builds/checksums/signing/cross-platform smoke tests, and npm
  provenance publishing.
- `.github/workflows/verify-gates-supplemental.yml` for docs drift, examples,
  conformance, benchmark, schema/policy sync, and additional parity gates.
- `.github/workflows/slsa-provenance.yml` for artifact preparation and SLSA Level 3
  provenance planning (disabled by default until the generator is pinned).

The verify workflow is the main pull-request gate. See `.github/workflows/x-harness-verify.yml` for the current pinned YAML. At a high level it runs four jobs:

- `quality` — Node 22 matrix for `typecheck`, `test:typecheck`, `build`, `lint`, `format`, and `test`.
- `go-quality` — Go 1.22 matrix for `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/x-harness`, and `npm run parity:check-go`.
  - **Race-detector timing**: `go test -race ./...` has passed in CI, but `internal/cli` is the slowest package (observed ~281 s with a 420 s timeout). Monitor race times and split or extend the timeout if `internal/cli` approaches the limit.
- `go-fuzz-smoke` — bounded fuzz target (`FuzzValidate` in `./internal/schema`).
- `verify-gates` — builds both CLIs and runs Go-native primary gates plus TypeScript compatibility parity gates.

### What the workflow does

1. Installs Node and Go dependencies/toolchains
2. Type-checks, test-typechecks, builds, lints, formats, and tests the TypeScript CLI
3. Runs Go tests, race detector, `go vet`, and `go build ./cmd/x-harness`
4. Runs `npm run parity:check-go` to validate that Go CLI behavior matches the committed TypeScript baseline
5. Runs a bounded Go fuzz smoke target (`FuzzValidate`)
6. Runs Go-native primary gates: policy matrix, strict verify, verify profile (`ci-standard`), policy explain, explain card, evidence run, docs drift, release verify-docs, doctor, examples verify, regression suite, adversarial benchmark, and conformance minimal and strict
   - Note: `conformance run` supports only `minimal` and `strict` profiles. `ci-standard` is a `xh verify` profile, not a conformance profile.
7. Runs TypeScript compatibility gates as a secondary validation layer

The release workflow also builds native Go binaries for Linux, macOS, and
Windows, generates SHA256 checksums, signs tag-release binaries with cosign,
and runs smoke tests on Linux/macOS/Windows amd64 artifacts plus Linux arm64.
The separate publish job attaches release artifacts and publishes the npm
tarball only after smoke jobs pass on tagged builds.

## Parity checks

`npm run parity:check-go` validates that the Go CLI output matches the committed TypeScript baseline for key commands. Verify parity compares normalized, contract-critical payloads and requires the Go JSON `schema_version`. Unsupported Go skips fail unless an explicit allowlist entry with an expiry exists. If parity fails, update the baseline with `npm run parity:capture-ts` only after confirming the Go behavior is correct.

## Local-build fallback

Source checkouts can run either the native Go binary or the TypeScript compatibility CLI. The Go CLI is the canonical primary runtime; TypeScript remains a compatibility baseline.

> **Concurrency warning**: Do not run `npm run release:local` concurrently with `npm run parity:check-go`, `npm run build`, or other npm workspace operations that touch `packages/cli/dist/`. Concurrent runs can race on build output and cause transient failures. If `release:local` fails, retry it solo before investigating.

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

## Real Sandbox Integration Tests

For local validation against throwaway codebases, run:

```bash
npm -w packages/cli run test:integration
```

The integration profile includes `sandbox-integration.test.ts`, which creates temporary app workspaces, initializes harness assets, runs `doctor`, `context`, `permissions`, and strict `verify`, checks non-Git mutation guard fallback, and confirms verifier-side mutation detection. Go-native boundary enforcement is covered by `go test ./internal/cli -run Boundary`.

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
