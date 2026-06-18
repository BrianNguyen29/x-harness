# Quickstart

Welcome to `x-harness`! Follow this guide to set up the CLI, run local verification checks, and initialize the harness in your project.

> [!NOTE]
> **Local Development**: build the native Go CLI with `go build ./cmd/x-harness` and run `./x-harness <command>`. The TypeScript compatibility CLI remains available after `npm install && npm run build` via `node packages/cli/dist/index.js <command>`.
>
> **Command syntax**:
>
> - **Terminal / shell:** `xh <command>` (e.g., `xh check`)
> - **Agent chat:** `/xh:<command>` (e.g., `/xh:check`). Legacy `/xh <command>` and `/xh-check` are also accepted where the adapter supports them.

---

## 🚀 Fast track

If you already have the CLI built, run the guided onboarding in one step:

```bash
xh start
```

This runs doctor, examples verify, and an init wizard preview in sequence. Add `--apply` to actually install assets.

---

## 🚀 Step-by-Step Setup

### 1. Install Dependencies & Build

First, clone the repository and build the native Go CLI:

```bash
go build ./cmd/x-harness
```

To use the TypeScript compatibility CLI instead:

```bash
npm install && npm run build
```

> [!TIP]
> **Source-pack limitation**: a source checkout after `npm run build` can run the Node fallback (`node packages/cli/dist/index.js`). The published npm tarball is **Go-only** and excludes `dist/`. If you run `npm pack` from a source checkout without injecting release Go binaries into `packages/cli/go-binaries/`, the wrapper will report a missing Go binary and missing Node fallback. To produce a runnable tarball, use the release workflow or manually inject a Go binary named `go-binaries/x-harness-<os>-<arch>` before packing.

### 2. Run the Health Check (Doctor)

Use the `doctor` command to verify that all schemas compile, YAML policies are valid, and required templates are present:

```bash
xh doctor --root . --json
```

You should see a JSON report detailing passing validations with `"healthy": true`.

### 3. Beginner Actions

`x-harness` provides 14 beginner-friendly actions:

| Action        | Description                                                                        |
| :------------ | :--------------------------------------------------------------------------------- |
| **`start`**   | Guided onboarding: doctor, examples verify, init wizard, next steps                |
| **`check`**   | Run read-only verification against a completion card                               |
| **`prepare`** | Check if workspace is ready for agent task handoff                                 |
| **`recover`** | Get recovery playbook suggestions from errors or trace                             |
| **`doctor`**  | Validate workspace health and configuration                                        |
| **`actions`** | List all beginner-friendly actions                                                 |
| **`status`**  | Show trace summary (alias for report without --metrics)                            |
| **`reset`**   | Clean generated harness state (requires --confirm)                                 |
| **`init`**    | Install core harness assets, schemas, policies, and adapters (default `--minimal`) |
| **`add`**     | Add a metadata helper file for compatibility modes                                 |
| **`learn`**   | Read-only concept tour for beginners                                               |
| **`quick`**   | Read-only next-action recommender for newcomers                                    |
| **`run`**     | Run a built-in workflow recipe                                                     |
| **`ci`**      | Run the built-in CI workflow                                                       |

### 4. Run Contract Oracle Checks (Optional)

Contract oracles are opt-in rule-based assertions. The default policy is empty-safe (no-op if no policy is present):

```bash
xh contract check --policy policies/contract-oracle.yaml --json .
```

### 5. Verify a Golden Example

The repository comes built-in with reference examples demonstrating different completion scenarios. Run verification against the "Success (Light Tier)" golden example using the `check` action:

```bash
xh check --card examples/golden/regression/success-light/completion-card.yaml
```

**Expected Success Output:**

```txt
outcome: success
acceptance_status: accepted
checks: 2 passed, 0 failed
```

_(The command returns an exit code of `0` because verification was successful and completion has been officially admitted)._

To run the same verification with contract oracle assertions enabled:

```bash
xh verify --card examples/golden/regression/success-light/completion-card.yaml --contract-oracles
```

Now, try verifying an example where standard tier verification is blocked due to missing evidence:

```bash
xh check --card examples/golden/regression/blocked-missing-evidence/completion-card.yaml
```

**Expected Withheld Output:**

```txt
outcome: failed
acceptance_status: withheld
checks: 0 passed, 5 failed
```

_(The command returns a non-zero exit code `1` because the evidence floor policy was not met. The task remains withheld)._

### 6. Initialize x-harness in Another Repository

To integrate `x-harness` into a separate development project, run the `init` command with the target directory:

```bash
# Minimal mode (default): AGENTS.md, X_HARNESS.md, verifier/runtime docs, handoff templates, admission policy, schemas
xh init --minimal ./my-project

# Standard mode: minimal assets + examples/01-solo-agent, examples/02-assisted-agent, schemas, policies, docs/ADAPTERS.md
xh init --standard ./my-project

# Full mode: standard assets + all examples, templates, adapters, and GitHub Actions
xh init --full ./my-project
```

If `init` finds conflicting harness files in the target workspace, it stops with a blocked summary and exits with a non-zero code instead of silently half-installing. Use `--force` only when you intend to overwrite those files.

### 7. Verify Your Own Completion Cards

When working on a task, write your completion card to `completion-card.yaml` and execute the verify gate using `check`. The Go CLI requires an explicit `--card` (or `--subagent-return`) path; it does not auto-discover `completion-card.yaml` in the current directory.

```bash
# Pass the completion card explicitly
xh check --card completion-card.yaml

# Advanced check with full check notes and handoff routing
xh check --card completion-card.yaml --verbose
```

---

## 📖 Next Docs to Read

To learn more about configuring and designing your agent verification workflow:

- 📑 [docs/SCHEMAS.md](SCHEMAS.md) — Learn about completion cards, subagent returns, and events validation schemas.
- 🚦 [docs/VERIFY_GATE.md](VERIFY_GATE.md) — Understand how the read-only admission verification policies operate.
- 📄 [docs/RUNTIME_CONTRACT.md](RUNTIME_CONTRACT.md) — Understand the runtime contract between harness components.
- 🔌 [docs/ADAPTERS.md](ADAPTERS.md) — Connect with Claude Code, Cursor, OpenCode, or Antigravity.
- 📚 [docs/README.md](README.md) — Browse the public documentation index.
