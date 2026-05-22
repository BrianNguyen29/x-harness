# Paradigm Comparison

`x-harness` is designed to bridge the gap between human intentions and agent implementations. This document compares different task verification paradigms to highlight where `x-harness` excels.

---

## 📊 Feature Matrix Comparison

| Feature                  | 📝 Plain Markdown Checklist | 🧪 Standard CI Test Suite              | 🤖 Heavyweight Agent Framework                 | ⚡ x-harness                                                            |
| :----------------------- | :-------------------------- | :------------------------------------- | :--------------------------------------------- | :---------------------------------------------------------------------- |
| **Complexity**           | Extremely Low (Zero Setup)  | Moderate (Config overhead)             | Very High (Substantial runtime dependencies)   | **Low (TypeScript CLI, offline-first)**                                 |
| **State Verification**   | Manual & subjective         | Automated but narrow (only tests code) | Managed but opaque (agent self-verifies)       | **Deterministic, read-only policy check**                               |
| **Handoff Structure**    | Ad-hoc templates            | Absent                                 | Built-in (but tightly coupled to runtime)      | **Decoupled tiered templates (`light`, `standard`, `deep`)**            |
| **Handoff Recovery**     | No guidance (manual path)   | Log output only                        | Agentic loops (can cause infinite retry loops) | **Structured recovery routing (JSON-defined owner/actions)**            |
| **Governance Gate**      | Human review required       | Passing build = Done                   | Agent handles final admission                  | **Fail-closed verify gate (Accepted / Withheld)**                       |
| **Platform Portability** | Universal                   | Universal                              | Tied to specific runtime/API/database          | **Universal (Adapters for Claude Code, Cursor, OpenCode, Antigravity)** |

---

## 🔍 Paradigm Breakdown

### 1. Plain Markdown Checklist

- **How it works**: Developers write a standard checklist (e.g., `todo.md`) and check items off manually.
- **The Issue**: Agents easily miss requirements, check boxes prematurely, or misinterpret success criteria without verification checks.
- **The x-harness Solution**: Converts checklists into structured tiered dispatches and checks execution against strict schemas via local CLI gates.

### 2. Standard CI Test Suite

- **How it works**: Standard testing tools (e.g., Jest, Vitest, PyTest) check if the code passes assertion statements.
- **The Issue**: Tests confirm code behavior, but cannot check if the agent documented findings, analyzed risks, declared untested regions, or respected handoff routing.
- **The x-harness Solution**: Mounts on top of tests, utilizing test runs as **evidence** (via `command_evidence`) to evaluate overall task completion quality, structure, and accountability.

### 3. Heavyweight Agent Frameworks

- **How it works**: Complex, multi-agent frameworks (e.g., CrewAI, Autogen) orchestrate agent tasks through databases, background daemons, or MCP servers.
- **The Issue**: High resource cost, complex initialization, vendor lock-in, and agents frequently self-verify their own code, leading to false-positives and overclaims.
- **The x-harness Solution**: Enforces a strict **read-only** verification barrier where implementation workers and verification gates are isolated, ensuring complete audit transparency without system overhead.
