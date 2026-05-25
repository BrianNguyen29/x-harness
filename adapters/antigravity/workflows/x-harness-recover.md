# x-harness Recover Workflow

## Goal

Handle blocked verification outcomes without converting blocked to success.

## Trigger

- Verification returns `blocked`.
- `acceptance_status: withheld`.

## Steps

1. **Read verify output**: Identify blocking_predicate and reason.
2. **Determine cause**:
   - Missing evidence?
   - Stale context?
   - Partial fix?
   - Missing owner?
3. **Assign recovery**:
   - next_action: specific step to resolve blocker.
   - owner: who is responsible for the next step.
4. **Execute recovery**:
   - If missing evidence: worker attaches evidence.
   - If stale context: refresh context.
   - If partial fix: return to implementation.
5. **Re-verify**: Run `node packages/cli/dist/index.js verify` again after recovery in this repository.

## Rules

- Never convert blocked directly to success.
- Always assign next_action and owner.
- Always re-run verification after recovery.
- PGV advice remains advisory-only during recovery.

## Output

```yaml
admission:
  outcome: blocked
  blocking_predicate: <predicate>
  reason: <reason>
acceptance_status: withheld

handoff:
  next_action: <next action>
  owner: <owner>
```

<!-- BEGIN X-HARNESS MANAGED CONTRACT: antigravity-recover-workflow-contract -->
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

<!-- END X-HARNESS MANAGED CONTRACT: antigravity-recover-workflow-contract -->
