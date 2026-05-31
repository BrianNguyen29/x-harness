# Documentation

This directory contains public, user-facing reference material for x-harness.
Implementation plans, scratch roadmaps, and internal operating notes should stay
out of the public package unless they define runtime behavior that the CLI
enforces.

## Start Here

- [Quickstart](QUICKSTART.md)
- [FAQ](FAQ.md)
- [Architecture](ARCHITECTURE.md)
- [Adapters](ADAPTERS.md)

## Runtime Contract

- [Verify Gate](VERIFY_GATE.md)
- [Runtime Contract](RUNTIME_CONTRACT.md)
- [Admission Policy](ADMISSION_POLICY.md)
- [Schemas](SCHEMAS.md)
- [Recovery](RECOVERY.md)
- [Packets](PACKETS.md)

## Operations

- [CI](CI.md) — TypeScript/Go dual-run, race/fuzz, release signing, and smoke gates
- [Report Formats](REPORT_FORMATS.md)
- [Cleanup](CLEANUP.md)
- [Release Security](RELEASE_SECURITY.md)
- [Release Candidate](RELEASE_CANDIDATE.md) — Release requirements and evidence floor
- [TypeScript Maintenance](TYPESCRIPT_MAINTENANCE.md) — Maintenance mode for the TypeScript CLI

## Reference

- [Conformance Strict Profile Spec](CONFORMANCE_STRICT_PROFILE.md) — Rules and verification criteria for `conformance run --profile strict`
