Kế hoạch áp dụng X_HARNESS_IMPLEMENTATION_BLUEPRINT.md vào repo
1. Cách hiểu blueprint
Blueprint định nghĩa x-harness như một harness nhẹ để kiểm soát completion claim:
One rule:
  completion is admitted, not merely claimed
One card:
  completion-card.yaml / completion card artifact
One verify command:
  x-harness verify
Core v0.1 theo blueprint gồm:
README.md
AGENTS.md
X_HARNESS.md
docs/
  QUICKSTART.md
  PRINCIPLES.md
  MODES.md
  RUNTIME_CONTRACT.md
  VERIFY_GATE.md
  ADMISSION_POLICY.md
  PGV_ADVISORY.md
  DENOMINATOR_POLICY.md
  INTEGRATION.md
  ADAPTERS.md
  ROADMAP.md
  FAQ.md
templates/
  SUBAGENT_TASK_light.md
  SUBAGENT_TASK_standard.md
  SUBAGENT_TASK_deep.md
  COMPLETION_CARD.md
  VERIFY_REPORT.md
schemas/
  completion-card.schema.json
  subagent-return.schema.json
  verify-event.schema.json
  pgv-advice.schema.json
policies/
  admission.yaml
packages/cli/
  init
  verify
  doctor
  report
adapters/
  generic
  claude-code
  cursor
  opencode
  antigravity
examples/
  00-minimal
  01-solo-agent
  02-assisted-agent
  03-multi-agent
  04-blocked-verification
2. Hiện trạng repo
Repo hiện đã có nền tốt:
✅ package root named x-harness
✅ TypeScript workspace
✅ packages/cli exists
✅ CLI has init/verify/doctor/report
✅ no Python core tooling
✅ AGENTS.md short
✅ X_HARNESS.md exists
✅ README primary branding x-harness
✅ completion card concept exists
✅ verify/admission logic exists
✅ tests pass
✅ doctor currently healthy
✅ adapters exist
✅ examples exist
✅ PGV advisory wording exists
✅ remote pushed commit a6d4243
Nhưng blueprint yêu cầu sâu hơn mức v0.1 scope check trước đó.
3. Gap map đầy đủ
Critical gaps — cần làm trước
C1. schemas/completion-card.schema.json đang quá lỏng
Hiện file này gần như stub:
schemas/completion-card.schema.json
Vấn đề:
additionalProperties: true
thiếu required fields
thiếu enum constraints
thiếu admission/verification/claim shape
Blueprint yêu cầu schema enforce các field như:
schema_version
task_id
tier
owner
accountable
claim.fix_status
verification.status
admission.outcome
acceptance_status
evidence
handoff.next_action
handoff.owner
Tác động:
verify gate có thể bị bypass về mặt cấu trúc.
Ưu tiên:
P0 / critical
C2. policies/admission.yaml còn là stub
Hiện:
policies/admission.yaml
chưa có đầy đủ:
candidate_completion.required
success_requires
reject_success_if
outcome_mapping
blocked_requires
Blueprint yêu cầu policy là contract rõ ràng cho admission.
Tác động:
Không đủ căn cứ machine-readable để giải thích vì sao completion accepted/withheld.
Ưu tiên:
P0 / critical
C3. x-harness verify chưa mặc định đọc completion-card.yaml
Hiện verify command chủ yếu theo model:
--claim
--evidence
--subagent-return
Blueprint muốn default flow:
x-harness verify
tự tìm:
completion-card.yaml
ở repo root/current dir.
Tác động:
Quickstart “one card + one verify command” chưa đúng hoàn toàn.
Ưu tiên:
P0 / critical
C4. doctor chưa đủ các check theo blueprint
Hiện doctor chủ yếu check:
required files exist
some dirs exist
no Python in CLI src
Blueprint muốn doctor check thêm:
schema validity
policy validity
adapter docs exist
no PGV-as-authority wording
AGENTS.md not too large
docs links valid
canonical tier labels only
no small/medium/large runtime tiers
completion card template exists and validates
Ưu tiên:
P0 / critical
C5. report output chưa theo Markdown report spec
Hiện report output là JSON event count.
Blueprint yêu cầu Markdown sections:
# x-harness Report
## Installed mode
## Templates
## Completion card
## Verification summary
## Blocked items
## Denominator warning
Và phải có warning kiểu:
Verify-event success must not be interpreted as task-level success without denominator review.
Ưu tiên:
P0 / critical
Major gaps — sau P0
M1. Claude Code adapter thiếu agents
Blueprint yêu cầu:
adapters/claude-code/agents/implementation-worker.md
adapters/claude-code/agents/admission-verifier.md
Hiện chưa có.
Ưu tiên:
P1
M2. adapters/claude-code/CLAUDE.md quá ngắn
Hiện chỉ là vài dòng. Blueprint muốn hướng dẫn rõ quy trình:
1. implementer produces candidate completion
2. verifier performs read-only verification
3. completion-card updated
4. only accepted if verify/admission pass
Ưu tiên:
P1
M3. Cursor adapter vẫn có file claimgate.mdc
Hiện:
adapters/cursor/rules/claimgate.mdc
Blueprint muốn:
adapters/cursor/rules/x-harness.mdc
với content rõ:
alwaysApply: true
light/standard/deep
read-only verify
PGV advisory-only
no accepted completion without admission
Ưu tiên:
P1
M4. Antigravity adapter sai structure so với blueprint
Hiện dùng:
adapters/antigravity/missions/
Blueprint muốn:
adapters/antigravity/rules/x-harness.md
adapters/antigravity/workflows/x-harness-implementation.md
adapters/antigravity/workflows/x-harness-verify.md
Ưu tiên:
P1
M5. Thiếu docs/ADAPTERS.md
Blueprint liệt kê doc này trong v0.1 docs.
Ưu tiên:
P1
Ghi chú: Có một mâu thuẫn nhỏ trong blueprint: một section liệt kê ADAPTERS.md, section repo tree lại không nêu rõ. Nhưng nên thêm vì chi phí thấp và hữu ích.
M6. Templates còn stub
Các file:
templates/COMPLETION_CARD.md
templates/VERIFY_REPORT.md
hiện chưa đủ dùng như scaffold thực tế.
Blueprint muốn completion card có YAML template đầy đủ.
Ưu tiên:
P1
M7. Examples chưa match blueprint naming/modes
Hiện có:
00-minimal
01-light-task
02-standard-task
03-blocked-verification
04-verify-fail-recovery
05-product-intake-to-accepted-completion
Blueprint muốn:
00-minimal
01-solo-agent
02-assisted-agent
03-multi-agent
04-blocked-verification
Tác động:
repo có nhiều example hơn minimum, nhưng naming không match blueprint.
Ưu tiên:
P1 hoặc P2
Minor gaps / cleanup
m1. Generic adapter quá ngắn
adapters/generic/AGENTS.md
Nên expand thành adapter contract tối thiểu.
m2. OpenCode verify-agent doc quá ngắn
adapters/opencode/verify-agent.md
Nên thêm read-only boundary, return schema, admission semantics.
m3. Legacy ClaimGate naming còn lại
Một số file legacy/compatibility vẫn hợp lệ nếu ghi rõ compatibility. Nhưng nên kiểm soát:
ClaimGate chỉ xuất hiện trong compatibility/legacy sections.
Không dùng làm primary CLI/package/docs branding.
4. Over-scope items cần xử lý cẩn thận
Repo hiện có nhiều thứ vượt v0.1 blueprint:
docs/ARCHITECTURE.md
docs/COMPARISON.md
docs/DECISIONS.md
docs/FEATURE_INTAKE.md
docs/PRODUCT_CONTRACT.md
docs/STORY_PACKET.md
docs/TEMPLATE_AUTHORING.md
docs/TEST_MATRIX.md
templates/AUDIT_REPORT.md
templates/CLAIM_PACKET.md
templates/DECISION_RECORD.md
templates/EVIDENCE_PACKET.md
templates/FEATURE_INTAKE.md
templates/PRODUCT_CONTRACT.md
templates/RECOVERY_PACKET.md
templates/STORY_PACKET.md
templates/TEST_MATRIX_ROW.md
schemas/audit-report.schema.json
schemas/claim.schema.json
schemas/decision-record.schema.json
schemas/evidence.schema.json
schemas/feature-intake.schema.json
schemas/product-contract.schema.json
schemas/recovery.schema.json
schemas/story.schema.json
schemas/task.schema.json
schemas/test-matrix.schema.json
policies/denominator.yaml
policies/escalation.yaml
policies/evidence.yaml
policies/ownership.yaml
policies/pgv.yaml
policies/rollback.yaml
policies/stale-ground.yaml
CLI commands:
  add
  handoff
  trace
Blueprint v0.1 muốn nhẹ. Vì vậy không nên xóa ngay. Kế hoạch tốt hơn:
Giữ lại nhưng đánh nhãn preview/roadmap
Không đưa vào init --minimal
Không làm doctor fail nếu thiếu
Không làm README quickstart phụ thuộc vào chúng
Có thể gom docs lại dưới "advanced / roadmap"
5. Plan triển khai chi tiết
Phase 0 — Baseline snapshot
Mục tiêu: xác nhận state hiện tại trước khi thay đổi.
Action items
- Chạy baseline checks:
npm run typecheck
npm test
npm run build
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js verify --help
node packages/cli/dist/index.js report --help
- Tạo một branch riêng:
git checkout -b blueprint-implementation
- Ghi lại baseline status:
git status --short
git log --oneline -5
Acceptance
Baseline clean hoặc biết rõ files đang modified.
Tests hiện tại pass trước khi thay đổi.
Phase 1 — Core admission contract
Mục tiêu: biến schema + policy thành contract thật.
Files
schemas/completion-card.schema.json
schemas/verify-event.schema.json
schemas/subagent-return.schema.json
schemas/pgv-advice.schema.json
policies/admission.yaml
packages/cli/src/core/admission.ts
packages/cli/tests/admission.test.ts
packages/cli/tests/validators.test.ts
Action items
- Rewrite schemas/completion-card.schema.json với required fields:
{
  "schema_version": "string",
  "task_id": "string",
  "tier": "light|standard|deep",
  "owner": "string",
  "accountable": "string",
  "claim": {
    "fix_status": "fixed|not_fixed|partial",
    "summary": "string",
    "evidence": "array"
  },
  "verification": {
    "status": "passed|failed|skipped|blocked",
    "checks": "array"
  },
  "admission": {
    "outcome": "pending|success|failed|blocked|skipped|timeout|error"
  },
  "acceptance_status": "accepted|withheld",
  "handoff": {
    "next_action": "string",
    "owner": "string"
  }
}
- Enforce canonical consistency:
verification.status=passed + all success criteria -> fix_status=fixed
fix_status=fixed + verification.status blocked/failed/skipped -> not accepted
acceptance_status=accepted only when admission.outcome=success
non-success outcome -> acceptance_status=withheld
blocked/skipped/failed requires handoff.next_action + handoff.owner
- Expand policies/admission.yaml:
version: 1
candidate_completion:
  required:
    - task_id
    - tier
    - owner
    - accountable
    - claim.fix_status
    - verification.status
    - evidence
    - handoff.next_action
    - handoff.owner
success_requires:
  - claim.fix_status == fixed
  - verification.status == passed
  - admission.outcome == success
  - acceptance_status == accepted
  - evidence.present == true
  - owner.present == true
  - accountable.present == true
reject_success_if:
  - verification.status in [failed, blocked, skipped]
  - admission.outcome in [failed, blocked, skipped, timeout, error]
  - claim.fix_status in [not_fixed, partial]
  - evidence.missing == true
  - handoff.owner.missing == true
outcome_mapping:
  success: accepted
  failed: withheld
  blocked: withheld
  skipped: withheld
  timeout: withheld
  error: withheld
- Decide YAML source-of-truth boundary:
Recommended Practical Max:
Keep admission.ts as implementation source.
Use admission.yaml as external contract loaded/validated by doctor.
Do not make runtime fully policy-interpreted yet.
Reason:
A generic YAML expression engine adds complexity and security risk.
Hard-coded TypeScript admission rules are easier to test.
YAML can document policy and be checked for expected keys.
- Add tests for invalid completion cards:
missing owner -> invalid
missing accountable -> invalid
invalid tier -> invalid
verification blocked + accepted -> rejected
fix_status partial + verification passed -> rejected/canonical contradiction
PGV high risk alone does not block if core admission succeeds
non-success outcome always withheld
blocked without next_action/owner invalid
Acceptance
npm test passes
schema rejects structurally invalid cards
admission rejects canonical contradictions
PGV remains advisory-only
Phase 2 — x-harness verify as default one-command gate
Mục tiêu: x-harness verify hoạt động theo blueprint.
Files
packages/cli/src/commands/verify.ts
packages/cli/src/core/schema.ts
packages/cli/src/core/admission.ts
packages/cli/tests/verify.test.ts
examples/00-minimal/completion-card.yaml
examples/03-blocked-verification/completion-card.yaml
Action items
- Add default lookup order:
--card <path> if provided
./completion-card.yaml
./completion-card.yml
./.x-harness/completion-card.yaml
- Keep backward-compatible flags:
--claim
--evidence
--subagent-return
but mark as:
advanced / compatibility mode
- Add output modes:
x-harness verify
x-harness verify --json
x-harness verify --card examples/00-minimal/completion-card.yaml
- Define exit codes:
0 = accepted
1 = withheld / invalid / verification failed
2 = usage/config error
- Output human-friendly default:
x-harness verify
Card: completion-card.yaml
Tier: light
Claim: fixed
Verification: passed
Admission: success
Acceptance: accepted
Result: ACCEPTED
- JSON output should include:
{
  "ok": true,
  "acceptance_status": "accepted",
  "admission_outcome": "success",
  "withheld_reason": null,
  "checks": []
}
- Add blocked example test:
verification.status=blocked
acceptance_status=withheld
handoff.owner present
handoff.next_action present
exit code 1
Acceptance
node packages/cli/dist/index.js verify --card examples/00-minimal/completion-card.yaml
node packages/cli/dist/index.js verify --card examples/03-blocked-verification/completion-card.yaml
Expected:
minimal accepted
blocked withheld with owner/action
Phase 3 — doctor as real repo health check
Mục tiêu: x-harness doctor kiểm tra đúng blueprint.
Files
packages/cli/src/commands/doctor.ts
packages/cli/tests/doctor.test.ts
Action items
- Add required file check:
README.md
AGENTS.md
X_HARNESS.md
docs/VERIFY_GATE.md
docs/RUNTIME_CONTRACT.md
docs/ADMISSION_POLICY.md
docs/PGV_ADVISORY.md
docs/DENOMINATOR_POLICY.md
docs/ROADMAP.md
templates/COMPLETION_CARD.md
templates/SUBAGENT_TASK_light.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
schemas/completion-card.schema.json
schemas/subagent-return.schema.json
schemas/verify-event.schema.json
schemas/pgv-advice.schema.json
policies/admission.yaml
- Add schema compile check:
AJV compiles every core schema.
No schema is empty/stub.
- Add policy key check:
candidate_completion exists
success_requires exists
reject_success_if exists
outcome_mapping exists
- Add no-Python-core check:
packages/cli/**/*.py must not exist
- Add PGV authority wording check:
Reject dangerous phrases in docs/core:
PGV blocks
PGV gates
PGV decides
PGV is authoritative
PGV overrides verify
Allow:
advisory-only
shadow-only
does not block
does not override
- Add tier label check:
Reject runtime docs/code use of:
small
medium
large
when referring to tiers.
- Add AGENTS size check:
AGENTS.md <= 150 lines
- Add adapter presence check:
adapters/generic
adapters/claude-code
adapters/cursor
adapters/opencode
adapters/antigravity
- Add link sanity check:
Practical Max version:
Check local markdown links only.
Do not implement full crawler.
Acceptance
node packages/cli/dist/index.js doctor --root .
Expected:
{
  "healthy": true,
  "checks": [...]
}
Phase 4 — Markdown report
Mục tiêu: report đúng blueprint và denominator discipline.
Files
packages/cli/src/commands/report.ts
packages/cli/tests/report.test.ts
docs/DENOMINATOR_POLICY.md
Action items
-   Change default output to Markdown.
-   Keep optional JSON:
x-harness report --json
- Markdown sections:
# x-harness Report
## Installed mode
## Templates
## Completion card
## Verification summary
## Blocked items
## Denominator warning
- Add denominator warning exactly or near-exact:
Verify-event success must not be interpreted as task-level success without denominator review.
- Include counts only with denominators:
accepted: 3/5 cards
blocked: 2/5 cards
not naked:
60% success
- If denominator unknown:
NOT_COMPUTABLE
Acceptance
node packages/cli/dist/index.js report
node packages/cli/dist/index.js report --json
Phase 5 — Templates become usable scaffolds
Mục tiêu: users can copy/fill templates directly.
Files
templates/COMPLETION_CARD.md
templates/VERIFY_REPORT.md
templates/SUBAGENT_TASK_light.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
Action items
- Expand COMPLETION_CARD.md with YAML fenced block:
schema_version: "0.1"
task_id: ""
tier: "light"
owner: ""
accountable: ""
claim:
  summary: ""
  fix_status: "fixed"
  evidence:
    - type: "command"
      value: ""
verification:
  status: "passed"
  checks:
    - name: ""
      status: "passed"
      note: ""
admission:
  outcome: "success"
acceptance_status: "accepted"
handoff:
  next_action: "none"
  owner: ""
- Add blocked example in same template or separate section:
verification:
  status: "blocked"
admission:
  outcome: "blocked"
acceptance_status: "withheld"
handoff:
  next_action: "Fix missing evidence"
  owner: "agent"
- Expand VERIFY_REPORT.md:
# Verify Report
## Scope
## Files inspected
## Checks
## Outcome
## Blockers
## Handoff
- Ensure task templates match schema:
fix_status semantics
verification.status semantics
next_action owner required
PGV advisory-only
Acceptance
Templates are directly usable by agents.
No template suggests claiming completion without accepted admission.
Phase 6 — Adapter alignment
Mục tiêu: adapters follow blueprint without becoming required default.
Files
docs/ADAPTERS.md
adapters/generic/AGENTS.md
adapters/claude-code/CLAUDE.md
adapters/claude-code/agents/implementation-worker.md
adapters/claude-code/agents/admission-verifier.md
adapters/cursor/rules/x-harness.mdc
adapters/cursor/rules/claimgate.mdc
adapters/antigravity/rules/x-harness.md
adapters/antigravity/workflows/x-harness-implementation.md
adapters/antigravity/workflows/x-harness-verify.md
adapters/opencode/verify-agent.md
adapters/opencode/README.md
Action items
-   Add docs/ADAPTERS.md.
-   Expand generic adapter with 7 rules:
Use light by default
Use standard for multi-step
Use deep only for risk/control decisions
Write completion card before claiming completion
Verifier is read-only
Non-success verify -> withheld
PGV advisory-only
- Add Claude Code worker/verifier agents:
implementation-worker.md
admission-verifier.md
-   Expand CLAUDE.md with workflow.
-   Rename Cursor rule:
claimgate.mdc -> x-harness.mdc
Options:
Preferred: keep claimgate.mdc as tiny compatibility pointer to x-harness.mdc.
-   Add Antigravity rules/workflows while optionally preserving existing missions as compatibility.
-   Expand OpenCode verify-agent doc.
Acceptance
Each adapter is understandable without reading full repo.
Adapter docs do not imply heavy runtime requirements.
Phase 7 — Examples restructure
Mục tiêu: examples match blueprint modes.
Recommended structure
examples/00-minimal/
examples/01-solo-agent/
examples/02-assisted-agent/
examples/03-multi-agent/
examples/04-blocked-verification/
examples/preview/05-product-intake-to-accepted-completion/
Action items
- Rename or duplicate examples carefully:
Current:
01-light-task -> 01-solo-agent
02-standard-task -> 02-assisted-agent
03-blocked-verification -> 04-blocked-verification
- Add 03-multi-agent example:
worker output
verifier output
completion-card.yaml
verify-report.md
- Move product-intake example to preview:
examples/preview/05-product-intake-to-accepted-completion
or keep with clear README:
Preview / roadmap example, not required v0.1 flow.
- Ensure every example has:
completion-card.yaml
README.md or short notes
expected verify outcome
Acceptance
x-harness verify --card examples/00-minimal/completion-card.yaml
x-harness verify --card examples/04-blocked-verification/completion-card.yaml
Phase 8 — Over-scope containment
Mục tiêu: giữ repo nhẹ mà không xóa tài liệu hữu ích.
Action items
- Add docs/ROADMAP.md section:
Advanced/preview artifacts
-   Add docs/ADVANCED.md or docs/PREVIEW_FEATURES.md only if needed.
-   Mark these as preview:
feature intake
story packet
product contract
test matrix
audit report
decision record
trace command
add command
handoff command
-   Ensure README.md quickstart does not reference preview features.
-   Ensure init --minimal does not copy preview files.
-   Decide whether doctor ignores preview files by default.
Recommended:
doctor checks core v0.1 only by default.
doctor --strict can check preview consistency later.
Acceptance
New user sees simple path.
Advanced artifacts don't redefine v0.1 default.
Phase 9 — Tests and verification
Mục tiêu: lock behavior.
Tests to add/update
packages/cli/tests/completion-card-schema.test.ts
packages/cli/tests/admission.test.ts
packages/cli/tests/verify.test.ts
packages/cli/tests/doctor.test.ts
packages/cli/tests/report.test.ts
packages/cli/tests/init.test.ts
Required test cases
- Valid light completion card accepted.
- Missing owner rejected.
- Missing accountable rejected.
- Invalid tier rejected.
- fix_status: partial + verification.status: passed rejected or withheld.
- verification.status: blocked requires next owner/action.
- verification.status: failed maps to withheld.
- verification.status: skipped maps to withheld.
- admission.outcome: timeout maps to withheld.
- PGV high risk does not directly block.
- doctor fails if schema stub detected.
- doctor fails if policy missing required keys.
- report includes denominator warning.
- init --minimal copies only required minimal files.
Validation commands
npm run typecheck
npm run build
npm test
npm run verify
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js verify --card examples/00-minimal/completion-card.yaml
node packages/cli/dist/index.js verify --card examples/04-blocked-verification/completion-card.yaml
node packages/cli/dist/index.js report
6. Recommended execution order
Nếu muốn áp dụng vào repo, mình đề xuất chia thành 4 PR/commit batches.
Batch 1 — Core contract
schemas/completion-card.schema.json
policies/admission.yaml
packages/cli/src/core/admission.ts
tests for admission/schema
Commit message:
Strengthen completion admission contract
Batch 2 — Verify/doctor/report CLI
packages/cli/src/commands/verify.ts
packages/cli/src/commands/doctor.ts
packages/cli/src/commands/report.ts
CLI tests
example completion-card.yaml fixtures
Commit message:
Align CLI with x-harness blueprint
Batch 3 — Templates/adapters/docs
templates/COMPLETION_CARD.md
templates/VERIFY_REPORT.md
docs/ADAPTERS.md
adapters/*
Commit message:
Expand templates and adapters for blueprint flow
Batch 4 — Examples and preview containment
examples/*
README.md
docs/ROADMAP.md
possibly docs/PREVIEW_FEATURES.md
Commit message:
Restructure examples and mark preview features
7. Decisions needed before implementation
Decision 1 — Should admission.yaml be runtime source of truth?
Recommended:
No, not yet.
Use:
admission.ts = executable source of truth
admission.yaml = documented policy contract validated by doctor
Reason:
Runtime YAML expression evaluation adds complexity.
Hard-coded TypeScript is safer and easier to test.
Decision 2 — Should over-scope files be moved or only labeled?
Recommended:
Label first, move only if README/doctor remains confusing.
Reason:
Moving many files creates churn.
Marking preview keeps history and avoids breaking existing references.
Decision 3 — Should legacy ClaimGate files remain?
Recommended:
Yes, but only as compatibility aliases.
Rules:
Primary branding must be x-harness.
ClaimGate allowed only in compatibility/legacy sections.
No CLI/package/docs primary path should use claimgate.
Decision 4 — Should verify keep old --claim/--evidence mode?
Recommended:
Yes, as compatibility/advanced mode.
Default should be:
x-harness verify --card completion-card.yaml
or simply:
x-harness verify
8. What not to implement
Do not implement as required defaults:
daemon
database
server
MCP server
runtime audit service
PGV runtime blocker
LLM semantic verifier
automatic adapter installation
mandatory multi-agent workflow
mandatory deep workflow
full product lifecycle system
These violate the lightweight v0.1 contract unless explicitly marked roadmap/preview.
9. Final actionable checklist
## Phase 0 — Baseline
[ ] Create branch `blueprint-implementation`
[ ] Run baseline typecheck/build/tests/doctor
[ ] Confirm clean git status
## Phase 1 — Core contract
[ ] Strengthen `schemas/completion-card.schema.json`
[ ] Expand `policies/admission.yaml`
[ ] Align `packages/cli/src/core/admission.ts`
[ ] Add schema/admission tests
## Phase 2 — CLI commands
[ ] Make `x-harness verify` default to `completion-card.yaml`
[ ] Preserve `--claim/--evidence/--subagent-return` as compatibility mode
[ ] Expand `doctor` checks
[ ] Change `report` default to Markdown
[ ] Add CLI tests
## Phase 3 — Templates/docs/adapters
[ ] Expand `templates/COMPLETION_CARD.md`
[ ] Expand `templates/VERIFY_REPORT.md`
[ ] Add `docs/ADAPTERS.md`
[ ] Expand generic adapter
[ ] Add Claude Code worker/verifier agents
[ ] Rename Cursor rule to `x-harness.mdc`
[ ] Add Antigravity rules/workflows
[ ] Expand OpenCode verify-agent doc
## Phase 4 — Examples/preview containment
[ ] Rename examples to solo/assisted/multi/blocked model
[ ] Add `03-multi-agent`
[ ] Move or label product-intake as preview
[ ] Ensure README quickstart remains minimal
[ ] Ensure `init --minimal` remains only 9 files
## Phase 5 — Final validation
[ ] Run `npm run typecheck`
[ ] Run `npm run build`
[ ] Run `npm test`
[ ] Run `npm run verify`
[ ] Run `x-harness doctor --root .`
[ ] Run `x-harness verify --card examples/00-minimal/completion-card.yaml`
[ ] Run `x-harness verify --card examples/04-blocked-verification/completion-card.yaml`
[ ] Run `x-harness report`
10. Recommendation
Nên triển khai theo thứ tự:
Batch 1: schema + admission policy
Batch 2: verify/doctor/report
Batch 3: templates/adapters/docs
Batch 4: examples + preview containment
Không nên bắt đầu bằng adapters/examples, vì phần quan trọng nhất của blueprint là:
completion-card.schema.json
admission.yaml
x-harness verify
doctor
report
Nếu các phần này chưa chuẩn thì adapters/docs chỉ “nói đúng” nhưng runtime chưa enforce đúng