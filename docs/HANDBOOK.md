# x-harness Technical Handbook

> A concise reference for operators and contributors.

## What x-harness is

A lightweight, offline-first, verify-gated harness for AI-agent workflows. It provides:

- Structured completion cards with evidence floors
- Read-only verification gates
- Deterministic admission policy
- Trace logging with hash-chain integrity

## What x-harness is NOT

- A daemon, database, server, or dashboard
- An auto-fix tool
- A self-admitting completion framework
- A broad lifecycle orchestrator

## Core concepts

### Completion is admitted, not claimed

Agents produce completion candidates. The read-only verifier decides admission based on repository policy.

### Canonical tiers

| Tier | Use case | Evidence floor |
|------|----------|----------------|
| light | Quick fixes, docs | files_changed + command_evidence or manual_rationale |
| standard | Normal features | light + evidence_scope + untested_regions |
| deep | Security-critical, architecture | standard + remaining_risks + rollback_policy + execution_controls |

### Verifier is read-only

The verifier inspects files, diffs, and trace events. It never edits source files during verification.

### PGV is advisory-only

Pre-gate validation provides guidance but never overrides the verify gate.

## CLI commands

```bash
# Verify a completion card
node packages/cli/dist/index.js verify --card completion-card.yaml

# Generate a handoff template
node packages/cli/dist/index.js handoff standard --title "Fix bug"

# Check workspace health
node packages/cli/dist/index.js doctor

# Generate audit report
node packages/cli/dist/index.js report --trace-dir .x-harness/traces

# Verify trace integrity
node packages/cli/dist/index.js trace verify-chain

# Generate recovery playbook
node packages/cli/dist/index.js recovery suggest --errors "tests failed" --outcome failed

# Check handoff readiness
node packages/cli/dist/index.js handoff readiness --interactive
```

## File organization

```
.
├── AGENTS.md                    # Agent contract
├── policies/
│   └── admission.yaml           # Admission policy
├── templates/
│   ├── COMPLETION_CARD.md
│   └── SUBAGENT_TASK_{light,standard,deep}.md
├── docs/                        # Documentation
├── schemas/                     # JSON schemas
└── .x-harness/
    └── traces/
        └── events.jsonl         # Trace log
```

## Evidence floor by tier

### light

- `files_changed` (non-empty)
- `command_evidence` or `manual_rationale`

### standard

- light requirements
- `evidence_scope_declared` (verifies / does_not_verify)
- `untested_regions_declared`

### deep

- standard requirements
- `remaining_risks_declared`
- `execution_controls_present`
- `rollback_policy_present`

## Recovery routing

When verification is blocked or failed, x-harness suggests recovery routes:

| Predicate | Next action | Owner |
|-----------|-------------|-------|
| evidence_missing | Attach validation evidence | implementation-worker |
| typecheck_failed | Fix types | implementation-worker |
| test_failed | Diagnose failing behavior | implementation-worker |
| approval_missing | Request human approval | user |
| verifier_not_read_only | Rerun with read-only verifier | admission-verifier |

## Trace hash chain

Trace events include `previous_hash` and `event_hash` fields forming a chain. Use `trace verify-chain` to detect tampering. Legacy events without hashes are skipped during verification.

## Non-goals

- npm publishing (not yet)
- Network calls
- Auto-commit behavior
- Database / server / dashboard
- Self-admitted completion
