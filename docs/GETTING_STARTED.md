# Getting Started

New to x-harness? Start here for the concepts, then move on to the hands-on guides.

## What problem does x-harness solve?

AI agents often claim "I'm done" before the work is truly complete. x-harness turns that claim into an **auditable admission decision** — `accepted` or `withheld` — against a policy stored in your repository.

It does **not** run your agents, replace CI, or guarantee correctness. It is a **read-only policy gate** that evaluates whether an agent's completion claim should be admitted.

## Core concepts

| Concept | Meaning |
| :-- | :-- |
| **Completion card** | A YAML file where an agent records what it claims to have done and the evidence it provides. |
| **Verify gate** | `xh verify` (alias `xh check`). Runs the read-only admission logic against schemas and policies. |
| **Accepted** | The card passes policy. Exit code `0`. The task is admitted as complete. |
| **Withheld** | Any non-success outcome (`failed`, `blocked`, `skipped`, `timeout`, `error`). Exit code `1`. The task is not admitted. |
| **Tier** | `light`, `standard`, or `deep`. Determines how much evidence the card must include. |
| **Evidence floor** | The minimum evidence required for a tier. For example, `standard` requires `files_changed`, `command_evidence`, `done_checklist`, and `prediction`. |

## Important: accepted ≠ correct

A passing `xh verify` means your **card matches the policy**. It does **not** mean the underlying code is bug-free, secure, or production-ready. The verifier is read-only and never edits your source to "fix" things while checking.

## Quick start

Run the guided onboarding:

```bash
xh start
```

This runs a read-only check of your workspace (doctor), verifies the bundled examples, and previews the init wizard.

## Next steps

1. **Hands-on** — [QUICKSTART.md](QUICKSTART.md): build the CLI and run your first verification.
2. **Tutorial** — [tutorials/first-accepted-card.md](tutorials/first-accepted-card.md): end-to-end walkthrough of a valid completion card.
3. **FAQ** — [FAQ.md](FAQ.md): common questions about Go vs TypeScript, LLM usage, and more.
4. **Deep dive** — [ARCHITECTURE.md](ARCHITECTURE.md): layer model, validation cycle, and design notes.

For the full command list, run `xh --help-all`. For the runtime contract, see [X_HARNESS.md](../X_HARNESS.md).
