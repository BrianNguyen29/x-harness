# Context Policy

The `context` command maintains a canonical, versioned context block inside `AGENTS.md`.

## Managed Block

`AGENTS.md` must contain a managed context block marked with:

```html
<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- generated-at: ISO8601 timestamp -->
<!-- context-hash: SHA-256 prefix -->
...
<!-- END X-HARNESS MANAGED CONTEXT -->
```

The block is generated from the canonical rules defined in `packages/cli/src/core/context.ts`.

## Commands

### Show context

```bash
node packages/cli/dist/index.js context
```

Outputs the compact canonical context.

### Verbose context

```bash
node packages/cli/dist/index.js context --verbose
```

Outputs the full rules with explanations and the current context hash.

### JSON output

```bash
node packages/cli/dist/index.js context --json
```

Returns parseable JSON containing:
- `context`: the canonical context text
- `hash`: the context hash
- `mode`: `compact` or `verbose`
- `agents_fresh`: whether the `AGENTS.md` managed block hash matches
- `agents_note`: human-readable freshness note

### Refresh managed block

```bash
node packages/cli/dist/index.js context --refresh
```

Replaces (or appends) the managed block in `AGENTS.md` with the current canonical context and a fresh hash.

## Freshness Rules

- **Fresh**: `AGENTS.md` contains a managed block and its `context-hash` matches the hash of the current canonical context.
- **Stale**: `AGENTS.md` contains a managed block but the hash does not match.
- **Missing**: `AGENTS.md` does not contain the managed block markers.

The `doctor` command includes a `context_freshness` check that reports the current state.
