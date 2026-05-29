# Governance Approval Workflow Design

> **Status:** Design-gated. Not yet implemented as a unified workflow.
> **Version:** 1.0-design
> **Scope:** This document defines the intended governance approval workflow for x-harness. Individual components (approval receipt schema, deep-tier approval gate, authority boundary checks, intervention validation) exist in the codebase; the end-to-end request/approve/status workflow described here is design-only and requires explicit approval before implementation begins.

---

## 1. Purpose and non-goals

### 1.1 Purpose

x-harness already enforces authority boundaries (`policies/authority.yaml`), admission policy (`policies/admission.yaml`), and tier-aware evidence floors. The governance approval workflow adds a deterministic, human-in-the-loop (HITL) gate for:

1. **Deep-tier tasks** that declare `governance.requires_human_approval: true`.
2. **High-risk command evidence** that triggers the permission intent classifier (`internal/classify`).
3. **Protected-path changes** that match `human_only` or `agent_proposable_human_approved` authority classes.
4. **Intake tier downgrades** that require explicit governance intervention.

The workflow must remain file-first, offline-capable, and compatible with the existing CLI-assisted architecture. It must not introduce a server, database, or background daemon.

### 1.2 Non-goals

- **Not a workflow engine.** x-harness does not queue tasks, send notifications, or manage state transitions beyond file artifacts.
- **Not an OAuth / SSO / RBAC provider.** Identity, authentication, and role-based access control are out of scope. Approvers are identified by arbitrary string labels (`approver: user-name`). Trust is established out-of-band (e.g., commit signing, PR review, or local policy).
- **Not a server or daemon.** The workflow operates via CLI commands and file artifacts only.
- **Not a replacement for admission policy.** Approval is an additive gate, not a bypass. A completed approval does not override a failing evidence floor or schema violation.
- **Not automatically enforced in CI** until the design is promoted from design-gated to implemented.

---

## 2. Layered governance model

Governance in x-harness is layered. Each layer is independently verifiable, and the verify gate evaluates them in sequence.

```txt
┌─────────────────────────────────────────────────────────────┐
│  LAYER 4: ADMISSION GATE                                    │
│  - Evidence floor, fix_status, verification.status          │
│  - admission.outcome / acceptance_status                    │
└──────────────────────────┬──────────────────────────────────┘
                           │ blocks if any layer fails
┌─────────────────────────────────────────────────────────────┐
│  LAYER 3: GOVERNANCE APPROVAL GATE                          │
│  - Human approval for deep/high-risk/protected paths        │
│  - approval_receipt validation                              │
│  - Intervention artifact validation                         │
└──────────────────────────┬──────────────────────────────────┘
                           │ blocks if approval missing/invalid
┌─────────────────────────────────────────────────────────────┐
│  LAYER 2: AUTHORITY BOUNDARY CHECK                          │
│  - policies/authority.yaml protected paths                  │
│  - Report-only by default; --enforce makes it blocking      │
└──────────────────────────┬──────────────────────────────────┘
                           │ warns or blocks depending on mode
┌─────────────────────────────────────────────────────────────┐
│  LAYER 1: INTAKE CLASSIFICATION                             │
│  - policies/intake.yaml signals and tier mapping            │
│  - Auto-escalation for high_risk signals                    │
└─────────────────────────────────────────────────────────────┘
```

### Layer interaction rules

1. **Intake (Layer 1)** is advisory at task creation time. It maps keywords to tiers and may auto-escalate. It does not block admission by itself.
2. **Authority (Layer 2)** is report-only by default (`policies/authority.yaml: report_only: true`). The `governance check` command emits warnings. With `--enforce`, it exits non-zero for violations. In admission context, authority violations are currently advisory-only unless paired with an explicit `governance.requires_human_approval` flag.
3. **Governance Approval (Layer 3)** is fail-closed for deep tasks that require human approval. Missing or invalid approval produces `admission.outcome: blocked` or `withheld`.
4. **Admission (Layer 4)** is the final gate. It evaluates the evidence floor, schema validity, and policy predicates. Governance approval is one of those predicates.

---

## 3. Approval artifact schema

### 3.1 Completion-card governance block (existing)

The completion card schema already defines a `governance` object:

```yaml
governance:
  risk_class: low | medium | high
  requires_human_approval: true | false
  approval_required_for:
    - "auth logic change"
  approval_status: not_required | pending | approved | rejected
  approver: "user-name"
  approval_artifact:
    path: "governance/approval-TASK-001.yaml"
    sha256: "abc123..."
```

**Implementation status:** `governance` block is schema-defined and validated by the verify gate. `approval_artifact` is parsed but the workflow that generates and validates the external artifact file is design-gated.

### 3.2 Approval receipt (existing, enforced in strict conformance)

The `approval_receipt` field on the completion card is used for high-risk command evidence:

```yaml
approval_receipt:
  decision: approved | rejected
  approver: "user-name"
  approved_at: "2026-05-29T12:00:00Z"
  classified_commands:
    - command: "rm -rf /var/data"
      risk: high
  aggregate_risk: low | medium | high
```

**Implementation status:** Schema is defined (`schemas/completion-card.schema.json`). Strict conformance enforces it for standard/deep cards containing high-risk commands. The receipt itself is expected to be produced by a human reviewer or a trusted local script; the workflow automation around generating it is design-gated.

### 3.3 Proposed standalone approval request artifact (design-gated)

A standalone artifact enables asynchronous review without embedding everything in the completion card.

Location: `.x-harness/approvals/approval-<task_id>.yaml`

```yaml
schema_version: "1"
approval_request:
  task_id: "TASK-001"
  requested_by: "agent-name"
  requested_at: "2026-05-29T10:00:00Z"
  authority_class: "human_only"
  paths:
    - "schemas/completion-card.schema.json"
  rationale: "Schema update required to add new prediction field"
  risk_class: high
  command_evidence:
    - command: "npm run typecheck"
      risk: low
approval_decision:
  decision: approved
  approver: "user-name"
  approved_at: "2026-05-29T12:00:00Z"
  conditions:
    - "Merge only after second reviewer sign-off"
  sha256: "sha256-of-request-payload"
```

**Design gate:** The standalone approval request artifact is **not yet implemented**. If implemented, the verify gate would check:

1. `approval_artifact.path` exists.
2. `approval_artifact.sha256` matches the file content.
3. `approval_decision.decision == approved`.
4. `approval_request.task_id` matches the completion card `task_id`.

---

## 4. Request / approve / status flow

### 4.1 Proposed workflow (design-gated)

```txt
[Agent completes work]
        │
        ▼
[Agent runs verify --governance-approval-request]
        │
        ├─► If no governance triggers → normal verify flow
        │
        └─► If governance triggers:
                │
                ▼
        [Generate approval request artifact]
                │
                ▼
        [Verify outcome: BLOCKED]
                │
                ▼
        [Handoff to user / approver]
                │
                ▼
        [Human reviews and edits artifact to add approval_decision]
                │
                ▼
        [Agent re-runs verify with approved artifact]
                │
                ▼
        [Verify outcome: PASSED or WITHHELD based on remaining checks]
```

### 4.2 CLI surface (proposed, design-gated)

No new CLI commands are proposed beyond what exists. Instead, the workflow uses existing commands with documented conventions:

| Step | Command | Status |
|---|---|---|
| Detect authority boundaries | `governance check --card completion-card.yaml` | Implemented |
| Validate intervention | `intervention validate --intervention path.yaml` | Implemented |
| Verify with governance | `verify --card completion-card.yaml` | Implemented; approval enforcement is active for deep-tier human approval |
| Generate approval request | `governance request --card completion-card.yaml` | **Design-gated** |
| Query approval status | `governance status --task-id TASK-001` | **Design-gated** |

### 4.3 Status transitions

| Status | Meaning | Next action |
|---|---|---|
| `not_required` | No governance trigger matched | Continue to admission |
| `pending` | Governance trigger matched; awaiting human review | Handoff to approver (`owner: user`) |
| `approved` | Human approval granted; artifact valid | Continue to admission |
| `rejected` | Human approval denied or conditions not met | Handoff to implementation-worker to revise |

---

## 5. HITL boundaries

### 5.1 Human-only paths

Paths classified as `human_only` in `policies/authority.yaml` must not be modified by agents without explicit approval. Examples:

- `schemas/**`
- `policies/*.yaml`
- `.github/workflows/*.yml`
- `package.json`

**Current behavior:** `governance check` emits a warning (report-only). `--enforce` exits non-zero but does not block admission unless paired with `governance.requires_human_approval: true` on the card.

### 5.2 Agent-proposable, human-approved paths

Paths classified as `agent_proposable_human_approved` may be proposed by agents but require human sign-off before admission. Example:

- `policies/recovery.yaml`

**Current behavior:** Same as human-only; the distinction is semantic until the workflow is implemented.

### 5.3 Deep-tier human approval

Any `deep` task may declare `governance.requires_human_approval: true`. The verify gate blocks admission unless `approval_status == approved`.

**Current behavior:** Implemented. The golden example `examples/golden/capability/deep-approval-required/` demonstrates this.

### 5.4 High-risk command approval

The permission intent classifier (`internal/classify`) inspects `command_evidence`. High-risk commands require an `approval_receipt`.

**Current behavior:** Enforced in strict conformance (`conformance run --profile strict`). Not enforced in default `verify` unless the card already contains a receipt.

---

## 6. Owner / accountable model

### 6.1 Roles

| Role | Responsibility | Typical value |
|---|---|---|
| `owner` | The agent or user who performed the work | `agent-name`, `fixer`, `oracle` |
| `accountable` | The human who is accountable for the outcome | `user-name`, `maintainer` |
| `approver` | The human who granted governance approval | `user-name`, `security-reviewer` |

### 6.2 Separation of duties

- The **owner** must not be the **approver** for the same governance gate. The verify gate may warn (advisory) if `owner == approver`.
- The **accountable** party is the fallback approver if no explicit `approver` is recorded.
- The **verifier** is always read-only and never acts as an approver.

### 6.3 Handoff ownership

When governance approval is pending, the handoff owner should be `user` (or the named approver), not the implementation worker:

```yaml
handoff:
  next_action: "Request human approval before admission."
  owner: user
```

This is already encoded in `policies/recovery.yaml` under `approval_missing` and `classifier_approval_required`.

---

## 7. Artifact paths

### 7.1 Current artifacts (implemented)

| Artifact | Path | Purpose |
|---|---|---|
| Completion card | `completion-card.yaml` | Primary claim and evidence container |
| Approval receipt | Embedded in completion card (`approval_receipt`) | High-risk command approval |
| Authority policy | `policies/authority.yaml` | Path classification |
| Intervention artifact | `.x-harness/interventions/` | Governance override records |

### 7.2 Proposed artifacts (design-gated)

| Artifact | Path | Purpose |
|---|---|---|
| Standalone approval request | `.x-harness/approvals/approval-<task_id>.yaml` | Decoupled approval record |
| Approval index | `.x-harness/approvals/index.yaml` | Registry of pending/approved requests |
| Waiver record | `.x-harness/conformance-waivers.yaml` | Downgrade blocking checks (see `CONFORMANCE_STRICT_PROFILE.md`) |

**Design gate rationale:** The standalone artifacts require a lifecycle convention (who creates, who updates, how to garbage-collect) that is not yet defined. Until then, approval information remains embedded in the completion card.

---

## 8. Fail-closed behavior

Governance approval is fail-closed in the following situations:

1. **Deep task + `requires_human_approval: true` + missing `approval_status: approved`** → admission blocked.
2. **Strict conformance + high-risk command + missing `approval_receipt`** → conformance failed.
3. **Protected path violation + `--enforce` flag** → `governance check` exits non-zero.
4. **Tier downgrade without intervention artifact** → admission blocked.

Governance approval is **not** fail-closed in these situations (advisory only):

1. **Default `verify` without `--strict`** — high-risk commands without a receipt produce a warning, not a block.
2. **Authority boundary check without `--enforce`** — report-only warnings.
3. **Missing `governance` block on a deep task** — no implicit human approval requirement; the task proceeds to normal admission.

---

## 9. Go / TypeScript parity status

| Component | Go (`internal/`, `cmd/x-harness`) | TypeScript (`packages/cli/`) | Parity |
|---|---|---|---|
| Deep-tier approval gate | Implemented (`internal/admission/admission.go`) | Implemented (`packages/cli/src/core/admission.ts`) | Maintained |
| Approval receipt validation | Implemented in strict conformance | Implemented in strict conformance | Maintained |
| Authority boundary check | Implemented (`internal/authority/`) | Implemented (`packages/cli/src/core/authority.ts`) | Maintained |
| Governance CLI (`governance check`) | Implemented (`internal/cli/governance.go`) | Implemented (`packages/cli/src/commands/governance.ts`) | Maintained |
| Intervention validation | Implemented (`internal/cli/governance.go`) | Implemented (`packages/cli/src/commands/governance.ts`) | Maintained |
| Standalone approval request artifact | **Not implemented** | **Not implemented** | N/A |
| Approval status query command | **Not implemented** | **Not implemented** | N/A |
| Workflow automation | **Not implemented** | **Not implemented** | N/A |

The Go implementation is the native runtime. The TypeScript implementation remains the compatibility baseline. Changes to governance behavior must keep Go, TypeScript, fixtures, and policy manifests aligned. The `doctor` and parity checks validate schema/policy health and Go-vs-TypeScript drift.

---

## 10. Explicitly out of scope

The following features are **explicitly excluded** from this design and are not planned for implementation unless a future design revision specifically adds them.

| Feature | Rationale |
|---|---|
| **OAuth / SSO integration** | Identity is out-of-band. x-harness approver labels are arbitrary strings. |
| **RBAC / permission roles** | No role hierarchy. Authority classes (`human_only`, `agent_editable`) are path-based, not identity-based. |
| **Server / daemon / web UI** | File-first, CLI-only architecture. No background service for approval queues. |
| **Email / Slack / webhook notifications** | No network calls. Approval handoff is communicated via file artifacts and `handoff.next_action`. |
| **Automatic approval via policy waivers** | Waivers exist for conformance checks, not for governance approval. A waiver cannot auto-approve a deep-tier human approval gate. |
| **Blockchain / immutable ledger** | `payload_hash` and `sha256` provide local integrity. No distributed consensus. |

---

## 11. Design gates and next steps

### 11.1 Gates blocking implementation

| Gate | Condition |
|---|---|
| G1 — Demand validation | At least one real CI or local workflow shows repeated friction with manual approval tracking |
| G2 — Artifact lifecycle | Convention defined for creating, updating, and garbage-collecting `.x-harness/approvals/` files |
| G3 — CLI surface stability | Decision on whether to add `governance request` / `governance status` or keep it card-embedded |
| G4 — Parity commitment | Go and TypeScript implementations agreed to maintain parity for any new commands |

### 11.2 Next step

When demand is validated (G1), the recommended implementation order is:

1. Add `.x-harness/approvals/` artifact generation to `governance check --card` (opt-in).
2. Update verify gate to resolve and validate standalone approval artifacts.
3. Add golden examples for the full request/approve/status flow.
4. Promote this document from `Status: design-gated` to `Status: implemented`.

---

## 12. References

- `docs/ADMISSION_POLICY.md` — evidence floor and rejection conditions
- `docs/VERIFY_GATE.md` — mutation guard and governance verification semantics
- `docs/CONFORMANCE_STRICT_PROFILE.md` — strict conformance approval receipt enforcement
- `docs/ARCHITECTURE.md` — layered model and file-first constraints
- `schemas/completion-card.schema.json` — `governance`, `approval_receipt`, `approval_artifact` fields
- `policies/authority.yaml` — protected path registry and authority classes
- `policies/recovery.yaml` — recovery routing for `approval_missing` and `classifier_approval_required`
- `policies/intake.yaml` — intake classification and escalation conditions
- `templates/COMPLETION_CARD.md` — deep example with governance block
- `examples/golden/capability/deep-approval-required/` — golden example of blocked deep approval
- `internal/admission/admission.go` — Go admission engine implementation
- `internal/authority/` — Go authority boundary implementation
- `internal/classify/classify.go` — permission intent classifier
- `X_HARNESS_ADMISSION_READINESS_UPDATED_ROADMAP.md` — Section 35 (Deferred: governance approval workflow)

---

*End of design document. The governance approval workflow is design-gated. Individual components (approval receipt, deep approval gate, authority checks) are implemented as documented in their respective files. The unified workflow automation is not yet implemented.*
