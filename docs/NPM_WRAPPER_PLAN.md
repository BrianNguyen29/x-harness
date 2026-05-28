# NPM Wrapper Plan

Goal: transition the `x-harness` npm package from a TypeScript (Node.js) CLI entrypoint to a platform-native Go binary wrapper. The published package is now Go-only; Node fallback is available only in source checkouts where `dist/index.js` exists.

## Background

- The TypeScript CLI lives in `packages/cli/dist/index.js` and is invoked via `node`.
- A Go implementation exists under `cmd/x-harness` and `internal/` with parity checks in `scripts/check-go-parity.mjs`.
- Phase 8 foundation introduced CI dual-run (Node + Go), Go release candidate binaries, checksums, and a local Go binary smoke test.
- Phase 10 flipped the wrapper default to Go.
- Phase 11 (current) removed TypeScript from the published runtime package.

## Strategy

1. **Ship Go binaries** in the npm package as the only runtime.
2. **Install-time platform selection** chooses the correct Go binary.
3. **No published Node fallback**; the wrapper exits with an error if the Go binary is missing and `dist/index.js` is not present.

## Platform Binary Selection

At install time (via the lightweight JS shim `bin/x-harness.js`):

1. Detect `process.platform` and `process.arch`.
2. Map to the Go release asset naming convention:
   - `linux`/`darwin`/`windows` × `amd64`/`arm64`
3. Look for the matching Go binary in the installed package directory.
4. If present and executable, use it.
5. If absent, exit with an error (published package) or fall back to Node (source checkout only).

Example mapping:

| platform | arch   | asset name                         |
|----------|--------|------------------------------------|
| linux    | x64    | `x-harness-${version}-linux-amd64` |
| linux    | arm64  | `x-harness-${version}-linux-arm64` |
| darwin   | x64    | `x-harness-${version}-darwin-amd64`|
| darwin   | arm64  | `x-harness-${version}-darwin-arm64`|
| win32    | x64    | `x-harness-${version}-windows-amd64.exe` |
| win32    | arm64  | `x-harness-${version}-windows-arm64.exe` |

## NPM Bin Shim

The `bin` field in `package.json` points to `bin/x-harness.js`. The launcher:

1. Resolves the package root.
2. Attempts to spawn the platform-matching Go binary.
3. If the Go binary exits with a non-executable / not-found error, and `dist/index.js` exists (source checkout), execs `node dist/index.js` with the same arguments.
4. Forwards `stdin`, `stdout`, `stderr`, and exit codes transparently.
5. If `dist/index.js` does not exist, prints an error and exits non-zero.

This keeps the `npm install` experience identical for consumers, with the published package now being a thin Go launcher.

## Fallback / Compatibility Window

- **Phase 8 foundation (complete)**: Go release binaries are built and uploaded as release artifacts with checksums.
- **Phase 8–9 wrapper implementation (complete)**: The npm package ships `bin/x-harness.js`. The launcher previously defaulted to Node compatibility mode and used a packaged Go binary only when `X_HARNESS_GO=1` is set.
- **Phase 10 (complete)**: JS launcher defaults to Go when the binary is present. Node fallback remains automatic via `X_HARNESS_GO=0` or when the Go binary is missing.
- **Phase 11 (current)**: TypeScript `dist/` is removed from the published npm package. The wrapper is Go-only for published consumers. Node fallback remains available in source checkouts when `dist/index.js` exists.
- **Phase 12+**: Node fallback is removed entirely from the wrapper; TypeScript source may be removed from the repository.

During the published Go-only phase:
- All existing `xh <command>` invocations continue to work as long as the platform-matching Go binary is bundled.
- The Node CLI remains the compatibility baseline for CI verification gates while Go parity checks run in CI.
- Go parity checks (`npm run parity:check-go`) and packed wrapper smoke tests must remain green before advancing phases.

## Asset Layout

Inside the published tarball:

```
x-harness/
  package.json
  bin/x-harness.js          # launcher shim
  go-binaries/
    x-harness-v1.2.3-linux-amd64
    x-harness-v1.2.3-darwin-arm64
    ...
  schemas/                   # runtime assets
  policies/
  templates/
  ...
```

The Go binaries are built by the release workflow (`.github/workflows/release.yml`) and injected into the npm tarball. The TypeScript `dist/` directory is **not** included in the published tarball.

## Security / Checksum Behavior

- Release workflow generates `sha256sum` checksums for every Go binary and writes `checksums.txt` into `go-binaries/`.
- The launcher shim may optionally verify the binary checksum on first run (or when `X_HARNESS_VERIFY_CHECKSUM=1`).
- Checksums are included in the npm tarball so consumers can audit them.
- No external network calls are required for verification; everything is local to the installed package.

## Rollback

If a Go binary release causes issues:

1. Source-checkout consumers can force Node fallback with `X_HARNESS_GO=0` (requires building TypeScript).
2. The npm package can be republished from the previous tag.
3. The release workflow can be reverted by restoring the `dist/` inclusion in `files`.

## Open Questions

- Go binaries are signed with cosign on tagged releases; the launcher shim may optionally verify signatures when `X_HARNESS_VERIFY_SIGNATURE=1` is set.
- Whether to publish platform-specific optional dependencies instead of bundling all binaries.
- Exact timing of Phase 12 based on Go parity maturity.
