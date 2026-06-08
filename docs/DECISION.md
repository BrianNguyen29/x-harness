# Decision Records

The `xh decision` command manages lightweight decision memory records (ADR-lite). Records are stored as plain YAML/JSON files and can be queried, linked to completion cards, and tracked by affected paths.

## Schema

- **Schema file**: `schemas/decision-record.schema.json`
- Required fields: `schema_version`, `id`, `decision`, `rationale`.
- Optional fields: `title`, `date`, `status`, `context`, `consequences`, `superseded_by`, `tags`, `affected_paths`, `notes`.

## Default storage

Records are written to `decisions/` by default. The directory is created on demand for default paths; explicit `--output` paths require the parent directory to exist.

## Commands

### `xh decision record`

Creates a decision record.

```bash
xh decision record --id <id> --decision "<text>" --rationale "<text>" [--title "<text>"] [--status proposed|accepted|superseded|deprecated] [--date <iso-date>] [--context "<text>"] [--consequence "<text>"] [--superseded-by <id>] [--tag <text> ...] [--affected-path <path> ...] [--note "<text>"] [--output <path>] [--json]
```

### `xh decision list`

Lists all decision records in the directory.

```bash
xh decision list [--dir <path>] [--json]
```

### `xh decision query`

Searches decision records by keyword across visible fields.

```bash
xh decision query --keyword "<text>" [--dir <path>] [--json]
```

### `xh decision affected`

Finds records whose `affected_paths` match a given path (supports glob patterns).

```bash
xh decision affected --path <path> [--dir <path>] [--json]
```

### `xh decision link`

Links decision references into a completion card’s `context_alignment.decision_refs` array.

```bash
xh decision link --card <path> --decision <id> [--decision <id> ...] [--out <path>] [--json]
```

Without `--out`, the card is updated in place.

## Exit codes

- `0` — success.
- `1` — record validation failure, query error, or link mismatch.
- `2` — usage error.
