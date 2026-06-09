# Golden Example: Blocked — Deep Approval Required

A deep-tier completion card that is withheld because it requires human approval.

## Scenario

An agent completes a deep-tier task and provides passing tests. The candidate claim shows `verification.status: passed` and `claim.fix_status: fixed`, appearing to be a successful completion. This card sets `governance.requires_human_approval: true`, so admission requires `governance.approval_status: approved`. Since approval is missing, the runtime admission overrides the candidate success and withholds the completion.

**Intent**: This example demonstrates that `outcome: success` + `acceptance_status: accepted` requires both a passing candidate claim AND admission policy satisfaction. Human approval is a gate that overrides candidate success at the admission layer.

## Files

- `input-task.md` — The original task description.
- `completion-card.yaml` — The agent's completion claim (missing human approval).
- `expected-verify-output.txt` — Expected quiet output from `x-harness verify`.
- `expected-final-response.md` — Expected agent final response.

## Expected verify outcome

```bash
xh verify --card examples/golden/capability/deep-approval-required/completion-card.yaml
# -> WITHHELD
# Reason: tier "deep" requires human approval
```

## Try it

Run the verification gate:

```bash
xh verify --card examples/golden/capability/deep-approval-required/completion-card.yaml
```
