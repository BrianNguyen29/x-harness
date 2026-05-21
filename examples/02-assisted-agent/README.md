# Assisted Agent Example

A worker produces claim and evidence; a verifier checks them before admission.

## Files

- `claim.yaml`: Worker claim.
- `evidence.yaml`: Supporting evidence.
- `subagent-task.md`: Task handoff.
- `verify-report.md`: Verifier report.

## Expected verify outcome

```bash
npx x-harness verify --claim examples/02-assisted-agent/claim.yaml --evidence examples/02-assisted-agent/evidence.yaml --tier standard --json
# -> accepted (legacy compatibility mode)
```
