# Cleanup

x-harness cleanup is conservative by design. Evidence and audit artifacts are preserved by default.

> [!NOTE]
> **Local Development Only**: `x-harness` is not yet published to npm. Use `node packages/cli/dist/index.js clean <options>` after building locally.

## Principles

- **Dry-run first**: Always run `node packages/cli/dist/index.js clean --dry-run` before mutating.
- **Never auto-delete evidence**: `completion-card.yaml`, archive, and verify reports are audit artifacts.
- **Protected paths**: templates, schemas, policies, docs, AGENTS.md, X_HARNESS.md, adapters, examples, and source code are never deleted by `clean`.
- **Explicit mutation**: `--tmp`, `--reset-card`, and `--archive-success` require `--force` to apply.

## Commands

```bash
# Preview what would be cleaned
node packages/cli/dist/index.js clean --dry-run --tmp

# Clean tmp and cache (safe)
node packages/cli/dist/index.js clean --tmp --force

# Reset completion card (renames to backup)
node packages/cli/dist/index.js clean --reset-card --force

# Archive a successful completion card
node packages/cli/dist/index.js clean --archive-success --force
```

## Safe cleanup targets

- `.x-harness/tmp/`
- `.x-harness/cache/`

## Never deleted by default

- `completion-card.yaml`
- `.x-harness/archive/`
- `templates/`, `schemas/`, `policies/`, `docs/`
- `AGENTS.md`, `X_HARNESS.md`
- `adapters/`, `examples/`, `packages/`

## Policy

See `policies/cleanup.yaml` for the canonical cleanup policy.
