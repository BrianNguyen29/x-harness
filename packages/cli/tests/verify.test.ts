import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("verify command", () => {
  it("accepts a passing fixture", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim", "tests/fixtures/claim-pass.yaml",
      "--evidence", "tests/fixtures/evidence-pass.yaml",
      "--subagent-return", "tests/fixtures/subagent-pass.yaml",
      "--tier", "standard",
      "--task-id", "TASK-001",
    ]);
    expect(exitCode).toBe(0);
    const event = JSON.parse(stdout);
    expect(event.outcome).toBe("success");
    expect(event.acceptance_status).toBe("accepted");
  });

  it("withholds on canonical contradiction", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--subagent-return", "tests/fixtures/subagent-contradiction.yaml",
      "--tier", "light",
      "--task-id", "TASK-002",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.outcome).toBe("failed");
    expect(event.acceptance_status).toBe("withheld");
    expect(event.errors[0]).toContain("canonical contradiction");
  });

  it("withholds when standard tier lacks evidence", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim", "tests/fixtures/claim-pass.yaml",
      "--tier", "standard",
      "--task-id", "TASK-003",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.outcome).toBe("failed");
    expect(event.acceptance_status).toBe("withheld");
  });
});
