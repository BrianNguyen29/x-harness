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

  it("passes admission with valid inputs", () => {
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

  it("withholds on stale ground", () => {
    const result = runAdmission({
      claim: { id: "C1" },
      staleGround: true,
    });
    expect(result.outcome).toBe("blocked");
    expect(result.acceptance_status).toBe("withheld");
    expect(result.errors[0]).toContain("stale_ground");
  });

  it("fails on canonical contradiction: passed + not_fixed", () => {
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

  it("fails when standard tier lacks evidence", () => {
    const result = runAdmission({
      claim: { id: "C1" },
      tier: "standard",
    });
    expect(result.outcome).toBe("failed");
    expect(result.errors[0]).toContain("requires evidence");
  });

  it("allows light tier without evidence", () => {
    const result = runAdmission({
      claim: { id: "C1" },
      tier: "light",
    });
    expect(result.outcome).toBe("success");
  });
});
