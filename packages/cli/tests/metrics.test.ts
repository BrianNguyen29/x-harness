import { describe, it, expect } from "vitest";
import { computeMetrics } from "../src/core/metrics.js";
import type { AdmissionInput } from "../src/core/admission.js";

describe("metrics", () => {
  const baseInput: AdmissionInput = {
    schema_version: "1",
    task_id: "TASK-001",
    tier: "standard",
    owner: "alice",
    accountable: "bob",
    claim: {
      fix_status: "fixed",
      summary: "Fixed the issue",
      evidence: [],
    },
    verification: {
      status: "passed",
      checks: [],
    },
    admission: {
      outcome: "success",
    },
    acceptance_status: "accepted",
    handoff: {
      next_action: "none",
      owner: "alice",
    },
    staleGround: false,
  };

  describe("computeMetrics", () => {
    it("computes metrics for a successful light-tier card", () => {
      const input: AdmissionInput = {
        ...baseInput,
        tier: "light",
        claim: {
          fix_status: "fixed",
          summary: "Fixed",
          evidence: ["a.ts"],
        },
      };
      const metrics = computeMetrics(input, {
        inputCardHash: "sha256:abc",
        policyHash: "sha256:def",
        verifyRuntimeMs: 150,
      });

      expect(metrics.cost.default_context_class).toBe("low");
      expect(metrics.cost.verify_runtime_ms).toBe(150);
      expect(metrics.state_consistency.owner_present).toBe(true);
      expect(metrics.state_consistency.accountable_present).toBe(true);
      expect(metrics.state_consistency.admission_mapping_valid).toBe(true);
      expect(metrics.replayability.completion_card_present).toBe(true);
      expect(metrics.replayability.input_card_hash_present).toBe(true);
      expect(metrics.replayability.policy_hash_present).toBe(true);
    });

    it("computes metrics for a deep-tier card with full evidence", () => {
      const input: AdmissionInput = {
        ...baseInput,
        tier: "deep",
        claim: {
          fix_status: "fixed",
          summary: "Complex fix",
          evidence: [],
        },
        evidence: {
          files_changed: ["a.ts", "b.ts"],
          verification_artifacts: [
            { kind: "unit_test", status: "passed" },
            { kind: "typecheck", status: "passed" },
          ],
          untested_regions: ["edge case handling"],
          remaining_risks: ["migration compatibility"],
          rollback_policy: ["git revert"],
          execution_controls: ["timeout: 300s"],
        },
      };
      const metrics = computeMetrics(input, {
        inputCardHash: "sha256:abc",
        policyHash: "sha256:def",
        verifyRuntimeMs: 300,
      });

      expect(metrics.cost.default_context_class).toBe("high");
      expect(metrics.verification_strength.command_evidence_count).toBe(2);
      expect(metrics.verification_strength.oracle_kinds).toContain("unit_test");
      expect(metrics.verification_strength.oracle_kinds).toContain("typecheck");
      expect(metrics.verification_strength.untested_regions_count).toBe(1);
      expect(metrics.verification_strength.remaining_risks_count).toBe(1);
      expect(metrics.state_consistency.files_changed_present).toBe(true);
    });

    it("detects admission mapping invalid when success but acceptance withheld", () => {
      const input: AdmissionInput = {
        ...baseInput,
        admission: { outcome: "success" },
        acceptance_status: "withheld",
      };
      const metrics = computeMetrics(input, {});

      expect(metrics.state_consistency.admission_mapping_valid).toBe(false);
    });

    it("detects admission mapping invalid when failed but acceptance accepted", () => {
      const input: AdmissionInput = {
        ...baseInput,
        admission: { outcome: "failed" },
        acceptance_status: "accepted",
      };
      const metrics = computeMetrics(input, {});

      expect(metrics.state_consistency.admission_mapping_valid).toBe(false);
    });

    it("detects blocked recovery ability when no handoff", () => {
      const input: AdmissionInput = {
        ...baseInput,
        admission: { outcome: "blocked" },
        acceptance_status: "withheld",
        handoff: {
          next_action: "",
          owner: "",
        },
      };
      const metrics = computeMetrics(input, {});

      expect(metrics.recovery_ability.blocked_has_next_action).toBe(false);
      expect(metrics.recovery_ability.blocked_has_owner).toBe(false);
      expect(metrics.recovery_ability.recovery_route_present).toBe(false);
    });

    it("allows empty optional fields in metrics", () => {
      const input: AdmissionInput = {
        schema_version: "1",
        task_id: "TASK-001",
        tier: "standard",
        owner: "",
        accountable: "",
        claim: {
          fix_status: "fixed",
          summary: "",
          evidence: [],
        },
        verification: {
          status: "passed",
          checks: [],
        },
        admission: {
          outcome: "success",
        },
        acceptance_status: "accepted",
        handoff: {
          next_action: "none",
          owner: "",
        },
        staleGround: false,
      };
      const metrics = computeMetrics(input, {});

      expect(metrics.state_consistency.owner_present).toBe(false);
      expect(metrics.state_consistency.accountable_present).toBe(false);
    });

    it("extracts oracle kinds from verification artifacts", () => {
      const input: AdmissionInput = {
        ...baseInput,
        evidence: {
          files_changed: ["a.ts"],
          verification_artifacts: [
            { kind: "unit_test", status: "passed" },
            { kind: "lint", status: "passed" },
            { kind: "build", status: "passed" },
          ],
        },
      };
      const metrics = computeMetrics(input, {});

      expect(metrics.verification_strength.oracle_kinds).toHaveLength(3);
      expect(metrics.verification_strength.oracle_kinds).toContain("unit_test");
      expect(metrics.verification_strength.oracle_kinds).toContain("lint");
      expect(metrics.verification_strength.oracle_kinds).toContain("build");
    });

    it("handles missing evidence gracefully", () => {
      const input: AdmissionInput = {
        ...baseInput,
        evidence: undefined,
      };
      const metrics = computeMetrics(input, {});

      expect(metrics.verification_strength.command_evidence_count).toBe(0);
      expect(metrics.verification_strength.oracle_kinds).toHaveLength(0);
      expect(metrics.verification_strength.untested_regions_count).toBe(0);
      expect(metrics.verification_strength.remaining_risks_count).toBe(0);
    });
  });
});
