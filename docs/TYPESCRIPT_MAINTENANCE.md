# TypeScript Maintenance and Freeze Policy

This document defines the maintenance policy for the TypeScript CLI after the Go rewrite becomes primary.

## Status

- **Freeze active**: TypeScript is no longer included in the published npm package runtime. The published package is a Go-only wrapper.
- **Source-checkout development**: TypeScript source (`packages/cli/src`) and compiled output (`packages/cli/dist`) remain in the repository for local development and CI compatibility gates.
- **Target**: After the compatibility window ends, TypeScript source may be removed from the repository entirely.
- **Active**: The wrapper defaults to Go and falls back to Node only when `dist/index.js` exists (source checkout or custom build).

## Freeze Criteria

TypeScript is frozen as a compatibility source when:

1. The Go CLI implements all primary commands with parity tests passing.
2. At least one release-candidate cycle has completed successfully with Go as the default.
3. Cross-platform smoke tests pass on Linux, macOS, and Windows.
4. The npm wrapper (`bin/x-harness.js`) defaults to Go and falls back to Node automatically.
5. No P0 or P1 bugs remain open against the Go CLI.

## Maintenance Mode Rules

Once frozen, the TypeScript source in `packages/cli/src` is governed by these rules:

1. **No new features**: New commands, flags, or behaviors are added only to the Go CLI.
2. **Bug fixes only**: Critical bug fixes that affect the source-checkout fallback path may be backported.
3. **Security patches**: Security issues in dependencies or the wrapper shim are patched promptly.
4. **Contract compatibility**: The TypeScript fallback must continue to produce the same admission decisions as the Go CLI for all golden fixtures.
5. **Deprecation timeline**: The TypeScript source remains in the repository for a minimum compatibility window (recommended: two minor releases or 90 days, whichever is longer).
6. **Removal**: TypeScript source removal is considered only after the compatibility window ends and no critical fallback usage is reported.

## What Is Frozen

- `packages/cli/src/` — TypeScript command implementations.
- `packages/cli/dist/` — compiled output (rebuilt from frozen source).
- `packages/cli/bin/x-harness.js` — wrapper shim behavior for Node fallback.

## What Remains Active

- `schemas/`, `policies/`, `templates/`, `adapters/`, `examples/` — shared runtime assets are not frozen; they are updated via the canonical contract generator and synced to both runtimes.
- `docs/` — documentation continues to evolve.

## Fallback Behavior

In the published npm package:

- The wrapper is Go-only; `dist/` is not included in the tarball.
- `X_HARNESS_GO=0` exits with an error because Node fallback is unavailable.

In a source checkout (or when `dist/index.js` exists):

- `X_HARNESS_GO=1` forces the Go binary.
- `X_HARNESS_GO=0` forces the Node fallback.
- Without the variable, the wrapper defaults to Go when a matching binary is present.
- If the Go binary is missing, the wrapper automatically falls back to Node.

## Migration for Consumers

No action is required for npm consumers. The package continues to install via `npm install x-harness` and expose `x-harness` / `xh` binaries.

Consumers who pinned to the TypeScript runtime behavior should verify against golden fixtures after updating.
