import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("progressive disclosure", () => {
  it("no args shows start-here guide", async () => {
    const { stdout, exitCode } = await execaNode([]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Start here");
    expect(stdout).toContain("check");
    expect(stdout).toContain("--help-all");
  });

  it("default help shows only beginner commands", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    // Beginner commands should be present
    expect(stdout).toContain("check");
    expect(stdout).toContain("prepare");
    expect(stdout).toContain("doctor");
    expect(stdout).toContain("actions");
    expect(stdout).toContain("status");
    expect(stdout).toContain("reset");
    expect(stdout).toContain("init");
    expect(stdout).toContain("add");
    // Advanced commands should not appear
    expect(stdout).not.toContain("packet");
    expect(stdout).not.toContain("intake");
    expect(stdout).not.toContain("governance");
    expect(stdout).not.toContain("federation");
    // Footer should be present
    expect(stdout).toContain("--help-all");
    expect(stdout).toContain("--help-maturity");
  });

  it("--help-all shows advanced commands", async () => {
    const { stdout, exitCode } = await execaNode(["--help-all"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("trace");
    expect(stdout).toContain("benchmark");
    expect(stdout).toContain("packet");
    expect(stdout).toContain("intake");
    expect(stdout).toContain("check");
    expect(stdout).toContain("doctor");
  });

  it("--help-maturity groups commands by maturity", async () => {
    const { stdout, exitCode } = await execaNode(["--help-maturity"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("stable:");
    expect(stdout).toContain("beta:");
    expect(stdout).toContain("experimental:");
    expect(stdout).toContain("check");
    expect(stdout).toContain("packet");
    expect(stdout).toContain("intake");
  });

  it("advanced commands still execute when called directly", async () => {
    const { stdout, exitCode } = await execaNode(["doctor"]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output).toHaveProperty("healthy");
  });
});
