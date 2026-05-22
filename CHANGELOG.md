# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.0] - 2026-05-22

### Added

- **TypeScript-First CLI Tool**: Shipped a CLI application supporting `init`, `add`, `handoff`, `verify`, `trace`, `report`, `clean`, `examples`, `context`, `doctor`, `recovery`, and `packet` commands.
- **Read-Only Verification Gate**: Enforced strict read-only audit validations preventing verifier mutation of workspace source files.
- **Robust JSON Schemas**: Added Ajv/JSON Schema validation for completion cards (`completion-card`), sub-agent returns (`subagent-return`), verification events (`verify-event`), and advisory reports (`pgv-advice`).
- **Fail-Closed YAML Policies**: Integrated `policies/admission.yaml` and `policies/recovery.yaml` to enforce strict evidence floors under `light`, `standard`, and `deep` tiers.
- **Handoff Tasks and Card Templates**: Created standard markdown templates for task delegation and completion records tracking.
- **Comprehensive Adapters Suite**: Delivered native setup rules and workflows for Generic, Claude Code, Cursor, OpenCode, and Antigravity.
- **Scenario Reference Suite**: Shipped 5 system examples and 9 fully validated golden verification scenarios covering success, blocked evidence, typecheck routing, and deep human approval gates.

### Changed

- **Branding & Naming Unified**: Standardized all references to `x-harness`.
- **Cleaned v0.1 Scope**: Focused schemas, policies, and code execution flow to enforce local, deterministic, offline-first runtimes.
- **Upgraded Internal CLI Dependencies**: Bumped package files to modern Node.js versions (`>= 20`), Vitest (`^4.1.7`), Commander (`^12.1.0`), and Ajv (`^8.20.0`).

### Removed

- **Redundant Aliases**: Phased out legacy ClaimGate runtime variables. The file `CLAIMGATE.md` was a historical placeholder and is not present in this release.
- **Outdated Missions Placeholders**: Removed empty placeholder documents under `adapters/antigravity/missions/`.
- **Redundant Documentation**: Consolidated integration guidelines under `docs/ADAPTERS.md` and removed `docs/INTEGRATION.md`.
