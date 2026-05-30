# Schemas

x-harness validates runtime artifacts using JSON schemas. The schema set is split into **core**, **compatibility**, **feature**, and **future** categories.

## Core schemas (v0.1)

These schemas are actively used by the verify gate and CLI.

| Schema                        | Purpose                                                     |
| :---------------------------- | :---------------------------------------------------------- |
| `completion-card.schema.json` | Validates the canonical completion card produced by agents. |
| `subagent-return.schema.json` | Validates structured return payloads from sub-agents.       |
| `verify-event.schema.json`    | Validates JSONL trace events emitted by the verify gate.    |
| `report.schema.json`          | Validates metrics and trace report JSON output.             |
| `pgv-advice.schema.json`      | Validates advisory-only PGV guidance packets.               |

### Artifact hardening fields (v0.1+)

`completion-card.schema.json` supports optional hardened artifact metadata on each `verification_artifacts` item. These fields improve traceability and reproducibility without blocking admission when absent:

- `exit_code` — integer exit code from the artifact command
- `started_at` / `ended_at` — ISO 8601 timestamps for the artifact run
- `stdout_hash` / `stderr_hash` — content hashes for captured output
- `artifact_path` — path to a persisted artifact file
- `artifact_hash` — hash of the persisted artifact
- `ci_run_url` — URI linking to the CI run that produced the artifact

These fields are optional in all tiers. For `standard` and `deep`, the admission engine may emit advisory notes when artifact metadata is sparse, but it does not block valid cards solely for missing these fields.

## Compatibility schemas (v0.1)

These schemas are kept for interoperability with claim/evidence workflows and existing examples. They are not required for the minimal harness flow.

| Schema                 | Purpose                                   |
| :--------------------- | :---------------------------------------- |
| `claim.schema.json`    | Validates a standalone claim artifact.    |
| `evidence.schema.json` | Validates a standalone evidence artifact. |

## Feature schemas (shipped with optional commands)

These schemas are included in the package and used by command-specific features outside the minimal verify flow.

| Schema                                | Purpose                                      |
| :------------------------------------ | :------------------------------------------- |
| `agent-profile.schema.json`           | Agent profile reporting/update artifacts.    |
| `approval-risk.schema.json`           | Approval-risk evaluation reports.            |
| `attribution.schema.json`             | Attribution report artifacts.                |
| `benchmark-report.schema.json`        | Benchmark JSON output.                       |
| `components-registry.schema.json`     | Component registry validation.               |
| `context-alignment.schema.json`       | Context alignment evidence for verify --context-floor. |
| `contract-oracle.schema.json`         | Contract oracle rule violations (grep_rules and dependency_rules) from `x-harness contract check --json`. |
| `cost-budget.schema.json`             | Cost budget policy/report structures.        |
| `episode-manifest.schema.json`        | Episode package manifests.                   |
| `evidence-index.schema.json`          | Evidence index entries.                      |
| `evolution-constitution.schema.json`  | Experimental evolution constitution files.   |
| `federation-pattern.schema.json`      | Federation pattern exchange.                 |
| `frozen-manifest.schema.json`         | Frozen transfer manifests.                   |
| `intervention.schema.json`            | Governance intervention artifacts.            |
| `packet.schema.json`                  | Immutable packet records.                    |
| `permissions.schema.json`             | Permissions policy validation.               |
| `withheld-reason.schema.json`        | Typed withheld reason taxonomy for blocked/failed outcomes. See [compatibility boundary](VERIFY_GATE.md) for runtime superset vs strict schema. |

## Future schemas (not shipped in v0.1)

The following schemas remain reserved for upcoming releases. They are **not present** in the v0.1 runtime schema directory and should not be referenced by v0.1 tooling.

- `story.schema.json` (planned for v0.2)
- `test-matrix.schema.json` (planned for v0.2)
- `task.schema.json` (planned for v0.2)
- `feature-intake.schema.json` (planned for v0.2)
- `product-contract.schema.json` (planned for v0.2)
- `recovery.schema.json` (planned for v0.3)
- `decision-record.schema.json` (planned for v0.2)

## Validation engine

The CLI uses **Ajv** for JSON Schema validation. No additional validation library (e.g., Zod) is required.
