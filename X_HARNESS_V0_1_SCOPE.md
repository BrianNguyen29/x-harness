# x-harness v0.1 Scope and Implementation Goals

## 1. Positioning

**x-harness** is a lightweight, verify-gated harness for AI-agent and sub-agent workflows.

It adds one operational rule to agentic development:

> Completion is admitted, not merely claimed.

Agents may perform work, propose completion, and return evidence. They may not self-admit completion. A result only counts as done after a read-only verification gate admits it.

x-harness is designed to be easy for newcomers to adopt in an existing repository within minutes. It must remain file-first, lightweight, and understandable without requiring a daemon, database, MCP server, or complex runtime.

## 2. What x-harness Is

x-harness is:

- A lightweight repository harness for AI-agent workflows.
- A file-based contract for task handoff, result evidence, and completion admission.
- A verify-gated workflow that separates agent claims from accepted completion.
- A tiered prompt and artifact system using `light`, `standard`, and `deep`.
- A TypeScript CLI that helps initialize, verify, inspect, and report harness state.
- A set of templates, schemas, policies, examples, and adapter docs for common AI coding tools.

## 3. What x-harness Is Not

x-harness is not:

- A full autonomous agent framework.
- A benchmark.
- A safety guarantee.
- A replacement for tests.
- A mandatory deep-governance system.
- A production reliability claim.
- A Python-first toolchain.
- A daemon, database, or server requirement.
- A system where PGV or advisory checks can accept completion by default.

## 4. v0.1 Scope

The v0.1 scope is intentionally narrow:

> One rule, one card, one verify command.

The goal is to make x-harness immediately useful without overengineering.

### Included in v0.1

- Public-facing README.
- `AGENTS.md` agent contract.
- `X_HARNESS.md` repository contract.
- `light`, `standard`, and `deep` handoff templates.
- A single completion artifact: `COMPLETION_CARD.md`.
- A read-only verification model.
- A minimal admission policy.
- Minimal JSON schemas.
- TypeScript CLI scaffold.
- Generic, Claude Code, Cursor, and OpenCode adapter docs.
- A few examples covering minimal, light, standard, and blocked verification flows.

### Excluded from v0.1

These are deliberately deferred to avoid heaviness:

- Full product intake lifecycle.
- Product contract management.
- Story packet lifecycle.
- Test matrix lifecycle.
- Decision records.
- Deep audit reports.
- Full recovery and rollback automation.
- MCP server.
- Runtime consistency audit.
- Advanced config contract linting.
- PGV runtime scoring.
- Automatic adapter installation.
- Multi-agent orchestration engine.
- Database-backed trace storage.

These may appear later as optional extensions, not as part of the default v0.1 workflow.

## 5. Core Workflow

The v0.1 workflow is:

```txt
Task
  -> Handoff Template
  -> Agent Result
  -> Completion Card
  -> Read-only Verify
  -> Accepted / Withheld
```

The full future workflow may become:

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

However, v0.1 must not force this full workflow on new users.

## 6. Core Invariants

x-harness must protect these invariants:

1. Completion is admitted, not merely claimed.
2. Agents may propose completion but may not self-admit completion.
3. `fix_status: fixed` is only a completion candidate.
4. `verification.status: passed` is only supporting evidence, not accepted completion.
5. Accepted completion requires a read-only verify gate success.
6. Verification must not edit source files.
7. Ambiguous cases are withheld, not accepted.
8. Failed, blocked, skipped, timeout, and error outcomes map to withheld.
9. Blocked is a valid outcome and must include a next owner or next action.
10. PGV is advisory-only by default.
11. `light`, `standard`, and `deep` are the only canonical tier labels.
12. Runtime handoffs must not use `small`, `medium`, or `large`.
13. Metrics must not confuse verify-event success with task-level success or production reliability.
14. The default path must stay lightweight.

## 7. Completion Semantics

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

These outcomes must always be withheld:

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

## 8. Tier Model

x-harness uses three canonical tiers.

### light

Use `light` for narrow, low-ceremony tasks.

Appropriate when:

- The task has one clear objective.
- The task is read-only or nearly read-only.
- The agent needs to inspect one to three files.
- The cost of being wrong is low to moderate.
- The answer should be short and immediately usable.

Avoid `light` when:

- The task spans many files or systems.
- Significant edits are expected.
- Multiple options must be compared.
- Deep verification, rollback, or escalation is needed.

### standard

Use `standard` for normal multi-step work.

Appropriate when:

- The task requires bounded synthesis.
- The agent compares two to four sources, modules, APIs, or options.
- The output must be synthesis-ready.
- Implementation needs a clear but lightweight verification plan.
- Some uncertainty must be surfaced explicitly.

### deep

Use `deep` only when the cost of being wrong is high.

Appropriate when:

- The task is high-stakes.
- The task has multiple dependencies or side effects.
- The task needs execution controls.
- The task needs rollback policy.
- The result will drive an important decision.

Deep must not become the default.

## 9. Minimal Repository Structure

The v0.1 repository should use this structure:

```txt
x-harness/
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

  templates/
    SUBAGENT_TASK_light.md
    SUBAGENT_TASK_standard.md
    SUBAGENT_TASK_deep.md
    COMPLETION_CARD.md
    VERIFY_REPORT.md

  schemas/
    completion-card.schema.json
    subagent-return.schema.json
    verify-event.schema.json
    pgv-advice.schema.json

  policies/
    admission.yaml

  packages/
    cli/
      package.json
      tsconfig.json
      src/
        index.ts
        commands/
          init.ts
          verify.ts
          doctor.ts
          report.ts
        core/
          paths.ts
          templates.ts
          admission.ts
          verify.ts
          report.ts
        validators/
          completionCard.ts
          subagentReturn.ts
          verifyEvent.ts

  adapters/
    generic/
      AGENTS.md
    claude-code/
      CLAUDE.md
      skills/
        verify/SKILL.md
    cursor/
      rules/
        x-harness.mdc
    opencode/
      README.md
      verify-agent.md

  examples/
    00-minimal/
    01-light-task/
    02-standard-task/
    03-blocked-verification/

  legacy/
    python/
      README.md
```

Python must not be part of the primary implementation path. If any Python tools exist, they must be placed only under `legacy/python/` or `tools/experimental/` and marked as non-canonical.

## 10. TypeScript CLI Scope

The canonical tooling is a TypeScript CLI.

### Required v0.1 Commands

```bash
npx x-harness init --minimal
npx x-harness verify
npx x-harness doctor
npx x-harness report
```

### Optional Future Commands

These may be added later:

```bash
npx x-harness init --standard
npx x-harness init --full
npx x-harness add product-layer
npx x-harness add adapters claude-code,cursor,opencode
npx x-harness intake new
npx x-harness story new
npx x-harness trace
```

## 11. `init --minimal` Requirements

`npx x-harness init --minimal` must create:

```txt
AGENTS.md
X_HARNESS.md
docs/VERIFY_GATE.md
docs/RUNTIME_CONTRACT.md
templates/SUBAGENT_TASK_light.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
templates/COMPLETION_CARD.md
policies/admission.yaml
```

It must not generate product-layer documents, full adapters, CI, or deep governance artifacts unless explicitly requested in a later version.

### Init Safety

The init command must support safe file behavior:

- It must not silently overwrite existing files.
- It should support `--dry-run`.
- It should support `--merge`.
- It should support `--force` with a clear warning.
- On conflict, it should list conflict paths and stop unless merge or force is provided.

## 12. Completion Card

The completion card is the primary v0.1 artifact.

Example:

```yaml
id: CC-001
task_id: TASK-001
tier: light

owner: implementation-worker
accountable: user

claim:
  fix_status: fixed
  summary: "Implemented the requested change."

evidence:
  files_changed: []
  commands_ran: []
  key_outputs: []

verification:
  status: passed
  checks: []

admission:
  outcome: pending
  acceptance_status: withheld

handoff:
  next_action: "Run npx x-harness verify."
  owner: user
```

The completion card should be easy to write manually, easy for an AI agent to fill, and easy for the CLI to validate.

## 13. Read-only Verify Gate

`npx x-harness verify` must be deterministic-first and read-only.

It may inspect:

- Handoff templates.
- Completion cards.
- Sub-agent return blocks.
- Schemas.
- Admission policy.
- Verify reports.
- Optional trace events.

It must not edit implementation files.

### v0.1 Verification Checks

The verify command should check:

- Tier is one of `light`, `standard`, `deep`.
- Runtime aliases `small`, `medium`, `large` are not used.
- Completion card has `owner` and `accountable`.
- `claim.fix_status` is one of `fixed`, `partial`, `not_fixed`.
- `verification.status` is one of `passed`, `failed`, `skipped`, `blocked`.
- `fix_status: fixed` does not automatically imply accepted completion.
- `verification.status: passed` does not automatically imply accepted completion.
- `admission.acceptance_status` is valid.
- `blocked` includes a next action or owner.
- PGV, if present, is advisory-only.

## 14. Admission Policy

The minimal admission policy should encode:

```yaml
version: 1

candidate_completion:
  required:
    - claim.fix_status: fixed
    - verification.status: passed

success_requires:
  - owner_present
  - accountable_present
  - evidence_present
  - verify_gate_invoked
  - verifier_read_only
  - no_unresolved_blocker
  - no_active_recovery

reject_success_if:
  fix_status:
    - partial
    - not_fixed
  verification_status:
    - failed
    - skipped
    - blocked
  stale_ground: true
  active_recovery: true
  timeout: true
  error: true

outcome_mapping:
  success:
    acceptance_status: accepted
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

## 15. PGV Scope

PGV is optional and advisory-only in v0.1.

If present, PGV may provide:

```yaml
pgv_advice:
  risk_score: 0.0
  claim_allowed: yes
  needs_escalation: none
  top_violation: null
  next_control_action: "No advisory action."
  verify_outcome_pred: passed
```

PGV must not:

- Admit completion.
- Override verify.
- Block runtime by default.
- Replace the admission policy.

## 16. Adapter Scope

v0.1 should include adapter documentation, not full automatic adapter installation.

### Generic Adapter

Provides a reusable `AGENTS.md` contract for any AI coding agent.

### Claude Code Adapter

Provides `CLAUDE.md` and a `verify/SKILL.md` that instructs the verifier to remain read-only.

### Cursor Adapter

Provides `.cursor/rules/x-harness.mdc` explaining:

- Use `light`, `standard`, `deep`.
- `fix_status: fixed` is candidate completion only.
- Verify gate is required before accepted completion.

### OpenCode Adapter

Provides documentation and a verify-agent prompt. Full OpenCode config generation is deferred.

## 17. Documentation Requirements

v0.1 must include:

```txt
README.md
AGENTS.md
X_HARNESS.md
docs/QUICKSTART.md
docs/PRINCIPLES.md
docs/MODES.md
docs/VERIFY_GATE.md
docs/RUNTIME_CONTRACT.md
docs/ADMISSION_POLICY.md
docs/PGV_ADVISORY.md
docs/ROADMAP.md
```

Documentation must be written for newcomers. It should avoid assuming the reader understands the paper, prior discussion, or previous repo names.

Every major doc should reinforce:

- Keep the default path light.
- Use `light` first.
- Use `standard` for normal structured work.
- Use `deep` only for high-risk work.
- Verify is read-only.
- Completion is admitted, not claimed.

## 18. Examples Requirements

v0.1 should include at least four examples.

### `examples/00-minimal/`

Shows the smallest possible setup.

### `examples/01-light-task/`

Shows a narrow task with a completion card and accepted verification.

### `examples/02-standard-task/`

Shows a structured task with standard handoff and verification checks.

### `examples/03-blocked-verification/`

Shows a blocked outcome with next owner and next action.

Each example should be readable without executing code.

## 19. GitHub Actions Scope

GitHub Actions are optional in v0.1. If included, keep them simple.

Recommended minimal workflow:

```yaml
name: x-harness

on:
  pull_request:
  push:
    branches: [main]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
      - run: npm install
      - run: npm run typecheck
      - run: npm run build
```

Do not add runtime audit, adapter smoke tests, or advanced CI in v0.1.

## 20. Denominator Discipline

x-harness must not overclaim its results.

Documentation and reports must avoid these conflations:

```txt
verify-event success != task-level success
verify-event success != production reliability
template lint pass != runtime correctness
PGV agreement != admission correctness
adapter smoke pass != safety guarantee
```

Reports should include this warning:

```txt
Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.
```

## 21. Acceptance Criteria for v0.1

The repo reaches v0.1 quality when all of the following are true:

1. The repo is named and documented as `x-harness`.
2. The README clearly states the core principle: completion is admitted, not merely claimed.
3. The default workflow is lightweight and understandable in under ten minutes.
4. The repo is file-first and does not require a daemon, database, server, MCP, or AI-specific runtime.
5. The TypeScript CLI is scaffolded with `init`, `verify`, `doctor`, and `report`.
6. Python is not part of core tooling.
7. The three canonical tiers are `light`, `standard`, and `deep`.
8. The runtime does not use `small`, `medium`, or `large`.
9. A completion card is the primary artifact.
10. Verify is read-only.
11. Completion candidate and accepted completion are clearly separated.
12. Failed, blocked, skipped, timeout, and error outcomes are withheld.
13. Blocked outcomes include a next owner or next action.
14. PGV is advisory-only.
15. Minimal schemas and admission policy exist.
16. Generic, Claude Code, Cursor, and OpenCode adapter docs exist.
17. At least four examples exist.
18. Documentation explicitly warns against overclaiming metrics.
19. Newcomers are not forced into product-intake, story-packet, or deep-audit workflows.
20. Future full-harness capabilities are documented as roadmap items, not default requirements.

## 22. Roadmap Beyond v0.1

### v0.2 — Standard Mode

Add optional product operating layer:

- Feature intake.
- Product contract.
- Story packet.
- Test matrix.
- Decision records.

### v0.3 — Trace and Recovery

Add:

- JSONL trace.
- Verify event emission.
- Recovery packet.
- Blocked task tracking.

### v0.4 — Adapter Pack

Add:

- Full OpenCode config examples.
- Claude Code skills.
- Cursor rules.
- Antigravity mission templates.
- Adapter init support.

### v0.5 — Full Mode

Add:

- GitHub Actions.
- Full schemas.
- Report generation.
- Optional audit reports.
- Optional deep-governance templates.

### v1.0 — Stable Contract

Stabilize:

- CLI commands.
- Schemas.
- Templates.
- Adapter contracts.
- Admission policy.
- Documentation.

## 23. Final Scope Statement

x-harness v0.1 is a lightweight verify-gated harness for AI-agent completion claims.

It should be simple enough for a new user to add to an existing repo quickly, but strict enough to prevent the most important failure mode: an agent claiming work is done without independent read-only admission.

The v0.1 design is intentionally constrained:

```txt
One rule: completion is admitted, not claimed.
One artifact: completion card.
One command: npx x-harness verify.
```

Everything else is optional, staged, and added only when the workflow needs it.
