# TypeScript Maintenance Policy

This document defines the maintenance policy for the TypeScript CLI after the Go rewrite became primary.

## Status

- TypeScript is no longer included in the published npm package runtime. The published package is a Go-only wrapper.
- TypeScript source (`packages/cli/src`) and compiled output (`packages/cli/dist`) remain in the repository for local development and CI compatibility gates.
- The wrapper defaults to Go and falls back to Node only when `dist/index.js` exists (source checkout or custom build).

## Maintenance Mode Rules

The TypeScript source in `packages/cli/src` is governed by these rules:

1. **No new features**: New commands, flags, or behaviors are added only to the Go CLI.
2. **Bug fixes only**: Critical bug fixes that affect the source-checkout fallback path may be backported.
3. **Security patches**: Security issues in dependencies or the wrapper shim are patched promptly.
4. **Contract compatibility**: The TypeScript fallback must continue to produce the same admission decisions as the Go CLI for all golden fixtures.

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

## Benchmark Convention: `.skip-ts-benchmark`

Golden fixtures that cover Go-only or opt-in behavior not implemented by the TypeScript compatibility benchmark include the marker `.skip-ts-benchmark` in their expected output filename. Examples include `blocked-contract-oracle` and `blocked-missing-context-ref`.

- **Purpose**: Signals that `examples verify` handles the fixture's expected output, but the TypeScript benchmark runner should skip it.
- **Behavior**: The benchmark runner detects the marker and excludes the fixture from the TS parity run; `examples verify` still validates it normally.
- **Usage**: Add `.skip-ts-benchmark` to the expected output filename when the test case covers behavior (e.g. Contract Oracle opt-in, missing context refs) that the TypeScript compatibility layer does not implement.
