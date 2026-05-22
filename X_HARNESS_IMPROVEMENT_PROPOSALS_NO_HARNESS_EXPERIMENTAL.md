# x-harness Improvement Proposal — Lean Formal Core, No harness-experimental Carryover

## 0. Purpose

Tài liệu này là bản đề xuất cải thiện `x-harness` theo hướng vận hành thực tế hơn cho workflow coding-agent / multi-agent.

Mục tiêu không phải là biến `x-harness` thành framework lớn. Mục tiêu là làm cho completion claim khó giả hơn, dễ kiểm chứng hơn, dễ audit hơn, và an toàn hơn khi phải withheld.

Định vị giữ nguyên:

```txt
x-harness = lightweight, offline-first, verify-gated completion/admission harness
```

Luồng cốt lõi:

```txt
agent work
  -> completion claim
  -> evidence
  -> read-only verify gate
  -> accepted or withheld
  -> trace / recovery / report
```

## 1. Explicit non-goals

Không triển khai các hướng sau trong core:

```txt
- Không biến x-harness thành agent runtime.
- Không biến x-harness thành product planning harness.
- Không tạo feature intake system.
- Không tạo story packet lifecycle.
- Không tạo product spec / product docs generator.
- Không tạo TEST_MATRIX kiểu planning artifact.
- Không tạo decision record framework mặc định.
- Không tạo workflow engine.
- Không tạo dashboard server.
- Không tạo storage/database layer.
- Không tạo plugin marketplace.
- Không dùng LLM verifier làm admission authority mặc định.
- Không yêu cầu daemon, MCP, database, cloud service, hoặc external service.
```

Các cải thiện được phép chỉ nằm trong phạm vi:

```txt
- context envelope ngắn cho agent;
- admission policy;
- completion-card schema;
- deterministic verify;
- read-only enforcement;
- evidence hardening;
- recovery routing;
- traceability;
- static report;
- Git-native packet lineage;
- docs kỹ thuật cho chính x-harness.
```

## 2. Core invariants không được phá vỡ

Các rule sau là bất biến:

```txt
1. Completion is admitted, not claimed.
2. Worker may propose completion, but cannot self-admit.
3. Verifier is read-only.
4. outcome=success là outcome duy nhất có thể accepted.
5. failed / blocked / skipped / timeout / error đều là withheld.
6. PGV hoặc advisory checks chỉ advisory-only, không thay admission verifier.
7. Deep mode là opt-in hoặc risk-triggered, không phải mặc định.
8. Context và skills phải load-on-demand.
9. Git là database mặc định.
10. YAML là protocol mặc định.
11. CLI là interface mặc định.
12. Markdown là handbook mặc định.
13. HTML report nếu có phải static và optional.
14. Không thêm default process weight nếu không tăng admission safety rõ ràng.
```

Accepted chỉ hợp lệ khi:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

Không được xem các tín hiệu sau là accepted:

```txt
fix_status: fixed
verification.status: passed
tests passed
agent confidence: HIGH
PGV says okay
context_acknowledged: true
human says looks good
```

Các tín hiệu đó có thể là evidence hoặc advisory, nhưng không phải admission result.

## 3. Design principle

Mỗi feature mới phải trả lời được câu hỏi:

```txt
Does this make completion harder to fake, easier to verify, and safer to withhold without adding default process weight?
```

Nếu câu trả lời là không, không đưa vào core.

Ưu tiên thiết kế:

```txt
formal depth, lean execution
```

Diễn giải:

```txt
- Acceptance predicate có thể formal, nhưng implementation phải tiered.
- Packetized state có thể có lineage, nhưng storage là file + Git.
- Recovery phải chặt, nhưng test bằng deterministic golden cases.
- Audit phải rõ, nhưng report là static HTML, không dashboard.
- Context phải fresh, nhưng ngắn và generated on demand.
- Policy phải machine-checkable, nhưng không thành workflow language lớn.
```

## 4. Roadmap tổng thể

### P0 — Làm ngay

P0 tập trung vào giảm sai protocol, giảm stale context, giảm overclaim, và thêm hard guard cho read-only verify.

```txt
P0.1  Correct README/paper wording and remove unsupported claims.
P0.2  Add x-harness context command.
P0.3  Generate/refresh AGENTS.md as x-harness agent contract with context hash.
P0.4  Auto-inject bounded context header into handoff.
P0.5  Add doctor freshness checks for AGENTS.md/context hash.
P0.6  Add read-only filesystem mutation guard for verify.
P0.7  Remove/defer redundant explain --for-agent.
P0.8  Do not add handoff --format prompt.
P0.9  Add minimal tests for context, handoff context injection, and read-only guard.
```

### P1 — Sau khi P0 ổn

P1 tập trung vào admission correctness, recovery determinism, evidence hardening, và policy-code drift control.

```txt
P1.1  Predicate tiering: Required / Conditional / Advisory.
P1.2  Recovery golden test suite.
P1.3  Evidence artifact hardening for standard/deep.
P1.4  Policy-as-source-of-truth or policy-code drift guard.
P1.5  context_acknowledged advisory check.
P1.6  Git context inheritance, opt-in only.
P1.7  GitHub Action/docs for consumer repositories.
P1.8  Handoff readiness questions, CLI-only, no product lifecycle artifacts.
```

### P2 — Khi repo đã ổn định

P2 tập trung vào auditability và reproducibility, không thêm runtime nặng.

```txt
P2.1  Git-native packet chain verify.
P2.2  Static single-file HTML audit report.
P2.3  Technical handbook and architecture docs.
P2.4  Static recovery playbook suggestions from trace.jsonl.
P2.5  Trace integrity hash chain.
```

### P3 — Chỉ làm nếu có demand thật

```txt
P3.1  YAML custom checks before plugin system.
P3.2  File-based local plugin system, disabled by default.
P3.3  Official plugins only after API stabilizes.
```

## 5. P0.1 — Correct wording and remove unsupported claims

### Problem

Các claim kiểu “guarantees correctness”, “truly done”, “100% recovery”, “token guaranteed”, hoặc score không có benchmark làm giảm độ tin cậy. `x-harness` nên claim đúng phạm vi: nó giúp admission chặt hơn, không chứng minh code đúng tuyệt đối.

### Required wording

Tránh:

```txt
ensure tasks are truly done
guarantees correctness
prevents hallucination
proves production reliability
recovery completeness 100%
<2000 tokens guaranteed
89/100 measured score
```

Dùng:

```txt
helps ensure completion claims are admitted under repository policy
makes completion claims more auditable
fails closed when required evidence is missing
keeps verifier read-only
provides deterministic local admission checks
context header target: <= 200 tokens
recovery coverage is based on golden cases, not production recovery effectiveness
```

### Files to inspect/modify

```txt
README.md
X_HARNESS.md
AGENTS.md
docs/*.md
packages/cli/src/commands/report.ts
packages/cli/src/commands/doctor.ts
```

### Acceptance criteria

```txt
[ ] No correctness guarantee language remains.
[ ] No unsupported numeric score is presented as measured result.
[ ] Token budgets are framed as targets, not guarantees.
[ ] README states x-harness is an admission harness, not a proof of correctness.
[ ] Paper reference uses arXiv:2605.18747 consistently if mentioned.
```

## 6. P0.2 — Add `x-harness context`

### Problem

Agent cần một context envelope ngắn và fresh. Bắt agent đọc toàn bộ README, templates, schema và policy sẽ tăng noise. Nếu không có context command, handoff dễ thiếu accepted/withheld semantics.

### Command surface

```bash
x-harness context
x-harness context --verbose
x-harness context --json
x-harness context --refresh
```

Không thêm `--watch` trong P0.

### Default output

Target: 100–200 tokens.

```txt
# x-harness context
Core: completion is admitted, not claimed.
Tier: use light | standard | deep; choose smallest sufficient tier.
Verify: create completion-card.yaml, then run npx x-harness verify.
Accepted: only outcome=success and acceptance_status=accepted.
Withheld: failed, blocked, skipped, timeout, error.
Verifier: read-only.
PGV: advisory-only.
```

### JSON output

```json
{
  "core_rule": "completion is admitted, not claimed",
  "tiers": ["light", "standard", "deep"],
  "accepted_if": {
    "outcome": "success",
    "acceptance_status": "accepted"
  },
  "withheld_outcomes": ["failed", "blocked", "skipped", "timeout", "error"],
  "verifier_mode": "read-only",
  "pgv": "advisory-only",
  "context_hash": "sha256:..."
}
```

### Verbose output

Có thể thêm:

```txt
- completion-card minimal fields;
- evidence expectations by tier;
- blocked vs failed explanation;
- recovery next action example;
- read-only verifier reminder;
- advisory vs required predicate explanation.
```

Nhưng vẫn không biến thành manual dài.

### Files to create/modify

```txt
packages/cli/src/commands/context.ts
packages/cli/src/index.ts
packages/cli/src/core/context/generator.ts
packages/cli/src/core/context/hash.ts
docs/CONTEXT_POLICY.md
tests/context.test.ts
```

### Implementation sketch

```ts
export interface ContextEnvelope {
  core_rule: string;
  tiers: string[];
  accepted_if: { outcome: "success"; acceptance_status: "accepted" };
  withheld_outcomes: string[];
  verifier_mode: "read-only";
  pgv: "advisory-only";
  context_hash: string;
}

export function generateContextEnvelope(mode: "short" | "verbose"): string { ... }
export function generateContextJson(): ContextEnvelope { ... }
```

### Acceptance criteria

```txt
[ ] x-harness context works.
[ ] x-harness context --verbose works.
[ ] x-harness context --json returns valid JSON.
[ ] x-harness context output is short by default.
[ ] No daemon/service/watch mode is introduced.
```

## 7. P0.3 — Generate/refresh AGENTS.md with context hash

### Problem

Agent thường đọc `AGENTS.md` đầu tiên. Nếu file này thiếu hoặc stale sau khi policy/schema/template đổi, agent có thể làm sai admission protocol.

### Goal

`x-harness init` tạo hoặc quản lý một block `AGENTS.md` ngắn, có freshness hash.

### Required header

```md
<!-- x-harness-context-hash: sha256:<hash> -->
<!-- generated-by: x-harness -->
<!-- generated-at: <ISO timestamp> -->
```

### Hash inputs

Core hash nên tính từ:

```txt
policies/admission.yaml
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md
X_HARNESS.md
```

Nếu repo có adapter-specific contracts, có thể thêm optional adapter hash:

```md
<!-- x-harness-adapter-context-hash: sha256:<hash> -->
```

Adapter hash chỉ tính khi adapter được enable.

### Existing AGENTS.md behavior

```txt
default:
  do not overwrite silently

--merge:
  insert/update x-harness managed block only

--force:
  overwrite only if file is recognized as generated by x-harness, or user explicitly confirms

--dry-run:
  print planned changes only
```

### Managed block

```md
<!-- BEGIN X-HARNESS MANAGED AGENT CONTRACT -->
<!-- x-harness-context-hash: sha256:<hash> -->
<!-- generated-by: x-harness -->
<!-- generated-at: <ISO timestamp> -->

# x-harness Agent Contract

Completion is admitted, not merely claimed.

Before reporting completion:

1. Create or update `completion-card.yaml`.
2. Record changed files and validation evidence.
3. Run the closest available deterministic checks.
4. Run `npx x-harness verify`.
5. Report accepted only if verification returns `outcome=success` and `acceptance_status=accepted`.
6. For failed, blocked, skipped, timeout, or error, report withheld with reason and next action.

Verifier is read-only.
PGV and advisory checks are advisory-only.
Use the smallest tier that preserves verification quality: light, standard, or deep.

<!-- END X-HARNESS MANAGED AGENT CONTRACT -->
```

### Files to modify

```txt
packages/cli/src/commands/init.ts
packages/cli/src/commands/context.ts
packages/cli/src/commands/doctor.ts
packages/cli/src/core/context/generator.ts
packages/cli/src/core/context/hash.ts
templates/AGENTS.md.hbs
docs/CONTEXT_POLICY.md
tests/context.test.ts
tests/init.test.ts
tests/doctor.test.ts
```

### Acceptance criteria

```txt
[ ] init --minimal creates AGENTS.md or managed block.
[ ] AGENTS.md contains x-harness-context-hash.
[ ] Existing non-generated AGENTS.md is not overwritten silently.
[ ] --merge updates only managed block.
[ ] --dry-run does not write.
[ ] AGENTS.md remains short, target <= 150 lines.
```

## 8. P0.4 — Auto-inject context header into handoff

### Problem

User hoặc agent có thể quên gọi `x-harness context`. Handoff nên tự kèm context header ngắn để giảm lỗi protocol.

### Required behavior

```bash
x-harness handoff standard --title "Fix login timeout"
```

Output prefix:

```txt
# --- BEGIN X-HARNESS CONTEXT ---
Core: completion is admitted, not claimed.
Tier: standard.
Verify: completion-card.yaml -> npx x-harness verify.
Accepted: success only.
Withheld: failed/blocked/skipped/timeout/error.
Verifier: read-only.
PGV: advisory-only.
# --- END X-HARNESS CONTEXT ---
```

### Options

```bash
x-harness handoff standard --no-context
x-harness handoff standard --context-max-lines 12
```

Không thêm:

```bash
x-harness handoff --format prompt
```

Vì output mặc định đã prompt-ready.

### Files to modify

```txt
packages/cli/src/commands/handoff.ts
packages/cli/src/core/context/generator.ts
tests/handoff.test.ts
docs/CONTEXT_POLICY.md
```

### Acceptance criteria

```txt
[ ] Handoff includes context header by default.
[ ] --no-context disables injection.
[ ] Header is bounded to max 12 lines by default.
[ ] Header includes accepted/withheld semantics.
[ ] No handoff --format prompt is added.
```

## 9. P0.5 — Doctor context freshness checks

### Problem

Context staleness làm agent dùng protocol cũ. Doctor cần phát hiện nhưng không fail hard mặc định.

### Required checks

```txt
AGENTS.md freshness:
  - find x-harness managed block;
  - extract x-harness-context-hash;
  - recompute hash;
  - warn if mismatch;
  - strict mode may fail.

Context source presence:
  - policy exists;
  - completion-card schema exists;
  - completion card template exists;
  - X_HARNESS.md exists if expected.

AGENTS.md size:
  - warn if managed block becomes too long.
```

### Output

```txt
warning: AGENTS.md is stale. Run: x-harness context --refresh
```

### Files to modify

```txt
packages/cli/src/commands/doctor.ts
packages/cli/src/core/context/hash.ts
tests/doctor.test.ts
```

### Acceptance criteria

```txt
[ ] doctor warns on stale AGENTS.md.
[ ] doctor suggests x-harness context --refresh.
[ ] doctor does not fail hard by default.
[ ] doctor --strict fails on stale context.
[ ] doctor warns if AGENTS.md managed block is too long.
```

## 10. P0.6 — Read-only filesystem mutation guard for verify

### Problem

“Verifier is read-only” không nên chỉ là documentation rule. `x-harness verify` cần có guard kỹ thuật để phát hiện verify command làm thay đổi source tree ngoài allowlist.

### Goal

Trước và sau verify, capture file state. Nếu verify làm thay đổi file không được phép, outcome phải withheld hoặc command phải fail.

### Default policy

```txt
Allowed writes during verify:
  - .x-harness/traces/* if tracing is enabled
  - .x-harness/reports/* if explicitly requested
  - temporary files under OS temp directory

Forbidden writes during verify:
  - source files
  - package files
  - schema/policy/template files
  - completion-card.yaml unless explicit --allow-card-normalization is used
```

### Suggested behavior

```bash
x-harness verify
```

Should:

```txt
1. Capture pre-verify git status and file hashes.
2. Run deterministic admission checks.
3. Capture post-verify status and hashes.
4. Compare against allowlist.
5. If unexpected mutation is found:
   - outcome: blocked or failed;
   - acceptance_status: withheld;
   - reason: verifier_not_read_only;
   - recovery route: restore unexpected changes or rerun verify in clean tree.
```

### Implementation options

Tier 1, minimal:

```txt
Use git status --porcelain before/after.
Detect new/modified/deleted files.
Warn/fail if unexpected changes appear.
```

Tier 2, stronger:

```txt
Hash tracked files before/after.
Hash untracked non-ignored files before/after.
Use allowlist glob matching.
```

Tier 3, future:

```txt
Sandbox or read-only mount.
Not required in P0.
```

### Files to create/modify

```txt
packages/cli/src/core/readonly/fsSnapshot.ts
packages/cli/src/core/readonly/allowlist.ts
packages/cli/src/commands/verify.ts
packages/cli/src/core/recovery.ts
policies/admission.yaml
tests/verify-readonly.test.ts
```

### Acceptance criteria

```txt
[ ] verify detects unexpected source mutation.
[ ] unexpected mutation prevents accepted outcome.
[ ] allowlisted trace writes do not block verify.
[ ] recovery route is verifier_not_read_only.
[ ] test covers modified, deleted, and newly created source files.
```

## 11. P0.7 — Remove/defer redundant commands and flags

### Decision

Do not implement:

```bash
x-harness explain --for-agent
x-harness handoff --format prompt
```

Reason:

```txt
- context --verbose covers agent explanation.
- handoff context injection makes output prompt-ready.
- extra flags increase CLI surface without improving admission safety.
```

### Acceptance criteria

```txt
[ ] No explain --for-agent command is added.
[ ] If already present, it redirects to context --verbose with deprecation notice.
[ ] No handoff --format prompt flag is added.
[ ] Docs point to context and handoff context injection.
```

## 12. P1.1 — Predicate tiering

### Problem

Acceptance predicates có giá trị, nhưng nếu mọi predicate đều hard-check cho mọi task thì light tier sẽ nặng và dễ overblock. Cần phân tầng predicate.

### Predicate categories

```txt
Required:
  Failure blocks success.

Conditional:
  Failure blocks success only when condition applies.

Advisory:
  Failure creates warning only unless policy explicitly promotes it.
```

### Required predicates

```txt
claim_packet_valid
verify_invoked
evidence_floor_met
owner_accountable_present
no_unresolved_blocker
no_active_recovery
admission_mapping_valid
verifier_read_only
```

### Conditional predicates

```txt
deep_escalation_path
rollback_policy_present
human_approval_present
high_risk_evidence_scope_present
security_or_deploy_evidence_present
```

Apply when:

```txt
tier: deep
risk_class: high
task touches auth/payment/database/deploy/security
policy requires human approval
```

### Advisory predicates

```txt
context_acknowledged
context_hash_current
stale_ground_heuristic
advisory_warnings_treated
veto_condition_scan
coverage_signal_present
```

### Output mapping

Required failure:

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
reason: <predicate_id>
```

Conditional failure when applicable:

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
reason: <predicate_id>
```

Advisory failure:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
warnings:
  - predicate: context_acknowledged
    message: context was not explicitly acknowledged
```

Advisory failure must not block by default.

### File layout

```txt
packages/cli/src/core/predicates/
  required.ts
  conditional.ts
  advisory.ts
  types.ts
  index.ts

packages/cli/src/core/admission.ts
policies/admission.yaml
docs/ADMISSION_POLICY.md
tests/admission.test.ts
```

### Predicate interface

```ts
export type PredicateSeverity = "required" | "conditional" | "advisory";
export type PredicateResult = {
  id: string;
  severity: PredicateSeverity;
  passed: boolean;
  applicable: boolean;
  message?: string;
  recoveryRoute?: string;
};
```

### Acceptance criteria

```txt
[ ] Required predicate failure withholds completion.
[ ] Conditional predicate failure only applies when condition matches.
[ ] Advisory predicate failure creates warning only.
[ ] Deep tier remains stricter than standard.
[ ] Light tier remains low ceremony.
[ ] Output includes predicate breakdown.
```

## 13. P1.2 — Recovery golden test suite

### Problem

Recovery logic dễ drift nếu chỉ nằm trong code. Không cần simulator; cần deterministic golden tests.

### Required golden cases

```txt
examples/golden/recovery/
  01-evidence-missing/
  02-evidence-scope-missing/
  03-ownership-missing/
  04-active-recovery/
  05-typecheck-failed/
  06-test-failed/
  07-deep-approval-missing/
  08-stale-ground/
  09-verifier-not-read-only/
  10-policy-schema-mismatch/
```

Mỗi case gồm:

```txt
input-card.yaml
expected-verify-output.json
expected-recovery.yaml
README.md
```

### Example expected recovery

```yaml
recovery:
  route: evidence_missing
  owner: implementation-worker
  next_action: "Attach validation evidence or explain why unavailable."
```

### Command integration

```bash
x-harness examples verify
```

Should include recovery golden cases.

Không thêm command riêng `x-harness test recovery` ở P1 nếu chưa cần.

### Files to modify

```txt
packages/cli/src/commands/examples.ts
packages/cli/src/core/recovery.ts
examples/golden/recovery/*
tests/recovery-golden.test.ts
```

### Acceptance criteria

```txt
[ ] At least 10 deterministic recovery cases exist.
[ ] Each case checks outcome, acceptance_status, route, owner, next_action.
[ ] examples verify runs recovery cases.
[ ] Recovery cases run in CI.
[ ] No recovery simulator is added.
```

## 14. P1.3 — Evidence artifact hardening

### Problem

Evidence card dễ bị self-reported. Agent có thể ghi “tests passed” mà không gắn artifact thực. Cần làm evidence khó giả hơn nhưng vẫn giữ light tier nhẹ.

### Tier policy

Light tier:

```txt
- allow concise evidence;
- no mandatory stdout/stderr hash;
- command/status recommended.
```

Standard tier:

```txt
- require command;
- require status;
- require exit_code when command is local;
- require verifies mapping;
- recommend stdout_hash/stderr_hash.
```

Deep tier:

```txt
- require command;
- require status;
- require exit_code;
- require artifact hash or CI reference when available;
- require evidence_scope;
- require untested_regions;
- require remaining_risks;
- require read_set/write_set.
```

### Schema extension

```yaml
verification_artifacts:
  - kind: typecheck
    command: npm run typecheck
    status: passed
    exit_code: 0
    stdout_hash: sha256:...
    stderr_hash: sha256:...
    artifact_path: .x-harness/artifacts/typecheck.log
    artifact_hash: sha256:...
    ci_run_url: null
    ran_at: 2026-05-22T10:00:00Z
    verifies:
      - TypeScript compiles
    does_not_verify:
      - Runtime behavior in production
```

### Important constraint

Không bắt buộc mọi project phải lưu log nặng. Nếu artifact path không có, standard tier vẫn có thể pass nếu policy cho phép. Deep tier nên nghiêm hơn.

### Files to modify

```txt
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md
packages/cli/src/core/predicates/required.ts
packages/cli/src/core/predicates/conditional.ts
packages/cli/src/core/metrics.ts
docs/ADMISSION_POLICY.md
tests/admission.test.ts
```

### Acceptance criteria

```txt
[ ] Standard tier blocks missing command/status evidence.
[ ] Deep tier blocks missing evidence_scope when required.
[ ] Deep tier blocks missing exit_code for local command evidence.
[ ] Light tier remains permissive.
[ ] Evidence artifact fields are optional where policy allows.
```

## 15. P1.4 — Policy source-of-truth or drift guard

### Problem

Nếu `policies/admission.yaml` chỉ là documentation-like artifact còn logic thực nằm trong TypeScript, policy và code có thể drift.

### Acceptable options

Option A — Policy-as-source-of-truth:

```txt
- admission.yaml declares predicates, severity, tier conditions, and recovery route.
- TypeScript loads and interprets policy.
- Tests validate example cards against policy.
```

Option B — Code remains source-of-truth but drift guard is explicit:

```txt
- admission.yaml includes policy_version and predicate list.
- TypeScript exports predicate registry.
- doctor compares registry against YAML.
- mismatch is warning by default, fail in --strict.
```

Preferred path:

```txt
P1: Option B first.
P2: Move toward Option A if stable.
```

### YAML shape

```yaml
policy_version: 1
predicates:
  required:
    - id: claim_packet_valid
      recovery_route: claim_packet_missing
    - id: verifier_read_only
      recovery_route: verifier_not_read_only
  conditional:
    - id: human_approval_present
      when:
        tier: deep
        risk_class: high
      recovery_route: approval_missing
  advisory:
    - id: context_acknowledged
      warning: context was not explicitly acknowledged
```

### Files to modify

```txt
policies/admission.yaml
packages/cli/src/core/predicates/index.ts
packages/cli/src/commands/doctor.ts
packages/cli/src/core/policy/loadPolicy.ts
packages/cli/src/core/policy/validatePolicy.ts
tests/policy-drift.test.ts
```

### Acceptance criteria

```txt
[ ] Predicate registry exists.
[ ] admission.yaml predicate list is checked by doctor.
[ ] Missing policy predicate triggers warning.
[ ] doctor --strict fails on policy-code drift.
[ ] No heavy workflow language is introduced.
```

## 16. P1.5 — `context_acknowledged` advisory check

### Problem

Biết agent dùng context version nào là hữu ích, nhưng không nên biến thành checkbox theater chặn completion mặc định.

### Schema extension

```yaml
metadata:
  context_acknowledged: true
  context_hash: sha256:abc123
```

### Policy

```txt
missing context_acknowledged:
  warning only

context_hash mismatch:
  warning only

strict mode:
  may promote to blocked later, but not default
```

### Files to modify

```txt
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md
packages/cli/src/core/predicates/advisory.ts
packages/cli/src/core/admission.ts
docs/CONTEXT_POLICY.md
tests/admission.test.ts
```

### Acceptance criteria

```txt
[ ] metadata.context_acknowledged is optional.
[ ] Missing field creates warning only.
[ ] Hash mismatch creates warning only.
[ ] Success is not blocked by this advisory check by default.
```

## 17. P1.6 — Git context inheritance, opt-in only

### Problem

Agent có thể thiếu các quyết định gần đây liên quan đến x-harness policy. Nhưng dump Git history vào context là nhiễu.

### Command

```bash
x-harness context --inherit-from-git --last 5
```

Chỉ lấy commit message có prefix:

```txt
[x-harness]
[xh]
```

### Output

```txt
# Recent x-harness decisions
- Raised standard evidence floor to require typecheck (abc123)
- Switched default test command from Jest to Vitest (def456)
```

### Constraints

```txt
- max 5 commits by default;
- do not include full commit body by default;
- do not inject into handoff by default;
- do not use as admission evidence;
- advisory context only.
```

### Files to modify

```txt
packages/cli/src/core/context/gitInherit.ts
packages/cli/src/commands/context.ts
tests/context.test.ts
docs/CONTEXT_POLICY.md
```

### Acceptance criteria

```txt
[ ] Only prefixed commits are included.
[ ] Output is short.
[ ] Feature is opt-in.
[ ] No Git history is injected by default.
```

## 18. P1.7 — GitHub Action/docs for consumer repositories

### Problem

Repo may already have internal CI, but users need a simple way to apply verify gate in their own repositories.

### Scope

Provide reusable docs and optional composite action. Do not build CI adapters for every platform.

### Example

```yaml
name: x-harness verify

on:
  pull_request:
  push:
    branches: [main]

jobs:
  x-harness:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
      - run: npm ci
      - run: npm run build --if-present
      - run: npx x-harness doctor --root .
      - run: npx x-harness examples verify
      - run: npx x-harness verify --root .
```

### Files to create

```txt
.github/actions/x-harness/action.yml
docs/CI.md
examples/ci/github-action.yml
```

### Acceptance criteria

```txt
[ ] GitHub Action usage is documented.
[ ] Composite action runs doctor and verify/examples verify.
[ ] No external service required.
[ ] Docs are clearly optional.
```

## 19. P1.8 — Handoff readiness questions, CLI-only

### Problem

Agent có thể chọn sai tier hoặc proceed khi task ambiguous. Cần readiness hint nhẹ, không phải product planning lifecycle.

### Command

```bash
x-harness handoff --interactive
```

Questions:

```txt
1. What is the task objective?
2. Is this low-risk, normal, or high-risk?
3. Does it touch auth/payment/database/deploy/security?
4. What evidence can verify it?
5. Is anything missing or ambiguous?
```

Output block:

```yaml
readiness:
  goal: "Fix login timeout"
  suggested_tier: standard
  proceed: true
  missing_information: []
  evidence_expected:
    - typecheck
    - unit_test
```

### Constraints

Không tạo:

```txt
- feature intake file;
- story packet;
- product spec;
- test matrix;
- decision record.
```

### Acceptance criteria

```txt
[ ] handoff --interactive suggests tier.
[ ] high-risk categories suggest deep.
[ ] no product lifecycle files are created.
[ ] output remains handoff-only.
```

## 20. P2.1 — Git-native packet chain verify

### Problem

Auditability tốt hơn nếu claim/evidence/recovery có lineage. Không cần packet store riêng; file + Git đủ.

### Packet layout

```txt
.x-harness/
  packets/
    control/
    claims/
    evidence/
    recovery/
  trace.jsonl
```

### Packet file example

```yaml
packet_id: claim-2026-05-22T10-15-30-task-123
task_id: task-123
type: claim
owner: implementation-worker
parent_packet: control-2026-05-22T10-00-01-task-123
created_at: 2026-05-22T10:15:30Z
content_hash: sha256:...
```

### Commands

```bash
x-harness packet create claim --task task-123
x-harness packet create evidence --task task-123 --parent claim-...
x-harness packet verify-chain --task task-123
```

### Git behavior

Default:

```txt
write files only
no git add
no git commit
```

Explicit:

```bash
x-harness packet create claim --task task-123 --git-add
x-harness packet create claim --task task-123 --git-commit
```

Never auto-commit by default.

### Files to create/modify

```txt
packages/cli/src/commands/packet.ts
packages/cli/src/core/packet/createPacket.ts
packages/cli/src/core/packet/verifyChain.ts
packages/cli/src/core/packet/hash.ts
schemas/packet.schema.json
docs/PACKETS.md
tests/packet.test.ts
```

### Acceptance criteria

```txt
[ ] Packet files are created under .x-harness/packets.
[ ] verify-chain checks parent lineage.
[ ] Missing parent blocks chain verification.
[ ] Hash mismatch blocks chain verification.
[ ] No auto-commit by default.
[ ] --git-add and --git-commit are explicit.
```

## 21. P2.2 — Static single-file HTML audit report

### Problem

CLI output tốt cho automation, nhưng human audit cần report dễ đọc. Không cần dashboard.

### Command

```bash
x-harness report --format html > audit.html
```

### Requirements

```txt
single file
no external JS
no external CSS
no tracking
no server
no React/Vue/Svelte
opens offline in browser
```

### Content

```txt
- card id
- task id
- tier
- predicate breakdown
- outcome / acceptance_status
- evidence summary
- warnings
- recovery route
- read-only guard result
- trace timeline if trace exists
- denominator warning
- policy hash
- context hash
```

### Denominator warning

Report must state:

```txt
This report summarizes admission events and evidence artifacts. It is not a task-level success rate or production reliability estimate unless an aligned task denominator is provided.
```

### Files to modify

```txt
packages/cli/src/commands/report.ts
packages/cli/src/core/report/html.ts
packages/cli/src/core/report/escapeHtml.ts
tests/report-html.test.ts
```

### Acceptance criteria

```txt
[ ] report --format html works.
[ ] HTML is one standalone file.
[ ] No external assets.
[ ] Includes denominator warning.
[ ] Escapes user-provided fields safely.
```

## 22. P2.3 — Technical handbook and architecture docs

### Goal

Tạo docs cho humans, nhưng không biến docs thành context bắt buộc cho mọi agent task.

### Files

```txt
docs/HANDBOOK.md
docs/ARCHITECTURE.md
docs/ADMISSION_POLICY.md
docs/CONTEXT_POLICY.md
docs/RECOVERY.md
docs/PACKETS.md
docs/CI.md
```

### Constraints

```txt
- README remains short.
- AGENTS.md remains short.
- context command remains short.
- docs are linked, not injected wholesale.
- docs explain x-harness internals, not product planning lifecycle.
```

### Acceptance criteria

```txt
[ ] Docs exist and are linked from README.
[ ] AGENTS.md does not duplicate handbook.
[ ] doctor warns if AGENTS.md managed block becomes too long.
[ ] Docs avoid unsupported correctness claims.
```

## 23. P2.4 — Static recovery playbook suggestions from trace.jsonl

### Problem

Trace can reveal repeated recovery patterns, but automatic learning can cause policy drift.

### Command

```bash
x-harness recovery suggest-playbook --from .x-harness/trace.jsonl
```

### Behavior

Default:

```txt
- read trace.jsonl;
- group repeated recovery routes;
- suggest candidate playbook entries;
- print deterministic YAML to stdout;
- do not mutate policy files.
```

Write mode:

```bash
x-harness recovery suggest-playbook --from .x-harness/trace.jsonl --write --force
```

Even with `--write`, output should be candidate-only unless user explicitly chooses destination.

### Candidate format

```yaml
candidate_playbooks:
  - route: evidence_missing
    observed_count: 12
    suggested_next_action: "Attach command evidence with exit_code and artifact hash."
    confidence: medium
    requires_review: true
```

### Files to create/modify

```txt
packages/cli/src/commands/recovery.ts
packages/cli/src/core/recovery/suggestPlaybook.ts
policies/recovery.yaml
docs/RECOVERY.md
tests/recovery-playbook.test.ts
```

### Acceptance criteria

```txt
[ ] command outputs deterministic suggestions.
[ ] no automatic policy mutation by default.
[ ] --write requires --force.
[ ] suggestions are marked requires_review.
[ ] no autonomous learning loop is introduced.
```

## 24. P2.5 — Trace integrity hash chain

### Problem

Trace JSONL dễ append/read, nhưng audit tốt hơn nếu mỗi event có hash chain. Không cần database.

### Trace event extension

```json
{
  "event_id": "evt_...",
  "task_id": "task-123",
  "event_type": "verify_completed",
  "created_at": "2026-05-22T10:00:00Z",
  "payload_hash": "sha256:...",
  "prev_event_hash": "sha256:...",
  "event_hash": "sha256:..."
}
```

### Command

```bash
x-harness trace verify-chain
```

### Acceptance criteria

```txt
[ ] New trace events include event_hash.
[ ] verify-chain detects modified historical event.
[ ] verify-chain detects broken prev_event_hash.
[ ] Existing trace files without hash are treated as legacy, not silently trusted.
```

## 25. P3.1 — YAML custom checks before plugin system

### Decision

Không triển khai plugin system trước khi YAML custom checks không còn đủ.

### YAML extension

```yaml
custom_checks:
  - id: security_audit_present
    required_for_tiers: [deep]
    evidence_kind: security_scan
    recovery_route: security_evidence_missing
```

### Constraints

```txt
- no arbitrary code execution;
- no marketplace;
- local policy only;
- doctor validates custom check schema;
- verify reports custom check results in predicate breakdown.
```

### Acceptance criteria

```txt
[ ] YAML custom checks can require evidence kind by tier.
[ ] Custom check failure maps to withheld when required.
[ ] No plugin runtime is introduced.
```

## 26. P3.2 — File-based plugin system, disabled by default

Chỉ cân nhắc nếu có nhu cầu thật.

### Constraints if implemented

```txt
- disabled by default;
- local trusted code only;
- explicit enable in .x-harness/plugins.yaml;
- no marketplace;
- no remote install;
- no network by default;
- no admission authority unless result maps through policy;
- plugin failures are withheld only if policy says required.
```

### Recommendation

Hoãn. YAML custom checks có khả năng đủ cho 80% use case.

## 27. Testing strategy

### Unit tests

```txt
context generation
context hash
AGENTS.md managed block update
handoff context injection
predicate evaluation
policy drift guard
read-only mutation guard
evidence artifact validation
recovery route mapping
trace hash chain
HTML report escaping
```

### Golden tests

```txt
examples/golden/admission/*
examples/golden/recovery/*
examples/golden/context/*
examples/golden/report/*
```

### Negative tests

```txt
- outcome=success but acceptance_status=withheld
- outcome=failed but acceptance_status=accepted
- tests passed but required evidence missing
- verification.status=passed but fix_status not fixed
- deep tier without human approval when policy requires it
- verify mutates source file
- stale AGENTS.md context hash
- malformed custom check policy
- trace event tampering
```

### CI requirements

```txt
npm ci
npm run typecheck
npm run lint
npm run test
npm run build
npx x-harness doctor --strict
npx x-harness examples verify
```

## 28. Rollout plan

### Phase 1 — Stabilize wording and context

```txt
1. Remove overclaim wording.
2. Add context command.
3. Add context hash.
4. Generate/update AGENTS.md managed block.
5. Inject context into handoff.
6. Add doctor freshness warning.
```

Expected impact:

```txt
- Agent cold-start improves.
- Protocol drift decreases.
- README becomes more defensible.
```

### Phase 2 — Strengthen verify boundary

```txt
1. Add read-only mutation guard.
2. Add predicate tiering.
3. Add recovery golden suite.
4. Add evidence artifact hardening.
5. Add policy-code drift guard.
```

Expected impact:

```txt
- Completion is harder to fake.
- Fail-closed behavior is more testable.
- Recovery routes become deterministic.
```

### Phase 3 — Improve auditability

```txt
1. Add packet chain verify.
2. Add static HTML report.
3. Add trace hash chain.
4. Add technical docs.
5. Add reviewed recovery playbook suggestions.
```

Expected impact:

```txt
- Auditors can inspect path from claim to evidence to verify.
- Reports become human-readable without dashboard.
- Trace tampering becomes detectable.
```

## 29. Anti-regression rules

Do not merge a feature if it violates any rule below:

```txt
[ ] It makes light tier heavy by default.
[ ] It lets worker self-admit completion.
[ ] It lets PGV become admission authority.
[ ] It treats tests passed as accepted.
[ ] It treats context_acknowledged as accepted.
[ ] It requires daemon, DB, cloud service, MCP, or LLM verifier.
[ ] It adds product planning lifecycle files.
[ ] It creates dashboard/server runtime.
[ ] It auto-commits Git changes by default.
[ ] It mutates policy automatically from trace without explicit review.
[ ] It adds plugin execution before YAML custom checks are exhausted.
```

## 30. Final implementation checklist

### P0 checklist

```txt
[ ] README/X_HARNESS wording corrected.
[ ] Unsupported score/token/correctness claims removed.
[ ] context command added.
[ ] context --verbose added.
[ ] context --json added.
[ ] context --refresh added.
[ ] AGENTS.md managed block generation added.
[ ] x-harness-context-hash added.
[ ] doctor detects stale context.
[ ] handoff context injection added.
[ ] handoff --no-context added.
[ ] read-only mutation guard added.
[ ] no explain --for-agent added.
[ ] no handoff --format prompt added.
```

### P1 checklist

```txt
[ ] Predicate modules added.
[ ] Required predicates block success.
[ ] Conditional predicates apply only when applicable.
[ ] Advisory predicates warn only.
[ ] Recovery golden cases added.
[ ] examples verify runs recovery cases.
[ ] Evidence artifact hardening added for standard/deep.
[ ] Policy-code drift guard added.
[ ] context_acknowledged advisory metadata added.
[ ] Git context inheritance opt-in added.
[ ] Consumer GitHub Action docs added.
[ ] handoff --interactive readiness added without lifecycle files.
```

### P2 checklist

```txt
[ ] packet create command added.
[ ] packet verify-chain command added.
[ ] no auto-commit by default.
[ ] report --format html added.
[ ] static HTML has no external assets.
[ ] denominator warning included.
[ ] technical docs added.
[ ] recovery suggest-playbook added.
[ ] no automatic policy mutation by default.
[ ] trace verify-chain added.
```

### P3 checklist

```txt
[ ] YAML custom checks implemented before plugins.
[ ] Plugin system remains deferred unless demand exists.
[ ] If plugins exist, they are disabled by default.
```

## 31. Definition of done

Implementation is acceptable when:

```txt
1. Minimal mode remains lightweight.
2. Agent context is short, generated, and freshness-checked.
3. Handoff is prompt-ready without adding format complexity.
4. Verify gate has a technical read-only guard.
5. Completion cannot be accepted unless outcome=success and acceptance_status=accepted.
6. Required/conditional/advisory predicates are separated.
7. Recovery behavior is covered by deterministic golden cases.
8. Evidence is harder to fake in standard/deep mode.
9. Policy-code drift is detectable.
10. Git integration never commits by default.
11. Audit report is static and optional.
12. Trace and packet lineage are inspectable.
13. README does not overclaim correctness.
14. No feature from product-planning harnesses is introduced.
```

## 32. Strategic summary

The highest-value direction for `x-harness` is not more framework surface. It is a sharper admission boundary.

Correct strategic target:

```txt
completion claims become auditable;
verification remains read-only;
evidence becomes harder to fake;
ambiguous cases fail closed;
recovery routes are deterministic;
context stays fresh without becoming heavy;
policy drift becomes detectable;
Git remains the database;
CLI remains the interface.
```

That is the path with the best value-to-complexity ratio.
