# ClaimGate Runtime Summary

ClaimGate separates four concepts:

1. **Execution** — an agent performs work.
2. **Claim** — an agent proposes that the work is complete.
3. **Acceptance** — a read-only verifier checks the claim and evidence.
4. **Completion** — the accepted result is surfaced.

A task is not complete because a worker says it is complete. A task is complete only after admission.

## File-first design

The source of truth is the repository files: Markdown templates, JSON Schemas, YAML policies, examples, and adapters. The TypeScript CLI validates and generates files, but does not replace the files as the canonical contract.
