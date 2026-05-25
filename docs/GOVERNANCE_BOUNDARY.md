# Governance Boundary

## Overview

The governance boundary layer classifies protected paths and reports authority violations. The default mode remains report-only for normal local development. Enforced mode is opt-in through `governance check --enforce`, `verify --governance-enforced`, or `policies/authority.yaml`.

## Authority Classes

### `agent_editable`

Files and paths that agents can freely modify as part of implementation work.

**Examples:**

- `packages/cli/src/**/*.ts`
- `tests/**/*.ts`
- `docs/**/*.md`

### `agent_proposable_human_approved`

Files agents may propose changes to, but require human approval before committing.

**Examples:**

- `policies/recovery.yaml`

### `human_only`

Files and paths that only humans may directly modify. Violations trigger warnings by default and violations in enforced mode.

**Examples:**

- `schemas/**`
- `policies/admission.yaml`
- `policies/permissions.yaml`
- `policies/authority.yaml`
- `packages/cli/src/core/admission.ts`
- `packages/cli/src/core/mutation-guard.ts`
- `.github/workflows/*.yml`
- `package.json`
- `package-lock.json`

## Protected Path Registry

The `policies/authority.yaml` file declares all protected paths with their authority classes and rationales. The registry is authoritative for governance boundary checking.

## Commands

### `xh governance check --card <path> [--json]`

Check files listed in a completion card's `evidence.files_changed` against the authority boundary.

### `xh governance check --card <path> --enforce [--json]`

Require verified approval artifacts for protected path changes and exit non-zero when approval is missing, out of scope, or hash-mismatched.

### `xh governance check --diff HEAD [--json]`

Check files changed in a Git diff against the authority boundary.

### `xh governance explain --path <path> [--json]`

Explain the authority class for a specific path.

### `xh governance list-protected [--json]`

List all protected paths and their authority classes.

### `xh intervention validate --intervention <path> [--json]`

Validate an intervention artifact against the schema.

## Report-Only Behavior

**Default governance is report-only:**

- All authority violations are reported as **warnings** (not errors)
- The CLI exits **0** on warnings (no admission block)
- `block_on_violations: false` in `policies/authority.yaml`
- This allows agents to learn the rules while building the fixture corpus

## Enforced Mode

Enforced mode is explicit and path-scoped. A protected path is allowed only when the completion card has:

```yaml
governance:
  approval_status: approved
  approval_artifact:
    path: .x-harness/approvals/APPROVAL-001.yaml
    sha256: sha256:<artifact-hash>
```

The approval artifact must exist under `.x-harness/approvals/`, match the declared SHA-256 hash, be registered in `.x-harness/approvals/registry.json`, have `decision: approved`, include `approved_by` and `approved_at`, and declare a `scope.paths` entry that covers the protected path.

Example approval artifact:

```yaml
approval_id: APPROVAL-001
decision: approved
approved_by: alice
approved_at: 2026-05-25T00:00:00.000Z
scope:
  paths:
    - schemas/completion-card.schema.json
```

`verify --governance-enforced` withholds admission for unapproved protected paths and routes recovery through `Fpermission`.

## Intervention Schema

Interventions are artifacts that track governance decisions. The schema (`schemas/intervention.schema.json`) requires:

- `actor` — Who initiated the intervention
- `task` — Task ID this applies to
- `scope` — Granularity: file, directory, path, global
- `decision` — allow, deny, flag, override
- `reason` — Human-readable rationale
- `expiration` — When the intervention expires

## Recovery Predicates

Two new predicates support governance:

- **`Fpermission`** — Authority boundary violation (human_only path modified)
  - Route: `user` owner, request human approval
- **`Fintervention`** — Intervention artifact required
  - Route: `implementation-worker` owner, review and resolve

## Enforcement Rollout

The adversarial benchmark runs governance-enforced verification for adversarial fixtures. Release gates still require `false_accept_count = 0` and `adversarial_false_accept_count = 0`.

Policy-level enforcement is disabled by default:

```yaml
authority_enforcement:
  mode: report_only
  admission_behavior: withhold
  require_verified_intervention: true
```

Operators can set `mode: enforced` only when they want protected-path approval checks to apply automatically.

## Exit Criteria (PR2)

- [x] `xh governance check` reports authority violations as warnings by default
- [x] `xh governance check --enforce` exits non-zero for unapproved protected paths
- [x] `verify --governance-enforced` withholds unapproved protected paths
- [x] `xh governance explain --path` correctly describes authority class
- [x] Intervention schema validates and `xh intervention validate` works
- [x] Recovery predicates for `Fpermission` and `Fintervention` are defined
