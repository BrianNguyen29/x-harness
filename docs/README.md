# Documentation

This directory contains public, user-facing reference material for x-harness.
Implementation plans, scratch roadmaps, and internal operating notes should stay
out of the public package unless they define runtime behavior that the CLI
enforces.

## Start Here

- [Getting Started](GETTING_STARTED.md) — conceptual primer (read this first)
- [Quickstart](QUICKSTART.md) — step-by-step setup and first verify
- [First Accepted Card](tutorials/first-accepted-card.md) — end-to-end beginner walkthrough
- [FAQ](FAQ.md)
- [Architecture](ARCHITECTURE.md)
- [Adapters](ADAPTERS.md)
- [Threat Model](THREAT_MODEL.md)

## Runtime Contract

- [Verify Gate](VERIFY_GATE.md)
- [Runtime Contract](RUNTIME_CONTRACT.md)
- [Admission Policy](ADMISSION_POLICY.md)
- [Schemas](SCHEMAS.md)
- [Recovery](RECOVERY.md)
- [Packets](PACKETS.md)
- [Boundary](BOUNDARY.md) — deterministic path-glob + import-regex policy enforcement
- [Intake](INTAKE.md) — task intake tiering and product-intent records
- [Decision](DECISION.md) — lightweight decision memory records (ADR-lite)
- [Evidence Provenance](EVIDENCE_PROVENANCE.md) — command evidence, CI binding, checksums, and attestation guidance

## Operations

- [CI](CI.md) — Go-native primary gates with TypeScript compatibility parity, race/fuzz, release signing, and smoke gates
- [Report Formats](REPORT_FORMATS.md)
- [Cleanup](CLEANUP.md)
- [Release Security](RELEASE_SECURITY.md)
- [Release Candidate](RELEASE_CANDIDATE.md) — Release requirements and evidence floor
- [TypeScript Maintenance](TYPESCRIPT_MAINTENANCE.md) — Maintenance mode for the TypeScript CLI

## Reference

- [Conformance Strict Profile Spec](CONFORMANCE_STRICT_PROFILE.md) — Rules and verification criteria for `conformance run --profile strict`
- [CLI Commands](CLI_COMMANDS.md) — generated command maturity matrix
