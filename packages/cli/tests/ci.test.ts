import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("ci command", () => {
  it("ci appears in default help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("ci");
  });

  it("ci appears in --help-maturity", async () => {
    const { stdout, exitCode } = await execaNode(["--help-maturity"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("ci");
    expect(stdout).toContain("beta:");
  });

  it("ci --dry-run outputs planned steps", async () => {
    const { stdout, exitCode } = await execaNode(["ci", "--dry-run"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("doctor");
    expect(stdout).toContain("examples_verify");
    expect(stdout).toContain("verify_ci_standard");
    expect(stdout).toContain("builtin:ci");
  });

  it("ci --dry-run --json outputs parseable result", async () => {
    const { stdout, exitCode } = await execaNode(["ci", "--dry-run", "--json"]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.recipe).toBe("builtin:ci");
    expect(result.ok).toBe(true);
    expect(result.steps).toBeInstanceOf(Array);
    expect(result.steps.length).toBeGreaterThan(0);
  });

  it("ci --help shows usage", async () => {
    const { stdout, exitCode } = await execaNode(["ci", "--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("ci");
  });
});
