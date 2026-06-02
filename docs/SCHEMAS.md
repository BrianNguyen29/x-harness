# Schemas

x-harness validates runtime artifacts using JSON schemas. The schema set is split into **core**, **compatibility**, and **feature** categories.

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
| `adapter-matrix.schema.json`          | Adapter capability matrix produced by `xh adapters list` and consumed by the adapter doctor. |
| `admission-card.schema.json`          | Admission card envelope emitted by `xh card` for downstream admission/verifier handoff. |
| `agent-profile.schema.json`           | Agent profile reporting/update artifacts.    |
| `approval-receipt.schema.json`        | Approval receipts (`decision`, `approver`, `classified_commands`, `aggregate_risk`) attached to high-risk `command_evidence` entries on standard/deep cards. |
| `approval-risk.schema.json`           | Approval-risk evaluation reports.            |
| `attribution.schema.json`             | Attribution report artifacts.                |
| `benchmark-report.schema.json`        | Benchmark JSON output.                       |
| `classifier.schema.json`              | Permission intent classifier result shape (`command`, `intents`, `risk`, `unknown`). |
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
| `release-evidence.schema.json`        | Release evidence bundles (`version`, `artifacts`, `conformance`) emitted by `xh release`. |
| `scanner.schema.json`                 | Static scanner results (`files_scanned`, `findings`, `summary`) from `xh scan` and the conformance scanner check. |
| `subagent-task.schema.json`           | Sub-agent task handoff envelopes (`task_id`, `tier`, `goal`, `scope`, `success_criteria`) used to dispatch bounded work. |
| `withheld-reason.schema.json`        | Typed withheld reason taxonomy for blocked/failed outcomes. See [compatibility boundary](VERIFY_GATE.md) for runtime superset vs strict schema. |

## Validation engine

The CLI uses **Ajv** for JSON Schema validation. No additional validation library (e.g., Zod) is required.
