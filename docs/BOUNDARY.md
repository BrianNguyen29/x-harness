# Boundary Policy

The `xh boundary` command is a deterministic policy checker that validates source-file imports against boundary rules. It is opt-in: when the policy file is missing, boundary commands exit `0` with a warning instead of failing.

## Policy file

- **Default location**: `policies/boundaries.yaml`
- **Schema**: `schemas/boundary-policy.schema.json`
- **Rule shape** (V1): each rule has `id`, `from` (path glob), `to_import` (regex), `action` (`allow` or `block`), `severity` (`low`, `medium`, `high`), optional `intermediate`, `allow` (exceptions), and `applies_to_languages`.

Matching uses simple glob + regex. There is no AST parsing, no semgrep, and no LLM involvement.

## Commands

### `xh boundary lint`

Validates that the boundary policy file is loadable and its rules are well-formed.

```bash
xh boundary lint [--policy <path>] [--root <dir>] [--format text|json]
```

### `xh boundary check`

Scans files and reports violations.

```bash
xh boundary check --all|--changed [--policy <path>] [--root <dir>] [--format text|json] [paths...]
```

- `--all` scans the entire tree under `--root` (default `.`).
- `--changed` scans only files from `git diff --name-only`.
- Positional paths limit the scan when using `--all`.

### `xh boundary explain <file>`

Shows which rules apply to a single file and whether any violations were found.

```bash
xh boundary explain <file> [--policy <path>] [--root <dir>] [--format text|json]
```

## Verify integration

The verify gate can enforce boundary findings via:

```bash
xh verify --card completion-card.yaml --boundary-enforce off|advisory|block_high|block_all [--boundary-policy <path>]
```

- `off` — boundary findings do not affect admission (default).
- `advisory` — findings are reported but do not block.
- `block_high` — high-severity violations block admission.
- `block_all` — any violation blocks admission.

### Boundary approvals in verify

`boundary_approvals` can suppress only matching boundary findings and only when each entry includes `rule_id`, `approver`, RFC3339 `approved_at`, and `reason`. Rule-only or malformed approvals are ignored. Boundary approvals do not override contract-oracle failures and do not grant admission authority by themselves.

## Exit codes

- `0` — no violations (or policy missing / lint passed).
- `1` — violations found, policy load error, or file not found.
- `2` — usage error.
