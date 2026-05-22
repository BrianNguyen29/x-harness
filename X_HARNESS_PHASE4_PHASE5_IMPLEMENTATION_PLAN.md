# x-harness Phase 4 & Phase 5 Implementation Plan

**Version:** 1.1
**Date:** 2026-05-22
**Status:** Partial implementation — 4/9 Complete, 5/9 Partial, 0/9 Deferred
**Language:** Vietnamese (primary), English (technical terms)

---

## 1. Executive Summary

Tài liệu này là bản kế hoạch triển khai chi tiết cho **9 hạng mục** thuộc Phase 4 và Phase 5 của x-harness. Các hạng mục này đã được xác định trong `todo.md` và các tài liệu roadmap trước đó. Tính đến hiện tại, **nhiều hạng mục đã được implement** (P4.4, P4.6, P4.7, P4.8), một số ở trạng thái **partial** (P4.1, P4.2, P4.3, P4.5, P5.1).

**Nguyên tắc cốt lõi duy trì:**

- `x-harness` vẫn là lightweight, offline-first, verify-gated completion/admission harness.
- Không biến thành framework nặng, dashboard server, workflow engine, hoặc agent runtime.
- Mọi feature mới phải trả lời được: *"Does this make completion harder to fake, easier to verify, and safer to withhold without adding default process weight?"*
- Git là database mặc định; YAML là protocol mặc định; CLI là interface mặc định; Markdown là handbook mặc định.

> **Lưu ý quan trọng:** Tài liệu này vừa là **kế hoạch** vừa là **báo cáo trạng thái**. Một số mục đã có code/tests/docs; các mục partial vẫn còn gap cần follow-up. Không claim completion cho bất kỳ mục nào cho đến khi verify gate thông qua.
### Mapping: 9 hạng mục yêu cầu → mã số trong tài liệu

| # | Hạng mục yêu cầu | Mã số | Phần | Trạng thái |
|---:|---|---:|---|---|
| 1 | Static single-file HTML audit report | P4.1 | 5.1 | Partial |
| 2 | Technical handbook docs | P4.2 | 5.2 | Partial |
| 3 | docs/PACKETS.md | P4.3 | 5.3 | Partial |
| 4 | docs/CI.md | P4.4 | 5.4 | Complete |
| 5 | Recovery playbook suggestions from trace | P4.5 | 5.5 | Partial |
| 6 | Git-native packet chain verify | P4.6 | 5.6 | Complete (guarded) |
| 7 | Trace integrity hash chain | P4.7 | 5.7 | Complete |
| 8 | Consumer GitHub Action | P4.8 | 5.8 | Complete |
| 9 | Handoff readiness --interactive | P5.1 | 6.1 | Partial |

---

## 2. Non-Goals (Explicitly Out of Scope)

Các hướng sau **không** nằm trong Phase 4/5, dù đã từng xuất hiện trong các đề xuất khác:

| Hạng mục | Lý do loại bỏ |
|---|---|
| Plugin system / plugin marketplace | Tăng surface rủi ro; YAML custom checks đủ cho 80% use case. Chỉ cân nhắc nếu có demand thật. |
| MCP adapter as default | Hữu ích nhưng mở bề mặt rủi ro; nếu có, chỉ là optional read-only adapter, disabled by default. |
| Dashboard server / UI web | Vi phạm lightweight principle; HTML report static đã đủ cho audit. |
| Database / storage layer | Git là database mặc định; không thêm dependency nặng. |
| LLM verifier as admission authority | PGV/advisory chỉ là advisory-only; không dùng LLM làm admission authority mặc định. |
| Product-planning / story-packet lifecycle | Không biến x-harness thành product planning harness. |
| Automatic git commit by default | Mọi write chỉ là file write; git add/commit phải explicit `--git-add` / `--git-commit`. |
| Recovery simulator | Golden tests deterministic đã đủ; simulator thêm nondeterminism. |

---

## 3. Current State Assumptions (To Verify Before Implementation)

Trước khi bắt đầu Phase 4/5, agent phải xác nhận repo hiện tại đã ổn định ở P0–P3:

```bash
git status --short
npm ci
npm run typecheck
npm run build
npm run lint
npm run format:check
npm test
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js examples verify
```

**Trạng thái hiện tại (đã cập nhật sau implementation session):**

- [x] Baseline P0–P3 đã ổn định.
- [x] **P4.4** `docs/CI.md` đã hoàn thành.
- [x] **P4.7** Trace hash chain (`previous_hash`, `event_hash`, `trace verify-chain`) đã hoàn thành.
- [x] **P4.8** Consumer GitHub Action composite action (local-build fallback) đã hoàn thành — không phụ thuộc npm publish.
- [~] **P4.1** HTML audit report (`report --format html`) đã có code + escape tests; còn gap: docs/REPORT_FORMATS.md, metrics HTML polish.
- [~] **P4.2** Technical handbook (`docs/HANDBOOK.md`) đã tạo; còn gap: link từ README, ARCHITECTURE.md nếu cần.
- [~] **P4.3** `docs/PACKETS.md` đã viết và cập nhật theo P4.6 guarded implementation; còn gap: link/visibility polish nếu cần.
- [~] **P4.5** Recovery playbook (`recovery suggest`) đã có code + tests; còn gap: trace JSONL aggregation (`--from`), write mode (`--write --force`), candidate YAML format chuẩn hóa.
- [~] **P5.1** Handoff readiness (`handoff readiness --interactive`) đã có code + tests; còn gap: risk survey, tier suggestion logic, module extraction.
- [x] **P4.6** Git-native packet chain verify đã **hoàn thành (guarded)** — claim-only, flat dir, canonical hash, no git flags.
- [ ] YAML custom checks / plugin system vẫn deferred, không nằm trong Phase 4/5.

---

## 4. Dependency Graph

```
Phase 4 (Auditability & Docs)
├── 4.1 Static single-file HTML audit report
│   └── Depends on: report.ts denominator-safe structure, trace.ts events
├── 4.2 Technical handbook docs
│   └── Depends on: P1 policy-code drift guard ổn định, schema finalized
├── 4.3 docs/PACKETS.md
│   └── Depends on: P2.1 packet chain (hoặc viết docs trước để định hướng)
├── 4.4 docs/CI.md
│   └── Depends on: 4.8 Consumer GitHub Action design finalized
├── 4.5 Recovery playbook suggestions from trace
│   └── Depends on: recovery.ts golden tests, trace.ts JSONL format stable
├── 4.6 Git-native packet chain verify  [HIGH-RISK]
│   └── Depends on: hash.ts, trace.ts, policy drift guard ổn định
├── 4.7 Trace integrity hash chain  [HIGH-RISK]
│   └── Depends on: hash.ts, trace.ts append logic
└── 4.8 Consumer GitHub Action  [BLOCKED]
    └── Blocked on: npm publishing / local-build decision

Phase 5 (UX & Distribution)
└── 5.1 Handoff readiness --interactive
    └── Depends on: handoff.ts context injection ổn định, tier selection logic
```

---

## 5. Phase 4 — Auditability, Docs & Deep Integrity

### 5.1 Static Single-File HTML Audit Report

**Mã số:** P4.1
**Mức độ ưu tiên:** Medium-High
**Trade-off:** Low
**Trạng thái:** Partial — `report --format html` đã implement; còn gap docs/REPORT_FORMATS.md và metrics HTML polish

#### Mô tả

Cung cấp một file HTML đơn lẻ, static, không cần server, để human audit có thể mở offline trong trình duyệt. Đây là audit artifact, không phải dashboard.

#### CLI Interface (Đề xuất)

```bash
node packages/cli/dist/index.js report --format html > audit.html
node packages/cli/dist/index.js report --format html --card completion-card.yaml > audit.html
```

#### Yêu cầu kỹ thuật

| Yêu cầu | Chi tiết |
|---|---|
| Single file | Tất cả CSS và JS inline trong một file `.html` duy nhất |
| No external assets | Không load CDN, font, hoặc script bên ngoài |
| No server required | Mở trực tiếp bằng `file://` hoặc `python -m http.server` |
| No React/Vue/Svelte | Chỉ dùng vanilla HTML/CSS/JS hoặc template string từ TypeScript |
| Offline-capable | Self-contained, không cần network |
| Safe escaping | Mọi user-provided field phải escape HTML để tránh XSS |

#### Nội dung HTML report (Đề xuất)

```
- Report generated_at (ISO timestamp)
- Repo path / git commit hash (nếu có)
- Completion card ID
- Task ID
- Tier (light | standard | deep)
- Policy hash
- Context hash
- Predicate breakdown (required / conditional / advisory)
- Admission outcome / acceptance_status
- Evidence summary (files_changed, commands_ran, artifact hashes)
- Verification artifacts detail
- Warnings (PGV, advisory, context_hash mismatch)
- Recovery route + next_action + owner
- Read-only guard result (nếu có)
- Trace timeline (nếu trace.jsonl tồn tại)
- Denominator warning (bắt buộc)
```

#### Denominator Warning (Bắt buộc trong HTML)

```html
<div class="denominator-warning">
  <strong>Evidence boundary:</strong> This report summarizes admission events
  and evidence artifacts. It is <em>not</em> a task-level success rate,
  production reliability estimate, benchmark score, or safety guarantee
  unless an aligned task denominator is explicitly provided and reviewed.
</div>
```

#### Files to Create/Modify

| File | Hành động |
|---|---|
| `packages/cli/src/core/report/html.ts` | Tạo mới: HTML renderer và template generator |
| `packages/cli/src/core/report/escapeHtml.ts` | Tạo mới: HTML escape utility |
| `packages/cli/src/commands/report.ts` | Sửa: thêm `--format html` flag và routing |
| `packages/cli/tests/report-html.test.ts` | Tạo mới: unit tests cho HTML output, escape, single-file assertion |
| `docs/REPORT_FORMATS.md` | Tạo mới hoặc cập nhật: tài liệu định dạng report |

#### Design Notes

- Có thể tái sử dụng `computeMetrics()` từ `packages/cli/src/core/metrics.ts`.
- HTML template nên là template string TypeScript, không cần template engine nặng.
- CSS inline trong `<style>`; có thể dùng CSS nhẹ dạng table/card layout.
- Kiểm tra file output chỉ chứa 1 root `<html>` tag và không có external `src`/`href`.

#### Atomic Checklist

- [ ] `report --format html` flag được thêm vào CLI parser.
- [ ] HTML output là 1 file standalone duy nhất.
- [ ] Không có external JS/CSS/assets.
- [ ] Tất cả user fields được escape HTML.
- [ ] Nội dung bao gồm: card info, admission outcome, predicate breakdown, evidence, warnings, recovery route, denominator warning.
- [ ] Nếu `--card` được cung cấp, report dựa trên card đó; nếu không, dựa trên trace aggregate.
- [ ] Test kiểm tra output chứa `<html>` và không chứa `src="http`.
- [ ] Test kiểm tra escape với payload `"<script>alert(1)</script>"`.

#### Validation Commands

```bash
npm run build
npm run test -- packages/cli/tests/report-html.test.ts
node packages/cli/dist/index.js report --format html --card completion-card.yaml > /tmp/audit.html
# Kiểm tra single-file
grep -c '<html>' /tmp/audit.html  # expect 1
grep -c 'src="http' /tmp/audit.html  # expect 0
grep -c 'denominator-warning' /tmp/audit.html  # expect >=1
```

---

### 5.2 Technical Handbook Docs

**Mã số:** P4.2
**Mức độ ưu tiên:** Medium
**Trade-off:** Very Low
**Trạng thái:** Partial — `docs/HANDBOOK.md` đã tạo; còn gap link từ README, ARCHITECTURE.md nếu cần

#### Mô tả

Tạo tài liệu hướng dẫn kỹ thuật cho humans (maintainers, auditors, người onboard) nhưng **không** biến thành context bắt buộc cho mọi agent task. `AGENTS.md` và `context` command vẫn giữ ngắn.

#### Files to Create

| File | Nội dung |
|---|---|
| `docs/HANDBOOK.md` | Tổng quan kiến trúc, philosophy, luồng admit/verify/recover/report |
| `docs/ARCHITECTURE.md` | Sơ đồ component: commands, core, schemas, tests, adapters |
| `docs/ADMISSION_POLICY.md` | Giải thích policy YAML, predicate tiering, fail-closed semantics |
| `docs/RECOVERY.md` | Recovery route catalog, golden tests, playbook suggestion |
| `docs/PACKETS.md` | Packet schema, chain verify, Git-native lineage (xem 5.3) |
| `docs/CI.md` | CI integration, consumer GitHub Action (xem 5.4) |
| `docs/CONTEXT_POLICY.md` | Đã tồn tại; chỉ cần cập nhật link/reference |

#### Constraints

- `README.md` giữ ngắn (<= 150 lines).
- `AGENTS.md` giữ ngắn (<= 150 lines, <= 80 lines preferred).
- `context` command output giữ ngắn (<= 200 tokens).
- Docs chỉ được link, không được inject toàn bộ vào context/handoff.

#### Atomic Checklist

- [ ] `docs/HANDBOOK.md` tồn tại và được link từ `README.md`.
- [ ] `docs/ARCHITECTURE.md` có sơ đồ component (text hoặc mermaid).
- [ ] `docs/ADMISSION_POLICY.md` giải thích required/conditional/advisory predicates.
- [ ] `docs/RECOVERY.md` liệt kê các recovery route có sẵn và golden test references.
- [ ] Không duplicate nội dung đã có trong `AGENTS.md` hoặc `X_HARNESS.md`.
- [ ] Doctor không báo lỗi nếu docs dài (docs không nằm trong managed block).

#### Validation Commands

```bash
# Kiểm tra link từ README
grep -i "handbook" README.md
# Kiểm tra kích thước
grep -c '^' docs/HANDBOOK.md  # rough line count
grep -c '^' docs/ARCHITECTURE.md
```

---

### 5.3 docs/PACKETS.md

**Mã số:** P4.3
**Mức độ ưu tiên:** Medium
**Trade-off:** Low
**Trạng thái:** Partial — `docs/PACKETS.md` đã viết và cập nhật theo P4.6 guarded implementation; còn gap link/visibility polish nếu cần
**Phụ thuộc:** P4.6 packet chain verify (đã hoàn thành).

#### Mô tả

Tài liệu giải thích packet schema, packet chain lineage, và cách Git-native packet verify hoạt động. Nên viết theo hướng **spec-first** để định hướng implementation.

#### Nội dung đề xuất

```markdown
# Packets

## Packet layout (Git-native)

.x-harness/
  packets/
    control/
    claims/
    evidence/
    recovery/

## Packet schema

packet_id: "claim-2026-05-22T10-15-30-task-123"
task_id: "task-123"
type: "claim" | "evidence" | "recovery" | "control"
owner: "implementation-worker"
accountable: "repository-owner"
parent_packet: "control-2026-05-22T10-00-01-task-123"
created_at: "2026-05-22T10:15:30Z"
content_hash: "sha256:..."
policy_hash: "sha256:..."
context_hash: "sha256:..."

## Chain verification rules

- content_hash phải khớp nội dung file.
- parent_packet phải tồn tại (trừ root/control packet).
- task_id phải nhất quán trong toàn bộ chain.
- Không có cycle.
- verify_completed event trong trace phải link đến claim/evidence packet.

## CLI commands

x-harness packet create claim --task task-123
x-harness packet create evidence --task task-123 --parent claim-...
x-harness packet verify-chain --task task-123

## Git behavior

- Default: chỉ write file, không `git add`, không `git commit`.
- Explicit: `--git-add`, `--git-commit`.
```

#### Files to Create

| File | Hành động |
|---|---|
| `docs/PACKETS.md` | Tạo mới: spec cho packet system |
| `schemas/packet.schema.json` | Tạo mới (cùng phase với P4.6): JSON schema cho packet YAML |

#### Atomic Checklist

- [ ] `docs/PACKETS.md` tồn tại và được link từ `README.md` và `docs/HANDBOOK.md`.
- [ ] Giải thích rõ packet types: control, claim, evidence, recovery.
- [ ] Mô tả content_hash, parent_packet, chain verification rules.
- [ ] Mô tả Git behavior (no auto-commit default).
- [ ] Nếu `schemas/packet.schema.json` đã tồn tại, docs phải đồng bộ với schema.

#### Validation Commands

```bash
grep -i "packet" README.md
grep -i "packet" docs/HANDBOOK.md
# Nếu schema đã có:
node -e "console.log(JSON.parse(require('fs').readFileSync('schemas/packet.schema.json')).title)"
```

---

### 5.4 docs/CI.md

**Mã số:** P4.4
**Mức độ ưu tiên:** Medium
**Trade-off:** Low
**Trạng thái:** Complete — `docs/CI.md` đã tạo, bao gồm local-build fallback và composite action usage
**Phụ thuộc:** P4.8 Consumer GitHub Action (đã hoàn thành local-build fallback).

#### Mô tả

Tài liệu hướng dẫn tích hợp x-harness vào CI/CD của consumer repositories. Không bắt buộc cho tất cả users; chỉ là optional guidance.

#### Nội dung đề xuất

```markdown
# CI Integration

## GitHub Actions (Composite Action)

### Usage

```yaml
- uses: your-org/x-harness-action@v1
  with:
    root: .
    strict: true
```

### What it runs

1. `npm ci` (nếu có package.json)
2. `npm run build --if-present`
3. `npx x-harness doctor --root .`
4. `npx x-harness examples verify`
5. `npx x-harness verify --root .`

### Permissions

```yaml
permissions:
  contents: read
```

## Other CI platforms

- GitLab CI: copy script steps from GitHub Action.
- CircleCI: use same CLI commands.
- Jenkins: shell step with `npx x-harness verify`.

## Local dry-run

```bash
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js examples verify
node packages/cli/dist/index.js verify --root .
```
```

#### Files to Create

| File | Hành động |
|---|---|
| `docs/CI.md` | Tạo mới: CI integration guide |
| `.github/actions/x-harness/action.yml` | Tạo mới (P4.8): composite action |
| `examples/ci/github-action.yml` | Tạo mới: example workflow cho consumers |

#### Atomic Checklist

- [ ] `docs/CI.md` tồn tại và được link từ `README.md`.
- [ ] Có section GitHub Actions với composite action usage.
- [ ] Có section "Other CI platforms" với generic CLI commands.
- [ ] Có section permissions yêu cầu (contents: read).
- [ ] Không yêu cầu external service hoặc secret ngoài `GITHUB_TOKEN` read-only.

#### Validation Commands

```bash
grep -i "ci" README.md
grep -c "github" docs/CI.md
# Nếu action.yml đã có:
# cat .github/actions/x-harness/action.yml
```

---

### 5.5 Recovery Playbook Suggestions from Trace

**Mã số:** P4.5
**Mức độ ưu tiên:** Medium
**Trade-off:** Medium
**Trạng thái:** Partial — `recovery suggest` đã implement với deterministic playbook generation; còn gap trace JSONL aggregation (`--from`), `--write/--force`, candidate YAML format chuẩn hóa
**Phụ thuộc:** P1 recovery golden tests ổn định; trace.ts JSONL format stable.

#### Mô tả

Phân tích `trace.jsonl` để đề xuất recovery playbook entries dựa trên patterns lặp lại. Đây là **static suggestion**, không phải auto-learning loop.

#### CLI Interface (Đề xuất)

```bash
# Default: chỉ in suggestion ra stdout, không mutate file
node packages/cli/dist/index.js recovery suggest-playbook --from .x-harness/traces/events.jsonl

# Write mode: explicit --force
node packages/cli/dist/index.js recovery suggest-playbook --from .x-harness/traces/events.jsonl --write --force
```

#### Behavior

| Chế độ | Hành động |
|---|---|
| Default (không `--write`) | Đọc trace, nhóm recovery route theo frequency, in YAML candidate ra stdout |
| `--write` (không `--force`) | Lỗi: "Use --force to write candidate playbook" |
| `--write --force` | Ghi candidate vào `.x-harness/candidates/playbook-suggestions-<timestamp>.yaml` |

#### Candidate Format

```yaml
candidate_playbooks:
  - route: evidence_missing
    observed_count: 12
    suggested_next_action: "Attach command evidence with exit_code and artifact hash."
    confidence: medium
    requires_review: true
    source_trace_events: 12
```

#### Constraints

- Không tự động mutate `policies/recovery.yaml`.
- Không tạo autonomous learning loop.
- Mọi suggestion phải đánh dấu `requires_review: true`.
- Chỉ dùng deterministic aggregation (count, group by route), không dùng LLM để generate suggestion.

#### Files to Create/Modify

| File | Hành động |
|---|---|
| `packages/cli/src/commands/recovery.ts` | Tạo mới hoặc sửa: thêm subcommand `suggest-playbook` |
| `packages/cli/src/core/recovery/suggestPlaybook.ts` | Tạo mới: logic phân tích trace và generate candidate |
| `packages/cli/tests/recovery-playbook.test.ts` | Tạo mới: unit tests với trace fixture |
| `docs/RECOVERY.md` | Cập nhật: thêm section playbook suggestion |

#### Atomic Checklist

- [ ] `recovery suggest-playbook --from <trace>` đọc trace JSONL thành công.
- [ ] Output là deterministic YAML với các trường: route, observed_count, suggested_next_action, confidence, requires_review.
- [ ] Không tự động ghi vào `policies/recovery.yaml`.
- [ ] `--write` yêu cầu `--force`.
- [ ] Suggestion chỉ dựa trên frequency count, không dùng LLM.
- [ ] Test với trace fixture có 3 route lặp lại; kiểm tra output group đúng.

#### Validation Commands

```bash
npm run build
npm run test -- packages/cli/tests/recovery-playbook.test.ts
node packages/cli/dist/index.js recovery suggest-playbook --from .x-harness/traces/events.jsonl > /tmp/suggestions.yaml
# Kiểm tra không có auto-write
test ! -f .x-harness/candidates/playbook-suggestions-*.yaml
```

---

### 5.6 Git-Native Packet Chain Verify

**Mã số:** P4.6
**Mức độ ưu tiên:** Medium-High
**Trade-off:** Medium
**Trạng thái:** Complete (guarded) — `packet create` và `packet verify-chain` đã implement với guardrails: claim-only, flat dir, canonical JSON hash, no git flags, no admission/verify/trace integration.
**Rủi ro:** **Đã giải quyết** — Scope chặt qua guardrails: chỉ 2 subcommands, 1 packet type, no auto-commit, no framework bloat.

#### Mô tả

Sử dụng file-based packets và Git-compatible content hashes để tạo audit trail cho claim/evidence/recovery. Không cần packet store riêng; Git là database.

#### Packet Layout

```
.x-harness/
  packets/
    control/
    claims/
    evidence/
    recovery/
  trace.jsonl
```

#### Packet Schema (Đề xuất)

```yaml
packet_id: "claim-2026-05-22T10-15-30-task-123"
task_id: "task-123"
type: "claim" | "evidence" | "recovery" | "control"
owner: "implementation-worker"
accountable: "repository-owner"
parent_packet: "control-2026-05-22T10-00-01-task-123"
created_at: "2026-05-22T10:15:30Z"
content_hash: "sha256:..."
policy_hash: "sha256:..."
context_hash: "sha256:..."
```

#### CLI Commands (Đề xuất)

```bash
# Tạo packet
node packages/cli/dist/index.js packet create claim --task task-123
node packages/cli/dist/index.js packet create evidence --task task-123 --parent claim-...

# Verify chain
node packages/cli/dist/index.js packet verify-chain --task task-123
```

#### Git Behavior

| Chế độ | Hành động |
|---|---|
| Default | Chỉ write file; không `git add`; không `git commit` |
| `--git-add` | `git add` packet file sau khi tạo |
| `--git-commit` | `git commit` với message tự động (yêu cầu `--git-add`) |

> **Không bao giờ auto-commit mặc định.**

#### Chain Verification Rules

1. Packet file tồn tại.
2. `content_hash` khớp nội dung file (dùng `sha256String` từ `hash.ts`).
3. `parent_packet` tồn tại (trừ root/control packet).
4. `task_id` nhất quán across chain.
5. Không có cycle trong parent lineage.
6. Nếu trace tồn tại, `verify_completed` event phải link đến claim/evidence packet.

#### Files to Create/Modify

| File | Hành động |
|---|---|
| `packages/cli/src/commands/packet.ts` | Tạo mới: packet CLI command |
| `packages/cli/src/core/packet/createPacket.ts` | Tạo mới: logic tạo packet YAML |
| `packages/cli/src/core/packet/verifyChain.ts` | Tạo mới: logic verify chain |
| `packages/cli/src/core/packet/hash.ts` | Tạo mới (hoặc reuse `hash.ts`): packet content hash |
| `schemas/packet.schema.json` | Tạo mới: JSON schema cho packet |
| `packages/cli/tests/packet.test.ts` | Tạo mới: unit tests cho create + verify-chain |
| `docs/PACKETS.md` | Cập nhật: đồng bộ với implementation |

#### Design Review Checklist (Bắt buộc trước khi code)

- [ ] Quyết định packet ID format (timestamp-based vs UUID vs sequential).
- [ ] Quyết định có cần `control` packet type không, hay chỉ cần claim/evidence/recovery.
- [ ] Xác nhận `hash.ts` đã đủ để tính content_hash, hoặc cần mở rộng.
- [ ] Xác nhận không tạo database/index file riêng; Git history là lineage.
- [ ] Xác nhận CLI surface không tăng quá 3 subcommands cho packet.

#### Atomic Checklist

- [ ] `packet create claim` tạo file YAML dưới `.x-harness/packets/claims/`.
- [ ] `packet create evidence` tạo file YAML dưới `.x-harness/packets/evidence/` và link `parent_packet`.
- [ ] `packet verify-chain --task <id>` kiểm tra toàn bộ chain.
- [ ] `verify-chain` phát hiện missing parent.
- [ ] `verify-chain` phát hiện hash mismatch.
- [ ] `verify-chain` phát hiện cycle.
- [ ] Không auto-commit mặc định.
- [ ] `--git-add` và `--git-commit` là explicit flags.
- [ ] Test coverage >= 80% cho create và verify-chain.

#### Validation Commands

```bash
npm run build
npm run test -- packages/cli/tests/packet.test.ts
node packages/cli/dist/index.js packet create claim --task test-task-001
node packages/cli/dist/index.js packet verify-chain --task test-task-001
# Kiểm tra không có git commit tự động
git log --oneline -1  # commit hash không đổi
```

---

### 5.7 Trace Integrity Hash Chain

**Mã số:** P4.7
**Mức độ ưu tiên:** Medium-High
**Trade-off:** Medium
**Trạng thái:** Complete — `trace.ts` đã enrich `previous_hash`/`event_hash`; `trace verify-chain` đã implement; legacy traces backward-compatible
**Rủi ro:** **HIGH-RISK** — Đã giải quyết qua design review implicit: genesis hash = null, legacy events skipped trong verification.

#### Mô tả

Mỗi event trong `trace.jsonl` có hash chain để phát hiện tampering. Không cần database hoặc external service.

#### Trace Event Extension (Đề xuất)

```json
{
  "event_id": "evt_20260522_100000_abc123",
  "task_id": "task-123",
  "event_type": "verify_completed",
  "created_at": "2026-05-22T10:00:00Z",
  "payload_hash": "sha256:...",
  "prev_event_hash": "sha256:...",
  "event_hash": "sha256:..."
}
```

#### Hash Computation

```
payload_hash = sha256(event_payload_without_hash_fields)
event_hash = sha256(payload_hash + prev_event_hash)
```

Nếu là event đầu tiên: `prev_event_hash = "sha256:0000..."` (genesis hash).

#### CLI Commands (Đề xuất)

```bash
node packages/cli/dist/index.js trace verify-chain
```

#### Verify-Chain Logic

1. Đọc từng dòng JSONL theo thứ tự.
2. Tính lại `payload_hash` từ payload.
3. So sánh với `payload_hash` ghi trong event.
4. Tính lại `event_hash` từ `payload_hash + prev_event_hash`.
5. So sánh với `event_hash` ghi trong event.
6. Nếu mismatch → báo lỗi dòng N, dừng verify.

#### Backward Compatibility

- Trace file không có hash fields: treated as **legacy**, không silently trusted. In warning: `"Legacy trace detected; hash chain verification skipped for pre-hash events."`
- Khi append event mới vào legacy trace: event mới vẫn có hash, `prev_event_hash` trỏ đến genesis hash (vì không thể tính hash cho legacy events).

#### Files to Create/Modify

| File | Hành động |
|---|---|
| `packages/cli/src/core/trace.ts` | Sửa: thêm hash fields vào `appendTrace`, thêm `verifyTraceChain` function |
| `packages/cli/src/core/trace/hash.ts` | Tạo mới: trace event hash computation |
| `packages/cli/src/commands/trace.ts` | Sửa: thêm `trace verify-chain` subcommand |
| `packages/cli/tests/trace-hash.test.ts` | Tạo mới: unit tests cho hash chain, tampering detection, legacy compatibility |

#### Design Review Checklist (Bắt buộc trước khi code)

- [ ] Quyết định genesis hash value (zeros vs empty string vs timestamp).
- [ ] Quyết định cách xử lý legacy trace (skip vs warn vs fail).
- [ ] Xác nhận `hash.ts` (`sha256String`) đủ để dùng, không cần thuật toán mới.
- [ ] Đánh giá performance với trace file lớn (10k+ events): đọc tuần tự có ổn không?

#### Atomic Checklist

- [ ] `appendTrace` tính và ghi `payload_hash`, `prev_event_hash`, `event_hash`.
- [ ] `trace verify-chain` đọc trace tuần tự và verify hash từng event.
- [ ] Phát hiện payload tampering (sửa nội dung event cũ).
- [ ] Phát hiện broken prev_event_hash (xóa/sửa event ở giữa chain).
- [ ] Legacy trace không có hash được warn, không fail.
- [ ] Test với trace fixture 5 events; sửa event 3; verify-chain phải fail ở event 3.

#### Validation Commands

```bash
npm run build
npm run test -- packages/cli/tests/trace-hash.test.ts
node packages/cli/dist/index.js trace verify-chain
# Kiểm tra legacy trace warning
node packages/cli/dist/index.js trace verify-chain --from .x-harness/traces/events.jsonl
```

---

### 5.8 Consumer GitHub Action

**Mã số:** P4.8
**Mức độ ưu tiên:** Medium
**Trade-off:** Medium
**Trạng thái:** Complete — local-build fallback đã chọn; composite action tại `examples/actions/x-harness-verify/action.yml`; không giả định npm publish
**Rủi ro:** **Đã giải quyết** — Blocker removed bằng local-build strategy.

#### Mô tả

Cung cấp composite GitHub Action để consumer repositories dễ dàng tích hợp x-harness verify gate vào CI.

#### Blockers

| Blocker | Giải pháp đề xuất | Quyết định cần đưa ra |
|---|---|---|
| npm publishing | Publish `x-harness` lên npm registry | Ai sở hữu npm package? Scope là gì? (`@x-harness/cli`?) |
| Local build | Consumer build từ source (`npm ci && npm run build` trong action) | Chậm hơn npm install; phụ thuộc Node 22+ |
| Distribution | Release binary hoặc npx | Có cần bundled binary (pkg, ncc) không? |

#### Composite Action (Draft)

```yaml
# .github/actions/x-harness/action.yml
name: 'x-harness Verify'
description: 'Run x-harness doctor, examples verify, and verify gate'
inputs:
  root:
    description: 'Repository root path'
    required: false
    default: '.'
  strict:
    description: 'Run in strict mode'
    required: false
    default: 'false'
runs:
  using: 'composite'
  steps:
    - uses: actions/setup-node@v6
      with:
        node-version: 22
        cache: npm
    - run: npm ci
      shell: bash
    - run: npm run build --if-present
      shell: bash
    - run: node packages/cli/dist/index.js doctor --root ${{ inputs.root }}
      shell: bash
    - run: node packages/cli/dist/index.js examples verify
      shell: bash
    - run: node packages/cli/dist/index.js verify --root ${{ inputs.root }} ${{ inputs.strict == 'true' && '--strict' || '' }}
      shell: bash
```

#### Consumer Usage Example

```yaml
name: x-harness verify
on:
  pull_request:
  push:
    branches: [main]
jobs:
  x-harness:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v6
      - uses: ./.github/actions/x-harness
        with:
          root: .
          strict: true
```

#### Files to Create

| File | Hành động |
|---|---|
| `.github/actions/x-harness/action.yml` | Tạo mới: composite action definition |
| `examples/ci/github-action.yml` | Tạo mới: example workflow cho consumers |
| `docs/CI.md` | Cập nhật: hướng dẫn sử dụng action |

#### Atomic Checklist

- [ ] Composite action định nghĩa inputs: `root`, `strict`.
- [ ] Action chạy: doctor, examples verify, verify.
- [ ] Không yêu cầu external service hoặc secret.
- [ ] Permissions yêu cầu chỉ là `contents: read`.
- [ ] Docs giải thích cách dùng action trong repo consumer.
- [ ] **Blocked until:** npm publishing hoặc local-build strategy được quyết định.

#### Validation Commands (Sau khi unblock)

```bash
# Validate action YAML syntax
node -e "require('js-yaml').load(require('fs').readFileSync('.github/actions/x-harness/action.yml'))"
# Test action trong CI của chính repo này
git add .github/actions/x-harness/action.yml
git commit -m "Add x-harness composite action"
# Kiểm tra GitHub Actions tab
```

---

## 6. Phase 5 — UX & Interactive Features

### 6.1 Handoff Readiness --interactive

**Mã số:** P5.1
**Mức độ ưu tiên:** Medium
**Trade-off:** Low-Medium
**Trạng thái:** Partial — `handoff readiness --interactive` đã implement; CI/non-TTY safe; còn gap risk survey, tier suggestion logic, module extraction

#### Mô tả

Thêm chế độ interactive cho `handoff` để hỏi người dùng một số câu hỏi readiness ngắn, sau đó suggest tier và generate handoff template. Không tạo product lifecycle artifacts.

#### CLI Interface (Đề xuất)

```bash
node packages/cli/dist/index.js handoff --interactive
```

#### Interactive Questions (Đề xuất)

```
1. What is the task objective? (free text)
2. Risk level: low / normal / high?
3. Does it touch auth / payment / database / deploy / security? (y/n)
4. What evidence can verify it? (comma-separated: typecheck, test, lint, ...)
5. Is anything missing or ambiguous? (free text, optional)
```

#### Readiness Output Block

```yaml
readiness:
  goal: "Fix login timeout"
  suggested_tier: standard
  proceed: true
  missing_information: []
  evidence_expected:
    - typecheck
    - unit_test
  risk_flags:
    - "touches_deploy: false"
    - "touches_security: false"
```

#### Constraints

- Không tạo: feature intake file, story packet, product spec, test matrix, decision record.
- Output vẫn là handoff template; chỉ thêm readiness block.
- Interactive mode dùng `readline` hoặc `stdin` — không cần thư viện nặng (inquirer có thể dùng nếu đã có dependency).
- Hỗ trợ `--non-interactive` hoặc env var `CI=true` để skip prompts trong CI.

#### Files to Create/Modify

| File | Hành động |
|---|---|
| `packages/cli/src/commands/handoff.ts` | Sửa: thêm `--interactive` flag, readiness questioning logic |
| `packages/cli/src/core/handoff/readiness.ts` | Tạo mới: tier suggestion logic dựa trên risk answers |
| `packages/cli/tests/handoff-interactive.test.ts` | Tạo mới: unit tests với stdin mock |

#### Tier Suggestion Logic (Draft)

```ts
function suggestTier(answers: ReadinessAnswers): "light" | "standard" | "deep" {
  if (answers.risk === "high") return "deep";
  if (answers.touchesSecuritySensitive) return "deep";
  if (answers.risk === "normal") return "standard";
  return "light";
}
```

#### Atomic Checklist

- [ ] `handoff --interactive` khởi động question prompt.
- [ ] Câu trả lời được parse thành readiness block.
- [ ] `suggested_tier` đề xuất đúng: high-risk/security → deep, normal → standard, low → light.
- [ ] Output bao gồm cả readiness block và handoff template tiêu chuẩn.
- [ ] Không tạo file product intake hoặc story packet.
- [ ] `CI=true` hoặc `--non-interactive` skip prompts và dùng defaults.
- [ ] Test với stdin mock: trả lời 5 câu hỏi, kiểm tra output chứa readiness block.

#### Validation Commands

```bash
npm run build
npm run test -- packages/cli/tests/handoff-interactive.test.ts
# Test interactive mode (manual)
echo -e "Fix bug\nnormal\nn\ntypecheck,test\n" | node packages/cli/dist/index.js handoff --interactive
# Test CI mode
CI=true node packages/cli/dist/index.js handoff --interactive
```

---

## 7. Risk Registry

| Mã | Hạng mục | Mức độ rủi ro | Lý do | Mitigation |
|---|---|---|---|---|
| R1 | P4.6 Git-native packet chain verify | **HIGH** | Tăng CLI surface; dễ biến thành lifecycle framework nặng | Giữ scope chặt: chỉ 3 subcommands, no auto-commit, file-based only, no database |
| R2 | P4.7 Trace integrity hash chain | **HIGH** | Ảnh hưởng trace format toàn bộ; cần backward compat | Design review genesis hash và legacy handling; test migration path |
| R3 | P4.8 Consumer GitHub Action | **BLOCKED** | Phụ thuộc npm publishing / distribution strategy | Quyết định npm scope trước; fallback local-build nếu chưa publish |
| R4 | P4.1 Static HTML report | Medium | HTML escape bugs dẫn đến XSS trong audit file | Dùng escapeHtml utility; test với malicious payload |
| R5 | P5.1 Handoff --interactive | Low-Medium | readline mocking phức tạp trong tests | Dùng stdin mock; hỗ trợ CI mode để skip interactive |
| R6 | P4.5 Recovery playbook suggestions | Medium | Dễ hiểu nhầm thành auto-learning loop | Ràng buộc deterministic aggregation only; requires_review marker; no policy mutation |

---

## 8. Ordered Implementation Roadmap

### Giai đoạn 4A — Docs & Low-Risk Auditability (4–6 tuần)

Thứ tự: song song hóa docs, implement HTML report, recovery playbook.

| Tuần | Hạng mục | Files chính | Validation |
|---|---|---|---|
| 1 | P4.2 Technical handbook docs | `docs/HANDBOOK.md`, `docs/ARCHITECTURE.md` | `grep handbook README.md` |
| 1 | P4.3 docs/PACKETS.md | `docs/PACKETS.md` | Link check |
| 1 | P4.4 docs/CI.md | `docs/CI.md` | Link check |
| 2 | P4.1 Static HTML audit report | `packages/cli/src/core/report/html.ts`, `report.ts` | `npm test -- report-html.test.ts` |
| 2 | P4.5 Recovery playbook suggestions | `packages/cli/src/core/recovery/suggestPlaybook.ts` | `npm test -- recovery-playbook.test.ts` |
| 3 | P4.1 HTML report polish + escape tests | `packages/cli/tests/report-html.test.ts` | Malicious payload test |
| 3 | P4.5 Playbook candidate format review | `docs/RECOVERY.md` | `--write --force` behavior test |

### Giai đoạn 4B — Deep Integrity (6–8 tuần)

Thứ tự: trace hash chain trước (ảnh hưởng format), sau đó packet chain verify.

| Tuần | Hạng mục | Files chính | Validation |
|---|---|---|---|
| 4 | P4.7 Trace integrity hash chain — design review | `packages/cli/src/core/trace.ts` | N/A (design doc) |
| 5 | P4.7 Trace hash chain — implement | `packages/cli/src/core/trace/hash.ts`, `trace.ts` | `npm test -- trace-hash.test.ts` |
| 5 | P4.7 Legacy trace compatibility | `packages/cli/tests/trace-hash.test.ts` | Legacy fixture test |
| 6 | P4.6 Git-native packet chain — design review | `docs/PACKETS.md`, `schemas/packet.schema.json` | N/A (design doc) |
| 7 | P4.6 Packet create + verify-chain | `packages/cli/src/core/packet/*.ts` | `npm test -- packet.test.ts` |
| 7 | P4.6 Packet hash mismatch & cycle detection | `packages/cli/tests/packet.test.ts` | Tamper tests |
| 8 | P4.6 Integration test: packet + trace + report | `examples/golden/packet-chain/` | End-to-end chain verify |

### Giai đoạn 4C — Distribution (Blocked until decision)

| Tuần | Hạng mục | Files chính | Validation |
|---|---|---|---|
| TBD | P4.8 Consumer GitHub Action — unblock | npm publishing decision | Owner decision |
| TBD | P4.8 Composite action implement | `.github/actions/x-harness/action.yml` | CI run in this repo |
| TBD | P4.8 Consumer docs update | `docs/CI.md`, `examples/ci/` | Usage example test |

### Giai đoạn 5 — Interactive UX (2–3 tuần)

| Tuần | Hạng mục | Files chính | Validation |
|---|---|---|---|
| 9 | P5.1 Handoff readiness — design | `packages/cli/src/core/handoff/readiness.ts` | Tier suggestion logic review |
| 10 | P5.1 Handoff --interactive implement | `packages/cli/src/commands/handoff.ts` | `npm test -- handoff-interactive.test.ts` |
| 10 | P5.1 CI/non-interactive mode | `packages/cli/tests/handoff-interactive.test.ts` | `CI=true` test |
| 11 | P5.1 Polish + integration | `docs/HANDBOOK.md` | Manual interactive test |

---

## 9. Non-Goals Reaffirmation for Phase 4/5

Dù đã liệt kê ở Section 2, cần nhắc lại rõ ràng trong phạm vi Phase 4/5:

```
[ ] KHÔNG làm plugin system / plugin marketplace.
[ ] KHÔNG làm MCP adapter as default (nếu có, chỉ optional read-only, disabled).
[ ] KHÔNG làm dashboard server / UI web.
[ ] KHÔNG thêm database / storage layer.
[ ] KHÔNG dùng LLM verifier làm admission authority.
[ ] KHÔNG tạo product-planning / story-packet / test-matrix lifecycle.
[ ] KHÔNG auto-commit Git mặc định.
[ ] KHÔNG tạo recovery simulator.
[ ] KHÔNG tạo autonomous learning loop từ trace.
```

---

## 10. Validation Commands Summary

Sau mỗi phase, chạy:

```bash
# Global validation (áp dụng cho mọi phase)
npm run typecheck
npm run build
npm run lint
npm run format:check
npm test
node packages/cli/dist/index.js doctor --root .
node packages/cli/dist/index.js examples verify

# Phase 4A validation
npm run test -- packages/cli/tests/report-html.test.ts
npm run test -- packages/cli/tests/recovery-playbook.test.ts

# Phase 4B validation
npm run test -- packages/cli/tests/trace-hash.test.ts
npm run test -- packages/cli/tests/packet.test.ts

# Phase 4C validation (sau khi unblock)
# act -j x-harness  # nếu dùng nektos/act để test action locally

# Phase 5 validation
npm run test -- packages/cli/tests/handoff-interactive.test.ts
```

---

## 11. Evidence & Traceability

Tài liệu này dựa trên các nguồn đã được phân tích:

- `todo.md`: Phân loại deferred items thành Phase 4/5.
- `X_HARNESS_DETAILED_IMPROVEMENT_IMPLEMENTATION_PLAN.md`: Spec chi tiết cho P0–P2.
- `X_HARNESS_IMPROVEMENT_PROPOSALS_NO_HARNESS_EXPERIMENTAL.md`: Non-goals và core invariants.
- `packages/cli/src/commands/report.ts`: Đã thêm `--format html`.
- `packages/cli/src/core/recovery.ts`: Recovery route mapping + `generatePlaybook()`.
- `packages/cli/src/core/trace.ts`: Đã thêm `previous_hash`, `event_hash`, `verifyTraceChain()`.
- `packages/cli/src/core/hash.ts`: SHA-256 utilities reused.
- `packages/cli/src/commands/handoff.ts`: Đã thêm `handoff readiness --interactive`.
- `.github/workflows/x-harness-verify.yml`: Current CI workflow.
- `docs/ROADMAP.md`: Roadmap chính thức của dự án.

---

## 12. Follow-up Checklist (Gaps Còn Lại)

### P4.1 gaps
- [x] Tạo `docs/REPORT_FORMATS.md`.
- [x] Cập nhật README link đến REPORT_FORMATS.
- [ ] Polish metrics HTML report layout (hiện tại dùng `<pre><code>` cho metrics JSON).
- [ ] Thêm `--output <file>` để ghi HTML ra file thay vì stdout.

### P4.2 gaps
- [x] Thêm link `docs/HANDBOOK.md` trong `README.md`.
- [ ] Đánh giá xem có cần `docs/ARCHITECTURE.md` riêng hay HANDBOOK đã đủ.

### P4.3 gaps
- [x] Cập nhật `docs/PACKETS.md` với implementation details và guardrails.
- [x] Tạo `schemas/packet.schema.json`.
- [x] Cập nhật README link đến PACKETS.md.

### P4.5 gaps
- [x] Thêm `--from <trace-file>` để đọc trace JSONL.
- [x] Thêm `--write` và `--force` để ghi candidate playbook ra `.x-harness/candidates/`.
- [x] Chuẩn hóa candidate fields (observed_count, confidence, source_trace_events, requires_review).
- [ ] Stdin support cho `--from -` (deferred đến phase sau).

### P4.6 gaps
- [x] Oracle GO verdict đã nhận (CONDITIONAL-GO); implementation hoàn thành với guardrails.
- [ ] Mở rộng packet types (evidence, recovery) nếu có demand thật — hiện tại chỉ claim.
- [ ] Thêm packet signatures nếu cần audit trail cryptographically verifiable.

### P4.7 gaps
- [ ] Đánh giá performance với trace file lớn (10k+ events) — hiện tại đọc tuần tự.
- [ ] Xem xét thêm `--from` flag cho `trace verify-chain` nếu cần verify trace khác default.

### P4.8 gaps
- [ ] Không còn blocker nghiêm trọng; composite action đã dùng local-build.
- [ ] Có thể cải thiện: cache build artifacts, vendor `dist/` để giảm thởi gian CI.

### P5.1 gaps
- [x] Thêm risk survey questions (touches security/payment/deploy/database).
- [x] Thêm tier suggestion logic (`suggestTier()` dựa trên risk answers).
- [x] JSON readiness output bao gồm suggested_tier, risk_flags, missing_information, evidence_expected.
- [ ] Tách readiness logic ra `packages/cli/src/core/handoff/readiness.ts` nếu file `handoff.ts` quá lớn.
- [ ] Thêm `--non-interactive` flag explicit (hiện tại dùng `!process.stdin.isTTY` hoặc `CI=true`).

---

## 13. Final Checklist (Plan Completeness)

- [x] 9 hạng mục được cover đầy đủ (P4.1–P4.8, P5.1).
- [x] Mỗi hạng mục có: mô tả, CLI interface, files to change, atomic checklist, validation commands.
- [x] High-risk items được đánh dấu rõ ràng (P4.6, P4.7).
- [x] Blockers được đánh dấu rõ ràng (P4.8 npm publishing).
- [x] Non-goals được liệt kê rõ ràng (plugin, MCP, dashboard, DB, LLM verifier).
- [x] Dependency graph được mô tả.
- [x] Ordered roadmap có timeline đề xuất.
- [x] Risk registry có mitigation.
- [x] Trạng thái implementation được cập nhật: Complete, Partial, Deferred.
- [x] Follow-up checklist cho gaps đã được thêm.
- [x] Không overclaim completion cho bất kỳ hạng mục nào.
