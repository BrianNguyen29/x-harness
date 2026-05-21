# Generic x-harness AGENTS.md

## Rules

1. **Use light by default**. Prefer the smallest tier that preserves correctness.
2. **Use standard for multi-step**. Use when the task involves research, review, synthesis, or bounded implementation across multiple files.
3. **Use deep only for risk/control decisions**. Use when the cost of being wrong is high, the ground is stale, or rollback is non-trivial.
4. **Write completion card before claiming completion**. A result with `fix_status: fixed` is only a candidate. Accepted completion requires a verified completion card.
5. **Verifier is read-only**. The verifier inspects files, evidence, and diffs. It must not edit source files or repair the work product while verifying.
6. **Non-success verify -> withheld**. Any verify outcome other than `success` results in `acceptance_status: withheld`.
7. **PGV advisory-only**. PGV risk levels are advisory. They never override verify and never grant admission authority by default.

## Tiers

Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

## Completion rule

Completion is accepted only when x-harness emits:

```yaml
verify_gate.outcome: success
acceptance_status: accepted
```

All other outcomes are withheld: `failed`, `blocked`, `skipped`, `timeout`, and `error`.
