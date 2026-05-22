# x-harness Roadmap

This document outlines the shipped milestones and the future planned capabilities for `x-harness`.

---

## 🚀 Shipped Milestones

### v0.1 — Minimal Verify-Gated Harness (Current Release)

The complete foundational core layer is fully built, tested, and shipped.

- **TypeScript-First CLI Application**: Implemented `init`, `add`, `handoff`, `verify`, `trace`, `report`, `clean`, `examples`, `context`, `doctor`, `recovery`, and `packet` commands.
- **Read-Only Verification Model**: Core engine executing deterministic local evaluation without code mutation.
- **Robust Validating Schemas**: Developed Ajv/JSON Schema for `completion-card`, `subagent-return`, `verify-event`, and `pgv-advice`.
- **Fail-Closed Admission Policy**: Integrated structured `policies/admission.yaml` and `policies/recovery.yaml` files.
- **Handoff & Completion Templates**: Provided clean structures for `light`, `standard`, and `deep` task dispatches.
- **Built-in Workflows & Examples**: Shipped 5 workflow reference configurations and 9 functional golden test case scenarios.
- **Multi-Platform Adapter Ecosystem**: Completed native integrations for Generic, Claude Code, Cursor, OpenCode, and Antigravity frameworks.

---

## 🗺️ Future Development Path

### v0.2 — Standard Product & Story Integration

Bring runtime automation and validation to standard product development documents.

- **Real Product Intake Validation**: Automatic JSON-schema mapping and syntax check for `feature-intake.md` templates.
- **Story & Test Matrix Runtime Support**: Automated correlation checking between declared implementation stories and verified test matrix columns.
- **Decision Record Linking**: Validate that architectural changes link to corresponding `DECISION_RECORD.md` templates.

### v0.3 — Richer Recovery & Automated Initialization

Strengthen the debugging paths and automate the setups across platforms.

- **Interactive Handoff Recovery**: Suggested console prompts dynamically outputting git command recovery lines (e.g. autocompiling a patch or auto-typechecking problem files).
- **Interactive Multi-Platform Adapter Init**: CLI commands to selectively setup individual adapters (e.g. `node packages/cli/dist/index.js init --adapter claude-code` in this repository).
- **Dynamic Policy Customization**: Interactive terminal prompts to easily modify admission threshold properties in `admission.yaml`.

### v0.4 — Deep Platform Integration

Direct native integrations to elevate developer environments.

- **GitHub Action verify-gate runner**: A lightweight, pre-packaged CI action that runs read-only verification on Pull Requests.
- **VSCode / Cursor Status Bar Helper**: A compact editor extension to display x-harness card verification state in the status bar.
- **Claude Code MCP Integration**: A model context protocol server exposing `x-harness verify` and `doctor` tools directly to MCP-enabled models.

### v0.5 — Full Audit & Deep Governance

Provide tooling for compliance-heavy enterprise pipelines.

- **Automated Audit Reports Compilation**: Generate complete, cryptographic-grade compliance files proving exact verification outcomes and replayability traces.
- **Governance Approvals Gateways**: Multi-owner validation keys in deep-tier completion cards.
- **Custom Policy Plugins**: Enable loading external JavaScript/TypeScript policy files to extend standard YAML check lists.

### v1.0 — Production-Stable Harness

- Freeze CLI command signatures, template formats, and schema keys.
- Complete documentation handbook and stable API references.
