# Intake Classification

Intake classification maps work signals to runtime tiers for x-harness task routing.

## Intake Labels

| Label       | Runtime Tier | Description                                                                               |
| ----------- | ------------ | ----------------------------------------------------------------------------------------- |
| `tiny`      | `light`      | Comment-only, documentation, formatting changes                                           |
| `normal`    | `standard`   | Routine implementation, standard refactors, normal bug fixes                              |
| `high_risk` | `deep`       | Auth, token, session, admission, schema, permissions, CI, release, destructive filesystem |

## Signal Mapping

### tiny → light

- Comment-only changes
- Documentation updates
- Formatting changes (prettier, ESLint fixes)
- Non-functional refactors

### normal → standard

- Routine implementation tasks
- Standard refactoring
- Normal bug fixes
- Dependency updates

### high_risk → deep

Auth, token, session, and related security-sensitive work always routes to `deep` tier:

- **auth**: Authentication or authorization logic changes
- **token**: Token handling, JWT, refresh tokens, OAuth
- **session**: Session management and state
- **admission**: Admission policy or verifier logic
- **schema**: Schema or contract changes
- **permissions**: File or system permissions
- **ci**: CI/CD pipeline changes
- **release**: Release or deployment logic
- **destructive_filesystem**: Destructive file operations

## Runtime Tiers

The runtime tiers remain: `light`, `standard`, `deep`.

Intake labels (`tiny`, `normal`, `high_risk`) are **separate** from runtime tiers and map to them for routing purposes.

## Usage

```bash
# Classify a task
xh intake classify --task "Fix refresh token race" --files src/auth/session.ts

# With change type signal
xh intake classify --task "Update comments" --files src/auth/session.ts --change comment-only

# Explain a completion card's declared or inferred intake tier
xh intake explain --card completion-card.yaml
```

## Admission Guard

When a completion card declares an `intake` block, `intake.mapped_tier` is checked against the card's runtime `tier`.

- Declared tier lower than mapped tier is a downgrade and is withheld.
- Downgrade requires `governance.approval_status: approved` with `approval_required_for` containing `tier_downgrade` or `intake_tier_downgrade`.
- Cards without an `intake` block are not hard-blocked by this guard; `xh intake explain --card` can still infer and report the likely tier from task text and changed files.

## Policy

See `policies/intake.yaml` for the full intake policy specification.
