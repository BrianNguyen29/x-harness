# Real-World Example: Next.js / TypeScript App Slice

**Status: docs-only fixture — not part of default `examples verify` CI.**

This example demonstrates how a standard-tier x-harness completion card looks for a realistic Next.js / TypeScript slice (e.g., adding or updating a product form or settings page). It is intended as a reference for teams adopting x-harness in React/Next.js codebases.

## What this demonstrates

- A plausible Next.js / TypeScript task scoped to a few app paths.
- A `standard` tier completion card with:
  - `state.read_set` and `state.write_set` scoped to app paths (`app/settings/page.tsx`, `app/settings/form.tsx`, etc.).
  - `evidence.command_evidence` showing `npm test` and `npm run typecheck`.
  - `evidence.verification_artifacts` with granular `verifies` / `does_not_verify` scope.
  - `untested_regions` and `remaining_risks` declarations.
  - `done_checklist` and `prediction` fields required for standard tier.
- How to verify the card directly with the Go CLI.

## Sample task

> Add a TypeScript settings page at `app/settings/page.tsx` with a form component in `app/settings/form.tsx`. Include client-side validation, unit tests, and ensure `npm run typecheck` and `npm run lint` pass.

See [`task.md`](task.md) for the full task description.

## How to use x-harness (high-level flow)

1. **Intake** — classify the task tier based on scope and risk. A single-page UI slice with tests is typically `standard`.
2. **Handoff** — the orchestrator routes to a fixer agent with the task, tier, and relevant context.
3. **Work** — the agent edits files, runs tests, typecheck, and lint.
4. **Complete** — the agent writes a `completion-card.yaml` with evidence, read/write sets, prediction, and checklist.
5. **Verify** — run the verify gate:
   ```bash
   go run ./cmd/x-harness verify --card examples/real-world/nextjs-app/completion-card.yaml --json
   ```
   or with the built binary:
   ```bash
   ./x-harness verify --card examples/real-world/nextjs-app/completion-card.yaml --json
   ```

## Why this is docs-only / not in default `examples verify` CI

- The Go examples verifier (`internal/cli/examples.go`) discovers only `examples/golden/` with whitelisted suites: `regression`, `capability`, and `adversarial`.
- The TypeScript examples verifier recursively scans only `examples/golden/`.
- Therefore `examples/real-world/nextjs-app/` is invisible to the default `examples verify` command and safe as a docs-only fixture.
- It can still be verified directly with `x-harness verify --card <path>`.

## Monorepo follow-up (deferred)

A monorepo variant (e.g., Next.js app + shared UI package) is deferred until the `read_set` / `write_set` cross-package contract is explicit in x-harness. When that contract lands, a follow-up slice can add `examples/real-world/nextjs-monorepo/` showing scoped evidence across package boundaries.

## Files

- `README.md` — this file
- `task.md` — sample task description
- `completion-card.yaml` — the completion card
- `evidence/typecheck.txt` — sample typecheck output
- `evidence/test.txt` — sample test output
- `evidence/lint.txt` — sample lint output

## Expected verify outcome

```bash
./x-harness verify --card examples/real-world/nextjs-app/completion-card.yaml --json
# -> admission.outcome: success, acceptance_status: accepted
```
