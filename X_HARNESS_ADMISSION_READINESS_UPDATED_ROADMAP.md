# x-harness — Updated Admission/Readiness Positioning and Implementation Roadmap

**Status:** mixed — reflects both implemented capabilities and forward-looking design. Sections marked `Implemented` describe code present in the repository as of the latest commit. Sections marked `Planned` are design targets and must not be treated as shipped features. Do not overclaim roadmap items as currently available.

**Audience:** AI coding agent, maintainer, implementation reviewer

**Scope:** update and extend `x-harness` after the Go rewrite while preserving its core identity as a file-first, deterministic-first admission/readiness harness.

**Non-scope:** this document does not turn `x-harness` into an agent runtime, product-planning framework, dashboard platform, hosted service, database-backed workflow engine, MCP platform, or plugin marketplace.

---

## 1. Strategic positioning

### 1.1 One-line positioning

`x-harness` is a Go-native, file-first admission and readiness harness for AI coding workflows.

A more explicit formulation:

```txt
x-harness does not run the agent.
x-harness does not replace CI.
x-harness does not guarantee that code is correct.

x-harness turns an AI coding agent's completion claim into an auditable admission decision:
accepted or withheld.
```

Vietnamese positioning:

```txt
x-harness là harness kiểm nhận hoàn thành và readiness, file-first, chạy bằng Go native,
dành cho workflow AI coding.

Nó không chạy agent và không thay CI.
Nó biến claim “đã xong” của AI coding agent thành quyết định kiểm nhận có thể audit:
accepted hoặc withheld.
```

### 1.2 Correct mental model

`x-harness` should sit in the AI coding stack as an **admission/readiness layer**:

```txt
AI coding agent / IDE / agent CLI
  Claude Code / Cursor / OpenCode / Antigravity / Codex CLI / other agent harnesses

Execution layer
  shell / tests / build / lint / typecheck / CI / optional sandbox

Admission and readiness layer
  x-harness

Repository truth
  completion-card.yaml
  policies/*.yaml
  schemas/*.json
  AGENTS.md
  adapter managed blocks
  trace JSONL
  packet files
  readiness reports
```

The agent performs the work. `x-harness` evaluates whether the completion claim is admissible under repository policy.

### 1.3 Core admission flow

```txt
agent work
  -> completion claim
  -> evidence
  -> read-only verify gate
  -> accepted or withheld
  -> trace / report / recovery / release evidence
```

Accepted completion is valid only when the admission contract says it is valid:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

The following are **not** sufficient by themselves:

```txt
fix_status: fixed
verification.status: passed
tests passed
agent confidence: high
PGV says okay
context_acknowledged: true
LLM judge says okay
```

### 1.4 Non-goals to keep explicit

`x-harness` should not become:

```txt
- agent runtime
- ReAct loop framework
- multi-agent orchestrator
- LLM provider abstraction
- MCP platform
- product planning system
- issue tracker
- backlog manager
- dashboard/server platform
- database-backed workflow engine
- deployment automation platform
- generic benchmark framework
- skill marketplace
- plugin marketplace
```

It can integrate with systems that provide those capabilities, but it should not own them.

---

## 2. Current implementation status

This table separates what is already in the repository from what remains design-only. Use it to avoid overclaiming in README, docs, and adapter instructions.

| Feature | Status | Notes |
|---|---|---|
| Go-native CLI (verify, doctor, report, benchmark, trace, handoff, init, add, recovery, packet, context, clean, examples, and advanced commands) | **Implemented** | Primary commands stable; some advanced commands are skeletal |
| TypeScript compatibility CLI | **Implemented (frozen)** | Source-checkout only; npm wrapper is Go-only |
| Tiered completion cards (light / standard / deep) | **Implemented** | `schemas/completion-card.schema.json` enforced |
| Read-only verify gate with mutation guard | **Implemented** | `--strict` path active in CI |
| Admission policy (`policies/admission.yaml`) | **Implemented** | Single source of truth for evidence floor |
| Recovery routing | **Implemented** | Basic routing with `next_action` and `owner` |
| Packet chain | **Implemented** | Immutable claim packets |
| Frozen artifacts, import/export | **Implemented** | |
| Golden examples (success, blocked, recovery) | **Implemented** | CI-enforced |
| Adversarial benchmark | **Implemented** | CI-enforced |
| Release candidate cycle | **Implemented** | Cross-platform smoke tests, checksums, cosign |
| NPM wrapper Go-only | **Implemented** | Phase 11 complete (`15afed0`) |
| Adapter files (Generic, Claude Code, Cursor, OpenCode, Antigravity) | **Implemented** | Manual; automated drift detection not yet built |
| Denominator-safe report JSON | **Implemented (minimal)** | Structured rate metrics with numerator/denominator/unit; task_completion_coverage not_computable; denominator_warning preserved |
| Failure taxonomy v2 | **Partial** | Basic predicates exist; typed taxonomy classes not yet implemented |
| AdmissionCard / X-HarnessCard | **Implemented (minimal)** | `card generate`, `card verify`, `schemas/admission-card.schema.json` |
| Readiness levels (task / PR / release) | **Implemented (minimal)** | `readiness task/pr/release`; `prepare` alias unchanged |
| Conformance suite | **Implemented (minimal)** | `conformance run --profile minimal` with CI gate |
| Release evidence bundle | **Implemented (minimal)** | Schema, generator, verify-evidence, and report implemented; SBOM / provenance / platform matrix remain planned (Section 11) |
| Denominator contract in reports | **Planned** | Section 12 |
| Failure taxonomy v2 | **Partial** | Section 13 |
| Permission intent classifier | **Implemented (minimal)** | `evidence classify --command` and `--card` implemented; admission blocking and report integration deferred (Section 14) |
| Approval receipt schema | **Planned** | Section 15 |
| Adapter matrix / eval / doctor | **Partial** | `adapters matrix`, `adapters eval`, and `adapters doctor` implemented; managed block drift checks implemented; strict conformance profile and adapter file generation remain planned (Section 16) |
| Admission skill-pack | **Planned / Conditional** | Section 17; P3 unless demand exists |
| Adapter/skill static scanner | **Planned** | Section 18 |
| Install profiles preview/apply | **Planned** | Section 19 |
| Profile recommend | **Planned** | Section 20 |
| Repair / uninstall preview/apply | **Planned** | Section 21 |
| Trace timeline / explain | **Planned** | Section 22 |
| Structured regression / capability / adversarial suites | **Partial** | Section 23; golden examples exist |
| Worktree-aware verification | **Planned** | Section 24 |
| Context GC / staleness doctor | **Planned** | Section 25 |
| Hooks bridge | **Planned / Conditional** | Section 26; P3 unless needed |
| MCP read-only evidence adapter | **Planned / Conditional** | Section 27; P3 unless needed |
| Sandbox bridge | **Planned / Conditional** | Section 28; P3 unless needed |

---

## 3. Lessons from broader harness engineering

### 3.1 Why the term "harness" needs a narrower local definition

Current harness engineering discussions use "harness" broadly. Depending on the source, an agent harness may include prompt files, tools, permissions, memory, MCP servers, skills, hooks, orchestration, sandboxing, observability, workflow state, evals, and human-in-the-loop controls.

That broad definition is useful for the field, but too broad for `x-harness`.

For `x-harness`, use this bounded definition:

```txt
General agent harness:
  model + context + tools + memory + runtime + permissions + orchestration + observability

Coding-agent ecosystem harness:
  agents + skills + hooks + rules + MCP + workflows + adapters

x-harness:
  completion claim + evidence + read-only verify gate + admission decision + readiness report
```

This narrow definition is the product boundary.

### 3.2 What to learn from ECC

ECC is a broad agent-harness ecosystem. Its useful lessons for `x-harness` are not "add many agents and skills." The useful lessons are packaging, compatibility, selective installation, adapter discipline, and operational repair.

Apply these lessons:

```txt
- selective profiles
- preview/apply install plan
- adapter compatibility matrix
- adapter eval
- namespaced managed files
- repair/uninstall flow
- optional hooks bridge
- minimal admission skill-pack
- adapter/skill static scanner
- permission intent classification
```

Do not copy these into core:

```txt
- many agents/skills/commands
- SQLite session store as core
- MCP auto-enable
- plugin marketplace
- full lifecycle platform
- LLM runtime/orchestrator
```

### 3.3 What to learn from OpenAI-style harness engineering

OpenAI-style harness engineering emphasizes that the repository should become a system of record for agents. A huge `AGENTS.md` is not ideal. `AGENTS.md` should act as a short map, while versioned repository files and structured docs hold the durable truth.

For `x-harness`, this means:

```txt
- AGENTS.md should stay short.
- Managed blocks should have hashes.
- Adapter instructions should be generated from the canonical contract.
- Context should be generated on demand.
- Doctor checks should catch stale contract wording.
- Repo-local files should be the source of truth.
```

### 3.4 What to learn from Anthropic-style long-running agent harnesses

Long-running agents often fail by making partial progress and then claiming completion too early. They also lose continuity across sessions unless progress/state is explicit.

For `x-harness`, the correct application is not to become a long-running agent runtime. The correct application is to strengthen the admission gate:

```txt
- completion card required
- evidence required
- strict provenance when configured
- mutation guard for read-only verification
- failure taxonomy for withheld claims
- recovery route with owner and next action
- trace timeline for reconstructability
```

### 3.5 What to learn from papers on harness and verify-gated completion

The important paper-level lesson is that completion must be separated from execution.

Execution may propose completion. Admission decides whether that claim may surface as accepted.

For `x-harness`, this reinforces:

```txt
- verifier is read-only
- ambiguous cases fail closed
- non-success outcomes are withheld
- PGV and advisory checks remain advisory-only
- evidence and packet lineage matter
- event-level metrics must not be reported as task-level reliability
```

---

## 4. Updated product definition

### 4.1 Product category

`x-harness` should define a focused category:

```txt
Admission/readiness harness for AI coding workflows
```

This category is narrower than agent-harness ecosystems and broader than a simple validation script.

### 4.2 Primary users

```txt
- Developers using Claude Code, Cursor, OpenCode, Antigravity, Codex CLI, or similar tools
- Maintainers reviewing AI-generated changes
- Teams adding evidence gates to AI-assisted PR workflows
- Open-source repos wanting stricter AI contribution discipline
- DevTools/platform engineers who need a local, file-first admission gate
```

### 4.3 Main use cases

```txt
1. Local AI coding verification
   Agent modifies code and creates a completion card.
   x-harness verifies whether the claim is admissible.

2. PR readiness
   CI runs x-harness doctor, verify, report, and conformance.
   A PR can be reviewed only after an accepted admission decision or explicit withheld reason.

3. Multi-agent handoff control
   Multiple agents can work, but only x-harness determines whether the final claim is accepted.

4. Release readiness
   x-harness collects conformance results, checksums, SBOM/provenance references, smoke tests, and release evidence.

5. Adapter conformance
   Claude/Cursor/OpenCode/Antigravity adapters can be checked for stale or incorrect contract wording.
```

### 4.4 Product claim boundaries

Avoid:

```txt
guarantees correctness
ensures tasks are truly done
prevents all bugs
solves hallucination
proves production reliability
replaces CI
```

Use:

```txt
helps ensure completion claims are admitted under repository policy
makes completion claims auditable
fails closed when evidence is weak
keeps verification read-only
separates agent work from admission
emits readiness evidence for review and release
```

---

## 5. README positioning update

Replace the top of README with wording like this:

```md
# x-harness

x-harness is a Go-native, file-first admission and readiness harness for AI coding workflows.

It does not run your agents.
It does not replace CI.
It does not guarantee that code is correct.

It does one bounded job:

AI agent work
  -> completion claim
  -> evidence
  -> read-only verify gate
  -> accepted or withheld

Use x-harness when you want AI-generated changes to carry explicit evidence,
fail closed when evidence is weak, and produce auditable readiness records before review,
merge, or release.
```

Add a short "What x-harness is not" section:

```md
## What x-harness is not

x-harness is not an agent runtime, product planning system, issue tracker,
LLM gateway, dashboard platform, deployment engine, or plugin marketplace.
It integrates with AI coding agents; it does not replace them.
```

Add a short "Core contract" section:

```md
## Core contract

Completion is admitted, not claimed.

An agent may propose completion, but accepted completion requires:

```yaml
admission:
  outcome: success
  acceptance_status: accepted
```

All non-success outcomes are withheld.
The verifier is read-only.
PGV and LLM advisory checks are advisory-only.
```

Add a short "Roadmap" link:

```md
See [`X_HARNESS_ADMISSION_READINESS_UPDATED_ROADMAP.md`](X_HARNESS_ADMISSION_READINESS_UPDATED_ROADMAP.md) for the current implementation status and phased execution plan.
```

---

## 6. Governance and operational policies

### 6.1 Schema evolution policy

Schema changes must follow these rules to keep the repository stable across the Go rewrite and adapter ecosystem:

```txt
1. Version bump required
   Any change to schemas/completion-card.schema.json or policies/admission.yaml
   must bump the version field and update the golden examples.

2. Breaking vs non-breaking
   - Removing a required field is breaking.
   - Adding a new required field is breaking.
   - Adding an optional field is non-breaking.
   - Changing outcome mapping is breaking.

3. Golden example update
   Every schema or policy change must update at least one golden example
   to exercise the new behavior.

4. Adapter eval re-run
   After breaking changes, run adapter eval against all adapter files
   before merging.

5. Backport to TypeScript compatibility layer
   Non-breaking additions may be backported to the frozen TypeScript CLI
   so parity tests continue to pass.
   Breaking changes are Go-only; TypeScript parity tests for the old
   schema version may be retired.

6. Deprecation window
   A field or predicate may be marked deprecated for at least one minor
   release before removal.
```

### 6.2 Evidence-floor source of truth

`policies/admission.yaml` is the single source of truth for the evidence floor.

```txt
- policies/admission.yaml defines required, recommended, and runtime-enforced evidence.
- schemas/completion-card.schema.json validates card structure; it does not override the evidence floor.
- AGENTS.md and adapter managed blocks are advisory context, not policy.
- No other file may override or contradict policies/admission.yaml.
- If a doc and the policy disagree, the policy wins and the doc is considered stale.
```

### 6.3 Command namespace and budget

To avoid command proliferation and keep the CLI predictable:

```txt
1. Namespace budget
   - Target: ~30 top-level commands (including aliases).
   - Current count: ~28 (including aliases such as check -> verify).
   - New commands require justification in a PR description.

2. Prefer flags over subcommands
   - If a behavior is a variation of an existing command, add a flag.
   - Example: --worktree-aware on verify, not a new top-level command.

3. Mutation declaration
   - Every command must document whether it mutates files.
   - Mutation-capable commands must support --preview / --check where practical.

4. No self-admission
   - No command may produce an accepted completion decision on its own.
   - Only the verify pipeline may emit admission outcomes.

5. Grouping
   - Future commands should be grouped under existing namespaces
     (e.g., x-harness doctor --staleness, not x-harness staleness-doctor).
```

### 6.4 Staged pipeline decision

The architecture in section 7.2 describes a staged verify pipeline. This is a design target, not the current implementation.

**Decision: defer staged pipeline refactor to post-P1.**

```txt
- P0 and P1 features must work with the current monolithic pipeline.
- The current pipeline is implemented in internal/admission and internal/cli.
- Stage permission registry and strict stage-mutation checks are future work.
- Do not refactor the pipeline until P0b and P1 features are stable.
- After stabilization, a staged refactor may be considered if it improves testability
  or auditability without changing external CLI behavior.
```

### 6.5 Backward compatibility

```txt
1. Command aliases preserved
   - prepare  -> handoff readiness (legacy alias, kept indefinitely)
   - check    -> verify
   - status   -> report
   - recover  -> recovery suggest
   - reset    -> clean --tmp --force

2. Proposed readiness commands
   - readiness task, readiness pr, readiness release are new commands.
   - They do not replace existing aliases; they complement them.
   - If readiness pr is implemented, prepare remains an alias for the legacy behavior.

3. TypeScript CLI freeze
   - TypeScript CLI is frozen as a compatibility layer.
   - Go CLI is the only evolving interface.
   - npm consumers see only Go binaries.

4. Golden example stability
   - Golden examples must not break across minor versions.
   - Breaking changes require a major version bump or deprecation window.

5. Admission policy stability
   - policies/admission.yaml semantics are stable.
   - Additions only; removals require a v2 policy file.
```

### 6.6 Test development strategy

```txt
1. Release-blocking regression tests
   - Golden examples in examples/golden/ must pass in CI.
   - Adversarial benchmark must pass in CI.

2. Unit test coverage
   - Every internal package must have *_test.go with coverage for core logic.
   - CLI handlers should have at least smoke tests for happy and error paths.

3. Parity tests
   - While the TypeScript compatibility layer exists, run parity checks
     (npm run parity:check-go) in CI.
   - Parity tests ensure Go and TypeScript produce identical admission decisions
     on golden fixtures.

4. New feature requirement
   - New features must include tests before merge.
   - Tests must exercise at least one success and one failure path.

5. CI order
   npm ci -> typecheck -> build -> lint -> format:check -> test
   -> strict verify fixture -> doctor --root . -> examples verify
   -> adversarial benchmark

6. Fixture discipline
   - Add a new golden example for every new predicate or rejection path.
   - Name fixtures descriptively (e.g., blocked-missing-evidence,
     blocked-policy-drift, success-standard-strict-provenance).
```

### 6.7 Module boundary plan

```txt
cmd/x-harness/          CLI entrypoint only. No business logic.
internal/cli/           Command handlers, flag parsing, and output formatting.
internal/admission/     Admission decision engine. Reads policy and card; emits outcome.
internal/schema/        JSON Schema loading and validation.
internal/policy/        Policy file (admission.yaml) loading and query helpers.
internal/doctor/        Workspace health checks: file presence, schema compilation,
                        policy validity, managed block hash checks.
internal/mutationguard/ Mutation detection for --strict verify.
internal/trace/         Trace log append and chain verification.
internal/evidence/      Evidence validation, indexing, and redaction.
internal/loader/        Completion card loader (YAML/JSON).
internal/repo/          Repository metadata (git state, path resolution).
internal/authority/     Permission intent classification and authority checks.
internal/cost/          Cost budget evaluation.
internal/components/    Component registry inspection.
internal/permissions/   Permission rule evaluation.
internal/prediction/    Prediction/checklist claim evaluation.
internal/episode/       Episode package creation and chain verification.
internal/attribution/   Attribution metadata evaluation.
internal/approvalrisk/  Approval risk evaluation.
internal/frozen/        Frozen artifact import/export.
internal/assets/        Embedded runtime assets (schemas, policies, templates).

Rules:
- No cross-import cycles.
- internal/cli may import any internal package.
- internal/admission may import schema, policy, evidence, loader, mutationguard.
- internal/doctor may import any internal package but must remain read-only by default.
- pkg/ public API is not exposed yet; everything stays internal until v1.0.
```

---

## 7. Updated architecture model

### 7.1 Four surfaces

`x-harness` should be described through four surfaces:

```txt
1. Contract surface
   policies, schemas, templates, managed adapter blocks, AdmissionCard

2. Admission surface
   verify pipeline, predicate tiers, mutation guard, evidence provenance, recovery routing

3. Readiness surface
   task readiness, PR readiness, release readiness, reports, release evidence bundle

4. Conformance surface
   conformance suite, adapter eval, doctor, context sync, regression/adversarial tests
```

This is more product-useful than a generic multi-plane agent architecture.

### 7.2 Staged verify pipeline (design target)

Keep the Go verify pipeline stage-based for future refactor:

```txt
SourceLoad
  -> SchemaValidation
  -> AdmissionInputBuild
  -> EvidenceFloor
  -> ChecklistPrediction
  -> ContradictionDetection
  -> Governance
  -> MutationGuard
  -> AdmissionDecision
  -> RecoveryRouting
  -> TraceAppend
  -> ResultRender
```

Add a stage permission registry:

```yaml
verify_stage_permissions:
  SourceLoad: read_only
  SchemaValidation: read_only
  AdmissionInputBuild: read_only
  EvidenceFloor: read_only
  ChecklistPrediction: read_only
  ContradictionDetection: read_only
  Governance: read_only
  MutationGuard: read_only
  AdmissionDecision: read_only
  RecoveryRouting: read_only
  TraceAppend: write_allowlisted
  ResultRender: stdout_only
```

Acceptance criteria (post-P1):

```txt
[ ] Every verify stage has a declared permission class.
[ ] verify --strict fails if a stage mutates outside allowlist.
[ ] doctor checks stage registry consistency.
[ ] TraceAppend remains the only write-capable verify stage by default.
```

**Important:** this staged architecture is a design target. P0 and P1 features must work with the current monolithic pipeline. Do not refactor the pipeline until after P0b and P1 are stable.

---

## 8. New artifact: X-HarnessCard / AdmissionCard

### 8.1 Goal

Create a portable artifact that describes how `x-harness` is configured and what admission/readiness contract it enforces in the repository.

This makes the harness auditable as a harness, not just as a CLI command.

### 8.2 Proposed file

```txt
.x-harness/admission-card.yaml
```

### 8.3 Schema example

```yaml
x_harness_card:
  version: 1
  repo: "example/repo"
  generated_by: "x-harness"
  generated_at: "2026-05-28T00:00:00Z"

  category: "admission-readiness-harness"

  admission_model:
    flow:
      - agent_work
      - completion_claim
      - evidence
      - read_only_verify_gate
      - accepted_or_withheld
    verifier_mode: read_only
    fail_closed: true

  source_of_truth:
    policies:
      - policies/admission.yaml
      - policies/recovery.yaml
    schemas:
      - schemas/completion-card.schema.json
    context:
      - AGENTS.md
      - docs/RUNTIME_CONTRACT.md
    trace:
      - .x-harness/traces/events.jsonl

  accepted_if:
    admission.outcome: success
    admission.acceptance_status: accepted

  withheld_outcomes:
    - failed
    - blocked
    - skipped
    - timeout
    - error

  pgv_authority: advisory_only
  llm_judge_authority: advisory_only

  readiness_levels:
    - task
    - pr
    - release

  mutation_guard:
    strict_supported: true
    write_allowlist:
      - .x-harness/traces/**
      - .x-harness/reports/**

  denominator_contract:
    ambiguous_success_rate_forbidden: true
    task_level_requires_aligned_denominator: true
```

### 8.4 Commands

```bash
x-harness card generate
x-harness card verify
x-harness card print --format yaml
x-harness card print --format json
```

### 8.5 Acceptance criteria

```txt
[ ] card generate creates .x-harness/admission-card.yaml.
[ ] card verify checks policy/schema/context references exist.
[ ] card verify checks accepted/withheld semantics match current contract.
[ ] doctor includes AdmissionCard validity.
[ ] report includes AdmissionCard hash.
```

---

## 9. Readiness levels

### 9.1 Goal

Make `x-harness` useful across local tasks, PR workflows, and releases without becoming a deployment platform.

### 9.2 Levels

```txt
Task readiness:
  A single task claim is admissible under repository policy.

PR readiness:
  A change is ready for review/merge consideration because strict verification,
  mutation guard, CI evidence, and report generation passed.

Release readiness:
  A release candidate has conformance results, release evidence bundle,
  artifact checksums, SBOM/provenance references, and smoke test evidence.
```

### 9.3 Commands

```bash
x-harness readiness task --card completion-card.yaml
x-harness readiness pr --card completion-card.yaml --strict
x-harness readiness release --evidence release-evidence.json
```

These can also be aliases over existing commands:

```txt
readiness task    -> verify + report summary
readiness pr      -> verify --strict + doctor + report
readiness release -> conformance + release evidence verification
```

**Backward compatibility note:** `prepare` remains an alias for the legacy `handoff readiness` behavior. It is not replaced by `readiness task`.

### 9.4 Acceptance criteria

```txt
[ ] Readiness level is explicit in JSON report.
[ ] Task readiness does not claim PR or release readiness.
[ ] PR readiness requires strict verification.
[ ] Release readiness requires release evidence bundle.
[ ] Reports do not collapse readiness levels into one success rate.
```

---

## 10. Conformance suite

### 10.1 Goal

Turn `x-harness` from a CLI implementation into a contract-checkable harness standard.

The conformance suite verifies that a repo or implementation obeys the `x-harness` admission/readiness contract.

### 10.2 Commands

```bash
x-harness conformance run
x-harness conformance run --profile minimal
x-harness conformance run --profile strict
x-harness conformance report --format json
x-harness conformance report --format markdown
```

### 10.3 Profiles

```txt
minimal:
  schema loads
  policies load
  AGENTS managed block valid
  basic completion card verifies
  non-success maps to withheld

strict:
  mutation guard
  strict evidence provenance
  adapter eval
  failure taxonomy
  denominator contract
  adversarial fixtures

release:
  strict profile
  cross-platform smoke evidence
  release evidence bundle
  checksums/SBOM/provenance references
```

### 10.4 Checks

```txt
- admission mapping: success -> accepted
- admission mapping: non-success -> withheld
- PGV cannot admit
- LLM advisory cannot admit
- mutation guard detects verifier mutation
- strict provenance blocks missing command fields
- managed blocks have no drift
- adapter wording is contract-consistent
- forbidden tier aliases are not used as active tiers
- report JSON uses denominator contract
- trace schema is valid
- recovery route includes owner and next_action
```

### 10.5 Acceptance criteria

```txt
[ ] conformance run exits non-zero on contract violation.
[ ] conformance report is machine-readable.
[ ] minimal profile is lightweight.
[ ] strict profile includes adversarial cases.
[ ] release profile requires release evidence bundle.
```

---

## 11. Release Evidence Bundle

### 11.1 Goal

Make release readiness auditable without making `x-harness` a deployment system.

### 11.2 Commands

```bash
x-harness release evidence --out release-evidence.json
x-harness release verify-evidence release-evidence.json
x-harness release report --format markdown
```

### 11.3 Schema example

```json
{
  "schema_version": "x-harness.release-evidence.v1",
  "version": "x.y.z",
  "commit": "abc123",
  "go_version": "go1.22.x",
  "platforms": [
    "linux-amd64",
    "linux-arm64",
    "darwin-amd64",
    "darwin-arm64",
    "windows-amd64"
  ],
  "artifacts": [
    {
      "name": "x-harness-linux-amd64",
      "sha256": "...",
      "smoke_test": "passed"
    }
  ],
  "sbom": {
    "path": "dist/sbom.spdx.json",
    "sha256": "..."
  },
  "provenance": {
    "path": "dist/provenance.intoto.jsonl",
    "sha256": "..."
  },
  "conformance": {
    "minimal": "passed",
    "strict": "passed",
    "adversarial": "passed"
  },
  "doctor": {
    "status": "healthy"
  },
  "context_sync": {
    "status": "no_drift"
  },
  "npm_wrapper_smoke": {
    "status": "passed"
  }
}
```

### 11.4 Acceptance criteria

```txt
[ ] Release evidence can be generated without network services.
[ ] Release evidence includes artifact hashes.
[ ] Release evidence includes conformance status.
[ ] Release evidence includes doctor/context sync status.
[ ] verify-evidence fails on missing artifact, checksum mismatch, or missing conformance.
```

---

## 12. Denominator contract in reports

### 12.1 Goal

Prevent misleading metrics such as ambiguous "success rate."

### 12.2 Required JSON shape

```json
{
  "metrics": {
    "verify_event_success_rate": {
      "numerator": 1791,
      "denominator": 1800,
      "unit": "verify_event",
      "not_task_level": true
    },
    "task_completion_coverage": {
      "status": "not_computable",
      "reason": "missing_aligned_task_denominator"
    },
    "withheld_rate": {
      "numerator": 9,
      "denominator": 1800,
      "unit": "verify_event"
    }
  }
}
```

### 12.3 Rules

```txt
- Do not emit generic success_rate.
- Every rate must have numerator, denominator, and unit.
- Event-level metrics must say they are event-level.
- Task-level coverage must be not_computable unless an aligned task denominator exists.
- Report must include denominator_warning when applicable.
```

### 12.4 Acceptance criteria

```txt
[ ] report JSON rejects ambiguous success_rate.
[ ] report includes denominator_warning.
[ ] HTML/Markdown reports display metric units clearly.
[ ] conformance suite checks denominator contract.
```

---

## 13. Failure taxonomy v2

### 13.1 Goal

Make withheld decisions actionable and auditable.

### 13.2 Schema example

```yaml
withheld_reason:
  class: evidence_missing
  blocking_predicate: evidence_floor_met
  stage: EvidenceFloor
  recoverability: recoverable
  owner: implementation-worker
  next_action: "Attach command evidence with exit_code and artifact_hash."
```

### 13.3 Classes

```txt
evidence_missing
evidence_invalid
evidence_scope_missing
ownership_missing
schema_invalid
policy_drift
mutation_detected
approval_missing
stale_context
command_risky
contradiction
recovery_active
verifier_not_read_only
forbidden_tier_alias
adapter_contract_drift
release_evidence_missing
trace_invalid
```

### 13.4 Required fields

```txt
class
blocking_predicate
stage
recoverability
next_action
```

Optional fields:

```txt
owner
missing_field
artifact
command
policy_path
schema_path
trace_event_id
```

### 13.5 Acceptance criteria

```txt
[ ] Every withheld result has a typed reason.
[ ] Every typed reason has next_action.
[ ] Recovery routing uses taxonomy classes.
[ ] report groups withheld results by class/stage/predicate.
[ ] conformance checks at least one case per core class.
```

---

## 14. Permission intent classifier

### 14.1 Goal

Detect risky evidence commands without becoming a runtime sandbox.

### 14.2 Command intent categories

```txt
read_files
write_files
delete_files
shell_exec
network_outbound
package_install
package_publish
secret_access
git_mutation
database_mutation
deploy_or_publish
permission_change
unknown
```

### 14.3 Evidence example

```yaml
verification_artifacts:
  - kind: test
    command: "go test ./..."
    exit_code: 0
    intent:
      - shell_exec
      - read_files
    risk: low

  - kind: release
    command: "npm publish"
    exit_code: 0
    intent:
      - shell_exec
      - network_outbound
      - package_publish
    risk: high
    approval_required: true
```

### 14.4 Policy

```txt
light:
  warn if intent missing

standard:
  warn or block based on policy if high-risk command is present

deep/strict:
  block high-risk command unless approval receipt exists
```

### 14.5 Commands

```bash
x-harness evidence classify --command "npm publish"
x-harness evidence classify --card completion-card.yaml
```

### 14.6 Acceptance criteria

```txt
[x] Classifier is deterministic.
[x] Classifier has tests for common command patterns.
[x] Unknown commands are not silently treated as low risk.
[ ] Strict high-risk command without approval is withheld.
    - Deferred: admission blocking requires approval receipt schema (Section 15).
    - Current behavior: classifier is advisory-only; policy marks report_only=true.
[ ] Command intent is included in reports.
    - Deferred: report integration pending approval receipt and admission pipeline hook.
```

### 14.7 Implementation status

```txt
Implemented (minimal):
- internal/classify/classify.go: deterministic ClassifyCommand with 13 intents.
- policies/classifier.yaml: fail-closed policy documenting intent taxonomy and risk levels.
- schemas/classifier.schema.json + packages/cli/schemas/classifier.schema.json: classification result schema.
- CLI: x-harness evidence classify --command <cmd> [--json]
- CLI: x-harness evidence classify --card <path> [--json]
  - Inspects evidence.command_evidence[].command and verification_artifacts[].command.
- Tests cover all required categories: git read, go/npm test, go/npm build, npm install,
  npm publish, rm -rf, curl/wget, git push, sed -i, aws/gcloud/az, psql/mysql/sqlite,
  unknown custom commands.

Deferred:
- Admission enforcement / approval receipt integration (Section 15).
- Report intent inclusion.
- stdout/stderr secret scanning.
- ML/dynamic classification.
```

---

## 15. Approval receipt schema

### 15.1 Goal

Make human approval auditable for deep/high-risk tasks without creating a workflow engine.

### 15.2 Schema example

```yaml
approval_receipt:
  decision: approved
  approver: "human"
  approved_at: "2026-05-28T00:00:00Z"
  classified_commands:
    - command: "go test ./..."
      risk: low
    - command: "make migrate-dry-run"
      risk: medium
  aggregate_risk: medium
```

### 15.3 Admission rule

```txt
standard + high/unknown intent + missing approval receipt => withheld
deep + medium/high/unknown intent + missing approval receipt => withheld
```

### 15.4 Acceptance criteria

```txt
[x] Approval receipt schema exists in schemas/approval-receipt.schema.json.
[x] Runtime copy synced to packages/cli/schemas/approval-receipt.schema.json.
[x] Approval receipt is optional for light tasks (advisory only).
[x] Approval receipt is required for standard tasks with high/unknown commands.
[x] Approval receipt is required for deep tasks with medium/high/unknown commands.
[x] Scope mismatch between approval and commands produces withheld.
[x] Invalid decision, missing approver, or insufficient aggregate_risk produces withheld.
[ ] Approval receipt hash is included in report (deferred to report integration).
```

---

## 16. Adapter matrix, eval, and doctor

### 16.1 Goal

Keep adapters contract-consistent across Claude Code, Cursor, OpenCode, Antigravity, Codex-style environments, and future integrations.

### 16.2 Commands

```bash
x-harness adapters matrix
x-harness adapters doctor
x-harness adapters eval
x-harness adapters eval --adapter claude-code
x-harness adapters eval --format json
```

### 16.3 Adapter matrix schema

```yaml
adapters:
  claude-code:
    files:
      - adapters/claude-code/CLAUDE.md
    supports:
      managed_context: true
      hooks: optional
      skills: true
      mcp: external_only
    x_harness_contract:
      completion_card: required
      verify_read_only: required
      accepted_withheld_semantics: required
      pgv_advisory_only: required

  cursor:
    files:
      - adapters/cursor/rules/x-harness.mdc
    supports:
      managed_context: true
      hooks: limited
      skills: false
      mcp: external_only
```

### 16.4 Eval checks

```txt
- Adapter contains completion-card instruction.
- Adapter explains accepted/withheld semantics.
- Adapter states verifier is read-only.
- Adapter states PGV/LLM advisory checks cannot admit.
- Adapter does not use forbidden active tier aliases.
- Adapter contains recovery behavior for withheld results.
- Adapter managed block hash is current.
- Adapter does not instruct agent to self-admit completion.
```

### 16.5 Acceptance criteria

```txt
[x] adapters matrix prints capability table.
[x] adapters eval exits non-zero on missing README/caps/formats.
[x] adapters doctor checks managed block hash drift.
[ ] adapters doctor is included in conformance strict profile (strict profile is P2).
[ ] adapter files are generated from canonical contract blocks (generator is future work).
```

---

## 17. Admission skill-pack

### 17.1 Goal

Provide a small skill bundle for agent environments that support skills, without creating a skill ecosystem.

### 17.2 Layout

```txt
skills/x-harness-admission/
  SKILL.md
  examples/completion-card.yaml
  scripts/verify.sh
  README.md
```

### 17.3 SKILL.md content should cover only

```txt
- What x-harness is
- When to create/update completion-card.yaml
- How to record evidence
- How to run x-harness verify
- What accepted means
- What withheld means
- Why the agent must not self-admit completion
```

### 17.4 SKILL.md must not include

```txt
- coding strategy
- product planning workflow
- broad agent personas
- deployment commands
- hidden network installs
- MCP auto-enable
```

### 17.5 Acceptance criteria

```txt
[ ] Skill-pack is optional.
[ ] Skill-pack is namespaced under x-harness.
[ ] Skill-pack is generated from canonical contract text.
[ ] Skill-pack passes x-harness scan skill.
[ ] Skill-pack is included in adapter eval when supported.
```

---

## 18. Adapter/skill static security scanner

### 18.1 Goal

Detect obvious risky instructions or commands in adapter/skill bundles.

### 18.2 Commands

```bash
x-harness scan adapter
x-harness scan skill ./skills/x-harness-admission
x-harness scan managed
```

### 18.3 Heuristics

Flag patterns such as:

```txt
curl ... | bash
wget ... | sh
rm -rf /
rm -rf .
chmod +x remote file
cat ~/.ssh
cat ~/.env
env | curl
printenv | curl
browser profile paths
MCP server auto-enable without approval
hook command outside x-harness namespace
write target outside allowlist
```

### 18.4 Severity

```txt
low:
  unusual wording or broad instruction

medium:
  network command or filesystem write without declared intent

high:
  secret access, destructive command, remote code execution, publish/deploy command
```

### 18.5 Acceptance criteria

```txt
[ ] Scanner is deterministic.
[ ] Scanner supports JSON output.
[ ] Scanner does not require network access.
[ ] High severity scanner finding blocks conformance strict unless explicitly waived.
[ ] Waivers must include reason and expiry.
```

---

## 19. Install profiles and preview/apply plan

### 19.1 Goal

Improve onboarding without making install heavy.

### 19.2 Profiles

```txt
minimal:
  AGENTS.md managed block
  completion-card template
  basic verify contract

standard:
  minimal + mutation guard policy + trace/report config + CI snippet

deep:
  standard + approval receipt + rollback policy + packet chain + release evidence

adapter-only:
  adapter managed files only

ci:
  doctor + examples verify + conformance minimal

release:
  conformance strict + release evidence bundle template
```

### 19.3 Commands

```bash
x-harness init --profile minimal --preview
x-harness init --profile standard --preview
x-harness init --profile standard --apply
x-harness init --profile deep --apply
```

### 19.4 Plan output

```txt
CREATE AGENTS.md managed block
CREATE .x-harness/admission-card.yaml
UPDATE adapters/claude-code/CLAUDE.md managed block
CREATE policies/mutation-guard.yaml
NOOP schemas/completion-card.schema.json
```

### 19.5 Rules

```txt
- Default is preview unless current UX already expects direct write.
- Apply must never overwrite unmanaged user content silently.
- Managed blocks must include markers and hashes.
- --force is required for destructive replacement.
```

### 19.6 Acceptance criteria

```txt
[ ] init --preview prints exact planned mutations.
[ ] init --apply performs only planned mutations.
[ ] Managed blocks are idempotent.
[ ] Unmanaged user content is preserved.
[ ] doctor can verify installed profile.
```

---

## 20. Profile recommendation

### 20.1 Goal

Help users choose the right `x-harness` profile without adding a planning system.

### 20.2 Command

```bash
x-harness profile recommend --goal "AI PR verification"
x-harness profile recommend --goal "release readiness"
x-harness profile recommend --goal "deep security-sensitive change"
```

### 20.3 Output example

```yaml
recommended_profile: standard
required_commands:
  - x-harness verify --strict
  - x-harness report --format json
recommended_checks:
  - mutation_guard
  - evidence_provenance
  - denominator_contract
not_needed:
  - packet_chain
  - release_evidence_bundle
reason: "PR verification needs strict evidence and mutation guard, not release controls."
```

### 20.4 Acceptance criteria

```txt
[ ] Recommendation is deterministic.
[ ] Recommendation does not create files.
[ ] Recommendation explains what is not needed.
[ ] Deep profile is recommended only for high-risk or release-like goals.
```

---

## 21. Repair and uninstall flow

### 21.1 Goal

Make managed installation safe and reversible.

### 21.2 Commands

```bash
x-harness repair --preview
x-harness repair --apply
x-harness uninstall --preview
x-harness uninstall --apply
```

### 21.3 Rules

```txt
- Only remove files or blocks with x-harness managed markers.
- Never delete user-authored content without --force.
- If marker/hash mismatch exists, require explicit confirmation or --force.
- Emit restore plan before mutation.
```

### 21.4 Acceptance criteria

```txt
[ ] uninstall --preview lists managed files/blocks.
[ ] uninstall --apply removes only managed content.
[ ] repair --preview shows drift and proposed fixes.
[ ] repair --apply restores managed blocks from canonical contract.
[ ] No unmanaged content is deleted in tests.
```

---

## 22. Trace readability mode

### 22.1 Goal

Make JSONL trace useful without building a dashboard.

### 22.2 Commands

```bash
x-harness trace timeline --task <task_id>
x-harness trace explain --task <task_id>
x-harness trace inspect --withheld
x-harness trace collapse --by stage
```

### 22.3 Timeline output example

```txt
task-123
  SourceLoad                 ok
  SchemaValidation           ok
  EvidenceFloor              blocked
    reason: evidence_missing
    predicate: evidence_floor_met
    next: attach command + exit_code + artifact_hash
  AdmissionDecision          withheld
  RecoveryRouting            evidence_missing
```

### 22.4 Acceptance criteria

```txt
[ ] timeline reconstructs stage sequence from trace events.
[ ] explain shows blocking predicate and next_action.
[ ] inspect --withheld groups withheld cases by taxonomy class.
[ ] No dashboard/server is introduced.
```

---

## 23. Regression, capability, and adversarial suites

### 23.1 Goal

Separate release-blocking regressions from capability exploration.

### 23.2 Suite layout

```txt
examples/golden/regression/
  success-light/
  success-standard/
  strict-provenance/
  mutation-guard/
  policy-drift/

examples/golden/capability/
  complex-deep-card/
  multi-agent-handoff/
  packet-chain/
  cross-platform-paths/

examples/golden/adversarial/
  spoofed-approval/
  verifier-mutation/
  hidden-dangerous-command/
  pgv-attempts-admit/
  llm-advisory-attempts-admit/
```

### 23.3 Commands

```bash
x-harness examples verify --suite regression
x-harness examples verify --suite capability
x-harness examples verify --suite adversarial
```

### 23.4 Policy

```txt
regression:
  release-blocking

adversarial:
  release-blocking

capability:
  non-blocking by default; used to evaluate future improvements
```

### 23.5 Acceptance criteria

```txt
[ ] Regression suite must pass in CI.
[ ] Adversarial suite must pass in CI.
[ ] Capability suite can report partial capability without blocking stable release.
[ ] Conformance strict includes regression and adversarial suites.
```

---

## 24. Worktree-aware verification metadata

### 24.1 Goal

Avoid trace/evidence confusion when several agents work in separate git worktrees.

### 24.2 Add fields

```json
{
  "worktree": {
    "root": "/path/to/worktree",
    "git_common_dir": "/path/to/repo/.git/worktrees/task-123",
    "branch": "agent/task-123",
    "commit": "abc123",
    "dirty_baseline_hash": "sha256:..."
  }
}
```

### 24.3 Commands

```bash
x-harness verify --worktree-aware
x-harness doctor --worktree
```

### 24.4 Acceptance criteria

```txt
[ ] Trace includes worktree metadata when enabled.
[ ] Artifact paths are checked against worktree root.
[ ] Mutation guard baseline is bound to the worktree root.
[ ] report displays branch/commit/worktree for audit.
```

---

## 25. Context garbage collection and staleness doctor

### 25.1 Goal

Prevent contract docs and adapter files from becoming stale or duplicated.

### 25.2 Commands

```bash
x-harness doctor --staleness
x-harness context gc --check
x-harness context gc --write
```

### 25.3 Checks

```txt
- orphaned managed blocks
- duplicate contract wording
- stale adapter hash
- README overclaim phrases
- docs links that point to missing files
- policy referenced but missing
- schema field documented but absent
- adapter mentions forbidden tier aliases
```

### 25.4 Acceptance criteria

```txt
[ ] context gc --check is non-mutating.
[ ] context gc --write only touches managed blocks or generated docs.
[ ] README overclaim phrases are detectable.
[ ] doctor --staleness integrates into conformance strict.
```

---

## 26. Optional hooks bridge

### 26.1 Goal

Provide integration hooks for supported agent environments without making hooks mandatory.

### 26.2 Commands

```bash
x-harness hooks install --adapter claude-code --preview
x-harness hooks install --adapter claude-code --profile strict --apply
x-harness hooks uninstall --adapter claude-code --preview
```

### 26.3 Allowed hook behaviors

```txt
- remind agent to update completion-card
- warn when agent claims done without verify success
- warn on risky command intent
- append trace if explicitly allowlisted
```

### 26.4 Forbidden hook behaviors

```txt
- self-admit completion
- silently mutate source files
- auto-enable MCP servers
- run remote install commands
- deploy/publish/merge
```

### 26.5 Acceptance criteria

```txt
[ ] Hooks are disabled by default.
[ ] Hook installation supports preview/apply.
[ ] Hooks are namespaced.
[ ] Hooks can be uninstalled safely.
[ ] Adapter eval checks hook contract if installed.
```

---

## 27. Optional MCP read-only evidence adapter

### 27.1 Goal

Allow MCP to provide read-only evidence in future integrations, without making MCP part of core admission authority.

### 27.2 Rules

```txt
- Disabled by default.
- Read-only tools only.
- Allowlist required.
- No MCP tool can admit completion.
- No MCP server auto-enable.
- MCP evidence must be recorded with source and hash.
```

### 27.3 Commands

```bash
x-harness mcp doctor
x-harness mcp evidence --server <name> --resource <resource>
```

### 27.4 Acceptance criteria

```txt
[ ] MCP adapter is optional.
[ ] MCP evidence is advisory/evidence input only.
[ ] Admission authority remains deterministic x-harness policy.
[ ] doctor flags MCP tools with write permissions.
```

---

## 28. Optional sandbox bridge

### 28.1 Goal

Integrate with existing sandbox tools without building a sandbox runtime.

### 28.2 Commands

```bash
x-harness verify --sandbox-command "docker run ..."
x-harness doctor --check-sandbox
```

### 28.3 Rules

```txt
- Sandbox is external.
- x-harness does not own container lifecycle in core.
- Sandbox result must produce evidence artifact with hash.
- Sandbox failure is withheld unless policy says otherwise.
```

### 28.4 Acceptance criteria

```txt
[ ] sandbox-command is optional.
[ ] sandbox evidence is captured as artifact metadata.
[ ] x-harness does not require Docker or any sandbox dependency by default.
```

---

## 29. Updated roadmap and execution checklist

### P0a — Documentation and governance foundation (docs-only)

Goal: make the repository self-consistent, avoid overclaim, and establish operational policies before writing new code.

```txt
[ ] README positioning rewrite: "Go-native, file-first admission and readiness harness"
[ ] README adds "What x-harness is not" section
[ ] README adds "Core contract" section
[ ] README adds roadmap link
[ ] docs/README.md adds roadmap link
[ ] Roadmap adds clear forward-looking status disclaimer
[ ] Roadmap adds current implementation status table
[ ] Roadmap adds schema evolution policy
[ ] Roadmap adds evidence-floor source-of-truth statement
[ ] Roadmap adds command namespace and budget rules
[ ] Roadmap adds staged pipeline decision (defer refactor; P0b/P1 work with monolithic pipeline)
[ ] Roadmap adds backward compatibility section
[ ] Roadmap adds test development strategy
[ ] Roadmap adds module boundary plan
[ ] Ensure docs are internally consistent with Go-only npm runtime and TypeScript freeze
```

### P0b — Core admission enhancements (code; monolithic pipeline)

Goal: improve verify output safety and auditability without expanding scope or refactoring the pipeline.

Constraint: all P0b changes must work with the current monolithic pipeline in internal/admission and internal/cli.

```txt
[x] Add denominator contract to report JSON
    - Reject ambiguous success_rate
    - Include numerator, denominator, unit on every rate
    - Add denominator_warning when applicable
    - Event-level metrics declare not_task_level
    - Task-level coverage is not_computable unless aligned denominator exists
[~] Add failure taxonomy v2 to verify result (minimal)
    - WithheldReason struct exists in admission.Result with class, stage, recoverability, next_action
    - Recovery routing uses taxonomy classes (basic predicates only; not full 17-class taxonomy)
    - report groups withheld results by class/stage/predicate: metrics report only; trace grouping deferred
[~] Add trace/report rendering for withheld reasons (metrics report only)
    - JSON metrics report includes withheld_reason block with failure_class, failure_stage, recoverability, next_action, blocking_predicate
    - Accepted card report omits withheld_reason
    - Schema validates both accepted and withheld metrics reports
    - Text report rendering of withheld_reason deferred
    - Doctor checks that withheld cards have typed reasons deferred
    - Trace report grouping deferred: trace event format does not reliably carry taxonomy classes today
[x] Add/update golden examples for new denominator and taxonomy behavior
    - Added expected-report-metrics.json fixtures to success-standard-scoped-evidence and blocked-missing-evidence golden examples
    - Fixtures exercise denominator-safe metrics (verify_event_success_rate, task_completion_coverage.not_computable, withheld_rate, denominator_warning)
    - Fixtures exercise structured admission.withheld_reason taxonomy (failure_class, failure_stage, recoverability, next_action, blocking_predicate)
    - Go test TestReportMetricsGoldenFixtures validates actual report output against fixtures
[x] Update parity tests if TypeScript compatibility layer affected
    - No-op: parity harness does not compare report metrics outputs; no TypeScript compatibility changes required
```

### P1 — Contract and conformance (code; monolithic pipeline)

Goal: make x-harness contract-checkable and adapter-disciplined.

Constraint: P1 changes must still work with the current monolithic pipeline.

```txt
[x] Add AdmissionCard generator and verify
    - card generate creates .x-harness/admission-card.yaml
    - card verify checks policy/schema/context references exist
    - doctor includes AdmissionCard validity (via schemas_compile)
    - report includes AdmissionCard hash (future)
[x] Add conformance run --profile minimal
    - Schema loads
    - Policies load
    - AGENTS managed block valid
    - Basic completion card verifies
    - Non-success maps to withheld
    - Exits non-zero on contract violation
[x] Add adapter matrix schema and adapters matrix command
    - Prints capability table
    - Includes all existing adapters
[x] Add adapters eval (minimal)
    - Exits non-zero on contract drift
    - Checks: each adapter in matrix has README.md and non-empty capabilities/formats
    - JSON and text output supported
    - Full contract checks (completion-card instruction, accepted/withheld semantics, etc.) deferred to adapters doctor / static scanner
[ ] Add adapter doctor
    - Included in conformance strict profile (strict profile itself is P2)
    - Checks managed block drift
[ ] Add adapter doctor
    - Included in conformance strict profile (strict profile itself is P2)
    - Checks managed block drift
[x] Add release evidence schema draft
    - JSON schema for release-evidence.v1
    - Includes artifact hashes, conformance status, doctor/context sync status
[x] Add release evidence generator
    - Generates without network services
    - Includes Go version, artifacts, conformance/doctor/context_sync status
    - SBOM/provenance references and full platform matrix remain planned
[x] Add release verify-evidence
    - Fails on missing artifact, checksum mismatch, or missing conformance
```

### P2 — Operational hardening (code)

Goal: improve safety, install UX, and trace inspectability.

```txt
[x] Add permission intent classifier (minimal)
    - Deterministic classification for common command patterns
    - Unknown commands are not silently low risk
    - Strict high-risk without approval is withheld: deferred until approval receipt schema
    - Intent included in reports: deferred until report integration
[x] Add approval receipt schema (minimal)
    - Schema: schemas/approval-receipt.schema.json + runtime copy
    - Completion card accepts optional top-level approval_receipt
    - Admission hook classifies evidence commands and enforces tier-based receipt requirement
    - Standard: high/unknown without receipt => withheld
    - Deep: medium/high/unknown without receipt => withheld
    - Light: advisory only, no block
    - Validation: approved decision, non-empty approver, matching command coverage, sufficient aggregate_risk
    - Taxonomy: classifier_approval_required maps to command_risky / request_approval / human_intervention
    - Registry, hash binding, expiry, report integration remain planned
[ ] Add adapter/skill static scanner
    - Deterministic, JSON output, no network required
    - High severity blocks conformance strict unless waived
    - Waivers include reason and expiry
[ ] Add optional admission skill-pack
    - Optional, namespaced under x-harness
    - Generated from canonical contract text
    - Passes x-harness scan skill
[ ] Add install profile preview/apply
    - init --preview prints exact planned mutations
    - init --apply performs only planned mutations
    - Managed blocks are idempotent
    - Unmanaged content preserved
    - doctor can verify installed profile
[ ] Add profile recommend
    - Deterministic, does not create files
    - Explains what is not needed
    - Deep profile recommended only for high-risk/release-like goals
[ ] Add repair/uninstall preview/apply
    - uninstall --preview lists managed files/blocks
    - uninstall --apply removes only managed content
    - repair --preview shows drift and proposed fixes
    - repair --apply restores managed blocks from canonical contract
    - No unmanaged content deleted in tests
[ ] Add trace timeline / trace explain
    - timeline reconstructs stage sequence from trace events
    - explain shows blocking predicate and next_action
    - inspect --withheld groups withheld cases by taxonomy class
    - No dashboard/server introduced
[ ] Add worktree-aware verification metadata
    - Trace includes worktree metadata when enabled
    - Artifact paths checked against worktree root
    - Mutation guard baseline bound to worktree root
    - report displays branch/commit/worktree
[ ] Add context GC / staleness doctor
    - context gc --check is non-mutating
    - context gc --write only touches managed blocks or generated docs
    - README overclaim phrases are detectable
    - doctor --staleness integrates into conformance strict
[ ] Structure regression / capability / adversarial suites
    - Regression suite is release-blocking
    - Adversarial suite is release-blocking
    - Capability suite is non-blocking by default
    - Conformance strict includes regression and adversarial
```

### P3 — Conditional future integrations

Goal: keep the core narrow; only add integrations when real demand exists.

```txt
[ ] Add optional hooks bridge (only if adapter demand exists; otherwise defer)
    - Disabled by default
    - Supports preview/apply
    - Namespaced and safely uninstallable
    - Adapter eval checks hook contract if installed
[ ] Add optional MCP read-only evidence adapter (only if needed)
    - Optional, advisory/evidence input only
    - Admission authority remains deterministic x-harness policy
    - doctor flags MCP tools with write permissions
[ ] Add optional sandbox bridge (only if needed)
    - Optional sandbox-command flag
    - Sandbox evidence captured as artifact metadata
    - No Docker or sandbox dependency by default
[ ] Add rebuildable trace index (only if JSONL query becomes too slow)
[ ] Defer plugin API until real demand exists
[ ] Do not add marketplace by default
```

---

## 30. Implementation rules for AI coding agent

When implementing this roadmap, follow these rules:

```txt
1. Do not add an agent runtime.
2. Do not add a database as source of truth.
3. Do not add a dashboard server.
4. Do not add MCP as mandatory dependency.
5. Do not give LLM/PGV admission authority.
6. Do not make hooks mandatory.
7. Do not auto-commit or auto-ship.
8. Do not silently overwrite unmanaged user content.
9. Do not introduce generic product planning/backlog/story lifecycle features.
10. Every new command must clarify whether it mutates files.
11. Every mutation-capable command must support preview/check mode where practical.
12. Every report metric must include unit and denominator.
13. Every withheld decision must include typed reason and next action.
14. Every adapter must be generated or validated against the canonical contract.
```

---

## 31. Suggested implementation sequence

### Slice 1 — Positioning and report safety (P0a + P0b start)

```txt
1. Update README positioning (P0a).
2. Add denominator contract to report JSON (P0b).
3. Add failure taxonomy v2 to verify result (P0b).
4. Add trace/report rendering for withheld reasons (P0b).
```

Why first: these changes reduce overclaim risk and improve audit value without expanding scope.

### Slice 2 — AdmissionCard and conformance minimal (P1)

```txt
1. Add .x-harness/admission-card.yaml generator.
2. Add card verify.
3. Add conformance run --profile minimal.
4. Add doctor integration.
```

Why second: this makes x-harness contract-checkable.

### Slice 3 — Adapter matrix/eval (P1)

```txt
[x] Add adapter matrix schema.
[x] Add adapters matrix command.
[x] Add adapters eval (minimal).
[x] Add adapters doctor (minimal).
[x] Add managed block drift checks (minimal).
```

Why third: adapter drift becomes a real risk once x-harness supports many coding agents.

### Slice 4 — Release evidence bundle (P1)

```txt
[x] 1. Add release evidence schema.
[x] 2. Add release evidence generator.
[x] 3. Add release verify-evidence.
[x] 4. Add release report (minimal; markdown and JSON; SBOM/provenance/platform matrix remain planned).
```

Why fourth: Go-native binary releases need machine-readable evidence.

### Slice 5 — Security hardening (P2)

```txt
1. Add permission intent classifier.
2. Add approval receipt schema.
3. Add adapter/skill scanner.
4. Add optional admission skill-pack.
```

Why fifth: this adds safety without shifting x-harness into runtime ownership.

---

## 32. Definition of done

This update is complete when:

```txt
[ ] README clearly positions x-harness as admission/readiness harness.
[ ] No README/doc claim implies correctness guarantee or production reliability.
[ ] AdmissionCard can be generated and verified.
[ ] Reports include denominator-safe metrics.
[ ] Every withheld result has typed reason and next action.
[ ] Conformance minimal profile runs successfully.
[ ] Conformance strict profile covers mutation guard, provenance, adapter eval, and adversarial cases.
[ ] Adapter matrix exists and adapter eval detects drift.
[ ] Release evidence bundle can be generated and verified.
[ ] Trace timeline/explain makes withheld decisions inspectable.
[ ] Install profiles support preview/apply.
[ ] Repair/uninstall only touches managed content.
[ ] Optional hooks/MCP/sandbox integrations remain disabled by default.
[ ] No database, daemon, API server, dashboard, or agent runtime is introduced.
```

---

## 33. Final strategic summary

`x-harness` should not compete with broad harness ecosystems by adding more agents, skills, hooks, personas, and workflows.

It should win by being the clearest admission/readiness boundary in the AI coding stack:

```txt
completion claims become auditable,
evidence becomes explicit,
verification remains read-only,
non-success fails closed,
withheld reasons are actionable,
adapters remain contract-consistent,
release readiness is backed by evidence,
and reports do not overclaim.
```

The correct direction is:

```txt
from CLI verification tool
  -> to conformance-grade admission/readiness harness
```

Keep the core narrow. Make the boundary sharper.

(End of file)
