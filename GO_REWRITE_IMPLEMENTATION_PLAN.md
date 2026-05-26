# x-harness Go Rewrite Implementation Plan

Status: planning baseline
Date: 2026-05-25
Scope: rewrite the canonical x-harness CLI/runtime from TypeScript to Go while preserving the current file-first verification contract.

Toolchain prerequisite: Phase 1 and later require a local Go toolchain capable
of running `go build ./cmd/x-harness` and `go test ./...`. The initial module
target is Go 1.22 and uses only the standard library until a later phase
requires additional dependencies.

## 1. Executive Decision

The Go rewrite should be implemented as a parity-first migration, not as a broad product expansion.

x-harness currently defines itself as a lightweight, offline-first, file-first, verify-gated CLI. The rewrite must preserve that shape:

- No daemon is required.
- No database is required.
- No API server is required.
- No LLM provider is required.
- No agent execution runtime is required.
- Repository files remain the source of truth.
- The verify gate remains read-only and fail-closed.

The rewrite target is a native Go CLI that improves distribution, startup, CI ergonomics, and filesystem-heavy paths while keeping the runtime contract stable.

## 2. Source Material

This plan consolidates:

- `x-harness-improvements.md`
- `X_HARNESS.md`
- `AGENTS.md`
- `docs/ARCHITECTURE.md`
- `docs/RUNTIME_CONTRACT.md`
- `docs/VERIFY_GATE.md`
- `docs/CI.md`
- current TypeScript source under `packages/cli/src`
- current test/golden fixtures under `packages/cli/tests`, `examples`, and `tests`

External implementation facts used for language/runtime choice:

- Go supports target selection with `GOOS` and `GOARCH` in the official toolchain.
  Reference: https://go.dev/cmd/compile/
- Rust has strong platform support, but target guarantees differ by tier.
  Reference: https://doc.rust-lang.org/rustc/platform-support.html
- Node supports single executable applications, but the workflow has packaging caveats and is still a Node-binary injection model.
  Reference: https://nodejs.org/api/single-executable-applications.html

## 3. Current Repository Baseline

Current implementation profile:

- Language: TypeScript, Node.js >= 20.
- CLI package: `packages/cli`.
- Runtime dependencies: `ajv`, `commander`, `fs-extra`, `yaml`.
- Source shape: command modules plus core modules.
- Public contract assets:
  - `schemas/*.json`
  - `policies/*.yaml`
  - `templates/*.md`
  - `adapters/*`
  - `examples/*`
  - `AGENTS.md`
  - `X_HARNESS.md`
  - `docs/*`

Current critical runtime semantics:

- Completion is admitted, not claimed.
- Agents may propose completion but may not self-admit completion.
- Accepted completion requires:

```yaml
admission:
  outcome: success
acceptance_status: accepted
```

- All non-success outcomes are withheld.
- Canonical tiers are only `light`, `standard`, and `deep`.
- `claim.fix_status` is canonical for completion cards.
- `result.fix_status` is compatibility-only for subagent returns.
- The verifier is read-only.
- `verify --strict` enables mutation guard and stricter evidence provenance.

## 4. Rewrite Goals

### 4.1 Primary Goals

- Produce a native `x-harness` binary.
- Preserve current verification semantics.
- Preserve current JSON/YAML contract behavior.
- Preserve golden fixture outcomes.
- Improve CLI startup and distribution.
- Improve mutation guard hashing and directory snapshot performance.
- Make verify pipeline stages explicit and testable.
- Make managed contract generation the single source of truth for docs/templates/adapters.
- Keep CI and release gates deterministic.

### 4.2 Non-Goals for the First Go Release

These items from `x-harness-improvements.md` are not part of the first Go rewrite:

- Full agent execution engine.
- ReAct loop.
- Multi-provider LLM integration.
- LLM streaming.
- Plugin registry.
- Third-party plugin sandbox.
- REST API server.
- WebSocket server.
- Web dashboard.
- Redis/PostgreSQL task queue.
- Docker sandbox runtime.
- Vector store.
- Multi-tenant persistence.

These may become future products, but adding them during the rewrite would combine a language migration with a major scope expansion and would create high contract drift risk.

## 5. Recommended Improvements to Apply

The following proposals from `x-harness-improvements.md` should be applied during the Go rewrite.

| Proposal                            | Decision       | Go rewrite interpretation                                                   |
| ----------------------------------- | -------------- | --------------------------------------------------------------------------- |
| Middleware/interceptor pipeline     | Apply          | Implement verify as explicit ordered stages.                                |
| Logger abstraction                  | Apply lightly  | Use small structured logger interface; avoid heavy framework.               |
| Hierarchical configuration          | Apply          | Resolve config from defaults, env, repo policy, CLI flags.                  |
| Repository-as-operating-system      | Apply          | Preserve `AGENTS.md` managed context and canonical reading order.           |
| Auto-discovery command registration | Defer          | Keep explicit command registration until parity is complete.                |
| Event bus                           | Defer          | Use direct stage results first; add events only if trace/report needs them. |
| DI container                        | Reject for v1  | Use Go interfaces and constructors.                                         |
| SQLite backend                      | Defer          | Keep file-first JSONL trace until query needs are proven.                   |
| Performance benchmarks              | Apply          | Add Go benchmark suite and parity latency reports.                          |
| Fuzzing/property tests              | Apply          | Fuzz YAML/JSON parsing and admission decision invariants.                   |
| Concurrency tests                   | Apply          | Cover mutation guard and parallel read-only verification.                   |
| Execution-based evaluation          | Apply narrowly | Validate evidence commands/artifacts; no Docker runtime in v1.              |
| Plugin system                       | Defer          | Keep adapters/templates as extension surface for v1.                        |
| LLM provider integration            | Defer          | Out of scope for verify-gated core.                                         |
| API/Web dashboard                   | Defer          | Out of scope for native CLI parity.                                         |

## 6. Architecture Target

### 6.1 Package Layout

Proposed Go layout:

```txt
cmd/x-harness/
  main.go

internal/cli/
  root.go
  commands.go
  output.go
  exit.go

internal/assets/
  locate.go
  manifest.go
  sync.go

internal/schema/
  loader.go
  validator.go
  errors.go

internal/policy/
  admission.go
  recovery.go
  permissions.go
  mutation_guard.go

internal/contract/
  model.go
  load.go
  render.go
  managed_blocks.go
  drift.go

internal/verify/
  pipeline.go
  stage_source_load.go
  stage_schema.go
  stage_admission_input.go
  stage_evidence.go
  stage_prediction.go
  stage_contradiction.go
  stage_governance.go
  stage_mutation_guard.go
  stage_admission.go
  stage_recovery.go
  stage_trace.go
  result.go

internal/admission/
  types.go
  decision.go
  evidence_floor.go
  prediction_checklist.go
  contradictions.go
  provenance.go

internal/recovery/
  routes.go
  suggest.go

internal/governance/
  changed_files.go
  tier_downgrade.go
  approval.go

internal/mutationguard/
  git.go
  fallback.go
  ignore_policy.go
  hash.go
  compare.go
  benchmark.go

internal/doctor/
  doctor.go
  assets.go
  managed_blocks.go
  policy_drift.go
  render.go

internal/benchmark/
  latency.go
  integration.go
  mutation_guard.go
  report.go

internal/examples/
  verify.go

internal/trace/
  event.go
  jsonl.go
  chain.go

internal/report/
  metrics.go
  render.go

internal/testutil/
  fixtures.go
  golden.go
```

### 6.2 Command Strategy

The Go CLI should keep the existing command names and aliases.

Primary parity commands:

```txt
verify
check
doctor
examples verify
context --contract
benchmark
handoff
prepare
report
status
trace
clean
reset
```

Secondary parity commands:

```txt
init
add
recovery
recover
packet
intake
governance
intervention
prediction
components
evidence
episode
attribution
permissions
evolve
export
import
frozen
federation
approval-risk
agent-profile
cost
actions
```

Do not remove TypeScript commands until the Go command has parity tests.

### 6.3 Verify Pipeline

The Go verify pipeline should be a small, ordered stage chain:

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

Each stage must:

- Receive an immutable input context where practical.
- Return a typed stage result.
- Append errors/notes/checks through controlled result methods.
- Avoid direct process exit.
- Avoid writing files unless the stage is explicitly a non-verify write stage.

The mutation guard must wrap the read-only pipeline:

```txt
before snapshot
  -> run read-only verify stages
after snapshot
  -> compare snapshots
  -> block if unexpected delta exists
```

## 7. Contract Parity Requirements

### 7.1 Admission Decision

The Go implementation must preserve:

- `success -> accepted`
- `failed -> withheld`
- `blocked -> withheld`
- `skipped -> withheld`
- `timeout -> withheld`
- `error -> withheld`

Admission success requires:

- `claim.fix_status == fixed`
- `verification.status == passed`
- `admission.outcome == success`
- `acceptance_status == accepted`
- non-empty evidence
- owner present
- accountable present
- evidence floor met
- admission mapping valid
- no unresolved blockers
- no active recovery
- verifier read-only
- `done_checklist` for standard/deep
- `prediction` for standard/deep

### 7.2 Tier Rules

Allowed active tiers:

```txt
light
standard
deep
```

Forbidden active aliases:

```txt
small
medium
large
```

### 7.3 Evidence Floor

Light:

```txt
files_changed + (command_evidence or manual_rationale)
```

Standard:

```txt
files_changed + command_evidence + done_checklist + prediction
```

Deep:

```txt
files_changed
command_evidence
evidence_scope_declared
untested_regions_declared
remaining_risks_declared
execution_controls_present
rollback_policy_present
done_checklist
prediction
verification_artifacts
state.read_set
state.write_set
```

### 7.4 Strict Evidence Provenance

For standard/deep cards, `verify --strict` must require command evidence and verification artifacts to include:

```txt
command
exit_code
runner
started_at
```

### 7.5 Checklist Honesty

The Go implementation must preserve checklist honesty checks:

- `done_checklist` must align with card state.
- `done_checklist` must align with evidence.
- `done_checklist` must align with verification artifacts.
- mismatch must produce a withheld result and a recovery route.

## 8. Mutation Guard Requirements

### 8.1 Git Workspace Mode

Use:

```bash
git status --porcelain=v1 -z --untracked-files=all
```

Then hash dirty/untracked file content with bounded concurrency.

Expected behavior:

- Does not require clean worktree.
- Detects unexpected deltas introduced during verification.
- Allows `.x-harness/` trace/cache writes only when allowlisted.
- Fails closed if baseline cannot be established in strict mode.

### 8.2 Non-Git Fallback Mode

Fallback snapshot must:

- Walk the directory tree.
- Hash regular files.
- Preserve symlink identity.
- Skip ignored paths.
- Use bounded concurrency.

Ignore sources:

- hard defaults:
  - `.git/`
  - `node_modules/`
  - `.x-harness/`
- root `.gitignore`
- `policies/mutation-guard.yaml` under `fallback_ignore`

Environment override:

```txt
X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY
```

Rules:

- default: `16`
- minimum effective value: `1`
- invalid or less than `1`: use default
- maximum cap: `64`

### 8.3 Benchmark Matrix

The Go benchmark must cover:

```txt
file counts: 100, 1000, 5000
concurrency: 1, 4, 16, 64
modes: git, non-git
```

Report fields:

```txt
mode
file_count
concurrency
duration_ms
hashed_paths
ok
```

## 9. Doctor Requirements

Go `doctor` must support:

```bash
x-harness doctor --root . --format json
x-harness doctor --root . --format text
x-harness doctor --root . --json
```

Default format remains JSON for automation compatibility.

Doctor checks:

- critical assets exist
- all schemas parse and compile
- all policies parse
- managed context block is present and valid
- managed runtime contract blocks are valid
- no forbidden active tier aliases in critical runtime docs/templates
- adapter contract wording is consistent
- `policies/mutation-guard.yaml` exists
- examples/golden can be discovered
- package/release asset manifest is coherent
- CI has strict verify path
- CI has doctor/examples/adversarial benchmark gates

## 10. Canonical Contract Generator

The Go rewrite must keep contract rendering as a first-class package.

Inputs:

```txt
policies/admission.yaml
policies/recovery.yaml
policies/intake.yaml
schemas/completion-card.schema.json
internal/contract/model.go
```

Managed targets:

```txt
AGENTS.md
docs/RUNTIME_CONTRACT.md
templates/SUBAGENT_TASK_light.md
templates/SUBAGENT_TASK_standard.md
templates/SUBAGENT_TASK_deep.md
templates/COMPLETION_CARD.md
adapters/generic/AGENTS.md
adapters/claude-code/CLAUDE.md
adapters/cursor/rules/x-harness.mdc
adapters/opencode/verify-agent.md
adapters/antigravity/*
```

Commands:

```bash
x-harness context --contract
x-harness context sync --check
x-harness context sync --write
```

Rules:

- Managed blocks must include a generator marker.
- Managed blocks must include a content hash.
- `sync --check` exits non-zero on drift.
- `sync --write` is not part of verify and may mutate files.
- Verify must never repair managed blocks.

## 11. Test Strategy

### 11.1 Parity Tests

For every parity command, compare TypeScript baseline and Go output semantically.

Minimum parity matrix:

| Area      | Required tests                                                   |
| --------- | ---------------------------------------------------------------- |
| verify    | all golden cards, adversarial cards, strict mode, mutation guard |
| doctor    | JSON default, text format, invalid format                        |
| examples  | expected pass/withheld outcomes                                  |
| context   | managed block hash stability                                     |
| benchmark | latency report shape, mutation guard matrix                      |
| report    | JSON/markdown/html shape where supported                         |
| trace     | append and verify-chain                                          |
| handoff   | light/standard/deep template generation                          |

### 11.2 Golden Fixtures

Golden tests must include:

- success light
- success standard scoped evidence
- blocked missing evidence
- failed invalid status
- withheld partial fix
- deep approval required
- multi-agent success
- adversarial spoofed approval
- adversarial verifier mutation
- adversarial hidden dangerous command
- adversarial PGV attempts to admit

### 11.3 Fuzz Tests

Fuzz targets:

- YAML/JSON completion cards
- admission input conversion
- evidence floor detection
- status contradiction detection
- managed block parser
- ignore pattern matching
- trace event parser

Invariants:

- verifier must not panic on malformed input
- malformed input must not become accepted
- non-success outcome must map to withheld
- forbidden tier aliases must not become runtime tiers
- PGV must not grant admission authority

### 11.4 Concurrency Tests

Concurrency tests:

- parallel non-Git snapshot hashing
- parallel Git dirty path hashing
- simultaneous read-only verify calls
- trace append behavior if enabled
- benchmark cleanup of temporary fixtures

## 12. CI Migration Plan

### 12.1 Dual-Run CI

During migration, CI must run both implementations:

```txt
npm ci
npm run build
npm run test
go test ./...
go build ./cmd/x-harness
TS verify golden
Go verify golden
TS doctor
Go doctor
TS examples verify
Go examples verify
parity diff report
```

### 12.2 Go CI Jobs

Required Go jobs:

```txt
go test ./...
go test -race ./...
go test -fuzz smoke targets in short mode
go vet ./...
go build ./cmd/x-harness
go run ./cmd/x-harness doctor --root . --format json
go run ./cmd/x-harness examples verify
go run ./cmd/x-harness benchmark --filter mutation-guard --json
```

### 12.3 Gate to Make Go Primary

Go becomes primary only when:

- all required Go commands exist
- all golden/adversarial parity tests pass
- CI dual-run passes for at least one full release candidate
- `doctor` reports healthy
- mutation guard strict path passes in CI
- release artifact smoke tests pass on Linux, macOS, and Windows
- docs/templates/adapters managed block drift is zero

## 13. Release Strategy

### 13.1 Native Artifacts

Release artifacts:

```txt
x-harness-linux-amd64
x-harness-linux-arm64
x-harness-darwin-amd64
x-harness-darwin-arm64
x-harness-windows-amd64.exe
```

Each release must include:

- checksums
- SBOM
- provenance if available
- packed CLI smoke report
- frozen compatibility report
- changelog entry

### 13.2 npm Wrapper

An npm wrapper package may remain useful for users who expect Node/npm install flows.

Wrapper behavior:

- install or select the platform binary
- expose `x-harness` and `xh`
- avoid running TypeScript runtime code in the steady state
- preserve package metadata and repository links

### 13.3 TypeScript Deprecation Path

Do not delete the TypeScript CLI immediately.

Recommended sequence:

1. Go and TypeScript run side by side.
2. Go becomes default binary in CI.
3. TypeScript CLI moves to maintenance mode.
4. TypeScript source moves to `legacy/typescript/` or remains archived on a branch after a stable Go release.
5. Remove TypeScript only after compatibility window is complete.

## 14. Implementation Phases

### Phase 0: Contract Freeze

Deliverables:

- capture TypeScript baseline outputs
- freeze golden fixtures
- document current JSON output shapes
- record current exit codes
- record current benchmark baseline

Acceptance gate:

- baseline artifacts are committed or stored under deterministic test fixtures
- no rewrite work begins before baseline exists

### Phase 1: Go CLI Skeleton

Deliverables:

- `go.mod`
- `cmd/x-harness/main.go`
- root command
- global `--help`
- version output
- exit handling
- JSON/text output helpers

Acceptance gate:

- `go build ./cmd/x-harness` succeeds
- `go test ./...` succeeds
- help output lists primary commands

### Phase 2: Schema, Policy, and Contract Loading

Deliverables:

- JSON/YAML loader
- JSON Schema validator
- policy loader
- contract model
- managed block parser

Acceptance gate:

- all current schemas parse and compile
- all current policies parse
- contract command renders current canonical rules

### Phase 3: Verify Pipeline Parity

Deliverables:

- source loader
- schema validation stage
- admission input builder
- evidence floor checks
- prediction/checklist checks
- contradiction checks
- admission decision
- recovery routing
- JSON/text verify output

Acceptance gate:

- all golden verify cases match expected semantic outcomes
- all adversarial cases remain withheld/blocked as expected
- no non-success outcome maps to accepted

### Phase 4: Strict Verify and Mutation Guard

Deliverables:

- Git mutation snapshot
- non-Git fallback snapshot
- ignore policy support
- bounded hashing concurrency
- strict provenance checks
- mutation guard benchmark

Acceptance gate:

- strict verify blocks unexpected mutation
- strict verify works in non-Git fixtures
- benchmark matrix reports git and non-Git modes

### Phase 5: Doctor and Managed Contract Sync

Deliverables:

- doctor JSON/text
- critical asset checks
- managed block drift checks
- forbidden alias checks
- `context sync --check`
- `context sync --write`

Acceptance gate:

- `doctor --format json` healthy on repo
- `doctor --format text` readable
- `context sync --check` passes with no drift

### Phase 6: Examples, Trace, Report, Handoff

Deliverables:

- `examples verify`
- trace append and verify-chain
- report metrics parity
- handoff generation
- beginner aliases

Acceptance gate:

- examples pass with expected outcomes
- trace chain verifies
- handoff templates match canonical tiers

### Phase 7: Remaining Command Parity

Deliverables:

- init/add/recovery
- packet/episode/evidence/attribution
- components/permissions/prediction
- frozen/evolve/federation/cost/approval-risk/agent-profile

Acceptance gate:

- command-specific parity tests pass
- no command introduces contract drift

### Phase 8: Dual-Run CI and Release Candidate

Deliverables:

- dual-run GitHub Actions
- Go release build workflow
- release smoke fixtures
- platform matrix
- npm wrapper plan

Acceptance gate:

- full dual-run CI passes
- release artifacts pass smoke tests
- Go CLI is ready to become primary

## 15. Work Breakdown Checklist

### Foundation

- [x] Create Go module.
- [x] Add root command and version.
- [x] Add command registration.
- [x] Add output rendering helpers.
- [x] Add typed CLI errors and exit codes.
- [x] Add repo root discovery.
- [x] Add asset locator.

### Contract

- [x] Port canonical contract model.
- [ ] Port managed block rendering.
- [ ] Port managed block validation.
- [x] Port runtime contract rendering.
- [ ] Add contract drift tests.

### Schema and Policy

- [x] Add YAML/JSON reader.
- [x] Add JSON Schema compiler.
- [x] Add completion card validation.
- [ ] Add subagent return validation.
- [ ] Add policy loaders.
- [ ] Add schema parity fixtures.

### Verify

- [x] Port source loading (minimal).
- [x] Port admission input builder (minimal).
- [x] Port evidence floor (light/standard/deep).
- [ ] Port strict provenance.
- [x] Port done checklist and prediction checks.
- [x] Port contradiction checks.
- [x] Port governance checks (deep approval, tier downgrade).
- [x] Port admission decision.
- [ ] Port recovery routing.
- [ ] Port trace event creation.
- [x] Port verify output renderers (minimal text + JSON).

### Mutation Guard

- [x] Port Git status snapshot.
- [x] Port dirty/untracked content hashing.
- [x] Port non-Git fallback snapshot.
- [x] Port ignore policy.
- [x] Port concurrency limit.
- [x] Port allowlist.
- [ ] Add mutation injection test hook.
- [ ] Add mutation guard benchmark.

### Doctor

- [x] Port critical assets list.
- [x] Port schema checks.
- [x] Port policy checks.
- [x] Port managed block checks.
- [ ] Port tier alias checks.
- [ ] Port component registry check.
- [x] Add `--format json|text`.
- [x] Preserve `--json`.

### Benchmarks

- [ ] Port latency benchmark.
- [ ] Port admission/adversarial benchmark.
- [ ] Port mutation guard benchmark.
- [ ] Add JSON report schema.
- [ ] Add markdown report renderer.

### CI and Release

- [ ] Add Go CI job.
- [x] Add dual-run parity harness (partial: Go vs TS baseline script).
- [ ] Add race test job.
- [ ] Add release build matrix.
- [ ] Add checksum generation.
- [ ] Add packed binary smoke tests.
- [ ] Add frozen compatibility smoke tests.

## 16. Risk Register

| Risk                                     | Impact   | Mitigation                                              |
| ---------------------------------------- | -------- | ------------------------------------------------------- |
| Contract drift                           | Critical | Golden parity tests and managed contract generator.     |
| JSON Schema behavior mismatch            | High     | Schema fixture suite and semantic validation snapshots. |
| YAML parsing differences                 | High     | Round-trip fixtures for policies/cards.                 |
| Output shape drift                       | High     | JSON snapshot tests for automation-facing commands.     |
| Exit code drift                          | High     | Explicit exit-code parity tests.                        |
| Rewrite scope creep                      | Critical | Keep LLM/API/DB/plugin runtime out of v1.               |
| Mutation guard false positives           | High     | Dedicated Git and non-Git fixtures.                     |
| Windows path issues                      | High     | Platform CI and path normalization tests.               |
| Release artifact mismatch                | Medium   | Cross-platform smoke tests and checksums.               |
| TypeScript/Go divergence during dual-run | High     | Freeze parity baseline and block drift in CI.           |

## 17. Final Go Rewrite Done Criteria

The rewrite is complete only when:

- Go CLI implements all primary commands.
- Go CLI implements all required secondary commands or has documented compatibility stubs.
- All golden fixtures pass.
- All adversarial fixtures remain withheld/blocked as expected.
- `verify --strict` passes in Git and non-Git workspaces.
- mutation guard benchmark exists and reports the required matrix.
- `doctor --format json` is healthy.
- `context sync --check` reports no drift.
- CI dual-run passes.
- release artifacts pass smoke tests on supported platforms.
- TypeScript implementation can be frozen without losing public behavior.

## 18. Recommended First Implementation Slice

Start with this slice:

1. Go module and CLI skeleton. *(done)*
2. Asset locator. *(done)*
3. YAML/JSON loader. *(done)*
4. Contract model and `context --contract`. *(done)*
5. Completion card schema validation. *(done)*
6. Minimal `verify --card`. *(done)*
7. Admission decision parity for golden fixtures. *(done)*
8. `doctor --format json`. *(done)*
9. Mutation guard Git snapshot. *(done)*
10. Dual-run parity test harness. *(done)*

This slice proves the rewrite can preserve the core contract before investing in the full command surface.
