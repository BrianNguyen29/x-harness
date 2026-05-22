import { describe, it, expect } from "vitest";
import { validate as validateClaim } from "../src/validators/claim.js";
import { validate as validateEvidence } from "../src/validators/evidence.js";
import { validate as validateCompletionCard } from "../src/validators/completionCard.js";
import { validate as validateSubagentReturn } from "../src/validators/subagentReturn.js";

describe("schema validators", () => {
  describe("claim schema", () => {
    it("accepts minimal valid claim", async () => {
      const result = await validateClaim({ fix_status: "fixed" });
      expect(result.valid).toBe(true);
    });

    it("accepts claim with all optional fields", async () => {
      const result = await validateClaim({
        id: "C1",
        fix_status: "fixed",
        summary: "Fixed the bug",
        evidence: ["file.ts"],
      });
      expect(result.valid).toBe(true);
    });

    it("rejects invalid fix_status", async () => {
      const result = await validateClaim({ fix_status: "done" });
      expect(result.valid).toBe(false);
    });

    it("rejects missing fix_status", async () => {
      const result = await validateClaim({});
      expect(result.valid).toBe(false);
    });
  });

  describe("evidence schema", () => {
    it("accepts minimal valid evidence", async () => {
      const result = await validateEvidence({
        files_changed: ["src/index.ts"],
      });
      expect(result.valid).toBe(true);
    });

    it("accepts evidence with all optional fields", async () => {
      const result = await validateEvidence({
        id: "E1",
        files_changed: ["src/index.ts"],
        verification_artifacts: [],
        untested_regions: [],
        remaining_risks: [],
        rollback_policy: [],
        execution_controls: [],
      });
      expect(result.valid).toBe(true);
    });

    it("rejects evidence without files_changed", async () => {
      const result = await validateEvidence({});
      expect(result.valid).toBe(false);
    });

    it("rejects files_changed with empty array", async () => {
      const result = await validateEvidence({ files_changed: [] });
      expect(result.valid).toBe(false);
    });

    it("rejects files_changed with non-string items", async () => {
      const result = await validateEvidence({ files_changed: [123] });
      expect(result.valid).toBe(false);
    });
  });

  describe("subagent-return schema", () => {
    it("accepts minimal valid subagent return", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "done", fix_status: "fixed", key_findings: [] },
        evidence: {},
        verification: { status: "passed" },
        confidence: "HIGH",
        handoff: { next_action: "none", owner: "alice" },
      });
      expect(result.valid).toBe(true);
    });

    it("accepts subagent return with timeout status", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "timeout", fix_status: "partial", key_findings: [] },
        evidence: {},
        verification: { status: "timeout" },
        confidence: "MED",
        handoff: { next_action: "retry", owner: "alice" },
      });
      expect(result.valid).toBe(true);
    });

    it("accepts subagent return with error status", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "error", fix_status: "not_fixed", key_findings: [] },
        evidence: {},
        verification: { status: "error" },
        confidence: "LOW",
        handoff: { next_action: "investigate", owner: "bob" },
      });
      expect(result.valid).toBe(true);
    });

    it("rejects invalid verification status", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "done", fix_status: "fixed", key_findings: [] },
        evidence: {},
        verification: { status: "unknown" },
        confidence: "HIGH",
        handoff: { next_action: "none", owner: "alice" },
      });
      expect(result.valid).toBe(false);
    });

    it("rejects invalid confidence", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "done", fix_status: "fixed", key_findings: [] },
        evidence: {},
        verification: { status: "passed" },
        confidence: "MEDIUM",
        handoff: { next_action: "none", owner: "alice" },
      });
      expect(result.valid).toBe(false);
    });
  });

  describe("completion-card schema", () => {
    const minimalCard = {
      schema_version: "1",
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: {
        fix_status: "fixed",
        summary: "done",
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
    };

    it("accepts minimal valid completion card", async () => {
      const result = await validateCompletionCard(minimalCard);
      expect(result.valid).toBe(true);
    });

    it("accepts card with all optional fields", async () => {
      const card = {
        ...minimalCard,
        id: "card-1",
        state: {
          read_set: ["a.ts"],
          write_set: ["b.ts"],
          assumptions: [],
          conflict_policy: {},
        },
        evidence: {
          files_changed: ["a.ts"],
          verification_artifacts: [],
          untested_regions: [],
          remaining_risks: [],
          rollback_policy: [],
          execution_controls: [],
        },
        governance: {
          risk_class: "low",
          requires_human_approval: false,
          approval_required_for: [],
          approval_status: "not_required",
          approver: undefined,
        },
        context_acknowledged: true,
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(true);
    });

    it("rejects card with invalid tier", async () => {
      const card = { ...minimalCard, tier: "small" };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects card with invalid acceptance_status", async () => {
      const card = { ...minimalCard, acceptance_status: "pending" };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects accepted-without-success admission", async () => {
      const card = {
        ...minimalCard,
        admission: { outcome: "failed" },
        acceptance_status: "accepted",
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });
  });
});
