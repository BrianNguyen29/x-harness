# Audit Roadmap

This document tracks the post-merge audit recommendations for x-harness and maps them to concrete deliverables, owners, and priority tiers.

## P1 — Immediate (Low-Risk Tooling & Docs)

These items are report-only, additive, and do not change runtime admission semantics.

- [x] **Audit roadmap** — this document (`docs/AUDIT_ROADMAP.md`)
- [x] **Schema/Policy sync check** — local script (`scripts/check-schema-policy-sync.mjs`) that compares root `schemas/` and `policies/` against `packages/cli/` copies, reports mismatches, and exits non-zero on drift.
- [x] **Security audit workflow** — CI workflow (`.github/workflows/security-audit.yml`) running `npm audit`, `govulncheck`, and optional secret-scan steps on a scheduled cadence.
- [x] **Policy authority matrix** — human-readable doc (`docs/POLICY_AUTHORITY_MATRIX.md`) derived from `policies/authority.yaml`.
- [x] **Coverage reporting** — report-only CI workflow (`.github/workflows/coverage-report.yml`) producing `npm run test:coverage` and Go coverage artifacts, plus documented local commands in this doc.

## P2 — Near-Term (CI Hardening & Drift Detection)

These items may require small changes to existing workflow files or policy manifests.

- [x] **Sync check in CI** — added as a step in `.github/workflows/verify-gates-supplemental.yml` (`node scripts/check-schema-policy-sync.mjs`) so schema/policy drift blocks merge. Verified as a supplemental gate, not required to gate merge until branch protection settings explicitly include `verify-gates-supplemental`.
- [ ] **Dependency update automation** — deferred. Requires `dependabot.yml` configuration and alert routing verification. Owner: user.
- [ ] **SBOM signing** — deferred. `sbom.yml` currently generates CycloneDX but does not attach a signed attestation or Sigstore bundle. SLSA provenance starter workflow (`slsa-provenance.yml`) is available as a stepping-stone for Go binary provenance, but does not sign the SBOM itself. Owner: user.
- [ ] **Admission engine drift test** — deferred. Needs automated comparison between Go-native and TypeScript compatibility admission decisions for golden examples. Owner: user.
- [ ] **Context floor enforcement** — deferred. Needs moving `policies/context-floor.yaml` from advisory to runtime-enforced in at least one profile. Owner: user.

## P3 — Future (Strategic & Governance)

These items involve design decisions, cross-team coordination, or larger refactors.

- [ ] **Policy change audit log** — deferred. Needs append-only log of every policy/schema change with author, timestamp, and PGV advice. Owner: user.
- [ ] **Intervention chain verification** — deferred. Needs end-to-end trace that a protected-path edit was preceded by a valid `intervention` record and approval receipt. Owner: user.
- [ ] **Agent-profile attestation** — deferred. Needs signed attestation that an agent profile was not modified between task start and completion. Owner: user.
- [ ] **Frozen manifest integrity gate** — deferred. Needs CI gate that verifies `frozen-manifest.schema.json` against a committed hash before release. Owner: user.
- [ ] **Oracle contract formalization** — deferred. Needs moving `policies/contract-oracle.yaml` from advisory to a formally checked contract with bounded execution controls. Owner: user.
- [ ] **Evolution constitution ratification** — deferred. Needs governance workflow for `schemas/evolution-constitution.schema.json` changes requiring multi-party approval. Owner: user.

## Running Checks Locally

### Schema / Policy Sync

```bash
node scripts/check-schema-policy-sync.mjs
```

The script compares root `schemas/` and `policies/` to their `packages/cli/` counterparts. It reports:

- **Missing files** — files present in root but absent in `packages/cli/`
- **Content mismatches** — files that differ byte-for-byte
- **Extra files** — files present in `packages/cli/` but absent in root (informational)
- **Ignored files** — files listed in `scripts/sync-ignore.json` if it exists

Exit code is `0` when all tracked files match, and `1` when any tracked file differs or is missing.

### Security Audit

```bash
# Node dependencies
npm audit --audit-level=high

# Go dependencies
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Coverage

```bash
# TypeScript / Node coverage
npm run test:coverage

# Go coverage (report-only, stdout)
go test -cover ./...

# Go coverage (detailed HTML)
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o go-coverage.html
```

## CI Workflows

| Workflow | File | Trigger | Purpose |
|----------|------|---------|---------|
| x-harness Verify | `.github/workflows/x-harness-verify.yml` | `push` / `pull_request` | Primary build/test/verify gate |
| x-harness Verify Gates Supplemental | `.github/workflows/verify-gates-supplemental.yml` | `push` / `pull_request` | Docs drift, examples, conformance, benchmark, schema/policy sync |
| Security Audit | `.github/workflows/security-audit.yml` | `schedule` / `workflow_dispatch` | Vulnerability scanning (`npm audit`, `govulncheck`) |
| Coverage Report | `.github/workflows/coverage-report.yml` | `schedule` / `workflow_dispatch` | Report-only coverage artifact generation |
| CodeQL | `.github/workflows/codeql.yml` | `push` / `pull_request` / `schedule` | Static analysis |
| SBOM | `.github/workflows/sbom.yml` | `workflow_dispatch` / `pull_request` | CycloneDX SBOM generation |
| Scorecard | `.github/workflows/scorecard.yml` | `push` / `schedule` | OpenSSF Scorecard |
| SLSA Provenance Starter | `.github/workflows/slsa-provenance.yml` | `workflow_dispatch` | Artifact preparation for SLSA Level 3 (disabled by default)

## References

- `policies/authority.yaml` — authority classes and protected paths
- `docs/THREAT_MODEL.md` — trust boundaries and attacker capabilities
- `docs/ADMISSION_POLICY.md` — admission evidence floor and rejection conditions
- `docs/CI.md` — CI integration details and job matrices
- `scripts/check-schema-policy-sync.mjs` — deterministic sync checker
