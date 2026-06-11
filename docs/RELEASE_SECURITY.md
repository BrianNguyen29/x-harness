# Release Security

x-harness release hardening is file-first and CI-enforced. The release path must preserve the verify gate contract: completion is admitted by `verify`, benchmark false accepts block release, and publish credentials do not grant admission authority.

## Required Gates

Before publish, the release workflow runs the following primary gates through the Go-native CLI. TypeScript compatibility gates are retained as a secondary validation layer run from source checkout.

### Primary Go-native gates

- `npm ci`
- `npm run typecheck`
- `npm run build`
- `npm run lint`
- `npm run format:check`
- `npm test`
- Build Go CLI: `go build -o ./x-harness ./cmd/x-harness`
- Strict verify gate: `./x-harness verify --card examples/ci/strict-verify/completion-card.yaml --strict --json`
- Workspace doctor: `./x-harness doctor --root . --json`
- Golden examples: `./x-harness examples verify --json`
- Adversarial benchmark gate: `./x-harness benchmark --filter adversarial --gate --json > benchmark-report.json`
- `npm -w packages/cli run pack:dry-run`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- Go-native regression suite: `./x-harness examples verify --suite=regression --json`
- Go-native conformance minimal gate: `./x-harness conformance run --profile minimal --json`
- Go-native conformance strict gate: `./x-harness conformance run --profile strict --json`
- `npm run parity:check-go`

### Release artifact generation

- Go release binary matrix build (`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`)
- SHA256 checksums for all binaries (`checksums.txt`)
- Go binary smoke test (`linux/amd64`)
- Sign Go binaries with cosign (tagged releases only)
- Copy Go binaries, signatures, and checksums into the npm package
- Build release tarball (`npm pack`)
- Generate CycloneDX SBOM (`sbom.cdx.json`)
- Generate `release-checksums.txt` covering all release artifacts

### Post-build smoke and compatibility

- Packed CLI smoke test (default Go path)
- Packed CLI Go smoke test (forced Go path, `X_HARNESS_GO=1`)
- Frozen transfer compatibility test
- ~~Packed CLI Node fallback smoke test~~: removed because the published tarball no longer ships `dist/index.js`.
- TypeScript compatibility gates (from source checkout):
  - `node packages/cli/dist/index.js verify --card examples/ci/strict-verify/completion-card.yaml --strict --json`
  - `node packages/cli/dist/index.js doctor --root .`
  - `node packages/cli/dist/index.js examples verify`
  - `node packages/cli/dist/index.js benchmark --filter adversarial --gate --json`

### Publish

- Attach release artifacts to the GitHub Release (tagged releases)
- Publish to npm with provenance (tagged releases; requires `NPM_TOKEN`)
- Cross-platform smoke tests (`linux/amd64`, `darwin/amd64`, `windows/amd64`)

The adversarial benchmark is a hard release gate: `false_accept_count` and `adversarial_false_accept_count` must both remain `0`. Adversarial cases also exercise governance-enforced verification for protected-path approval spoofing.

> **Arm64 build-vs-smoke gap**: arm64 binaries are included in the release matrix but are not smoke-tested in CI because GitHub-hosted arm64 runners are not universally available for Linux and Windows. The cross-platform smoke job covers amd64 only and is skipped automatically if the release job fails.

## Package Assets

`packages/cli/scripts/sync-package-assets.mjs` copies the canonical repository assets into the package directory during packaging. The packed package must include:

- `bin`
- `go-binaries`
- `schemas`
- `policies`
- `templates`
- `adapters`
- `examples`
- `docs`
- `components`
- `tools`
- `AGENTS.md`
- `X_HARNESS.md`
- `README.md`
- `CHANGELOG.md`
- `LICENSE`

The TypeScript CLI resolves package assets through `packages/cli/src/core/assets.ts`, and the Go CLI resolves source/package assets through `internal/assets`. This keeps `xh init` and `xh examples verify` working from source, `npm link`, global install, `npx`, `pnpm dlx`, GitHub Actions temp repos, and native Go binary smoke tests.

> **Note**: `dist/` is intentionally excluded from the published tarball. The npm package is a Go-only wrapper.

## Provenance

Tagged releases publish with npm provenance:

```bash
npm publish --workspace x-harness --provenance --access public
```

The release job uses GitHub OIDC (`id-token: write`) and `actions/setup-node` with the npm registry. If npm trusted publishing is configured, prefer it over long-lived publish tokens. If a token is still required, it must be scoped to package publish and stored as `NPM_TOKEN`.

## SBOM

The release workflow and dedicated SBOM workflow generate a CycloneDX SBOM:

```bash
npm sbom --workspace x-harness --sbom-format cyclonedx --sbom-type application > sbom.cdx.json
```

SBOM artifacts are uploaded with release artifacts. Release review should retain the SBOM alongside the npm pack manifest and benchmark report.

## GitHub Release Artifacts

Tagged releases automatically attach the following artifacts to the GitHub Release via the release workflow:

- All platform Go binaries (`x-harness-<version>-<os>-<arch>`)
- Cosign signatures (`.sig`) and certificates (`.pem`) for each binary
- `checksums.txt` (SHA256 of Go binaries)
- `release-checksums.txt` (SHA256 of all release artifacts)
- CycloneDX SBOM (`sbom.cdx.json`)
- npm pack manifest (`npm-pack.json`)
- npm tarball (`x-harness-<version>.tgz`)
- Scoop manifest (`scoop/x-harness.json`)
- Homebrew formula (`x-harness.rb`)
- Adversarial benchmark report (`benchmark-report.json`)

Consumers can verify a downloaded binary against the release checksums:

```bash
sha256sum -c checksums.txt
```

And verify the cosign signature (requires `cosign` CLI):

```bash
cosign verify-blob \
  --signature x-harness-v1.2.3-linux-amd64.sig \
  --certificate x-harness-v1.2.3-linux-amd64.pem \
  --certificate-identity-regexp='https://github.com/BrianNguyen29/x-harness/.github/workflows/release.yml@refs/tags/.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  x-harness-v1.2.3-linux-amd64
```

## Sigstore And SLSA

Tagged releases sign Go binaries with cosign (keyless via GitHub OIDC):

```bash
cosign sign-blob --yes \
  --output-signature "${bin}.sig" \
  --output-certificate "${bin}.pem" \
  "$bin"
```

Signature (`.sig`) and certificate (`.pem`) artifacts are uploaded alongside the binaries. Consumers can verify with:

```bash
cosign verify-blob \
  --signature x-harness-v1.2.3-linux-amd64.sig \
  --certificate x-harness-v1.2.3-linux-amd64.pem \
  --certificate-identity-regexp='https://github.com/BrianNguyen29/x-harness/.github/workflows/release.yml@refs/tags/.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  x-harness-v1.2.3-linux-amd64
```

In addition to signing, releases must keep:

- npm provenance enabled for tagged publish.
- CI-generated SBOM attached as an artifact.
- `npm pack --dry-run` output reviewed in CI.
- Packed CLI smoke test proving `xh init`, `xh verify`, and `xh doctor` run from the tarball.
- Frozen compatibility proving the packed CLI can export, verify, and merge-import a frozen bundle.
- Go binary checksums and smoke evidence proving the native binary can run `doctor`, `examples verify`, and golden `verify` locally.
- Cross-platform smoke tests on `linux/amd64`, `darwin/amd64`, and `windows/amd64` via the release workflow.
- Arm64 build-vs-smoke gap documented: arm64 binaries are built but not smoke-tested in CI.

## Homebrew Formula Maintenance

The release workflow generates `x-harness.rb` from `scripts/generate-homebrew-formula.sh` using the release checksums. To update a tap repository:

1. Download the generated formula from the GitHub Release assets or CI artifacts.
2. Copy it into the tap repository as `Formula/x-harness.rb`.
3. Commit and push the change.

Example:

```bash
curl -LO https://github.com/BrianNguyen29/x-harness/releases/download/v0.99.0-rc1/x-harness.rb
cp x-harness.rb homebrew-x-harness/Formula/x-harness.rb
cd homebrew-x-harness
git add Formula/x-harness.rb
git commit -m "x-harness 0.99.0-rc1"
git push
```

Consumers can then install or upgrade:

```bash
brew tap BrianNguyen29/x-harness
brew install x-harness
brew upgrade x-harness
```
