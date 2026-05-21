# x-harness Implementation Blueprint

**Repository name:** `x-harness`  
**Document purpose:** This file is the implementation brief for AI agents. It defines the scope, goals, architecture, repository structure, required files, CLI behavior, adapter behavior, acceptance criteria, and non-negotiable rules for implementing `x-harness`.

---

## 1. Product Positioning

`x-harness` is a lightweight verify-gated harness for AI-agent workflows.

It is designed to work with solo AI agents, assisted single-agent workflows using deterministic checks, and multi-agent workflows with separated worker/verifier roles. It should support AI development tools such as Antigravity, Claude Code, Cursor, OpenCode, and generic `AGENTS.md`-compatible agents.

`x-harness` is not another agent platform. It is a repo-level contract that tells any AI agent when it is allowed to say “done”.

### Core positioning statement

```txt
x-harness is a lightweight verify-gated harness for AI-agent workflows, usable by one agent but strongest with separated worker/verifier roles.
```

### Core rule

```txt
Completion is admitted, not merely claimed.
```

An agent may claim that work is fixed. It may not self-admit that the work is accepted. Accepted completion requires a read-only verification gate.

---

## 2. Design Philosophy

`x-harness` combines four ideas.

First, long-running agent workflows need structured artifacts. Agents should not rely only on chat history or free-form progress notes. They need handoff templates, completion cards, verification records, and compact repo-local instructions.

Second, harnesses should preserve context through files. The repository should be the source of truth. Agents should read short instructions first, then progressively load deeper docs, templates, and policies only when needed.

Third, the verifier must be separated from the worker role as much as possible. In multi-agent workflows this means different agents. In single-agent workflows this means separated phases: work mode, claim mode, read-only verify mode, final response.

Fourth, the design must preserve the paper’s core values:

- execution is not acceptance;
- workers propose completion, verifiers admit or withhold;
- verification is read-only;
- ambiguous cases fail closed;
- blocked is a valid outcome;
- evidence is structured;
- PGV or advisory checks do not become admission authority by default;
- metrics must not overclaim production reliability or task-level success.

---

## 3. Non-Goals

`x-harness` must not become heavy by default.

It is not:

- a full autonomous agent runtime;
- a benchmark;
- a safety guarantee;
- a replacement for tests;
- a database-backed task manager;
- a required MCP server;
- a mandatory multi-agent system;
- a long manual stuffed into agent context;
- a framework that forces deep governance on small tasks.

The default workflow must remain small enough for a new user to adopt in 5–10 minutes.

---

## 4. Supported Operating Modes

`x-harness` supports three operating modes.

### 4.1 Solo Agent Mode

Use this when one AI agent performs the task.

Example tools:

- Claude Code single agent;
- Cursor Agent;
- Antigravity agent;
- OpenCode agent;
- ChatGPT/Codex-style coding agent.

Flow:

```txt
User task
  -> agent works
  -> agent creates completion-card.yaml
  -> agent switches to read-only verify mode
  -> agent runs npx x-harness verify
  -> final response: accepted or withheld
```

Solo mode reduces premature “done” claims. It does not provide fully independent verification.

Prompt example:

```txt
Use x-harness solo-agent mode.

Implement the requested change.
Do not say the task is done immediately after editing.
Create completion-card.yaml.
Run read-only verification with npx x-harness verify.
Only say completed if verification succeeds.
If verification is blocked or failed, report withheld completion with next action.
```

### 4.2 Assisted Agent Mode

Use this when one agent works, but deterministic tools provide evidence.

Examples:

- tests;
- typecheck;
- lint;
- schema validation;
- grep/glob checks;
- build command;
- repo-specific validation scripts.

Flow:

```txt
agent work
  -> tests/typecheck/lint/checks
  -> completion card
  -> x-harness verify
  -> accepted or withheld
```

This should be the recommended default for most users.

### 4.3 Multi-Agent Mode

Use this when roles are separated.

Typical roles:

- orchestrator;
- implementation worker;
- evidence collector;
- read-only admission verifier;
- reviewer/escalation role for deep tasks.

Flow:

```txt
orchestrator
  -> implementation-worker
  -> completion-card/evidence
  -> admission-verifier
  -> accepted/withheld
```

Multi-agent mode is the strongest form because worker and verifier are separated.

Prompt example:

```txt
Use x-harness multi-agent mode.

Agent 1: implement the requested change and create completion-card.yaml.
Agent 2: act as read-only admission-verifier and run npx x-harness verify.
The final response must say accepted only if verify outcome is success.
```

---

## 5. Scope for v0.1

`x-harness` v0.1 should be lightweight.

The v0.1 scope is:

```txt
one rule + one card + one verify command
```

Meaning:

```txt
One rule: completion is admitted, not claimed.
One card: completion-card.yaml.
One command: npx x-harness verify.
```

### Included in v0.1

- Public-facing `README.md`
- `AGENTS.md`
- `X_HARNESS.md`
- `docs/QUICKSTART.md`
- `docs/PRINCIPLES.md`
- `docs/MODES.md`
- `docs/RUNTIME_CONTRACT.md`
- `docs/VERIFY_GATE.md`
- `docs/ADMISSION_POLICY.md`
- `docs/PGV_ADVISORY.md`
- `docs/DENOMINATOR_POLICY.md`
- `docs/INTEGRATION.md`
- `docs/ADAPTERS.md`
- `docs/ROADMAP.md`
- `docs/FAQ.md`
- `templates/SUBAGENT_TASK_light.md`
- `templates/SUBAGENT_TASK_standard.md`
- `templates/SUBAGENT_TASK_deep.md`
- `templates/COMPLETION_CARD.md`
- `templates/VERIFY_REPORT.md`
- `schemas/completion-card.schema.json`
- `schemas/subagent-return.schema.json`
- `schemas/verify-event.schema.json`
- `schemas/pgv-advice.schema.json`
- `policies/admission.yaml`
- TypeScript CLI with `init`, `verify`, `doctor`, and `report`
- adapter docs for generic agents, Claude Code, Cursor, OpenCode, and Antigravity
- examples for solo, assisted, multi-agent, and blocked verification

### Excluded from v0.1 default

These may be roadmap or optional later work, but should not be part of the default user path:

- full product intake lifecycle;
- story packet lifecycle;
- test matrix lifecycle;
- decision records;
- deep audit reports;
- MCP server;
- PGV runtime scoring;
- runtime consistency audit;
- automatic adapter installation;
- complex GitHub Actions;
- database-backed traces;
- LLM semantic verifier by default.

---

## 6. Repository Structure

Implement this structure.

```txt
x-harness/
  README.md
  AGENTS.md
  X_HARNESS.md
  package.json
  tsconfig.base.json
  LICENSE
  CHANGELOG.md
  CONTRIBUTING.md
  SECURITY.md

  docs/
    QUICKSTART.md
    PRINCIPLES.md
    MODES.md
    RUNTIME_CONTRACT.md
    VERIFY_GATE.md
    ADMISSION_POLICY.md
    PGV_ADVISORY.md
    DENOMINATOR_POLICY.md
    INTEGRATION.md
    ADAPTERS.md
    ROADMAP.md
    FAQ.md

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
          fs.ts
          config.ts
        validators/
          completionCard.ts
          subagentReturn.ts
          verifyEvent.ts
          pgvAdvice.ts

  adapters/
    generic/
      AGENTS.md

    claude-code/
      CLAUDE.md
      skills/
        x-harness-verify/
          SKILL.md
      agents/
        implementation-worker.md
        admission-verifier.md

    cursor/
      rules/
        x-harness.mdc

    opencode/
      README.md
      verify-agent.md

    antigravity/
      README.md
      rules/
        x-harness.md
      workflows/
        x-harness-implementation.md
        x-harness-verify.md

  examples/
    00-minimal/
    01-solo-agent/
    02-assisted-agent/
    03-multi-agent/
    04-blocked-verification/

  legacy/
    python/
      README.md
```

Python must not exist in the core tooling path. If Python appears, it must be under `legacy/python/` or `tools/experimental/`, with clear non-canonical status.

---

## 7. Core Documentation Requirements

### 7.1 README.md

The README must be concise and user-facing.

It must explain:

- what x-harness is;
- what it is not;
- the core rule;
- the minimal install/init flow;
- how to use it with one agent;
- how to use it with multiple agents;
- how to run verification;
- how adapters work;
- why the design is lightweight.

Required opening:

```md
# x-harness

x-harness is a lightweight verify-gated harness for AI-agent workflows.

It adds one rule to agentic development:

> Completion is admitted, not merely claimed.

Agents can claim work is fixed. x-harness decides whether that claim is allowed to count as done.
```

README must emphasize:

```txt
No daemon.
No database.
No MCP required.
No mandatory multi-agent setup.
No mandatory deep workflow.
```

### 7.2 AGENTS.md

`AGENTS.md` must be short. It is a map, not a manual.

Maximum recommended size: 100–150 lines.

It must include the agent contract, requiring agents to use the smallest suitable tier, create/update `completion-card.yaml`, run `npx x-harness verify`, and only report accepted completion when the verify result is `outcome: success` and `acceptance_status: accepted`.

### 7.3 X_HARNESS.md

`X_HARNESS.md` is the project overview.

It must explain:

- the default workflow;
- the three operating modes;
- the minimal artifact set;
- the distinction between claimed completion and accepted completion.

Required workflow:

```txt
Work -> Claim -> Verify -> Accepted or withheld
```

---

## 8. Tier Templates

`x-harness` must support exactly three canonical tier labels:

```txt
light
standard
deep
```

The runtime must not use:

```txt
small
medium
large
```

### 8.1 Light Tier

Use for narrow, low-risk tasks.

Required return shape:

```yaml
result:
  summary: <one-line outcome>
  fix_status: <fixed|not_fixed|partial>
  key_findings: []

evidence:
  files_changed: []
  commands_ran: []
  key_outputs: []

verification:
  status: <passed|failed|skipped|blocked>

confidence: <LOW|MED|HIGH>

handoff:
  next_action: <next step> (owner: <agent|user>)
```

### 8.2 Standard Tier

Use for bounded implementation, review, or synthesis.

Required additions over light:

```yaml
result:
  recommendations: []
  unsupported_or_unclear: []

evidence:
  sources_consulted: []

verification:
  checks:
    - name: <check>
      status: <passed|failed|skipped|blocked>
      note: <note>
```

### 8.3 Deep Tier

Use only when the cost of being wrong is high.

Required additions:

```yaml
execution_controls:
  mode: <read_only|limited_edit|full_edit>
  max_files_changed: <N|n/a>
  stop_conditions: []
  failure_fallback: <what to do if blocked>

rollback_policy:
  class: <none|soft|code_revert|state_restore>
  trigger: <when>
  owner: <agent|user>
  validation: <check>
```

Deep must not be the default.

---

## 9. Completion Card

`completion-card.yaml` is the main v0.1 artifact.

Template:

```yaml
id: CC-001
task_id: TASK-001
tier: standard

owner: implementation-worker
accountable: user

claim:
  fix_status: fixed
  summary: "Implemented requested change."

evidence:
  files_changed:
    - src/example.ts
  commands_ran:
    - command: npm test
      status: passed
  key_outputs:
    - "All tests passed."

verification:
  status: passed
  checks:
    - name: evidence_present
      status: passed
    - name: owner_present
      status: passed
    - name: blocked_state
      status: passed

admission:
  outcome: pending
  acceptance_status: withheld

handoff:
  next_action: "Run npx x-harness verify."
  owner: user

pgv_advice: null
```

After success:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

If evidence is missing:

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: evidence_floor_met
  reason: "No validation evidence attached."

handoff:
  next_action: "Attach validation output and rerun verification."
  owner: implementation-worker
```

---

## 10. Admission Policy

`policies/admission.yaml` must define the admission rules.

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
  - evidence_floor_met
  - no_unresolved_blocker
  - no_active_recovery
  - verifier_read_only

reject_success_if:
  fix_status:
    - partial
    - not_fixed

  verification_status:
    - failed
    - skipped
    - blocked

  evidence_quality:
    - missing
    - weak

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

The CLI must enforce this distinction:

```txt
fix_status=fixed + verification.status=passed => ready for admission
verify outcome=success + acceptance_status=accepted => accepted completion
```

---

## 11. Schemas

### 11.1 completion-card.schema.json

Must validate:

- `id`
- `task_id`
- `tier`
- `owner`
- `accountable`
- `claim.fix_status`
- `claim.summary`
- `evidence.files_changed`
- `evidence.commands_ran`
- `verification.status`
- `admission.outcome`
- `admission.acceptance_status`
- `handoff.next_action`
- `handoff.owner`

Enums:

```txt
tier: light | standard | deep
fix_status: fixed | not_fixed | partial
verification.status: passed | failed | skipped | blocked
admission.outcome: pending | success | failed | blocked | skipped | timeout | error
acceptance_status: accepted | withheld
```

### 11.2 subagent-return.schema.json

Must validate the common return groups:

```yaml
result:
  summary:
  fix_status:
  key_findings:

evidence:
  files_changed:
  commands_ran:
  key_outputs:

verification:
  status:

confidence:

handoff:
  next_action:
```

### 11.3 verify-event.schema.json

Define for future compatibility even if v0.1 does not implement full trace.

```json
{
  "event_id": "VE-001",
  "event_type": "verify_completed",
  "task_id": "TASK-001",
  "tier": "standard",
  "verifier": "x-harness",
  "verifier_mode": "read_only",
  "outcome": "blocked",
  "acceptance_status": "withheld",
  "blocking_predicate": "evidence_floor_met",
  "blocked_reason_class": "missing_validation_evidence",
  "next_owner": "implementation-worker",
  "next_action": "Attach validation output and rerun verify.",
  "created_at": "2026-01-01T00:00:00.000Z"
}
```

### 11.4 pgv-advice.schema.json

PGV must be advisory-only.

```yaml
risk_score: 0.0-1.0
claim_allowed: yes|no
needs_escalation: none|oracle|council
top_violation: string|null
next_control_action: string
verify_outcome_pred: passed|failed|blocked|uncertain
```

---

## 12. TypeScript CLI Requirements

The canonical tooling is TypeScript.

No Python should exist in primary tooling.

### 12.1 Root package.json

```json
{
  "name": "x-harness-repo",
  "private": true,
  "type": "module",
  "workspaces": [
    "packages/*"
  ],
  "scripts": {
    "build": "npm -w packages/cli run build",
    "dev": "npm -w packages/cli run dev",
    "test": "npm -w packages/cli run test",
    "typecheck": "npm -w packages/cli run typecheck",
    "lint": "npm -w packages/cli run lint",
    "verify": "npm run typecheck && npm run test"
  }
}
```

### 12.2 packages/cli/package.json

```json
{
  "name": "x-harness",
  "version": "0.1.0",
  "description": "A lightweight verify-gated harness for AI-agent workflows.",
  "type": "module",
  "bin": {
    "x-harness": "./dist/index.js"
  },
  "scripts": {
    "build": "tsc",
    "dev": "tsx src/index.ts",
    "test": "vitest run",
    "typecheck": "tsc --noEmit",
    "lint": "eslint ."
  },
  "dependencies": {
    "ajv": "^8.17.1",
    "commander": "^12.1.0",
    "fs-extra": "^11.2.0",
    "yaml": "^2.5.1",
    "zod": "^3.23.8"
  },
  "devDependencies": {
    "@types/fs-extra": "^11.0.4",
    "@types/node": "^22.0.0",
    "tsx": "^4.19.0",
    "typescript": "^5.5.0",
    "vitest": "^2.0.0",
    "eslint": "^9.0.0"
  }
}
```

### 12.3 Required commands

v0.1 commands:

```bash
npx x-harness init --minimal
npx x-harness verify
npx x-harness doctor
npx x-harness report
```

### 12.4 init --minimal behavior

Must create:

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

Safety rules:

- do not silently overwrite;
- support `--dry-run`;
- support `--merge`;
- support `--force`;
- if conflicts exist without merge/force, stop and list conflicts.

### 12.5 verify behavior

`npx x-harness verify` is read-only.

It must check:

- `completion-card.yaml` exists;
- tier is `light`, `standard`, or `deep`;
- runtime tier does not use `small`, `medium`, or `large`;
- owner and accountable exist;
- `claim.fix_status` is valid;
- `verification.status` is valid;
- evidence exists;
- PGV, if present, is advisory-only;
- `fix_status=fixed` does not automatically mean accepted;
- success maps to accepted;
- failed/blocked/skipped/timeout/error map to withheld;
- blocked has next action and owner.

It must output a clear result:

```txt
success | failed | blocked | skipped | timeout | error
```

### 12.6 doctor behavior

`npx x-harness doctor` must check:

- required files;
- schema validity;
- policy validity;
- adapter docs exist;
- no Python in core tooling;
- no PGV-as-authority wording;
- `AGENTS.md` is not overly large;
- docs links exist;
- canonical tier labels are used.

### 12.7 report behavior

`npx x-harness report` must generate Markdown:

```md
# x-harness Report

## Installed mode

## Templates

## Completion card

## Verification summary

## Blocked items

## Denominator warning

Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.
```

---

## 13. Adapter Requirements

Adapters must be lightweight docs/rules/skills. They must not require heavy runtime or auto-install in v0.1.

### 13.1 Generic Adapter

`adapters/generic/AGENTS.md` must contain a generic version of the agent contract.

It must instruct any agent to:

- use `light`, `standard`, or `deep`;
- create `completion-card.yaml`;
- run `npx x-harness verify`;
- only accept completion on success;
- treat PGV as advisory-only;
- keep verification read-only.

### 13.2 Claude Code Adapter

Files:

```txt
adapters/claude-code/
  CLAUDE.md
  skills/
    x-harness-verify/
      SKILL.md
  agents/
    implementation-worker.md
    admission-verifier.md
```

`CLAUDE.md` must say:

```md
# x-harness

This repo uses x-harness.

Before final response:

1. Create or update completion-card.yaml.
2. Run npx x-harness verify.
3. Say accepted only if outcome is success.
4. Otherwise say withheld and include next action.
```

`SKILL.md` must say:

```md
---
description: Run x-harness read-only verification
allowed-tools: Read, Grep, Glob, Bash
---

You are running x-harness verification.

Rules:
- Do not edit files.
- Inspect completion-card.yaml.
- Run npx x-harness verify.
- Return success only if admission policy passes.
- Treat PGV as advisory-only.
```

`implementation-worker.md` must state that the worker may edit source files and must create `completion-card.yaml`, but may not mark completion accepted.

`admission-verifier.md` must state that the verifier must not edit files and that only success maps to accepted.

### 13.3 Antigravity Adapter

Files:

```txt
adapters/antigravity/
  README.md
  rules/
    x-harness.md
  workflows/
    x-harness-implementation.md
    x-harness-verify.md
```

Rule content:

```md
# x-harness Rule

This workspace uses x-harness.

Before reporting a task as done:

1. Create or update completion-card.yaml.
2. Record claim, evidence, files changed, commands run, and verification status.
3. Run npx x-harness verify.
4. Only report completion if verify outcome is success.
5. If verify returns blocked, failed, skipped, timeout, or error, report withheld completion and include next action.

Do not edit files during verify mode.
Do not treat PGV advice as admission authority.
```

Implementation workflow:

```md
# x-harness implementation workflow

1. Classify tier: light, standard, deep.
2. Use matching handoff template.
3. Implement change.
4. Create completion-card.yaml.
5. Run project checks if available.
6. Run npx x-harness verify.
7. Final response:
   - success => accepted
   - otherwise => withheld + blocker + next action
```

Verify workflow:

```md
# x-harness read-only verify workflow

Rules:
- Read-only.
- Do not edit source files.
- Inspect completion-card.yaml.
- Inspect changed files and evidence.
- Run npx x-harness verify.
- Return success, failed, blocked, skipped, timeout, or error.

Only success means accepted.
All other outcomes mean withheld.
```

### 13.4 Cursor Adapter

File:

```txt
adapters/cursor/rules/x-harness.mdc
```

Content:

```md
---
alwaysApply: true
---

This repo uses x-harness.

Use the smallest tier that preserves correctness:
- light for narrow work
- standard for normal implementation or bounded synthesis
- deep for high-risk work

Do not mark work complete solely from implementation output.
fix_status: fixed means candidate completion, not accepted completion.
Completion requires npx x-harness verify success.

Verification is read-only.
PGV is advisory-only.
```

### 13.5 OpenCode Adapter

Files:

```txt
adapters/opencode/
  README.md
  verify-agent.md
```

`verify-agent.md`:

```md
# x-harness verify agent

You are a read-only verifier.

Inspect:
- completion-card.yaml
- changed files
- test outputs
- evidence

Run:

npx x-harness verify

Do not edit source files.

Only outcome success maps to accepted.
Everything else maps to withheld.
```

---

## 14. User Workflows

### 14.1 Generic usage

```bash
cd my-repo
npx x-harness init --minimal
```

Then ask the agent:

```txt
Implement this change using x-harness.
Create completion-card.yaml and run npx x-harness verify before final response.
Only say done if verification succeeds.
```

### 14.2 Claude Code usage

```bash
npx x-harness init --minimal
```

Copy or install adapter docs from:

```txt
adapters/claude-code/
```

Prompt:

```txt
Implement login validation using x-harness solo-agent mode.
Do not say done until completion-card.yaml exists and npx x-harness verify succeeds.
```

Multi-agent prompt:

```txt
Use implementation-worker to implement the change.
Then use admission-verifier to run x-harness read-only verification.
Only report accepted if verify returns success.
```

### 14.3 Antigravity usage

```bash
npx x-harness init --minimal
```

Use:

```txt
adapters/antigravity/rules/x-harness.md
adapters/antigravity/workflows/x-harness-implementation.md
adapters/antigravity/workflows/x-harness-verify.md
```

Prompt:

```txt
Add password validation to the signup form.
Use x-harness workflow.
Do not mark this complete until x-harness verify passes.
```

### 14.4 Cursor usage

Install rule:

```txt
adapters/cursor/rules/x-harness.mdc
```

Prompt:

```txt
Apply this change using x-harness. Create completion-card.yaml and run npx x-harness verify before final response.
```

### 14.5 OpenCode usage

Use:

```txt
adapters/opencode/verify-agent.md
```

Prompt:

```txt
Use x-harness. Implement the task, create completion-card.yaml, then invoke the read-only verify agent.
```

---

## 15. Token and Complexity Controls

The implementation must remain light.

Rules:

1. `AGENTS.md` must be a short map, not a long manual.
2. Light tier is the default.
3. Deep tier is opt-in.
4. Completion card is the main artifact in v0.1.
5. No mandatory product intake, story packet, or test matrix in v0.1.
6. No database.
7. No daemon.
8. No mandatory MCP.
9. No LLM semantic verifier by default.
10. CLI is deterministic-first.
11. Adapters are docs/rules/skills, not heavy runtime wrappers.
12. Do not force multi-agent setup.
13. Do not stuff all instructions into every prompt.
14. Use progressive disclosure: `AGENTS.md` points to docs and templates.

---

## 16. Denominator and Claims Policy

`x-harness` must not overclaim.

Reports and docs must include:

```txt
Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.
```

Forbidden claims:

- “x-harness guarantees correctness”
- “x-harness makes agents reliable”
- “x-harness proves production safety”
- “verify success equals task success”
- “PGV agreement equals admission correctness”

Allowed claims:

- “x-harness reduces premature completion claims”
- “x-harness separates claimed completion from accepted completion”
- “x-harness provides a lightweight read-only verify gate”
- “x-harness structures evidence for agent handoffs”
- “x-harness supports solo, assisted, and multi-agent workflows”

---

## 17. Roadmap

### v0.1 — Lightweight Core

Implement:

- `AGENTS.md`
- `X_HARNESS.md`
- light/standard/deep templates
- completion card
- admission policy
- TypeScript CLI: `init`, `verify`, `doctor`, `report`
- generic/Claude/Cursor/OpenCode/Antigravity adapter docs
- examples: solo, assisted, multi-agent, blocked

### v0.2 — Optional Standard Mode

Add:

- optional feature intake;
- optional story packet;
- optional test matrix;
- trace JSONL;
- verify event schema enforcement.

### v0.3 — Adapter Expansion

Add:

- richer Claude Code subagents/skills;
- Antigravity workflows;
- Cursor rule pack;
- OpenCode verify profile examples.

### v0.4 — Deep Mode

Add:

- recovery packet;
- rollback policy;
- audit report;
- advanced blocked instrumentation.

### v1.0 — Stable Contract

Stabilize:

- CLI;
- schemas;
- templates;
- adapter docs;
- admission semantics;
- examples.

---

## 18. Implementation Acceptance Criteria

AI agents implementing this repo must satisfy all criteria below.

### Repository criteria

- [ ] Repository is named `x-harness`.
- [ ] README describes x-harness as a lightweight verify-gated harness.
- [ ] README does not describe x-harness as a full autonomous agent framework.
- [ ] `AGENTS.md` is short and points to deeper docs.
- [ ] `X_HARNESS.md` explains the workflow and supported modes.
- [ ] File structure matches the v0.1 scope.
- [ ] No Python exists in core tooling.
- [ ] Python, if present, is only under `legacy/python/` or `tools/experimental/`.

### Workflow criteria

- [ ] Solo Agent Mode is documented.
- [ ] Assisted Agent Mode is documented.
- [ ] Multi-Agent Mode is documented.
- [ ] User workflow examples exist for Claude Code.
- [ ] User workflow examples exist for Antigravity.
- [ ] User workflow examples exist for Cursor.
- [ ] User workflow examples exist for OpenCode.
- [ ] Generic usage exists for any `AGENTS.md`-compatible agent.

### Template criteria

- [ ] `SUBAGENT_TASK_light.md` exists.
- [ ] `SUBAGENT_TASK_standard.md` exists.
- [ ] `SUBAGENT_TASK_deep.md` exists.
- [ ] `COMPLETION_CARD.md` exists.
- [ ] Deep tier is not positioned as default.
- [ ] Runtime labels are only `light`, `standard`, `deep`.

### Admission criteria

- [ ] `completion-card.yaml` is the main v0.1 artifact.
- [ ] `policies/admission.yaml` exists.
- [ ] `fix_status=fixed` is candidate completion only.
- [ ] `verification.status=passed` is candidate completion only.
- [ ] Only verify outcome `success` maps to `accepted`.
- [ ] `failed`, `blocked`, `skipped`, `timeout`, and `error` map to `withheld`.
- [ ] Blocked outcomes require next action and owner.
- [ ] Verification is read-only.
- [ ] PGV is advisory-only.

### CLI criteria

- [ ] TypeScript CLI exists under `packages/cli`.
- [ ] Root `package.json` uses npm workspaces.
- [ ] CLI package is named `x-harness`.
- [ ] `npx x-harness init --minimal` is documented.
- [ ] `npx x-harness verify` is documented.
- [ ] `npx x-harness doctor` is documented.
- [ ] `npx x-harness report` is documented.
- [ ] CLI is deterministic-first.
- [ ] CLI does not require a daemon, database, or MCP.

### Adapter criteria

- [ ] Generic adapter exists.
- [ ] Claude Code adapter exists.
- [ ] Claude Code verify skill exists.
- [ ] Claude Code implementation-worker and admission-verifier examples exist.
- [ ] Antigravity rule exists.
- [ ] Antigravity implementation and verify workflows exist.
- [ ] Cursor rule exists.
- [ ] OpenCode verify agent exists.
- [ ] No adapter requires Python as primary path.

### Complexity criteria

- [ ] `AGENTS.md` is not a large manual.
- [ ] Light tier is default.
- [ ] Deep tier is opt-in.
- [ ] Product intake/story/test matrix are not required in v0.1.
- [ ] No required LLM semantic verifier.
- [ ] No required MCP server.
- [ ] No required database.
- [ ] No long-running service.

### Reporting criteria

- [ ] `npx x-harness report` includes denominator warning.
- [ ] Docs do not claim correctness guarantees.
- [ ] Docs do not conflate verify success with task-level success.
- [ ] Docs do not promote PGV to admission authority.

---

## 19. Recommended Implementation Order

AI agents should implement in this order.

### Step 1 — Repo skeleton

Create root package files, docs, templates, schemas, policies, adapters, examples, and `packages/cli`.

### Step 2 — Core docs

Create `README.md`, `AGENTS.md`, `X_HARNESS.md`, `docs/QUICKSTART.md`, `docs/MODES.md`, `docs/VERIFY_GATE.md`, `docs/ADMISSION_POLICY.md`, and `docs/PGV_ADVISORY.md`.

### Step 3 — Templates and policy

Create the light, standard, and deep templates, completion card, verify report, and admission policy.

### Step 4 — Schemas

Create schemas for completion cards, sub-agent returns, verify events, and PGV advice.

### Step 5 — CLI scaffold

Create a commander-based CLI with `init`, `verify`, `doctor`, and `report`.

### Step 6 — Adapters

Create generic, Claude Code, Antigravity, Cursor, and OpenCode adapters.

### Step 7 — Examples

Create examples for minimal usage, solo agent, assisted agent, multi-agent, and blocked verification.

### Step 8 — Verification

Run:

```bash
npm install
npm run typecheck
npm run build
npm test
npx x-harness doctor
npx x-harness verify
```

If commands cannot run, document why and return `verification.status: blocked` or `skipped`.

---

## 20. Final Implementation Rule for Agents

When implementing this repo, do not optimize for maximum governance. Optimize for adoption.

The repo should feel like this:

```txt
Add x-harness to my repo.
Tell my agent to use it.
Agent creates completion-card.yaml.
Run npx x-harness verify.
Only then say done.
```

The final product must preserve the core values:

```txt
worker claims
verifier admits
ambiguity is withheld
PGV is advisory
evidence is structured
light stays light
deep is optional
the repo remains easy to adopt
```
