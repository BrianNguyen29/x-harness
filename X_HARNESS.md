# x-harness Repository Contract

x-harness is a lightweight, verify-gated harness for AI-agent and sub-agent workflows.

## Core Rule

> Completion is admitted, not merely claimed.

Agents may perform work and propose completion. Agents may not self-admit completion. A result only counts as done after a read-only verification gate admits it.

## Design Principles

- **File-first**: The source of truth is repository files — Markdown templates, JSON schemas, YAML policies, examples, and adapters. The CLI validates and generates files, but does not replace the files as the canonical contract.
- **Lightweight**: No daemon, database, server, MCP, or AI-specific runtime is required.
- **TypeScript-first**: Core tooling is TypeScript. Python utilities, if ever added, must live under `legacy/python/` or `tools/experimental/` and be marked non-canonical.
- **Tiered**: Use `light`, `standard`, and `deep` for task handoffs. Do not use `small`, `medium`, or `large` in active runtime handoffs.
- **Verify-gated**: Verification is read-only and must not edit source files.
- **Advisory PGV**: PGV is optional and advisory-only. It never overrides verify and never grants admission authority by default.

## Workflow

The v0.1 workflow is intentionally narrow:

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

```bash
npx x-harness init --minimal
npx x-harness verify
npx x-harness doctor
npx x-harness report
npx x-harness report --metrics --card completion-card.yaml
```

## Repository Structure

```txt
README.md
AGENTS.md
X_HARNESS.md
package.json
tsconfig.base.json

docs/
  QUICKSTART.md
  PRINCIPLES.md
  MODES.md
  VERIFY_GATE.md
  RUNTIME_CONTRACT.md
  ADMISSION_POLICY.md
  PGV_ADVISORY.md
  ROADMAP.md
  RECOVERY.md
  METRICS.md

templates/
  SUBAGENT_TASK_light.md
  SUBAGENT_TASK_standard.md
  SUBAGENT_TASK_deep.md
  COMPLETION_CARD.md
  HARNESS_CHANGE_CONTRACT.md

schemas/
  completion-card.schema.json
  subagent-return.schema.json
  verify-event.schema.json
  pgv-advice.schema.json
  claim.schema.json
  evidence.schema.json

policies/
  admission.yaml
  recovery.yaml

examples/
  00-minimal/
  01-solo-agent/
  02-assisted-agent/
  03-multi-agent/
  04-blocked-verification/
  golden/
    success-light/
    success-standard-scoped-evidence/
    blocked-missing-evidence/
    blocked-missing-evidence-scope/
    failed-invalid-status/
    failed-typecheck-recovery-route/
    withheld-partial-fix/
    deep-approval-required/
    multi-agent-success/

adapters/
  generic/
  claude-code/
  cursor/
  opencode/
```
