# x-harness Codebase Map

This is a single-file orientation map for how the repository works. It describes
current behavior only; it is not a roadmap.

---

## 1. Entry points

- **Native Go CLI** — `cmd/x-harness/main.go` calls `internal/cli.Run`, which
dispatches to handlers in `internal/cli/`.
- **TypeScript compatibility CLI** — `packages/cli/src/index.ts` uses Commander
and routes to commands under `packages/cli/src/commands/`.
- **Built-in CI recipe** — `xh ci` and `xh run builtin:ci` execute a local-only
workflow (Go implementation in `internal/cli/run.go`).

---

## 2. Core pipeline

A typical verification run flows like this:

```text
Agent writes code
Agent writes completion-card.yaml
        │
        ▼
xh verify --card completion-card.yaml
        │
        ├─► Load JSON schemas from schemas/
        ├─► Validate card structure (required fields, types)
        ├─► Load policies/admission.yaml
        ├─► Evaluate evidence floor against claim + verification
        ├─► Run optional enforce stages if requested
        │       --boundary-enforce, --context-enforce,
        │       --decision-enforce, --intent-enforce,
        │       --contract-oracles, --context-floor, --strict
        │
        ▼
outcome: success / failed / blocked / skipped / timeout / error
acceptance_status: accepted or withheld
recovery routing → handoff.next_action + handoff.owner
```

`xh check` is an alias for `xh verify`. The verifier is strictly read-only and
never mutates source while checking.

---

## 3. Key modules

| Module | Responsibility |
| :----- | :------------- |
| `internal/cli` | Command dispatch and all subcommand handlers (`verify`, `doctor`, `examples`, `boundary`, `release`, `conformance`, `evidence`, `policy`, `explain`, etc.). |
| `internal/admission` | Evidence-floor evaluation and admission policy mapping. |
| `internal/schema` | JSON Schema validation; includes a bounded Go fuzz target (`FuzzValidate`). |
| `internal/boundary` | `policies/boundaries.yaml` lint, `check --all/--changed`, and `explain`. V1 uses path-glob + import regex only. |
| `internal/contextmanifest` | Context-floor and context-enforce checks. |
| `internal/contract` | Contract-oracle rule evaluation (`grep_rules`, `dependency_rules`). |
| `internal/doctor` | Workspace health checks: schemas, policies, templates, adapter links, staleness, overclaim, docs drift. Optional deterministic repair with `--fix --confirm`. |
| `internal/worktree` | Git worktree helpers (changed files, root discovery). |
| `internal/loader` | Load completion cards, schemas, and policies. |
| `internal/mutationguard` | Detect unexpected source mutation during verify. |
| `internal/approvalrisk` | Advisory approval-risk scoring (never blocks admission). |
| `internal/repo` | Repository root discovery. |
| `internal/assets` | Embedded install assets used by `xh init`. |
| `internal/attribution` | Episode attribution reporting. |

---

## 4. Data flow

1. The agent (or developer) invokes a CLI command.
2. The handler loads inputs (card path, root, flags).
3. Schema validation runs first; a schema failure exits `1` immediately.
4. For verify, the admission engine loads `policies/admission.yaml` and checks
the evidence floor and required fields per tier (`light`, `standard`, `deep`).
5. Optional enforce flags may add blocking predicates (`boundary`, `context`,
`decision`, `intent`, `contract oracle`, `mutation guard`).
6. The result is emitted as structured JSON/YAML/text and, with `--trace`,
appended to `.x-harness/traces/events.jsonl`.
7. Withheld outcomes include a structured recovery object routing the work to
the appropriate owner.

---

## 5. Where to find X

| What | Where |
| :--- | :---- |
| Authoritative contract | `X_HARNESS.md` |
| Agent rules + managed context | `AGENTS.md` |
| JSON schemas | `schemas/` |
| Admission, recovery, boundary policies | `policies/` |
| Handoff templates | `templates/SUBAGENT_TASK_{light,standard,deep}.md` |
| Golden examples | `examples/golden/` (28 scenario directories; 26 card-backed fixtures + 2 conformance-strict reference scenarios) |
| CI example fixtures | `examples/ci/` |
| Real-world examples | `examples/real-world/` |
| Platform adapters | `adapters/{generic,claude-code,cursor,opencode,antigravity,codex}/` |
| Public docs | `docs/` |
| Go CLI source | `cmd/x-harness/` + `internal/` |
| TypeScript CLI source | `packages/cli/src/` |
| Tests | Go: `*_test.go` next to source; TS: `*.test.ts` under `packages/cli/src/` |
| CI workflows | `.github/workflows/` |
| Packaging / release scripts | `packaging/`, `scripts/` |

---

## 6. Go vs TypeScript asymmetries

- **Canonical runtime**: the Go CLI is the primary runtime. The TypeScript CLI is
a source-checkout compatibility baseline only.
- **Published package**: the npm package ships a Go-binary wrapper; the Node
fallback requires building from source (`npm install && npm run build`).
- **Parity checks**: `npm run parity:check-go` validates Go CLI output against
the committed TypeScript baseline; `npm run parity:capture-ts` updates the
baseline after deliberate changes.
- **Go-only advanced commands**: `explain`, `conformance`, `release`, `boundary`,
`policy`, `scan`, `card`, `readiness`, `adapters`, `repair`, `uninstall`.
These are not implemented in the TypeScript CLI.
- **Maturity labels**: `xh --help-all` and `xh --help-maturity` expose maturity
(`stable`, `beta`, `experimental`, `skeletal`) for every command.

---

## 7. CI / release flow

The main PR gate is `.github/workflows/x-harness-verify.yml`. It runs:

- `quality` — Node 22 matrix for `typecheck`, `test:typecheck`, `build`, `lint`, `format:check`, `test`.
- `go-quality` — Go 1.25.9 matrix for `go test`, `go test -race`, `go vet`,
`go build ./cmd/x-harness`, and `npm run parity:check-go`.
- `go-fuzz-smoke` — bounded Go fuzz target (`FuzzValidate`).
- `verify-gates` — builds both CLIs and runs Go-native primary gates
(strict verify, policy matrix/explain, explain card, evidence run, docs drift,
release verify-docs, doctor, examples verify, regression suite, adversarial
benchmark, conformance minimal and strict) plus TypeScript compatibility parity gates.

Release workflow (`.github/workflows/release.yml`) builds cross-platform Go
binaries, generates SHA256 checksums, signs with cosign, runs platform smoke
tests, and publishes from a dedicated publish job only after smoke passes. SBOM
generation is handled by the release workflow and `.github/workflows/sbom.yml`.

---

## 8. Docs / adapters

- Public docs live in `docs/` and describe current CLI behavior, schemas,
policies, and verification workflows.
- `AGENTS.md` contains a managed context block that is the runtime source of
truth for agent instructions.
- Adapters are thin convention files; they do not fork the contract:
  - `adapters/generic/` — plain Markdown conventions
  - `adapters/claude-code/` — Claude Code skills and `CLAUDE.md`
  - `adapters/cursor/` — Cursor rule file (`.cursor/rules/x-harness.mdc`)
  - `adapters/opencode/` — OpenCode agent docs
  - `adapters/antigravity/` — rules and workflows
  - `adapters/codex/` — Codex adapter (repo-root `AGENTS.md`)

---

## 9. Known correctness gaps / deferred items

- Some commands are declared in the Go CLI skeleton but are not yet fully
implemented; they return a stub usage message. Maturity `experimental` or
`skeletal` indicates this.
- The TypeScript CLI does not implement Go-only commands; parity checks focus
on the overlapping subset.
- Boundary checks are V1 regex-based only; they do not use AST, semgrep, or LLM
analysis.
- Adapter enforcement depends on the host platform reading its convention files;
the harness itself does not monitor or enforce adapter behavior.
- `x-harness` is stable (`1.0.0`). A passing verify gate means the card
meets the policy, not that the underlying code is bug-free.
