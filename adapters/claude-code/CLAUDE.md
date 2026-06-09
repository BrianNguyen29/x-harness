# Claude Code x-harness Instructions

This adapter integrates x-harness with Claude Code.

## Workflow

1. **Orchestrator** (you or a dispatcher) selects a tier and writes a task.
2. **Implementation worker** performs the task and produces a completion card.
3. **Admission verifier** runs read-only verification against the card.
4. **Outcome**: accepted or withheld. If withheld, worker receives handoff guidance.

## Agent roles

- `agents/implementation-worker.md`: Performs edits, writes completion card.
- `agents/admission-verifier.md`: Read-only inspector; runs `x-harness verify`.

## Skills

- `skills/verify/SKILL.md`: Read-only verification skill.
- `skills/handoff/SKILL.md`: Tiered handoff creation.
- `skills/recovery/SKILL.md`: Recovery packet preparation.

## Quick commands

```bash
# Verify a completion card (use check or verify)
xh check --card completion-card.yaml --strict
# Source-checkout fallback:
# node packages/cli/dist/index.js check --card completion-card.yaml --strict

# Prepare workspace for handoff
xh prepare --json
# node packages/cli/dist/index.js prepare --json

# Recover from errors
xh recover --errors "test failed"
# node packages/cli/dist/index.js recover --errors "test failed"

# Check repo health
xh doctor --root .
# node packages/cli/dist/index.js doctor --root .

# View trace summary
xh report
# node packages/cli/dist/index.js report
```

## Constraints

- Use `light` by default.
- Verifier is read-only.
- PGV is advisory-only.
- Non-success verify outcomes are always withheld.
- Standard tasks require `done_checklist` and `prediction`; evidence scope (`verifies` / `does_not_verify`) is recommended.
- Deep tasks require scoped evidence, `state.read_set`, `state.write_set`, `done_checklist`, and `prediction`; governance is required for high-risk changes.

## Authoritative hierarchy

Chat summaries are non-authoritative. `completion-card.yaml` and `xh verify` output (or `node packages/cli/dist/index.js verify` as source-checkout fallback) are authoritative for completion state in this repository.

<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- generated-at: 2026-05-25T03:35:44.539Z -->
<!-- context-hash: 8817d535c4e04a79 -->

# x-harness Canonical Context

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

## Source-of-Truth Reading Order

The managed context block in AGENTS.md is authoritative. Files are read in this order:

1. AGENTS.md (managed block)
1. X_HARNESS.md
1. policies/admission.yaml
1. policies/recovery.yaml
1. policies/intake.yaml
1. schemas/completion-card.schema.json

## Rules

### Completion is admitted, not claimed
Agents may propose completion but cannot self-admit. A completion card with `claim.fix_status: fixed` is only a completion candidate. Compatibility subagent returns may use `result.fix_status`.

### Verifier is read-only
The verifier may inspect files, evidence, diffs, and trace events. It must not edit source files or repair the work product while verifying.

### Success is the only accepted outcome
`admission.outcome: success` and `acceptance_status: accepted` are required for admission. All other outcomes are withheld.

### Canonical tiers
Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

### PGV is advisory-only
Pre-gate validation (PGV) advice never overrides the verify gate and never grants admission authority by default.

<!-- END X-HARNESS MANAGED CONTEXT -->

<!-- BEGIN X-HARNESS MANAGED CONTRACT: claude-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: ec6438371a039c93 -->

## Generated Adapter Contract

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

## Evidence Floor

- **light**: files_changed + (command_evidence or manual_rationale).
- **standard**: files_changed + command_evidence + done_checklist + prediction.
- **deep**: files_changed + command_evidence + evidence_scope_declared + untested_regions_declared + remaining_risks_declared + execution_controls_present + rollback_policy_present + done_checklist + prediction. Runtime-enforced: verification_artifacts, state.read_set, state.write_set.

## Strict Evidence Provenance

- verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.
- verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.

<!-- END X-HARNESS MANAGED CONTRACT: claude-contract -->
