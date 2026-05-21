# x-harness: Các cải thiện giá trị nhất, ít trade-off nhất

**Mục tiêu:** Tập hợp các cải thiện nên áp dụng vào `x-harness` để tăng giá trị thực tế, tăng khả năng adoption, cải thiện vận hành dài hạn, nhưng vẫn giữ core nhẹ, ít tốn token và không biến `x-harness` thành một product planning harness hay agent runtime nặng.

---

## 1. Nguyên tắc chọn cải thiện

Một cải thiện chỉ nên đưa vào `x-harness` nếu thỏa các điều kiện sau:

```txt
1. Tăng giá trị rõ ràng cho user hoặc agent.
2. Không làm core phình to.
3. Không tăng context/token mặc định.
4. Không phá read-only verification.
5. Không biến PGV hoặc LLM evaluator thành authority.
6. Không bắt user dùng multi-agent.
7. Không thêm daemon, database, MCP server, hoặc runtime riêng.
8. Có thể bỏ qua nếu user chỉ dùng minimal mode.
```

Core của `x-harness` vẫn phải là:

```txt
one rule + one card + one verify command
```

Cụ thể:

```txt
One rule: completion is admitted, not claimed.
One card: completion-card.yaml.
One command: npx x-harness verify.
```

Các phần mở rộng phải là optional hoặc load-on-demand.

---

## 2. Định vị cần giữ

`x-harness` không nên trở thành:

```txt
- product planning harness
- skill marketplace
- Claude-only plugin
- SDD framework
- agent runtime
- workflow engine
- full governance platform
```

Định vị nên giữ:

```txt
x-harness = lightweight verify-gated completion/admission harness
```

Nói rõ hơn:

```txt
agent work
  -> completion claim
  -> structured evidence
  -> read-only verify
  -> accepted or withheld
```

Câu chốt:

```txt
x-harness does not help agents plan more work by default.
It prevents agents from claiming done before completion is admitted.
```

---

## 3. Nhóm cải thiện nên tích hợp ngay

Đây là các cải thiện có tỷ lệ giá trị/trade-off cao nhất.

---

# P0.1 — CLI output quiet/json/verbose

## Vấn đề

Nếu CLI output quá dài, agent có xu hướng paste nhiều log vào final response, làm tăng token và nhiễu. Nếu output quá ít, CI/debug khó dùng.

## Cải thiện

`npx x-harness verify` nên quiet by default.

Default output:

```txt
outcome: success
acceptance_status: accepted
checks: 8 passed, 0 failed, 0 blocked
```

Verbose output:

```bash
npx x-harness verify --verbose
```

JSON output:

```bash
npx x-harness verify --json
```

Report output:

```bash
npx x-harness report
```

## Expected behavior

```txt
- Default: ngắn, đủ để agent biết accepted/withheld.
- --verbose: giải thích từng check.
- --json: dùng cho CI/tooling.
- report: tạo báo cáo đọc được bởi người.
```

## Giá trị

```txt
- Giảm token trong agent response.
- Dễ dùng trong CI.
- Ít noise.
- Không làm core phức tạp.
```

## Trade-off

```txt
Rất thấp.
```

## Ưu tiên

```txt
P0 — nên làm ngay.
```

---

# P0.2 — clean command

## Vấn đề

`completion-card.yaml`, verify reports, tmp files, cache và archive có thể tạo rác trong repo nếu dùng lâu.

Nhưng không được auto-delete evidence vì `completion-card.yaml` là audit artifact.

## Cải thiện

Thêm command:

```bash
npx x-harness clean --dry-run
npx x-harness clean --tmp
npx x-harness clean --reset-card
npx x-harness clean --archive-success
```

## Artifact layout

```txt
.x-harness/
  tmp/
  cache/
  archive/
```

## Rules

```txt
- Không auto-delete evidence.
- Không xóa archive mặc định.
- Không xóa templates/schemas/policies/docs.
- Không xóa AGENTS.md hoặc X_HARNESS.md.
- --dry-run phải có.
- reset-card và archive-success phải explicit.
```

## Cleanup policy

`policies/cleanup.yaml`:

```yaml
cleanup:
  auto_delete_evidence: false
  tmp_retention_days: 7
  archive_success_default: false
  reset_card_requires_explicit_command: true
  dry_run_first_recommended: true

safe_delete:
  - .x-harness/tmp/
  - .x-harness/cache/

never_delete_by_default:
  - completion-card.yaml
  - .x-harness/archive/
  - templates/
  - schemas/
  - policies/
  - docs/
  - AGENTS.md
  - X_HARNESS.md
```

## Giá trị

```txt
- Repo sạch hơn.
- Dùng lâu không bị rác.
- Giữ được audit trail.
- Không tăng token mặc định.
```

## Trade-off

```txt
Thấp.
```

## Ưu tiên

```txt
P0 — nên làm ngay hoặc v0.1.5.
```

---

# P0.3 — x-harness-verify skill

## Vấn đề

Agent dễ mắc lỗi vừa verify vừa sửa, hoặc tự coi `fix_status: fixed` là done.

## Cải thiện

Thêm skill ngắn để ép read-only verification.

## Skill content

```md
# x-harness-verify

Use this skill to verify a completion claim.

Rules:
- Read-only.
- Do not edit files.
- Inspect `completion-card.yaml`.
- Inspect evidence if available.
- Run `npx x-harness verify`.
- Return accepted only if outcome is success.
- Return withheld for failed, blocked, skipped, timeout, or error.
- PGV advice is advisory-only.

Do not treat these as accepted completion by themselves:
- `fix_status: fixed`
- `verification.status: passed`
- `pgv_advice.claim_allowed: yes`
```

## Adapter placement

```txt
adapters/claude-code/skills/x-harness-verify/SKILL.md
adapters/antigravity/workflows/x-harness-verify.md
adapters/opencode/agents/x-harness-verify.md
```

Cursor có thể dùng rule thay vì skill riêng:

```txt
adapters/cursor/rules/x-harness.mdc
```

## Giá trị

```txt
- Củng cố read-only verification.
- Giảm premature done.
- Tăng worker/verifier separation.
- Tăng adapter usability.
```

## Trade-off

```txt
Thấp. Skill chỉ load khi verify.
```

## Ưu tiên

```txt
P0 — nên làm ngay.
```

---

# P0.4 — x-harness-recover skill

## Vấn đề

Khi verify trả `blocked`, nhiều workflow không biết xử lý tiếp. Agent có thể bỏ cuộc hoặc tự biến blocked thành done.

## Cải thiện

Thêm skill xử lý blocked/withheld.

## Skill content

```md
# x-harness-recover

Use this skill when verification returns blocked.

Rules:
- Do not convert blocked to success.
- Identify blocking predicate.
- Assign next owner.
- Define next action.
- If evidence is missing, ask worker to attach evidence.
- If task is partial, return to work mode.
- If owner is missing, assign one before retry.
- Re-run verification after recovery.
```

## Required blocked output

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: evidence_floor_met
  reason: "No validation evidence attached."

handoff:
  next_action: "Attach validation output and rerun verification."
  owner: implementation-worker
```

## Giá trị

```txt
- Blocked trở thành trạng thái vận hành được.
- Không ép false success.
- Tăng paper alignment.
- Tăng độ tin cậy của final response.
```

## Trade-off

```txt
Thấp-vừa. Thêm một skill, nhưng không load mặc định.
```

## Ưu tiên

```txt
P0 — nên làm sớm.
```

---

# P0.5 — Golden examples

## Vấn đề

Thiết kế tốt nhưng không có ví dụ chạy được thì adoption yếu. User cần thấy output thực tế.

## Cải thiện

Thêm golden examples.

## Structure

```txt
examples/golden/
  success-light/
  blocked-missing-evidence/
  failed-invalid-status/
  withheld-partial-fix/
  multi-agent-success/
```

Mỗi example gồm:

```txt
README.md
input-task.md
completion-card.yaml
expected-verify-output.txt
expected-final-response.md
```

## Command

```bash
npx x-harness examples verify
```

## CI

```yaml
name: x-harness examples

on:
  pull_request:
  push:
    branches: [main]

jobs:
  examples:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
      - run: npm install
      - run: npm run build
      - run: npx x-harness doctor
      - run: npx x-harness examples verify
```

## Giá trị

```txt
- Chứng minh CLI hoạt động.
- User học bằng ví dụ.
- CI kiểm được.
- Không tăng token mặc định.
```

## Trade-off

```txt
Thấp.
```

## Ưu tiên

```txt
P0 — nên làm ngay.
```

---

# P0.6 — doctor nâng cấp

## Vấn đề

Repo có thể drift dần: `AGENTS.md` quá dài, PGV bị viết như authority, verifier được phép edit, tier label không chuẩn.

## Cải thiện

`npx x-harness doctor` nên kiểm thêm:

```txt
- AGENTS.md quá dài.
- Có small/medium/large thay vì light/standard/deep.
- Có wording PGV như authority.
- Có verifier instruction cho phép edit.
- Missing completion-card template.
- Missing admission policy.
- Missing cleanup policy nếu clean enabled.
- Adapter files bị stale.
- Archive/cache chưa có .gitignore phù hợp.
- Python xuất hiện trong core tooling.
- Docs link bị hỏng.
```

## Giá trị

```txt
- Giữ repo không drift.
- Phát hiện overengineering.
- Bảo vệ lightweight default.
- Tăng public maturity.
```

## Trade-off

```txt
Thấp.
```

## Ưu tiên

```txt
P0 — nên làm sớm.
```

---

## 4. Nhóm cải thiện P1: nên thêm, nhưng opt-in

Các cải thiện này có giá trị cao nhưng phải optional để không tăng default weight.

---

# P1.1 — Adapter installer nhẹ

## Vấn đề

Adapter docs tốt nhưng user vẫn phải copy thủ công.

## Cải thiện

Thêm command:

```bash
npx x-harness add adapter claude-code
npx x-harness add adapter cursor
npx x-harness add adapter antigravity
npx x-harness add adapter opencode
```

Command này chỉ copy docs/rules/skills. Không tạo runtime riêng.

## Claude Code output

```txt
CLAUDE.md
.claude/skills/x-harness-verify/SKILL.md
.claude/skills/x-harness-recover/SKILL.md
.claude/skills/x-harness-clean/SKILL.md
.claude/agents/implementation-worker.md
.claude/agents/admission-verifier.md
```

## Cursor output

```txt
.cursor/rules/x-harness.mdc
```

## Antigravity output

```txt
.antigravity/rules/x-harness.md
.antigravity/workflows/x-harness-verify.md
.antigravity/workflows/x-harness-recover.md
.antigravity/workflows/x-harness-clean.md
```

## OpenCode output

```txt
.opencode/agents/x-harness-verify.md
.opencode/agents/x-harness-recover.md
```

## Giá trị

```txt
- Dễ adopt hơn.
- Cross-agent support rõ hơn.
- Không vendor lock-in.
- Không làm core nặng.
```

## Trade-off

```txt
Thấp nếu chỉ copy file.
```

## Ưu tiên

```txt
P1 — nên làm sau P0.
```

---

# P1.2 — x-harness-scope skill

## Vấn đề

Một số task mơ hồ. Nếu agent làm ngay, verify sau đó khó đánh giá.

Không nên copy full feature intake/story packet. Chỉ cần scope tối thiểu.

## Cải thiện

Thêm skill:

```md
# x-harness-scope

Use this skill before implementation when a request is ambiguous or under-specified.

Purpose:
Decide whether the task is ready for work and verification.

Output:

scope:
  goal: <one clear objective>
  in:
    - <in scope>
  out:
    - <out of scope>
  risks:
    - <risk>
  suggested_tier: <light|standard|deep>
  missing_information:
    - <question>
  proceed:
    yes_or_no: <yes|no>
    reason: <why>

Rules:
- Keep scope short.
- Do not create full product lifecycle artifacts.
- Do not create a story packet by default.
- If missing information blocks verification, return proceed: no.
- Recommend the smallest tier that preserves correctness.
- Deep is only for high-risk work.
```

## Giá trị

```txt
- Cải thiện product/spec clarity.
- Không biến x-harness thành planning harness.
- Chỉ dùng khi request mơ hồ.
```

## Trade-off

```txt
Thấp nếu skill ngắn và optional.
```

## Ưu tiên

```txt
P1.
```

---

# P1.3 — TASK_READINESS opt-in

## Vấn đề

Đôi khi cần artifact để ghi readiness, nhưng không nên tạo story/test matrix mặc định.

## Cải thiện

Thêm optional readiness layer:

```bash
npx x-harness add readiness-layer
```

Creates:

```txt
docs/TASK_READINESS.md
templates/TASK_READINESS.md
schemas/task-readiness.schema.json
```

Template:

```yaml
id: TR-001
goal: <one objective>
ready_for_work: <yes|no>
suggested_tier: <light|standard|deep>

scope:
  in: []
  out: []

verification_expectation:
  evidence_needed:
    - <test/check/file/manual>
  not_directly_verifiable:
    - <item + reason>

risks:
  - <risk>

missing_information:
  - <question>

next_action:
  owner: <user|agent>
  action: <action>
```

## Giá trị

```txt
- Tăng task readiness.
- Hỗ trợ task mơ hồ.
- Không copy full product lifecycle.
```

## Trade-off

```txt
Thấp nếu opt-in.
Vừa nếu đưa vào default.
```

## Ưu tiên

```txt
P1/P2, opt-in only.
```

---

# P1.4 — profiles solo/assisted/multi

## Vấn đề

Người dùng khác nhau cần entrypoint khác nhau.

## Cải thiện

Thêm profiles:

```bash
npx x-harness init --profile solo
npx x-harness init --profile assisted
npx x-harness init --profile multi
```

Profile definitions:

```txt
solo:
  AGENTS.md
  X_HARNESS.md
  completion card template
  admission policy

assisted:
  solo + evidence examples for test/typecheck/lint

multi:
  assisted + worker/verifier prompts
```

`--minimal` vẫn là default.

## Giá trị

```txt
- Onboarding tốt hơn.
- Không bắt multi-agent.
- Không làm default nặng.
```

## Trade-off

```txt
Thấp-vừa.
```

## Ưu tiên

```txt
P1/P2.
```

---

## 5. Nhóm cải thiện nên trì hoãn hoặc tránh

---

# Avoid/P3.1 — Full product lifecycle

Không nên thêm mặc định:

```txt
- full feature intake
- story packet
- test matrix lifecycle
- architecture decision system
- backlog/roadmap management
```

Lý do:

```txt
- Làm x-harness giống product planning harness.
- Tăng token và process cost.
- Làm mờ định vị verify-gated completion.
```

Nếu cần, chỉ thêm dạng external optional module trong tương lai.

---

# Avoid/P3.2 — LLM semantic verifier mặc định

Không nên để LLM evaluator quyết định accepted completion.

Nếu có, chỉ advisory:

```yaml
pgv_advice:
  status: advisory
  claim_allowed: no
```

Không được làm:

```txt
LLM verifier says yes -> accepted
```

Lý do:

```txt
- Tốn token.
- Tốn chi phí.
- Dễ tạo false confidence.
- Mâu thuẫn deterministic-first.
```

---

# Avoid/P3.3 — Runtime orchestration

Không nên thêm:

```txt
- daemon
- database
- MCP server bắt buộc
- agent runtime riêng
- workflow engine riêng
```

Lý do:

```txt
- Làm mất lợi thế lightweight.
- Làm user khó adopt.
- Đưa x-harness vào cạnh tranh với agent runtime thay vì repo-level contract.
```

---

# Avoid/P3.4 — Skill sprawl

Không nên có 50–200 skills.

Nên giữ bộ skill nhỏ:

```txt
1. x-harness-adopt
2. x-harness-scope
3. x-harness-implement
4. x-harness-verify
5. x-harness-recover
6. x-harness-clean
```

Mỗi skill phải ngắn, trigger rõ, output rõ.

---

## 6. Bài học chọn lọc từ các repo khác

## 6.1 Từ harness-experimental

Nên lấy:

```txt
- Repo-level source of truth.
- Installer có dry-run/merge/force.
- Docs có cấu trúc.
```

Không nên lấy:

```txt
- Full product lifecycle làm default.
- Feature intake/story/test matrix bắt buộc.
```

Áp dụng:

```bash
npx x-harness init --minimal --dry-run
npx x-harness init --minimal --merge
npx x-harness init --minimal --force
```

---

## 6.2 Từ ai-harness-template

Nên lấy:

```txt
- Profiles.
- Optional gates.
- Install UX rõ.
```

Không nên lấy:

```txt
- Methodology bundle lớn.
- Hook/gate/security quá nhiều mặc định.
```

Áp dụng:

```bash
npx x-harness init --profile solo
npx x-harness init --profile assisted
npx x-harness init --profile multi
```

---

## 6.3 Từ Claude Code Harness

Nên lấy:

```txt
- Plan/work/review loop rõ.
- Reviewer role riêng.
```

Không nên lấy:

```txt
- Claude-only runtime.
- Hook engine làm core.
```

Áp dụng:

```txt
implementation-worker
admission-verifier
```

Nhưng chỉ dưới dạng adapter.

---

## 6.4 Từ ECC / skill-heavy repos

Nên lấy:

```txt
- Cross-tool adapter pattern.
- DRY skill content.
```

Không nên lấy:

```txt
- Hàng chục/hàng trăm skill.
- Skill sprawl.
```

Áp dụng:

```txt
Chỉ 6 skill chính.
```

---

## 6.5 Từ Deep Agents

Nên lấy:

```txt
- Subagent/context awareness.
```

Không nên lấy:

```txt
- Agent runtime.
- Framework dependency.
```

Áp dụng:

```txt
x-harness hỗ trợ multi-agent nhưng không trở thành runtime.
```

---

## 6.6 Từ Superpowers

Nên lấy:

```txt
- Skill trigger rõ.
- Skill phải được dùng đúng lúc.
```

Không nên lấy:

```txt
- Skill framework làm trung tâm.
```

Áp dụng:

```txt
Mỗi x-harness skill có trigger, rules, output, stop condition.
```

---

## 6.7 Từ 0xjgv/harness

Nên lấy:

```txt
- Quiet by default.
- CI read-only.
- Quality guardrails thực dụng.
```

Không nên lấy:

```txt
- Auto-fix trong verify.
```

Áp dụng:

```txt
npx x-harness verify = read-only
npx x-harness doctor = diagnostic
npx x-harness clean = explicit mutation
```

---

## 7. Repo structure đề xuất sau P0/P1

```txt
x-harness/
  README.md
  AGENTS.md
  X_HARNESS.md

  docs/
    QUICKSTART.md
    PRINCIPLES.md
    MODES.md
    VERIFY_GATE.md
    RUNTIME_CONTRACT.md
    ADMISSION_POLICY.md
    CLEANUP.md
    USE_WITH_AI_TOOLS.md
    TASK_READINESS.md

  templates/
    COMPLETION_CARD.md
    VERIFY_REPORT.md
    SUBAGENT_TASK_light.md
    SUBAGENT_TASK_standard.md
    SUBAGENT_TASK_deep.md
    TASK_READINESS.md

  schemas/
    completion-card.schema.json
    verify-event.schema.json
    pgv-advice.schema.json
    task-readiness.schema.json

  policies/
    admission.yaml
    cleanup.yaml

  packages/
    cli/
      src/
        commands/
          init.ts
          verify.ts
          doctor.ts
          report.ts
          clean.ts
          examples.ts
          add.ts

  adapters/
    claude-code/
      skills/
        x-harness-verify/
        x-harness-recover/
        x-harness-clean/
        x-harness-scope/
      agents/
        implementation-worker.md
        admission-verifier.md

    antigravity/
      workflows/
        x-harness-verify.md
        x-harness-recover.md
        x-harness-clean.md
        x-harness-scope.md

    cursor/
      rules/
        x-harness.mdc

    opencode/
      agents/
        x-harness-verify.md
        x-harness-recover.md

  examples/
    00-minimal/
    01-solo-agent/
    02-assisted-agent/
    03-multi-agent/
    04-blocked-verification/
    golden/
      success-light/
      blocked-missing-evidence/
      failed-invalid-status/
      withheld-partial-fix/
```

Minimal mode vẫn chỉ cần:

```txt
AGENTS.md
X_HARNESS.md
templates/COMPLETION_CARD.md
policies/admission.yaml
npx x-harness verify
```

---

## 8. Prioritized roadmap

## Phase P0 — Highest value, lowest trade-off

```txt
1. Quiet/json/verbose verify output
2. clean command
3. x-harness-verify skill
4. x-harness-recover skill
5. golden examples
6. doctor nâng cấp
```

Target outcome:

```txt
- Ít token hơn.
- Repo sạch hơn.
- Verify rõ hơn.
- Blocked xử lý tốt hơn.
- Có evidence adoption.
```

---

## Phase P1 — Adoption and usability

```txt
7. add adapter <tool>
8. x-harness-scope skill
9. TASK_READINESS opt-in
10. examples verify command
11. profiles solo/assisted/multi
```

Target outcome:

```txt
- Dễ áp dụng vào nhiều AI tools.
- Task mơ hồ được scope nhẹ.
- Không bắt product planning.
```

---

## Phase P2 — Advanced opt-in

```txt
12. deep-governance opt-in
13. recovery packet
14. audit report
15. rollback policy
```

Target outcome:

```txt
- Hỗ trợ high-risk workflows.
- Không ảnh hưởng minimal mode.
```

---

## Phase P3 — Avoid unless strongly justified

```txt
16. LLM semantic verifier
17. MCP server
18. database
19. daemon
20. full product lifecycle
```

Target outcome:

```txt
Avoid by default.
Only consider if explicit user demand proves value.
```

---

## 9. Bảng giá trị/trade-off

| Cải thiện | Giá trị | Trade-off | Default? | Ưu tiên |
|---|---:|---:|---:|---:|
| Quiet/json/verbose output | Cao | Rất thấp | Có | P0 |
| `clean` command | Cao | Thấp | Có | P0 |
| `x-harness-verify` skill | Cao | Thấp | Optional load | P0 |
| `x-harness-recover` skill | Cao | Thấp-vừa | Optional load | P0 |
| Golden examples | Cao | Thấp | Không load | P0 |
| Doctor nâng cấp | Cao | Thấp | Có | P0 |
| Adapter installer | Cao | Thấp | Opt-in | P1 |
| `x-harness-scope` skill | Vừa-cao | Thấp | Optional load | P1 |
| TASK_READINESS | Vừa | Thấp nếu opt-in | Opt-in | P1 |
| Profiles | Vừa | Thấp-vừa | Optional | P1 |
| Deep governance | Cao cho high-risk | Vừa-cao | Opt-in | P2 |
| LLM semantic verifier | Vừa | Cao | Không | P3 |
| Runtime orchestration | Không cần cho core | Cao | Không | Avoid |
| Full product lifecycle | Vừa | Cao | Không | Avoid |

---

## 10. Acceptance criteria cho mọi cải thiện

Một PR/cải thiện chỉ nên được merge nếu đạt:

```txt
1. Không tăng default prompt/context đáng kể.
2. Không bắt user dùng multi-agent.
3. Không thêm runtime dependency nặng.
4. Không làm verifier có quyền edit.
5. Không biến PGV thành authority.
6. Không tạo product lifecycle mặc định.
7. Có --dry-run nếu command mutate files.
8. Có example hoặc test.
9. Có docs ngắn.
10. Có thể bỏ qua nếu user chỉ muốn minimal mode.
```

---

## 11. Kết luận

Các cải thiện có giá trị nhất và ít trade-off nhất cho `x-harness` là:

```txt
1. Quiet/json/verbose verify output
2. clean command
3. x-harness-verify skill
4. x-harness-recover skill
5. golden examples + examples verify
6. doctor nâng cấp
7. add adapter <tool>
8. optional x-harness-scope + TASK_READINESS
```

Không nên tích hợp vào default:

```txt
- full product lifecycle
- semantic LLM verifier mặc định
- runtime orchestration
- daemon/DB/MCP bắt buộc
- skill library lớn
```

Định hướng đúng:

```txt
x-harness lấy những bài học tốt nhất từ các harness khác,
nhưng chỉ giữ phần làm tăng admission quality,
không giữ phần làm tăng process weight.
```

Câu chốt:

```txt
x-harness should integrate the smallest mechanisms that make completion harder to fake, easier to verify, and safer to withhold.
```

Đây là cách tăng giá trị mà vẫn giữ được lợi thế chính: nhẹ, ít token, dễ áp dụng, cross-agent, và tập trung vào verify-gated completion.
