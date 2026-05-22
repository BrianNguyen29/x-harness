# SUBAGENT_TASK Template — `standard`

Use for bounded multi-step research, review, synthesis, or implementation.

```md
## Task: <3-5 word description>

### Goal

<one clear objective>

### Context

- repo: <project path>
- refs: [<ref1>, <ref2>]
- constraints: [<c1>, <c2>]

### Scope

- IN: [<area1>, <area2>]
- OUT: [<excluded1>]

### Inputs

- required: <named values>
- optional: <nice-to-have values>

### Tools

- allowed: <tools>
- preferred: <preferred tool>

### Verification

- <bounded check 1>
- <bounded check 2>

### Success Criteria

- <criterion 1>
- <criterion 2>
- <criterion 3>

### Return (required schema)

result:
summary: <one-line outcome>
fix_status: <fixed|not_fixed|partial>
key_findings: []
recommendations: []
unsupported_or_unclear: []
evidence:
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
pgv_advice: null
```

### Evidence scope (recommended for standard)

When providing evidence, include scoped verification:

```yaml
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
```
