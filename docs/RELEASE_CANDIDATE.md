# Release Candidate Requirements

This document defines the release-candidate (RC) requirements, including dual-run CI, Go parity verification, packaging, and cross-platform smoke evidence.

## Overview

A release candidate is a tagged pre-release that must pass the full verification gate before promotion to a stable release. The RC requirements ensure:

- TypeScript and Go implementations produce identical outcomes on golden fixtures.
- Release artifacts (npm tarball and Go binaries) are healthy.
- Cross-platform smoke tests pass on Linux, macOS, and Windows.

## Release Requirements

1. **Dual-run CI passes**
   - Node.js quality gates: typecheck, test:typecheck, build, lint, format, test.
   - Go quality gates: `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/x-harness`.
   - Go parity check: `npm run parity:check-go`.
   - Go-native primary gates (required):
     - `./x-harness verify --card examples/ci/strict-verify/completion-card.yaml --strict --json`
     - `./x-harness doctor --root . --json`
     - `./x-harness examples verify --json`
     - `./x-harness benchmark --filter adversarial --json`
   - TypeScript compatibility gates (secondary, from source checkout):
     - `node packages/cli/dist/index.js verify --card examples/ci/strict-verify/completion-card.yaml --strict --json`
     - `node packages/cli/dist/index.js doctor --root .`
     - `node packages/cli/dist/index.js examples verify`
     - `node packages/cli/dist/index.js benchmark --filter adversarial --json`

2. **Go release binary build**
   - Binaries built for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.
   - `ldflags -X main.version=${VERSION}` injects the tag version into the Go CLI.

3. **Checksums and signing**
   - `sha256sum` checksums generated for all binaries.
   - Binaries signed with Sigstore cosign on tag releases.

4. **Injection into npm package**
   - Signed Go binaries, Sigstore bundles, and `checksums.txt` copied into `packages/cli/go-binaries/` before `npm pack`.
   - The packed tarball contains the platform-native Go binaries and the wrapper shim, but **not** the Node.js fallback `dist/`.

5. **Packed tarball smoke tests**
   - **Default Go path**: install the tarball into a temp project and run `xh doctor`, `xh verify`, and frozen transfer commands (wrapper defaults to Go when the binary is present).
   - **Forced Go path**: install the tarball into a temp project and run `X_HARNESS_GO=1 xh --version`, `X_HARNESS_GO=1 xh doctor`, and `X_HARNESS_GO=1 xh examples verify`.

6. **Cross-platform smoke**
   - Download release artifacts on `ubuntu-latest`, `macos-latest`, and `windows-latest`.
   - Run `tests/smoke/go-binary-smoke.sh` against the platform-matching binary.

7. **Evidence retention**
   - `benchmark-report.json` and `.x-harness/release/**` uploaded as CI artifacts.
   - SBOM (`sbom.cdx.json`) and npm pack manifest (`npm-pack.json`) retained.

8. **GitHub Release artifact attachment**
   - For tagged releases, the publish job attaches all artifacts to the GitHub Release after packed-CLI and platform smoke jobs pass:
     - Go binaries, Sigstore bundles (`.sigstore.json`), and `checksums.txt`.
     - `release-checksums.txt` covering all release artifacts.
     - CycloneDX SBOM (`sbom.cdx.json`).
     - Go module SBOM source (`go-modules.sbom.json`).
     - Scoop manifest (`scoop/x-harness.json`).
     - Homebrew formula (`x-harness.rb`).
     - npm tarball (`x-harness-*.tgz`) and npm pack manifest (`npm-pack.json`).
     - Adversarial benchmark report (`benchmark-report.json`).
   - Consumers can verify downloads with `sha256sum -c checksums.txt` and `cosign verify-blob --bundle`.
   - RC tags may skip npm publish when `NPM_TOKEN` is not configured; stable tags must fail closed without npm publish credentials.

## Local Release Notes

- `release:local` runs `build`, `typecheck`, `lint`, `format:check`, and `test:release` in sequence. Do **not** run it concurrently with `parity:check-go`, `npm run build`, or other npm workspace operations that touch the same `packages/cli` build output. Concurrent runs can race on `dist/` and cause transient failures. If `release:local` fails, run it solo before investigating.

## Wrapper Default-to-Go

The npm wrapper (`bin/x-harness.js`) defaults to the Go binary when a platform-matching binary is present. Node fallback is available **only** in source checkouts where `dist/index.js` exists; the published package is Go-only.

The criteria required before flipping the default:

1. All primary Go commands exist and pass golden/adversarial parity tests.
2. At least one full RC cycle has passed with dual-run CI green.
3. `doctor` reports healthy on all supported platforms.
4. Mutation guard strict path passes in CI.
5. Release artifact smoke tests pass on Linux, macOS, and Windows.
6. Docs/templates/adapters managed-block drift is zero.
7. No critical open issues blocking Go-primary usage.

## Evidence Floor for RC Admission

An RC completion card should include:

- `files_changed`: release workflow, smoke scripts, wrapper shim, tests.
- `command_evidence`: CI links, benchmark reports, parity check output.
- `done_checklist`: all items in the release requirements above.
- `prediction`: risk assessment for the release (e.g., platform coverage, rollback plan).

For a deep-tier RC, additionally declare:

- `evidence_scope_declared`: which platforms and commands were smoke-tested.
- `untested_regions_declared`: any platforms or edge cases not covered.
- `remaining_risks_declared`: known risks (e.g., Windows path handling, signature verification optional).
- `execution_controls_present`: checksums, signing, provenance.
- `rollback_policy_present`: republish previous tag, restore `dist/` inclusion if needed.
