# SUBAGENT_TASK Template — `deep`

Use for high-stakes, multi-step, multi-source, high-drift, or high-cost-of-being-wrong work.

```md
## Task: <3-5 word description>

### Meta

- task_id: <id>
- priority: <low|medium|high|critical>
- agent_hint: <preferred specialist role>

### Goal

<one clear objective>

### Context

- repo: <project path>
- branch: <branch or n/a>
- refs: [<ref1>, <ref2>]
- background: [<fact1>, <fact2>]
- assumptions: [<a1>, <a2>]
- constraints: [<c1>, <c2>]

### Scope

- IN: [<area1>, <area2>]
- OUT: [<excluded1>]

### Inputs

- required: <named values>
- optional: <nice-to-have>

### Tools

- allowed: <tools>
- preferred: <preferred>
- disallowed: <tools not to use>

### Execution

- mode: <read_only|limited_edit|full_edit>
- max_files_changed: <N or n/a>
- stop_conditions: [<cond1>, <cond2>]
- failure_fallback: <what to do if blocked>

### Rollback Policy

- class: <none|soft|code_revert|state_restore>
- trigger: <when to rollback>
- owner: <agent responsible>
- validation: <post-rollback check>

### Success Criteria

- <criterion 1>
- <criterion 2>
- <criterion 3>

### Return (required schema)

result:
summary: <one-line outcome>
fix_status: <fixed|not_fixed|partial>
key_findings: []
decisions: []
recommendations: []
unsupported_or_unclear: []
evidence:
files_read: []
files_changed: []
commands_ran: []
sources_consulted: []
key_outputs: []
verification:
status: <passed|failed|skipped|blocked>
checks: []
confidence: <LOW|MED|HIGH>
handoff:
next_action: <next step> (owner: <agent|user>)
escalation_needed: <yes|no>
pgv_advice: null
```

### Evidence scope (required for deep)

Deep tasks must declare:

```yaml
state:
  read_set:
    - <files read>
  write_set:
    - <files changed>
  assumptions:
    - "<assumption 1>"
  conflict_policy:
    if_files_changed_after_claim: "rerun verify"
    if_tests_changed_after_claim: "rerun evidence"

evidence:
  verification_artifacts:
    - kind: unit_test
      command: npm test -- <feature>
      status: passed
      verifies:
        - "<what this check proves>"
      does_not_verify:
        - "<what this check does not prove>"
      confidence: medium
  untested_regions:
    - "<what was not tested>"
  remaining_risks:
    - "<what risks remain>"

governance:
  risk_class: high
  requires_human_approval: true
  approval_required_for:
    - "<high-risk category>"
  approval_status: pending
  approver: user
```
