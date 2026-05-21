# x-harness: Đề xuất nâng điểm tối đa các tiêu chí yếu

**Tên repo:** `x-harness`  
**Mục tiêu tài liệu:** Định nghĩa chiến lược nâng các tiêu chí còn yếu của `x-harness` để vượt các harness kiểu product/workflow như `harness-experimental`, nhưng không bắt chước chúng, không làm core nặng, không tăng token cost, và vẫn giữ bản sắc verify-gated completion.

---

## 1. Nguyên tắc chiến lược

`x-harness` không nên thắng bằng cách trở thành một product planning harness đầy đủ. Nó nên thắng bằng cách giữ đúng bản sắc riêng:

```txt
x-harness = verify-gated completion contract
          + optional skills
          + adapter packs
          + golden examples
          + lightweight readiness/cleanup/recovery modules
```

Core vẫn phải là:

```txt
one rule + one card + one verify command
```

Cụ thể:

```txt
One rule: completion is admitted, not claimed.
One card: completion-card.yaml.
One command: npx x-harness verify.
```

Không được làm core phình to thành:

```txt
feature intake -> story packet -> test matrix -> decision record -> audit report
```

Những phần đó có thể tồn tại dưới dạng optional module nếu cần, nhưng không được là default.

---

## 2. Vì sao không nên bắt chước harness-experimental

`harness-experimental` mạnh ở upstream workflow:

```txt
human intent -> feature intake -> story -> validation expectation -> implementation
```

`x-harness` nên mạnh ở downstream admission:

```txt
work -> claim -> evidence -> read-only verify -> accepted/withheld
```

Câu phân biệt:

```txt
harness-experimental organizes work before implementation.
x-harness governs completion before saying done.
```

Vì vậy `x-harness` không cần copy full product/spec/story lifecycle. Nếu copy, repo sẽ mất lợi thế:

- nhẹ;
- dễ áp dụng;
- không tốn token;
- không overengineering;
- dùng được với single-agent;
- không yêu cầu multi-agent setup;
- không yêu cầu full product operating process.

---

## 3. Các tiêu chí yếu cần nâng

Các tiêu chí hiện còn thấp hoặc chưa tối ưu:

```txt
1. Product/spec clarity
2. Story/test readiness
3. Cleanup/retention
4. Real-world adoption proof
5. Public maturity
6. Adapter usability
7. Blocked/recovery handling
8. Tooling completeness
9. Optional deep governance
```

Cách nâng không phải là thêm full framework, mà là thêm các lớp nhẹ, opt-in:

```txt
1. Skill layer
2. Readiness layer
3. Cleanup layer
4. Golden examples
5. Adapter packs
6. Recovery discipline
7. Optional deep-governance
```

---

## 4. Kiến trúc nâng điểm

Công thức:

```txt
Core stays minimal.
Skills improve usability.
Adapters map skills to each AI tool.
Optional modules handle advanced cases.
Golden examples prove adoption.
```

Kiến trúc sau khi nâng:

```txt
x-harness/
  core:
    AGENTS.md
    X_HARNESS.md
    templates/
    schemas/
    policies/
    packages/cli/

  skill layer:
    x-harness-adopt
    x-harness-scope
    x-harness-implement
    x-harness-verify
    x-harness-recover
    x-harness-clean

  adapter layer:
    claude-code/
    antigravity/
    cursor/
    opencode/
    generic/

  proof layer:
    examples/
    golden/
    CI smoke tests

  optional layers:
    readiness-layer
    deep-governance
    cleanup/retention
```

---

## 5. Skill layer để nâng usability mà không làm core nặng

Skills là adapter/procedure layer. Skills không thay core.

```txt
x-harness core = rule + artifact + policy + verify command
skills = cách đóng gói procedure để AI agent dùng core dễ hơn
```

Skills phải là optional. Không nhồi toàn bộ skills vào `AGENTS.md`.

`AGENTS.md` chỉ nên trỏ:

```md
For implementation, use x-harness-implement.
For verification, use x-harness-verify.
For ambiguous scope, use x-harness-scope.
For blocked outcomes, use x-harness-recover.
For artifact cleanup, use x-harness-clean.
```

---

## 6. Skill 1: x-harness-adopt

### Mục tiêu

Giúp user áp dụng `x-harness` vào repo hiện có mà không tạo quá nhiều file.

### Trigger

```txt
Add x-harness to this repo.
Set up x-harness minimal mode.
Adopt x-harness for this project.
```

### Behavior

```txt
1. Run or simulate: npx x-harness init --minimal.
2. Add AGENTS.md.
3. Add X_HARNESS.md.
4. Add verify docs and templates.
5. Add policies/admission.yaml.
6. Do not add product lifecycle files by default.
7. Do not add deep governance by default.
8. Ask before adding tool-specific adapters.
9. Preserve existing AGENTS.md if present.
10. Use dry-run before writing when possible.
```

### Skill content

```md
# x-harness-adopt

Use this skill to add x-harness to an existing repository.

Default adoption goal:

- one rule
- one card
- one verify command

Steps:

1. Check whether the repo already has `AGENTS.md`, `CLAUDE.md`, `.cursor/`, `.opencode/`, or Antigravity config.
2. Add only minimal x-harness files by default.
3. Do not install readiness-layer, deep-governance, or product lifecycle files unless the user asks.
4. Do not overwrite existing files without explicit permission.
5. Prefer dry-run output before write operations.

Required minimal files:

- `AGENTS.md`
- `X_HARNESS.md`
- `docs/VERIFY_GATE.md`
- `docs/RUNTIME_CONTRACT.md`
- `templates/SUBAGENT_TASK_light.md`
- `templates/SUBAGENT_TASK_standard.md`
- `templates/SUBAGENT_TASK_deep.md`
- `templates/COMPLETION_CARD.md`
- `policies/admission.yaml`

Non-negotiable:

- Do not make deep workflow default.
- Do not add full product planning by default.
- Do not treat x-harness as an agent framework.
```

### Tiêu chí được nâng

```txt
Ease of adoption
Public maturity
Single-agent usability
Adapter entrypoint clarity
```

---

## 7. Skill 2: x-harness-scope

### Mục tiêu

Nâng `Product/spec clarity` mà không bắt chước full feature intake/story packet.

Skill này chỉ trả lời:

```txt
Task này đã đủ rõ để làm và verify chưa?
```

### Trigger

```txt
Use x-harness to scope this task.
This request is ambiguous; scope it before implementation.
Decide whether this task is ready for work.
```

### Output

```yaml
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
```

### Skill content

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

### Optional template

`templates/TASK_READINESS.md`:

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

### Tiêu chí được nâng

```txt
Product/spec clarity: 6.5 -> 8.2
Story/test readiness: 6.0 -> 7.8
Overengineering control: stays high
```

---

## 8. Skill 3: x-harness-implement

### Mục tiêu

Giúp agent implement nhưng không tự claim accepted completion.

### Trigger

```txt
Implement this using x-harness.
Use x-harness workflow for this change.
Make this change but verify before final response.
```

### Behavior

```txt
1. Choose tier: light | standard | deep.
2. Implement task.
3. Record evidence.
4. Create or update completion-card.yaml.
5. Set admission pending/withheld.
6. Do not say done.
7. Hand off to x-harness-verify.
```

### Skill content

```md
# x-harness-implement

Use this skill when implementing a user task under x-harness.

Rules:

- Do not mark the task done after editing.
- Create or update `completion-card.yaml`.
- Record claim, evidence, files changed, commands run, and verification status.
- Set `admission.outcome: pending` until verification runs.
- Set `admission.acceptance_status: withheld` until verification succeeds.
- Hand off to `x-harness-verify` before final response.

Tier selection:

- light: narrow, low-risk task
- standard: normal implementation or bounded synthesis
- deep: high-risk task

Completion claim is not accepted completion.

Required completion card fields:

- id
- task_id
- tier
- owner
- accountable
- claim.fix_status
- claim.summary
- evidence.files_changed
- evidence.commands_ran
- verification.status
- admission.outcome
- admission.acceptance_status
- handoff.next_action
- handoff.owner
```

### Tiêu chí được nâng

```txt
Single-agent usability
Multi-agent handoff
Adoption
Completion discipline
```

---

## 9. Skill 4: x-harness-verify

### Mục tiêu

Đây là skill quan trọng nhất. Nó thực hiện read-only admission verification.

### Trigger

```txt
Verify this with x-harness.
Run x-harness verification.
Use x-harness read-only verify mode.
```

### Behavior

```txt
1. Do not edit files.
2. Read completion-card.yaml.
3. Inspect evidence.
4. Run npx x-harness verify.
5. Return one outcome:
   - success
   - failed
   - blocked
   - skipped
   - timeout
   - error
6. Only success maps to accepted.
7. Everything else maps to withheld.
```

### Skill content

```md
# x-harness-verify

Use this skill to verify a completion claim.

Rules:

- Read-only.
- Do not edit source files.
- Inspect `completion-card.yaml`.
- Inspect changed files and evidence if available.
- Run `npx x-harness verify`.
- Return one outcome:
  - success
  - failed
  - blocked
  - skipped
  - timeout
  - error

Admission mapping:

- success -> accepted
- failed -> withheld
- blocked -> withheld
- skipped -> withheld
- timeout -> withheld
- error -> withheld

Only success maps to accepted.
Everything else maps to withheld.

PGV advice is advisory-only.

Do not treat:
- `fix_status: fixed`
- `verification.status: passed`
- `pgv_advice.claim_allowed: yes`

as accepted completion by themselves.
```

### Tiêu chí được nâng

```txt
Read-only verification
Verify-gated completion
Worker/verifier separation
Paper alignment
Adapter usability
```

---

## 10. Skill 5: x-harness-recover

### Mục tiêu

Xử lý blocked outcome mà không làm mềm thành success.

### Trigger

```txt
Recover this x-harness blocked task.
Verification returned blocked; determine next action.
Handle this withheld completion.
```

### Behavior

```txt
1. Identify blocking predicate.
2. Identify blocked reason.
3. Assign next owner.
4. Define next action.
5. Return to work mode only if needed.
6. Re-run verification after recovery.
7. Never convert blocked directly to success.
```

### Skill content

```md
# x-harness-recover

Use this skill when verification returns blocked.

Rules:

- Do not convert blocked to success.
- Identify blocking predicate.
- Assign next owner.
- Define next action.
- If missing evidence, ask worker to attach evidence.
- If stale context, refresh context.
- If partial fix, return to work mode.
- If owner is missing, assign one before retry.
- After recovery, run verification again.

Required blocked output:

admission:
  outcome: blocked
  acceptance_status: withheld
  blocking_predicate: <predicate>
  reason: <reason>

handoff:
  next_action: <next action>
  owner: <owner>
```

### Tiêu chí được nâng

```txt
Blocked/recovery handling: 7.0 -> 9.0
Paper alignment: 9.5 -> 9.7
Deep readiness
Operational stability
```

---

## 11. Skill 6: x-harness-clean

### Mục tiêu

Giữ repo sạch mà không mất audit/evidence.

### Trigger

```txt
Clean x-harness artifacts.
Reset x-harness card.
Archive successful completion card.
```

### Behavior

```txt
1. Dry-run first.
2. Clean tmp/cache safely.
3. Do not auto-delete evidence.
4. Archive/reset only when explicit.
5. Never delete templates/schemas/policies/docs.
```

### Commands

```bash
npx x-harness clean --dry-run
npx x-harness clean --tmp
npx x-harness clean --reset-card
npx x-harness clean --archive-success
```

### Skill content

```md
# x-harness-clean

Use this skill to clean generated x-harness artifacts safely.

Rules:

- Never auto-delete evidence.
- Never delete templates, schemas, policies, docs, AGENTS.md, or X_HARNESS.md.
- Prefer dry-run first.
- Safe cleanup targets:
  - `.x-harness/tmp/`
  - `.x-harness/cache/`
- Resetting `completion-card.yaml` must be explicit.
- Archiving successful cards must be explicit.

Recommended commands:

- `npx x-harness clean --dry-run`
- `npx x-harness clean --tmp`
- `npx x-harness clean --reset-card`
- `npx x-harness clean --archive-success`
```

### Cleanup policy

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

### Tiêu chí được nâng

```txt
Cleanup/retention: 7.5 -> 9.0
Repo hygiene
Long-term usability
```

---

## 12. Adapter mapping cho skills

### 12.1 Claude Code

Create:

```txt
adapters/claude-code/skills/
  x-harness-adopt/SKILL.md
  x-harness-scope/SKILL.md
  x-harness-implement/SKILL.md
  x-harness-verify/SKILL.md
  x-harness-recover/SKILL.md
  x-harness-clean/SKILL.md

adapters/claude-code/agents/
  implementation-worker.md
  admission-verifier.md
```

Each skill should be short, tool-limited, and refer back to:

```txt
completion-card.yaml
templates/
policies/admission.yaml
npx x-harness verify
```

### 12.2 Antigravity

Create:

```txt
adapters/antigravity/workflows/
  x-harness-adopt.md
  x-harness-scope.md
  x-harness-implement.md
  x-harness-verify.md
  x-harness-recover.md
  x-harness-clean.md
```

These are workflow equivalents of the skills.

### 12.3 Cursor

Create:

```txt
adapters/cursor/rules/x-harness.mdc
```

Cursor should not need every skill as a separate file in v0.1. The rule should reference the procedures by name and point to docs.

### 12.4 OpenCode

Create:

```txt
adapters/opencode/agents/
  x-harness-verify.md
  x-harness-recover.md
```

OpenCode can use verify/recover agents first. Implement/adopt/scope can remain generic prompt procedures unless needed.

---

## 13. Optional readiness-layer

To improve product/spec clarity without copying a product harness, add a readiness layer.

Command:

```bash
npx x-harness add readiness-layer
```

Creates:

```txt
docs/TASK_READINESS.md
templates/TASK_READINESS.md
schemas/task-readiness.schema.json
```

Purpose:

```txt
Confirm task readiness before implementation.
Do not manage full product lifecycle.
Do not create story/test matrix by default.
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

This raises planning clarity while preserving x-harness identity.

---

## 14. Golden examples and adoption proof

To raise real-world adoption proof, add golden examples.

Structure:

```txt
examples/
  00-minimal/
  01-solo-agent/
  02-assisted-agent/
  03-multi-agent/
  04-blocked-verification/
  05-claude-code/
  06-antigravity/
  07-cursor/
  08-opencode/
  golden/
    success-light/
    blocked-missing-evidence/
    failed-invalid-status/
    withheld-partial-fix/
    multi-agent-success/
```

Each example must include:

```txt
README.md
input-task.md
completion-card.yaml
expected-verify-output.txt
expected-final-response.md
```

Add command:

```bash
npx x-harness examples verify
```

CI should run:

```bash
npm install
npm run build
npx x-harness doctor
npx x-harness examples verify
```

This turns adoption claims into testable evidence.

---

## 15. CLI commands after enhancement

Core v0.1 commands:

```bash
npx x-harness init --minimal
npx x-harness verify
npx x-harness doctor
npx x-harness report
```

Enhanced commands:

```bash
npx x-harness clean --dry-run
npx x-harness clean --tmp
npx x-harness clean --reset-card
npx x-harness clean --archive-success

npx x-harness examples verify

npx x-harness add adapter claude-code
npx x-harness add adapter antigravity
npx x-harness add adapter cursor
npx x-harness add adapter opencode

npx x-harness add readiness-layer
```

Optional later:

```bash
npx x-harness add deep-governance
```

---

## 16. Optional deep-governance

Deep governance should not be default. It is opt-in.

Command:

```bash
npx x-harness add deep-governance
```

Creates:

```txt
docs/DEEP_MODE.md
templates/RECOVERY_PACKET.md
templates/AUDIT_REPORT.md
schemas/recovery.schema.json
schemas/audit-report.schema.json
policies/rollback.yaml
```

Purpose:

```txt
Handle high-risk, multi-step, high-cost tasks.
```

Deep mode must preserve:

```txt
success -> accepted
blocked -> withheld + recovery owner/action
failed -> withheld
skipped -> withheld
timeout/error -> withheld
```

---

## 17. Do not do

Do not add the following to default:

```txt
- full feature intake
- full story packet
- full test matrix lifecycle
- architecture decision system
- MCP server
- database
- daemon
- LLM semantic verifier by default
- PGV blocking authority
- deep governance for every task
```

Do not position x-harness as:

```txt
x-harness = harness-experimental + verify gate
```

Position it as:

```txt
x-harness = verify-gated completion contract + optional skills
```

---

## 18. Updated score targets

| Criterion | Before | Enhancement | Target |
|---|---:|---|---:|
| Product/spec clarity | 6.5 | `x-harness-scope`, `TASK_READINESS.md` | 8.2 |
| Story/test readiness | 6.0 | readiness-layer | 7.8 |
| Cleanup/retention | 7.5 | `x-harness-clean`, cleanup policy | 9.0 |
| Real-world adoption proof | 5.0 | golden examples + CI smoke | 8.5 |
| Public maturity | 5.5 | docs + examples + npm + CI | 8.5 |
| Adapter usability | 7.5 | skill packs + add adapter | 9.4 |
| Blocked recovery | 7.0 | `x-harness-recover` | 9.0 |
| Tooling completeness | 8.9 | clean/examples/add/scope | 9.3 |
| Paper alignment | 9.5 | blocked instrumentation + recovery | 9.7 |
| Overall | 9.1 | optional skill/proof/readiness layers | 9.4–9.5 |

---

## 19. Implementation roadmap

### Phase 1 — Core completion harness

```txt
- init
- verify
- doctor
- report
- completion-card
- admission policy
- 3 tier templates
- basic adapters
```

Target score:

```txt
8.6 / 10
```

### Phase 2 — Skill layer

```txt
- x-harness-adopt
- x-harness-scope
- x-harness-implement
- x-harness-verify
- x-harness-recover
- x-harness-clean
- docs/USE_WITH_AI_TOOLS.md
```

Target score:

```txt
9.0 / 10
```

### Phase 3 — Proof layer

```txt
- golden examples
- npx x-harness examples verify
- CI smoke tests
- expected outputs
```

Target score:

```txt
9.2 / 10
```

### Phase 4 — Optional readiness + cleanup

```txt
- readiness-layer
- task-readiness template/schema
- clean command
- cleanup policy
```

Target score:

```txt
9.3 / 10
```

### Phase 5 — Deep opt-in

```txt
- recovery packet optional
- rollback policy optional
- audit report optional
```

Target score:

```txt
9.4–9.5 / 10
```

---

## 20. Acceptance criteria for this enhancement plan

The enhancement is complete when:

```txt
1. Core remains minimal.
2. Skills are optional.
3. Adapters map skills to each AI tool.
4. Readiness layer does not become product planning.
5. Cleanup does not auto-delete evidence.
6. Golden examples exist and are verifiable.
7. Blocked recovery is explicit and owner-tagged.
8. Deep governance is opt-in.
9. No default MCP, DB, daemon, semantic verifier, or PGV blocking authority.
10. Accepted/withheld semantics remain unchanged.
```

---

## 21. Final strategic statement

`x-harness` should beat other harnesses not by planning more work, but by making every AI-agent completion claim auditable, verifiable, and safely withheld unless admitted.

The winning formula:

```txt
minimal core
optional skills
tool-specific adapters
golden examples
readiness without product bloat
cleanup without evidence loss
recovery without false success
```

That keeps `x-harness` lightweight, flexible, stable, easy to integrate, low-token, and aligned with verify-gated completion.
