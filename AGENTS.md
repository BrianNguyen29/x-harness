# x-harness Agent Contract

This repository uses x-harness.

Agents may perform work and propose completion. Agents may not self-admit completion.

A result with `fix_status: fixed` is only a completion candidate. Accepted completion requires a read-only verify gate pass.

## Canonical tiers

Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.

## Completion rule

Completion is accepted only when x-harness emits:

```yaml
verify_gate.outcome: success
acceptance_status: accepted
```

All other outcomes are withheld: `failed`, `blocked`, `skipped`, `timeout`, and `error`.

## Verifier rule

The verifier is read-only. It may inspect files, tasks, stories, templates, returns, evidence, diffs, command output, and trace events. It must not edit source files or repair the work product while verifying.

## PGV rule

PGV advice is advisory-only. It never overrides verify and never grants admission authority by default.
