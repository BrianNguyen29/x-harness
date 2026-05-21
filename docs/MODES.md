# Operational Modes and Inventory

`x-harness` is highly modular, designed to run in three operational modes depending on the integration depth, project size, and governance needs: **Minimal**, **Standard**, and **Full**.

---

## 🎛️ Operational Modes

### 1. Minimal Mode (Foundational)
*   **Target**: Basic agent setups, fast proof-of-concept pipelines, or lightweight workflows.
*   **Focus**: Standardizing developer and agent contracts, checking simple check-off files, and running local verification scripts.
*   **Requirements**: Agent contract, runtime contract, verify gate, and three tier templates (`light`, `standard`, `deep`).

### 2. Standard Mode (Recommended)
*   **Target**: Professional development projects and mid-sized codebases.
*   **Focus**: Structural enforcement using JSON Schema validation, deterministic admission gates, and detailed evidence declaring scopes.
*   **Requirements**: Adds completion card structures, JSON schema validators, strict admission/recovery policies, and comprehensive golden testing examples.

### 3. Full Mode (Enterprise/Governance)
*   **Target**: Compliance-heavy enterprise pipelines, multi-platform editor environments, and automated CI/CD.
*   **Focus**: Absolute task auditability, platform-specific adapters (Claude Code, Cursor, OpenCode, Antigravity), custom verify gate actions, and deep local metrics.
*   **Requirements**: Adds specialized adapters, complete local metrics reporting, local doctor diagnostics, and advanced verify-report utilities.

---

## 📊 File Inventory Matrix

The table below catalogs which assets and folders are active under each operational mode:

| File / Component Path | 🔍 Minimal | 🧪 Standard | ⚡ Full | Description |
| :--- | :---: | :---: | :---: | :--- |
| [AGENTS.md](../AGENTS.md) | **Required** | **Required** | **Required** | Core agent contract & governance terms |
| [X_HARNESS.md](../X_HARNESS.md) | **Required** | **Required** | **Required** | Root map & integration directory |
| `templates/SUBAGENT_TASK_light.md` | **Required** | **Required** | **Required** | Light-tier task handoff template |
| `templates/SUBAGENT_TASK_standard.md`| **Required** | **Required** | **Required** | Standard-tier task handoff template |
| `templates/SUBAGENT_TASK_deep.md` | **Required** | **Required** | **Required** | Deep-tier task handoff template |
| `docs/RUNTIME_CONTRACT.md` | **Required** | **Required** | **Required** | Constraints governing agent operations |
| `docs/VERIFY_GATE.md` | **Required** | **Required** | **Required** | Read-only verify gate declarations |
| `schemas/completion-card.schema.json`| ❌ Optional | **Required** | **Required** | JSON Schema for verifying completion cards|
| `schemas/subagent-return.schema.json`| ❌ Optional | **Required** | **Required** | JSON Schema validating subagent outcomes|
| `policies/admission.yaml` | ❌ Optional | **Required** | **Required** | Set of admission and acceptance criteria |
| `policies/recovery.yaml` | ❌ Optional | **Required** | **Required** | Configured actions on withheld/failed gates|
| `adapters/generic/` | ❌ Optional | ❌ Optional | **Required** | Broad editor integration instructions |
| `adapters/claude-code/` | ❌ Optional | ❌ Optional | **Required** | Custom rules/skills for Anthropic Claude |
| `adapters/cursor/` | ❌ Optional | ❌ Optional | **Required** | Custom rules/instructions for Cursor IDE |
| `adapters/opencode/` | ❌ Optional | ❌ Optional | **Required** | Configuration and setups for OpenCode |
| `adapters/antigravity/` | ❌ Optional | ❌ Optional | **Required** | Integration scripts for Antigravity |
| `docs/METRICS.md` | ❌ Optional | ❌ Optional | **Required** | Diagnostics and evaluation metric matrices |

---

## 🚀 Scaling Between Modes

Moving between modes is seamless and requires zero code modification:
1. **Initialize**: Begin with **Minimal Mode** by placing `AGENTS.md` and the templates in your repository.
2. **Automate**: Transition to **Standard Mode** by installing the `x-harness` CLI to automatically validate templates using schemas.
3. **Govern**: Scale to **Full Mode** by activating platform adapters and integrating `x-harness verify` into your GitHub Actions or GitLab CI runner pipelines.
