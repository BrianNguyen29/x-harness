---
description: Run x-harness read-only verification
trigger:
  - "Verify this with x-harness"
  - "Run x-harness verification"
  - "Check completion card"
allowed-tools: Read, Grep, Glob, Bash
---

# x-harness-verify

Use this skill to verify a completion claim.

## Rules

- **Read-only**. Do not edit source files.
- Inspect `completion-card.yaml`.
- Inspect changed files and evidence if available.
- Run `npx x-harness verify`.
- Return one outcome:
  - `success`
  - `failed`
  - `blocked`
  - `skipped`
  - `timeout`
  - `error`

## Admission mapping

- `success` -> `accepted`
- `failed` -> `withheld`
- `blocked` -> `withheld`
- `skipped` -> `withheld`
- `timeout` -> `withheld`
- `error` -> `withheld`

Only `success` maps to `accepted`. Everything else maps to `withheld`.

## PGV

PGV advice is advisory-only. It never overrides verify and never grants admission authority.

## Do not treat as accepted completion

- `fix_status: fixed`
- `verification.status: passed`
- `pgv_advice.claim_allowed: yes`

## Stop condition

Return the verify outcome and handoff. Do not edit files to fix findings.
