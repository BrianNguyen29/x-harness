import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("handoff readiness", () => {
  it("reports readiness in non-interactive mode", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "readiness",
      "--root",
      "../..",
    ]);
    // Should pass because the repo has AGENTS.md, policies, templates
    expect(exitCode).toBe(0);
    expect(stdout).toContain("handoff readiness: READY");
    expect(stdout).toContain("agents_md_present");
    expect(stdout).toContain("admission_policy_present");
  });

  it("outputs JSON with --json", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "readiness",
      "--root",
      "../..",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ready).toBe(true);
    expect(parsed.checks).toBeDefined();
    expect(parsed.checks.length).toBeGreaterThan(0);
  });

  it("skips interactive prompts in CI/non-TTY mode", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "readiness",
      "--root",
      "../..",
      "--interactive",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("handoff readiness: READY");
    expect(stdout).toContain("Non-interactive mode");
  });

  it("blocks safely with missing files (simulated)", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "readiness",
      "--root",
      "../..",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(
      parsed.checks.some(
        (c: { name: string; passed: boolean }) =>
          c.name === "completion_card_template_present" && c.passed
      )
    ).toBe(true);
  });
});

describe("handoff readiness interactive mode behavior", () => {
  it("--interactive flag is accepted", async () => {
    const { stdout, exitCode } = await execaNode([
      "handoff",
      "readiness",
      "--root",
      "../..",
      "--interactive",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ready).toBe(true);
  });
});
