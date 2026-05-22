# Frequently Asked Questions (FAQ)

---

## 🛠️ Environment & Tooling

### Why a TypeScript CLI?

Today, this repository is easiest to run in local development mode via `node packages/cli/dist/index.js ...` after `npm install` and `npm run build`. If a published package is added later, the same workflows can be exposed through a shorter installed CLI wrapper. The authoritative source of truth for the harness contract remains the repository file configurations (`policies/`, `schemas/`, `templates/`).

### Is Python required?

No. Core tooling, schemas validation, and policies evaluation are strictly TypeScript-first. Any experimental Python scripts or tools will live isolated in experimental subdirectories and are marked non-canonical.

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

---

## 🔌 Integration & Adapters

### Do I need to adopt every adapter in the repository?

No. You only need to adopt the adapter matching your specific agent or development platform. If you are developing code without platform-specific agents, you can use the **Generic** adapter which operates on standard Markdown conventions and completion cards.

### Which schema file should I start with?

Start with the `completion-card` schema (`schemas/completion-card.schema.json`). This is the primary data structure utilized by workers to claim completion, and it is the file evaluated by the `verify` gate.

### Can I use `x-harness` in agent frameworks other than OpenCode?

Yes! Through the platform adapters, `x-harness` seamlessly supports Claude Code, Cursor, OpenCode, Antigravity, and any generic development tool that can read Markdown instructions and run CLI commands.
