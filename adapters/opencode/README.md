# OpenCode Adapter

## Purpose

This adapter integrates `x-harness` with the **OpenCode** agent platform. It enables orchestrators to automatically route tasks to worker agents, gather completion cards, and verify them via OpenCode verify-agent files.

## Install

To install this adapter in your repository, run the copy command:

```bash
cp -r adapters/opencode/* .
```

This copies the verify agent instructions, orchestrator snippets, and OpenCode JSON environment example configurations to your workspace.

## Files included

- `verify-agent.md`: The instruction file read by the OpenCode verification agent. This guides the agent to act strictly in a read-only capacity.
- `opencode.example.json`: A template demonstrating how to register implementation worker agent pipelines in OpenCode.
- `opencode.verify.example.json`: A template configuration detailing how to register the verifier agent task and commands in OpenCode.
- `orchestrator_append.example.md`: Snippet to append to orchestrator handoffs, instructing the OpenCode dispatcher on handling outputs.

## Beginner-Friendly Actions

| Action       | Alias for              | Description                                              |
| :----------- | :--------------------- | :------------------------------------------------------- |
| **`prepare`** | `handoff readiness`   | Check if workspace is ready for agent task handoff        |
| **`check`**  | `verify`               | Run read-only verification against a completion card      |
| **`recover`** | `recovery suggest`    | Get recovery playbook suggestions from errors or trace     |
| **`doctor`** | (standalone)           | Validate workspace health and configuration               |
| **`actions`** | (standalone)           | List all beginner-friendly actions                        |
| **`status`** | `report` (no --metrics) | Show trace summary or card metrics                      |
| **`reset`**  | `clean --tmp --force` | Clean generated harness state (requires --confirm)        |

**Slash commands for agent adapters:** `/xh-check`, `/xh-prepare`, `/xh-recover`, `/xh-doctor`, `/xh-actions`, `/xh-status`, `/xh-reset`

## Workflow & Configurations

### 1. Set Up Worker and Verifier in OpenCode

You can use `opencode.example.json` and `opencode.verify.example.json` as guides to register agent containers or tasks in OpenCode:

- Copy `verify-agent.md` to your project root or specific task configurations directory (e.g. `.opencode/agents/`).
- The worker executes standard instructions, writes changes to the source files, and exports `completion-card.yaml`.
- The verifier loads `verify-agent.md`, mounts the workspace, and runs the read-only check command:
  ```bash
  node packages/cli/dist/index.js check --card completion-card.yaml
  # or: node packages/cli/dist/index.js verify --card completion-card.yaml
  ```

### 2. Output Analysis

- **Accepted**: Verification succeeds (exit code `0`). Handoff routes successfully to the orchestrator.
- **Withheld**: Verification is blocked or failed (exit code `1`). OpenCode handles the recovery guide generated in JSON format to re-dispatch to the worker.

## Constraints

- **Verifier is read-only**: Under no circumstances should the verifier agent rewrite files or edit code in the mounted workspace.
- **Strict Tiering**: Ensure that dispatches strictly declare standard `light`, `standard`, or `deep` tiers.
- **Fail-Closed**: Non-success outcomes are always withheld (`withheld`).
- **No heavy runtime required**: Fully local, offline-first.

## When to use

Use this adapter when running tasks inside the **OpenCode** container or agent environment. It standardizes the registration of implementation and verify agent blocks, ensuring OpenCode dispatchers can run verification commands cleanly.
