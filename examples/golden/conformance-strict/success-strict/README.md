# success-strict

Strict conformance pass fixture.

This directory documents the expected output when the `strict` conformance
profile passes on a healthy repository.  The report below was captured from a
real successful run (`x-harness conformance run --profile strict --json`).

## Report snapshot

```json
{
  "profile": "strict",
  "ok": true,
  "checks": [
    {"name": "critical_files_exist",         "status": "passed", "note": "all critical files present"},
    {"name": "schemas_compile",              "status": "passed", "note": "27 schema(s) compiled"},
    {"name": "policies_parse",               "status": "passed", "note": "19 policy file(s) parsed"},
    {"name": "agents_managed_context",       "status": "passed"},
    {"name": "golden_success_light",         "status": "passed", "note": "outcome=success acceptance=accepted"},
    {"name": "golden_blocked_missing_evidence", "status": "passed", "note": "outcome=failed acceptance=withheld"},
    {"name": "denominator_contract",         "status": "passed", "note": "report schema validates denominator-safe rate metrics"},
    {"name": "mutation_guard_verified",      "status": "passed", "note": "no unexpected changes detected"},
    {"name": "scanner_high_severity_clear",  "status": "passed", "note": "no high or medium severity findings (31 files scanned)"},
    {"name": "worktree_metadata_valid",      "status": "passed", "note": "root=... branch=... commit=..."},
    {"name": "adapter_doctor_no_drift",      "status": "passed", "note": "21 adapter file(s) checked"},
    {"name": "context_gc_no_stale_drift",    "status": "passed", "note": "AGENTS.md managed context block is fresh; no dead internal doc links"},
    {"name": "approval_receipt_for_high_risk", "status": "passed", "note": "3 approval fixture(s) validated"},
    {"name": "regression_suite_passed",      "status": "passed", "note": "19 fixture(s) matched"},
    {"name": "adversarial_suite_passed",     "status": "passed", "note": "3 fixture(s) matched"}
  ]
}
```

All 15 strict checks must report `passed` for the overall strict conformance run
to be considered successful.

## Strict v1 minimal vs residual

The following checks are implemented and blocking in strict v1 minimal:
- `adapter_doctor_no_drift` — README existence is enforced; managed block drift is enforced.
- `context_gc_no_stale_drift` — AGENTS.md managed block freshness and dead internal doc links (`docs/*.md`) are enforced.
- `worktree_metadata_valid` — Git worktree metadata and golden fixture artifact path scoping are enforced.

The following are explicitly deferred (see `docs/CONFORMANCE_STRICT_PROFILE.md` Section 7):
- Overclaim phrase detection
- Schema-field consistency parser
- Full runtime card path scoping
- Network / remote URL checks
- Waiver subsystem
- CI gate activation
- Adapter capability/formats advisory (no machine-readable standard yet)
