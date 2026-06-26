# Policy Authority Matrix

This document is a human-readable rendering of `policies/authority.yaml`. It defines who may modify what, and the rationale for each protected path. The YAML file is the authoritative source; this page is a convenience reference.

## Authority Classes

| Class | Description | Typical Examples |
|-------|-------------|-----------------|
| `agent_editable` | Agents may modify freely as part of normal implementation. | `packages/cli/src/**/*.ts`, `tests/**/*.ts`, `docs/**/*.md` |
| `agent_proposable_human_approved` | Agents may propose changes, but a human must approve before commit. | `policies/recovery.yaml` |
| `human_only` | Only humans may directly modify. | `schemas/**`, core engine files, workflow files, policy files, manifest files |

## Protected Path Registry

| Path | Authority | Rationale |
|------|-----------|-----------|
| `schemas/**` | `human_only` | Schema definitions are authoritative contracts |
| `policies/admission.yaml` | `human_only` | Admission policy defines success criteria |
| `policies/permissions.yaml` | `human_only` | Permissions policy defines authority boundaries |
| `policies/authority.yaml` | `human_only` | Authority policy defines governance boundaries |
| `policies/recovery.yaml` | `agent_proposable_human_approved` | Recovery routing may be updated by agents with human approval |
| `packages/cli/src/core/admission.ts` | `human_only` | Core admission engine governs success criteria |
| `packages/cli/src/core/mutation-guard.ts` | `human_only` | Mutation guard prevents unauthorized file changes |
| `packages/cli/src/core/permissions.ts` | `human_only` | Permissions engine evaluates command and capability boundaries |
| `packages/cli/src/commands/permissions.ts` | `human_only` | Permissions command exposes policy decisions |
| `.github/workflows/*.yml` | `human_only` | CI/CD workflow gates must not be bypassed by agents |
| `.github/workflows/x-harness-verify.yml` | `human_only` | Primary verification workflow gate |
| `package.json` | `human_only` | Package manifest controls build/test commands |
| `package-lock.json` | `human_only` | Lockfile is generated from `package.json` changes |
| `packages/cli/src/core/authority.ts` | `human_only` | Authority classification logic |
| `packages/cli/src/validators/*.ts` | `human_only` | Validators enforce schema contracts |
| `packages/cli/src/commands/governance.ts` | `human_only` | Governance commands implement boundary reporting |
| `templates/**` | `human_only` | Task templates define agent workflows |

## Enforcement Mode

| Setting | Value | Note |
|---------|-------|------|
| `report_only` | `true` | Enforcement is currently advisory. The CLI warns but does not block. |
| `admission_behavior` | `withhold` | When enforcement is enabled, protected-path violations map to `withheld`. |
| `require_verified_intervention` | `true` | Requires an approval artifact scoped to each protected path. |
| `governance_check.behavior` | `warn` | Governance checks emit warnings. |
| `exit_on_warnings` | `false` | Warnings do not cause nonzero exit codes. |
| `block_on_violations` | `false` | Violations do not block admission in report-only mode. |

## How to Use This Matrix

1. **Before editing a file**, check its authority. If it is `human_only`, stop and route the change through a human with the appropriate approval context.
2. **Before proposing a policy change**, verify that the corresponding `packages/cli/policies/` or `packages/cli/schemas/` copy is synchronized (see `scripts/check-schema-policy-sync.mjs`).
3. **When running CI**, the `doctor` and `policy matrix` commands reflect this authority configuration. The Go-native CLI is the canonical source of truth.

## References

- `policies/authority.yaml` — authoritative source
- `docs/THREAT_MODEL.md` — trust boundaries and failure capabilities
- `docs/ADMISSION_POLICY.md` — admission behavior and evidence floor
- `scripts/check-schema-policy-sync.mjs` — drift detection for protected-path copies
