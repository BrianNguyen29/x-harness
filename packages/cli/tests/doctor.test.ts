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
    expect(report.checks.length).toBeGreaterThan(0);
    for (const check of report.checks) {
      expect(check.status).toBe("pass");
    }
  });

  it("fails when critical assets are missing", async () => {
    const { exitCode } = await execaNode(["doctor", "--root", "/tmp/nonexistent-x-harness"]);
    expect(exitCode).toBe(1);
  });

  it("includes schema compile check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const schemaCheck = report.checks.find((c: any) => c.name === "schema_compile");
    expect(schemaCheck).toBeDefined();
    expect(schemaCheck.status).toBe("pass");
  });

  it("includes policy key check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const policyCheck = report.checks.find((c: any) => c.name === "policy_keys");
    expect(policyCheck).toBeDefined();
    expect(policyCheck.status).toBe("pass");
  });

  it("includes no-python-core check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const pyCheck = report.checks.find((c: any) => c.name === "no_python_core");
    expect(pyCheck).toBeDefined();
    expect(pyCheck.status).toBe("pass");
  });

  it("includes PGV authority wording check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const pgvCheck = report.checks.find((c: any) => c.name === "pgv_authority_wording");
    expect(pgvCheck).toBeDefined();
    expect(pgvCheck.status).toBe("pass");
  });

  it("includes tier label check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const tierCheck = report.checks.find((c: any) => c.name === "tier_labels");
    expect(tierCheck).toBeDefined();
    expect(tierCheck.status).toBe("pass");
  });

  it("includes AGENTS size check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const agentsCheck = report.checks.find((c: any) => c.name === "agents_size");
    expect(agentsCheck).toBeDefined();
    expect(agentsCheck.status).toBe("pass");
  });

  it("includes adapter presence check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const adapterCheck = report.checks.find((c: any) => c.name === "adapters_present");
    expect(adapterCheck).toBeDefined();
    expect(adapterCheck.status).toBe("pass");
  });

  it("includes evidence scope support check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find((c: any) => c.name === "evidence_scope_support");
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
  });

  it("includes read-only verifier check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find((c: any) => c.name === "read_only_verifier");
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
  });

  it("includes no heavy runtime check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find((c: any) => c.name === "no_heavy_runtime");
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
  });
});
