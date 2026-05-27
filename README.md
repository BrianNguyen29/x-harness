# ⚡ x-harness

[![Verify](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml/badge.svg)](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml)
[![CodeQL](https://github.com/BrianNguyen29/x-harness/actions/workflows/codeql.yml/badge.svg)](https://github.com/BrianNguyen29/x-harness/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/BrianNguyen29/x-harness/badge)](https://scorecard.dev/viewer/?uri=github.com/BrianNguyen29/x-harness)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Node.js Version](https://img.shields.io/badge/Node.js-%3E%3D20-blue.svg)](package.json)
[![Language: Go](https://img.shields.io/badge/Language-Go-00ADD8.svg)](go.mod)
[![Compatibility: TypeScript](https://img.shields.io/badge/Compatibility-TypeScript-blue.svg)](tsconfig.base.json)

`x-harness` is a lightweight, offline-first, verify-gated harness for orchestrating and verifying AI-agent workflows. It enforces structured sub-agent handoffs, read-only validation gates, and fail-closed completion rules to help ensure completion claims are admitted only when they satisfy repository verification policy.

> [!NOTE]
> **Local Development**: the Go CLI is the native rewrite target and the TypeScript CLI remains the compatibility baseline during migration. Build locally with `go build ./cmd/x-harness` and run `./x-harness <command>`, or build the TypeScript CLI with `npm install && npm run build` and run `node packages/cli/dist/index.js <command>`.

---

## 🎯 Core Philosophy

> **Completion is admitted, not merely claimed.**

Traditional agent harnesses focus primarily on generation (what code or text the agent should write). `x-harness` shifts the focus to verification and governance. It provides orchestrators and developers with a deterministic, rule-based gate to verify agent work products against strict repository policies.

In `x-harness`, an agent stating `fix_status: fixed` or running a passing test is simply producing a **completion candidate**. Actual completion is only **accepted** when the read-only verification gate runs its policy and admits the work.

---

## 🔰 Beginner's Fast-Track Guide (5-Minute Tour)

If you are new to `x-harness`, follow this step-by-step walkthrough to get up and running in minutes!

### Step 1: Install and Compile

Clone the repository and build the native Go CLI locally:

```bash
# Build the Go CLI binary
go build ./cmd/x-harness
```

The TypeScript CLI remains available as a compatibility baseline:

```bash
npm install
npm run build
```

### Step 2: Seven Canonical Actions

`x-harness` exposes seven beginner-friendly actions. Use these to interact with the harness:

| Action        | Alias for               | Description                                            |
| :------------ | :---------------------- | :----------------------------------------------------- |
| **`prepare`** | `handoff readiness`     | Check if workspace is ready for agent task handoff     |
| **`check`**   | `verify`                | Run read-only verification against a completion card   |
| **`recover`** | `recovery suggest`      | Get recovery playbook suggestions from errors or trace |
| **`doctor`**  | (standalone)            | Validate workspace health and configuration            |
| **`actions`** | (standalone)            | List all beginner-friendly actions                     |
| **`status`**  | `report` (no --metrics) | Show trace summary or card metrics                     |
| **`reset`**   | `clean --tmp --force`   | Clean generated harness state (requires --confirm)     |

You can use either the alias or the full command:

```bash
# These are equivalent:
./x-harness check --card completion-card.yaml
./x-harness verify --card completion-card.yaml

# These are equivalent:
./x-harness prepare --json
./x-harness handoff readiness --json

# These are equivalent:
./x-harness recover --errors "test failed"
./x-harness recovery suggest --errors "test failed"

# status shows trace summary:
./x-harness status
./x-harness report

# reset cleans harness state safely:
./x-harness reset --confirm
```

**Slash commands for agent adapters:** When integrating with agent platforms (Claude Code, Cursor, etc.), use slash-facing syntax like `/xh-check`, `/xh-prepare`, `/xh-recover`, `/xh-doctor`, `/xh-actions`, `/xh-status`, `/xh-reset` to invoke these actions.

### Step 3: Run Your First Verification

`x-harness` comes with pre-packaged reference scenarios called "Golden Examples". Let's run a verification against a successful task claim:

```bash
./x-harness check --card examples/golden/success-light/completion-card.yaml
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
./x-harness check --card examples/golden/blocked-missing-evidence/completion-card.yaml
```

> **Expected Output:**
>
> ```yaml
> outcome: failed
> acceptance_status: withheld
> ```
>
> _Note: The CLI returns exit code `1` (fail-closed) because the card failed the required evidence floor rules._

### Step 4: Initialize a New Workspace

To start using `x-harness` in a separate development project, run the `init` command in the root of your project:

```bash
# Set up a Minimal workspace (default; installs agents contract, templates, and policies)
./x-harness init --minimal
```

If the target directory already contains conflicting harness files, `init` stops with a blocked summary and exits non-zero. Re-run with `--force` only when you intentionally want to overwrite those files, or use `--merge` if/when you want non-destructive merge behavior.

### Step 5: Dispatch a Task Handoff

When assigning a task to an agent, generate a structured handoff prompt. For example, to dispatch a normal task:

```bash
./x-harness handoff standard --title "Fix Checkout Page Button Alignment"
```

This generates a markdown file matching the `standard` tier containing explicit file sets, required evidence checklists, and rollback definitions.

### Step 6: Validate Workspace Health

Run the diagnostics command at any time to verify that all schemas, policies, templates, and links are healthy:

```bash
./x-harness doctor --json
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

`x-harness` now has a Go-native CLI rewrite in active parity mode, with the TypeScript CLI retained as the compatibility baseline until the native binary becomes primary.

The Go binary is the native source-checkout path. TypeScript compatibility remains available through `node packages/cli/dist/index.js` and is still used as the parity baseline during the dual-run window.

### Core Commands

| Command         | Usage                                                                                                                                                                                                                                                              | Description                                                                                                                |
| :-------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------- |
| **`init`**      | `node packages/cli/dist/index.js init [target_dir] [--minimal / --standard / --full]`                                                                                                                                                                              | Installs the core harness assets, schemas, policies, and adapters. Default is `--minimal`.                                 |
| **`handoff`**   | `node packages/cli/dist/index.js handoff <light / standard / deep> [--title <text>] [--task <text>]`                                                                                                                                                               | Generates a clean markdown handoff task prompt structure.                                                                  |
| **`add`**       | `node packages/cli/dist/index.js add <claim / evidence / completion-card> [key=value]`                                                                                                                                                                             | Adds a metadata helper file for compatibility modes.                                                                       |
| **`verify`**    | `./x-harness verify [--card <path>] [--json] [--verbose] [--trace] [--trace-dir <dir>] [--subagent-return <path>] [--tier <tier>] [--task-id <id>] [--mutation-guard] [--strict]`                                                                                 | Executes the read-only verification policy against a completion card or compatibility subagent return. Supports tracing.   |
| **`doctor`**    | `./x-harness doctor [--root <path>] [--json] [--format <json\|text>]`                                                                                                                                                                                              | Checks critical file presence, schemas compilation, policies, and wording. JSON remains the default output for automation. |
| **`report`**    | `./x-harness report [--metrics] [--card <path>] [--json] [--format <markdown\|json>]`                                                                                                                                                                              | Summarizes verification events or calculates local card metrics. HTML remains available through the TypeScript CLI.        |
| **`trace`**     | `node packages/cli/dist/index.js trace add [--outcome <status>] [--task-id <id>] [--acceptance-status <status>] [--tier <tier>] [--claim-id <id>] [--evidence-id <id>]`                                                                                            | Manually appends verify events to the trace log. Supports full event metadata.                                             |
| **`clean`**     | `node packages/cli/dist/index.js clean [--tmp / --reset-card / --archive-success] [--force]`                                                                                                                                                                       | Defaults to a dry run; add `--force` to mutate tmp artifacts, reset a completion card, or archive accepted-card snapshots. |
| **`context`**   | `node packages/cli/dist/index.js context [--verbose / --json / --refresh] [--root <path>]`                                                                                                                                                                         | Shows canonical context and refreshes the AGENTS.md managed block.                                                         |
| **`examples`**  | `node packages/cli/dist/index.js examples`                                                                                                                                                                                                                         | Lists or copies built-in test-cases showing successful and blocked runs.                                                   |
| **`recovery`**  | `node packages/cli/dist/index.js recovery suggest [--errors <text>] [--outcome <status>] [--from <trace-file>] [--write] [--force] [--json]`                                                                                                                       | Generates structured recovery playbook suggestions from errors or trace files. Supports JSON output and candidate writing. |
| **`packet`**    | `node packages/cli/dist/index.js packet create --card <path>` or `packet verify-chain --task-id <id>`                                                                                                                                                              | Creates immutable claim packets from completion cards and verifies packet chain integrity.                                 |
| **`benchmark`** | `./x-harness benchmark [--filter <latency\|adversarial\|mutation-guard>] [--commands <list>] [--iterations <n>] [--mutation-files <list>] [--mutation-concurrency <list>] [--json]`                                                                               | Measures command latency, adversarial fixtures, and mutation guard git/non-git fallback latency.                           |

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

Running `./x-harness verify --trace` logs a JSONL event detailing the verification runtime parameters. These events can be aggregated using `./x-harness report` to track task success rates and blocked items over time.

### Deterministic Offline Metrics

`./x-harness report --metrics` calculates metrics under five categories:

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
├── cmd/
│   └── x-harness/          # Go CLI entrypoint
├── internal/               # Go runtime packages
├── packages/
│   └── cli/                # TypeScript compatibility CLI source
│       ├── src/
│       │   ├── commands/   # command-line sub-commands
│       │   ├── core/       # admission, metrics, and recovery engines
│       │   └── validators/ # Ajv schema validation
│       └── tests/          # CLI Unit and Integration tests
├── templates/              # Markdown templates for tasks & completion cards
├── schemas/                # JSON schemas for validating claims & cards
├── policies/               # admission and recovery YAML policies
├── docs/                   # Public user and contributor reference docs
├── adapters/               # Platform-specific instructions and rules
├── examples/               # Reference scenarios & golden test cases
```

---

## 📚 Documentation

| Document                                               | Description                                                                  |
| ------------------------------------------------------ | ---------------------------------------------------------------------------- |
| [`docs/README.md`](docs/README.md)                     | Public documentation index                                                   |
| [`docs/QUICKSTART.md`](docs/QUICKSTART.md)             | Quick start guide                                                            |
| [`docs/FAQ.md`](docs/FAQ.md)                           | Frequently asked questions                                                   |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)         | Architectural design, layer model, and validation flow                       |
| [`docs/SCHEMAS.md`](docs/SCHEMAS.md)                   | JSON schema inventory and validation guide                                   |
| [`docs/ADMISSION_POLICY.md`](docs/ADMISSION_POLICY.md) | Fail-closed admission rules and evidence floors                              |
| [`docs/VERIFY_GATE.md`](docs/VERIFY_GATE.md)           | Read-only verification gate mechanics                                        |
| [`docs/RUNTIME_CONTRACT.md`](docs/RUNTIME_CONTRACT.md) | Runtime contract between components                                          |
| [`docs/PACKETS.md`](docs/PACKETS.md)                   | Packet design spec and claim-only implementation guide                       |
| [`docs/RECOVERY.md`](docs/RECOVERY.md)                 | Recovery routing and playbook generation                                     |
| [`docs/ADAPTERS.md`](docs/ADAPTERS.md)                 | Platform adapter guide (Generic, Claude Code, Cursor, OpenCode, Antigravity) |
| [`docs/REPORT_FORMATS.md`](docs/REPORT_FORMATS.md)     | Report output formats: Markdown, JSON, HTML                                  |
| [`docs/CI.md`](docs/CI.md)                             | CI integration guide and local-build composite action                        |
| [`docs/CLEANUP.md`](docs/CLEANUP.md)                   | Cleanup and maintenance operations                                           |
| [`docs/RELEASE_SECURITY.md`](docs/RELEASE_SECURITY.md) | Release, SBOM, and provenance checks                                         |
| [`docs/NPM_WRAPPER_PLAN.md`](docs/NPM_WRAPPER_PLAN.md) | Plan for npm package transition to native Go binaries                        |

---

## 🤝 Project Health & Contribution

- **License**: MIT (`LICENSE`)
- **Contribution Guidelines**: See `CONTRIBUTING.md` and `templates/HARNESS_CHANGE_CONTRACT.md` before making harness-sensitive changes.
- **Project Health Checks**: Execute `./x-harness doctor` or `node packages/cli/dist/index.js doctor` regularly to ensure files, schemas, and policies are valid and aligned.
