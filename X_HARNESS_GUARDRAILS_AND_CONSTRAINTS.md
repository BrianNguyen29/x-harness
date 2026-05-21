# x-harness: Guardrails và Constraints chống Overengineering

**Mục tiêu tài liệu:** Định nghĩa bộ guardrails và constraints cho `x-harness` để giữ repo nhẹ, ít tốn token, không bị phình thành product planning harness, agent runtime, workflow engine, hoặc skill marketplace.  
**Nguyên tắc nền:** `x-harness` là một lightweight verify-gated completion/admission harness.

---

## 1. Tóm tắt nguyên tắc cốt lõi

`x-harness` phải luôn giữ core tối giản:

```txt
one rule + one card + one verify command
```

Cụ thể:

```txt
One rule: completion is admitted, not claimed.
One card: completion-card.yaml.
One command: npx x-harness verify.
```

`x-harness` không cố thay thế AI tools, không cố làm agent runtime, không cố làm product planning framework. Nó chỉ thêm một lớp admission nhẹ để kiểm soát khi nào AI agent được phép nói task đã xong.

Flow chuẩn:

```txt
agent work
  -> completion claim
  -> structured evidence
  -> read-only verification
  -> accepted or withheld
```

Accepted chỉ xảy ra khi:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

Mọi outcome khác đều là withheld.

---

## 2. Non-negotiable constraints

Các constraint này không được phá vỡ trong bất kỳ PR, feature, adapter, skill hoặc roadmap item nào.

```txt
1. Core stays minimal.
2. Verification is read-only.
3. Completion is admitted, not claimed.
4. success is the only accepted outcome.
5. failed/blocked/skipped/timeout/error are withheld.
6. PGV is advisory-only.
7. Deep mode is opt-in.
8. Skills are load-on-demand.
9. No daemon, DB, mandatory MCP, or mandatory LLM verifier.
10. Do not turn x-harness into a product planning harness.
```

Câu hỏi kiểm mọi quyết định thiết kế:

```txt
Does this make completion harder to fake, easier to verify, and safer to withhold without adding default process weight?
```

Nếu câu trả lời là không, feature không nên vào core.

---

## 3. Core scope guardrails

### G1 — Core phải luôn tối giản

Core của `x-harness` chỉ gồm:

```txt
AGENTS.md
X_HARNESS.md
completion-card.yaml
templates/COMPLETION_CARD.md
policies/admission.yaml
npx x-harness verify
```

Không biến core thành full workflow framework.

Không để minimal mode tạo quá nhiều file.

Default path phải ngắn:

```txt
Implement task.
Create completion-card.yaml.
Run npx x-harness verify.
Report accepted or withheld.
```

---

### G2 — x-harness không phải product planning harness

Không đưa các thành phần sau vào default:

```txt
feature intake
story packet
test matrix lifecycle
architecture decision system
roadmap/backlog management
product contract lifecycle
```

Những thành phần này chỉ được tồn tại dưới dạng optional module nếu thật sự cần.

Allowed:

```bash
npx x-harness add readiness-layer
```

Not allowed by default:

```txt
npx x-harness init --minimal
  -> creates feature intake
  -> creates story packet
  -> creates test matrix
  -> creates decision records
```

Lý do: `x-harness` tối ưu cho “before saying done”, không phải “before work planning”.

---

### G3 — x-harness không phải agent runtime

Không thêm mặc định:

```txt
daemon
database
mandatory MCP server
workflow engine
agent orchestration runtime
long-running background service
task queue
agent memory service
hosted control plane
```

`x-harness` là repo-level contract, không phải runtime platform.

Allowed:

```txt
file-based docs
file-based templates
file-based policies
TypeScript CLI
adapter docs/rules/skills
```

Not allowed by default:

```txt
background process
state server
external orchestration engine
```

---

### G4 — Không vendor lock-in

Core không được phụ thuộc vào:

```txt
Claude Code
Antigravity
Cursor
OpenCode
Codex
Gemini CLI
Windsurf
any specific AI tool
```

Adapters chỉ là lớp mỏng:

```txt
core contract -> adapter docs/rules/skills
```

Tất cả adapters phải dùng cùng admission semantics:

```txt
success -> accepted
failed/blocked/skipped/timeout/error -> withheld
```

Không tạo rule riêng cho từng tool làm lệch core policy.

---

## 4. Token and context guardrails

### G5 — AGENTS.md là map, không phải manual

`AGENTS.md` phải ngắn.

Giới hạn khuyến nghị:

```txt
AGENTS.md <= 100–150 dòng
```

Nội dung chỉ nên gồm:

```txt
core rule
canonical tier labels
where templates live
how to verify
accepted/withheld mapping
read-only verification rule
PGV advisory-only rule
blocked next-action rule
```

Không nhồi toàn bộ docs vào `AGENTS.md`.

Không đưa vào `AGENTS.md`:

```txt
full workflow explanations
all examples
all adapters
deep mode details
long rationale
full schema descriptions
long marketing text
```

---

### G6 — Progressive disclosure bắt buộc

Agent chỉ đọc phần cần thiết.

Expected context loading:

```txt
task nhỏ:
  AGENTS.md
  completion-card template

verify task:
  x-harness-verify skill
  completion-card.yaml

blocked task:
  x-harness-recover skill
  completion-card.yaml

cleanup task:
  x-harness-clean skill

ambiguous task:
  x-harness-scope skill
  optional TASK_READINESS.md

high-risk task:
  deep docs
  rollback policy
```

Không yêu cầu agent đọc mọi thứ trong:

```txt
docs/
templates/
schemas/
policies/
adapters/
examples/
```

cho mọi task.

---

### G7 — Skills phải load-on-demand

Skills là procedure wrappers, không phải core.

Bộ skill khuyến nghị:

```txt
x-harness-adopt
x-harness-scope
x-harness-implement
x-harness-verify
x-harness-recover
x-harness-clean
```

Không tạo skill library lớn.

Không auto-load toàn bộ skills vào mọi prompt.

Mỗi skill phải có:

```txt
trigger
rules
allowed actions
output
stop condition
```

---

### G8 — CLI output quiet by default

Default output của `verify` phải ngắn.

Ví dụ:

```txt
outcome: success
acceptance_status: accepted
checks: 8 passed, 0 failed, 0 blocked
```

Chi tiết chỉ hiện khi gọi:

```bash
npx x-harness verify --verbose
npx x-harness verify --json
npx x-harness report
```

Lý do:

```txt
- giảm log noise
- giảm token nếu agent paste output
- dễ dùng trong CI
- vẫn debug được khi cần
```

---

## 5. Verification guardrails

### G9 — Verification luôn read-only

Verifier không được sửa file.

Không dùng instruction kiểu:

```txt
verify and fix
validate and patch
review and update
check and repair
```

Verify mode chỉ được:

```txt
Read
Grep
Glob
Bash read-only checks
schema validation
policy validation
```

Nếu phát hiện lỗi cần sửa, verifier trả:

```txt
withheld completion
reason
next_action
owner
```

Sau đó phải quay lại work mode.

---

### G10 — Worker không được self-admit completion

Worker có thể claim:

```yaml
claim:
  fix_status: fixed
```

Nhưng worker không được tự quyết accepted completion.

Worker chỉ tạo candidate completion.

Admission chỉ được quyết bởi verify gate.

---

### G11 — fixed không đồng nghĩa accepted

Các trạng thái sau không tự đủ để nói done:

```txt
fix_status: fixed
verification.status: passed
tests passed
PGV says okay
agent confidence: HIGH
manual inspection looks good
```

Chỉ trạng thái này mới accepted:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

---

### G12 — Fail closed

Các outcome sau đều phải withheld:

```txt
failed
blocked
skipped
timeout
error
```

Mapping bắt buộc:

```yaml
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

Không được cho phép:

```txt
blocked but probably okay
failed but acceptable
skipped but done
timeout but likely passed
```

---

### G13 — Blocked là outcome hợp lệ

Blocked không phải failure của agent. Blocked là trạng thái an toàn khi thiếu điều kiện admit.

Blocked phải có:

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: <predicate>
  reason: <reason>

handoff:
  next_action: <next action>
  owner: <owner>
```

Không được ép blocked thành success.

Không được bỏ qua blocked reason.

---

## 6. Evidence guardrails

### G14 — Evidence tối thiểu, không documentation overload

Completion card cần đủ để verify, không cần thành report dài.

Minimum evidence:

```yaml
evidence:
  files_changed: []
  commands_ran: []
  key_outputs: []
```

Không yêu cầu mọi task phải có:

```txt
long audit report
full test matrix
architecture decision record
product spec
rollback plan
```

trừ khi tier là deep hoặc module opt-in yêu cầu.

---

### G15 — Evidence floor thay đổi theo tier

Evidence yêu cầu theo tier:

```txt
light:
  files_changed
  one reasonable check or explanation

standard:
  files_changed
  commands_ran
  checks list
  key outputs

deep:
  standard evidence
  execution_controls
  rollback_policy
  risk/blocker notes
```

Không bắt task `light` đi qua `deep` evidence.

Không dùng `deep` mặc định.

---

### G16 — Không chấp nhận evidence mơ hồ

Không đủ:

```txt
I checked it.
Looks good.
Tests should pass.
Implemented successfully.
Should work now.
No issues found.
```

Cần cụ thể:

```yaml
commands_ran:
  - command: npm run typecheck
    status: passed
  - command: npm test -- product-form
    status: passed

key_outputs:
  - "Typecheck completed successfully."
  - "Product form tests passed."
```

---

### G17 — Không có validation thì blocked hoặc explain rõ

Nếu task cần validation nhưng repo không có runnable command:

```yaml
verification:
  status: blocked

admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: evidence_floor_met
  reason: "No runnable validation command was available."

handoff:
  next_action: "Add a validation command or provide accepted manual verification evidence."
  owner: user
```

Không được tự success khi không có evidence floor.

---

## 7. Tier guardrails

### G18 — Chỉ dùng 3 tier canonical

Allowed:

```txt
light
standard
deep
```

Forbidden:

```txt
small
medium
large
simple
complex
enterprise
basic
advanced
critical
```

Lý do: tránh drift terminology và tránh agent chọn tier theo cảm tính.

---

### G19 — Dùng tier nhỏ nhất đủ verify

Rule:

```txt
Use the smallest tier that preserves verification quality.
```

Tier guidance:

```txt
light:
  typo
  copy update
  CSS nhỏ
  README nhỏ
  one-file low-risk fix

standard:
  web/app implementation bình thường
  form validation
  API handler
  unit test
  refactor nhỏ/vừa
  bug fix nhiều file nhưng bounded

deep:
  auth
  payment
  database migration
  deploy pipeline
  security-sensitive logic
  permission model
  irreversible state change
  large refactor
```

---

### G20 — Deep phải opt-in hoặc risk-triggered

Deep không được là default.

Deep chỉ dùng khi:

```txt
cost of being wrong is high
change may affect data/security/payment/auth/deploy
rollback/recovery cần explicit
task có nhiều bước hoặc rủi ro tích lũy
```

Không dùng deep cho:

```txt
typo
CSS nhỏ
copy update
README update
simple UI label
low-risk component cleanup
```

---

## 8. Cleanup and artifact guardrails

### G21 — Không auto-delete evidence

Không tự xóa:

```txt
completion-card.yaml
.x-harness/archive/
verify reports
audit-relevant artifacts
```

Trừ khi user gọi explicit cleanup command.

Evidence là phần của audit trail.

---

### G22 — Cleanup phải explicit và có dry-run

Các command hợp lệ:

```bash
npx x-harness clean --dry-run
npx x-harness clean --tmp
npx x-harness clean --reset-card
npx x-harness clean --archive-success
```

Mutation command phải có cơ chế an toàn:

```txt
--dry-run
--force where appropriate
clear list of affected files
```

Không xóa file âm thầm.

---

### G23 — Chỉ tmp/cache là safe-delete mặc định

Safe-delete:

```txt
.x-harness/tmp/
.x-harness/cache/
```

Không safe-delete mặc định:

```txt
completion-card.yaml
.x-harness/archive/
templates/
schemas/
policies/
docs/
AGENTS.md
X_HARNESS.md
CLAUDE.md
adapter rules
```

---

## 9. Adapter and skill guardrails

### G24 — Adapter chỉ copy rule/skill/docs

Allowed:

```txt
CLAUDE.md
.claude/skills/*
.claude/agents/*
.cursor/rules/*
.antigravity/rules/*
.antigravity/workflows/*
.opencode/agents/*
```

Not allowed by default:

```txt
background services
custom agent runtime
tool-specific database
hidden state manager
persistent daemon
```

Adapter phải là thin wrapper quanh core contract.

---

### G25 — Adapter không được thay đổi admission semantics

Tất cả adapters phải dùng cùng mapping:

```txt
success -> accepted
failed -> withheld
blocked -> withheld
skipped -> withheld
timeout -> withheld
error -> withheld
```

Không adapter nào được định nghĩa thêm:

```txt
warning -> accepted
partial -> accepted
manual_ok -> accepted
pgv_ok -> accepted
```

---

### G26 — Skill không được admit completion

Skill có thể hướng dẫn agent, nhưng authority vẫn là:

```bash
npx x-harness verify
```

Không skill nào được nói:

```txt
I reviewed this, so it is accepted.
```

Skill chỉ có thể:

```txt
run verify
interpret verify result
return accepted/withheld according to policy
```

---

### G27 — Skill phải ngắn và có trigger rõ

Mỗi skill cần:

```txt
trigger
purpose
rules
allowed actions
required output
stop condition
```

Không viết skill như một long-form manual.

Không đưa toàn bộ docs vào skill.

---

## 10. Tooling constraints

### C1 — Core tooling dùng TypeScript

Core CLI dùng TypeScript.

Allowed:

```txt
packages/cli/src/**/*.ts
```

Not allowed in core:

```txt
scripts/*.py
packages/cli/**/*.py
```

Nếu có Python, chỉ ở:

```txt
legacy/python/
tools/experimental/
```

và phải ghi rõ non-canonical.

---

### C2 — No mandatory external services

Core không được yêu cầu:

```txt
cloud account
hosted API
database
MCP server
LLM evaluator
browser automation service
external queue
remote control plane
```

`npx x-harness verify` phải chạy local-first.

---

### C3 — Deterministic-first

`verify` phải ưu tiên deterministic checks:

```txt
schema validation
policy mapping
file presence
tier validation
owner/accountable validation
status checks
evidence checks
command status checks
read-only verifier check
```

LLM/semantic review nếu có chỉ advisory.

---

### C4 — Mutating commands phải có safety flags

Commands mutate files phải hỗ trợ:

```bash
--dry-run
--merge
--force
```

hoặc safety equivalent.

Áp dụng cho:

```txt
init
add adapter
add readiness-layer
clean
```

---

### C5 — Init không được overwrite im lặng

Nếu file tồn tại, behavior mặc định:

```txt
stop and list conflicts
```

Chỉ overwrite khi user dùng:

```bash
--force
```

Chỉ merge khi user dùng:

```bash
--merge
```

---

## 11. Claim and marketing constraints

### C6 — Không overclaim

Không được nói:

```txt
x-harness guarantees correctness
x-harness makes agents reliable
x-harness proves production safety
verify success equals task success
PGV agreement equals correctness
x-harness prevents all agent errors
x-harness replaces human review
```

Allowed claims:

```txt
x-harness reduces premature completion claims
x-harness separates claimed from accepted completion
x-harness provides a lightweight read-only verify gate
x-harness structures evidence for AI-agent handoff
x-harness supports solo, assisted, and multi-agent workflows
x-harness helps make completion claims auditable
```

---

### C7 — Denominator warning bắt buộc trong report

`npx x-harness report` nên luôn có:

```txt
Verify-event success must not be interpreted as task-level success,
production reliability, benchmark success, or safety guarantee.
```

---

### C8 — Accepted chỉ có nghĩa trong policy hiện tại

Accepted nghĩa là:

```txt
The completion claim passed the configured x-harness admission policy.
```

Accepted không nghĩa là:

```txt
The software is correct in production.
The implementation is bug-free.
The task is globally solved.
No human review is needed.
```

---

## 12. Documentation constraints

### C9 — Docs phải phân tầng

Không dồn vào một file.

Recommended structure:

```txt
AGENTS.md        = map ngắn
X_HARNESS.md     = overview
docs/            = deeper explanation
templates/       = reusable artifacts
schemas/         = machine-readable contracts
policies/        = admission rules
adapters/        = tool-specific integration
examples/        = proof and learning
```

---

### C10 — README không được thành full manual

README chỉ cần:

```txt
what it is
what it is not
quickstart
core rule
minimal workflow
adapter links
examples
```

Không đưa toàn bộ policy/schema/tier/deep mode vào README.

---

### C11 — Examples không được là required context

Examples dùng để:

```txt
learn
test
demonstrate
CI smoke
```

Không được yêu cầu agent đọc examples cho mọi task.

---

## 13. Feature design constraints

### C12 — Mọi feature mới phải có opt-in/default classification

Mỗi feature phải được gắn nhãn:

```txt
core-default
optional
adapter-only
experimental
legacy
```

Rule:

```txt
Nếu feature tăng token/process weight, nó không được là core-default.
```

---

### C13 — Mọi feature mới phải có rollback/removal path

Nếu feature thêm file vào repo, cần trả lời:

```txt
How does user remove or clean it?
Does clean command touch it?
Is it archived?
Is it ignored by git?
```

Không thêm artifact không có lifecycle.

---

### C14 — Mọi feature mới phải có test/example tối thiểu

Feature không có test/example thì không nên merge.

Minimum:

```txt
unit test
golden example
doctor check
or docs example
```

---

### C15 — Không thêm abstraction nếu file convention đủ dùng

Ưu tiên:

```txt
file convention
schema
policy YAML
CLI check
```

Tránh:

```txt
service registry
plugin runtime
state database
complex DSL
multi-process orchestration
```

---

## 14. Practical merge checklist

Mỗi PR thêm feature mới phải trả lời:

```txt
1. Feature này có tăng default token/context không?
2. Có làm AGENTS.md dài hơn không?
3. Có bắt user dùng multi-agent không?
4. Có thêm runtime/service dependency không?
5. Có làm verifier được edit file không?
6. Có làm PGV thành authority không?
7. Có tạo product lifecycle mặc định không?
8. Có --dry-run nếu mutate files không?
9. Có giữ success/withheld mapping không?
10. Có example/test không?
11. Có làm minimal mode nặng hơn không?
12. Có thể bỏ qua nếu user chỉ muốn x-harness minimal không?
```

Nếu câu trả lời vi phạm guardrail, feature phải:

```txt
be rejected
or moved to optional module
or moved to experimental
or removed from default path
```

---

## 15. Doctor checks nên enforce

`npx x-harness doctor` nên kiểm:

```txt
AGENTS.md <= configured max lines
canonical tiers only: light/standard/deep
no small/medium/large tier labels
PGV is advisory-only
verifier instructions do not allow edits
completion-card template exists
admission policy exists
cleanup policy exists if clean enabled
adapter files exist if adapter installed
adapter files do not change admission mapping
no Python in core tooling
no mandatory MCP/DB/daemon references in core docs
docs links resolve
examples verify if present
deep mode not enabled by default
```

---

## 16. Minimal mode constraints

`npx x-harness init --minimal` may create:

```txt
AGENTS.md
X_HARNESS.md
docs/VERIFY_GATE.md
docs/RUNTIME_CONTRACT.md
templates/COMPLETION_CARD.md
templates/SUBAGENT_TASK_light.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
policies/admission.yaml
```

It must not create by default:

```txt
feature intake
story packet
test matrix
decision records
deep governance files
runtime services
tool-specific adapters
MCP config
database config
large examples
```

---

## 17. Adapter constraints

`npx x-harness add adapter <tool>` may create only tool-specific docs/rules/skills.

It must not:

```txt
change core admission policy
install background service
create tool-specific database
override completion-card schema
make tool-specific accepted semantics
```

Allowed adapter commands:

```bash
npx x-harness add adapter claude-code
npx x-harness add adapter antigravity
npx x-harness add adapter cursor
npx x-harness add adapter opencode
```

Each adapter must point back to:

```txt
completion-card.yaml
policies/admission.yaml
npx x-harness verify
```

---

## 18. Cleanup constraints

`npx x-harness clean` must be conservative.

Allowed:

```bash
npx x-harness clean --dry-run
npx x-harness clean --tmp
npx x-harness clean --reset-card
npx x-harness clean --archive-success
```

Forbidden by default:

```txt
delete archive
delete completion-card without backup/reset intent
delete docs/templates/schemas/policies
delete adapter files
delete evidence
```

If archive pruning is added later, it must be explicit:

```bash
npx x-harness clean --archive --before 30d --dry-run
```

---

## 19. Skill constraints

Each skill should be short.

Recommended maximum:

```txt
~80–120 lines per skill
```

Skill must include:

```txt
description
trigger
rules
allowed actions
required output
stop condition
```

Skill must not:

```txt
duplicate full docs
replace CLI verification
admit completion by itself
make deep mode default
make product planning mandatory
```

---

## 20. Tier constraints

Tier labels are policy-bearing terms.

Allowed:

```txt
light
standard
deep
```

Tier behavior:

```txt
light:
  low-risk, narrow, minimal evidence

standard:
  normal web/app implementation, evidence + checks

deep:
  high-risk, rollback/recovery controls required
```

Forbidden aliases:

```txt
small
medium
large
tiny
huge
simple
complex
critical
enterprise
```

If a user says “small”, agent should map it to `light` internally but write canonical `light`.

---

## 21. Admission policy constraints

Admission policy must be simple and stable.

Required concepts:

```yaml
candidate_completion:
  required:
    - claim.fix_status: fixed
    - verification.status: passed

success_requires:
  - owner_present
  - accountable_present
  - evidence_present
  - evidence_floor_met
  - no_unresolved_blocker
  - verifier_read_only

reject_success_if:
  fix_status:
    - partial
    - not_fixed
  verification_status:
    - failed
    - skipped
    - blocked
  timeout: true
  error: true
```

Outcome mapping must remain:

```yaml
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

## 22. Final acceptance criteria for x-harness itself

A version of `x-harness` is acceptable if:

```txt
1. Minimal setup is understandable in under 5 minutes.
2. User can use it with one short prompt.
3. Agent does not need to read long docs for simple tasks.
4. completion-card.yaml is the main artifact.
5. verify is deterministic-first and read-only.
6. blocked/failed/skipped/timeout/error are withheld.
7. skills are optional and load-on-demand.
8. adapters are thin and cross-agent.
9. cleanup is safe and explicit.
10. docs do not overclaim reliability or correctness.
```

---

## 23. Short policy block for CONTRIBUTING.md

This block can be copied directly into `CONTRIBUTING.md`:

```md
## x-harness design constraints

x-harness is a lightweight verify-gated completion harness.

Non-negotiables:

1. Core stays minimal.
2. Verification is read-only.
3. Completion is admitted, not claimed.
4. success is the only accepted outcome.
5. failed/blocked/skipped/timeout/error are withheld.
6. PGV is advisory-only.
7. Deep mode is opt-in.
8. Skills are load-on-demand.
9. No daemon, database, mandatory MCP, or mandatory LLM verifier.
10. Do not turn x-harness into a product planning harness.

Before adding a feature, ask:

Does this make completion harder to fake, easier to verify, and safer to withhold without adding default process weight?

If not, it should not be added to core.
```

---

## 24. Final statement

`x-harness` should stay small by default and strict at the admission boundary.

It should integrate only the smallest mechanisms that make completion:

```txt
harder to fake
easier to verify
safer to withhold
```

without adding default process weight.

That is the core constraint that prevents overengineering.
