# Schemas

x-harness validates runtime artifacts using JSON schemas. The schema set is intentionally narrow in v0.1 and split into **core**, **compatibility**, and **future** categories.

## Core schemas (v0.1)

These schemas are actively used by the verify gate and CLI.

| Schema | Purpose |
| :--- | :--- |
| `completion-card.schema.json` | Validates the canonical completion card produced by agents. |
| `subagent-return.schema.json` | Validates structured return payloads from sub-agents. |
| `verify-event.schema.json` | Validates JSONL trace events emitted by the verify gate. |
| `pgv-advice.schema.json` | Validates advisory-only PGV guidance packets. |

## Compatibility schemas (v0.1)

These schemas are kept for interoperability with claim/evidence workflows and existing examples. They are not required for the minimal harness flow.

| Schema | Purpose |
| :--- | :--- |
| `claim.schema.json` | Validates a standalone claim artifact. |
| `evidence.schema.json` | Validates a standalone evidence artifact. |

## Future schemas (not shipped in v0.1)

The following schemas are reserved for upcoming releases. They are **not present** in the v0.1 runtime schema directory and should not be referenced by v0.1 tooling.

- `story.schema.json` (planned for v0.2)
- `test-matrix.schema.json` (planned for v0.2)
- `task.schema.json` (planned for v0.2)
- `feature-intake.schema.json` (planned for v0.2)
- `product-contract.schema.json` (planned for v0.2)
- `audit-report.schema.json` (planned for v0.5)
- `recovery.schema.json` (planned for v0.3)
- `decision-record.schema.json` (planned for v0.2)

## Validation engine

The CLI uses **Ajv** for JSON Schema validation. No additional validation library (e.g., Zod) is required.
