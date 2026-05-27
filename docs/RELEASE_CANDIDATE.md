# Release Candidate Cycle

This document describes the x-harness release-candidate (RC) process, including dual-run CI, Go parity verification, packaging, and cross-platform smoke evidence.

## Overview

A release candidate is a tagged pre-release that must pass the full verification gate before promotion to a stable release. The RC cycle ensures:

- TypeScript and Go implementations produce identical outcomes on golden fixtures.
- Release artifacts (npm tarball and Go binaries) are healthy.
- Cross-platform smoke tests pass on Linux, macOS, and Windows.

## RC Checklist

1. **Dual-run CI passes**
   - Node.js quality gates: typecheck, build, lint, format, test.
   - Go quality gates: `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/x-harness`.
   - Go parity check: `npm run parity:check-go`.

2. **Go release binary build**
   - Binaries built for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.
   - `ldflags -X main.version=${VERSION}` injects the tag version into the Go CLI.

3. **Checksums and signing**
   - `sha256sum` checksums generated for all binaries.
   - Binaries signed with Sigstore cosign on tag releases.

4. **Injection into npm package**
   - Signed Go binaries, signatures, certificates, and `checksums.txt` copied into `packages/cli/go-binaries/` before `npm pack`.
   - The packed tarball therefore contains both the Node.js fallback and the platform-native Go binaries.

5. **Packed tarball smoke tests**
   - **Node path**: install the tarball into a temp project and run `xh doctor`, `xh verify`, and frozen transfer commands.
   - **Go path**: install the tarball into a temp project and run `X_HARNESS_GO=1 xh --version`, `X_HARNESS_GO=1 xh doctor`, and `X_HARNESS_GO=1 xh examples verify`.

6. **Cross-platform smoke**
   - Download release artifacts on `ubuntu-latest`, `macos-latest`, and `windows-latest`.
   - Run `tests/smoke/go-binary-smoke.sh` against the platform-matching binary.

7. **Evidence retention**
   - `benchmark-report.json` and `.x-harness/release/**` uploaded as CI artifacts.
   - SBOM (`sbom.cdx.json`) and npm pack manifest (`npm-pack.json`) retained.

## Wrapper Default-to-Go Criteria

The npm wrapper (`bin/x-harness.js`) currently defaults to the Node.js runtime unless `X_HARNESS_GO=1` is set. The criteria for flipping the default to Go are:

1. All primary Go commands exist and pass golden/adversarial parity tests.
2. At least one full RC cycle has passed with dual-run CI green.
3. `doctor` reports healthy on all supported platforms.
4. Mutation guard strict path passes in CI.
5. Release artifact smoke tests pass on Linux, macOS, and Windows.
6. Docs/templates/adapters managed-block drift is zero.
7. No critical open issues blocking Go-primary usage.

Until these criteria are met, `X_HARNESS_GO=1` remains the opt-in path.

## Evidence Floor for RC Admission

An RC completion card should include:

- `files_changed`: release workflow, smoke scripts, wrapper shim, tests.
- `command_evidence`: CI links, benchmark reports, parity check output.
- `done_checklist`: all items in the RC checklist above.
- `prediction`: risk assessment for the release (e.g., platform coverage, rollback plan).

For a deep-tier RC, additionally declare:

- `evidence_scope_declared`: which platforms and commands were smoke-tested.
- `untested_regions_declared`: any platforms or edge cases not covered.
- `remaining_risks_declared`: known risks (e.g., Windows path handling, signature verification optional).
- `execution_controls_present`: checksums, signing, provenance.
- `rollback_policy_present`: force Node fallback via `X_HARNESS_GO=0`, republish previous tag.
