# Real-World Example: Monorepo Cross-Package Change

**Status: docs-only fixture — not part of default `examples verify` CI.**

This example demonstrates how a standard-tier x-harness completion card looks for a realistic monorepo cross-package change (e.g., adding shared validation logic consumed by a web app package). It is intended as a reference for teams adopting x-harness in monorepo codebases.

## What this demonstrates

- A plausible monorepo task scoped to two packages: `packages/shared` and `packages/web`.
- The **V1 monorepo path convention**:
  - `state.read_set`, `state.write_set`, and `evidence.files_changed` use **flat string arrays**.
  - Paths are **repo-root-relative**, use **forward slashes**, have **no leading `./`**, and are **not absolute**.
  - Package prefix convention: `packages/<name>/...` (package = first two path segments).
- A `standard` tier completion card with:
  - Cross-package `read_set` / `write_set` covering `packages/shared/...` and `packages/web/...`.
  - `evidence.command_evidence` showing `npm test -- shared`, `npm test -- web`, and `npm run typecheck`.
  - `evidence.verification_artifacts` per package with granular `verifies` / `does_not_verify` scope.
  - `untested_regions` and `remaining_risks` declarations.
  - `done_checklist` and `prediction` fields required for standard tier.
- How to verify the card directly with the Go CLI.

## Sample task

> Add a shared email/display-name validator in `packages/shared/src/validators/settings.ts` and consume it from `packages/web/app/settings/form.tsx`. Include unit tests in both packages and ensure `npm run typecheck` passes.

See [`task.md`](task.md) for the full task description.

## How to use x-harness (high-level flow)

1. **Intake** — classify the task tier based on scope and risk. A cross-package change with tests in each package is typically `standard`.
2. **Handoff** — the orchestrator routes to a fixer agent with the task, tier, and relevant context.
3. **Work** — the agent edits files, runs tests per package, and runs typecheck.
4. **Complete** — the agent writes a `completion-card.yaml` with evidence, read/write sets, prediction, and checklist.
5. **Verify** — run the verify gate:
   ```bash
   go run ./cmd/x-harness verify --card examples/real-world/monorepo/completion-card.yaml --json
   ```
   or with the built binary:
   ```bash
   ./x-harness verify --card examples/real-world/monorepo/completion-card.yaml --json
   ```

## Why this is docs-only / not in default `examples verify` CI

- The Go examples verifier (`internal/cli/examples.go`) discovers only `examples/golden/` with whitelisted suites: `regression`, `capability`, and `adversarial`.
- The TypeScript examples verifier recursively scans only `examples/golden/`.
- Therefore `examples/real-world/monorepo/` is invisible to the default `examples verify` command and safe as a docs-only fixture.
- It can still be verified directly with `x-harness verify --card <path>`.

## Monorepo V1 path convention (flat arrays)

- **Correct:**
  ```yaml
  state:
    read_set:
      - packages/shared/src/validators/settings.ts
      - packages/web/app/settings/form.tsx
    write_set:
      - packages/shared/src/validators/settings.ts
      - packages/shared/src/validators/settings.test.ts
      - packages/web/app/settings/form.tsx
      - packages/web/app/settings/form.test.tsx
  evidence:
    files_changed:
      - packages/shared/src/validators/settings.ts
      - packages/shared/src/validators/settings.test.ts
      - packages/web/app/settings/form.tsx
      - packages/web/app/settings/form.test.tsx
  ```
- **Incorrect:** nested objects, leading `./`, absolute paths, backslashes.

## Optional boundary policy

An optional [`policies/boundaries.yaml`](policies/boundaries.yaml) demonstrates a concise boundary rule:
- `packages/web` may depend on `packages/shared`.
- `packages/web` may **not** depend on `packages/internal`.

This file is **reference-only** and not required by admission in V1.

## Files

- `README.md` — this file
- `task.md` — sample task description
- `completion-card.yaml` — the completion card
- `evidence/typecheck.txt` — sample typecheck output
- `evidence/test-web.txt` — sample web package test output
- `evidence/test-shared.txt` — sample shared package test output
- `policies/boundaries.yaml` — optional boundary policy reference

## Expected verify outcome

```bash
./x-harness verify --card examples/real-world/monorepo/completion-card.yaml --json
# -> admission.outcome: success, acceptance_status: accepted
```
