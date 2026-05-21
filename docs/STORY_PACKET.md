# Story Packet

```yaml
id: STORY-001
feature_intake: FI-001
title: <story title>
tier: <light|standard|deep>
owner: <agent|user>
accountable: <agent|user>
goal: <one objective>
context:
  product_contract_refs: []
  architecture_refs: []
  prior_decisions: []
scope:
  in: []
  out: []
acceptance_criteria:
  - id: AC-1
    text: <criterion>
    verification: <test/check/manual review>
handoff:
  template: <SUBAGENT_TASK_light|standard|deep>
  task_type: <review|research|implementation|mixed>
claim_requirements:
  requires_completion_card: true
  requires_claim_packet: false
  requires_evidence_packet: false
validation:
  commands: []
  expected_evidence: []
blocked_policy:
  require_next_owner: true
  require_next_action: true
```
