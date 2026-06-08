# Golden Example: Context Manifest Present

A standard-tier completion card whose top-level `context_manifest`
field references a manifest file. The fixture exercises the schema
acceptance path: the admission layer accepts the card because
`context_manifest` is optional at the schema level and the verify-layer
staleness gate (`--context-enforce`) is not exercised by `xh examples
verify`.

## Scenario

A standard-tier agent submits a completion card that includes a
`context_manifest` path. The admission layer should accept the card
because the field is schema-valid and the admission engine does not
enforce manifest staleness (that enforcement lives in the verify
pipeline). The fixture documents the schema shape and provides a
baseline for future verify-layer gate tests.

## Files

- `completion-card.yaml` — standard-tier card with a non-empty
  `context_manifest` string field pointing to the companion manifest.
- `context-manifest.yaml` — a minimal valid context manifest with one
  entry (the README) so the fixture is self-contained.
- `expected-verify-output.txt` — expected quiet summary from
  `xh examples verify` (outcome: success, accepted, 2 passed).
- `README.md` — this file.

## Expected outcome

```yaml
outcome: success
acceptance_status: accepted
checks: 2 passed, 0 failed
```

## Try it

```bash
xh examples verify --suite=regression --json
```

The regression suite must admit this fixture under
`regression/context-manifest-present`.

## What this fixture intentionally does NOT cover

- The verify-layer `context_stale` blocking predicate is wired to
  `--context-enforce block` and the `ci-strict`/`governed-deep`
  profiles. `xh examples verify` does not apply those flags, so a
  blocked-by-stale-manifest fixture would not be exercised
  deterministically by the regression engine.
- The light-tier path is silent: a light-tier card with no
  `context_manifest` emits no advisory note.
- The fixture does not exercise the `xh context manifest write/check`
  flows; those are covered by unit tests in
  `internal/cli/verify_context_enforce_test.go` and
  `packages/cli/tests/context-manifest.test.ts`.
