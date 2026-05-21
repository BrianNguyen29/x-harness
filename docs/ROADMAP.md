# x-harness Roadmap

This document describes planned capabilities beyond v0.1. v0.1 users are not required to adopt any of these features.

## v0.1 — Minimal Harness

Current release. One rule, one card, one verify command.

- Public-facing README.
- `AGENTS.md` agent contract.
- `X_HARNESS.md` repository contract.
- `light`, `standard`, and `deep` handoff templates.
- `COMPLETION_CARD.md` completion artifact.
- Read-only verification model.
- Minimal admission policy.
- Minimal JSON schemas.
- TypeScript CLI scaffold.
- Generic, Claude Code, Cursor, and OpenCode adapter docs.
- Five examples: minimal, solo-agent, assisted-agent, multi-agent, blocked-verification.

## v0.2 — Standard Mode

Optional product operating layer:

- Feature intake.
- Product contract.
- Story packet.
- Test matrix.
- Decision records.

## v0.3 — Trace and Recovery

- JSONL trace.
- Verify event emission.
- Recovery packet.
- Blocked task tracking.

## v0.4 — Adapter Pack

- Full OpenCode config examples.
- Claude Code skills.
- Cursor rules.
- Antigravity mission templates.
- Adapter init support.

## v0.5 — Full Mode

- GitHub Actions.
- Full schemas.
- Report generation.
- Optional audit reports.
- Optional deep-governance templates.

## v1.0 — Stable Contract

Stabilize:

- CLI commands.
- Schemas.
- Templates.
- Adapter contracts.
- Admission policy.
- Documentation.

## Advanced / preview artifacts

The following are preview or advanced features. They are not required for v0.1 and do not redefine the default lightweight flow:

- Feature intake and story packets (v0.2 preview).
- Product contract and test matrix (v0.2 preview).
- Audit reports and decision records (v0.5 preview).
- `trace` command (v0.3 preview).
- `add` and `handoff` commands (v0.4 preview).
