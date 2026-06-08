# Intake

The `xh intake` command evaluates task intake tiering and produces structured product-intent records. It is used to classify work, explain tier mappings from completion cards, and generate lightweight handoff suggestions.

## Policy file

- **Default location**: `policies/intake.yaml`
- If the policy is missing, intake subcommands that require it exit with an error.

## Commands

### `xh intake classify`

Classifies a task description into an intake label and runtime tier based on signals in the policy.

```bash
xh intake classify --task "<description>" [--files <path,path>] [--change <type>] [--root <dir>] [--json]
```

Example:

```bash
xh intake classify --task "Add OAuth login flow" --files "src/auth/login.go,src/auth/oauth.go" --json
```

### `xh intake explain`

Explains the intake mapping for an existing completion card, including whether the declared tier matches the mapped tier and whether a downgrade or intervention is required.

```bash
xh intake explain --card <path> [--root <dir>] [--json]
```

### `xh intake contract`

Generates a product-intent record (`schemas/product-intent.schema.json`).

```bash
xh intake contract --id <id> --goal "<text>" --acceptance "<criterion>" [--visible true|false] [--non-goal "<text>"] [--protected-behavior "<text>"] [--ambiguity none|unresolved|partial] [--note "<text>"] [--output <path>] [--json]
```

You can also ingest a markdown description with `--from <markdown-path>` instead of passing individual flags. When using `--from`, the markdown must include an `## Acceptance` (or `## Acceptance Criteria`) section with at least one item. If you use individual flags instead, at least one `--acceptance` is required.

### `xh intake handoff --tier auto`

Uses the intake classifier to pick `light`, `standard`, or `deep`, then prints a minimal handoff command suggestion. For explicit tiers, use `xh handoff <tier>` instead.

```bash
xh intake handoff --tier auto --task "<description>" [--file <path> ...] [--root <dir>] [--json]
```

## Exit codes

- `0` — success.
- `1` — classification mismatch, load error, or validation failure.
- `2` — usage error.
