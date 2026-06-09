import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("run command", () => {
  it("run appears in default help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("run");
  });

  it("run appears in --help-maturity", async () => {
    const { stdout, exitCode } = await execaNode(["--help-maturity"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("run");
    expect(stdout).toContain("beta:");
  });

  it("run --list outputs builtin:ci", async () => {
    const { stdout, exitCode } = await execaNode(["run", "--list"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("builtin:ci");
  });

  it("run --list --json outputs parseable recipes", async () => {
    const { stdout, exitCode } = await execaNode(["run", "--list", "--json"]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.recipes).toBeInstanceOf(Array);
    expect(result.recipes).toContain("builtin:ci");
  });

  it("run builtin:ci --dry-run outputs planned steps", async () => {
    const { stdout, exitCode } = await execaNode([
      "run",
      "builtin:ci",
      "--dry-run",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("doctor");
    expect(stdout).toContain("examples_verify");
    expect(stdout).toContain("verify_ci_standard");
  });

  it("run builtin:ci --dry-run --json outputs parseable result", async () => {
    const { stdout, exitCode } = await execaNode([
      "run",
      "builtin:ci",
      "--dry-run",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.recipe).toBe("builtin:ci");
    expect(result.ok).toBe(true);
    expect(result.steps).toBeInstanceOf(Array);
    expect(result.steps.length).toBeGreaterThan(0);
  });

  it("run unknown recipe exits with error", async () => {
    const { stderr, exitCode } = await execaNode(["run", "unknown"]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("unknown recipe");
  });

  it("run --help shows usage", async () => {
    const { stdout, exitCode } = await execaNode(["run", "--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("run");
  });
});
