# x-harness Repository Contract

x-harness is a lightweight, verify-gated harness for AI-agent and sub-agent workflows.

## Core Rule

> Completion is admitted, not merely claimed.

Agents may perform work and propose completion. Agents may not self-admit completion. A result only counts as done after a read-only verification gate admits it.

## Design Principles

- **File-first**: The source of truth is repository files — Markdown templates, JSON schemas, YAML policies, examples, and adapters. The CLI validates and generates files, but does not replace the files as the canonical contract.
- **Lightweight**: No daemon, database, server, MCP, or AI-specific runtime is required.
- **Go-native, file-first**: The Go CLI is the canonical runtime; TypeScript provides source-checkout compatibility. Python utilities, if ever added, must live under `legacy/python/` or `tools/experimental/` and be marked non-canonical.
- **Tiered**: Use `light`, `standard`, and `deep` for task handoffs. Do not use `small`, `medium`, or `large` in active runtime handoffs.
- **Verify-gated**: Verification is read-only and must not edit source files.
- **Advisory PGV**: PGV is optional and advisory-only. It never overrides verify and never grants admission authority by default.

## Workflow

The harness workflow is intentionally narrow:

```txt
Task
  -> Handoff Template
  -> Agent Result
  -> Completion Card
  -> Read-only Verify
  -> Accepted / Withheld
```

## Canonical Tiers

- **light**: Narrow, low-ceremony tasks. One clear objective, read-only or nearly read-only, one to three files.
- **standard**: Normal multi-step work. Bounded synthesis, two to four sources, clear but lightweight verification plan. Evidence scope (verifies/does_not_verify/untested_regions) is recommended.
- **deep**: High-stakes work only. Multiple dependencies, rollback policy, execution controls, state read/write sets, and governance approval if required. Must not become the default.

## Completion Semantics

### Candidate Completion

A result becomes a completion candidate when:

```yaml
result:
  fix_status: fixed
verification:
  status: passed
```

This does not mean the task is accepted.

### Accepted Completion

A task is accepted only when the verify gate emits:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

### Withheld Completion

These outcomes are always withheld:

```yaml
failed:
  acceptance_status: withheld
blocked:
  acceptance_status: withheld
skipped:
  acceptance_status: withheld
timeout:
  acceptance_status: withheld
error:
  acceptance_status: withheld
```

## CLI Commands

> **Local Development**: build the native Go CLI with `go build ./cmd/x-harness` and run `./x-harness <command>`. The TypeScript compatibility CLI remains available after `npm run build` via `node packages/cli/dist/index.js <command>`.

```bash
# Beginner-friendly actions (primary interface)
xh check --card completion-card.yaml
xh prepare --json
xh recover --errors "tests failed"
xh doctor
xh actions
xh status
xh reset --confirm
xh init --minimal

# Advanced commands
xh handoff standard --title "Fix bug"
xh report --metrics --card completion-card.yaml
xh packet create --card completion-card.yaml
```

Core command set includes:
- Beginner actions: `check` (alias for verify), `prepare` (alias for handoff readiness), `recover` (alias for recovery suggest), `doctor`, `actions`, `status`, `reset`, `init`, `add`, `start`, `learn`, `quick`, `run`, `ci`
- Advanced commands: `handoff`, `verify`, `trace`, `report`, `clean`, `examples`, `context`, `benchmark`, `recovery`, `packet`, `intake`, `governance`, `intervention`, `prediction`, `components`, `evidence`, `episode`, `attribution`, `permissions`, `evolve`, `export`, `import`, `frozen`, `federation`, `approval-risk`, `agent-profile`, `cost`, `profile`, plus Go-only `contract`, `explain`, `conformance`, `release`, `boundary`, `policy`, `scan`, `card`, `readiness`, `adapters`, `repair`, `uninstall`

Run `xh --help` for beginner-friendly commands. Use `xh --help-all` for the full command registry.

## Repository Structure

```txt
README.md
AGENTS.md
X_HARNESS.md
package.json
tsconfig.base.json

docs/
  README.md
  QUICKSTART.md
  VERIFY_GATE.md
  RUNTIME_CONTRACT.md
  ADMISSION_POLICY.md
  SCHEMAS.md
  ADAPTERS.md
  CI.md
  RELEASE_SECURITY.md
  (See docs/ for additional reference pages)


templates/
  SUBAGENT_TASK_light.md
  SUBAGENT_TASK_standard.md
  SUBAGENT_TASK_deep.md
  COMPLETION_CARD.md
  HARNESS_CHANGE_CONTRACT.md

schemas/                          # Key schemas (published contract; do not edit directly for runtime)
  completion-card.schema.json
  subagent-return.schema.json
  verify-event.schema.json
  pgv-advice.schema.json
  claim.schema.json
  evidence.schema.json
  packet.schema.json              # Packet chain schema
  contract-oracle.schema.json
  boundary-policy.schema.json     # Boundary policy for `xh boundary`

policies/
  admission.yaml
  approval-risk.yaml
  authority.yaml
  boundaries.yaml                 # Boundary policy consumed by `xh boundary`
  classifier.yaml
  cleanup.yaml
  contract-oracle.yaml
  cost-budget.yaml
  denominator.yaml
  escalation.yaml
  evidence.yaml
  federation.yaml
  intake.yaml
  mutation-guard.yaml
  ownership.yaml
  permissions.yaml
  pgv.yaml
  recovery.yaml
  rollback.yaml
  scanner.yaml
  stale-ground.yaml
  context-floor.yaml

examples/
  00-minimal/
  01-solo-agent/
  02-assisted-agent/
  03-multi-agent/
  04-blocked-verification/
  golden/
    regression/
      success-light/
      success-standard-scoped-evidence/
      blocked-missing-evidence/
      blocked-missing-evidence-scope/
      blocked-contract-oracle/
      failed-invalid-status/
      blocked-tier-downgrade/
      blocked-weak-prediction/
    capability/
      deep-approval-required/
      failed-typecheck-recovery-route/
      multi-agent-success/
      withheld-partial-fix/
    adversarial/
      blocked-missing-done-checklist/
      standard-approval-missing/
      standard-approval-present/
    conformance-strict/
      success-strict/
      blocked-strict-mutation-guard/
    recovery/
      routes.yaml
  actions/
    x-harness-verify/            # GitHub Actions composite action

adapters/
  generic/
  claude-code/
  cursor/
  opencode/
  antigravity/                    # Antigravity-specific constraints & workflows
  codex/                          # Codex adapter (repo-root AGENTS.md)
```

**Schema Canonical Strategy**: Root `schemas/` is the published contract. Runtime copies live in `packages/cli/schemas/`. Keep both copies synchronized when schema contracts change.
