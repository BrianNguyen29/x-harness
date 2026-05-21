# Completion Card Template

Copy this template, fill in the fields, and save as `completion-card.yaml`.

## Accepted example — light

```yaml
schema_version: "0.1"
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
schema_version: "0.1"
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
schema_version: "0.1"
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
schema_version: "0.1"
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

- Do not claim `fix_status: fixed` unless `verification.status: passed`.
- `acceptance_status: accepted` is only valid when `admission.outcome: success`.
- `blocked`, `failed`, `skipped` outcomes must include `handoff.next_action` and `handoff.owner`.
- PGV advice is advisory-only and never overrides admission.
- The verifier is read-only; it does not edit files to fix findings.
- For `standard` tier, `verification_artifacts`, `verifies`, `does_not_verify`, and `untested_regions` are recommended.
- For `deep` tier, evidence scope fields and `state.read_set/write_set` are required.
- For `deep` tier with `governance.requires_human_approval: true`, `approval_status` must be `approved`.
