# ClaimGate

ClaimGate is a lightweight, verify-gated operating harness for AI-agent workflows.

It helps teams turn human intent into agent-ready work, route that work through tiered sub-agent handoffs, and prevent premature completion claims with a read-only verification gate.

## Core principle

**Completion is admitted, not merely claimed.**

Most harnesses help agents decide what to build. ClaimGate also helps orchestrators decide whether the result is allowed to count as done.

## What ClaimGate provides

- Product/spec intake.
- Product contract, story packet, test matrix, and decision record templates.
- Tiered sub-agent handoff templates: `light`, `standard`, `deep`.
- Completion card for low-overhead work.
- Claim and evidence packets for standard/deep work.
- Read-only verify-gated admission.
- Fail-closed outcomes: failed, blocked, skipped, timeout, error are withheld.
- PGV advisory-only governance.
- JSONL trace events.
- TypeScript-first CLI.
- Adapter packs for OpenCode, Claude Code, Cursor, Antigravity, and generic `AGENTS.md` workflows.

## What ClaimGate is not

ClaimGate is not a full autonomous agent framework. It is not a benchmark. It is not a safety guarantee. It does not replace tests. It does not treat `fix_status: fixed` as accepted completion.

## Modes

```bash
npx claimgate init --minimal
npx claimgate init --standard
npx claimgate init --full --adapters generic,opencode,claude-code,cursor,antigravity
```

Minimal mode installs only the core agent contract, runtime contract, verify gate, and three handoff templates.

Standard mode adds product intake, story packets, test matrix, completion cards, claim/evidence schemas, and verification artifacts.

Full mode adds adapters, policies, examples, GitHub Actions, schemas, and runtime-audit guidance.

## End-to-end flow

```txt
Human intent
  -> Feature Intake
  -> Product Contract Delta
  -> Story Packet
  -> Tier Selection
  -> Sub-Agent Handoff
  -> Agent Work
  -> Claim/Evidence
  -> Read-only Verify Gate
  -> Accepted / Withheld
  -> Test Matrix + Decision Record + Trace + Report
```

## Canonical tiers

Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

## TypeScript-first tooling

ClaimGate core tooling is TypeScript-first.

```bash
npm install
npm run build
npx claimgate verify
npx claimgate doctor
npx claimgate report
```

Python is not part of the canonical core tooling path. Python utilities, if ever added, must live under `legacy/python/` or `tools/experimental/` and must be marked non-canonical.
