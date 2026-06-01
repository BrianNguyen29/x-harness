# Approval-Risk Advisory Example

A standalone, self-contained example that demonstrates the approval-risk
advisory note emitted by `x-harness verify`. The example keeps the
repository's default `policies/approval-risk.yaml` disabled (so the wider
repo is not affected) and ships a local policy under
`policies/approval-risk.yaml` with `enabled: true`.

## What this example proves

1. The approval-risk engine is wired into the verify pipeline.
2. A locally enabled policy emits the canonical advisory note:
   `approval-risk advisory: score=<n> risk_class=<class> signals=[...] required_approvals=<n>`.
3. The note is **advisory only**. It never alters `admission.outcome`,
   `acceptance_status`, errors, the blocking predicate, or
   `admission_authority`. A card with a `human_only` change can still be
   accepted; the advisory just signals that follow-up governance may be
   required.
4. The repository default policy remains `enabled: false` and is not
   touched by this example.

## Files

| File | Purpose |
| --- | --- |
| `completion-card.yaml` | A `standard`-tier completion card that claims a change to `sample-protected.yaml` (classified as `human_only` in the local authority policy). |
| `sample-protected.yaml` | A trivial file referenced by the card so the example has a concrete artifact at the protected path. |
| `policies/approval-risk.yaml` | Local policy with `enabled: true`. Mirrors the repository default shape and thresholds. |
| `policies/authority.yaml` | Minimal authority policy that classifies example-only paths as `human_only`. |

## Default policy is unchanged

The repository-wide `policies/approval-risk.yaml` continues to ship with
`enabled: false`. This example flips the bit only inside its own
`policies/` directory. Running verify from anywhere else (including the
repo root) keeps the default-disabled behavior.

## How to run

The verify command uses the working directory as the policy root, so the
example should be run from inside its own folder:

```bash
# from the repository root
(cd examples/approval-risk-advisory && \
  node ../../packages/cli/dist/index.js verify --card completion-card.yaml --json)
```

If the build is fresh, run `npm run build` first.

### Expected output

The JSON output should report:

- `ok: true`
- `admission_outcome: "success"`
- `acceptance_status: "accepted"`
- a `checks` entry whose `note` starts with
  `approval-risk advisory: score=...` (this is the advisory surface)

To confirm the advisory is policy-gated, temporarily flip
`policies/approval-risk.yaml` to `enabled: false` and re-run. The note
should disappear while the outcome remains `success` / `accepted`.

### Direct approval-risk evaluation

The same engine is also exposed as a subcommand:

```bash
(cd examples/approval-risk-advisory && \
  node ../../packages/cli/dist/index.js approval-risk evaluate \
    --card completion-card.yaml --json)
```

This prints the raw `ApprovalRiskReport` (score, risk_class, signals,
required_approvals, `admission_authority: false`).

## Notes for reviewers

- `governance.approval_status` is intentionally **not** set on the card,
  so the engine emits the `missing_governance_approval` signal in
  addition to `human_only_path`. The combined score demonstrates that
  the advisory surfaces both the path classification and the missing
  approval.
- The example is excluded from the `examples verify` golden runner
  because the golden runner does not execute the approval-risk engine
  and would otherwise ignore the local policy.
