# Conformance Strict Profile Specification

> **Status:** Implemented (strict v1 minimal)
> **Version:** 1.0-minimal
> **Scope:** This document defines the behavior of `conformance run --profile strict`. The strict v1 minimal profile is implemented and enforced. Residual follow-ups (overclaim detection, schema-field consistency parser, full runtime card path scoping, network/remote checks, waiver subsystem, CI gate) are documented below and are not claimed as implemented.

---

## 1. Purpose and non-goals

### 1.1 Purpose

The strict conformance profile turns `x-harness` from a self-checking CLI into a contract-checkable admission/readiness harness standard. It verifies that a repository (and the implementation verifying it) satisfies a superset of the minimal profile, plus operational hardening checks that reduce the risk of overclaim, silent drift, and unverified mutations.

### 1.2 Non-goals

- The strict profile is **not** a release readiness gate by itself. Release readiness requires the separate release evidence bundle (see `docs/RELEASE_CANDIDATE.md`).
- The strict profile does **not** turn `x-harness` into an agent runtime, CI system, MCP platform, dashboard, sandbox, or database-backed workflow engine.
- The strict profile does **not** guarantee code correctness; it guarantees conformance to the x-harness admission contract.
- Hooks, MCP adapters, sandbox bridges, and server/daemon runtime features are explicitly out of scope for strict profile v1 (see Section 9).

---

## 2. Relationship to `minimal`

`strict` is a **superset** of `minimal`.

| Aspect | `minimal` | `strict` |
|---|---|---|
| Critical files exist | Yes | Yes (inherited) |
| Schemas compile | Yes | Yes (inherited) |
| Policies parse | Yes | Yes (inherited) |
| AGENTS.md managed block valid | Yes | Yes (inherited) |
| Golden examples pass | Yes | Yes (inherited, expanded) |
| Denominator contract | Yes | Yes (inherited) |
| Mutation guard | No | **Blocking** |
| Approval receipt / provenance | No | **Blocking for high-risk** |
| Adapter doctor / managed block drift | No | **Blocking** |
| Scanner high-severity findings | No | **Blocking** |
| Regression and adversarial suites | No | **Blocking** |
| Context GC / staleness checks | No | **Blocking** |
| Worktree metadata / strict path enforcement | No | **Blocking** |

A repository or implementation that fails minimal automatically fails strict.

---

## 3. Blocking checks vs advisory checks

### 3.1 Blocking checks (fail-closed)

Blocking checks set `report.OK = false` and cause `conformance run --profile strict` to exit with a non-zero code. The JSON report marks individual checks as `"failed"`.

### 3.2 Advisory checks

Advisory checks produce `"advisory"` status notes in the report but do **not** set `report.OK = false`. They are intended for human review during PR or release readiness workflows.

### 3.3 Waiver interaction

When Phase 4 waiver support is implemented, a blocking check may be downgraded to advisory if a **valid waiver** is present (see Section 7). An expired, unscoped, or untraceable waiver does **not** downgrade a blocking check.

In strict v1, the waiver subsystem is not yet runtime-enforced. High severity scanner findings and other blocking checks remain blocking with no waiver downgrade.

---

## 4. Strict blocking checks (implemented)

### 4.1 Minimal conformance baseline

Strict runs the entire minimal profile first. If any minimal check fails, strict exits immediately with the minimal report and a non-zero code.

### 4.2 Mutation guard / dirty worktree evidence floor

**Check:** `mutation_guard_verified`

- Strict requires a Git workspace. Non-Git workspaces cannot pass strict conformance.
- In a Git workspace, `strict` runs a bounded before/after snapshot of the working tree and verifies that no unexpected file changes occurred during the conformance run.
- The `.x-harness/` directory and its contents are allowlisted.
- If no baseline can be established, the check fails fail-closed.
- **Note:** This check verifies that the *conformance runner itself* did not mutate source files. It does not require a clean worktree prior to execution.

### 4.3 Approval receipt / provenance for high-risk commands

**Check:** `approval_receipt_for_high_risk`

- Uses the permission intent classifier (`internal/classify`) to inspect `command_evidence` entries on standard and deep completion cards.
- High-risk or unknown-risk commands without an explicit `approval_receipt` block admission.
- The check validates that any present receipt includes: approved decision, non-empty approver, matching command coverage, and sufficient aggregate risk classification.
- Light-tier cards are advisory-only; standard requires high/unknown without receipt => fail; deep requires medium/high/unknown without receipt => fail.
- **Note:** The approval receipt schema is implemented (`schemas/approval-receipt.schema.json`). This check validates its enforcement in conformance context.

### 4.4 Adapter doctor / managed block drift

**Check:** `adapter_doctor_no_drift`

- Runs `adapters doctor` (or its programmatic equivalent) against all registered adapters.
- **Implemented:** Missing adapter README is blocking. Each subdirectory under `adapters/` must contain a `README.md`.
- **Deferred:** Non-empty capabilities/formats declaration is advisory-only because no machine-readable standard exists yet.
- Managed blocks in adapter files must have matching contract hashes.
- Fails if any adapter has managed block drift or missing contract references.
- This check is intentionally scoped to adapter files under `adapters/`; it does not validate external agent behavior. External URL or network checks are out of scope for v1 and may be added as an opt-in `--remote` flag in the future if needed.

### 4.5 Scanner high-severity findings and waiver policy

**Check:** `scanner_high_severity_clear`

- Runs the deterministic static scanner (`scan adapter`, `scan skill`, `scan managed`) using built-in regex heuristics.
- Any `high` severity finding blocks strict conformance.
- `medium` severity findings are advisory in strict v1.
- When Phase 4 waiver support is implemented, a valid waiver may downgrade a high severity finding to advisory.
- The scanner is report-only in minimal; strict makes it blocking.

### 4.6 Regression and adversarial suites

**Check:** `regression_suite_passed`
**Check:** `adversarial_suite_passed`

- Runs the structured regression suite and verifies that all fixtures under `examples/golden/regression/` produce expected outcomes.
- Runs the adversarial benchmark suite (`benchmark --filter adversarial`) and verifies that adversarial fixtures are correctly withheld.
- Either suite failing causes strict conformance to fail.
- The capability suite is advisory indefinitely and non-blocking in strict v1.

### 4.7 Context GC / staleness checks

**Check:** `context_gc_no_stale_drift`

- **Implemented:** Stale managed block detection (AGENTS.md hash/body validation) and dead internal docs link detection scoped to `docs/*.md` and repo-local links only.
- **Deferred:** Overclaim phrase detection and full runtime card path scoping are not yet implemented.
- Fails if `doctor --staleness` reports unmanaged drift between canonical contracts and adapter instructions.
- This check prevents README/docs from silently diverging from `policies/admission.yaml` and `schemas/`.

### 4.8 Worktree metadata and strict path enforcement

**Check:** `worktree_metadata_valid`

- Verifies that trace and report outputs include worktree metadata when enabled (branch, commit, worktree root).
- **Implemented:** Golden completion-card artifact path scoping — artifact-like paths in golden fixtures under `examples/golden/` are validated to not escape the worktree root (no absolute paths or `../` traversal).
- **Deferred:** Full runtime card path scoping for arbitrary verification inputs is not yet implemented.
- Verifies that mutation guard baselines are bound to the worktree root (not to an arbitrary parent directory).
- **Note:** Strict path enforcement does not reject paths outside the worktree; it reports them as findings and fails conformance.

### 4.9 Denominator-safe report metrics

**Check:** `denominator_contract_enforced`

- Verifies that generated report JSON does not contain ambiguous `success_rate` fields.
- Verifies that every rate metric includes `numerator`, `denominator`, and `unit`.
- Verifies that `task_completion_coverage` is `not_computable` unless an aligned task denominator exists.
- Inherited from minimal; strict adds enforcement that new report schemas do not regress this contract.

---

## 5. Exit code / JSON output semantics

### 5.1 Exit codes

| Code | Meaning |
|---|---|
| `0` | Strict conformance passed. All blocking checks passed. Advisory notes may be present. |
| `1` | At least one blocking check failed. |
| `2` | Usage error (unknown profile, missing required flag, etc.). |

### 5.2 JSON output shape

```json
{
  "profile": "strict",
  "ok": false,
  "checks": [
    { "name": "critical_files_exist", "status": "passed", "note": "" },
    { "name": "mutation_guard_verified", "status": "failed", "note": "unexpected delta: src/main.go" },
    { "name": "scanner_high_severity_clear", "status": "advisory", "note": "1 medium finding in adapters/cursor/rules.md" }
  ],
  "waivers": [
    { "check": "scanner_high_severity_clear", "rule_id": "example", "reason": "reviewed", "expires": "2026-06-01" }
  ]
}
```

- `ok` is `true` only when every blocking check has `status: passed`.
- `waivers` is present only when at least one waiver was evaluated.
- Advisory checks do not affect `ok`.

---

## 6. Waiver policy

### 6.1 Principles

- Waivers must be **explicit** (declared in a machine-readable file, not inferred).
- Waivers must be **scoped** (apply to a specific check or scanner rule ID, not a blanket override).
- Waivers should be **expiring** when possible (an `expires` field or a max TTL).
- Waivers are **advisory until implemented**; the waiver subsystem itself is not yet runtime-enforced.

### 6.2 Proposed waiver file

Location: `.x-harness/conformance-waivers.yaml`

```yaml
waivers:
  - check: scanner_high_severity_clear
    rule_id: remote-pipe-shell
    reason: "Intentional integration test helper; not used in production path."
    expires: "2026-06-01"
    approved_by: "maintainer-review"
```

### 6.3 Invalid waivers

A waiver is invalid and ignored if:
- `expires` is in the past.
- `reason` is empty or generic (e.g., "ok", "approved").
- `check` does not match a known strict check name.
- `rule_id` is required but missing (for scanner waivers).
- The waiver file itself fails YAML parsing.

---

## 7. Residual follow-ups (not implemented)

The following hardening gaps are **explicitly deferred** from strict v1 minimal. They are documented here to avoid false claims of completeness.

| Feature | Rationale | Status |
|---|---|---|
| Overclaim phrase detection | Requires stable heuristic; false-positive risk | Deferred to v1.1 or later |
| Schema-field consistency parser | Needs structured diff against canonical context | Deferred to v1.1 or later |
| Full runtime card path scoping | Currently scoped to golden fixtures only | Deferred to v1.1 or later |
| Network / remote URL checks | No external network calls in deterministic checks | Deferred; may be opt-in `--remote` flag |
| Waiver subsystem | Not yet runtime-enforced; blocking checks remain blocking | Deferred to Phase 4 |
| CI gate for strict | Documented but not activated in workflow | Deferred until profile is stable |
| Adapter capability/formats advisory | No machine-readable standard exists yet | Advisory-only when standard emerges |

## 8. Deferred / P3 exclusions

The following features are **explicitly excluded** from strict profile v1. They may be added in future iterations if real demand exists and they do not expand x-harness into runtime ownership.

| Feature | Rationale | Future trigger |
|---|---|---|
| Hooks bridge | Out of scope for admission contract | Adapter demand for pre/post verify hooks |
| MCP read-only evidence adapter | Out of scope; admission authority stays deterministic | Real need for MCP tool evidence ingestion |
| Sandbox bridge | No Docker or sandbox dependency by default | Need for sandboxed command evidence |
| Dashboard / server / runtime daemon | File-first, CLI-only architecture | Not planned unless architecture changes |
| Database-backed state | Source of truth is repo-local files | Not planned |
| Plugin marketplace | Core must stay narrow | Not planned |

---

## 9. Implementation plan after approval

The core strict runner (Phases 1–2) is **implemented** as of strict v1 minimal. The remaining phases are deferred.

### 9.1 Phase 1 — Core strict runner ✅ Done

1. ✅ Add `conformance.RunStrict(root string) *Report` in `internal/conformance/`.
2. ✅ Implement each blocking check as a standalone function, reusing existing internal packages where possible (`mutationguard`, `scanner`, `admission`, `repo`, etc.).
3. ✅ Wire `RunStrict` into `internal/cli/conformance.go` so `conformance run --profile strict` invokes it.
4. ✅ Update `handleConformanceRun` to accept `strict` as a valid profile.

### 9.2 Phase 2 — Fixtures and tests ✅ Done

1. ✅ Add golden fixtures for strict success and strict failure cases under `examples/golden/conformance-strict/`.
2. ✅ Add unit tests in `internal/conformance/conformance_test.go` for each strict check.
3. ✅ Add CLI-level tests in `internal/cli/conformance_test.go` for profile parsing and exit codes.
4. Add parity checks if the TypeScript compatibility layer exposes conformance. *(TypeScript layer does not expose strict; not required for v1 minimal.)*

### 9.3 Phase 3 — Optional CI gate ⏳ Deferred

1. Update `.github/workflows/x-harness-verify.yml` with a `conformance strict` job (disabled by default until the profile is stable).
2. Document the CI gate in `docs/CI.md`.
3. Update `docs/RELEASE_CANDIDATE.md` to reference strict conformance as a recommended (not required) pre-release check.

### 9.4 Phase 4 — Waiver subsystem ⏳ Deferred

1. Add waiver file parser and validator.
2. Integrate waiver evaluation into strict check functions.
3. Add `conformance report --format markdown` support for waiver annotations.

---

## 10. Approved policy decisions

The following decisions have been reviewed and approved. They are encoded below for reference during implementation.

1. **Waiver file location:** `.x-harness/conformance-waivers.yaml`.
2. **Capability suite blocking:** Advisory indefinitely; non-blocking in strict v1.
3. **Medium severity scanner findings:** Advisory in strict v1.
4. **Non-git fallback:** Strict requires a Git workspace. Non-Git workspaces cannot pass strict conformance.
5. **Exit code `2` usage:** Reserved for usage errors only. Conformance failures use exit code `1`.
6. **Golden example expansion:** Strict-specific fixtures live under `examples/golden/conformance-strict/` (e.g., `success-strict`, `blocked-strict-mutation-guard`).
7. **Adapters doctor scope:** File-local only in v1. External URL or network checks are out of scope and may be added as an opt-in `--remote` flag in the future if needed.
8. **Release profile boundary:** Independent gate. Strict conformance is recommended but not a hard prerequisite for release.

---

## 11. References

- `internal/conformance/conformance.go` — current minimal and strict implementation
- `internal/cli/conformance.go` — current CLI wiring
- `docs/VERIFY_GATE.md` — mutation guard and read-only verification semantics
- `docs/ADMISSION_POLICY.md` — evidence floor and tier requirements
- `docs/CI.md` — CI workflow definitions
- `docs/RELEASE_CANDIDATE.md` — release readiness criteria
- `policies/admission.yaml` — admission policy manifest
- `X_HARNESS_ADMISSION_READINESS_UPDATED_ROADMAP.md` — roadmap Sections 10, 16, 23, 24, 25
- `internal/mutationguard/mutationguard.go` — mutation guard implementation
- `internal/scanner/scanner.go` — static scanner rules and severity model
- `internal/classify/classify.go` — permission intent classifier
- `internal/adaptercheck/adaptercheck.go` — adapter doctor implementation
- `internal/contextcheck/contextcheck.go` — context GC / staleness checks
- `internal/worktree/worktree.go` — worktree metadata collection

---

*End of specification. The strict v1 minimal profile is implemented and runtime-enforced. Residual follow-ups are documented in Section 7.*
