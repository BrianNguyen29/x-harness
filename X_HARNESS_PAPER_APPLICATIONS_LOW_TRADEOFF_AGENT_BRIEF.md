# x-harness: Đề xuất áp dụng giá trị từ paper Code as Agent Harness với trade-off thấp

**Mục tiêu tài liệu:** Cung cấp một bản brief đầy đủ, chi tiết và có thể giao trực tiếp cho AI agent để triển khai các cải thiện giá trị nhất vào `x-harness`, dựa trên phân tích paper `Code as Agent Harness` và các nguyên tắc đã chốt của x-harness.

**Định vị giữ nguyên:** `x-harness` là một lightweight verify-gated completion/admission harness. Nó không phải agent runtime, không phải product planning harness, không phải workflow engine, không phải skill marketplace.

---

## 1. Tóm tắt chiến lược

Paper `Code as Agent Harness` nhấn mạnh code và file artifacts có thể trở thành substrate cho agent reasoning, execution feedback, state, verification và coordination. Với `x-harness`, không nên áp dụng các phần runtime nặng của paper. Nên áp dụng các cơ chế giúp completion claim:

```txt
harder to fake
easier to verify
safer to withhold
more auditable
less likely to create false confidence
```

Core của `x-harness` vẫn giữ:

```txt
one rule + one card + one verify command
```

Cụ thể:

```txt
One rule: completion is admitted, not claimed.
One card: completion-card.yaml.
One command: npx x-harness verify.
```

Các cải thiện trong tài liệu này phải giữ các ràng buộc:

```txt
- Không thêm daemon.
- Không thêm database.
- Không bắt buộc MCP.
- Không dùng LLM verifier làm authority mặc định.
- Không biến x-harness thành agent runtime.
- Không biến x-harness thành product planning harness.
- Không tăng token mặc định đáng kể.
- Không bắt task nhỏ dùng deep workflow.
```

---

## 2. Các áp dụng giá trị nhất, ít trade-off nhất

Các áp dụng được đề xuất theo thứ tự ưu tiên:

```txt
P0:
1. Evidence scope: verifies / does_not_verify / untested_regions / remaining_risks
2. verify --json để audit/replay
3. report --metrics để đo verification strength, recovery, state consistency
4. recovery_routing cho blocked/failed outcomes
5. authoritative artifact hierarchy

P1:
6. state.read_set / state.write_set nhẹ cho standard/deep
7. human approval fields cho deep tasks
8. doctor checks cho evidence scope, context policy, read-only verifier

P2:
9. harness change contract cho PRs thay đổi chính x-harness
10. richer replay archive
```

Không áp dụng vào core mặc định:

```txt
- automatic topology mutation
- self-evolving harness engine
- mandatory semantic verifier
- GUI/OS/browser automation harness
- runtime blackboard/database
- full multi-agent orchestration engine
```

---

## 3. P0.1 — Evidence scope

### 3.1 Vấn đề

Nếu `completion-card.yaml` chỉ ghi:

```yaml
commands_ran:
  - command: npm test
    status: passed
```

thì agent hoặc user có thể hiểu nhầm:

```txt
tests passed = task correct = safe to say done
```

Đây là false confidence. Test pass chỉ chứng minh một phạm vi cụ thể, không chứng minh toàn bộ correctness.

### 3.2 Mục tiêu

Nâng evidence từ pass/fail lên scoped evidence:

```txt
What did this check verify?
What did it not verify?
What remains untested?
What risks remain?
```

### 3.3 Schema đề xuất

Thêm vào `completion-card.yaml`:

```yaml
evidence:
  verification_artifacts:
    - kind: unit_test
      command: npm test -- product-form
      status: passed
      verifies:
        - "empty product name is rejected"
        - "invalid price blocks submit"
      does_not_verify:
        - "server-side validation"
        - "browser visual layout"
      confidence: medium

    - kind: typecheck
      command: npm run typecheck
      status: passed
      verifies:
        - "TypeScript type consistency"
      does_not_verify:
        - "runtime browser behavior"
      confidence: medium

  untested_regions:
    - "No E2E browser validation test was run."

  remaining_risks:
    - "Server-side validation may still be needed."
```

### 3.4 Allowed artifact kinds

Schema nên cho phép các `kind` sau:

```txt
typecheck
unit_test
integration_test
e2e_test
lint
build
static_analysis
security_scan
fuzz
performance_profile
manual_review
model_critique
custom
```

### 3.5 Tier behavior

Không ép mọi task viết scope dài. Áp dụng theo tier:

```txt
light:
  verification_artifacts optional
  verifies / does_not_verify optional
  untested_regions optional

standard:
  verification_artifacts recommended
  verifies recommended
  does_not_verify recommended when claim could be overread
  untested_regions recommended

deep:
  verification_artifacts required
  verifies required
  does_not_verify required
  untested_regions required
  remaining_risks required
```

### 3.6 CLI verify behavior

`npx x-harness verify` nên kiểm:

```txt
- verification_artifacts là array nếu tồn tại.
- kind hợp lệ.
- status hợp lệ: passed | failed | blocked | skipped | timeout | error.
- command phải có nếu kind là command-based check.
- verifies không được rỗng với tier deep.
- does_not_verify hoặc untested_regions phải có với tier deep.
- standard nếu thiếu scope thì warning, không nhất thiết fail.
- deep nếu thiếu scope thì blocked.
```

### 3.7 Acceptance impact

Policy không nên nói “test passed => success”. Policy nên nói:

```txt
success requires evidence_floor_met
evidence_floor_met depends on tier and claim scope
```

Với `standard`, thiếu scope có thể warning. Với `deep`, thiếu scope phải withheld/blocked.

### 3.8 File cần sửa

```txt
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
docs/VERIFY_GATE.md
docs/ADMISSION_POLICY.md
packages/cli/src/validators/completionCard.ts
packages/cli/src/core/admission.ts
examples/golden/success-light/
examples/golden/blocked-missing-evidence/
```

### 3.9 Trade-off

```txt
Value: very high
Token cost: low if optional by tier
Runtime cost: low
Complexity: low-medium
Risk: over-documentation if enforced too strongly
Mitigation: light optional, standard recommended, deep required
```

---

## 4. P0.2 — verify --json for audit/replay

### 4.1 Vấn đề

CLI output ngắn tốt cho token, nhưng CI/audit cần output có cấu trúc.

### 4.2 Mục tiêu

Thêm JSON output để reconstruct verify decision mà không tăng output mặc định.

Command:

```bash
npx x-harness verify --json
```

Optional archive:

```bash
npx x-harness verify --json > .x-harness/archive/CC-001.verify-report.json
```

### 4.3 JSON schema đề xuất

```json
{
  "card_id": "CC-001",
  "task_id": "PRODUCT-FORM-VALIDATION",
  "schema_version": 1,
  "input_card_hash": "sha256:...",
  "policy_hash": "sha256:...",
  "checks": [
    {
      "name": "owner_present",
      "status": "passed",
      "severity": "error"
    },
    {
      "name": "evidence_floor_met",
      "status": "passed",
      "severity": "error"
    },
    {
      "name": "evidence_scope_declared",
      "status": "warning",
      "severity": "warning",
      "note": "verification_artifacts missing does_not_verify for standard tier"
    }
  ],
  "decision": {
    "outcome": "success",
    "acceptance_status": "accepted"
  },
  "denominator_warning": "Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee."
}
```

### 4.4 Default output vẫn quiet

Default:

```txt
outcome: success
acceptance_status: accepted
checks: 8 passed, 0 failed, 1 warning, 0 blocked
```

Verbose:

```bash
npx x-harness verify --verbose
```

JSON:

```bash
npx x-harness verify --json
```

### 4.5 File cần sửa

```txt
packages/cli/src/commands/verify.ts
packages/cli/src/core/report.ts
schemas/verify-report.schema.json
docs/VERIFY_GATE.md
docs/CI.md
examples/golden/*/expected-verify-output.json
```

### 4.6 Trade-off

```txt
Value: high
Token cost: none by default
Runtime cost: very low
Complexity: low
Risk: none significant
```

---

## 5. P0.3 — report --metrics

### 5.1 Vấn đề

Chỉ biết success/failed chưa đủ để đánh giá harness có tốt không. Cần đo verification strength, recovery quality và state consistency.

### 5.2 Mục tiêu

Thêm:

```bash
npx x-harness report --metrics
```

### 5.3 Metrics đề xuất

```yaml
metrics:
  verification_strength:
    command_evidence_count: 2
    oracle_kinds:
      - unit_test
      - typecheck
    untested_regions_count: 1
    remaining_risks_count: 1

  state_consistency:
    owner_present: true
    accountable_present: true
    files_changed_present: true
    admission_mapping_valid: true

  recovery_ability:
    blocked_has_next_action: true
    blocked_has_owner: true
    recovery_route_present: true

  replayability:
    completion_card_present: true
    input_card_hash_present: true
    policy_hash_present: true

  cost:
    default_context_class: low
    verify_runtime_ms: 842
```

### 5.4 Metric classes

Nên dùng class đơn giản thay vì scoring phức tạp:

```txt
low | medium | high
weak | adequate | strong
present | missing
```

Không tạo benchmark claim.

### 5.5 Report phải có denominator warning

Bắt buộc include:

```txt
Verify-event success must not be interpreted as task-level success,
production reliability, benchmark success, or safety guarantee.
```

### 5.6 File cần sửa

```txt
packages/cli/src/commands/report.ts
packages/cli/src/core/report.ts
docs/METRICS.md
docs/DENOMINATOR_POLICY.md
schemas/verify-report.schema.json
examples/golden/*/expected-report.md
```

### 5.7 Trade-off

```txt
Value: high
Token cost: none unless user asks report
Runtime cost: low
Complexity: low-medium
Risk: metric overclaim
Mitigation: class-based metrics + denominator warning
```

---

## 6. P0.4 — Recovery routing

### 6.1 Vấn đề

Blocked/failed outcomes thường thiếu route hành động. Agent có thể trả lời mơ hồ:

```txt
Verification failed. Please check.
```

Cần chuyển thành next action có owner.

### 6.2 Mục tiêu

Thêm policy routing cho failure/blocker types.

### 6.3 Policy đề xuất

`policies/recovery.yaml` hoặc section trong `policies/admission.yaml`:

```yaml
recovery_routing:
  evidence_missing:
    next_action: "Attach validation evidence or explain why unavailable."
    owner: implementation-worker

  evidence_scope_missing:
    next_action: "Declare what each validation artifact verifies and does not verify."
    owner: implementation-worker

  typecheck_failed:
    next_action: "Return to implementation-worker for type repair."
    owner: implementation-worker

  test_failed:
    next_action: "Diagnose failing behavior and update implementation or tests."
    owner: implementation-worker

  lint_failed:
    next_action: "Fix lint issues or justify why the lint rule is not applicable."
    owner: implementation-worker

  build_failed:
    next_action: "Fix build failure before requesting admission."
    owner: implementation-worker

  approval_missing:
    next_action: "Request human approval before admission."
    owner: user

  conflicting_scope:
    next_action: "Ask user to clarify task scope."
    owner: user

  verifier_not_read_only:
    next_action: "Rerun verification with a read-only verifier."
    owner: admission-verifier
```

### 6.4 Completion card blocked example

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: evidence_scope_declared
  reason: "Validation artifacts do not declare what they verify."

handoff:
  next_action: "Declare verifies/does_not_verify for each validation artifact and rerun verification."
  owner: implementation-worker
```

### 6.5 CLI behavior

`verify` should:

```txt
- Detect blocking predicate.
- Map predicate to recovery route if available.
- Populate suggested next_action in JSON/report output.
- Not mutate completion-card.yaml by default.
```

Optional future command:

```bash
npx x-harness recover --suggest
```

Not required for P0.

### 6.6 File cần sửa

```txt
policies/admission.yaml
policies/recovery.yaml
packages/cli/src/core/admission.ts
packages/cli/src/core/recovery.ts
packages/cli/src/commands/verify.ts
docs/VERIFY_GATE.md
docs/RECOVERY.md
adapters/claude-code/skills/x-harness-recover/SKILL.md
adapters/antigravity/workflows/x-harness-recover.md
examples/golden/blocked-missing-evidence/
```

### 6.7 Trade-off

```txt
Value: high
Token cost: low
Runtime cost: low
Complexity: low-medium
Risk: policy bloat if too many routes
Mitigation: start with 8–10 route types only
```

---

## 7. P0.5 — Authoritative artifact hierarchy

### 7.1 Vấn đề

Trong multi-agent hoặc long-running sessions, chat summary, agent memory và repo files có thể mâu thuẫn. Cần định nghĩa artifact nào là authoritative.

### 7.2 Mục tiêu

Thêm vào `docs/RUNTIME_CONTRACT.md`:

```txt
Authoritative artifacts:

1. Source files and git diff are authoritative for implementation state.
2. completion-card.yaml is authoritative for completion claim state.
3. policies/admission.yaml is authoritative for admission policy.
4. npx x-harness verify output is authoritative for accepted/withheld mapping.
5. Chat summaries are non-authoritative.
```

### 7.3 Adapter rule

Adapters phải nhắc:

```txt
If chat says done but completion-card.yaml says withheld, treat completion as withheld.
If completion-card.yaml claims accepted but verify output disagrees, verify output wins.
```

### 7.4 File cần sửa

```txt
docs/RUNTIME_CONTRACT.md
AGENTS.md
X_HARNESS.md
adapters/generic/AGENTS.md
adapters/claude-code/CLAUDE.md
adapters/antigravity/rules/x-harness.md
adapters/cursor/rules/x-harness.mdc
adapters/opencode/agents/x-harness-verify.md
```

### 7.5 Trade-off

```txt
Value: high for long-running and multi-agent tasks
Token cost: very low
Runtime cost: none
Complexity: very low
Risk: none significant
```

---

## 8. P1.1 — state.read_set / state.write_set

### 8.1 Vấn đề

Trong project lớn, nhiều agent hoặc nhiều session có thể sửa cùng vùng code. Cần khai báo nhẹ agent đã đọc gì, viết gì, assumptions gì.

Không nên tạo transaction runtime. Chỉ cần artifact nhẹ.

### 8.2 Schema đề xuất

```yaml
state:
  read_set:
    - components/product-form.tsx
    - lib/products.ts
  write_set:
    - components/product-form.tsx
    - components/product-form.test.tsx
  assumptions:
    - "createProduct API contract remains unchanged."
  conflict_policy:
    if_files_changed_after_claim: "rerun verify"
    if_tests_changed_after_claim: "rerun evidence"
```

### 8.3 Tier behavior

```txt
light:
  state block optional

standard:
  state block recommended for multi-file changes

deep:
  state block required
```

### 8.4 Verify behavior

`verify` should:

```txt
- Warn if files_changed not included in write_set when write_set exists.
- Warn if deep tier lacks read_set/write_set.
- Block deep success if write_set missing.
- Optionally warn if git diff changed after card timestamp.
```

Do not implement complex git transaction logic in v0.1.

### 8.5 File cần sửa

```txt
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
packages/cli/src/validators/completionCard.ts
docs/RUNTIME_CONTRACT.md
docs/VERIFY_GATE.md
examples/golden/multi-agent-success/
```

### 8.6 Trade-off

```txt
Value: high for large projects
Token cost: low
Runtime cost: low
Complexity: medium
Risk: too much ceremony for small tasks
Mitigation: optional for light, recommended standard, required deep
```

---

## 9. P1.2 — Human approval fields for deep tasks

### 9.1 Vấn đề

High-risk actions như auth, payment, database migration, deployment, credentials hoặc production data không nên được admitted chỉ vì tests pass.

### 9.2 Schema đề xuất

```yaml
governance:
  risk_class: high
  requires_human_approval: true
  approval_required_for:
    - "database migration"
    - "auth logic change"
    - "production deploy"
  approval_status: pending
  approver: user
```

Allowed approval status:

```txt
not_required
pending
approved
rejected
```

### 9.3 Admission rule

```yaml
reject_success_if:
  approval_required_but_missing: true
```

Nếu approval pending:

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: approval_required
  reason: "Deep-risk task requires human approval before admission."
```

### 9.4 Scope

Chỉ áp dụng cho `deep` hoặc risk-triggered tasks:

```txt
auth
payment
database
deploy
security
production data
credentials
permission model
irreversible state change
```

Không áp dụng cho `light` hoặc `standard` mặc định.

### 9.5 File cần sửa

```txt
schemas/completion-card.schema.json
templates/SUBAGENT_TASK_deep.md
templates/COMPLETION_CARD.md
policies/admission.yaml
docs/DEEP_MODE.md
docs/ADMISSION_POLICY.md
packages/cli/src/core/admission.ts
examples/04-blocked-verification/
```

### 9.6 Trade-off

```txt
Value: high for high-risk tasks
Token cost: none for light/standard
Runtime cost: low
Complexity: low-medium
Risk: overblocking if applied too broadly
Mitigation: deep only
```

---

## 10. P1.3 — Doctor checks for context and evidence policy

### 10.1 Vấn đề

Repo có thể drift: `AGENTS.md` dài, skills bị nhồi, verifier có quyền edit, evidence scope bị bỏ qua, PGV thành authority.

### 10.2 Doctor checks đề xuất

`npx x-harness doctor` nên kiểm:

```txt
Context policy:
- AGENTS.md <= configured max lines.
- AGENTS.md links to docs/templates instead of duplicating them.
- examples are not required context.
- skills are separate and load-on-demand.

Evidence policy:
- completion-card schema supports verification_artifacts.
- standard/deep templates include evidence scope.
- deep requires untested_regions or remaining_risks.

Verification policy:
- verifier instructions do not allow edits.
- adapters preserve success/withheld mapping.
- PGV is advisory-only.

Tier policy:
- only light/standard/deep are used.
- no small/medium/large aliases.

Runtime policy:
- no daemon, DB, mandatory MCP in core docs.
- no Python in core tooling.
```

### 10.3 File cần sửa

```txt
packages/cli/src/commands/doctor.ts
packages/cli/src/core/doctor.ts
docs/CONTEXT_POLICY.md
docs/PRINCIPLES.md
examples/golden/
```

### 10.4 Trade-off

```txt
Value: high
Token cost: none
Runtime cost: low
Complexity: medium
Risk: doctor false positives
Mitigation: warnings before hard failures for docs style checks
```

---

## 11. P2.1 — Harness change contract

### 11.1 Vấn đề

x-harness itself can drift. A change intended to improve safety can accidentally weaken read-only verification, increase default tokens, or overclaim success.

### 11.2 Template đề xuất

`templates/HARNESS_CHANGE_CONTRACT.md`:

```md
# Harness Change Contract

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
```

### 11.3 Apply in CONTRIBUTING.md

Mọi PR thay đổi admission policy, schemas, templates, CLI verify, adapters, skills phải include change contract.

### 11.4 Trade-off

```txt
Value: medium-high for project maintainability
Token cost: none for users
Runtime cost: none
Complexity: low
Risk: contributor friction
Mitigation: require only for harness-sensitive changes
```

---

## 12. Updated completion-card.yaml proposal

Phiên bản standard sau khi áp dụng P0/P1:

```yaml
id: CC-001
task_id: PRODUCT-FORM-VALIDATION
tier: standard

owner: implementation-worker
accountable: user

claim:
  fix_status: fixed
  summary: "Added product form validation."

state:
  read_set:
    - components/product-form.tsx
    - lib/products.ts
  write_set:
    - components/product-form.tsx
    - components/product-form.test.tsx
  assumptions:
    - "createProduct API contract remains unchanged."
  conflict_policy:
    if_files_changed_after_claim: "rerun verify"
    if_tests_changed_after_claim: "rerun evidence"

evidence:
  files_changed:
    - components/product-form.tsx
    - components/product-form.test.tsx

  verification_artifacts:
    - kind: unit_test
      command: npm test -- product-form
      status: passed
      verifies:
        - "empty product name is rejected"
        - "invalid price blocks submit"
      does_not_verify:
        - "server-side validation"
        - "browser visual layout"
      confidence: medium

    - kind: typecheck
      command: npm run typecheck
      status: passed
      verifies:
        - "TypeScript type consistency"
      does_not_verify:
        - "runtime browser behavior"
      confidence: medium

  untested_regions:
    - "No E2E browser validation test was run."

  remaining_risks:
    - "Server-side validation may still be needed."

verification:
  status: passed
  checks:
    - name: evidence_present
      status: passed
    - name: evidence_scope_declared
      status: passed
    - name: evidence_floor_met
      status: passed

admission:
  outcome: pending
  acceptance_status: withheld

handoff:
  next_action: "Run read-only x-harness verification."
  owner: admission-verifier

pgv_advice: null
```

Deep extension:

```yaml
execution_controls:
  mode: limited_edit
  max_files_changed: 8
  stop_conditions:
    - "Unexpected payment provider behavior."
    - "Auth regression test fails."
    - "Database migration required."
  failure_fallback: "Stop and return withheld completion with recovery owner."

rollback_policy:
  class: code_revert
  trigger: "Failed smoke test, auth regression, or user rejection."
  owner: implementation-worker
  validation: "Re-run relevant tests and typecheck."

governance:
  risk_class: high
  requires_human_approval: true
  approval_required_for:
    - "auth logic change"
  approval_status: pending
  approver: user
```

---

## 13. Updated admission policy proposal

```yaml
version: 1

candidate_completion:
  required:
    - claim.fix_status: fixed
    - verification.status: passed

success_requires:
  - owner_present
  - accountable_present
  - evidence_present
  - evidence_floor_met
  - admission_mapping_valid
  - no_unresolved_blocker
  - no_active_recovery
  - verifier_read_only

evidence_floor:
  light:
    required:
      - files_changed
    one_of:
      - command_evidence
      - manual_rationale

  standard:
    required:
      - files_changed
      - command_evidence
    recommended:
      - evidence_scope_declared
      - untested_regions_declared

  deep:
    required:
      - files_changed
      - command_evidence
      - evidence_scope_declared
      - untested_regions_declared
      - remaining_risks_declared
      - execution_controls_present
      - rollback_policy_present

reject_success_if:
  fix_status:
    - partial
    - not_fixed

  verification_status:
    - failed
    - skipped
    - blocked

  evidence_quality:
    - missing
    - weak

  approval_required_but_missing: true
  timeout: true
  error: true

outcome_mapping:
  success:
    acceptance_status: accepted
  failed:
    acceptance_status: withheld
  blocked:
    acceptance_status: withheld
  skipped:
    acceptance_status: withheld
  timeout:
    acceptance_status: withheld
  error:
    acceptance_status: withheld
```

---

## 14. Implementation tasks for AI agent

### Task 1 — Extend completion-card schema

Implement:

```txt
evidence.verification_artifacts[]
evidence.untested_regions[]
evidence.remaining_risks[]
state.read_set[]
state.write_set[]
state.assumptions[]
state.conflict_policy
governance fields for deep
```

Acceptance:

```txt
- Existing minimal cards still validate.
- New standard/deep cards validate.
- Invalid artifact kind fails schema.
- Invalid status fails schema.
```

### Task 2 — Update templates

Update:

```txt
templates/COMPLETION_CARD.md
templates/SUBAGENT_TASK_light.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
```

Acceptance:

```txt
- Light remains short.
- Standard includes evidence scope as recommended.
- Deep includes evidence scope, state, rollback, governance.
```

### Task 3 — Update admission policy and validator

Implement evidence floor logic.

Acceptance:

```txt
- Light card can pass without scope if evidence floor met.
- Standard card missing scope gets warning, not hard fail.
- Deep card missing scope is blocked.
- approval_required_but_missing blocks deep success.
```

### Task 4 — Add verify --json

Implement structured JSON output.

Acceptance:

```txt
- Default output remains quiet.
- --json outputs valid JSON.
- JSON includes card_id, task_id, checks, decision, warning.
- CI can consume output.
```

### Task 5 — Add report --metrics

Implement metrics report.

Acceptance:

```txt
- report --metrics includes verification_strength.
- includes state_consistency.
- includes recovery_ability.
- includes denominator warning.
```

### Task 6 — Add recovery routing

Implement recovery route mapping.

Acceptance:

```txt
- evidence_missing maps to implementation-worker.
- typecheck_failed maps to implementation-worker.
- approval_missing maps to user.
- blocked output includes suggested next action.
```

### Task 7 — Add authoritative artifact docs

Update docs and adapters.

Acceptance:

```txt
- docs/RUNTIME_CONTRACT.md includes hierarchy.
- AGENTS.md references it briefly.
- Claude/Antigravity/Cursor/OpenCode adapters follow same rule.
```

### Task 8 — Update doctor

Add checks for:

```txt
AGENTS.md length
read-only verifier
PGV advisory-only
canonical tiers
no runtime bloat
evidence scope templates
deep not default
```

Acceptance:

```txt
- doctor passes on golden examples.
- doctor warns, not fails, for standard missing optional scope.
- doctor fails if verifier can edit.
```

### Task 9 — Add golden examples

Add examples:

```txt
examples/golden/success-standard-scoped-evidence/
examples/golden/blocked-missing-evidence-scope/
examples/golden/deep-approval-required/
examples/golden/failed-typecheck-recovery-route/
```

Acceptance:

```txt
npx x-harness examples verify
```

passes.

---

## 15. Files to create or modify

```txt
docs/
  METRICS.md
  CONTEXT_POLICY.md
  RECOVERY.md
  RUNTIME_CONTRACT.md
  VERIFY_GATE.md
  ADMISSION_POLICY.md
  DENOMINATOR_POLICY.md

templates/
  COMPLETION_CARD.md
  SUBAGENT_TASK_light.md
  SUBAGENT_TASK_standard.md
  SUBAGENT_TASK_deep.md
  HARNESS_CHANGE_CONTRACT.md

schemas/
  completion-card.schema.json
  verify-report.schema.json

policies/
  admission.yaml
  recovery.yaml

packages/cli/src/
  commands/
    verify.ts
    report.ts
    doctor.ts
  core/
    admission.ts
    recovery.ts
    report.ts
    metrics.ts
    hash.ts
  validators/
    completionCard.ts

adapters/
  generic/AGENTS.md
  claude-code/CLAUDE.md
  claude-code/skills/x-harness-verify/SKILL.md
  claude-code/skills/x-harness-recover/SKILL.md
  antigravity/rules/x-harness.md
  antigravity/workflows/x-harness-verify.md
  antigravity/workflows/x-harness-recover.md
  cursor/rules/x-harness.mdc
  opencode/agents/x-harness-verify.md
  opencode/agents/x-harness-recover.md

examples/golden/
  success-standard-scoped-evidence/
  blocked-missing-evidence-scope/
  deep-approval-required/
  failed-typecheck-recovery-route/
```

---

## 16. Guardrails for implementation agent

Do not implement:

```txt
- agent runtime
- database
- mandatory MCP server
- semantic LLM verifier as admission authority
- automatic topology mutation
- browser automation harness
- product planning lifecycle by default
```

Preserve:

```txt
- minimal mode remains minimal
- light tier remains lightweight
- verify is read-only
- success is the only accepted outcome
- failed/blocked/skipped/timeout/error are withheld
- PGV is advisory-only
- deep is opt-in
- skills are load-on-demand
```

---

## 17. Final acceptance criteria

The implementation is acceptable if:

```txt
1. Existing minimal x-harness flow still works.
2. Existing simple completion-card.yaml remains valid.
3. New scoped evidence cards validate.
4. verify --json works.
5. report --metrics works.
6. recovery routing suggests owner + next action.
7. deep tasks can require human approval.
8. standard tasks are not overblocked by missing optional scope.
9. light tasks remain low ceremony.
10. doctor detects read-only/PGV/tier/context violations.
11. golden examples pass.
12. No runtime/server/database/MCP dependency is introduced.
```

---

## 18. Strategic summary

The most valuable adoption from the paper is not adding a bigger harness runtime. It is improving the quality of the admission boundary.

The key shift:

```txt
from:
  checks passed

to:
  these checks passed,
  they verify this scope,
  they do not verify that scope,
  remaining risks are declared,
  and the claim is admissible under the current policy
```

This keeps `x-harness` lightweight while making completion claims substantially more auditable and less misleading.
