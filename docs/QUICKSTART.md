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

## Fast Track

In the project where you want to use x-harness, start with the core loop:

```bash
xh init --minimal
xh doctor --root . --json
xh verify --card completion-card.yaml
```

If you are only trying the x-harness source checkout, skip `init` and verify a
bundled fixture:

```bash
xh doctor --root . --json
xh verify --card examples/golden/regression/success-light/completion-card.yaml
```

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

### 2. Initialize A Workspace

Install the minimal harness assets in the target repository:

```bash
xh init --minimal ./my-project
```

Use `--standard` or `--full` only when you intentionally want examples,
adapters, or GitHub Actions copied into the target repository.

### 3. Run the Health Check (Doctor)

Use the `doctor` command to verify that all schemas compile, YAML policies are valid, and required templates are present:

```bash
xh doctor --root . --json
```

You should see a JSON report detailing passing validations with `"healthy": true`.

### 4. Run Your First Verification

Verification is the central command. Use `verify` or its beginner alias
`check`:

```bash
xh verify --card completion-card.yaml
# same gate:
xh check --card completion-card.yaml
```

### 5. Run Contract Oracle Checks (Optional)

Contract oracles are opt-in rule-based assertions. The default policy is empty-safe (no-op if no policy is present):

```bash
xh contract check --policy policies/contract-oracle.yaml --json .
```

### 6. Verify a Golden Example

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
- 🧾 [docs/EVIDENCE_PROVENANCE.md](EVIDENCE_PROVENANCE.md) — Capture command evidence with hashes, CI binding, and provenance.
- 🛡️ [docs/THREAT_MODEL.md](THREAT_MODEL.md) — Understand what x-harness does and does not protect against.
- 📄 [docs/RUNTIME_CONTRACT.md](RUNTIME_CONTRACT.md) — Understand the runtime contract between harness components.
- 🔌 [docs/ADAPTERS.md](ADAPTERS.md) — Connect with Claude Code, Cursor, OpenCode, or Antigravity.
- 📚 [docs/README.md](README.md) — Browse the public documentation index.
