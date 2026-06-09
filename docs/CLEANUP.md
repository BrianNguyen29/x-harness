# Cleanup

x-harness cleanup is conservative by design. Evidence and audit artifacts are preserved by default.

> [!NOTE]
> **Local Development**: use `./x-harness clean <options>` after `go build ./cmd/x-harness`, or the TypeScript compatibility path `node packages/cli/dist/index.js clean <options>` after `npm run build`.

## Principles

- **Dry-run first**: Always run `./x-harness clean --dry-run` before mutating.
- **Never auto-delete evidence**: `completion-card.yaml`, archive, and verify reports are audit artifacts.
- **Protected paths**: templates, schemas, policies, docs, AGENTS.md, X_HARNESS.md, adapters, examples, and source code are never deleted by `clean`.
- **Explicit mutation**: `--tmp`, `--reset-card`, and `--archive-success` require `--force` to apply.

## Commands

```bash
# Preview what would be cleaned
./x-harness clean --dry-run --tmp

# Clean tmp and cache (safe)
./x-harness clean --tmp --force

# Reset completion card (renames to backup)
./x-harness clean --reset-card --force

# Archive a successful completion card
./x-harness clean --archive-success --force
```

## Safe cleanup targets

- `.x-harness/tmp/`
- `.x-harness/cache/`

## Local generated artifacts

The following directories are created during local development or release builds and are safe to remove manually. They are ignored by `.gitignore` and must not be committed.

- `.x-harness/release/` — Release build artifacts, packed tarballs, Go binaries, and checksums. Safe to delete manually when not actively cutting a release:
  ```bash
  rm -rf .x-harness/release/
  ```
- `.x-harness/tmp/` — Temporary and cache files. Prefer the CLI command when available:
  ```bash
  ./x-harness clean --tmp --force
  ```
- `.codegraph/` — IDE or tooling index files. Safe to delete manually:
  ```bash
  rm -rf .codegraph/
  ```

## Never deleted by default

- `completion-card.yaml`
- `.x-harness/archive/`
- `templates/`, `schemas/`, `policies/`, `docs/`
- `AGENTS.md`, `X_HARNESS.md`
- `adapters/`, `examples/`, `packages/`

## Policy

See `policies/cleanup.yaml` for the canonical cleanup policy.
