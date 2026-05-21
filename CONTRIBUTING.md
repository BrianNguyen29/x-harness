# Contributing

Thank you for contributing to x-harness.

## Core invariants

Preserve these invariants in every change:

- Completion is admitted, not merely claimed.
- The verifier is read-only.
- PGV advice is advisory-only and never grants admission authority.
- Non-success outcomes are withheld.
- Core tooling remains TypeScript-first and file-first.
- No daemon, database, server, MCP service, or external credential is required by default.

## Development workflow

```bash
npm install
npm run typecheck
npm run build
npm test
npm run verify
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js examples verify
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
