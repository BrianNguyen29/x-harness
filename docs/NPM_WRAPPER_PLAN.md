# NPM Wrapper Plan

Goal: transition the `x-harness` npm package from a TypeScript (Node.js) CLI entrypoint to a platform-native Go binary wrapper, while preserving backward compatibility and a safe fallback window.

## Background

- The TypeScript CLI lives in `packages/cli/dist/index.js` and is invoked via `node`.
- A Go implementation exists under `cmd/x-harness` and `internal/` with parity checks in `scripts/check-go-parity.mjs`.
- Phase 8 introduces CI dual-run (Node + Go) and release candidate readiness.

## Strategy

1. **Ship both binaries** in the npm package for a compatibility window.
2. **Install-time platform selection** chooses the correct Go binary or falls back to the Node implementation.
3. **No publish behavior change** until the wrapper is fully validated.

## Platform Binary Selection

At install time (via a `postinstall` script or a lightweight JS shim):

1. Detect `process.platform` and `process.arch`.
2. Map to the Go release asset naming convention:
   - `linux`/`darwin`/`windows` × `amd64`/`arm64`
3. Look for the matching Go binary in the installed package directory.
4. If present and executable, use it.
5. If absent or execution fails, fall back to the Node implementation.

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

The `bin` field in `package.json` currently points to a JS file. The wrapper plan replaces it with a lightweight Node.js launcher script (`bin/x-harness.js`) that:

1. Resolves the package root.
2. Attempts to spawn the platform-matching Go binary.
3. If the Go binary exits with a non-executable / not-found error, execs `node dist/index.js` with the same arguments.
4. Forwards `stdin`, `stdout`, `stderr`, and exit codes transparently.

This keeps the `npm install` experience identical for consumers.

## Fallback / Compatibility Window

- **Phase 8–9**: Go binary is included in the npm tarball but the JS launcher defaults to Go only when `X_HARNESS_GO=1` is set. Default remains Node.
- **Phase 10**: JS launcher defaults to Go when the binary is present. Node fallback is automatic.
- **Phase 11+**: Node fallback is removed; package becomes a thin wrapper around the Go binary.

During the compatibility window:
- All existing `xh <command>` invocations continue to work.
- The Node CLI remains the source of truth for verification gates.
- Go parity checks (`npm run parity:check-go`) must remain green before advancing phases.

## Asset Layout

Inside the published tarball:

```
x-harness/
  package.json
  bin/x-harness.js          # launcher shim
  dist/index.js             # TypeScript CLI (fallback)
  go-binaries/
    x-harness-v1.2.3-linux-amd64
    x-harness-v1.2.3-darwin-arm64
    ...
  schemas/                   # runtime assets
  policies/
  templates/
  ...
```

The Go binaries are built by the release workflow (`.github/workflows/release.yml`) and injected into the package before `npm pack`.

## Security / Checksum Behavior

- Release workflow generates `sha256sum` checksums for every Go binary and writes `checksums.txt` into `go-binaries/`.
- The launcher shim may optionally verify the binary checksum on first run (or when `X_HARNESS_VERIFY_CHECKSUM=1`).
- Checksums are included in the npm tarball so consumers can audit them.
- No external network calls are required for verification; everything is local to the installed package.

## Rollback

If a Go binary release causes issues:

1. Consumers can force Node fallback with `X_HARNESS_GO=0`.
2. The npm package can be republished from the previous Node-only tag.
3. The release workflow can be reverted by removing the Go build steps.

## Open Questions

- Whether to sign Go binaries (cosign / GPG) in addition to SHA256.
- Whether to publish platform-specific optional dependencies instead of bundling all binaries.
- Exact timing of Phase 10/11 based on Go parity maturity.
