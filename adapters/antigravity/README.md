# Antigravity Adapter

## Purpose

This adapter integrates `x-harness` with the **Antigravity** agent platform. It provides the constraints, rules, and workflows to govern task execution and verification for Antigravity-driven agents.

## Install

To install this adapter in your project, copy the configuration files and workflows to your repository:

```bash
cp -r adapters/antigravity/* .
```

This will place the rules under `rules/` and execution workflows under `workflows/` in your workspace, allowing your Antigravity agent system to discover and follow them.

## Files included

- `rules/x-harness.md`: Core rules for Antigravity agents defining handoff tiers, completion invariants, and verifier requirements.
- `workflows/x-harness-implementation.md`: Workflow detailing the implementation stage and completion card generation.
- `workflows/x-harness-verify.md`: Workflow for running the read-only verify gate.
- `workflows/x-harness-recover.md`: Detailed guidance on handling task failures and routing blocked actions.

## Workflow

1. **Initiate Task**: Dispatch a task utilizing one of the tiered handoff templates.
2. **Execute & Test**: The implementation agent performs changes, tests execution, and writes a `completion-card.yaml`.
3. **Trigger Verify Gate**: Run the verification utility in read-only mode:
   ```bash
   node packages/cli/dist/index.js verify --card completion-card.yaml
   ```
4. **Resolution**: If verification succeeds, mark as `accepted`. If it fails, routing guidelines from `x-harness-recover.md` are executed based on the blocking predicates.

## Constraints

- **Strict Read-Only Verification**: The verifier must not write files or attempt to fix compilation/logical errors during the check phase.
- **Strict Tiers**: Antigravity runs must strictly use `light`, `standard`, and `deep` tiers.
- **Fail-Closed**: Non-success outcomes are always withheld (`withheld`).
- **No heavy runtime required**: Fully local, offline-first.

## When to use

Use this adapter when running the **Antigravity** agent framework to ensure that its development loops, verification checks, and recovery routines align seamlessly with the repository's x-harness policies.
