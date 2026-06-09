import { describe, it, expect } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { execaNode } from "../src/test-helpers.js";

describe("quick command", () => {
  it("quick appears in default help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("quick");
  });

  it("quick appears in --help-maturity", async () => {
    const { stdout, exitCode } = await execaNode(["--help-maturity"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("quick");
    expect(stdout).toContain("beta:");
  });

  it("quick --json outputs parseable result", async () => {
    const { stdout, exitCode } = await execaNode(["quick", "--json"]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(typeof result.root).toBe("string");
    expect(result.root.length).toBeGreaterThan(0);
    expect(typeof result.recommendation).toBe("string");
    expect(result.recommendation.length).toBeGreaterThan(0);
    expect(typeof result.reason).toBe("string");
    expect(result.reason.length).toBeGreaterThan(0);
    expect(Array.isArray(result.next_steps)).toBe(true);
    expect(result.next_steps.length).toBeGreaterThan(0);
    expect(Array.isArray(result.detected_signals)).toBe(true);
  });

  it("quick default text output includes sections", async () => {
    const { stdout, exitCode } = await execaNode(["quick"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Next-action recommender");
    expect(stdout).toContain("root:");
    expect(stdout).toContain("recommendation:");
    expect(stdout).toContain("reason:");
    expect(stdout).toContain("Next steps:");
    expect(stdout).toContain("xh run builtin:ci --dry-run");
    expect(stdout).toContain("xh learn");
  });

  it("quick --help shows usage", async () => {
    const { stdout, exitCode } = await execaNode(["quick", "--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("quick");
  });

  it("quick in empty dir recommends start", async () => {
    const { stdout, exitCode } = await execaNode([
      "quick",
      "--root",
      "/tmp",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.recommendation).toContain("xh start");
    const hasStartOrInit = result.next_steps.some(
      (s: string) => s === "xh start" || s === "xh init"
    );
    expect(hasStartOrInit).toBe(true);
  });

  it("quick excludes .x-harness/tmp and .x-harness/cache cards", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-quick-test-"));
    fs.mkdirSync(path.join(tmpDir, ".x-harness", "tmp"), { recursive: true });
    fs.mkdirSync(path.join(tmpDir, ".x-harness", "cache"), { recursive: true });
    fs.writeFileSync(path.join(tmpDir, "AGENTS.md"), "# AGENTS\n");
    fs.writeFileSync(
      path.join(tmpDir, ".x-harness", "tmp", "completion-card.yaml"),
      "task_id: tmp\n"
    );
    fs.writeFileSync(
      path.join(tmpDir, ".x-harness", "cache", "completion-card.yaml"),
      "task_id: cache\n"
    );
    fs.writeFileSync(
      path.join(tmpDir, "completion-card.yaml"),
      "task_id: real\n"
    );

    const { stdout, exitCode } = await execaNode([
      "quick",
      "--root",
      tmpDir,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    const cards = result.detected_signals.filter((s: string) =>
      s.startsWith("completion_card:")
    );
    expect(cards).toEqual(["completion_card:completion-card.yaml"]);
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });
});
