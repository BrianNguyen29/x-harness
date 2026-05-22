# x-harness Improvement Implementation Plan

**Mục tiêu:** tài liệu này là bản hướng dẫn triển khai chi tiết để một AI coding agent có thể cải thiện `x-harness` theo hướng thực dụng, có kiểm chứng, ít trade-off, không biến repo thành framework nặng.

**Phạm vi:** chỉ đề xuất cải thiện trực tiếp cho `x-harness`. , không thêm feature-intake/product-planning/story-packet/test-matrix/decision-record lifecycle.

**Định vị giữ nguyên:** `x-harness` là một lightweight, offline-first, verify-gated completion/admission harness.

Nó làm một việc chính:

```txt
agent work
  -> completion claim
  -> evidence
  -> read-only verify gate
  -> accepted or withheld
```

Nó không phải:

```txt
- agent runtime
- workflow engine
- product planning harness
- benchmark framework
- dashboard platform
- plugin marketplace
- storage/database layer
- mandatory MCP runtime
- mandatory LLM judge/verifier
```

---

## 0. Executive summary

Các cải thiện có ROI cao nhất và ít trade-off nhất:

```txt
P0:
1. Sửa README/report wording để không overclaim correctness.
2. Thêm x-harness context và AGENTS.md freshness hash.
3. Tự chèn context header ngắn vào handoff.
4. Thêm read-only mutation guard cho verify.
5. Làm report denominator-safe.

P1:
6. Evidence artifact hardening cho standard/deep.
7. Predicate tiering: Required / Conditional / Advisory.
8. Policy-code drift guard.
9. Recovery golden tests.
10. Task lifecycle trace ledger.
11. OTel-compatible JSONL naming.
12. Human approval chỉ cho deep/high-risk.
13. context_acknowledged advisory-only.
14. Git context inheritance opt-in only.

P2:
15. Git-native packet chain verify.
16. Static single-file HTML audit report.
17. Optional sandbox bridge, không tự build sandbox.
18. Human handbook docs, không làm agent context nặng.
19. Static recovery playbook suggestions from trace.jsonl.

P3:
20. MCP read-only adapter, disabled by default.
21. YAML custom checks trước plugin system.
22. Plugin system chỉ làm nếu có demand thật.
```

Nguyên tắc cốt lõi:

```txt
Completion is admitted, not merely claimed.
Worker may propose completion, but cannot self-admit.
Verifier is read-only.
success is the only accepted outcome.
failed / blocked / skipped / timeout / error are withheld.
PGV/advisory checks remain advisory-only unless explicitly promoted by policy.
Deep mode is opt-in or risk-triggered.
Git is the database.
YAML is the protocol.
CLI is the interface.
Markdown is the handbook.
HTML report is static and optional.
Verification is deterministic-first.
```

---

## 1. Source-informed rationale

Các đề xuất này dựa trên các nhóm nguồn đã phân tích trước đó:

1. **Verify-gated completion architecture**: tách execution, claim, acceptance và completion; worker chỉ propose claim; admission verifier read-only giữ quyền veto; ambiguous case fail-closed; packetized evidence và trace giúp audit.
2. **Harness engineering practice**: context nên ngắn, fresh, load-on-demand; `AGENTS.md` không nên thành manual lớn; repository knowledge nên là system of record.
3. **Guardrails và human review**: human review nên dùng có mục tiêu cho deep/high-risk hoặc side-effecting actions, không áp dụng mặc định cho mọi task.
4. **SWE-bench / sandboxed evaluation**: evidence đáng tin phải là executable check có command, exit code, artifact/hash, không chỉ là lời agent tự khai.
5. **Policy-as-code practice**: policy nên tách khỏi enforcement logic hoặc có drift guard để tránh YAML nói một kiểu, TypeScript admission engine làm kiểu khác.
6. **OpenTelemetry / observability practice**: không cần collector service, nhưng JSONL trace nên dùng field naming dễ map về traces/logs/metrics sau này.
7. **SLSA/provenance practice**: artifact evidence nên có provenance tối thiểu như command, timestamp, exit code, artifact hash.
8. **MCP/security practice**: MCP hữu ích cho context/tool integration nhưng mở bề mặt rủi ro; nếu có, chỉ nên là optional read-only adapter, disabled by default.

Điều quan trọng: không được trình bày `x-harness` như công cụ đảm bảo correctness, production reliability hoặc giảm bug đã được chứng minh. Claim hợp lý là:

```txt
x-harness helps make completion claims harder to fake,
easier to verify,
safer to withhold,
and more auditable under repository policy.
```

---

## 2. Non-goals

Không triển khai các mục sau trong roadmap này:

```txt
- Full agent runtime.
- Task planner / product planner.
- Feature intake lifecycle.
- Story packet lifecycle.
- Product docs generator.
- Test matrix generator.
- Decision-record lifecycle.
- Dashboard/server UI.
- Database-backed event store.
- Mandatory MCP integration.
- Mandatory LLM-as-judge/verifier.
- Plugin marketplace.
- Recovery simulator.
- Automatic learning loop that mutates policy.
- Automatic git commit by default.
```

Nếu một proposal làm `x-harness` thành “orchestration framework”, bỏ khỏi core.

Câu hỏi bắt buộc trước mỗi feature:

```txt
Does this make completion harder to fake,
easier to verify,
and safer to withhold,
without adding default process weight?
```

Nếu câu trả lời không rõ là “có”, không đưa vào core.

---

## 3. Current-state assumptions to verify first

Trước khi code, agent phải kiểm tra repo hiện tại. Không giả định blind.

Run:

```bash
git status --short
npm ci
npm run typecheck
npm run build
npm test
npx x-harness doctor --root .
npx x-harness examples verify
```

Inspect các file sau:

```txt
README.md
AGENTS.md
X_HARNESS.md
policies/admission.yaml
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md

packages/cli/src/index.ts
packages/cli/src/commands/init.ts
packages/cli/src/commands/handoff.ts
packages/cli/src/commands/verify.ts
packages/cli/src/commands/doctor.ts
packages/cli/src/commands/report.ts
packages/cli/src/commands/trace.ts

packages/cli/src/core/admission.ts
packages/cli/src/core/recovery.ts
packages/cli/src/core/metrics.ts
packages/cli/src/core/trace.ts
packages/cli/tests/*
.github/workflows/*
```

Expected initial gap list to confirm:

```txt
- No x-harness context command.
- AGENTS.md may be static and lacks freshness hash.
- handoff likely lacks bounded context header.
- verify read-only may be a contract, not a filesystem-enforced invariant.
- evidence fields may be self-reported and lack provenance.
- admission.yaml may not be true source of truth.
- report may not distinguish event-level vs task-level denominators strongly enough.
- trace may not have full lifecycle events.
- HTML report may be absent or not standalone.
```

---

## 4. Global implementation rules

### 4.1 Backward compatibility

Do not break existing commands:

```txt
x-harness init
x-harness handoff
x-harness add
x-harness verify
x-harness doctor
x-harness report
x-harness trace
x-harness clean
x-harness examples
```

New behavior should be additive or warning-only unless explicitly configured as strict.

### 4.2 Strict mode

Use strict mode only where needed:

```bash
x-harness verify --strict
x-harness doctor --strict
```

Default behavior should avoid unnecessary hard failures for advisory/context features.

### 4.3 Output contract

All machine-readable output must support JSON:

```bash
x-harness context --json
x-harness verify --json
x-harness doctor --json
x-harness report --json
```

If a command already has a JSON contract, do not break it.

### 4.4 Fail-closed admission

Only this is accepted:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

All other outcomes are withheld:

```txt
failed
blocked
skipped
timeout
error
missing_outcome
unknown
```

Never treat these as accepted:

```txt
fix_status: fixed
verification.status: passed
tests passed
agent confidence: HIGH
PGV says okay
context_acknowledged: true
human says looks fine
```

### 4.5 PGV/advisory boundary

PGV/advisory checks may produce warnings, scores, hints, or recommended recovery actions.

They must not:

```txt
- override core admission
- convert withheld to accepted
- mutate the work product
- become mandatory admission authority by default
```

### 4.6 No auto-commit

No new command may run `git add` or `git commit` unless user explicitly passes a flag like:

```bash
--git-add
--git-commit
```

Default must be file write only, or dry-run.

---

## 5. Roadmap overview

| Priority | Item | Value | Trade-off | Default behavior |
|---|---|---:|---:|---|
| P0 | Claim-boundary wording | High | Very low | Replace overclaims |
| P0 | context command + AGENTS hash | High | Low | Enabled |
| P0 | Handoff context header | High | Low | Enabled, disable with `--no-context` |
| P0 | Read-only mutation guard | Very high | Medium-low | Warn or fail in strict |
| P0 | Denominator-safe report | High | Low | Enabled |
| P1 | Evidence provenance | Very high | Medium | Required for standard/deep |
| P1 | Predicate tiering | Very high | Medium | Required/conditional/advisory |
| P1 | Policy-code drift guard | High | Low | Doctor check |
| P1 | Recovery golden tests | High | Low | CI |
| P1 | Task lifecycle trace ledger | High | Medium-low | Enabled when trace used |
| P1 | OTel-compatible JSONL | Medium-high | Low | Schema-compatible |
| P1 | Human approval for deep/high-risk | Medium-high | Low | Conditional |
| P1 | context_acknowledged advisory | Medium | Low | Warning only |
| P1 | Git context inheritance | Medium | Low | Opt-in only |
| P2 | Git-native packet chain | Medium-high | Medium | Optional |
| P2 | Static HTML report | Medium-high | Low | Optional output |
| P2 | Sandbox bridge | Medium | Low | Optional hook |
| P2 | Handbook docs | Medium | Low | Human-only docs |
| P2 | Recovery playbook suggestion | Medium | Medium | No auto-promotion |
| P3 | MCP read-only adapter | Conditional | Medium-high | Disabled by default |
| P3 | YAML custom checks | Medium | Medium | Optional |
| P3 | Plugin system | Low now | High | Defer |

---

# PART A — P0 Implementation

---

## 6. P0.1 — Correct claim boundary and remove unsupported claims

### Problem

Some wording may imply `x-harness` guarantees correctness or ensures tasks are truly done. This is unsupported. `x-harness` should claim admission control, inspectability, evidence discipline, and fail-closed behavior under repository policy.

### Required wording policy

Avoid:

```txt
ensure tasks are truly done
guarantees correctness
guarantees no hallucination
guarantees bug reduction
production-proven reliability
100% recovery completeness
<2000 tokens guaranteed
89/100 quality score
```

Use:

```txt
helps ensure completion claims are admitted under repository policy
makes completion claims harder to fake and easier to audit
withholds non-success outcomes fail-closed
supports deterministic-first verification
provides evidence and trace structure for review
```

### Files to inspect/modify

```txt
README.md
X_HARNESS.md
docs/*
packages/cli/src/commands/report.ts
packages/cli/src/commands/doctor.ts
examples/*
templates/*
```

### Required content changes

1. Replace correctness claims.
2. Add a clear “Evidence boundary” paragraph:

```md
## Evidence boundary

x-harness is an admission-control harness. It does not prove task correctness,
production reliability, safety guarantees, or end-to-end task success. It helps
enforce repository policy by requiring structured completion claims, evidence,
read-only verification, fail-closed non-success outcomes, and auditable traces.
```

3. Add “Metrics boundary” paragraph:

```md
## Metrics boundary

Verify-event success rates are event-level accounting measures. They are not
task-level completion rates unless the report explicitly defines an aligned
task denominator. Reports must distinguish verify events, task lifecycle
events, and unknown denominators.
```

### Acceptance criteria

```txt
[ ] README does not claim correctness guarantees.
[ ] README explains admission-control boundary.
[ ] Report wording distinguishes event-level and task-level metrics.
[ ] No unsupported score/token/performance claim remains.
[ ] Paper/reference IDs are consistent if mentioned.
[ ] Tests/snapshots updated.
```

### Suggested test

Add snapshot test for README/report phrases if existing test framework makes this easy. Otherwise add doctor check:

```txt
doctor warning: README contains overclaim phrase "guarantees correctness"
```

---

## 7. P0.2 — Add `x-harness context` command

### Problem

Agents need a short, fresh context envelope. They should not read all docs/templates for every task.

### Goal

Add:

```bash
x-harness context
x-harness context --verbose
x-harness context --json
x-harness context --refresh
```

Do not add `--watch` in P0.

### Default output

Target: 100–200 tokens.

Example:

```txt
# x-harness context
Core: completion is admitted, not claimed.
Tier: use light | standard | deep; choose smallest sufficient tier.
Verify: create/update completion-card.yaml, then run x-harness verify.
Accepted: only outcome=success and acceptance_status=accepted.
Withheld: failed, blocked, skipped, timeout, error.
Verifier: read-only.
PGV: advisory-only.
Evidence: record changed files and validation artifacts.
```

### Verbose output

`x-harness context --verbose` may include:

```txt
- verify-gated workflow
- completion-card required fields
- accepted/withheld mapping
- tier selection guidance
- blocked/recovery example
- evidence artifact expectations
```

Still keep it bounded. Do not print full manuals.

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
  "evidence_policy": "record changed files and validation artifacts",
  "context_hash": "sha256:...",
  "core_context_hash": "sha256:...",
  "adapter_context_hash": "sha256:..."
}
```

### Suggested file structure

Create:

```txt
packages/cli/src/commands/context.ts
packages/cli/src/core/context/generator.ts
packages/cli/src/core/context/hash.ts
packages/cli/src/core/context/agents.ts
docs/CONTEXT_POLICY.md
packages/cli/tests/context.test.ts
```

Modify:

```txt
packages/cli/src/index.ts
packages/cli/src/commands/init.ts
packages/cli/src/commands/doctor.ts
packages/cli/src/commands/handoff.ts
```

### Hash inputs

Core context hash should include:

```txt
policies/admission.yaml
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md
X_HARNESS.md
README.md section containing core admission contract
```

Adapter context hash should include enabled adapter artifacts, if present:

```txt
adapters/claude*
adapters/cursor*
adapters/opencode*
adapters/antigravity*
```

If adapter files do not exist or no adapter is enabled:

```txt
adapter_context_hash: null
```

### Pseudocode

```ts
export async function computeContextHash(root: string): Promise<ContextHashes> {
  const coreFiles = [
    "policies/admission.yaml",
    "schemas/completion-card.schema.json",
    "templates/COMPLETION_CARD.md",
    "X_HARNESS.md",
    "README.md"
  ];

  const coreDigest = await hashExistingFiles(root, coreFiles);

  const adapterFiles = await findEnabledAdapterFiles(root);
  const adapterDigest = adapterFiles.length > 0
    ? await hashExistingFiles(root, adapterFiles)
    : null;

  const combined = sha256([coreDigest, adapterDigest ?? ""].join("\n"));

  return {
    context_hash: `sha256:${combined}`,
    core_context_hash: `sha256:${coreDigest}`,
    adapter_context_hash: adapterDigest ? `sha256:${adapterDigest}` : null
  };
}
```

### CLI behavior

```bash
x-harness context
```

Prints short context.

```bash
x-harness context --verbose
```

Prints longer bounded context.

```bash
x-harness context --json
```

Prints valid JSON only.

```bash
x-harness context --refresh
```

Regenerates x-harness managed block in `AGENTS.md` and updates hash.

### Acceptance criteria

```txt
[ ] `x-harness context` works.
[ ] Default output is short.
[ ] `x-harness context --verbose` works.
[ ] `x-harness context --json` returns valid JSON only.
[ ] `x-harness context --refresh` updates managed AGENTS.md block.
[ ] Context hash changes when policy/schema/template changes.
[ ] No daemon/service/watch mode added.
[ ] Tests cover text, verbose, json, refresh.
```

---

## 8. P0.3 — Generate/update AGENTS.md with freshness hash

### Problem

`AGENTS.md` is likely the first file coding agents read. If it is stale, the agent may follow outdated admission rules.

### Goal

`init` creates or updates a short `AGENTS.md` with freshness hash.

### Header

Generated or managed block should include:

```md
<!-- x-harness-context-hash: sha256:<hash> -->
<!-- x-harness-core-context-hash: sha256:<hash> -->
<!-- x-harness-adapter-context-hash: sha256:<hash-or-none> -->
<!-- generated-by: x-harness -->
<!-- generated-at: <ISO timestamp> -->
```

### Managed block markers

Use markers so existing user content is preserved:

```md
<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
...
<!-- END X-HARNESS MANAGED CONTEXT -->
```

### Default behavior

If `AGENTS.md` does not exist:

```txt
create file
```

If `AGENTS.md` exists and contains managed block:

```txt
update managed block only
```

If `AGENTS.md` exists and has no managed block:

```txt
default: do not overwrite silently
suggest: x-harness context --refresh --merge
```

Support:

```bash
x-harness context --refresh --merge
x-harness context --refresh --force
x-harness context --refresh --dry-run
```

If global flags are already available, reuse them.

### Minimal AGENTS content

Keep <=150 lines, preferably <=80 lines.

```md
# x-harness Agent Contract

This repository uses x-harness.

Core rule: completion is admitted, not merely claimed.

Before reporting completion:
1. Create or update `completion-card.yaml`.
2. Record changed files and validation evidence.
3. Run the closest available checks.
4. Run `x-harness verify`.
5. Report accepted only if verification returns:
   - outcome: success
   - acceptance_status: accepted
6. For failed, blocked, skipped, timeout, or error, report withheld with reason and next action.

Verification is read-only.
PGV and advisory checks are advisory-only.

Canonical tiers:
- light
- standard
- deep

Use the smallest tier that preserves verification quality.

Useful commands:
- `x-harness context`
- `x-harness handoff <tier> --title "..."`
- `x-harness verify`
- `x-harness doctor`
```

### Doctor freshness check

Add check:

```txt
AGENTS.md freshness:
  - find managed block
  - extract context hash
  - recompute current context hash
  - warn if mismatch
```

Default: warning only.

Strict mode:

```bash
x-harness doctor --strict
```

May fail if stale.

Output:

```txt
warning: AGENTS.md is stale. Run: x-harness context --refresh
```

### Acceptance criteria

```txt
[ ] init creates AGENTS.md for new repo.
[ ] AGENTS.md includes context hashes.
[ ] Existing AGENTS.md is not overwritten silently.
[ ] Managed block update preserves user content.
[ ] doctor warns on stale context.
[ ] doctor --strict fails on stale context.
[ ] AGENTS.md remains short.
```

---

## 9. P0.4 — Auto-inject bounded context header into handoff

### Problem

Agents or users may forget to call `x-harness context`. Handoff output should be prompt-ready and include the core admission contract.

### Goal

All generated handoffs include a compact context header by default.

### Output example

```txt
# --- BEGIN X-HARNESS CONTEXT ---
Core: completion is admitted, not claimed.
Tier: standard.
Verify: completion-card.yaml -> x-harness verify.
Accepted: outcome=success and acceptance_status=accepted only.
Withheld: failed/blocked/skipped/timeout/error.
Verifier: read-only.
PGV/advisory: advisory-only.
Evidence: record changed files and validation artifacts.
# --- END X-HARNESS CONTEXT ---

## Task
Fix login timeout
```

### Flags

Add:

```bash
x-harness handoff standard --title "Fix login timeout" --no-context
x-harness handoff standard --title "Fix login timeout" --context-max-lines 12
```

Default bound:

```txt
max 12 lines
target <=200 tokens
```

Do not add:

```bash
handoff --format prompt
```

### Files

Modify:

```txt
packages/cli/src/commands/handoff.ts
packages/cli/src/core/context/generator.ts
packages/cli/tests/handoff.test.ts
docs/CONTEXT_POLICY.md
```

### Acceptance criteria

```txt
[ ] handoff includes context by default.
[ ] --no-context disables header.
[ ] --context-max-lines bounds the header.
[ ] Header includes accepted/withheld semantics.
[ ] Header includes read-only verifier.
[ ] Header includes PGV/advisory boundary.
[ ] No handoff --format prompt flag is added.
```

---

## 10. P0.5 — Read-only mutation guard for verify

### Problem

“Verifier is read-only” should be technically checked. Otherwise it remains a convention.

### Goal

Detect whether `x-harness verify` mutates repository files outside an allowlist.

### Default policy

Default:

```txt
warn on unexpected mutation
```

Strict:

```txt
fail on unexpected mutation
```

Recommended strict default in CI:

```bash
x-harness verify --strict
```

### Allowlist

Allowed writes during verify:

```txt
.x-harness/traces/**
.x-harness/reports/**
.x-harness/tmp/**
```

Only allow these if the command explicitly writes trace/report.

Not allowed:

```txt
src/**
packages/**
schemas/**
policies/**
templates/**
README.md
AGENTS.md
package.json
package-lock.json
pnpm-lock.yaml
yarn.lock
```

### Dirty repo handling

If repo is already dirty before verify:

```txt
- capture baseline git status
- capture tracked file hash map if feasible
- after verify, compare to baseline
- only report new mutations introduced by verify
```

Do not require clean working tree by default. In strict mode, allow user to opt in:

```bash
x-harness verify --strict --require-clean
```

### Suggested implementation

Create:

```txt
packages/cli/src/core/readonly/mutationGuard.ts
packages/cli/tests/readonly-mutation-guard.test.ts
```

Interface:

```ts
export interface MutationGuardOptions {
  root: string;
  allowGlobs: string[];
  requireClean?: boolean;
}

export interface MutationSnapshot {
  gitStatus: string[];
  trackedHashes?: Record<string, string>;
  untrackedFiles?: string[];
}

export interface MutationResult {
  mutated: boolean;
  added: string[];
  modified: string[];
  deleted: string[];
  allowed: string[];
  unexpected: string[];
}
```

Pseudocode:

```ts
const before = await captureMutationSnapshot(root);

const result = await runCoreVerify();

const after = await captureMutationSnapshot(root);
const diff = compareSnapshots(before, after);

const unexpected = diff.changed.filter(path => !matchesAllowlist(path));

if (unexpected.length > 0) {
  warnings.push({
    id: "verifier_mutated_files",
    severity: strict ? "error" : "warning",
    files: unexpected
  });

  if (strict) {
    return withheld("failed", "verifier_mutated_files");
  }
}
```

### Admission interaction

If mutation guard fails in strict mode:

```yaml
admission:
  outcome: failed
  acceptance_status: withheld
recovery:
  route: verifier_mutated_files
  owner: admission-verifier
  next_action: "Re-run verification without mutating source files or move side effects outside verify."
```

If non-strict:

```yaml
warnings:
  - id: verifier_mutated_files
```

### Acceptance criteria

```txt
[ ] verify captures before/after mutation snapshot.
[ ] unexpected source mutation is detected.
[ ] allowlisted trace/report writes do not fail.
[ ] dirty repo baseline does not create false positives.
[ ] --strict fails/withholds on mutation.
[ ] non-strict warns.
[ ] golden test covers mutated source file.
```

---

## 11. P0.6 — Denominator-safe report wording and metrics

### Problem

Verify-event success is not task-level success. Reports must not collapse event-level, task-level, and unknown denominator metrics.

### Goal

Make `report` explicitly denominator-safe.

### Required report sections

```txt
1. Verify event accounting
2. Task lifecycle accounting
3. Admission outcome accounting
4. Withheld/recovery accounting
5. Unknown denominator warnings
```

### Required metrics

```yaml
verify_event_accounting:
  verify_completed_events: 0
  known_outcome_verify_events: 0
  success_verify_events: 0
  blocked_verify_events: 0
  failed_verify_events: 0
  skipped_verify_events: 0
  timeout_verify_events: 0
  error_verify_events: 0
  missing_outcome_verify_events: 0
  verify_event_success_rate: null

task_lifecycle_accounting:
  task_created_events: 0
  unique_task_ids: 0
  tasks_with_at_least_one_verify: 0
  task_completed_events: 0
  task_completed_with_verify: 0
  task_level_coverage: null
  denominator_status: "unknown | aligned | partial"

admission_accounting:
  accepted_count: 0
  withheld_count: 0
  accepted_rate: null
  withheld_rate: null

denominator_warnings:
  - "Verify-event success is not task-level success."
  - "Task-level coverage requires aligned task_created/task_completed/verify_completed events."
```

### Report text

Use:

```txt
Verify-event success rate
```

Do not use:

```txt
Success rate
Task success rate
Reliability rate
```

unless the denominator supports it.

### Acceptance criteria

```txt
[ ] report distinguishes event-level and task-level metrics.
[ ] report includes denominator warnings.
[ ] no ambiguous “success rate” label.
[ ] JSON output exposes denominator_status.
[ ] tests cover missing denominator case.
```

---

# PART B — P1 Implementation

---

## 12. P1.1 — Evidence artifact hardening

### Problem

Evidence can be self-reported. A completion card that says “tests passed” is weaker than a completion card that records command, exit code, timestamps, and artifact hash.

### Goal

Strengthen evidence schema for standard/deep without making light tier heavy.

### Schema additions

Modify:

```txt
schemas/completion-card.schema.json
templates/COMPLETION_CARD.md
docs/ADMISSION_POLICY.md
```

Add optional-but-preferred fields for all tiers:

```yaml
verification_artifacts:
  - kind: typecheck
    command: npm run typecheck
    status: passed
    exit_code: 0
    cwd: .
    started_at: "2026-05-22T10:00:00Z"
    ended_at: "2026-05-22T10:00:15Z"
    stdout_hash: "sha256:..."
    stderr_hash: "sha256:..."
    artifact_path: ".x-harness/artifacts/typecheck.log"
    artifact_hash: "sha256:..."
    verifies:
      - "packages/cli/src/core/admission.ts"
    does_not_verify:
      - "runtime behavior in production"
```

### Tier rules

Light:

```txt
- Evidence may be concise.
- At least changed files and nearest validation statement.
- Provenance fields recommended, not required.
```

Standard:

```txt
- At least one verification artifact required.
- Each artifact should include command and status.
- If command was run locally, exit_code required.
- If artifact_path exists, artifact_hash required.
```

Deep:

```txt
- All standard requirements.
- Evidence scope required.
- Untested regions required, even if empty.
- Remaining risks required, even if empty.
- Read/write set required.
- For high-risk categories, human approval may be required by policy.
```

### Helper command: optional artifact capture

Add only if simple:

```bash
x-harness evidence run --kind typecheck -- npm run typecheck
```

This command can:

```txt
- run command
- capture stdout/stderr under .x-harness/artifacts/
- compute hashes
- output YAML snippet for completion-card.yaml
```

If this command expands too much, defer. Do not block schema changes on it.

### Acceptance criteria

```txt
[ ] schema supports command, exit_code, timestamps, artifact hashes.
[ ] standard tier requires stronger evidence than light.
[ ] deep tier requires evidence scope and risks.
[ ] invalid evidence produces withheld outcome.
[ ] template shows provenance example.
[ ] tests cover missing exit_code/artifact_hash for standard/deep.
```

---

## 13. P1.2 — Predicate tiering: Required / Conditional / Advisory

### Problem

Paper-style acceptance predicates are useful, but hard-checking all predicates for every task would make light tier too heavy.

### Goal

Refactor admission checks into:

```txt
Required
Conditional
Advisory
```

### Required predicates

Required failure blocks success.

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

Conditional failure blocks success only when condition applies.

```txt
deep_escalation_path
rollback_policy_present
human_approval_present
security_or_deploy_review_present
evidence_scope_present
read_write_set_present
```

Apply when:

```txt
tier: deep
risk_class: high
task touches auth/payment/database/deploy/security
policy requires it
```

### Advisory predicates

Advisory failure creates warnings only by default.

```txt
context_acknowledged
context_hash_current
stale_ground_heuristic
pgv_warning_treated
git_context_inherited
veto_condition_scan
```

### Suggested file structure

```txt
packages/cli/src/core/predicates/
  types.ts
  required.ts
  conditional.ts
  advisory.ts
  index.ts
```

### Predicate result interface

```ts
export type PredicateTier = "required" | "conditional" | "advisory";

export interface PredicateResult {
  id: string;
  tier: PredicateTier;
  passed: boolean;
  applicable: boolean;
  severity: "error" | "warning" | "info";
  reason?: string;
  recoveryRoute?: string;
  evidence?: unknown;
}
```

### Admission mapping

If required fails:

```yaml
admission:
  outcome: blocked | failed
  acceptance_status: withheld
```

If conditional applies and fails:

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
```

If advisory fails:

```yaml
warnings:
  - id: context_acknowledged
    message: "context was not explicitly acknowledged"
```

Do not block success unless policy explicitly promotes advisory.

### Policy YAML

Extend:

```yaml
predicates:
  required:
    - claim_packet_valid
    - verify_invoked
    - evidence_floor_met
    - owner_accountable_present
    - no_unresolved_blocker
    - no_active_recovery
    - admission_mapping_valid
    - verifier_read_only

  conditional:
    deep:
      - evidence_scope_present
      - read_write_set_present
      - rollback_policy_present
    high_risk:
      - human_approval_present
      - security_or_deploy_review_present

  advisory:
    - context_acknowledged
    - context_hash_current
    - stale_ground_heuristic
    - pgv_warning_treated
```

### Acceptance criteria

```txt
[ ] required predicate failure withholds completion.
[ ] conditional predicate failure applies only when condition matches.
[ ] advisory predicate failure creates warning only.
[ ] light tier remains low ceremony.
[ ] deep tier enforces configured deep/high-risk predicates.
[ ] predicate breakdown appears in verify JSON/report.
[ ] tests cover pass/fail/applicability cases.
```

---

## 14. P1.3 — Policy-code drift guard

### Problem

If `admission.yaml` and TypeScript admission logic drift, users cannot trust policy as the contract.

### Goal

Add doctor/lint check that verifies policy predicates and code predicates are synchronized.

### Command

```bash
x-harness doctor --policy-drift
x-harness doctor --strict --policy-drift
```

Or include in default doctor as warning.

### Checks

```txt
- Every predicate implemented in code is declared in admission.yaml.
- Every predicate declared in admission.yaml has implementation.
- Every required predicate has at least one test/golden case.
- Every conditional predicate declares applicability.
- Every advisory predicate declares warning behavior.
- verify output includes policy_hash.
```

### Files

```txt
packages/cli/src/core/policy/drift.ts
packages/cli/src/core/policy/load.ts
packages/cli/src/commands/doctor.ts
packages/cli/tests/policy-drift.test.ts
docs/ADMISSION_POLICY.md
```

### Implementation approach

Avoid heavy policy engine initially. Do not add OPA/Rego.

Use a lightweight manifest:

```ts
export const implementedPredicates = {
  claim_packet_valid: runClaimPacketValid,
  verify_invoked: runVerifyInvoked,
  evidence_floor_met: runEvidenceFloorMet,
  ...
};
```

Doctor compares:

```txt
Object.keys(implementedPredicates)
vs
flatten(policy.predicates)
```

### Acceptance criteria

```txt
[ ] doctor reports missing predicate implementation.
[ ] doctor reports implementation not declared in policy.
[ ] strict mode fails on drift.
[ ] verify JSON includes policy_hash.
[ ] docs state whether YAML is source-of-truth or synchronized contract.
```

---

## 15. P1.4 — Recovery golden test suite

### Problem

Recovery routing should be deterministic and regression-tested. A simulator is unnecessary and would add nondeterminism.

### Goal

Add golden recovery cases.

### Directory

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
  09-verifier-mutated-files/
  10-policy-drift/
```

Each case:

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

### CLI integration

`x-harness examples verify` should include these cases.

Optional future:

```bash
x-harness test recovery
```

Do not add separate command unless it keeps code simpler.

### CI

CI must run:

```bash
npm test
npx x-harness examples verify
```

### Acceptance criteria

```txt
[ ] At least 10 recovery golden cases.
[ ] Each case checks outcome.
[ ] Each case checks recovery route.
[ ] Each case checks owner.
[ ] Each case checks next_action.
[ ] examples verify runs recovery cases.
[ ] CI includes recovery cases.
[ ] No recovery simulator added.
```

---

## 16. P1.5 — Task lifecycle trace ledger

### Problem

Verify events alone cannot support task-level coverage or task-level success claims. Need aligned lifecycle events.

### Goal

Add minimal task lifecycle event schema.

### Events

```txt
task_created
claim_created
verify_started
verify_completed
recovery_created
task_completed
task_abandoned
task_failed
```

### Event fields

```json
{
  "schema_version": "1.0",
  "timestamp": "2026-05-22T10:00:00Z",
  "event": "verify_completed",
  "trace_id": "trace-...",
  "span_id": "span-...",
  "parent_span_id": "span-...",
  "task_id": "task-123",
  "claim_id": "claim-123",
  "tier": "standard",
  "agent_role": "admission-verifier",
  "admission": {
    "outcome": "blocked",
    "acceptance_status": "withheld"
  },
  "policy_hash": "sha256:...",
  "context_hash": "sha256:..."
}
```

### File location

Prefer one current canonical path:

```txt
.x-harness/trace.jsonl
```

If current repo uses:

```txt
.x-harness/traces/events.jsonl
```

Then either preserve and document it, or add compatibility alias. Avoid having two competing trace paths.

### Report usage

Report should calculate:

```txt
tasks_with_at_least_one_verify
task_completed_with_verify
tasks_with_recovery
task_completed_after_recovery
unknown_denominator_count
```

Only calculate task-level coverage if:

```txt
task_created events exist
task_id is stable
verify_completed events link to task_id
task_completed events link to task_id
```

### Acceptance criteria

```txt
[ ] trace emits lifecycle events when trace enabled.
[ ] verify_completed links to task_id and claim_id.
[ ] report uses lifecycle events when available.
[ ] report warns when denominator is incomplete.
[ ] tests cover multi-verify same task.
[ ] tests cover blocked then success same task.
```

---

## 17. P1.6 — OTel-compatible JSONL naming

### Problem

Trace should be local JSONL, but field names should be easy to export to OpenTelemetry later.

### Goal

Use OTel-friendly naming without adding collector/service dependencies.

### Naming recommendation

Use both ergonomic and nested fields if needed:

```json
{
  "trace_id": "...",
  "span_id": "...",
  "parent_span_id": "...",
  "event.name": "verify_completed",
  "task.id": "task-123",
  "agent.role": "admission-verifier",
  "tool.name": "x-harness verify",
  "admission.outcome": "blocked",
  "admission.acceptance_status": "withheld",
  "artifact.hash": "sha256:..."
}
```

Or nested version:

```json
{
  "trace": { "id": "...", "span_id": "...", "parent_span_id": "..." },
  "event": { "name": "verify_completed" },
  "task": { "id": "task-123" },
  "agent": { "role": "admission-verifier" },
  "admission": { "outcome": "blocked", "acceptance_status": "withheld" }
}
```

Pick one style and document it.

### Do not add

```txt
- OpenTelemetry collector
- remote exporter
- service dependency
- dashboard
```

Optional future:

```bash
x-harness trace export --format otel-json
```

### Acceptance criteria

```txt
[ ] trace schema documented.
[ ] trace events include trace_id/span_id.
[ ] fields map cleanly to task/admission/artifact concepts.
[ ] no external observability service added.
```

---

## 18. P1.7 — Human approval only for deep/high-risk

### Problem

Human approval improves safety on sensitive tasks but creates friction if applied to every task.

### Goal

Make human approval conditional.

### Policy

Require human approval when:

```txt
tier: deep
risk_class: high
task touches auth/payment/database/deploy/security
policy.requires_human_approval: true
```

Do not require human approval for light tier.

### Schema

```yaml
human_approval:
  required: true
  approved: false
  approver: null
  approved_at: null
  reason: "Touches deploy pipeline"
```

### Admission behavior

If approval required but missing:

```yaml
admission:
  outcome: blocked
  acceptance_status: withheld
recovery:
  route: approval_missing
  owner: human-reviewer
  next_action: "Obtain explicit approval for high-risk/deep task before admission."
```

### Acceptance criteria

```txt
[ ] light tier does not require approval by default.
[ ] standard tier does not require approval unless policy/risk triggers.
[ ] deep/high-risk can require approval.
[ ] missing approval withholds completion.
[ ] report shows approval predicate.
```

---

## 19. P1.8 — `context_acknowledged` advisory-only

### Problem

It is useful to know whether agent used current context, but blocking completion based on a checkbox can become theater.

### Goal

Add optional metadata and warning only.

### Schema

```yaml
metadata:
  context_acknowledged: true
  context_hash: "sha256:..."
```

### Behavior

If missing:

```txt
warning: context was not explicitly acknowledged
```

If hash mismatch:

```txt
warning: context hash does not match current x-harness context
```

Do not block by default.

Strict mode may optionally promote later, but not in P1 unless explicitly configured:

```yaml
policy:
  promote_advisory:
    - context_hash_current
```

### Acceptance criteria

```txt
[ ] metadata.context_acknowledged is optional.
[ ] missing field creates warning only.
[ ] hash mismatch creates warning only.
[ ] success is not blocked by default.
[ ] docs explain advisory-only semantics.
```

---

## 20. P1.9 — Git context inheritance, opt-in only

### Problem

Recent harness policy decisions may be in commit history, but dumping Git history into context is noisy and risky.

### Goal

Add opt-in Git context inheritance.

### Command

```bash
x-harness context --inherit-from-git --last 5
```

### Include only commit messages with prefix

```txt
[x-harness]
[xh]
```

Example:

```txt
[x-harness] Raised standard evidence floor to require typecheck
[xh] Switched default test command from Jest to Vitest
```

### Output

```txt
# Recent x-harness decisions
- Raised standard evidence floor to require typecheck (abc123)
- Switched default test command from Jest to Vitest (def456)
```

### Constraints

```txt
max commits: 5 by default
max tokens: roughly 100
do not include full commit body by default
do not auto-inject into handoff by default
```

### Acceptance criteria

```txt
[ ] only prefixed commits are included.
[ ] output is short.
[ ] feature is opt-in.
[ ] no Git history is injected by default.
[ ] tests cover prefix filtering.
```

---

# PART C — P2 Implementation

---

## 21. P2.1 — Git-native packet chain verify

### Problem

Packetized state improves auditability, but custom packet store/database would be overengineering.

### Goal

Use file-based packets and Git-compatible content hashes.

### Layout

```txt
.x-harness/
  packets/
    control/
    claims/
    evidence/
    recovery/
  trace.jsonl
```

### Packet schema

```yaml
packet_id: claim-2026-05-22T10-15-30-task-123
task_id: task-123
type: claim
owner: implementation-worker
accountable: repository-owner
parent_packet: control-2026-05-22T10-00-01-task-123
created_at: 2026-05-22T10:15:30Z
content_hash: sha256:...
policy_hash: sha256:...
context_hash: sha256:...
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
do not git add
do not git commit
```

Explicit only:

```bash
x-harness packet create claim --task task-123 --git-add
x-harness packet create claim --task task-123 --git-commit
```

### Chain verification

Check:

```txt
- packet file exists
- content_hash matches content
- parent_packet exists unless root/control
- task_id consistent across chain
- owner present
- no cycle
- verify_completed links to claim/evidence packet if trace exists
```

### Acceptance criteria

```txt
[ ] packet files created under .x-harness/packets.
[ ] verify-chain checks parent lineage.
[ ] verify-chain catches missing parent.
[ ] verify-chain catches hash mismatch.
[ ] no auto-commit by default.
[ ] --git-add and --git-commit are explicit.
```

---

## 22. P2.2 — Static single-file HTML audit report

### Goal

Provide an offline audit artifact without dashboard/server.

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
- report generated_at
- repo path / git commit if available
- card id
- task id
- tier
- policy hash
- context hash
- predicate breakdown
- outcome / acceptance_status
- evidence summary
- artifact hashes
- warnings
- recovery route
- trace timeline if trace exists
- denominator warning
```

### HTML security

Escape all user-provided content. Do not render raw HTML from cards/traces.

### Acceptance criteria

```txt
[ ] report --format html works.
[ ] output is one standalone HTML file.
[ ] no external assets.
[ ] denominator warning included.
[ ] predicate breakdown included.
[ ] evidence summary included.
[ ] user-provided strings escaped.
```

---

## 23. P2.3 — Optional sandbox bridge

### Problem

Sandboxed execution is useful, but implementing a sandbox in `x-harness` would bloat scope.

### Goal

Support optional external sandbox command hook.

### Commands

```bash
x-harness verify --sandbox-command "docker run --rm -v $PWD:/work -w /work node:20 npm test"
x-harness doctor --check-sandbox
```

Alternative config:

```yaml
sandbox:
  enabled: false
  command: "docker run --rm -v $PWD:/work -w /work node:20"
```

### Constraints

```txt
- no built-in sandbox runtime
- no Docker dependency by default
- no Kubernetes/cloud integration in core
- sandbox bridge only runs explicitly configured commands
```

### Acceptance criteria

```txt
[ ] verify can invoke configured sandbox command.
[ ] sandbox is disabled by default.
[ ] doctor can validate sandbox config if present.
[ ] failure in sandbox produces evidence/recovery route.
[ ] no Docker/cloud dependency added to core.
```

---

## 24. P2.4 — Human handbook docs, not heavy agent context

### Goal

Add comprehensive human docs without bloating `AGENTS.md` or handoff context.

### Files

```txt
docs/HANDBOOK.md
docs/ARCHITECTURE.md
docs/ADMISSION_POLICY.md
docs/CONTEXT_POLICY.md
docs/RECOVERY.md
docs/TRACE_SCHEMA.md
docs/CI.md
```

### Constraints

```txt
README remains short.
AGENTS.md remains short.
handoff context remains short.
Docs are for humans and deep reference.
```

### Doctor checks

```txt
- AGENTS.md line count <= 150 warning
- handoff context max lines respected
- docs links from README valid if link checker exists
```

### Acceptance criteria

```txt
[ ] docs exist.
[ ] README links to docs.
[ ] AGENTS.md links to docs but does not duplicate.
[ ] doctor warns if AGENTS.md too long.
```

---

## 25. P2.5 — Static recovery playbook suggestions from trace

### Problem

Trace can help improve recovery guidance, but automatic learning/policy mutation is risky.

### Goal

Add deterministic suggestion command.

### Command

```bash
x-harness recovery suggest-playbook --from .x-harness/trace.jsonl
```

Optional write:

```bash
x-harness recovery suggest-playbook --from .x-harness/trace.jsonl --write --force
```

### Behavior

Default:

```txt
print candidate suggestions only
do not mutate policy
```

With `--write --force`:

```txt
write candidate file under .x-harness/suggestions/recovery-playbook.yaml
```

Do not directly mutate `policies/recovery.yaml` unless command explicitly says so and user confirms.

### Suggested output

```yaml
suggestions:
  - route: evidence_missing
    observed_count: 12
    proposed_next_action: "Attach validation artifact with command, exit_code, and hash."
    confidence: "medium"
    source: "trace.jsonl"
```

### Acceptance criteria

```txt
[ ] command outputs deterministic candidate suggestions.
[ ] no automatic policy mutation by default.
[ ] --write requires --force.
[ ] suggestions include source counts.
[ ] tests cover deterministic output.
```

---

# PART D — P3 Deferred/Conditional Features

---

## 26. P3.1 — MCP read-only adapter

### Decision

Do not implement in P0/P1/P2 unless there is explicit demand.

### If implemented

Constraints:

```txt
- read-only resources/tools only by default
- disabled by default
- allowlist tool names
- no admission authority
- no mutation tools
- no automatic trust propagation
- trace every MCP access
```

### Example config

```yaml
mcp:
  enabled: false
  mode: read-only
  allowed_servers:
    - ci-status
  allowed_tools:
    - get_ci_run
    - get_test_artifact
  admission_authority: false
```

### Acceptance criteria if implemented

```txt
[ ] disabled by default.
[ ] read-only mode enforced.
[ ] allowlist required.
[ ] MCP outputs treated as evidence inputs, not admission authority.
[ ] MCP access appears in trace.
```

---

## 27. P3.2 — YAML custom checks before plugin system

### Goal

Allow lightweight extension without plugin API.

### Config

```yaml
custom_checks:
  - id: security_audit_present
    required_for_tiers: [deep]
    evidence_kind: security_scan
    recovery_route: security_evidence_missing
```

### Constraints

```txt
- declarative only
- no arbitrary code execution
- no marketplace
- no external network
```

### Acceptance criteria

```txt
[ ] custom YAML checks can require evidence kind.
[ ] custom check failure maps to withheld/recovery.
[ ] no arbitrary code execution.
```

---

## 28. P3.3 — Plugin system deferred

### Decision

Do not build plugin system unless real users require it.

Reasons:

```txt
- API stability burden
- security questions
- arbitrary code execution risk
- support burden
- scope creep toward framework
```

If implemented later:

```txt
- file-based only
- local trusted code only
- disabled by default
- explicit enable in .x-harness/plugins.yaml
- no marketplace
- no remote install
```

---

# PART E — CI and test plan

---

## 29. CI baseline

Update or confirm CI runs:

```bash
npm ci
npm run typecheck
npm run build
npm test
npx x-harness doctor --root .
npx x-harness examples verify
```

If using package workspaces, adapt commands to current repo.

### Add CI checks after implementation

```bash
npx x-harness context --json
npx x-harness doctor --policy-drift
npx x-harness examples verify
```

Optional strict check:

```bash
npx x-harness verify --strict
```

Only if repo has stable completion card in CI.

---

## 30. Required tests by feature

### Context tests

```txt
[ ] default output contains core rule.
[ ] default output under line/token limit.
[ ] --verbose includes evidence/tier guidance.
[ ] --json parses as JSON.
[ ] hash changes when policy file changes.
[ ] --refresh updates AGENTS.md managed block.
[ ] existing AGENTS.md user content preserved.
```

### Handoff tests

```txt
[ ] context header included by default.
[ ] --no-context disables header.
[ ] --context-max-lines respected.
[ ] no --format prompt flag documented/accepted.
```

### Mutation guard tests

```txt
[ ] no mutation passes.
[ ] allowlisted trace write passes.
[ ] source mutation warns in non-strict.
[ ] source mutation fails in strict.
[ ] dirty baseline does not false-positive.
```

### Evidence tests

```txt
[ ] light permits concise evidence.
[ ] standard requires artifact.
[ ] standard command artifact missing exit_code fails if command-run evidence.
[ ] deep requires evidence scope.
[ ] deep requires read/write set.
[ ] artifact hash mismatch fails.
```

### Predicate tests

```txt
[ ] each required predicate has pass/fail case.
[ ] conditional predicate not applicable does not block.
[ ] conditional predicate applicable and failing blocks.
[ ] advisory predicate warning does not block by default.
[ ] policy can promote advisory only if explicitly configured.
```

### Policy drift tests

```txt
[ ] policy declares missing implementation -> doctor warns/fails strict.
[ ] implementation missing from policy -> doctor warns/fails strict.
[ ] policy hash appears in verify output.
```

### Recovery golden tests

```txt
[ ] all golden cases run.
[ ] expected outcome matches.
[ ] expected recovery route matches.
[ ] expected owner matches.
[ ] expected next_action matches.
```

### Trace/report tests

```txt
[ ] lifecycle events serialize as JSONL.
[ ] multi-verify same task counted correctly.
[ ] blocked then success not counted as two tasks.
[ ] report labels verify-event success correctly.
[ ] report warns on unknown denominator.
[ ] html report escapes user strings.
```

---

# PART F — Suggested implementation sequence for an AI coding agent

---

## 31. Phase 0 — Repo reconnaissance

1. Run baseline commands.
2. Inspect existing CLI architecture.
3. Identify command registration pattern.
4. Identify test style.
5. Identify trace/report output formats.
6. Create small plan.
7. Do not implement all features in one PR.

Suggested first PR split:

```txt
PR 1: wording/overclaim + denominator-safe report wording.
PR 2: context command + AGENTS freshness hash.
PR 3: handoff context injection.
PR 4: read-only mutation guard.
PR 5: predicate tiering + policy drift guard.
PR 6: evidence provenance + recovery golden tests.
PR 7: trace lifecycle + HTML report.
```

---

## 32. Phase 1 — P0 implementation

### Step 1: Claim boundary wording

- Search for overclaim phrases.
- Replace with admission-control wording.
- Add evidence boundary.
- Add metric boundary.
- Run tests.

### Step 2: Context core

- Add context generator.
- Add context hash module.
- Add context command.
- Register command.
- Add tests.

### Step 3: AGENTS refresh

- Add managed block logic.
- Update init behavior.
- Update doctor freshness check.
- Add tests.

### Step 4: Handoff context

- Inject generated context header by default.
- Add `--no-context`.
- Add max-lines bound.
- Add tests.

### Step 5: Mutation guard

- Implement snapshot/compare.
- Integrate into verify.
- Add strict behavior.
- Add recovery route.
- Add tests.

### Step 6: Denominator-safe report

- Add metric labels.
- Add denominator warnings.
- Add JSON fields.
- Add tests.

---

## 33. Phase 2 — P1 implementation

### Step 1: Predicate tiering

- Create predicate modules.
- Move admission checks gradually.
- Keep existing outputs compatible.
- Add predicate breakdown to verify JSON.

### Step 2: Policy drift guard

- Add predicate manifest.
- Add doctor check.
- Add strict mode.
- Add tests.

### Step 3: Evidence provenance

- Extend schema.
- Extend template.
- Update admission evidence floor.
- Add tests.

### Step 4: Recovery golden suite

- Add cases.
- Extend examples verify.
- Ensure CI runs.

### Step 5: Trace lifecycle

- Add event schema.
- Add lifecycle event emitters.
- Update report.
- Add tests.

---

## 34. Phase 3 — P2 implementation

Only after P0/P1 are stable.

1. Add packet chain.
2. Add HTML report.
3. Add optional sandbox bridge.
4. Add docs.
5. Add recovery suggestion command.

---

# PART G — Output examples

---

## 35. Expected `x-harness context --json`

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
  "evidence_policy": "record changed files and validation artifacts",
  "context_hash": "sha256:example",
  "core_context_hash": "sha256:example-core",
  "adapter_context_hash": null
}
```

---

## 36. Expected verify JSON predicate breakdown

```json
{
  "admission": {
    "outcome": "blocked",
    "acceptance_status": "withheld"
  },
  "policy_hash": "sha256:...",
  "context_hash": "sha256:...",
  "predicates": [
    {
      "id": "claim_packet_valid",
      "tier": "required",
      "applicable": true,
      "passed": true
    },
    {
      "id": "evidence_floor_met",
      "tier": "required",
      "applicable": true,
      "passed": false,
      "severity": "error",
      "reason": "standard tier requires at least one verification artifact",
      "recoveryRoute": "evidence_missing"
    },
    {
      "id": "context_acknowledged",
      "tier": "advisory",
      "applicable": true,
      "passed": false,
      "severity": "warning",
      "reason": "context was not explicitly acknowledged"
    }
  ],
  "warnings": [
    {
      "id": "context_acknowledged",
      "message": "context was not explicitly acknowledged"
    }
  ],
  "recovery": {
    "route": "evidence_missing",
    "owner": "implementation-worker",
    "next_action": "Attach validation evidence or explain why unavailable."
  }
}
```

---

## 37. Expected denominator-safe report excerpt

```txt
# x-harness report

## Verify event accounting

Known-outcome verify events: 42
Success verify events: 37
Blocked verify events: 5
Verify-event success rate: 37/42 = 88.10%

This is an event-level accounting measure. It is not a task-level success rate.

## Task lifecycle accounting

Unique task IDs: 18
Tasks with at least one verify: 16
Tasks completed with verify: 12
Task-level coverage: not computed

Warning: task-level coverage requires aligned task_created, verify_completed,
and task_completed events.
```

---

## 38. Expected mutation guard warning

```txt
warning: verifier mutated files outside allowlist:
  - packages/cli/src/core/admission.ts

Verification must be read-only. Re-run verification without modifying source
files, or move repair work outside the verify step.
```

Strict output:

```yaml
admission:
  outcome: failed
  acceptance_status: withheld
recovery:
  route: verifier_mutated_files
  owner: admission-verifier
  next_action: "Re-run verification without mutating source files."
```

---

# PART H — Definition of Done

The implementation is acceptable when:

```txt
1. Minimal mode remains lightweight.
2. README/report no longer overclaim correctness.
3. `x-harness context` exists and is short by default.
4. `AGENTS.md` has managed context hash and doctor freshness check.
5. Handoff includes bounded context by default.
6. `--no-context` works.
7. Verify read-only mutation guard exists.
8. Denominator-safe report separates event-level and task-level metrics.
9. Predicate tiers preserve fail-closed admission.
10. Evidence provenance exists for standard/deep.
11. Policy-code drift guard exists.
12. Recovery golden tests run in CI.
13. Task lifecycle trace events exist when tracing is enabled.
14. JSONL trace schema is documented.
15. context_acknowledged remains advisory-only by default.
16. Git context inheritance is opt-in only.
17. No daemon, DB, dashboard server, mandatory MCP, mandatory LLM verifier, or plugin marketplace is introduced.
18. Git integration never commits by default.
19. HTML report, if added, is static and optional.
20. Plugin system remains deferred unless explicitly demanded.
```

---

# PART I — Anti-regression checklist

Before merging any change, verify:

```txt
[ ] `npm run typecheck` passes.
[ ] `npm run build` passes.
[ ] `npm test` passes.
[ ] `x-harness doctor --root .` passes or only expected warnings.
[ ] `x-harness examples verify` passes.
[ ] Existing CLI commands still work.
[ ] No new default daemon/service/database.
[ ] No new mandatory external dependency.
[ ] No new hidden network call.
[ ] No auto-commit behavior.
[ ] Non-success outcomes remain withheld.
[ ] PGV/advisory does not override admission.
[ ] README does not claim correctness guarantees.
```

---

# PART J — Agent instruction block

Use this block when assigning the implementation to a coding agent.

```txt
You are implementing x-harness improvements.

Do not turn x-harness into an agent runtime, workflow engine, dashboard,
database-backed system, benchmark framework, product-planning harness, or
plugin marketplace.

Do not import improvements from harness-experimental. Do not implement feature
intake, story packets, product docs, test matrix lifecycle, or decision-record
lifecycle.

Preserve the core contract:
- completion is admitted, not claimed
- worker may propose completion, but cannot self-admit
- verifier is read-only
- success is the only accepted outcome
- failed/blocked/skipped/timeout/error are withheld
- PGV/advisory checks are advisory-only by default
- deep mode is opt-in or risk-triggered

Implement in small PR-sized steps:
1. Fix overclaim wording and denominator-safe report labels.
2. Add x-harness context and AGENTS.md freshness hash.
3. Add handoff context injection.
4. Add read-only mutation guard.
5. Add predicate tiering and policy-code drift guard.
6. Add evidence provenance fields and recovery golden tests.
7. Add task lifecycle trace ledger and optional static HTML report.

Every feature must answer yes to:
Does this make completion harder to fake, easier to verify, and safer to
withhold without adding default process weight?

Run typecheck/build/test/doctor/examples verify before final response.
```

---

## 39. Final strategic note

`x-harness` should not win by having the most features. It should win by having the sharpest admission boundary:

```txt
completion claims become auditable
verification remains read-only
ambiguous cases fail closed
evidence is harder to fake
context stays short and fresh
reports do not overclaim
recovery has owner and next action
```

That is the highest-value direction with the least operational trade-off.
