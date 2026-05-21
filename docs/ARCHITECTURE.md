# Architectural Design

`x-harness` is built on a **file-first, CLI-assisted** architecture. It prioritizes local configuration files and deterministic verification logic over network daemons, background databases, or heavy agent runtime services.

---

## 🗺️ Architectural Layer Model

```txt
┌────────────────────────────────────────────────────────┐
│                    ADAPTER LAYER                       │
│    (Claude Code  /  Cursor  /  OpenCode  /  Antigravity)│
└──────────────────────────┬─────────────────────────────┘
                           │ 1. Mount Rules/Workflows
                           ▼
┌────────────────────────────────────────────────────────┐
│                   TOOLING & CLI LAYER                  │
│       (init, handoff, add, verify, doctor, report)     │
└──────┬───────────────────┬─────────────────────────────┘
       │                   │
       │ 2a. Validate      │ 2b. Evaluate
       ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────────┐
│  VALIDATOR   │   │  ADMISSION   │   │     METRICS      │
│    LAYER     │   │ CONTROL LAYER│   │  REPORTING LAYER │
│(Ajv / Zod)   │   │(admission.ts)│   │  (metrics.ts)    │
└──────┬───────┘   └──────┬───────┘   └────────┬─────────┘
       │                  │                    │
       │ Load             │ Load Policies      │ Calculate
       ▼                  ▼                    ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────────┐
│ SCHEMAS FILE │   │ POLICIES FILE│   │   TRACING FILE   │
│   (schemas/) │   │  (policies/) │   │  (.x-harness/)   │
└──────────────┘   └──────────────┘   └──────────────────┘
```

---

## 🧱 Key Architectural Layers

### 1. Adapter Layer
Translates platform-specific conventions into unified `x-harness` parameters. It maps Cursor rules, Claude Code skills, and OpenCode workflows to standard input requirements without altering core execution behaviors.

### 2. Tooling & CLI Layer
Provides developer utilities to scaffolding templates (`init`, `handoff`), modify files (`add`), clear logs (`clean`), and execute audits (`verify`, `doctor`, `report`). Built strictly in TypeScript to guarantee optimal portability.

### 3. Validator Layer
Enforces complete structure verification on inputs via **Ajv (JSON Schema)** and **Zod (Types)** validation engines. This layer ensures that completion cards, sub-agent returns, and events perfectly comply with expected schemas prior to verification.

### 4. Admission Control Layer
Loads `policies/admission.yaml` and executes the core verification logic. It operates in a **strictly read-only** mode, ensuring the verification process does not mutate the directory files to fix logical or lint failures during checking.

### 5. Metrics & Reporting Layer
Computes deterministic, local-first performance metrics analyzing verification strength, state consistency, recovery ability, replayability, and execution costs without relying on external SaaS APIs or monitoring dashboards.

---

## 🔄 Core Validation Handoff Cycle

The interaction sequence for a standard verification run:

```txt
[Developer / Agent]
       │
       │ 1. Run CLI command: "npx x-harness verify --card completion-card.yaml"
       ▼
[CLI / index.ts]
       │
       │ 2. Load schemas (schemas/completion-card.schema.json)
       ▼
[Validators / completionCard.ts]
       │
       ├─► [FAIL] ──► Exits with Status 1 (Schema Validation Error)
       │
       └─► [PASS] ──► Loads Admission Rules (policies/admission.yaml)
                      │
                      ▼
             [Core / admission.ts]
                      │
                      ├─► [FAIL] ──► Suggested recovery route ──► Exit Status 1
                      │
                      └─► [PASS] ──► Output success ───────────► Exit Status 0
```
