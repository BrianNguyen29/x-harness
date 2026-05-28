# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.99.0-rc1] - 2026-05-28

### Added

- **Go-native CLI rewrite**: Introduced a Go CLI (`cmd/x-harness`) with parity-check against the TypeScript baseline. All primary commands are implemented in Go.
- **Release binary matrix**: Go release binaries built for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, and `windows/arm64`.
- **Sigstore signing**: Tagged releases sign Go binaries with cosign (keyless via GitHub OIDC).
- **CycloneDX SBOM**: Release workflow generates and attaches an SBOM.
- **Packed CLI Go smoke test**: Validates the Go binary path inside the npm tarball.
- **Frozen transfer compatibility**: Validates export, verify, and import of frozen bundles from the packed tarball.
- **Cross-platform smoke tests**: Smoke tests on `ubuntu-latest`, `macos-latest`, and `windows-latest` using the amd64 binaries.

### Changed

- **Primary runtime**: Go CLI is now the recommended local development runtime; TypeScript CLI remains the compatibility baseline.
- **Version**: Bumped to `0.99.0-rc1` to signal the release-candidate stage.

### Changed

- **NPM package is Go-only**: The published npm package no longer includes the TypeScript `dist/` runtime. The wrapper (`bin/x-harness.js`) is now a thin Go-binary launcher. Runtime dependencies (`ajv`, `commander`, `fs-extra`, `yaml`) have been moved to `devDependencies`.
- **Node fallback source-checkout only**: The wrapper falls back to `node dist/index.js` only when the file exists (source checkout or custom build). In the published package, missing Go binaries result in an error instead of a silent Node fallback.

### Known Limitations

- **NPM publish token**: The release workflow requires the `NPM_TOKEN` secret for npm publish; the latest RC run failed publish due to a missing token.
- **Arm64 smoke gap**: Arm64 binaries are built but not smoke-tested in CI because arm64 runners are not universally available.

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
