---
description: Guides the Claude Code agent through completing a task, selecting a handoff tier, building a valid completion-card, and performing a clean task handoff.
allowed-tools: read_file, write_file, run_command, grep_search
---

# Claude Code Handoff Skill: Task Completion & Verification

This skill guides the Claude Code agent through the structured task handoff process, ensuring absolute compliance with `x-harness` governance rules.

---

## 🧭 Handoff Workflow

When you are ready to conclude a task, you must execute the following systematic steps:

```mermaid
graph TD
    A[Step 1: Select Canonical Tier] --> B[Step 2: Collect Evidence]
    B --> C[Step 3: Generate completion-card.yaml]
    C --> D[Step 4: Execute Local Verification Gate]
    D -->|Passed| E[Step 5: Propose Completion]
    D -->|Withheld/Failed| F[Step 6: Follow Recovery Path]
```

---

## 📋 Action Steps

### Step 1: Select the Canonical Handoff Tier
Review your task's complexity and impact to choose the narrowest tier that guarantees complete correctness.

> [!WARNING]
> You MUST ONLY use the canonical tier labels: `light`, `standard`, or `deep`.
> Do NOT use the words "small", "medium", or "large" anywhere in active runtime handoffs, completion cards, or logs, as this will trigger strict policy verification failures.

*   **`light` Tier**: Used for minor, superficial changes (e.g. styling, fixing typos, editing documentation). Requires basic summary claims.
*   **`standard` Tier**: Used for standard features, logical changes, or normal bugfixes. Requires declaring file read/write sets, local unit test runs, and documenting any untested regions.
*   **`deep` Tier**: Used for major structural code modifications, security-sensitive changes, or migrations. Requires independent read-only reviewer validation, cryptographic-grade evidence, and a comprehensive risk assessment.

### Step 2: Collect Evidence & Artifacts
Gather all execution details from your workspace:
1. **File sets**: Track all files you read and write during task execution.
2. **Local test runs**: Run the corresponding test suites and copy their exact terminal output commands.
3. **TypeScript checks**: Run typechecks (e.g. `npm run build` or `tsc`) to ensure no syntax/compilation issues exist.

### Step 3: Generate the completion-card.yaml
Create a standard `completion-card.yaml` at the root of the workspace or in your designated handoff directory following the `templates/COMPLETION_CARD.md` template.
Ensure the following blocks are populated matching the tier requirements:
*   **claim**: Containing a brief summary of what changed and the fix status.
*   **evidence**: Declaring file diffs and exact test commands run.
*   **state**: Mapping absolute read and write sets.

### Step 4: Run the Local Verify Gate
Never propose completion without verifying your work product first! Run the local verification gate using `check`:
```bash
node packages/cli/dist/index.js check --card completion-card.yaml
# or: node packages/cli/dist/index.js verify --card completion-card.yaml
```
*   **Outcome - Success**: If it outputs `outcome: success` with `acceptance_status: accepted`, proceed to Step 5.
*   **Outcome - Withheld**: If the verify gate is withheld or fails, look at the returned validation errors, perform the necessary repairs, and re-run verification.

### Step 5: Propose Handoff & Completion
Construct a clean final response for the user, referencing the generated completion card and presenting your validation logs. Remember: *Agents may propose handoffs, but only the read-only verify gate can admit task completion!*
