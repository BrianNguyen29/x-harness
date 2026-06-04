# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- **`xh boundary {lint,check,explain}` (PR #59)**: Deterministic path-glob + import-regex boundary policy checker. Loads `policies/boundaries.yaml` (schema: `schemas/boundary-policy.schema.json`, mirrored to `packages/cli/schemas/boundary-policy.schema.json`); supports `lint`, `check --all|--changed`, and `explain <file>`; JSON and text output; V1 zero-false-positive on the x-harness repo with the shipped example policy. Boundary checks are opt-in (missing policy is a warning, not a failure).
- **`verify --contract-oracles`**: New opt-in flag that runs rule-based oracle assertions from `policies/contract-oracle.yaml` (or custom path via `--contract-oracles-policy`). Supports both `grep_rules` and `dependency_rules` (line-level import scanning). Default policy is empty/safe; no assertions run without a policy file.
- **Contract Oracle `dependency_rule` MVP**: Line-level import scanning rule type alongside `grep_rules`. Fields: `id`, `description`, `file_pattern`, `forbidden_imports`, optional `allowed_imports`, optional `exclude`, optional `message`. Detection scans for import-like lines (Go `import`, TypeScript/JavaScript `import ... from`, `require()`) and matches import paths against `forbidden_imports` substrings. No AST, package graph, or lockfile parsing.
- **withheld_reason migration-mode planning docs**: Added `### Migration modes` subsection to `docs/VERIFY_GATE.md` documenting compatibility-superset (current), transitional strict mode (`--strict-withheld-reason` flag, now implemented), and strict-only (future).
- **`verify --strict-withheld-reason`**: New opt-in flag that omits legacy `failure_class` and `failure_stage` fields from `withheld_reason` JSON/text output and shows schema enum `recoverability` instead of the legacy value. Default output unchanged.
- **Contract Oracle grep_rule MVP**: Added `contract-oracle.schema.json` for `x-harness contract check --json` output validation. Mirrored to `packages/cli/schemas/contract-oracle.schema.json`. Documented in `docs/SCHEMAS.md` feature schema table.
- **Strict-schema withheld-reason fixture**: Added `packages/cli/tests/fixtures/withheld-reason-strict.json` as a file-based strict-schema-valid sample, loaded by the corresponding schema test.
- **Strict-schema boundary documentation**: Added `Compatibility boundary` subsection to `docs/VERIFY_GATE.md` and footnote in `docs/SCHEMAS.md` explaining runtime superset vs strict schema target for `withheld_reason`.
- **Strict-schema validator tests for withheld-reason**: Added tests in `packages/cli/tests/schema.test.ts` confirming strict schema accepts canonical target (no legacy fields) and rejects runtime compatibility superset (legacy fields).
- **Typed withheld_reason output**: Go runtime now emits structured `class`, `stage`, and `owner` fields in `withheld_reason` output while preserving legacy `failure_class`/`failure_stage` for backward compatibility.
- **Schema recoverability field**: Added `schema_recoverability` to `withheld_reason` output as schema enum value (`automatic`, `manual`, `blocked`, `unknown`) derived from legacy `recoverability`. Legacy `recoverability` values unchanged for backward compatibility. Compatibility phase field.
- **Context Floor MVP**: `verify --context-floor` validates presence, tier, and file refs for context alignment. Anchors are stripped for file existence (hard anchor enforcement deferred).
- **Blocked context fixture**: Added `examples/golden/regression/blocked-missing-context-ref/` golden fixture covering `--context-floor` failure due to missing referenced file.
- **doctor --context**: Scans `examples` + `skills`, validates referenced files, skips cards without `context_alignment`, reports missing anchors as warnings only, counts unreadable/unparseable cards.
- **doctor --staleness / --overclaim**: Validates managed context freshness and hash; checks for overclaim phrases.
- **Schema additions**: `context-alignment.schema.json` (context alignment evidence) and `withheld-reason.schema.json` (typed withheld reason taxonomy) added to schema inventory.
- **Recovery hardening**: `context_floor_blocked` and `stale_ground` predicates now routed in recovery.
- **Documentation completeness pass**: Updated stale roadmap statuses for implemented strict/conformance features; added `prediction` to the standard-tier evidence floor in `docs/ADMISSION_POLICY.md`; added missing README for the `blocked-weak-prediction` golden fixture.

### Changed

- Roadmap explicitly marks the `release` conformance profile as deferred/planned; only `minimal` and `strict` profiles are currently implemented.
- `X_HARNESS.md` command list expanded to include `conformance`, `scan`, `adapters`, `card`, `profile`, `readiness`, `release`, and `benchmark`.

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
- **NPM package is Go-only**: The published npm package no longer includes the TypeScript `dist/` runtime. The wrapper (`bin/x-harness.js`) is now a thin Go-binary launcher. Runtime dependencies (`ajv`, `commander`, `fs-extra`, `yaml`) have been moved to `devDependencies`.

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
