# Feature Intake

Feature intake converts human intent into scoped, agent-ready work.

```yaml
id: FI-001
title: <feature or change>
request_type: <new_feature|change_request|bugfix|research|maintenance|harness_improvement>
risk_lane: <light|standard|deep>
summary: <short summary>
problem:
  user_need: <need>
  current_gap: <gap>
scope:
  in: []
  out: []
affected_surfaces:
  product_docs: []
  architecture: []
  tests: []
  stories: []
  code_areas: []
risk_checklist:
  auth: false
  authorization: false
  data_model: false
  migration: false
  security: false
  external_provider: false
  public_contract: false
  cross_platform: false
  weak_validation: false
routing:
  recommended_tier: <light|standard|deep>
  reason: <why>
```

Route to `deep` for auth, security, data loss, migration, public contract changes, external provider behavior, release-critical work, or weak validation.
