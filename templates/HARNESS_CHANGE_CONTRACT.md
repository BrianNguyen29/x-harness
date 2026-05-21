# Harness Change Contract

Use this template for PRs that modify admission policy, schemas, templates, CLI verify, adapters, or skills.

## Component modified

- <cli|policy|schema|template|adapter|docs>

## Target failure mode

- <premature_done|weak_evidence|blocked_without_owner|token_bloat|adapter_drift>

## Predicted improvement

- <what should improve>

## Must preserve

- Verification is read-only.
- success is the only accepted outcome.
- failed/blocked/skipped/timeout/error are withheld.
- PGV is advisory-only.
- minimal mode remains lightweight.
- deep remains opt-in.

## Falsifying evaluation

- <test/example/doctor check that would prove this change harmful>

## Rollback plan

- <how to revert>

## Cost impact

default_token_impact: <none|low|medium|high>
runtime_impact: <none|low|medium|high>
