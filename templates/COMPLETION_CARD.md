# Completion Card Template

Copy this template, fill in the fields, and save as `completion-card.yaml`.

## Accepted example

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

## Blocked example

```yaml
schema_version: "0.1"
task_id: "TASK-002"
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

## Rules

- Do not claim `fix_status: fixed` unless `verification.status: passed`.
- `acceptance_status: accepted` is only valid when `admission.outcome: success`.
- `blocked`, `failed`, `skipped` outcomes must include `handoff.next_action` and `handoff.owner`.
- PGV advice is advisory-only and never overrides admission.
- The verifier is read-only; it does not edit files to fix findings.
