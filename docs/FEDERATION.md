# Federation

Federation is an optional enterprise feature for sharing anonymized failure patterns across repositories. It is disabled by default, requires explicit opt-in for export, and has no admission authority.

## Safety Boundary

- No raw source code is exported.
- No raw logs or command output are exported.
- No completion-card contents are exported.
- Secret-like values are rejected during export/import validation.
- Imported patterns are stored as local advisory data only.
- Federation never changes `admission.outcome` or `acceptance_status`.

## Policy

The policy lives at `policies/federation.yaml`:

```yaml
federation:
  enabled: false
  require_opt_in: true
  require_redaction: true
  tenant_boundary: required
```

`enabled: false` is intentional. Local export is allowed only when the operator passes `--opt-in`, `--redacted`, and a tenant id for scoped hashing.

## Export

Export reads a local evidence index and writes JSONL records validated by `schemas/federation-pattern.schema.json`.

```bash
node packages/cli/dist/index.js federation export-patterns \
  --index evidence/index.jsonl \
  --out .x-harness/federation/patterns.jsonl \
  --tenant tenant-local \
  --opt-in \
  --redacted \
  --json
```

The exported file includes hashed tenant/source ids, hashed predicates, hashed component hints, optional benchmark metrics, retention metadata, and `admission_authority: false`.

## Import

Import verifies the schema and secret scan before planning or writing.

```bash
node packages/cli/dist/index.js federation import-patterns .x-harness/federation/patterns.jsonl --json
```

Default import is dry-run. To store patterns locally:

```bash
node packages/cli/dist/index.js federation import-patterns .x-harness/federation/patterns.jsonl --merge --json
```

Imported data is written under `.x-harness/federation/` unless a different target inside the repository is provided.
