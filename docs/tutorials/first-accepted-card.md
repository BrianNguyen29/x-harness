# First Accepted Card

A current-behavior walkthrough: scaffold a fresh workspace with the
`xh init --wizard`, run a card through the full `verify → explain →
recover → accepted` cycle, and finish with `xh examples verify`. Every
command in this guide ships in the current Go CLI; nothing is aspirational.

## Prerequisites

Build the Go CLI once and keep the binary on your `PATH` (or invoke it
as `./x-harness` from the repo root):

```bash
go build ./cmd/x-harness
```

From here on, `xh` means `./x-harness` (the repo binary) or the
installed alias if you copied it to `~/bin/xh`.

## 1. Scaffold a workspace (preview, then apply)

`xh init --wizard` is a deterministic 3-step plan (profile → planned
actions → apply decision) that wraps the existing `xh init` copy logic.
It is stdin-free and safe in CI.

```bash
# Preview only — does not write any file.
xh init /tmp/first-accepted --wizard --wizard-dry-run

# Apply the minimal profile to the target directory.
xh init /tmp/first-accepted --wizard
```

Expected end-state of the apply step:

```text
# xh init --wizard complete
  profile: minimal
  target:  /tmp/first-accepted
x-harness init (minimal) complete: 10 assets copied
```

## 2. Health-check the workspace

`xh doctor` validates that the scaffolded assets compile, policies
parse, and the managed-context block in `AGENTS.md` is fresh:

```bash
cd /tmp/first-accepted
xh doctor --root . --json
# -> "healthy": true
```

## 3. Verify a card with `--profile light-local`

Copy the runnable recovered-flow example into your workspace and run
the verify gate against its **initial withheld** card:

```bash
cp <repo>/examples/golden/regression/success-recovered-flow/initial-withheld-card.yaml card.yaml
xh verify --card card.yaml --profile light-local
# -> outcome: failed, acceptance_status: withheld
```

A withheld result is the expected starting point — that is the entry
to the recovery flow.

## 4. Explain and recover

Use `xh explain --card` to surface the blocking predicate, then
preview the conservative patch before applying it:

```bash
xh explain --card card.yaml
# -> blocking_predicates: [schema_invalid], next_action: review_and_resubmit

xh recover --patch-card card.yaml --evidence <repo>/examples/golden/regression/success-recovered-flow/src/utils/formatSlug.go
# -> preview: ops that would_set handoff.next_action / handoff.owner / claim.evidence

xh recover --patch-card card.yaml --confirm --evidence <repo>/examples/golden/regression/success-recovered-flow/src/utils/formatSlug.go
# -> backup: card.yaml.bak.<unix-ms>
```

The patcher is intentionally narrow: it fills the handoff block and
appends the `--evidence` file; it never overwrites a user-provided
scalar. Writing `evidence.manual_rationale` remains the agent's job —
see the worked example below for that hand-off.

## 5. Re-verify until accepted

`xh recover --patch-card` removes the schema-level blocks but the
light-tier evidence floor still wants a `manual_rationale` or
`command_evidence`. The recovered-flow example shows the full path:

- See [Golden Example: Success — Recovered Flow](../../examples/golden/regression/success-recovered-flow/README.md)
  for a step-by-step reproduction, then re-verify:

```bash
xh verify --card card.yaml --profile light-local
# -> outcome: success, acceptance_status: accepted
```

A non-zero exit means the evidence floor is not met yet; loop between
step 4 and step 5 until the output is `accepted`.

## 6. Run the bundled examples gate

Once your own card is accepted, sanity-check the regression suite
ships clean:

```bash
xh examples verify --suite=regression --json
# -> "ok": true, "passed" equals "total"
```

## Next docs

- [Quickstart](../QUICKSTART.md) — full beginner command tour.
- [Recovery](../RECOVERY.md) — predicate-to-action routing table.
- [Verify Gate](../VERIFY_GATE.md) — admission policy details.
- [Admission Policy](../ADMISSION_POLICY.md) — evidence floor per tier.
