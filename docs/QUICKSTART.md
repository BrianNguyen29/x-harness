# Quickstart

Welcome to `x-harness`! Follow this guide to set up the CLI, run local verification checks, and initialize the harness in your project.

> [!NOTE]
> **Local Development Only**: `x-harness` is not yet published to npm. Use `node packages/cli/dist/index.js <command>` after building locally (`npm run build`).

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

### 3. Beginner Actions

`x-harness` provides seven beginner-friendly actions:

| Action       | Description                                              |
| :----------- | :------------------------------------------------------- |
| **`check`**  | Run read-only verification against a completion card        |
| **`prepare`** | Check if workspace is ready for agent task handoff        |
| **`recover`** | Get recovery playbook suggestions from errors or trace     |
| **`doctor`** | Validate workspace health and configuration                |
| **`actions`** | List all beginner-friendly actions                        |
| **`status`** | Show trace summary (alias for report without --metrics)  |
| **`reset`**  | Clean generated harness state (requires --confirm)        |

### 4. Verify a Golden Example

The repository comes built-in with reference examples demonstrating different completion scenarios. Run verification against the "Success (Light Tier)" golden example using the `check` action:

```bash
node packages/cli/dist/index.js check --card examples/golden/success-light/completion-card.yaml
```

**Expected Success Output:**

```txt
outcome: success
acceptance_status: accepted
checks: 1 passed, 0 failed
```

_(The command returns an exit code of `0` because verification was successful and completion has been officially admitted)._

Now, try verifying an example where standard tier verification is blocked due to missing evidence:

```bash
node packages/cli/dist/index.js check --card examples/golden/blocked-missing-evidence/completion-card.yaml
```

**Expected Withheld Output:**

```txt
outcome: blocked
acceptance_status: withheld
checks: 1 passed, 1 failed
```

_(The command returns a non-zero exit code `1` because the evidence floor policy was not met. The task remains withheld)._

### 5. Initialize x-harness in Another Repository

To integrate `x-harness` into a separate development project, run the `init` command in the root of that project:

```bash
# Minimal mode (default; installs core agent rules, verify gate config, and handoff templates)
node packages/cli/dist/index.js init --minimal

# Standard mode (Minimal + schemas, policies, and solo agent examples)
node packages/cli/dist/index.js init --standard

# Full mode (Standard + multi-agent examples, platform adapters, and GitHub Actions)
node packages/cli/dist/index.js init --full
```

If `init` finds conflicting harness files in the target workspace, it stops with a blocked summary and exits with a non-zero code instead of silently half-installing. Use `--force` only when you intend to overwrite those files.

### 6. Verify Your Own Completion Cards

When working on a task, write your completion card to `completion-card.yaml` and execute the verify gate using `check`:

```bash
# Default verify path looks for 'completion-card.yaml' in current directory
node packages/cli/dist/index.js check

# Advanced check with full check notes and handoff routing
node packages/cli/dist/index.js check --card completion-card.yaml --verbose
```

---

## 📖 Next Docs to Read

To learn more about configuring and designing your agent verification workflow:

- 📑 [docs/SCHEMAS.md](SCHEMAS.md) — Learn about completion cards, subagent returns, and events validation schemas.
- 🚦 [docs/VERIFY_GATE.md](VERIFY_GATE.md) — Understand how the read-only admission verification policies operate.
- 🔌 [docs/ADAPTERS.md](ADAPTERS.md) — Connect with Claude Code, Cursor, OpenCode, or Antigravity.
- 🧠 [docs/PRINCIPLES.md](PRINCIPLES.md) — Explore the core design decisions behind file-first lightweight governance.
