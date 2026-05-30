# Golden Example: Blocked — Missing Context Ref

A standard-tier completion card that is withheld because a referenced context file does not exist.

## Scenario

An agent claims completion for a standard-tier task with a `context_alignment` block. The block includes `stale_ground_checked: true` and at least one non-empty ref array, but one of the referenced files (`src/docs/missing-product-contract.md`) does not exist. When `--context-floor` is enabled, the verify gate withholds admission because the context floor requires all referenced files to exist.

## Fixture Purpose

This fixture validates the `--context-floor` failure path where:
- `context_alignment` is present
- `stale_ground_checked` is true
- A ref array is non-empty
- But a referenced file is missing

Without `--context-floor`, this card would pass because the `context_alignment` block structure is otherwise valid.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The agent's completion claim (references missing file).
- `expected-verify-output.txt` — Expected output when verifying without `--context-floor` (card passes).
- `expected-context-floor-output.txt` — Expected output when verifying with `--context-floor` (card is withheld).
- `README.md` — This file.

## Expected verify outcome

```bash
./x-harness verify --card examples/golden/regression/blocked-missing-context-ref/completion-card.yaml --context-floor --json
# -> FAILED (exit non-zero)
# JSON fields: admission_outcome=failed, acceptance_status=withheld
# withheld_reason: class=context_floor_blocked, stage=context, owner=implementation-worker
#                  failure_class=context_missing, failure_stage=context_floor
```

## Note on examples verify

`examples verify` runs without `--context-floor`, so it uses `expected-verify-output.txt` (card passes with `outcome: success`). With `--context-floor`, the verify uses `expected-context-floor-output.txt` (card fails with `admission_outcome: failed`, `acceptance_status: withheld`, `blocking_predicate: context_floor_blocked`). The dedicated CLI test `TestVerifyContextFloorMissingFileRef` exercises the `--context-floor` failure path.