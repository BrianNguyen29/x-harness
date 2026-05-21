# Quickstart

Welcome to `x-harness`! Follow this guide to set up the CLI, run local verification checks, and initialize the harness in your project.

---

## 🚀 Step-by-Step Setup

### 1. Install Dependencies & Build
First, clone the repository, install the dependencies, and compile the TypeScript source code for the CLI tool:
```bash
# Install node packages
npm install

# Build the CLI application
npm run build
```

### 2. Run the Health Check (Doctor)
Use the `doctor` command to verify that all schemas compile, YAML policies are valid, and required templates are present:
```bash
node packages/cli/dist/index.js doctor --root .
```
You should see a JSON report detailing passing validations with `"healthy": true`.

### 3. Verify a Golden Example
The repository comes built-in with reference examples demonstrating different completion scenarios. Run verification against the "Success (Light Tier)" golden example:
```bash
node packages/cli/dist/index.js verify --card examples/golden/success-light/completion-card.yaml
```

**Expected Success Output:**
```txt
outcome: success
acceptance_status: accepted
checks: 1 passed, 0 failed
```
*(The command returns an exit code of `0` because verification was successful and completion has been officially admitted).*

Now, try verifying an example where standard tier verification is blocked due to missing evidence:
```bash
node packages/cli/dist/index.js verify --card examples/golden/blocked-missing-evidence/completion-card.yaml
```

**Expected Withheld Output:**
```txt
outcome: blocked
acceptance_status: withheld
checks: 1 passed, 1 failed
```
*(The command returns a non-zero exit code `1` because the evidence floor policy was not met. The task remains withheld).*

### 4. Initialize x-harness in Another Repository
To integrate `x-harness` into a separate development project, run the `init` command in the root of that project:
```bash
# Minimal mode (Installs core agent rules, verify gate config, and handoff templates)
npx x-harness init --minimal

# Standard mode (Minimal + schemas, policies, and solo agent examples)
npx x-harness init --standard

# Full mode (Standard + multi-agent examples, platform adapters, and GitHub Actions)
npx x-harness init --full
```

### 5. Verify Your Own Completion Cards
When working on a task, write your completion card to `completion-card.yaml` and execute the local verify gate:
```bash
# Default verify path looks for 'completion-card.yaml' in current directory
npx x-harness verify

# Advanced check with full check notes and handoff routing
npx x-harness verify --card completion-card.yaml --verbose
```

---

## 📖 Next Docs to Read
To learn more about configuring and designing your agent verification workflow:
- 📑 [docs/SCHEMAS.md](SCHEMAS.md) — Learn about completion cards, subagent returns, and events validation schemas.
- 🚦 [docs/VERIFY_GATE.md](VERIFY_GATE.md) — Understand how the read-only admission verification policies operate.
- 🔌 [docs/ADAPTERS.md](ADAPTERS.md) — Connect with Claude Code, Cursor, OpenCode, or Antigravity.
- 🧠 [docs/PRINCIPLES.md](PRINCIPLES.md) — Explore the core design decisions behind file-first lightweight governance.
