# x-harness Agent Contract

This repository uses x-harness.

Agents may perform work and propose completion. Agents may not self-admit completion.

A result with `fix_status: fixed` is only a completion candidate. Accepted completion requires a read-only verify gate pass.

## Canonical tiers

Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

## Completion rule

Completion is accepted only when x-harness emits:

```yaml
admission.outcome: success
acceptance_status: accepted
```

All other outcomes are withheld: `failed`, `blocked`, `skipped`, `timeout`, and `error`.

## Verifier rule

The verifier is read-only. It may inspect files, tasks, stories, templates, returns, evidence, diffs, command output, and trace events. It must not edit source files or repair the work product while verifying.

## PGV rule

PGV advice is advisory-only. It never overrides verify and never grants admission authority by default.

---

## Project shape

- TypeScript-first, file-first, lightweight harness. No daemon, database, server, or runtime required.
- Workspace root delegates to `packages/cli` for build, dev, test, typecheck, lint, and verify.
- CLI is local development only; build first, then invoke via `node packages/cli/dist/index.js <command>`.

## Commands

Root workspace scripts (`package.json`):
- `npm run build` — compile CLI (`tsc`)
- `npm run test` — run unit tests (`vitest run`)
- `npm run typecheck` — typecheck without emit (`tsc --noEmit`)
- `npm run lint` — lint CLI (`eslint .`)
- `npm run verify` — alias for `typecheck && test`

CLI commands (`packages/cli/src/index.ts`):
`init`, `add`, `handoff`, `verify`, `trace`, `report`, `clean`, `examples`, `context`, `doctor`, `recovery`, `packet`

CI order (`.github/workflows/x-harness-verify.yml`):
`npm ci` → `typecheck` → `build` → `lint` → `format:check` → `test` → `doctor --root .` on Node 22.

## Verification & completion semantics

- Candidate: `result.fix_status: fixed` + `verification.status: passed` does **not** mean accepted.
- Accepted only when `admission.outcome: success` and `acceptance_status: accepted`.
- All non-success outcomes map to `withheld` and require `handoff.next_action` and `handoff.owner`.

## Completion card & evidence floor gotchas

Required top-level fields (`schemas/completion-card.schema.json`):
`schema_version`, `task_id`, `tier`, `owner`, `accountable`, `claim`, `verification`, `admission`, `acceptance_status`, `handoff`

Admission policy (`policies/admission.yaml`) requires:
- `claim.fix_status`, `verification.status`, `claim.evidence`, `handoff.next_action`, `handoff.owner`
- Success additionally requires: `fixed` + `passed` + `success` + `accepted` + non-empty evidence + owner + accountable + `evidence_floor_met` + `admission_mapping_valid` + no unresolved blockers + read-only verifier

Evidence floor by tier:
- **light**: `files_changed` + (`command_evidence` or `manual_rationale`)
- **standard**: `files_changed` + `command_evidence`
- **deep**: `files_changed` + `command_evidence` + `evidence_scope` + `untested_regions` + `remaining_risks` + `execution_controls` + `rollback_policy`

## Where to look

- Full contract: `X_HARNESS.md`
- Admission rules: `policies/admission.yaml`
- Recovery rules: `policies/recovery.yaml`
- Card schema: `schemas/completion-card.schema.json`
- Handoff templates: `templates/SUBAGENT_TASK_{light,standard,deep}.md`
- Platform adapters: `adapters/opencode/`, `adapters/claude-code/`, `adapters/cursor/`, `adapters/generic/`, `adapters/antigravity/`
- Golden examples: `examples/golden/`
<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- generated-at: 2026-05-22T12:04:44.119Z -->
<!-- context-hash: fc32fc67ec9fc441 -->

# x-harness Canonical Context

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

## Rules

### Completion is admitted, not claimed
Agents may propose completion but cannot self-admit. A result with `fix_status: fixed` is only a completion candidate.

### Verifier is read-only
The verifier may inspect files, evidence, diffs, and trace events. It must not edit source files or repair the work product while verifying.

### Success is the only accepted outcome
`admission.outcome: success` and `acceptance_status: accepted` are required for admission. All other outcomes are withheld.

### Canonical tiers
Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

### PGV is advisory-only
Pre-gate validation (PGV) advice never overrides the verify gate and never grants admission authority by default.

<!-- END X-HARNESS MANAGED CONTEXT -->
