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
| standard | Normal features | files_changed + command_evidence (evidence_scope recommended) |
| deep | Security-critical, architecture | files_changed + command_evidence + evidence_scope + untested_regions + remaining_risks + rollback_policy + execution_controls + state.read_set + state.write_set |

### Verifier is read-only

The verifier inspects files, diffs, and trace events. It never edits source files during verification.

### PGV is advisory-only

Pre-gate validation provides guidance but never overrides the verify gate.

## CLI commands

The CLI is invoked as `node packages/cli/dist/index.js <command> [options]`.

### Beginner-friendly actions (primary interface)

| Action       | Alias for              | Description                                              |
| :----------- | :--------------------- | :------------------------------------------------------- |
| **`prepare`** | `handoff readiness`   | Check if workspace is ready for agent task handoff        |
| **`check`**  | `verify`               | Run read-only verification against a completion card        |
| **`recover`** | `recovery suggest`     | Get recovery playbook suggestions from errors or trace     |
| **`doctor`** | (standalone)           | Validate workspace health and configuration                |
| **`actions`** | (standalone)           | List all beginner-friendly actions                        |
| **`status`** | `report` (no --metrics)| Show trace summary or card metrics                       |
| **`reset`**  | `clean --tmp --force` | Clean generated harness state (requires --confirm)        |

**Slash commands for agent adapters:** `/xh-check`, `/xh-prepare`, `/xh-recover`, `/xh-doctor`, `/xh-actions`, `/xh-status`, `/xh-reset`

### Beginner action examples

```bash
# Verify a completion card (primary beginner action)
node packages/cli/dist/index.js check --card completion-card.yaml
# or: node packages/cli/dist/index.js verify --card completion-card.yaml

# Check handoff readiness
node packages/cli/dist/index.js prepare --json
# or: node packages/cli/dist/index.js handoff readiness --json

# Generate recovery playbook
node packages/cli/dist/index.js recover --errors "tests failed" --outcome failed
# or: node packages/cli/dist/index.js recovery suggest --errors "tests failed" --outcome failed

# Check workspace health
node packages/cli/dist/index.js doctor

# Show trace summary
node packages/cli/dist/index.js status

# Reset harness state (safe cleanup)
node packages/cli/dist/index.js reset --confirm

# List all actions
node packages/cli/dist/index.js actions
```

### Advanced commands

```bash
# Initialize a workspace
node packages/cli/dist/index.js init --minimal

# Generate a handoff template
node packages/cli/dist/index.js handoff standard --title "Fix bug"

# Generate audit report
node packages/cli/dist/index.js report --trace-dir .x-harness/traces

# Add metadata helpers
node packages/cli/dist/index.js add <claim|evidence|completion-card> [key=value]

# Verify trace integrity
node packages/cli/dist/index.js trace verify-chain

# Create a claim packet from a completion card
node packages/cli/dist/index.js packet create --card completion-card.yaml

# Verify packet chain integrity
node packages/cli/dist/index.js packet verify-chain --task-id <task-id>

# Manage temporary artifacts
node packages/cli/dist/index.js clean [--tmp|--reset-card|--archive-success] [--force]

# List or copy built-in examples
node packages/cli/dist/index.js examples

# Show canonical context
node packages/cli/dist/index.js context [--verbose|--json|--refresh]
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

- `files_changed` (non-empty)
- `command_evidence`
- `evidence_scope_declared` (recommended, not required)
- `untested_regions_declared` (recommended, not required)

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
