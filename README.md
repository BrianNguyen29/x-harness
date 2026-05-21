# ⚡ x-harness

[![Verify](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml/badge.svg)](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Node.js Version](https://img.shields.io/badge/Node.js-%3E%3D20-blue.svg)](package.json)
[![Language: TypeScript](https://img.shields.io/badge/Language-TypeScript-blue.svg)](tsconfig.base.json)

`x-harness` is a lightweight, offline-first, verify-gated harness for orchestrating and verifying AI-agent workflows. It enforces structured sub-agent handoffs, read-only validation gates, and fail-closed completion rules to ensure that tasks are truly done before they are admitted.

---

## 🎯 Core Philosophy

> **Completion is admitted, not merely claimed.**

Traditional agent harnesses focus primarily on generation (what code/text to write). `x-harness` shifts the focus to verification and governance. It provides orchestrators and developers with a deterministic, rule-based gate to verify agent work products against strict repository policies.

In `x-harness`, an agent stating `fix_status: fixed` or running a passing test is simply producing a **completion candidate**. Actual completion is only **accepted** when the read-only verification gate runs its policy and admits the work.

---

## 🧱 Key Features & Capabilities

- 🚦 **Read-Only Verification Gate**: Enforces that verification agents/tools inspect evidence without mutating the codebase to fix issues during validation.
- 📦 **Tiered Handoff Templates**: Clean, standard markdown templates for task dispatch across three canonical tiers: `light`, `standard`, and `deep`.
- 🔌 **Platform-Agnostic Adapters**: Native configurations and instructions for popular agent environments (Generic, Claude Code, Cursor, OpenCode, Antigravity).
- 🧩 **Schema-Validated Completion Cards**: Standardized YAML/JSON metadata (Claim, Evidence, Verification, Handoff) for recording and auditing task outcomes.
- 🔄 **Fail-Closed & Recovery Routing**: Any verification result other than a total pass is withheld (`withheld`). Under failure, the harness yields structured recovery actions (e.g. `evidence_missing` routes back to the worker, `approval_missing` routes to the user).
- 📊 **Deterministic Local Metrics**: Generates local, offline reports analyzing verification strength, state consistency, recovery ability, replayability, and runtime cost.
- 🧠 **Advisory-Only Policy Governance (PGV)**: Guides agents through safe practices without granting them final admission authority.

---

## 📐 Architecture & Workflow

```text
  ┌──────────────┐
  │  Agent Task  │
  └──────┬───────┘
         │  1. Dispatch
         ▼
  ┌──────────────┐
  │ Handoff Tier ├───────────► [ light / standard / deep ]
  └──────┬───────┘
         │  2. Execute
         ▼
  ┌──────────────┐
  │ Agent Worker │◄─── (Performs implementation & testing)
  └──────┬───────┘
         │  3. Claim
         ▼
  ┌──────────────┐
  │  Completion  │◄─── (Outputs YAML Completion Card containing
  │     Card     │      Claims, Evidence, and Handoff data)
  └──────┬───────┘
         │  4. Verify
         ▼
  ┌──────────────┐
  │  Read-Only   │◄─── (Loads schemas & admission policies;
  │ Verify Gate  │      Validates card structure & evidence floor)
  └──────┬───────┘
         │  5. Decision
         ▼
  ┌──────────────┐
  │  Outcome     ├───► [ ACCEPTED / WITHHELD ]
  └──────────────┘
```

---

## 🎚️ Canonical Handoff Tiers

Task delegation in `x-harness` uses **only** the following three canonical tiers. The use of non-canonical labels (e.g. *small*, *medium*, *large*) in active runtime handoffs is forbidden.

| Tier | Complexity & Scope | Evidence Floor | Human Approval |
| :--- | :--- | :--- | :--- |
| **`light`** | Narrow, low-ceremony tasks (1-3 files changed; read-only or nearly read-only). | Files changed + command evidence OR manual rationale. | Optional |
| **`standard`** | Normal multi-step work (bounded synthesis, multiple sources). | Files changed + command evidence. Evidence scope recommended. | Optional |
| **`deep`** | High-stakes operations (multiple dependencies, architectural risks, migration impact). | Files changed + command evidence + evidence scope + untested regions + remaining risks + rollback policy. | Required |

---

## 🛠️ Command-Line Interface (CLI)

`x-harness` features a TypeScript-first CLI tool to initialize templates, run verify gates, audit project health, and compute performance metrics.

### Installation & Compilation

```bash
# Install dependencies
npm install

# Compile the TypeScript CLI source code
npm run build
```

### Core Commands

| Command | Usage | Description |
| :--- | :--- | :--- |
| **`init`** | `npx x-harness init [target_dir] [--minimal / --standard / --full]` | Installs the core harness assets, schemas, policies, and adapters. |
| **`handoff`** | `npx x-harness handoff <light / standard / deep> [--title <text>] [--task <text>]` | Generates a clean markdown handoff task prompt structure. |
| **`add`** | `npx x-harness add <claim / evidence / story / completion-card> [key=value]` | Adds a metadata helper file for compatibility modes. |
| **`verify`** | `npx x-harness verify [--card <path>] [--json] [--verbose]` | Executes the read-only verification policy against a completion card. |
| **`doctor`** | `npx x-harness doctor [--root <path>]` | Checks critical file presence, schemas compilation, policies, and wording. |
| **`report`** | `npx x-harness report [--metrics] [--card <path>] [--json]` | Summarizes verification events or calculates local card metrics. |
| **`trace`** | `npx x-harness trace add [--outcome <status>] [--task-id <id>]` | Manually appends verification completion events to the trace log. |
| **`clean`** | `npx x-harness clean [--all]` | Cleans up temporary validation artifacts, reports, and logs. |
| **`examples`** | `npx x-harness examples` | Lists or copies built-in test-cases showing successful and blocked runs. |

---

## 🔌 Multi-Platform Adapters

To integrate `x-harness` into your agent environment of choice, use the following adapter pathways located under the [adapters](adapters) directory:

- **Generic Markdown** ([adapters/generic](adapters/generic)): Simple, system-agnostic conventions utilizing `AGENTS.md` and standard completion templates.
- **Claude Code** ([adapters/claude-code](adapters/claude-code)): Integrates via `CLAUDE.md`, defining worker/verifier roles and equipping Claude Code with specialized local verification skills.
- **Cursor** ([adapters/cursor](adapters/cursor)): Leverages Cursor's rules system via `.cursor/rules/x-harness.mdc` to guide the IDE agent dynamically.
- **OpenCode** ([adapters/opencode](adapters/opencode)): Leverages the `verify-agent.md` setup for orchestrating and verifying agent work inside OpenCode.
- **Antigravity** ([adapters/antigravity](adapters/antigravity)): Connects with Antigravity systems through strict constraint policies and workflow specifications.

---

## 🚦 Verification Policies & Recovery

### Fail-Closed Decision Engine
The verification engine inspects the completion card (`completion-card.yaml`) and runs it against the `policies/admission.yaml` config.
- **Accepted**: Verification succeeds, schema is valid, evidence floor is met, and `fix_status` is `fixed`.
- **Withheld**: Any other outcome (failed, blocked, skipped, timeout, or error). The exit code is non-zero (`1`).

### Recovery Routing
If a task is withheld, the engine evaluates the failure predicate and suggests a next step:

```json
"recovery": {
  "predicate": "evidence_missing",
  "next_action": "Attach validation evidence or explain why unavailable.",
  "owner": "implementation-worker"
}
```

Common recovery paths include routing back to the `implementation-worker` for test repairs, or asking the `user` for manual approvals or scope clarifications.

---

## 📈 Tracing & Local Metrics

### Traces
Running `npx x-harness verify --trace` logs a JSONL event detailing the verification runtime parameters. These events can be aggregated using `npx x-harness report` to track task success rates and blocked items over time.

### Deterministic Offline Metrics
`npx x-harness report --metrics` calculates metrics under five categories:
1. **Verification Strength**: Count of artifacts, kind of oracles (unit tests, typecheck), and count of untested/remaining risks.
2. **State Consistency**: Checks if owners are declared and if card status maps to admission outcome.
3. **Recovery Ability**: Checks if withheld cards contain actionable next owners and paths.
4. **Replayability**: Computes card and policy SHA-256 hashes to guarantee reproducibility.
5. **Cost**: Tracks runtime class (low/medium/high) and verify execution time.

> [!WARNING]
> **Denominator Rule**: Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or a safety guarantee. A template lint pass is not runtime correctness.

---

## 📁 Repository Directory Structure

```text
├── packages/
│   └── cli/                # TypeScript CLI Tool Source Code
│       ├── src/
│       │   ├── commands/   # command-line sub-commands
│       │   ├── core/       # admission, metrics, and recovery engines
│       │   └── validators/ # Zod & Ajv schemas validation
│       └── tests/          # CLI Unit and Integration tests
├── templates/              # Markdown templates for tasks & completion cards
├── schemas/                # JSON schemas for validating claims & cards
├── policies/               # admission and recovery YAML policies
├── docs/                   # Full reference documentation files
├── adapters/               # Platform-specific instructions and rules
└── examples/               # Reference scenarios & golden test cases
```

---

## 🤝 Project Health & Contribution

- **License**: MIT (`LICENSE`)
- **Contribution Guidelines**: See `CONTRIBUTING.md` and `templates/HARNESS_CHANGE_CONTRACT.md` before making harness-sensitive changes.
- **Project Health Checks**: Execute `npx x-harness doctor` regularly to ensure files, schemas, and policies are valid and aligned.
