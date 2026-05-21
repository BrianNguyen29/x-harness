import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("verify command", () => {
  it("accepts a passing fixture via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim", "tests/fixtures/claim-pass.yaml",
      "--evidence", "tests/fixtures/evidence-pass.yaml",
      "--subagent-return", "tests/fixtures/subagent-pass.yaml",
      "--tier", "standard",
      "--task-id", "TASK-001",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(true);
    expect(event.acceptance_status).toBe("accepted");
  });

  it("withholds on canonical contradiction via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--subagent-return", "tests/fixtures/subagent-contradiction.yaml",
      "--tier", "light",
      "--task-id", "TASK-002",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
    expect(event.withheld_reason).toContain("canonical contradiction");
  });

  it("withholds when standard tier lacks evidence via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim", "tests/fixtures/claim-pass.yaml",
      "--tier", "standard",
      "--task-id", "TASK-003",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
  });

  it("accepts a completion card via --card", async () => {
    const cardPath = path.join(repoRoot, "examples", "00-minimal", "completion-card.yaml");
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card", cardPath,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.acceptance_status).toBe("accepted");
  });

  it("withholds a blocked completion card via --card", async () => {
    const cardPath = path.join(repoRoot, "examples", "04-blocked-verification", "completion-card.yaml");
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card", cardPath,
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.acceptance_status).toBe("withheld");
  });

  it("prints quiet output by default", async () => {
    const cardPath = path.join(repoRoot, "examples", "00-minimal", "completion-card.yaml");
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card", cardPath,
    ]);
    expect(exitCode).toBe(0);
    const lines = stdout.split("\n").filter((l) => l.trim().length > 0);
    expect(lines.length).toBeLessThanOrEqual(3);
    expect(stdout).toContain("outcome: success");
    expect(stdout).toContain("acceptance_status: accepted");
  });

  it("prints verbose output with --verbose", async () => {
    const cardPath = path.join(repoRoot, "examples", "00-minimal", "completion-card.yaml");
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card", cardPath,
      "--verbose",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("ACCEPTED");
    expect(stdout).toContain("Tier: light");
  });

  it("prints withheld verbose output for blocked card with --verbose", async () => {
    const cardPath = path.join(repoRoot, "examples", "04-blocked-verification", "completion-card.yaml");
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card", cardPath,
      "--verbose",
    ]);
    expect(exitCode).toBe(1);
    expect(stdout).toContain("WITHHELD");
    expect(stdout).toContain("Handoff:");
  });
});
