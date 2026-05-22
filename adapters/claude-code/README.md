# Claude Code Adapter

## Purpose

This adapter integrates `x-harness` with the **Claude Code** agent platform. It allows Claude Code to act as an implementation worker (drafting changes and producing a completion card) or as an admission verifier (performing read-only checks to authorize completion).

## Install

To install this adapter into your repository, run the following command:

```bash
cp -r adapters/claude-code/* .
```

This copies the `CLAUDE.md` instructions, the `agents/` role definitions, and the custom verification `skills/` directly to your project root where Claude Code can discover and execute them.

## Files included

- `CLAUDE.md`: Authoritative instructions read by Claude Code on project initialization.
- `agents/implementation-worker.md`: Guide describing the worker role's responsibilities for implementation and completion card creation.
- `agents/admission-verifier.md`: Guide describing the read-only inspector role's verification instructions.
- `skills/verify/SKILL.md`: Verification skill instruction.
- `skills/handoff/SKILL.md`: Skills instruction for handoff creation.
- `skills/recovery/SKILL.md`: Skills instruction for task recovery.

## Workflow

1. **Orchestrator** (user or parent agent) writes a task using a handoff template.
2. **Implementation Worker** (Claude Code) performs edits, tests the changes, and generates the `completion-card.yaml`.
3. **Admission Verifier** (Claude Code) runs read-only verification. In this repository, use:
   ```bash
   node packages/cli/dist/index.js verify --card completion-card.yaml
   ```
4. **Outcome**: Accepted (status code `0`) or Withheld (status code `1`). If withheld, the next actions are routed based on recovery rules.

## Constraints

- **Verifier is read-only**: The verifier must not write or edit files to fix validation issues during the verification stage.
- **Fail-closed**: Any outcome other than success is withheld (`withheld`).
- **PGV is advisory-only**: Policy Governance recommendations never override verify gate outcomes.
- **No heavy runtime required**: No background daemon, server, database, or MCP is needed.

## When to use

Use this adapter when you are running **Claude Code** as your primary developer or verification assistant. It instructs the LLM on how to follow `x-harness` completion rules and equips it with context-sensitive verification skills.
