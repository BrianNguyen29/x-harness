# Completion Card Template

Copy this template, fill in the fields, and save as `completion-card.yaml`.

## Accepted example — light

```yaml
schema_version: "1"
task_id: "TASK-001"
tier: "light"
owner: "agent-name"
accountable: "user-name"
claim:
  summary: "Fixed the login redirect bug"
  fix_status: "fixed"
  evidence:
    - type: "command"
      value: "npm test -- login"
    - type: "file"
      value: "src/auth/redirect.ts"
evidence:
  files_changed:
    - src/auth/redirect.ts
  command_evidence:
    - command: npm test -- login
      exit_code: 0
verification:
  status: "passed"
  checks:
    - name: "unit_tests"
      status: "passed"
      note: "all auth tests pass"
    - name: "typecheck"
      status: "passed"
      note: "tsc --noEmit clean"
admission:
  outcome: "success"
acceptance_status: "accepted"
handoff:
  next_action: "none"
  owner: "agent-name"
```

## Accepted example — standard with scoped evidence

```yaml
schema_version: "1"
task_id: "TASK-002"
tier: "standard"
owner: "agent-name"
accountable: "user-name"
state:
  read_set:
    - src/auth/redirect.ts
  write_set:
    - src/auth/redirect.ts
    - src/auth/redirect.test.ts
evidence:
  files_changed:
    - src/auth/redirect.ts
    - src/auth/redirect.test.ts
  command_evidence:
    - command: npm test -- redirect
      exit_code: 0
    - command: npm run typecheck
      exit_code: 0
  verification_artifacts:
    - kind: unit_test
      command: npm test -- redirect
      status: passed
      verifies:
        - "redirect preserves query params"
      does_not_verify:
        - "browser visual layout"
      confidence: medium
    - kind: typecheck
      command: npm run typecheck
      status: passed
      verifies:
        - "TypeScript type consistency"
      does_not_verify:
        - "runtime behavior"
      confidence: medium
  untested_regions:
    - "No E2E browser test was run."
claim:
  summary: "Fixed the login redirect bug with scoped evidence"
  fix_status: "fixed"
  evidence:
    - type: "command"
      value: "npm test -- redirect"
    - type: "file"
      value: "src/auth/redirect.ts"
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: "Login redirect preserves query params"
  expected_effect: "Redirect unit tests and typecheck pass"
  measurable_signal: "npm test -- redirect && npm run typecheck"
  falsification_method: "Run the listed commands and inspect failing assertions"
  horizon: same_verify
  confidence: medium
  verdict:
    status: pending
verification:
  status: "passed"
  checks:
    - name: "unit_tests"
      status: "passed"
    - name: "typecheck"
      status: "passed"
admission:
  outcome: "success"
acceptance_status: "accepted"
handoff:
  next_action: "none"
  owner: "agent-name"
```

## Blocked example

```yaml
schema_version: "1"
task_id: "TASK-003"
tier: "light"
owner: "agent-name"
accountable: "user-name"
claim:
  summary: "Partial fix for payment webhook; missing integration test environment"
  fix_status: "partial"
  evidence:
    - type: "note"
      value: "unit tests pass locally; no staging env to run integration tests"
evidence:
  files_changed:
    - src/payments/webhook.ts
  manual_rationale: "Unit tests passed locally; staging integration environment unavailable."
verification:
  status: "blocked"
  checks:
    - name: "integration_tests"
      status: "blocked"
      note: "staging environment unavailable"
admission:
  outcome: "blocked"
acceptance_status: "withheld"
handoff:
  next_action: "Fix missing evidence"
  owner: "agent-name"
```

## Deep example with governance

```yaml
schema_version: "1"
task_id: "TASK-004"
tier: "deep"
owner: "agent-name"
accountable: "user-name"
state:
  read_set:
    - src/auth/oauth.ts
  write_set:
    - src/auth/oauth.ts
    - src/auth/oauth.test.ts
evidence:
  files_changed:
    - src/auth/oauth.ts
    - src/auth/oauth.test.ts
  command_evidence:
    - command: npm test -- oauth
      exit_code: 0
    - command: npm run typecheck
      exit_code: 0
  verification_artifacts:
    - kind: unit_test
      command: npm test -- oauth
      status: passed
      verifies:
        - "PKCE flow completes"
      does_not_verify:
        - "production IdP behavior"
      confidence: medium
  untested_regions:
    - "No production IdP integration test was run."
  remaining_risks:
    - "Production IdP rate limits may differ."
  rollback_policy:
    - "Revert src/auth/oauth.ts and src/auth/oauth.test.ts to previous commits; re-run tests to confirm"
  execution_controls:
    - "Review OAuth token storage before merging"
    - "Require security audit sign-off before production deployment"
governance:
  risk_class: high
  requires_human_approval: true
  approval_required_for:
    - "auth logic change"
  approval_status: approved
  approver: user
claim:
  summary: "Implemented OAuth2 PKCE flow"
  fix_status: "fixed"
  evidence:
    - type: "command"
      value: "npm test -- oauth"
    - type: "file"
      value: "src/auth/oauth.ts"
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: "OAuth2 PKCE flow is implemented"
  expected_effect: "OAuth unit tests and typecheck pass"
  measurable_signal: "npm test -- oauth && npm run typecheck"
  falsification_method: "Run the listed commands and inspect failing assertions"
  horizon: same_verify
  confidence: medium
  verdict:
    status: pending
verification:
  status: "passed"
  checks:
    - name: "unit_tests"
      status: "passed"
    - name: "typecheck"
      status: "passed"
admission:
  outcome: "success"
acceptance_status: "accepted"
handoff:
  next_action: "none"
  owner: "agent-name"
```

## Rules

- `claim.fix_status: fixed` is only a completion candidate; accepted success also requires `verification.status: passed`, `admission.outcome: success`, and `acceptance_status: accepted`.
- `acceptance_status: accepted` is only valid when `admission.outcome: success`.
- `blocked`, `failed`, `skipped` outcomes must include `handoff.next_action` and `handoff.owner`.
- PGV advice is advisory-only and never overrides admission.
- The verifier is read-only; it does not edit files to fix findings.
- For `standard` tier, `done_checklist` and `prediction` are required; `verification_artifacts`, `verifies`, `does_not_verify`, and `untested_regions` are recommended.
- For `deep` tier, `done_checklist`, `prediction`, evidence scope fields, and `state.read_set/write_set` are required.
- For `deep` tier with `governance.requires_human_approval: true`, `approval_status` must be `approved`.

<!-- BEGIN X-HARNESS MANAGED CONTRACT: completion-card-template-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: c101e078ad9bdfb5 -->

## Generated Handoff Contract

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

Required completion card fields:

- schema_version
- task_id
- tier
- owner
- accountable
- claim
- verification
- admission
- acceptance_status
- handoff

## Evidence Floor

- **light**: files_changed + (command_evidence or manual_rationale).
- **standard**: files_changed + command_evidence + done_checklist + prediction.
- **deep**: files_changed + command_evidence + evidence_scope_declared + untested_regions_declared + remaining_risks_declared + execution_controls_present + rollback_policy_present + done_checklist + prediction. Runtime-enforced: verification_artifacts, state.read_set, state.write_set.

## Strict Evidence Provenance

- verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.
- verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.

Completion cards use claim.fix_status as the canonical fix-status field. Subagent returns may use result.fix_status only in compatibility return payloads.

<!-- END X-HARNESS MANAGED CONTRACT: completion-card-template-contract -->
