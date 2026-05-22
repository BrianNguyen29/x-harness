import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("handoff command", () => {
  it("outputs light tier template", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "light",
      "--title",
      "Fix bug",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("SUBAGENT_TASK light");
    expect(stdout).toContain("Fix bug");
    expect(stdout).toContain("next_action");
  });

  it("outputs standard tier template with task description", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "standard",
      "--title",
      "Refactor auth",
      "--task",
      "Split auth module into services.",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("SUBAGENT_TASK standard");
    expect(stdout).toContain("Refactor auth");
    expect(stdout).toContain("Split auth module into services.");
  });

  it("outputs deep tier template", async () => {
    const { stdout, exitCode } = await execaNode(["handoff", "deep"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("SUBAGENT_TASK deep");
    expect(stdout).toContain("Untitled");
  });

  it("includes context header by default for standard tier", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "standard",
      "--title",
      "Test",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("## Context");
    expect(stdout).toContain("Completion is admitted, not claimed.");
  });

  it("omits context header with --no-context", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "standard",
      "--title",
      "Test",
      "--no-context",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).not.toContain("## Context");
    expect(stdout).not.toContain("Completion is admitted, not claimed.");
  });

  it("shows help for invalid tier", async () => {
    const { stdout, stderr, exitCode } = await execaNode([
      "handoff",
      "invalid",
    ]);
    expect(exitCode).not.toBe(0);
    const output = stdout + stderr;
    expect(output).toMatch(/error|unknown|help/i);
  });
});
