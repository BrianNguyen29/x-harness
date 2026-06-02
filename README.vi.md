# ⚡ x-harness

> **Hoàn thành phải được xét duyệt, không phải tự xưng.**
> Một harness xác minh nhẹ, ưu tiên tệp tin, dành cho quy trình làm việc với AI agent.

[![Verify CI](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml/badge.svg)](https://github.com/BrianNguyen29/x-harness/actions/workflows/x-harness-verify.yml)
[![CodeQL](https://github.com/BrianNguyen29/x-harness/actions/workflows/codeql.yml/badge.svg)](https://github.com/BrianNguyen29/x-harness/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/BrianNguyen29/x-harness/badge)](https://scorecard.dev/viewer/?uri=github.com/BrianNguyen29/x-harness)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Node.js ≥ 20](https://img.shields.io/badge/Node.js-%3E%3D20-blue.svg)](packages/cli/package.json)
[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](go.mod)

[Tiếng Việt](README.vi.md) | [English](README.md)

---

## x-harness là gì?

`x-harness` là một **harness xác minh** nhỏ gọn và có nguyên tắc rõ ràng dành cho các AI coding agent. Nó **không** chạy agent của bạn, không thay thế CI, và không đảm bảo rằng code là đúng. Nó làm đúng một việc có giới hạn:

> Biến lời tuyên bố "tôi đã xong" của một AI agent thành **một quyết định xét duyệt có thể kiểm tra được** — `accepted` (chấp nhận) hoặc `withheld` (giữ lại) — theo chính sách của repository.

Nó hoạt động **cục bộ**, **offline**, và **ưu tiên tệp tin**. Không cần daemon, không cần database, không cần server, không cần MCP, không cần gọi LLM, không cần thông tin đăng nhập mạng. Nguồn chân lý nằm ngay trong các tệp tin trong repository của bạn: schema, policy, template, và completion card.

### Khác với các công cụ hiện có ở điểm nào?

| Mối quan tâm | Công cụ AI thông thường | x-harness |
| :-- | :-- | :-- |
| Ai quyết định "xong"? | Chính agent đó | **Read-only verification gate** đặt trong repo này |
| Trạng thái lưu ở đâu? | Một server từ xa / SaaS | **Tệp tin** trong repository của bạn |
| Có cần runtime không? | Thường cần daemon, MCP, hoặc dịch vụ cloud | **Không.** Chỉ một binary tĩnh |
| Quyết định có kiểm tra được không? | Ẩn bên trong model hoặc dashboard | **Completion card có cấu trúc** + trace JSONL |
| Việc xác minh có đáng tin? | Lẫn với phần sinh nội dung | **Tách biệt.** `verify` không sửa sản phẩm |
| Có fail-closed không? | Thường là "cố gắng tốt nhất" | **Có.** Bất kỳ kết quả nào khác `success` đều là `withheld` |

Nói ngắn gọn: `x-harness` **không phải** là một agent runtime, issue tracker, hệ thống lập kế hoạch, LLM gateway, hay deployment engine. Nó là một **cổng chính sách** quyết định xem tuyên bố "đã xong" của agent có nên được chấp nhận hay không.

---

## Ý tưởng cốt lõi trong 60 giây

```text
   Agent viết code
        │
        ▼
   Agent viết "completion card" (một tệp YAML nhỏ)
        │
        ▼
   xh check --card completion-card.yaml
        │
        ▼
   Read-only verification gate đánh giá card
   dựa trên schema + policy (không sửa source)
        │
        ▼
   ┌───────────────────────────────────────────┐
   │  acceptance_status: accepted   → exit 0  │   ✅ xong
   │  acceptance_status: withheld   → exit 1  │   🚧 chưa xong, có recovery path
   └───────────────────────────────────────────┘
```

Verifier là **read-only**. Nó chỉ đọc card và bằng chứng của bạn; nó không bao giờ sửa source để "vá" kết quả khi đang kiểm tra. Agent phải tự tạo ra một card đạt yêu cầu.

---

## Khái niệm cho người mới (đọc trước tiên)

| Thuật ngữ | Ý nghĩa trong x-harness |
| :-- | :-- |
| **Completion card** | Một tệp YAML (ví dụ `completion-card.yaml`) nơi agent ghi lại những gì nó claim đã làm, kèm bằng chứng gì. |
| **Verify gate** | Lệnh `xh check` / `xh verify`. Nó chạy logic admission read-only. |
| **Accepted** | Xác minh đạt. Exit code `0`. Công việc chính thức hoàn tất. |
| **Withheld** | Mọi kết quả không thành công (`failed`, `blocked`, `skipped`, `timeout`, `error`). Exit code `1`. |
| **Tier** | Một trong `light`, `standard`, hoặc `deep`. Quy định lượng bằng chứng cần có. |
| **PGV** | Pre-Gate Validation. **Chỉ mang tính tham khảo.** Có thể gợi ý, nhưng không bao giờ cấp quyền admission. |
| **Adapter** | Một bộ tệp quy ước nhỏ (ví dụ `CLAUDE.md`, `.cursor/rules/x-harness.mdc`) cho một nền tảng agent cụ thể. |

---

## Cài đặt (từ source checkout)

`x-harness` cung cấp **native Go CLI** (khuyến nghị) và **TypeScript compatibility CLI** (chỉ dùng cho source-checkout fallback).

### Phương án A — Native Go CLI (khuyến nghị)

```bash
# Build một binary tĩnh duy nhất
go build ./cmd/x-harness

# Kiểm tra
./x-harness --version
```

> Yêu cầu **Go 1.22+**. Binary `./x-harness` thu được là tự chứa, không phụ thuộc ngoài.

### Phương án B — TypeScript compatibility CLI (source checkout)

Chỉ dùng khi bạn muốn chạy parity baseline từ source:

```bash
npm install
npm run build
node packages/cli/dist/index.js --version
```

> Yêu cầu **Node.js ≥ 20**. Gói `x-harness` trên npm hiện là một wrapper chỉ chứa Go; fallback Node chỉ chạy được khi bạn cài từ source và tồn tại thư mục `dist/`.

### Binary đóng gói sẵn từ release

Các native binary cho `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, và `windows/arm64` được đính kèm trong từng [GitHub release](https://github.com/BrianNguyen29/x-harness/releases). Tải binary phù hợp với nền tảng của bạn, đặt nó vào `PATH`, và đổi tên thành `xh` (hoặc `x-harness`).

### Trình quản lý gói (Windows / macOS / Linux)

Manifest cho Scoop và Homebrew được sinh tự động từ [`packaging/`](packaging) và [`scripts/`](scripts) khi release. Chúng sẽ xuất hiện trong các bucket tương ứng khi các bucket đó được công bố; cho đến lúc đó, vui lòng dùng binary đóng gói sẵn ở trên hoặc build từ source.

---

## Bắt đầu nhanh (5 phút)

### 1. Kiểm tra sức khỏe workspace

```bash
./x-harness doctor
```

Lệnh này xác nhận rằng schema, policy, template, và liên kết adapter đều tồn tại và nhất quán. Tìm `"healthy": true` trong đầu ra JSON (`./x-harness doctor --json`).

### 2. Chạy lần xác minh đầu tiên

Repo có sẵn các **golden example** — các kịch bản tham chiếu đã được xác thực trước. Thử một kịch bản biết chắc là đạt:

```bash
xh check --card examples/golden/regression/success-light/completion-card.yaml
```

Đầu ra mong đợi:

```yaml
outcome: success
acceptance_status: accepted
checks: 2 passed, 0 failed
```

Bây giờ thử một kịch bản thiếu bằng chứng bắt buộc:

```bash
xh check --card examples/golden/regression/blocked-missing-evidence/completion-card.yaml
# exit code 1
```

Đầu ra mong đợi:

```yaml
outcome: failed
acceptance_status: withheld
checks: 0 passed, 5 failed
```

Công việc bị **withheld**, không bị âm thầm đánh dấu là xong. Đây là cơ chế *fail-closed* mặc định.

### 3. Khởi tạo một workspace mới

Để bắt đầu dùng `x-harness` trong một dự án khác, chạy `init` tại thư mục gốc của dự án đó:

```bash
xh init --minimal        # mặc định: contracts, templates, policies
# xh init --standard     # thêm schema và ví dụ solo-agent
# xh init --full         # thêm ví dụ multi-agent, adapter, và GitHub Action
```

`init` sẽ dừng với tóm tắt blocked nếu phát hiện tệp xung đột; chỉ chạy lại với `--force` khi bạn thực sự muốn ghi đè các tệp đó.

### 4. Gửi một task handoff

```bash
xh handoff standard --title "Sửa lỗi canh lề nút checkout"
```

Lệnh này sinh một tệp Markdown có cấu trúc, kèm tập tệp tường minh, checklist bằng chứng, và định nghĩa rollback cho tier `standard`.

---

## Chín hành động cho người mới

Đây là những hành động bạn sẽ dùng trong 95% trường hợp. Danh sách đầy đủ nằm ở `xh --help-all`.

| Hành động | Mô tả |
| :-- | :-- |
| **`check`** | Chạy read-only verification gate trên một completion card. |
| **`prepare`** | Kiểm tra workspace đã sẵn sàng cho một handoff agent hay chưa. |
| **`recover`** | Sinh recovery playbook từ một thông báo lỗi hoặc trace. |
| **`doctor`** | Xác nhận sức khỏe workspace (schema, policy, liên kết, độ tươi). |
| **`actions`** | Liệt kê các hành động cho người mới (danh sách này). |
| **`status`** | Hiển thị tóm tắt trace hoặc metrics của card. |
| **`reset`** | Dọn trạng thái harness đã sinh (cần `--confirm`). |
| **`init`** | Cài đặt tài sản harness vào một workspace mục tiêu. |
| **`add`** | Thêm tệp trợ giúp metadata (claim, evidence, hoặc completion card). |

> **Terminal**: `xh <action>` (ví dụ `xh check`)
> **Chat với agent**: `/xh <action>` (ví dụ `/xh check`)

Các lệnh nâng cao (`handoff`, `verify`, `report`, `packet`, `conformance`, `benchmark`, `contract`, `release`, …) được mô tả trong [`docs/`](docs).

---

## Các tier handoff chuẩn

Việc phân công task chỉ dùng **đúng ba tier** này. Các nhãn `small`, `medium`, `large` không được phép xuất hiện trong runtime handoff.

| Tier | Khi nào dùng | Sàn bằng chứng tối thiểu | Cần phê duyệt? |
| :-- | :-- | :-- | :-- |
| **`light`** | Công việc nhỏ, ít ceremony (1–3 tệp, gần như read-only). | `files_changed` + (`command_evidence` _hoặc_ `manual_rationale`). | Không bắt buộc |
| **`standard`** | Công việc nhiều bước thông thường, tổng hợp có giới hạn. | `files_changed` + `command_evidence` + `done_checklist` + `prediction`. | Không bắt buộc |
| **`deep`** | Công việc rủi ro cao: thay đổi kiến trúc, migration, nhiều phụ thuộc. | Tất cả những gì `standard` yêu cầu, **cộng thêm** `evidence_scope`, `untested_regions`, `remaining_risks`, `execution_controls`, `rollback_policy`, `state.read_set`, `state.write_set`. | Bắt buộc |

Xem [`docs/ADMISSION_POLICY.md`](docs/ADMISSION_POLICY.md) để biết đầy đủ quy tắc.

---

## Luồng chạy của một verify

```text
┌──────────────┐
│ Agent worker │  viết code + viết completion card
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ xh check ... │  nạp schema, nạp policy, đánh giá sàn bằng chứng
└──────┬───────┘
       │  read-only
       ▼
┌──────────────┐
│   Outcome    ├─── accepted   (exit 0)
│              ├─── withheld   (exit 1, có recovery routing)
└──────────────┘
```

Các verify stage tùy chọn, opt-in (mặc định tắt):

- `xh verify --contract-oracles` — assertion theo rule ở mức dòng (`grep_rules`, `dependency_rules`).
- `xh verify --context-floor` — kiểm tra sự tồn tại tối thiểu của file/tham chiếu.
- `xh verify --strict` — chế độ strict-schema cho đầu ra `withheld_reason`.
- `xh verify --mutation-guard` — phát hiện mọi thay đổi source phía verifier.

Khi verify thất bại, engine sinh một đối tượng recovery có cấu trúc, định tuyến công việc về đúng chủ thể (ví dụ `evidence_missing` → `implementation-worker`, `approval_missing` → `user`).

---

## Adapter cho từng nền tảng

`x-harness` **không phụ thuộc nền tảng**. Hãy chọn adapter phù hợp với cách bạn đang làm việc:

| Adapter | Khi nào dùng | Tệp chính |
| :-- | :-- | :-- |
| [Generic](adapters/generic) | Bạn muốn quy ước Markdown thuần, không bị khóa nền tảng. | `AGENTS.md` |
| [Claude Code](adapters/claude-code) | Bạn dùng Claude Code. | `CLAUDE.md`, agent worker / verifier, skills |
| [Cursor](adapters/cursor) | Bạn dùng Cursor. | `.cursor/rules/x-harness.mdc` |
| [OpenCode](adapters/opencode) | Bạn dùng OpenCode. | `verify-agent.md`, agent worker / verifier |
| [Antigravity](adapters/antigravity) | Bạn dùng Antigravity. | rules + workflows trong `rules/` và `workflows/` |

Bạn chỉ cần **một**. Adapter là lớp vỏ mỏng quanh cùng một CLI; chúng không phân nhánh contract.

---

## Tài liệu

| Tài liệu | Nội dung |
| :-- | :-- |
| [`docs/QUICKSTART.md`](docs/QUICKSTART.md) | Hướng dẫn cài đặt cục bộ từng bước và lần verify đầu tiên. |
| [`docs/FAQ.md`](docs/FAQ.md) | Câu hỏi thường gặp (Go vs TS, có dùng LLM không, …). |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Mô hình phân lớp, chu trình xác thực, ghi chú thiết kế. |
| [`docs/VERIFY_GATE.md`](docs/VERIFY_GATE.md) | Cách read-only verify gate hoạt động. |
| [`docs/ADMISSION_POLICY.md`](docs/ADMISSION_POLICY.md) | Quy tắc admission fail-closed và sàn bằng chứng. |
| [`docs/SCHEMAS.md`](docs/SCHEMAS.md) | Danh mục JSON schema. |
| [`docs/RECOVERY.md`](docs/RECOVERY.md) | Routing recovery và sinh playbook. |
| [`docs/ADAPTERS.md`](docs/ADAPTERS.md) | Hướng dẫn adapter đầy đủ và tham chiếu chọn tier. |
| [`docs/CONFORMANCE_STRICT_PROFILE.md`](docs/CONFORMANCE_STRICT_PROFILE.md) | Quy tắc strict-profile và tiêu chí xác minh. |
| [`docs/TYPESCRIPT_MAINTENANCE.md`](docs/TYPESCRIPT_MAINTENANCE.md) | Chính sách bảo trì cho fallback TypeScript. |
| [`docs/CI.md`](docs/CI.md) | Tích hợp CI và cổng dual-run. |
| [`docs/RELEASE_SECURITY.md`](docs/RELEASE_SECURITY.md) | Ký release, SBOM, và provenance. |
| [`docs/RELEASE_CANDIDATE.md`](docs/RELEASE_CANDIDATE.md) | Checklist release-candidate. |
| [`docs/PACKETS.md`](docs/PACKETS.md) | Claim packet bất biến và tính toàn vẹn chuỗi. |
| [`docs/REPORT_FORMATS.md`](docs/REPORT_FORMATS.md) | Định dạng đầu ra report (Markdown, JSON, HTML). |

Hợp đồng có thẩm quyền nằm ở [`X_HARNESS.md`](X_HARNESS.md).

---

## Tình trạng dự án

- **Phiên bản**: `0.99.0-rc1` (release candidate). CLI đã đủ tính năng cho hợp đồng v0.x, nhưng dự án vẫn **tiền-1.0**. Hãy ghim phiên bản của bạn và lường trước một số thay đổi nhỏ về contract trước khi đạt `1.0`.
- **Runtime chính**: Go CLI (khuyến nghị). TypeScript CLI chỉ còn là source-checkout compatibility baseline, không còn được đóng gói trong gói npm đã phát hành.
- **Không có tuyên bố production**: Một lần `xh check` đạt **không phải** là bảo đảm đúng đắn. Nó có nghĩa là card của bạn khớp với policy. Xem [`docs/VERIFY_GATE.md`](docs/VERIFY_GATE.md) để biết gate kiểm tra và không kiểm tra những gì.

---

## Đóng góp

Hoan nghênh mọi đóng góp. Vui lòng đọc [`CONTRIBUTING.md`](CONTRIBUTING.md) trước.

Các thay đổi nhạy cảm với harness (admission policy, schema, template, CLI verify, adapter, skills) phải đính kèm một bản [`templates/HARNESS_CHANGE_CONTRACT.md`](templates/HARNESS_CHANGE_CONTRACT.md) đã hoàn thiện và phải pass `./x-harness doctor`, `./x-harness examples verify`, và `./x-harness benchmark --filter adversarial --gate` ở máy local.

Mọi người đóng góp được kỳ vọng tuân thủ [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).

---

## Bảo mật

Vui lòng **không** mở issue công khai cho các lỗ hổng nghi ngờ. Dùng [GitHub private vulnerability reporting](https://github.com/BrianNguyen29/x-harness/security/advisories/new) hoặc liên hệ maintainer qua kênh riêng. Xem [`SECURITY.md`](SECURITY.md) để biết quy trình tiết lộ đầy đủ và các phiên bản được hỗ trợ.

---

## Giấy phép

[MIT](LICENSE) — Bản quyền (c) 2026 Brian Nguyen.
