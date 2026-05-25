# Optional Enterprise Controls

This page documents optional controls that help enterprise operators reason about approval risk, observed agent capability, and cost budget. They are disabled or advisory by default and do not grant admission authority.

## Invariants

- Admission still comes only from `verify` producing `admission.outcome: success` with `acceptance_status: accepted`.
- These controls report advisory metadata. They do not override the read-only verifier.
- `approval-risk` does not perform personal scoring. It scores task and path risk signals only.
- `agent-profile` stores observed failure modes from benchmark reports. It does not encode model preference.
- `cost` can return a non-zero status only when the local cost policy is enabled and the caller passes `--enforce`; it still does not change completion-card admission.

## Approval Risk

`policies/approval-risk.yaml` defines additive task/path signals, thresholds, and suggested approval counts. The default policy is disabled.

```bash
node packages/cli/dist/index.js approval-risk evaluate --card completion-card.yaml --json
node packages/cli/dist/index.js approval-risk check --card completion-card.yaml
```

The report validates against `schemas/approval-risk.schema.json` and always includes:

- `personal_scoring: false`
- `admission_authority: false`
- `policy_enabled`
- `risk_class`
- `required_approvals`

## Agent Profiles

Agent profiles summarize observed failure modes from benchmark output. The default profile path is `.x-harness/agent-profiles/<agent-id>.json`.

```bash
node packages/cli/dist/index.js agent-profile update --agent claude-code@x.y --from-benchmark benchmark-report.json
node packages/cli/dist/index.js agent-profile report --agent claude-code@x.y --json
node packages/cli/dist/index.js agent-profile report --profile .x-harness/agent-profiles/claude-code_x.y.json
```

The profile validates against `schemas/agent-profile.schema.json` and always includes `advisory_only: true` and `admission_authority: false`.

## Cost Awareness

`policies/cost-budget.yaml` defines local spend and token budgets. The default policy is disabled.

```bash
node packages/cli/dist/index.js cost check --actual-usd 1.25 --input-tokens 10000 --output-tokens 2000 --json
node packages/cli/dist/index.js cost report --from cost-report.json --json
```

When the policy is disabled, over-budget reports are informational. When the policy is enabled and the caller passes `--enforce`, `cost check` can exit non-zero so CI or a wrapper script can stop a run. This is an external budget control, not admission authority.

## Federation Relationship

Federation remains the only cross-repository optional exchange layer. It accepts anonymized, redacted patterns only and rejects raw source, raw logs, or raw completion-card content. Approval risk, agent profiles, and cost reports stay local unless an operator explicitly exports redacted summaries through a separate process.
