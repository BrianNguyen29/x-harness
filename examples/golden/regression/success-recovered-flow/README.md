# Golden Example: Success — Recovered Flow

A real-world example that walks through the full `withheld → explain → recover
→ patch → accepted` lifecycle for a small Go CLI task, using only the
Go CLI commands that already exist in this repository.

## Scenario

A developer adds a small utility `formatSlug` to a Go CLI. The initial
completion card they write omits a few handoff fields and the light-tier
evidence floor, so the card is **withheld** by admission. The example
documents the deterministic recovery path that takes it to **accepted**:

1. Run `xh verify` and see `outcome: failed`, `acceptance_status: withheld`.
2. Run `xh explain --card` to surface the blocking predicate and the
   next-action hint.
3. Run `xh recover --patch-card` (preview only, no file mutation).
4. Run `xh recover --patch-card --confirm --evidence <file>` to apply a
   conservative patch with a backup, then re-run `xh verify` and reach
   `outcome: success`, `acceptance_status: accepted`.

This example does **not** promise any new product behavior. The
commands above are the exact CLI surface that ships today. Anything
outside that surface is out of scope for this slice.

## Files

- `src/utils/formatSlug.go` — placeholder Go source file referenced by
  the completion card. Not part of the buildable Go CLI; only used as
  a path in evidence and to make the example self-contained.
- `completion-card.yaml` — the final accepted completion card. This is
  the file that `xh examples verify --suite=regression` reads.
- `expected-verify-output.txt` — expected quiet summary from
  `xh verify --card completion-card.yaml`. Picked up by both the Go
  CLI (`internal/cli/examples.go`) and the TypeScript CLI
  (`packages/cli/src/commands/examples.ts`).
- `initial-withheld-card.yaml` — the **starting** withheld card. The Go
  and TypeScript `examples verify` implementations look up the
  example by directory name and only read `completion-card.yaml`, so
  this file is not part of the regression scan. It is provided so a
  reader can copy it into a scratch directory and reproduce the
  recovery flow literal.
- `README.md` — this file.

## Try it (Go CLI, end-to-end)

```bash
# 0. Build the Go CLI once.
go build ./cmd/x-harness

# 1. Confirm the new example is picked up by the regression suite and
#    admitted.
./x-harness examples verify --suite=regression --json

# 2. Reproduce the recovery flow in a scratch directory.
mkdir -p /tmp/p2-s4-recovered-flow
cp examples/golden/regression/success-recovered-flow/initial-withheld-card.yaml \
   /tmp/p2-s4-recovered-flow/card.yaml

# 2a. Initial verify: WITHHELD.
./x-harness verify --card /tmp/p2-s4-recovered-flow/card.yaml
# -> outcome: failed, acceptance_status: withheld

# 2b. Explain what is blocking admission.
./x-harness explain --card /tmp/p2-s4-recovered-flow/card.yaml
# -> blocking_predicates: [schema_invalid], etc.

# 2c. Preview the conservative patch (no file mutation by default).
./x-harness recover --patch-card /tmp/p2-s4-recovered-flow/card.yaml
# -> ops: would_set handoff.next_action, would_set handoff.owner

# 2d. Apply the patch with an explicit --confirm and add evidence in
#     the same pass. The patcher creates a sibling .bak.<unix-ms>
#     file before writing.
./x-harness recover --patch-card /tmp/p2-s4-recovered-flow/card.yaml \
  --confirm --evidence src/utils/formatSlug.go

# 2e. Re-verify. The deterministic patcher filled the handoff block
#     and appended the evidence file to claim.evidence /
#     evidence.files_changed. The remaining evidence floor
#     (manual_rationale or command_evidence) is still the agent's
#     responsibility; this example mirrors that hand-off explicitly.
./x-harness verify --card /tmp/p2-s4-recovered-flow/card.yaml
# -> outcome: success, acceptance_status: accepted (after the agent
#    fills evidence.manual_rationale, see completion-card.yaml).
```

## Try it (TypeScript CLI, parity check)

```bash
# Build once.
npm run build

# The TypeScript CLI does not expose --suite; it scans all suites.
node packages/cli/dist/index.js examples verify
# -> ✓ regression/success-recovered-flow: success (accepted)
```

## What the example intentionally does NOT cover

- The patcher is intentionally narrow. It only fills
  `handoff.next_action`, `handoff.owner`, `claim.evidence`, and
  `evidence.files_changed` when those fields are empty. It never
  overwrites a user-provided scalar and never edits source files.
- The `xh recover --patch-card` flag is opt-in. Default behavior is
  preview-only, matching the Phase 1 read-only verifier contract.
- The recovery flow leaves one step to the agent: writing
  `evidence.manual_rationale` (or running `xh evidence run` to
  produce `evidence.command_evidence`). The patcher cannot fabricate
  a rationale; doing so would forge evidence, which the contract
  forbids. See `internal/cli/recover_patch.go` for the full
  patch-category list and the deferred-risks section of
  `de_xuat_cai_thien_x_harness-2.md` § P2-S3.
- Next.js/TypeScript and monorepo real-world examples are deferred
  to a follow-up slice. This example focuses on a single Go CLI
  file to keep the diff surgical and the verifier scope unchanged.
