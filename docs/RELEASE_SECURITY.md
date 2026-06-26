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
- Copy Go binaries, Sigstore bundles, and checksums into the npm package
- Build release tarball (`npm pack`)
- Generate CycloneDX SBOM (`sbom.cdx.json`)
- Generate Go module SBOM source (`go-modules.sbom.json`)
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

- Cross-platform smoke tests (`linux/amd64`, `darwin/amd64`, `windows/amd64`)
- Linux arm64 smoke test (`linux/arm64`)
- Attach release artifacts to the GitHub Release only after smoke jobs pass (tagged releases; RC tags are marked prerelease and not latest)
- Publish to npm with provenance only after smoke jobs pass. Stable release tags require `NPM_TOKEN`; RC tags attach signed prerelease GitHub Release artifacts, are not marked latest, and intentionally skip npm publish.

The adversarial benchmark is a hard release gate: `false_accept_count` and `adversarial_false_accept_count` must both remain `0`. Adversarial cases also exercise governance-enforced verification for protected-path approval spoofing.

> **Arm64 build-vs-smoke gap**: Linux arm64 is smoke-tested on `ubuntu-24.04-arm`. Darwin arm64 and Windows arm64 binaries are built and checksummed but are not executed because GitHub-hosted runners for those targets are not universally available.

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
npm publish .x-harness/release/x-harness-<version>.tgz --provenance --access public --tag next
```

The publish job uses GitHub OIDC (`id-token: write`) and `actions/setup-node` with the npm registry. The build/release job uses OIDC for cosign signing but does not publish. If npm trusted publishing is configured, prefer it over long-lived publish tokens. If a token is still required, it must be scoped to package publish and stored as `NPM_TOKEN`. Stable tags fail closed when `NPM_TOKEN` is missing; RC tags publish signed prerelease GitHub Release artifacts only, are not marked latest, and intentionally skip npm publish.

## SBOM

The release workflow and dedicated SBOM workflow generate a CycloneDX SBOM:

```bash
npm sbom --workspace x-harness --sbom-format cyclonedx --sbom-type application > sbom.cdx.json
```

SBOM artifacts are uploaded with release artifacts. Release review should retain the SBOM alongside the npm pack manifest and benchmark report.

## GitHub Release Artifacts

On tagged releases, the publish job attaches the following artifacts to the GitHub Release only after packed-CLI and platform smoke jobs pass:

- All platform Go binaries (`x-harness-<version>-<os>-<arch>`)
- Sigstore bundle files (`.sigstore.json`) for each binary
- `checksums.txt` (SHA256 of Go binaries)
- `release-checksums.txt` (SHA256 of all release artifacts)
- CycloneDX SBOM (`sbom.cdx.json`)
- Go module SBOM source (`go-modules.sbom.json`)
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
  --bundle x-harness-v1.2.3-linux-amd64.sigstore.json \
  --certificate-identity-regexp='https://github.com/BrianNguyen29/x-harness/.github/workflows/release.yml@refs/tags/.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
  x-harness-v1.2.3-linux-amd64
```

## Sigstore And SLSA

Tagged releases sign Go binaries with cosign (keyless via GitHub OIDC):

```bash
cosign sign-blob --yes --bundle "${bin}.sigstore.json" "$bin"
```

Sigstore bundle (`.sigstore.json`) artifacts are uploaded alongside the binaries. Consumers can verify with:

```bash
cosign verify-blob \
  --bundle x-harness-v1.2.3-linux-amd64.sigstore.json \
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
- Go binary checksums, Sigstore bundles, and smoke evidence proving the native binary can run `doctor`, `examples verify`, and golden `verify` locally.
- Cross-platform smoke tests on `linux/amd64`, `darwin/amd64`, and `windows/amd64` via the release workflow.
- Arm64 build-vs-smoke gap documented: arm64 binaries are built but not smoke-tested in CI.

## Homebrew Formula Maintenance

The release workflow generates `x-harness.rb` from `scripts/generate-homebrew-formula.sh` using the release checksums. To update a tap repository:

1. Download the generated formula from the GitHub Release assets or CI artifacts.
2. Copy it into the tap repository as `Formula/x-harness.rb`.
3. Commit and push the change.

Example:

```bash
curl -LO https://github.com/BrianNguyen29/x-harness/releases/download/<tag>/x-harness.rb
cp x-harness.rb homebrew-x-harness/Formula/x-harness.rb
cd homebrew-x-harness
git add Formula/x-harness.rb
git commit -m "x-harness <tag>"
git push
```

Consumers can then install or upgrade:

```bash
brew tap BrianNguyen29/x-harness
brew install x-harness
brew upgrade x-harness
```

## SLSA Provenance Plan

The release workflow already produces Sigstore bundles (cosign) and npm provenance. To reach SLSA Level 3 for Go binaries, the following starter workflow and plan are provided in `.github/workflows/slsa-provenance.yml`:

1. **Artifact preparation** — `.github/workflows/slsa-provenance.yml` builds the Go binary, generates a SHA256 checksum, and uploads the artifact with a `hashes` output.
2. **SLSA attestation generation** — once the official `slsa-framework/slsa-github-generator` reusable workflow is pinned to a verified SHA, a `provenance` job can be added that calls the generic generator with the prepared base64-subjects.
   - Pin verification steps are documented in the workflow file:
     - Visit the generator releases page and copy the full commit SHA.
     - Verify the SHA via an independent channel (Sigstore certificate or project release notes).
     - Replace `<PINNED_SHA>` with the verified SHA.
     - Uncomment the provenance job and enable the release trigger.
     - Run from a test tag to confirm artifact hashes match.
3. **Attestation attachment** — the generated `slsa-attestation.intoto.jsonl` is uploaded to the GitHub Release alongside the existing Sigstore bundles and checksums.
4. **Consumer verification** — consumers can verify the SLSA attestation with the `slsa-verifier` CLI or rely on the existing cosign bundle and release checksums as a secondary signal.

The starter workflow is disabled by default (`workflow_dispatch` only) so the generator pin can be reviewed and enabled without changing the release critical path. No unpinned third-party actions are used in the active workflow.

## Branch Protection and CODEOWNERS Backup

### After-the-fact direct-push guard

Because branch protection settings are pending backup-owner decisions, a file-only CI guard is in place:

- `.github/workflows/branch-discipline.yml` runs on every `push` to `main`.
- It checks whether `HEAD` is a merge commit (two or more parents).
- If `HEAD` has fewer than two parents, the job fails with a message directing changes through pull requests.
- This does not block PR merge commits (which are merge commits) or `pull_request` events themselves.

### Required settings for `main`

- **Require pull request before merging** — all commits to `main` must come through a PR.
- **Require status checks to pass** — the following workflow jobs are required:
  - `quality` matrix jobs (typecheck, build, lint, format, test)
  - `go-quality` matrix jobs (test, race, vet, build, parity)
  - `go-fuzz-smoke`
  - `verify-gates`
  - `verify-gates-supplemental` (when the supplemental workflow is enabled)
- **Require signed commits** — all commits should be GPG or SSH signed where feasible.
- **Restrict pushes that create files** — only allow pushes from the release workflow or automated bots for specific paths.
- **Require CODEOWNERS review** for paths covered by `.github/CODEOWNERS`.

### CODEOWNERS backup recommendation

The current `.github/CODEOWNERS` assigns critical paths (`.github/workflows/`, `policies/`, `schemas/`, `internal/admission/`) to a single owner. To avoid a single point of failure, a backup owner must be added before stable release. Because no confirmed backup owner is currently known, the requirement is documented in `.github/CODEOWNERS` as a comment rather than assigning a guessed handle.

Once a backup owner is confirmed out-of-band, add them to each protected path, e.g.:

```
.github/workflows/ @BrianNguyen29 @secondary-owner
policies/ @BrianNguyen29 @secondary-owner
schemas/ @BrianNguyen29 @secondary-owner
internal/admission/ @BrianNguyen29 @secondary-owner
```

Ensure the secondary owner is a repository admin or has the `maintain` role so they can approve and merge when the primary owner is unavailable. Keep the backup owner list short (two or three individuals) to preserve review velocity while eliminating single-person dependency.

> **Note**: Changing `.github/CODEOWNERS` itself requires the existing owner approval. Any backup-owner addition should be proposed in a dedicated PR with out-of-band confirmation from the proposed backup owner. The current file documents this requirement as a comment pending confirmation.
