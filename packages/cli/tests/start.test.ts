import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("start command", () => {
  it("start appears in default help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("start");
  });

  it("start appears in --help-maturity", async () => {
    const { stdout, exitCode } = await execaNode(["--help-maturity"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("start");
    expect(stdout).toContain("beta:");
  });

  it("start --json outputs parseable result", async () => {
    const { stdout, exitCode } = await execaNode([
      "start",
      "--skip-doctor",
      "--skip-examples",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.ok).toBe(true);
    expect(result.steps).toBeInstanceOf(Array);
    expect(result.steps.length).toBeGreaterThan(0);
    expect(result.next_steps).toBeInstanceOf(Array);
  });

  it("start default text output includes steps", async () => {
    const { stdout, exitCode } = await execaNode([
      "start",
      "--skip-doctor",
      "--skip-examples",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("xh start - Guided onboarding");
    expect(stdout).toContain("init wizard");
    expect(stdout).toContain("Next steps");
  });

  it("start invalid profile exits with error", async () => {
    const { stderr, exitCode } = await execaNode([
      "start",
      "--profile",
      "bogus",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("invalid profile");
  });

  it("start --lang vi outputs Vietnamese text", async () => {
    const { stdout, exitCode } = await execaNode([
      "start",
      "--skip-doctor",
      "--skip-examples",
      "--lang",
      "vi",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Hướng dẫn bắt đầu");
    expect(stdout).toContain("Bước tiếp theo:");
  });

  it("start --lang vi --json keeps English/machine-readable output", async () => {
    const { stdout, exitCode } = await execaNode([
      "start",
      "--skip-doctor",
      "--skip-examples",
      "--lang",
      "vi",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.ok).toBe(true);
    expect(result.next_steps.length).toBeGreaterThan(0);
    expect(result.next_steps[0]).toContain("verification");
  });
});
