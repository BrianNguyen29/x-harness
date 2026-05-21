Kế hoạch đánh giá chuyên sâu x-harness
Approach
Mục tiêu là thực hiện một đợt audit/readiness review toàn diện cho x-harness sau các commit gần đây, tập trung vào 4 trục:
1. Đồng bộ thông tin: docs/templates/schemas/policies/adapters/examples/CLI có nói cùng một contract không.
2. Nhất quán kỹ thuật: schema/admission/verify/report/doctor/examples/tests có cùng semantics không.
3. Ổn định vận hành: typecheck/build/test/doctor/examples verify/metrics/clean đều chạy ổn định.
4. Đánh giá chất lượng: xác định điểm mạnh, rủi ro, over-scope, missing docs, future work.
Sẽ dùng lại session phù hợp:
explorer: exp-1 Analyze paper brief
fixer: fix-2 Integrate paper brief; fix-1 Implement golden examples
Nhưng vì đây là audit/analysis, nên ưu tiên:
explorer → discovery/map
oracle → strategic review nếu phát hiện rủi ro design/consistency
verify → final read-only verification gate nếu có thay đổi hoặc claim hoàn tất
Scope
In
- Kiểm tra đồng bộ toàn repo:
README.md
AGENTS.md
X_HARNESS.md
docs/*
templates/*
schemas/*
policies/*
packages/cli/src/*
packages/cli/tests/*
examples/*
adapters/*
package.json / package-lock.json
- Kiểm tra các contract chính:
completion-card schema
admission policy
verify output
report metrics
doctor checks
clean safety
examples verify
golden examples
recovery routing
evidence scope
state/governance fields
PGV advisory-only
read-only verifier
no heavy runtime default
- Kiểm tra repo stability:
git status
npm run typecheck
npm run build
npm test
npm run verify
x-harness doctor --root .
x-harness examples verify
x-harness report --metrics ...
x-harness verify --json ...
x-harness clean --tmp
- Đánh giá chất lượng:
Completeness
Consistency
Safety
Backward compatibility
Operational readiness
Documentation sync
Test coverage
Adoption readiness
Scope control
Maintainability
Out
Không làm ngay nếu chưa được yêu cầu:
Không thêm daemon/database/server/MCP
Không thêm CI phức tạp
Không thêm adapter installer nếu chưa qua review
Không thêm task-readiness/product-planning layer
Không thêm deep-governance runtime nặng
Không rewrite architecture lớn
Không commit/push nếu chỉ audit
Action Items
1. Map current repository state
Dùng @explorer để lập bản đồ hiện trạng sau commit mới nhất:
git log --oneline -10
git status --short
package scripts
CLI commands
docs inventory
schema inventory
policy inventory
template inventory
adapter inventory
example/golden inventory
test inventory
Deliverable:
repo_state_map.md hoặc summary trong chat
Các điểm cần xác nhận:
latest commit = 1e6f47b
working tree clean
không còn Zone.Identifier untracked
node_modules không bị stage
planning docs đã tracked đúng
2. Build a source-of-truth matrix
Tạo matrix so sánh “ai nói gì” cho cùng một semantics.
Matrix columns
Concept
Schema source
Policy source
CLI behavior
Template wording
Docs wording
Adapter wording
Example coverage
Test coverage
Status
Gap
Concepts cần kiểm tra
accepted completion
withheld completion
fix_status semantics
verification.status semantics
admission.outcome mapping
PGV advisory-only
read-only verifier
evidence scope
tier floors: light/standard/deep
state.read_set/write_set
governance.human_approval
recovery routing
denominator warning
report metrics
clean safety
golden examples
Deliverable:
contract_consistency_matrix.md
3. Audit schema/policy/runtime alignment
Kiểm tra 3 lớp phải khớp:
schemas/completion-card.schema.json
policies/admission.yaml
packages/cli/src/core/admission.ts
Checks
- Schema fields optional/required đúng như thiết kế.
- Optional paper fields không phá backward compatibility.
- admission.ts enforce tier floors đúng.
- admission.yaml không hứa thứ mà runtime không enforce.
- Runtime không enforce thứ mà docs/policy không nói.
- Canonical contradiction vẫn bị chặn:
accepted + non-success
passed + not_fixed/partial
non-success + accepted
deep without evidence scope
deep approval pending
Deliverable:
schema_policy_runtime_alignment.md
4. Audit CLI behavior and output contracts
Kiểm tra từng command:
x-harness init
x-harness verify
x-harness doctor
x-harness report
x-harness clean
x-harness examples verify
x-harness add
x-harness handoff
x-harness trace
Specific checks
verify
quiet default <= 3 lines
--verbose full detail
--json enriched output
backward compatibility fields preserved:
  ok
  acceptance_status
  admission_outcome
  withheld_reason
hash fields present:
  input_card_hash
  policy_hash
recovery route present for blocked/failed
denominator warning present
doctor
required assets include X_HARNESS.md
schema compile
policy keys
evidence scope support
read-only verifier
no heavy runtime
templates present
cleanup policy
local markdown links
report
default Markdown unaffected
--json unaffected
--metrics works
--metrics --json works
denominator warning present
NOT_COMPUTABLE used when denominator absent
clean
dry-run default
--force required for mutation
protected paths not deleted
completion-card archive/reset safe
Deliverable:
cli_contract_audit.md
5. Audit docs/templates/adapters synchronization
Kiểm tra đồng bộ wording và tránh overclaim.
Docs
README.md
X_HARNESS.md
docs/VERIFY_GATE.md
docs/ADMISSION_POLICY.md
docs/RUNTIME_CONTRACT.md
docs/DENOMINATOR_POLICY.md
docs/PGV_ADVISORY.md
docs/METRICS.md
docs/RECOVERY.md
docs/CLEANUP.md
docs/ADAPTERS.md
docs/ROADMAP.md
Templates
templates/COMPLETION_CARD.md
templates/VERIFY_REPORT.md
templates/SUBAGENT_TASK_light.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
templates/HARNESS_CHANGE_CONTRACT.md
Adapters
adapters/generic/AGENTS.md
adapters/claude-code/CLAUDE.md
adapters/claude-code/skills/*
adapters/cursor/rules/*
adapters/opencode/*
adapters/antigravity/*
Checks
- Không có wording “PGV gates/blocks/authoritative”.
- Không có “verify success = task success” overclaim.
- Không có mandatory deep/multi-agent workflow.
- Không có mandatory daemon/DB/MCP/server.
- Adapter docs không override runtime contract.
- Templates không tạo canonical contradiction by default.
- Deep template có đủ evidence/state/governance.
- Standard template khuyến nghị scope nhưng không hard-block quá mức.
Deliverable:
docs_templates_adapters_sync_audit.md
6. Audit examples and golden examples
Kiểm tra:
examples/00-minimal
examples/01-solo-agent
examples/02-assisted-agent
examples/03-multi-agent
examples/04-blocked-verification
examples/preview
examples/golden/*
Checks
- Mỗi golden scenario có:
README.md
input-task.md
completion-card.yaml
expected-verify-output.txt
expected-final-response.md
- examples verify pass cho 9 scenarios.
- Expected outputs khớp actual outputs.
- Examples cover:
success-light
standard scoped success
blocked missing evidence
blocked missing evidence scope
deep approval required
failed invalid status
failed typecheck recovery route
multi-agent success
withheld partial fix
- Preview examples không bị README/doctor coi là required.
Deliverable:
examples_coverage_audit.md
7. Run full stability validation
Chạy validation ở repo root:
npm run typecheck
npm run build
npm test
npm run verify
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js examples verify
node packages/cli/dist/index.js examples verify --json
node packages/cli/dist/index.js verify --card examples/golden/success-standard-scoped-evidence/completion-card.yaml --json
node packages/cli/dist/index.js verify --card examples/golden/deep-approval-required/completion-card.yaml --json
node packages/cli/dist/index.js report --metrics --card examples/golden/success-standard-scoped-evidence/completion-card.yaml --json
node packages/cli/dist/index.js clean --tmp
Expected:
typecheck pass
build pass
tests pass
verify pass
doctor healthy=true
examples verify 9/9 pass
verify success returns accepted
verify deep approval returns withheld
metrics JSON valid
clean stays dry-run/no destructive action
Deliverable:
validation_evidence_packet.md
8. Perform strategic quality evaluation
Dùng @oracle nếu discovery thấy rủi ro, hoặc để đánh giá tổng thể.
Rubric
Chấm 1–5 hoặc LOW/MED/HIGH cho:
Core clarity
Contract consistency
Evidence discipline
Failure handling
Recovery quality
Doc sync
Adapter readiness
Example coverage
Backward compatibility
Operational stability
Scope control
Maintainability
Adoption readiness
Paper/application usefulness
Output
strengths
risks
high-value fixes
deferred items
do-not-add items
release readiness verdict
Deliverable:
quality_evaluation.md
9. Fix only low-risk inconsistencies
Nếu audit phát hiện gap nhỏ, xử lý bằng @fixer theo scope hẹp.
Allowed fixes:
doc wording mismatch
template field mismatch
missing example README note
missing test assertion
doctor required asset mismatch
minor CLI output mismatch
policy wording mismatch
Do not fix automatically without review if:
requires schema breaking change
changes admission semantics
removes existing public fields
adds new CLI surface
changes default behavior
touches package config broadly
Deliverable:
patch summary + validation rerun
10. Final verify gate and release/readiness report
Sau audit/fixes:
1. Run scoped @verify.
2. Produce final readiness report.
Verify refs
x-harness
Final report sections
Executive summary
Current commit
Working tree status
Validation commands and outputs
Consistency matrix summary
Known limitations
Deferred roadmap items
Release readiness verdict
Commit/push recommendation if files changed
Deliverable:
x-harness_readiness_report.md hoặc final chat report
Validation
Minimum validation set:
npm run typecheck
npm run build
npm test
npm run verify
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js examples verify
node packages/cli/dist/index.js report --metrics --card examples/golden/success-standard-scoped-evidence/completion-card.yaml --json
Optional targeted checks:
node packages/cli/dist/index.js verify --card examples/golden/deep-approval-required/completion-card.yaml --json
node packages/cli/dist/index.js verify --card examples/golden/blocked-missing-evidence-scope/completion-card.yaml --json
node packages/cli/dist/index.js clean --tmp
Risk Focus
Đặc biệt cần soi kỹ các điểm sau:
1. Doctor required assets không được lệch với repo thực tế.
2. Deep tier evidence floors không được vô tình áp dụng cho light/standard.
3. verify --json backward compatibility không được phá.
4. report --metrics không được imply benchmark/safety success.
5. PGV không được thành gate/blocker.
6. Recovery routing không được tự động claim completion.
7. Golden examples expected output không được drift.
8. Templates không được tạo default contradiction.
9. Preview docs/examples không được trở thành required core.
10. No heavy runtime wording không được false positive/false negative.