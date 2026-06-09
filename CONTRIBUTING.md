# Contributing

Thank you for contributing to x-harness.

## Core invariants

Preserve these invariants in every change:

- Completion is admitted, not merely claimed.
- The verifier is read-only.
- PGV advice is advisory-only and never grants admission authority.
- Non-success outcomes are withheld.
- Core tooling is Go-native with a TypeScript compatibility baseline, and remains file-first.
- No daemon, database, server, MCP service, or external credential is required by default.

## Development workflow

```bash
npm install
npm run typecheck
npm run build
npm test
npm run verify
go test ./...
go build ./cmd/x-harness
npm run parity:check-go
./x-harness doctor --root .
./x-harness examples verify
```

Before opening a pull request, include the relevant command output or explain why a check could not be run.

## Harness Change Contract

PRs that modify admission policy, schemas, templates, CLI verify, adapters, or skills must include a completed `templates/HARNESS_CHANGE_CONTRACT.md`.

## Pull request expectations

- Keep changes small and scoped.
- Add or update tests for behavior changes.
- Update docs/templates/examples when contracts change.
- Do not include secrets or real customer/user data.
- Mark roadmap/preview functionality as optional; do not make it a default requirement.

## Repository hygiene

- **Package assets are mirrored.** Repository root assets (`schemas/`, `policies/`, `templates/`, `adapters/`, `docs/`, `examples/`, etc.) are the source of truth. `packages/cli/scripts/sync-package-assets.mjs` copies them into `packages/cli/` during packaging. Do not edit the mirrored copies under `packages/cli/` directly; edit the root assets and re-run the sync script.
- **Clean local artifacts.** Generated local directories such as `.x-harness/release/`, `.x-harness/tmp/`, and `.codegraph/` are ignored by `.gitignore` and must not be committed. Remove them before opening a pull request if they appear in your working tree.
- **Synthetic secrets only.** Test fixtures that contain secrets are intentionally fake (synthetic) values used for redaction and security tests. Never use real credentials, tokens, or customer data in tests or fixtures.
