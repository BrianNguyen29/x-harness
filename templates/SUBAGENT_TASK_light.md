# SUBAGENT_TASK Template — `light`

Use for narrow, low-ceremony work.

```md
## Task: <3-5 word description>

### Goal
<one clear objective>

### Context
- repo: <project path or name>
- refs: <file:line or prior note, if any>
- constraints: <key constraints, if any>

### Scope
- IN: <what to do>
- OUT: <what not to do>

### Success Criteria
- <criterion 1>
- <criterion 2>

### Return (required schema)
result:
  summary: <one-line outcome>
  fix_status: <fixed|not_fixed|partial>
  key_findings: []
evidence:
  files_changed: []
  commands_ran: []
  key_outputs: []
verification:
  status: <passed|failed|skipped|blocked>
confidence: <LOW|MED|HIGH>
handoff:
  next_action: <next step> (owner: <agent|user>)
pgv_advice: null
```
