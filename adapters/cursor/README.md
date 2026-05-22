# Cursor Adapter

## Purpose

This adapter integrates `x-harness` with the **Cursor** IDE agent platform. It provides Cursor with rules (`.cursorrules` or `.mdc` formats) to guide Cursor's composer or chat agent when working inside the repository.

## Install

To install this adapter in your project, copy the MDC rules file to the `.cursor/rules/` directory in your workspace:

```bash
mkdir -p .cursor/rules && cp adapters/cursor/rules/x-harness.mdc .cursor/rules/
```

Once copied, Cursor will automatically detect and apply these rules to all agent sessions inside the workspace.

## Files included

- `rules/x-harness.mdc`: The main Cursor rule file. It uses frontmatter configurations (`alwaysApply: true`) to ensure Cursor always follows `x-harness` protocols.
- `rules/claimgate.mdc`: (If present / legacy pointer) Backward-compatibility rule pointing to `x-harness.mdc`.

## Workflow

1. **Developer Handoff**: A task is dispatched to the Cursor Composer or agent.
2. **Work Phase**: The agent performs the required changes, implements tests, and documents findings.
3. **Card Creation**: The agent creates or updates the `completion-card.yaml` detailing claims and evidence.
4. **Verification**: In this repository, the developer or agent triggers `node packages/cli/dist/index.js verify` to check the card.
5. **Admission**: Completion is only accepted when verification succeeds.

## Constraints

- **Advisory-only rule**: Cursor rules act as a guide for the agent's behavior; they do not automatically execute verification or enforce policies. In this repository, run `node packages/cli/dist/index.js verify` to perform verification.
- **Strict Tier Labels**: The rules instruct Cursor to use only the canonical tiers (`light`, `standard`, `deep`).
- **No heavy runtime required**: Fully local, offline-first.

## When to use

Use this adapter if you or your team use the **Cursor** editor as the primary IDE or agent interface. It ensures Cursor agents adhere to the read-only verification model, fill out completion cards correctly, and use the correct handoff tiers.
