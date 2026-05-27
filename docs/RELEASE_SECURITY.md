# Release Security

x-harness release hardening is file-first and CI-enforced. The release path must preserve the verify gate contract: completion is admitted by `verify`, benchmark false accepts block release, and publish credentials do not grant admission authority.

## Required Gates

Before publish, the release workflow runs:

- `npm ci`
- `npm run typecheck`
- `npm run build`
- `npm run lint`
- `npm run format:check`
- `npm test`
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/x-harness`
- `npm run parity:check-go`
- `./x-harness verify --card examples/ci/strict-verify/completion-card.yaml --strict --json`
- `./x-harness doctor --root .`
- `./x-harness examples verify`
- `./x-harness benchmark --filter adversarial --json`
- TypeScript compatibility smoke/parity through `npm run parity:check-go`
- `npm -w packages/cli run pack:dry-run`
- Packed CLI smoke test from the generated `.tgz`
- Frozen transfer compatibility from the generated `.tgz`
- Go release binary matrix build, SHA256 checksums, and Go binary smoke test

The adversarial benchmark is a hard release gate: `false_accept_count` and `adversarial_false_accept_count` must both remain `0`. Adversarial cases also exercise governance-enforced verification for protected-path approval spoofing.

## Package Assets

`packages/cli/scripts/sync-package-assets.mjs` copies the canonical repository assets into the package directory during `npm run build` and `prepack`. The packed package must include:

- `dist`
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
- Cross-platform smoke tests on linux-amd64, darwin-amd64, and windows-amd64 via the release workflow.
