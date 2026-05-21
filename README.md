# x-harness

x-harness is a lightweight, verify-gated harness for AI-agent workflows.

It helps teams route work through tiered sub-agent handoffs and prevent premature completion claims with a read-only verification gate.

## Core principle

**Completion is admitted, not merely claimed.**

Most harnesses help agents decide what to build. x-harness also helps orchestrators decide whether the result is allowed to count as done.

## What x-harness provides

- Tiered sub-agent handoff templates: `light`, `standard`, `deep`.
- Completion card for low-overhead work.
- Read-only verify-gated admission.
- Fail-closed outcomes: failed, blocked, skipped, timeout, error are withheld.
- PGV advisory-only governance.
- JSONL trace events.
- TypeScript-first CLI.
- Adapter docs for OpenCode, Claude Code, Cursor, and generic `AGENTS.md` workflows.

## What x-harness is not

x-harness is not a full autonomous agent framework. It is not a benchmark. It is not a safety guarantee. It does not replace tests. It does not treat `fix_status: fixed` as accepted completion.

## Modes

```bash
npx x-harness init --minimal
npx x-harness init --standard
npx x-harness init --full --adapters generic,opencode,claude-code,cursor
```

Minimal mode installs only the core agent contract, runtime contract, verify gate, and three handoff templates.

Standard mode adds examples, completion cards, claim/evidence schemas, and verification artifacts.

Full mode adds adapters, policies, examples, GitHub Actions, schemas, and runtime-audit guidance.

## End-to-end flow

```txt
Task
  -> Handoff Template
  -> Agent Result
  -> Completion Card
  -> Read-only Verify
  -> Accepted / Withheld
```

## Canonical tiers

Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

## TypeScript-first tooling

x-harness core tooling is TypeScript-first.

```bash
npm install
npm run build
npx x-harness verify
npx x-harness doctor
npx x-harness report
```

Python is not part of the canonical core tooling path. Python utilities, if ever added, must live under `legacy/python/` or `tools/experimental/` and must be marked non-canonical.

## Recovery routing

Blocked and failed verifications include a suggested recovery route:

```bash
npx x-harness verify --card completion-card.yaml --json
```

Look for `recovery.next_action` and `recovery.owner` in the JSON output to route the next step.

## ClaimGate compatibility

x-harness is the evolution of the ClaimGate concept. `CLAIMGATE.md` is preserved as a backward-compatible alias.
