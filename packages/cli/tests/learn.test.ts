import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("learn command", () => {
  it("learn does not appear in default help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).not.toContain("learn");
  });

  it("learn appears in --help-all", async () => {
    const { stdout, exitCode } = await execaNode(["--help-all"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("learn");
  });

  it("learn appears in --help-maturity", async () => {
    const { stdout, exitCode } = await execaNode(["--help-maturity"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("learn");
    expect(stdout).toContain("beta:");
  });

  it("learn --json outputs parseable result with sections and next_steps", async () => {
    const { stdout, exitCode } = await execaNode(["learn", "--json"]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.sections).toBeInstanceOf(Array);
    expect(result.sections.length).toBeGreaterThan(0);
    expect(result.next_steps).toBeInstanceOf(Array);
    expect(result.next_steps.length).toBeGreaterThan(0);
  });

  it("learn default text output includes sections", async () => {
    const { stdout, exitCode } = await execaNode(["learn"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Concept tour");
    expect(stdout).toContain("Overview");
    expect(stdout).toContain("Core concepts");
    expect(stdout).toContain("Tiers and evidence");
    expect(stdout).toContain("Next steps");
  });

  it("learn --help shows usage", async () => {
    const { stdout, exitCode } = await execaNode(["learn", "--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("learn");
  });

  it("learn --lang vi shows Vietnamese output", async () => {
    const { stdout, exitCode } = await execaNode(["learn", "--lang", "vi"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Khái niệm cơ bản");
    expect(stdout).toContain("Tổng quan");
    expect(stdout).toContain("Bước tiếp theo:");
  });

  it("learn --lang vi --json keeps English JSON", async () => {
    const { stdout, exitCode } = await execaNode([
      "learn",
      "--lang",
      "vi",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.sections[0].title).toBe("Overview");
  });
});
