# ⚡ x-harness

[![Verify](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml/badge.svg)](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Node.js Version](https://img.shields.io/badge/Node.js-%3E%3D20-blue.svg)](package.json)
[![Language: TypeScript](https://img.shields.io/badge/Language-TypeScript-blue.svg)](tsconfig.base.json)

`x-harness` is a lightweight, offline-first, verify-gated harness for orchestrating and verifying AI-agent workflows. It enforces structured sub-agent handoffs, read-only validation gates, and fail-closed completion rules to help ensure completion claims are admitted only when they satisfy repository verification policy.

> [!NOTE]
> **Local Development Only**: `x-harness` is not yet published to npm. To use the CLI locally, clone the repository, run `npm install`, and `npm run build`. Then invoke the CLI with `node packages/cli/dist/index.js <command>`.

---

## 🎯 Core Philosophy

> **Completion is admitted, not merely claimed.**

Traditional agent harnesses focus primarily on generation (what code or text the agent should write). `x-harness` shifts the focus to verification and governance. It provides orchestrators and developers with a deterministic, rule-based gate to verify agent work products against strict repository policies.

In `x-harness`, an agent stating `fix_status: fixed` or running a passing test is simply producing a **completion candidate**. Actual completion is only **accepted** when the read-only verification gate runs its policy and admits the work.

---

## 🔰 Beginner's Fast-Track Guide (5-Minute Tour)

If you are new to `x-harness`, follow this step-by-step walkthrough to get up and running in minutes!

### Step 1: Install and Compile

Clone the repository and build the TypeScript CLI application locally:

```bash
# Install dependencies
npm install

# Compile the TypeScript CLI source code
npm run build
```

### Step 2: Run Your First Verification

`x-harness` comes with pre-packaged reference scenarios called "Golden Examples". Let's run a verification against a successful task claim:

```bash
node packages/cli/dist/index.js verify --card examples/golden/success-light/completion-card.yaml
```

> **Expected Output:**
>
> ```yaml
> outcome: success
> acceptance_status: accepted
> ```
>
> _Note: The CLI returns exit code `0` because the card meets all the light-tier policy requirements._

Now, let's run verification on a card that is missing mandatory evidence scopes:

```bash
node packages/cli/dist/index.js verify --card examples/golden/blocked-missing-evidence/completion-card.yaml
```

> **Expected Output:**
>
> ```yaml
> outcome: failed
> acceptance_status: withheld
> ```
>
> _Note: The CLI returns exit code `1` (fail-closed) because the card failed the required evidence floor rules._

### Step 3: Initialize a New Workspace

To start using `x-harness` in a separate development project, run the `init` command in the root of your project:

```bash
# Set up a Minimal workspace (default; installs agents contract, templates, and policies)
node packages/cli/dist/index.js init --minimal
```

If the target directory already contains conflicting harness files, `init` stops with a blocked summary and exits non-zero. Re-run with `--force` only when you intentionally want to overwrite those files, or use `--merge` if/when you want non-destructive merge behavior.

### Step 4: Dispatch a Task Handoff

When assigning a task to an agent, generate a structured handoff prompt. For example, to dispatch a normal task:

```bash
node packages/cli/dist/index.js handoff standard --title "Fix Checkout Page Button Alignment"
```

This generates a markdown file matching the `standard` tier containing explicit file sets, required evidence checklists, and rollback definitions.

### Step 5: Validate Workspace Health

Run the diagnostics command at any time to verify that all schemas, policies, templates, and links are healthy:

```bash
node packages/cli/dist/index.js doctor
```

---

## 🧱 Key Features & Capabilities

- 🚦 **Read-Only Verification Gate**: Enforces that verification agents or tools inspect evidence without mutating the codebase to fix issues during validation.
- 📦 **Tiered Handoff Templates**: Clean, standard markdown templates for task dispatch across three canonical tiers: `light`, `standard`, and `deep`.
- 🔌 **Platform-Agnostic Adapters**: Native configurations and instructions for popular agent environments (Generic, Claude Code, Cursor, OpenCode, Antigravity).
- 🧩 **Schema-Validated Completion Cards**: Standardized YAML/JSON metadata (Claim, Evidence, Verification, Handoff) for recording and auditing task outcomes.
- 🔄 **Fail-Closed & Recovery Routing**: Any verification result other than a total pass is withheld (`withheld`). Under failure, the harness yields structured recovery actions (e.g. `evidence_missing` routes back to the worker, `approval_missing` routes to the user).
- 📊 **Deterministic Local Metrics**: Generates local, offline reports analyzing verification strength, state consistency, recovery ability, replayability, and runtime cost.
- 🧠 **Advisory-Only Pre-Gate Validation (PGV)**: Guides agents through safe practices without granting them final admission authority.

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

Task delegation in `x-harness` uses **only** the following three canonical tiers. The use of non-canonical labels in active runtime handoffs is forbidden.

| Tier           | Complexity & Scope                                                                     | Evidence Floor                                                                                            | Human Approval |
| :------------- | :------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------- | :------------- |
| **`light`**    | Narrow, low-ceremony tasks (1-3 files changed; read-only or nearly read-only).         | Files changed + command evidence OR manual rationale.                                                     | Optional       |
| **`standard`** | Normal multi-step work (bounded synthesis, multiple sources).                          | Files changed + command evidence. Evidence scope recommended.                                             | Optional       |
| **`deep`**     | High-stakes operations (multiple dependencies, architectural risks, migration impact). | Files changed + command evidence + evidence scope + untested regions + remaining risks + rollback policy. | Required       |

---

## 🛠️ Command-Line Interface (CLI)

`x-harness` features a TypeScript-first CLI tool to initialize templates, run verify gates, audit project health, and compute performance metrics.

### Core Commands

| Command        | Usage                                                                                                | Description                                                                                |
| :------------- | :--------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------- |
| **`init`**     | `node packages/cli/dist/index.js init [target_dir] [--minimal / --standard / --full]`                | Installs the core harness assets, schemas, policies, and adapters. Default is `--minimal`. |
| **`handoff`**  | `node packages/cli/dist/index.js handoff <light / standard / deep> [--title <text>] [--task <text>]` | Generates a clean markdown handoff task prompt structure.                                  |
| **`add`**      | `node packages/cli/dist/index.js add <claim / evidence / completion-card> [key=value]`               | Adds a metadata helper file for compatibility modes.                                       |
| **`verify`**   | `node packages/cli/dist/index.js verify [--card <path>] [--json] [--verbose]`                        | Executes the read-only verification policy against a completion card.                      |
| **`doctor`**   | `node packages/cli/dist/index.js doctor [--root <path>]`                                             | Checks critical file presence, schemas compilation, policies, and wording.                 |
| **`report`**   | `node packages/cli/dist/index.js report [--metrics] [--card <path>] [--json]`                        | Summarizes verification events or calculates local card metrics.                           |
| **`trace`**    | `node packages/cli/dist/index.js trace add [--outcome <status>] [--task-id <id>]`                    | Manually appends verification completion events to the trace log.                          |
| **`clean`**    | `node packages/cli/dist/index.js clean [--tmp / --reset-card / --archive-success] [--force]`         | Defaults to a dry run; add `--force` to mutate tmp artifacts, reset a completion card, or archive accepted-card snapshots. |
| **`context`**  | `node packages/cli/dist/index.js context [--verbose / --json / --refresh] [--root <path>]`            | Shows canonical context and refreshes the AGENTS.md managed block.                         |
| **`examples`** | `node packages/cli/dist/index.js examples`                                                           | Lists or copies built-in test-cases showing successful and blocked runs.                   |
| **`recovery`** | `node packages/cli/dist/index.js recovery suggest [--errors <text>] [--outcome <status>]`            | Generates structured recovery playbook suggestions based on failure predicates.             |
| **`packet`**   | `node packages/cli/dist/index.js packet create --card <path>` or `packet verify-chain --task-id <id>` | Creates immutable claim packets from completion cards and verifies packet chain integrity.   |

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

Running `node packages/cli/dist/index.js verify --trace` logs a JSONL event detailing the verification runtime parameters. These events can be aggregated using `node packages/cli/dist/index.js report` to track task success rates and blocked items over time.

### Deterministic Offline Metrics

`node packages/cli/dist/index.js report --metrics` calculates metrics under five categories:

1. **Verification Strength**: Count of artifacts, kind of oracles (unit tests, typecheck), and count of untested/remaining risks.
2. **State Consistency**: Checks if owners are declared and if card status maps to admission outcome.
3. **Recovery Ability**: Checks if withheld cards contain actionable next owners and paths.
4. **Replayability**: Computes card and policy SHA-256 hashes to guarantee reproducibility.
5. **Cost**: Tracks runtime class (low/moderate/high) and verify execution time.

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
│       │   └── validators/ # Ajv schema validation
│       └── tests/          # CLI Unit and Integration tests
│   templates/              # Markdown templates for tasks & completion cards
│   schemas/                # JSON schemas for validating claims & cards
│   policies/               # admission and recovery YAML policies
│   docs/                   # Full reference documentation files
│   adapters/               # Platform-specific instructions and rules
│   examples/               # Reference scenarios & golden test cases
```

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [`docs/HANDBOOK.md`](docs/HANDBOOK.md) | Technical handbook: philosophy, tiers, CLI reference, evidence floors |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Architectural design, layer model, and validation flow |
| [`docs/PRINCIPLES.md`](docs/PRINCIPLES.md) | Core design principles and philosophy |
| [`docs/SCHEMAS.md`](docs/SCHEMAS.md) | JSON schema inventory and validation guide |
| [`docs/ADMISSION_POLICY.md`](docs/ADMISSION_POLICY.md) | Fail-closed admission rules and evidence floors |
| [`docs/VERIFY_GATE.md`](docs/VERIFY_GATE.md) | Read-only verification gate mechanics |
| [`docs/RUNTIME_CONTRACT.md`](docs/RUNTIME_CONTRACT.md) | Runtime contract between components |
| [`docs/PACKETS.md`](docs/PACKETS.md) | Packet design spec and claim-only implementation guide |
| [`docs/RECOVERY.md`](docs/RECOVERY.md) | Recovery routing and playbook generation |
| [`docs/ADAPTERS.md`](docs/ADAPTERS.md) | Platform adapter guide (Generic, Claude Code, Cursor, OpenCode, Antigravity) |
| [`docs/CI.md`](docs/CI.md) | CI integration guide and local-build composite action |
| [`docs/REPORT_FORMATS.md`](docs/REPORT_FORMATS.md) | Report output formats: Markdown, JSON, HTML |
| [`docs/METRICS.md`](docs/METRICS.md) | Metrics computation and interpretation |
| [`docs/PGV_ADVISORY.md`](docs/PGV_ADVISORY.md) | Pre-gate validation advisory policy |
| [`docs/CONTEXT_POLICY.md`](docs/CONTEXT_POLICY.md) | Context management and freshness policy |
| [`docs/DENOMINATOR_POLICY.md`](docs/DENOMINATOR_POLICY.md) | Success denominator interpretation rules |
| [`docs/MODES.md`](docs/MODES.md) | Operational modes and configuration |
| [`docs/COMPARISON.md`](docs/COMPARISON.md) | Comparison with other agent frameworks |
| [`docs/FAQ.md`](docs/FAQ.md) | Frequently asked questions |
| [`docs/QUICKSTART.md`](docs/QUICKSTART.md) | Quick start guide |
| [`docs/CLEANUP.md`](docs/CLEANUP.md) | Cleanup and maintenance operations |
| [`docs/TEMPLATE_AUTHORING.md`](docs/TEMPLATE_AUTHORING.md) | Template authoring guide |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Project roadmap and future plans |

---

## 🤝 Project Health & Contribution

- **License**: MIT (`LICENSE`)
- **Contribution Guidelines**: See `CONTRIBUTING.md` and `templates/HARNESS_CHANGE_CONTRACT.md` before making harness-sensitive changes.
- **Project Health Checks**: Execute `node packages/cli/dist/index.js doctor` regularly to ensure files, schemas, and policies are valid and aligned.
