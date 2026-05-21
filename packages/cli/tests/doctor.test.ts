import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("doctor command", () => {
  it("passes when all critical assets are present", async () => {
    const { stdout, exitCode } = await execaNode(["doctor", "--root", repoRoot]);
    expect(exitCode).toBe(0);
    const report = JSON.parse(stdout);
    expect(report.healthy).toBe(true);
    expect(report.missing_count).toBe(0);
  });

  it("fails when critical assets are missing", async () => {
    const { exitCode } = await execaNode(["doctor", "--root", "/tmp/nonexistent-x-harness"]);
    expect(exitCode).toBe(1);
  });
});
