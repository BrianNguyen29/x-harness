import { describe, it, expect } from "vitest";
import { validate as validateClaim } from "../src/validators/claim.js";
import { validate as validateEvidence } from "../src/validators/evidence.js";
import { validate as validateSubagentReturn } from "../src/validators/subagentReturn.js";
import { validate as validateVerifyEvent } from "../src/validators/verifyEvent.js";

describe("validators", () => {
  it("validates a valid claim", async () => {
    const result = await validateClaim({ id: "C1", fix_status: "fixed" });
    console.log("claim result:", result);
    expect(result.valid).toBe(true);
  });

  it("rejects an invalid claim (not an object)", async () => {
    const result = await validateClaim("not-an-object");
    expect(result.valid).toBe(false);
    expect(result.errors.length).toBeGreaterThan(0);
  });

  it("validates a valid evidence packet", async () => {
    const result = await validateEvidence({ id: "E1", evidence_quality: "sufficient" });
    console.log("evidence result:", result);
    expect(result.valid).toBe(true);
  });

  it("validates a valid subagent return", async () => {
    const result = await validateSubagentReturn({
      result: { summary: "done", fix_status: "fixed", key_findings: [] },
      evidence: {},
      verification: { status: "passed" },
      confidence: "HIGH",
      handoff: { next_action: "none" },
    });
    console.log("subagent result:", result);
    expect(result.valid).toBe(true);
  });

  it("rejects a subagent return missing required fields", async () => {
    const result = await validateSubagentReturn({ result: {} });
    expect(result.valid).toBe(false);
    expect(result.errors.length).toBeGreaterThan(0);
  });

  it("validates a verify event", async () => {
    const result = await validateVerifyEvent({
      event_id: "VE-1",
      event_type: "verify_completed",
      task_id: "T1",
      tier: "light",
      verifier: "claimgate",
      verifier_mode: "read_only",
      outcome: "success",
      acceptance_status: "accepted",
      created_at: "2026-01-01T00:00:00Z",
    });
    console.log("verify event result:", result);
    expect(result.valid).toBe(true);
  });

  it("rejects a verify event with invalid tier", async () => {
    const result = await validateVerifyEvent({
      event_id: "VE-1",
      event_type: "verify_completed",
      task_id: "T1",
      tier: "small",
      verifier: "claimgate",
      verifier_mode: "read_only",
      outcome: "success",
      acceptance_status: "accepted",
      created_at: "2026-01-01T00:00:00Z",
    });
    expect(result.valid).toBe(false);
  });
});
