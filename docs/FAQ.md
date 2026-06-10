# Frequently Asked Questions (FAQ)

---

## 🛠️ Environment & Tooling

### Why a Go CLI with TypeScript compatibility?

The native Go CLI improves startup, distribution, and release artifact ergonomics. The TypeScript CLI remains as a compatibility baseline during migration so parity can be checked before Go becomes the default package entrypoint. The authoritative source of truth for the harness contract remains the repository file configurations (`policies/`, `schemas/`, `templates/`).

### Is Python required?

No. Core tooling, schema validation, and policy evaluation are implemented in the Go CLI with TypeScript compatibility retained during migration. Any experimental Python scripts or tools will live isolated in experimental subdirectories and are marked non-canonical.

### Do I need a database, background server, or MCP configuration?

No. `x-harness` is entirely local, offline-first, and file-based. It has no long-running daemons, network connections, or database dependencies.

---

## 🚦 Verification & Outcomes

### Does `x-harness` require an LLM or AI model to verify?

No. Unlike generative tools, verification checks in `x-harness` are entirely deterministic-first. They evaluate JSON/YAML fields, check schema structures, and validate evidence floors using standard code logic.

### What happens when verification fails or is blocked?

The verification tool yields a non-zero exit code (`1`) and outputs a structured **recovery path**. This recovery object identifies the blocking predicate (e.g. `evidence_missing`) and suggests a direct owner (e.g. `implementation-worker`) and recovery action.

### What is the difference between "Accepted" and "Withheld" outcomes?

- **Accepted**: Verification has fully passed and the task is admitted (returns exit code `0`).
- **Withheld**: Any other outcome (failed, blocked, skipped, timeout, error). The task is withheld and not considered completed (returns exit code `1`).

### Does `x-harness` replace my unit and integration testing frameworks?

No. `x-harness` sits on top of your test suite. It uses your test logs and code compilation as **evidence** (via `command_evidence`) to decide if a task should be admitted, but it does not execute the actual application tests itself.

### What is Contract Oracle?

Contract Oracle is an **opt-in** verification stage that enforces line-level policy rules via `verify --contract-oracles` or as a standalone check via `contract check`. It supports two rule types:

- **`grep_rules`**: pattern matches against file contents (no AST, package graph, or lockfile parsing).
- **`dependency_rules`**: checks for required or forbidden file references.

Contract Oracle is off by default and must be explicitly enabled. It is useful for enforcing lightweight contractual constraints without full static analysis.

---

## 🔌 Integration & Adapters

### Do I need to adopt every adapter in the repository?

No. You only need to adopt the adapter matching your specific agent or development platform. If you are developing code without platform-specific agents, you can use the **Generic** adapter which operates on standard Markdown conventions and completion cards.

### Which schema file should I start with?

Start with the `completion-card` schema (`schemas/completion-card.schema.json`). This is the primary data structure utilized by workers to claim completion, and it is the file evaluated by the `verify` gate.

### Can I use `x-harness` in agent frameworks other than OpenCode?

Yes! Through the platform adapters, `x-harness` seamlessly supports Claude Code, Cursor, OpenCode, Antigravity, Codex, and any generic development tool that can read Markdown instructions and run CLI commands.
