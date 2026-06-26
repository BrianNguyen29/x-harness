# Deep & Governed Verification

This tutorial covers the `deep` tier and the `governed-deep` verify profile. These are used when the task is high-risk, long-running, or requires explicit human approval before admission.

> Every behavior described here is supported by the current Go CLI and TypeScript compatibility baseline. Nothing is aspirational.

---

## When to use deep tier

Choose `deep` when:

- The change touches critical infrastructure (auth, billing, deployment pipelines).
- The task spans multiple modules and has significant rollback risk.
- You need enforced evidence scope, declared untested regions, and explicit execution controls.

## Evidence floor for deep

A `deep` card must satisfy the standard floor **plus** the following:

| Field | Required | Notes |
| :-- | :-- | :-- |
| `evidence.files_changed` | Yes | Every file the agents touched. |
| `evidence.command_evidence` | Yes | Commands with `runner`, `exit_code`, `started_at`. |
| `evidence.verification_artifacts` | Runtime-enforced | Must include `kind`, `status`, and `command` details for strict profiles. |
| `evidence.untested_regions` | Yes | Explicitly declare what is not covered. |
| `evidence.remaining_risks` | Yes | Known risks after the fix. |
| `evidence.execution_controls` | Yes | How the change was scoped or gated. |
| `evidence.rollback_policy` | Yes | Steps to revert if the change causes issues. |
| `state.read_set` | Runtime-enforced | Files read during the task. |
| `state.write_set` | Runtime-enforced | Files written during the task. |
| `done_checklist` | Yes | Cross-checked against evidence and state. |
| `prediction` | Yes | Claim, expected effect, falsification method, and horizon. |

> See [ADMISSION_POLICY.md](../ADMISSION_POLICY.md) for the full generated contract.

## Mutation guard

Deep-tier verification automatically enables the **mutation guard** (`--mutation-guard`). The guard snapshots the workspace before and after the verify pipeline and fails closed if any unexpected file change is detected. This guarantees the verifier remains read-only.

```bash
xh verify --card completion-card.yaml --profile governed-deep
# mutation guard is on by default for this profile
```

For latency benchmarks of the guard:

```bash
xh benchmark --filter mutation-guard --json
```

## Human approval gate

If the card sets `governance.requires_human_approval: true`, admission is blocked unless `governance.approval_status` is `approved`.

```yaml
governance:
  risk_class: high
  requires_human_approval: true
  approval_status: approved   # pending or missing blocks admission
  approver: "security-lead"
```

> Example: `examples/golden/capability/deep-approval-required/` shows a card that is withheld because approval is missing, even though all other checks pass.

## Enforce flags and profiles

The verify gate supports four preset profiles that control how strict the checks are:

| Profile | Mutation guard | Context floor | Contract oracles | Boundary / Decision / Intent / Context enforce |
| :-- | :-- | :-- | :-- | :-- |
| `light-local` | Off | No | Off | Advisory only, never blocks |
| `ci-standard` | On | Yes | Off | Advisory only |
| `ci-strict` | On | Yes | On | Blocks high/critical boundary violations and missing decision refs |
| `governed-deep` | On | Yes | On | Blocks **all** boundary violations, missing decision refs, missing/blank intent ref, and stale context manifests |

Explicit flags always override profile defaults:

```bash
xh verify --card completion-card.yaml \
  --profile governed-deep \
  --boundary-enforce block_all \
  --decision-enforce block \
  --intent-enforce block \
  --context-enforce block
```

## Context manifest freshness

Governed profiles enforce context manifest checks. Generate a manifest of tracked file hashes before the task:

```bash
xh context manifest write --manifest .x-harness/context-manifest.yaml
```

Verify it during the gate:

```bash
xh context manifest check --manifest .x-harness/context-manifest.yaml
```

When `--context-enforce block` is enabled, a stale manifest blocks admission.

## Boundary policy enforcement

Boundary checks (`xh boundary`) are opt-in and loaded from `policies/boundaries.yaml`. Under `governed-deep`, all boundary violations are blocking unless suppressed by a valid `boundary_approvals` entry:

```yaml
boundary_approvals:
  - rule_id: "no-db-import-in-frontend"
    approver: "architect"
    approved_at: "2026-06-26T10:00:00Z"
    reason: "Migration window approved"
```

Malformed or rule-only entries are ignored and the finding remains blocking.

## Step-by-step deep verification

### 1. Prepare the card

Fill every deep-tier field. Use `xh evidence run -- <command>` to produce timestamped, hash-backed command evidence whenever possible.

### 2. Write the context manifest

```bash
xh context manifest write
```

### 3. Run the gate with governed-deep

```bash
xh verify --card completion-card.yaml --profile governed-deep
```

### 4. Interpret withheld results

If the result is `withheld`, inspect the JSON output:

```json
{
  "withheld_reason": {
    "class": "evidence_floor_missing",
    "stage": "admission",
    "blocking_predicate": "deep_evidence_floor",
    "owner": "worker",
    "next_action": "add_verification_artifacts_and_state_sets"
  }
}
```

Use `xh explain --card completion-card.yaml` for a human-readable summary, then patch and re-verify.

### 5. Approval (if required)

If governance requires human approval, add the `approved` status and the approver identity to the card, then re-verify.

### 6. Archive artifacts

Keep the card, the verify JSON output, and the context manifest in the workspace for audit.

---

## Real-world examples in this repo

- `examples/golden/capability/deep-approval-required/` — Blocked because human approval is missing.
- `examples/golden/regression/blocked-missing-evidence-scope/` — Blocked because deep scope fields are absent.
- `examples/ci/governed-deep-verify/` — CI fixture for the governed-deep profile.

## Next docs

- [Multi-Agent Workflow](multi-agent-workflow.md) — Standard-tier collaborative tasks.
- [Verify Gate](../VERIFY_GATE.md) — Full failure taxonomy, enforce flags, and profile definitions.
- [Admission Policy](../ADMISSION_POLICY.md) — Evidence floor and rejection conditions.
- [Evidence Provenance](../EVIDENCE_PROVENANCE.md) — Hash-backed command evidence and CI binding.
