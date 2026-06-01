import { describe, it, expect } from "vitest";
import * as path from "node:path";
import * as fs from "node:fs";
import { fileURLToPath } from "node:url";
import * as YAML from "yaml";
import {
  runAdmission,
  acceptanceStatus,
  type AdmissionInput,
} from "../src/core/admission.js";
import { evaluateEscalation } from "../src/core/admission-evidence.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const escalationPolicyPath = path.join(repoRoot, "policies", "escalation.yaml");

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
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [
          {
            command: "npm test",
            exit_code: 0,
            runner: "local-vitest",
            started_at: "2026-05-25T00:00:00.000Z",
          },
        ],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("withholds on stale ground", () => {
    const result = runAdmission({
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
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
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(
      result.errors.some((e) => e.includes("canonical contradiction"))
    ).toBe(true);
  });

  it("fails when standard tier lacks files_changed", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("files_changed"))).toBe(true);
  });

  it("allows light tier with files_changed and manual_rationale", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
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
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
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
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("missing accountable"))).toBe(
      true
    );
  });

  it("rejects invalid tier", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "small" as "light" | "standard" | "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
      },
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
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) => e.includes("canonical contradiction") && e.includes("blocked")
      )
    ).toBe(true);
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
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some((e) => e.includes("canonical contradiction"))
    ).toBe(true);
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
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(result.notes.some((n) => n.includes("PGV"))).toBe(true);
  });

  it("blocks PGV advice that attempts to grant admission authority", () => {
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
      pgv_advice: { admission_authority: true },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors.some((e) => e.includes("PGV"))).toBe(true);
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
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors.some((e) => e.includes("non-success outcome"))).toBe(
      true
    );
  });

  it("rejects success outcome when verification did not pass", () => {
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
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "manual verification was blocked",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some((e) =>
        e.includes('success requires verification.status "passed"')
      )
    ).toBe(true);
  });

  it("rejects success outcome when acceptance is withheld", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "withheld",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) =>
          e.includes('admission.outcome is "success"') &&
          e.includes('acceptance_status is "withheld"')
      )
    ).toBe(true);
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
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.next_action"))).toBe(
      true
    );
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
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.next_action"))).toBe(
      true
    );
    expect(result.errors.some((e) => e.includes("handoff.owner"))).toBe(true);
  });

  // Backward compatibility: subagentReturn shape
  it("passes admission with valid subagentReturn inputs", () => {
    const result = runAdmission({
      claim: { id: "C1" },
      evidence: {
        id: "E1",
        owner: "alice",
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      subagentReturn: {
        result: { fix_status: "fixed" },
        verification: { status: "passed" },
      },
      tier: "standard",
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("fails on canonical contradiction via subagentReturn: passed + not_fixed", () => {
    const result = runAdmission({
      claim: { id: "C1" },
      evidence: { id: "E1", owner: "alice", files_changed: ["a.ts"] },
      subagentReturn: {
        result: { fix_status: "partial" },
        verification: { status: "passed" },
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(
      result.errors.some((e) => e.includes("canonical contradiction"))
    ).toBe(true);
  });

  it("blocks standard subagentReturn compatibility without done_checklist and prediction", () => {
    const result = runAdmission({
      subagentReturn: {
        result: { fix_status: "fixed" },
        verification: { status: "passed" },
        evidence: {
          files_changed: ["a.ts"],
          command_evidence: [{ command: "npm test", exit_code: 0 }],
        },
        handoff: { next_action: "none", owner: "alice" },
      },
      tier: "standard",
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors.some((e) => e.includes("done_checklist"))).toBe(true);
    expect(result.errors.some((e) => e.includes("prediction"))).toBe(true);
  });

  it("fails when claim.fix_status and result.fix_status disagree", () => {
    const result = runAdmission({
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      evidence: {
        id: "E1",
        owner: "alice",
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      subagentReturn: {
        result: { fix_status: "partial" },
        verification: { status: "passed" },
      },
      tier: "standard",
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(
      result.errors.some((e) =>
        e.includes('claim.fix_status is "fixed" but result.fix_status')
      )
    ).toBe(true);
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
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) => e.includes("deep") && e.includes("verification_artifacts")
      )
    ).toBe(true);
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          { kind: "unit_test", command: "npm test", status: "passed" },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
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

  // context_acknowledged advisory tests
  it("advises but does not block when context_acknowledged is missing", () => {
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
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some(
        (n) => n.includes("context_acknowledged") && n.includes("advisory-only")
      )
    ).toBe(true);
  });

  it("advises but does not block when context_acknowledged is false", () => {
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
      context_acknowledged: false,
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some(
        (n) => n.includes("context_acknowledged") && n.includes("advisory-only")
      )
    ).toBe(true);
  });

  it("does not add context_acknowledged advisory when true", () => {
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
      context_acknowledged: true,
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(result.notes.some((n) => n.includes("context_acknowledged"))).toBe(
      false
    );
  });

  it("advises standard tier when artifact metadata is sparse", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [{ kind: "unit_test", status: "passed" }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(result.notes.some((n) => n.includes("artifact metadata"))).toBe(
      true
    );
  });

  it("advises deep tier when artifact metadata is sparse", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(result.notes.some((n) => n.includes("artifact metadata"))).toBe(
      true
    );
  });

  it("does not add artifact metadata advisory when quality is present", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            exit_code: 0,
            started_at: "2026-05-22T10:00:00Z",
          },
        ],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(result.notes.some((n) => n.includes("artifact metadata"))).toBe(
      false
    );
  });

  // Batch A contract fixes tests

  it("rejects light tier without command_evidence or manual_rationale", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) =>
          e.includes("light") &&
          (e.includes("command_evidence") || e.includes("manual_rationale"))
      )
    ).toBe(true);
  });

  it("accepts light tier with command_evidence", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("accepts light tier with manual_rationale", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("rejects standard tier without command_evidence", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) => e.includes("standard") && e.includes("command_evidence")
      )
    ).toBe(true);
  });

  it("rejects timeout without handoff.next_action", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "timeout", checks: [] },
      handoff: { next_action: "", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple fix",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.next_action"))).toBe(
      true
    );
  });

  it("rejects timeout without handoff.owner", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "timeout", checks: [] },
      handoff: { next_action: "review", owner: "" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple fix",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.owner"))).toBe(true);
  });

  it("rejects error without handoff.next_action", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "error", checks: [] },
      handoff: { next_action: "", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple fix",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.next_action"))).toBe(
      true
    );
  });

  it("rejects admission outcome timeout without handoff", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "timeout" },
      handoff: { next_action: "", owner: "" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple fix",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.next_action"))).toBe(
      true
    );
  });

  it("rejects admission outcome error without handoff", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "error" },
      handoff: { next_action: "", owner: "" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple fix",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("handoff.owner"))).toBe(true);
  });

  // Deep tier command_evidence enforcement
  it("rejects deep tier without command_evidence", () => {
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
        files_changed: ["a.ts"],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) => e.includes("deep") && e.includes("command_evidence")
      )
    ).toBe(true);
  });

  it("accepts deep tier with command_evidence", () => {
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  // Deep tier missing state tests
  it("rejects deep tier missing state and uses state_read_write_missing predicate", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      // No state provided
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("state.write_set"))).toBe(true);
    expect(result.errors.some((e) => e.includes("state.read_set"))).toBe(true);
    expect(result.blocking_predicate).toBe("state_read_write_missing");
  });

  it("rejects deep tier with partial state and uses state_read_write_missing predicate", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"] }, // missing write_set
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("state.write_set"))).toBe(true);
    expect(result.blocking_predicate).toBe("state_read_write_missing");
  });

  // Done checklist and prediction tests for standard tier
  it("blocks standard tier missing done_checklist", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("done_checklist"))).toBe(true);
    expect(result.blocking_predicate).toBe("done_checklist_missing");
  });

  it("blocks standard tier missing prediction", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: false,
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("prediction"))).toBe(true);
    expect(result.blocking_predicate).toBe("prediction_missing");
  });

  it("blocks standard tier with weak prediction missing required fields", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        // Missing required fields: claim, expected_effect, falsification_method, horizon
        measurable_signal: "some metric",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("prediction.claim"))).toBe(
      true
    );
    expect(
      result.errors.some((e) => e.includes("prediction.expected_effect"))
    ).toBe(true);
    expect(
      result.errors.some((e) => e.includes("prediction.falsification_method"))
    ).toBe(true);
    expect(result.errors.some((e) => e.includes("prediction.horizon"))).toBe(
      true
    );
    expect(result.blocking_predicate).toBe("prediction_invalid");
  });

  it("accepts standard tier with valid done_checklist and prediction", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests and verify pass",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  // Done checklist and prediction tests for deep tier
  it("blocks deep tier missing done_checklist", () => {
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("done_checklist"))).toBe(true);
    expect(result.blocking_predicate).toBe("done_checklist_missing");
  });

  it("blocks deep tier missing prediction", () => {
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: false,
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("prediction"))).toBe(true);
    expect(result.blocking_predicate).toBe("prediction_missing");
  });

  // Cross-check test: done_checklist.prediction_declared=true but prediction missing
  it("blocks when done_checklist.prediction_declared is true but prediction is missing", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true, // Says prediction is declared but...
      },
      // prediction is missing
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) =>
          e.includes("done_checklist.prediction_declared") &&
          e.includes("prediction is missing")
      )
    ).toBe(true);
    expect(result.blocking_predicate).toBe(
      "done_checklist_prediction_mismatch"
    );
  });

  it("blocks when done_checklist claims evidence is missing but evidence is present", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: false,
        evidence_attached: false,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors).toContain(
      "done_checklist.evidence_attached is false but evidence is present"
    );
    expect(result.blocking_predicate).toBe("done_checklist_mismatch");
  });

  it("blocks when checklist declares read/write sets but present state is incomplete", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"] },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors).toContain(
      "done_checklist.read_write_sets_declared is true but state.write_set is missing"
    );
    expect(result.blocking_predicate).toBe("done_checklist_mismatch");
  });

  it("blocks strict mode when checklist declares read/write sets but state is missing", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      strict: true,
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [
          {
            command: "npm test",
            exit_code: 0,
            runner: "local-vitest",
            started_at: "2026-05-25T00:00:00.000Z",
          },
        ],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors).toContain(
      "done_checklist.read_write_sets_declared is true but state is missing"
    );
    expect(result.blocking_predicate).toBe("done_checklist_mismatch");
  });

  it("blocks when checklist declares scoped artifacts but artifacts lack scope", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      strict: true,
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [
          {
            command: "npm test",
            exit_code: 0,
            runner: "local-vitest",
            started_at: "2026-05-25T00:00:00.000Z",
          },
        ],
        verification_artifacts: [
          {
            kind: "unit_test",
            status: "passed",
            command: "npm test",
            exit_code: 0,
            runner: "local-vitest",
            started_at: "2026-05-25T00:00:00.000Z",
          },
        ],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: false,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors).toContain(
      "done_checklist.scope_explained is true but verification_artifacts lacks verifies/does_not_verify scope"
    );
    expect(result.errors).toContain(
      "done_checklist.coverage_gap_declared is true but no untested_regions or artifact does_not_verify scope is present"
    );
    expect(result.blocking_predicate).toBe("done_checklist_mismatch");
  });

  it("cross-checks optional done_checklist honesty on light tier", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "manual smoke check",
      },
      done_checklist: {
        evidence_attached: false,
        prediction_declared: false,
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors).toContain(
      "done_checklist.evidence_attached is false but evidence is present"
    );
    expect(result.blocking_predicate).toBe("done_checklist_mismatch");
  });

  it("blocks intake tier downgrade without intervention approval", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["src/auth/session.ts"],
        manual_rationale: "Fixture for downgrade guard",
      },
      intake: {
        classification: "high_risk",
        mapped_tier: "deep",
        rationale: "Auth/session work must route to deep",
        signals: ["auth", "session"],
        auto_escalated: true,
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("Fintervention");
    expect(result.errors.some((e) => e.includes("tier downgrade"))).toBe(true);
  });

  it("allows intake tier downgrade with approved governance intervention", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["src/auth/session.ts"],
        manual_rationale: "Fixture for approved downgrade",
      },
      governance: {
        requires_human_approval: true,
        approval_required_for: ["tier_downgrade"],
        approval_status: "approved",
        approver: "maintainer",
      },
      intake: {
        classification: "high_risk",
        mapped_tier: "deep",
        rationale: "Auth/session work must route to deep",
        signals: ["auth", "session"],
        auto_escalated: true,
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some((note) => note.includes("tier downgrade approved"))
    ).toBe(true);
  });

  // Approval receipt tests
  it("withholds standard tier with high-risk command missing approval receipt", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["scripts/clean.sh"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(
      result.errors.some(
        (e) => e.includes("approval receipt") && e.includes("high-risk")
      )
    ).toBe(true);
    expect(result.blocking_predicate).toBe("classifier_approval_required");
  });

  it("accepts standard tier with high-risk command and valid approval receipt", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      approval_receipt: {
        decision: "approved",
        approver: "user",
        classified_commands: [{ command: "rm -rf dist", risk: "high" }],
        aggregate_risk: "high",
      },
      evidence: {
        files_changed: ["scripts/clean.sh"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some((n) => n.includes("approval_receipt validated"))
    ).toBe(true);
  });

  it("accepts standard tier with low-risk command without approval receipt", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("withholds deep tier with medium-risk command missing approval receipt", () => {
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
        files_changed: ["a.ts"],
        command_evidence: [{ command: "go build ./cmd/app", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors.some((e) => e.includes("approval receipt"))).toBe(
      true
    );
    expect(result.blocking_predicate).toBe("classifier_approval_required");
  });

  it("accepts deep tier with medium-risk command and valid approval receipt", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      approval_receipt: {
        decision: "approved",
        approver: "user",
        classified_commands: [
          { command: "go build ./cmd/app", risk: "medium" },
        ],
        aggregate_risk: "medium",
      },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "go build ./cmd/app", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("withholds when approval receipt decision is not approved", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      approval_receipt: {
        decision: "denied",
        approver: "user",
        classified_commands: [{ command: "rm -rf dist", risk: "high" }],
        aggregate_risk: "high",
      },
      evidence: {
        files_changed: ["scripts/clean.sh"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) => e.includes("decision is") && e.includes("denied")
      )
    ).toBe(true);
  });

  it("withholds when approval receipt approver is missing", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      approval_receipt: {
        decision: "approved",
        approver: "",
        classified_commands: [{ command: "rm -rf dist", risk: "high" }],
        aggregate_risk: "high",
      },
      evidence: {
        files_changed: ["scripts/clean.sh"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("approver is required"))).toBe(
      true
    );
  });

  it("withholds when approval receipt classified_commands is empty", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      approval_receipt: {
        decision: "approved",
        approver: "user",
        classified_commands: [],
        aggregate_risk: "high",
      },
      evidence: {
        files_changed: ["scripts/clean.sh"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some((e) => e.includes("classified_commands is required"))
    ).toBe(true);
  });

  it("withholds when approval receipt does not cover all high-risk commands", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      approval_receipt: {
        decision: "approved",
        approver: "user",
        classified_commands: [{ command: "rm -rf other", risk: "high" }],
        aggregate_risk: "high",
      },
      evidence: {
        files_changed: ["scripts/clean.sh"],
        command_evidence: [
          { command: "rm -rf dist", exit_code: 0 },
          { command: "rm -rf other", exit_code: 0 },
        ],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) => e.includes("does not cover command") && e.includes("rm -rf dist")
      )
    ).toBe(true);
  });

  it("withholds when approval receipt aggregate_risk is below threshold", () => {
    const result = runAdmission({
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      approval_receipt: {
        decision: "approved",
        approver: "user",
        classified_commands: [{ command: "rm -rf dist", risk: "high" }],
        aggregate_risk: "low",
      },
      evidence: {
        files_changed: ["scripts/clean.sh"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some(
        (e) =>
          e.includes("aggregate_risk") && e.includes("below required threshold")
      )
    ).toBe(true);
  });

  // Tier guard tests
  it("blocks light tier with high-risk file path", () => {
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
      evidence: {
        files_changed: ["schemas/completion-card.schema.json"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(
      result.errors.some((e) =>
        e.includes(
          "tier guard: light tier declared but high-risk files detected"
        )
      )
    ).toBe(true);
  });

  it("warns light tier with high-risk command but does not block", () => {
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
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
    });
    expect(result.outcome).toBe("success");
    expect(
      result.notes.some((n) =>
        n.includes("tier guard warning: light tier with high-risk command(s)")
      )
    ).toBe(true);
  });

  it("warns standard tier with both high-risk files and high-risk commands", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["internal/admission/roles.ts"],
        command_evidence: [{ command: "rm -rf dist", exit_code: 0 }],
      },
      approval_receipt: {
        decision: "approved",
        approver: "user",
        classified_commands: [{ command: "rm -rf dist", risk: "high" }],
        aggregate_risk: "high",
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(
      result.notes.some((n) =>
        n.includes(
          "tier guard warning: standard tier with both high-risk files"
        )
      )
    ).toBe(true);
  });

  it("allows deep tier with high-risk file path", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        files_changed: ["internal/authority/roles.yaml"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  // Verify-stage v1 auto-escalation guard tests. Wording and predicate
  // must stay parity-safe with the Go implementation in
  // internal/admission/escalation.go and the policy in
  // policies/escalation.yaml.
  it("blocks light tier with policy path under v1 escalation", () => {
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
      evidence: {
        files_changed: ["policies/admission.yaml"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("tier_escalation_required");
    expect(
      result.errors.some((e) => e.includes("tier escalation required"))
    ).toBe(true);
  });

  it("blocks standard tier with auth path under v1 escalation", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["src/auth/session.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("tier_escalation_required");
  });

  it("allows deep tier with schema path under v1 escalation", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        files_changed: ["schemas/completion-card.schema.json"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("does not escalate safe docs-only light cards", () => {
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
      evidence: {
        files_changed: ["docs/readme.md"],
        manual_rationale: "doc tweak",
      },
    });
    expect(result.outcome).toBe("success");
    expect(
      result.errors.some((e) => e.includes("tier escalation required"))
    ).toBe(false);
  });

  it("bypasses v1 escalation when governance intervention is approved", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["policies/admission.yaml"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      governance: {
        requires_human_approval: true,
        approval_status: "approved",
        approver: "maintainer",
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(
      result.notes.some((n) =>
        n.includes(
          "tier escalation bypassed by approved governance intervention"
        )
      )
    ).toBe(true);
  });

  it("blocks standard tier with .github/workflows path under v1 escalation", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: [".github/workflows/x-harness-verify.yml"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("tier_escalation_required");
  });

  it("blocks light tier with migrations path under v1 escalation", () => {
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
      evidence: {
        files_changed: ["migrations/0001_init.sql"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("tier_escalation_required");
  });

  // Context floor tests
  it("blocks standard tier when context_floor is enabled and context_alignment is missing", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: true,
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("context_alignment"))).toBe(
      true
    );
    expect(result.blocking_predicate).toBe("context_floor_blocked");
  });

  it("blocks deep tier when context_floor is enabled and context_pack_id is missing", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: true,
      context_alignment: {
        stale_ground_checked: true,
        product_contract_refs: ["docs/product.md"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors.some((e) => e.includes("context_pack_id"))).toBe(true);
    expect(result.blocking_predicate).toBe("context_floor_blocked");
  });

  it("passes standard tier with valid context_alignment when context_floor is enabled", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: true,
      context_alignment: {
        stale_ground_checked: true,
        product_contract_refs: ["README.md"],
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("blocks standard tier when a referenced file is missing under context_floor", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: true,
      context_alignment: {
        stale_ground_checked: true,
        product_contract_refs: ["nonexistent/path/contract.md"],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("context_floor_blocked");
    expect(result.errors).toContain(
      "referenced file does not exist: nonexistent/path/contract.md"
    );
  });

  it("strips #anchor suffix before checking referenced file existence", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: true,
      context_alignment: {
        stale_ground_checked: true,
        product_contract_refs: ["README.md#section"],
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
  });

  it("blocks standard tier when a context_evidence ref is missing under context_floor", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: true,
      context_alignment: {
        stale_ground_checked: true,
        product_contract_refs: ["README.md"],
        context_evidence: [
          { ref: "nonexistent/evidence.md", kind: "contract_oracle" },
        ],
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("context_floor_blocked");
    expect(result.errors).toContain(
      "context_evidence ref file does not exist: nonexistent/evidence.md"
    );
  });

  it("blocks deep tier when a referenced file is missing under context_floor", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: true,
      context_alignment: {
        stale_ground_checked: true,
        product_contract_refs: ["nonexistent/deep/contract.md"],
        context_pack_id: "deep-pack-1",
      },
    });
    expect(result.outcome).toBe("failed");
    expect(result.blocking_predicate).toBe("context_floor_blocked");
    expect(result.errors).toContain(
      "referenced file does not exist: nonexistent/deep/contract.md"
    );
  });

  it("advises but does not block light tier when context_floor is enabled", () => {
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
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
      contextFloor: true,
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some((n) =>
        n.includes("context floor advisory only for light tier")
      )
    ).toBe(true);
  });

  it("does not evaluate context floor when contextFloor is false", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
      contextFloor: false,
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(result.notes.some((n) => n.includes("context floor"))).toBe(false);
  });

  // product_intent.status advisory tests (advisory-only; never blocks admission)
  it("advises standard tier when product_intent.status is missing", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some(
        (n) =>
          n.includes("product_intent.status not declared") &&
          n.includes("advisory-only")
      )
    ).toBe(true);
  });

  it("advises standard tier when product_intent.status is unknown", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      product_intent: { status: "unknown" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some(
        (n) =>
          n.includes("product_intent.status is unknown") &&
          n.includes("advisory-only")
      )
    ).toBe(true);
  });

  it("does not advise standard tier when product_intent.status is aligned", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      product_intent: { status: "aligned" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.notes.some((n) => n.includes("product_intent.status"))).toBe(
      false
    );
  });

  it("does not advise standard tier for unreviewed/disputed/not_applicable", () => {
    for (const status of ["unreviewed", "disputed", "not_applicable"]) {
      const result = runAdmission({
        schema_version: "1",
        task_id: "T1",
        tier: "standard",
        owner: "alice",
        accountable: "bob",
        claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
        verification: { status: "passed", checks: [] },
        admission: { outcome: "success" },
        acceptance_status: "accepted",
        handoff: { next_action: "none", owner: "alice" },
        product_intent: { status },
        evidence: {
          files_changed: ["a.ts"],
          command_evidence: [{ command: "npm test", exit_code: 0 }],
        },
        done_checklist: {
          source_of_truth_read: true,
          scope_explained: true,
          read_write_sets_declared: true,
          evidence_attached: true,
          coverage_gap_declared: true,
          risk_and_rollback_declared: true,
          prediction_declared: true,
        },
        prediction: {
          claim: "Task completes successfully",
          expected_effect: "Tests pass",
          falsification_method: "Run tests",
          horizon: "same_verify",
        },
      });
      expect(result.outcome).toBe("success");
      expect(
        result.notes.some((n) => n.includes("product_intent.status"))
      ).toBe(false);
    }
  });

  it("advises deep tier when product_intent.status is missing", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some(
        (n) =>
          n.includes("product_intent.status not declared") &&
          n.includes("advisory-only")
      )
    ).toBe(true);
  });

  it("advises deep tier when product_intent.status is unknown", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "deep",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      handoff: { next_action: "none", owner: "alice" },
      state: { read_set: ["a.ts"], write_set: ["a.ts"] },
      product_intent: { status: "unknown" },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
        verification_artifacts: [
          {
            kind: "unit_test",
            command: "npm test",
            status: "passed",
            verifies: ["x"],
            does_not_verify: ["y"],
          },
        ],
        untested_regions: ["no e2e"],
        remaining_risks: ["prod untested"],
        rollback_policy: ["revert commit"],
        execution_controls: ["feature flag"],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.acceptance_status).toBe("accepted");
    expect(
      result.notes.some(
        (n) =>
          n.includes("product_intent.status is unknown") &&
          n.includes("advisory-only")
      )
    ).toBe(true);
  });

  it("does not advise light tier for missing product_intent.status", () => {
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
      evidence: {
        files_changed: ["a.ts"],
        manual_rationale: "simple doc fix",
      },
    });
    expect(result.outcome).toBe("success");
    expect(result.notes.some((n) => n.includes("product_intent.status"))).toBe(
      false
    );
  });

  it("treats empty/whitespace product_intent.status as missing on standard", () => {
    const result = runAdmission({
      schema_version: "1",
      task_id: "T1",
      tier: "standard",
      owner: "alice",
      accountable: "bob",
      claim: { fix_status: "fixed", summary: "done", evidence: ["e1"] },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "none", owner: "alice" },
      product_intent: { status: "   " },
      evidence: {
        files_changed: ["a.ts"],
        command_evidence: [{ command: "npm test", exit_code: 0 }],
      },
      done_checklist: {
        source_of_truth_read: true,
        scope_explained: true,
        read_write_sets_declared: true,
        evidence_attached: true,
        coverage_gap_declared: true,
        risk_and_rollback_declared: true,
        prediction_declared: true,
      },
      prediction: {
        claim: "Task completes successfully",
        expected_effect: "Tests pass",
        falsification_method: "Run tests",
        horizon: "same_verify",
      },
    });
    expect(result.outcome).toBe("success");
    expect(
      result.notes.some((n) => n.includes("product_intent.status not declared"))
    ).toBe(true);
  });

  // Verify-stage v1 auto-escalation drift guard. Loads the canonical
  // policies/escalation.yaml file and behaviorally checks that each
  // declared high_risk_path_pattern triggers the TypeScript
  // evaluateEscalation() guard when declared against a light-tier
  // completion card, and that the policy scalar fields (required_tier,
  // blocked_predicate, bypass) match the expected values. Parity-safe
  // with internal/admission/admission_test.go::TestEscalationDriftGuard.
  describe("verify-stage escalation drift guard", () => {
    function loadEscalationPolicy(): Record<string, unknown> {
      const raw = fs.readFileSync(escalationPolicyPath, "utf8");
      return YAML.parse(raw) as Record<string, unknown>;
    }

    function asRecord(value: unknown): Record<string, unknown> | undefined {
      if (value && typeof value === "object" && !Array.isArray(value)) {
        return value as Record<string, unknown>;
      }
      return undefined;
    }

    function buildLightCardWithFile(filePath: string): AdmissionInput {
      return {
        schema_version: "1",
        task_id: "T-DRIFT",
        tier: "light",
        owner: "alice",
        accountable: "bob",
        claim: { fix_status: "fixed", summary: "s", evidence: ["e"] },
        verification: { status: "passed", checks: [] },
        admission: { outcome: "success" },
        acceptance_status: "accepted",
        handoff: { next_action: "n", owner: "o" },
        evidence: {
          files_changed: [filePath],
          command_evidence: [{ command: "npm test", exit_code: 0 }],
        },
      };
    }

    it("matches policy declared scalar fields and bypass", () => {
      const policy = loadEscalationPolicy();
      const verifyStage = asRecord(
        asRecord(policy["verify_stage_escalation"])?.["v1"]
      );
      expect(verifyStage).toBeDefined();
      expect(verifyStage?.["required_tier"]).toBe("deep");
      expect(verifyStage?.["blocked_predicate"]).toBe(
        "tier_escalation_required"
      );
      const bypass = verifyStage?.["bypass"];
      expect(Array.isArray(bypass)).toBe(true);
      expect((bypass as unknown[]).includes("approved_tier_downgrade")).toBe(
        true
      );
    });

    it("triggers evaluateEscalation for every declared pattern", () => {
      const policy = loadEscalationPolicy();
      const verifyStage = asRecord(
        asRecord(policy["verify_stage_escalation"])?.["v1"]
      );
      const patterns = verifyStage?.["high_risk_path_patterns"];
      expect(Array.isArray(patterns)).toBe(true);
      const list = patterns as unknown[];
      expect(list.length).toBeGreaterThan(0);

      for (const raw of list) {
        if (typeof raw !== "string") continue;
        const pattern = raw;
        const lower = pattern.toLowerCase();
        const sample = lower.endsWith("/")
          ? `${pattern}drift-guard.ts`
          : `${pattern}/drift-guard.ts`;
        const card = buildLightCardWithFile(sample);
        const result = evaluateEscalation(card);
        const hasEscalationError = result.errors.some(
          (e) => e.predicate === "tier_escalation_required"
        );
        expect(hasEscalationError).toBe(true);
      }
    });

    it("does not trigger evaluateEscalation for safe docs path", () => {
      const card = buildLightCardWithFile("docs/readme.md");
      const result = evaluateEscalation(card);
      expect(
        result.errors.some((e) => e.predicate === "tier_escalation_required")
      ).toBe(false);
    });
  });
});
