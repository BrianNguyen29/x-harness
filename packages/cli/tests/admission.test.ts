import { describe, it, expect } from "vitest";
import { runAdmission, acceptanceStatus } from "../src/core/admission.js";

describe("admission", () => {
  it("accepts success outcome", () => {
    expect(acceptanceStatus("success")).toBe("accepted");
  });

  it("withholds all non-success outcomes", () => {
    expect(acceptanceStatus("failed")).toBe("withheld");
    expect(acceptanceStatus("blocked")).toBe("withheld");
    expect(acceptanceStatus("skipped")).toBe("withheld");
    expect(acceptanceStatus("timeout")).toBe("withheld");
    expect(acceptanceStatus("error")).toBe("withheld");
  });

  it("passes admission with valid completion card inputs", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: {
        fix_status: "fixed",
        summary: "done",
        evidence: ["e1"],
      },
      verification: {
        status: "passed",
        checks: ["c1"],
      },
      admission: {
        outcome: "success",
      },
      acceptance_status: "accepted",
      handoff: {
        next_action: "none",
        owner: "alice",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("withholds on stale ground", () => {
    const result = runAdmission({
      claim: { fix_status: "fixed", summary: "done", evidence: [] },
      staleGround: true,
    });
    expect(result.outcome).toBe("blocked");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors[0]).toContain("stale_ground");
  });

  it("fails on canonical contradiction: passed + not_fixed", () => {
    const result = runAdmission({
      task_id: "T1",
      owner: "alice",
      accountable: "bob",
      claim: {
        fix_status: "partial",
        summary: "partial",
        evidence: ["e1"],
      },
      verification: {
        status: "passed",
        checks: [],
      },
      handoff: { next_action: "review", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors[0]).toContain("canonical contradiction");
  });

  it("fails when standard tier lacks evidence", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: [] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors[0]).toContain("requires evidence");
  });

  it("allows light tier without evidence", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: [] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
    });
    expect(result.outcome).toBe("success");
  });

  // New tests for Batch 1 contract
  it("rejects missing owner", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("missing owner"))).toBe(true);
  });

  it("rejects missing accountable", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("missing accountable"))).toBe(true);
  });

  it("rejects invalid tier", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "small" as any,
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("invalid tier"))).toBe(true);
  });

  it("rejects verification blocked + accepted", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "blocked", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "review", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("canonical contradiction") && e.includes("blocked"))).toBe(true);
  });

  it("rejects fix_status partial + verification passed", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "partial", summary: "partial", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "review", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("canonical contradiction"))).toBe(true);
  });

  it("PGV high risk alone does not block if core admission succeeds", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      pgv_risk: "HIGH",
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(result.notes.some((n) => n.includes("PGV"))).toBe(true);
  });

  it("non-success outcome always withheld", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "failed" },
      acceptance_status: "accepted",
      handoff: { next_action: "review", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors.some((e) => e.includes("non-success outcome"))).toBe(true);
  });

  it("rejects blocked without next_action/owner", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "blocked", checks: [] },
      admission: { outcome: "blocked" },
      handoff: { next_action: "", owner: "" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.next_action"))).toBe(true);
    expect(result.errors.some((e) => e.includes("handoff.owner"))).toBe(true);
  });

  it("rejects verification failed without handoff owner/action", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "not_fixed", summary: "failed", evidence: ["e1"] },
      verification: { status: "failed", checks: [] },
      handoff: { next_action: "", owner: "" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.next_action"))).toBe(true);
    expect(result.errors.some((e) => e.includes("handoff.owner"))).toBe(true);
  });

  // Backward compatibility: subagentReturn shape
  it("passes admission with valid subagentReturn inputs", () => {
    const result = runAdmission({
      claim: { id: "C1" },
      evidence: { id: "E1", owner: "alice" },
      subagentReturn: {
        result: { fix_status: "fixed" },
        verification: { status: "passed" },
      },
      tier: "standard",
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("fails on canonical contradiction via subagentReturn: passed + not_fixed", () => {
    const result = runAdmission({
      subagentReturn: {
        result: { fix_status: "partial" },
        verification: { status: "passed" },
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors[0]).toContain("canonical contradiction");
  });

  // Evidence floor tests for deep tier
  it("blocks deep tier missing verification_artifacts", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("deep") && e.includes("verification_artifacts"))).toBe(true);
    expect(result.blocking_predicate).toBe("evidence_scope_missing");
  });

  it("blocks deep tier missing evidence scope", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        verification_artifacts: [
          { kind: "unit_test", command: "npm test", status: "passed" },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("evidence scope"))).toBe(true);
  });

  it("blocks deep tier missing state.read_set/write_set", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        verification_artifacts: [
          { kind: "unit_test", command: "npm test", status: "passed", verifies: ["x"], does_not_verify: ["y"] },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("state.write_set"))).toBe(true);
    expect(result.errors.some((e) => e.includes("state.read_set"))).toBe(true);
  });

  it("accepts deep tier with full evidence floor", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        verification_artifacts: [
          { kind: "unit_test", command: "npm test", status: "passed", verifies: ["x"], does_not_verify: ["y"] },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  // Governance approval tests
  it("blocks deep tier with pending human approval", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        verification_artifacts: [
          { kind: "unit_test", command: "npm test", status: "passed", verifies: ["x"], does_not_verify: ["y"] },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
      },
      governance: {
        risk_class: "high",
        requires_human_approval: true,
        approval_required_for: ["auth change"],
        approval_status: "pending",
        approver: "user",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("human approval"))).toBe(true);
    expect(result.blocking_predicate).toBe("approval_missing");
  });

  it("accepts deep tier with approved human approval", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        verification_artifacts: [
          { kind: "unit_test", command: "npm test", status: "passed", verifies: ["x"], does_not_verify: ["y"] },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
      },
      governance: {
        risk_class: "high",
        requires_human_approval: true,
        approval_required_for: ["auth change"],
        approval_status: "approved",
        approver: "user",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });
});
