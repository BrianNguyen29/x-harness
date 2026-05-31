import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import fs from "fs-extra";
import { fileURLToPath } from "node:url";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { MANAGED_CONTRACT_TARGETS } from "../src/core/contract.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

describe("context command", () => {
  it("outputs compact context by default", async () => {
    const { stdout, exitCode } = await execaNode(["context"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Completion is admitted, not claimed.");
    expect(stdout).not.toContain("context-hash");
  });

  it("outputs verbose context with hash", async () => {
    const { stdout, exitCode } = await execaNode(["context", "--verbose"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("# x-harness Canonical Context");
    expect(stdout).toContain("context-hash:");
  });

  it("outputs valid JSON with required fields", async () => {
    const { stdout, exitCode } = await execaNode(["context", "--json"]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed).toHaveProperty("context");
    expect(parsed).toHaveProperty("hash");
    expect(parsed).toHaveProperty("mode", "compact");
    expect(parsed).toHaveProperty("agents_fresh");
    expect(parsed).toHaveProperty("agents_note");
    expect(typeof parsed.hash).toBe("string");
    expect(typeof parsed.agents_fresh).toBe("boolean");
  });

  it("outputs generated canonical runtime contract", async () => {
    const { stdout, exitCode } = await execaNode(["context", "--contract"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("# x-harness Generated Runtime Contract");
    expect(stdout).toContain("claim.fix_status");
    expect(stdout).toContain("result.fix_status");
    expect(stdout).toContain("contract-hash:");
  });

  it("outputs generated contract JSON", async () => {
    const { stdout, exitCode } = await execaNode([
      "context",
      "--contract",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.contract.fixStatus.completionCard).toContain(
      "claim.fix_status"
    );
    expect(parsed.contract.fixStatus.subagentReturn).toContain(
      "result.fix_status"
    );
    expect(parsed.markdown).toContain("Generated Runtime Contract");
    expect(typeof parsed.hash).toBe("string");
  });

  it("refreshes managed contract blocks in docs/templates/adapters", async () => {
    const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-contract-"));
    try {
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.mkdirSync(path.join(tmpDir, "schemas"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );
      fs.copyFileSync(
        path.join(repoRoot, "schemas", "completion-card.schema.json"),
        path.join(tmpDir, "schemas", "completion-card.schema.json")
      );
      for (const file of MANAGED_CONTRACT_TARGETS.map(
        (target) => target.path
      )) {
        fs.mkdirSync(path.dirname(path.join(tmpDir, file)), {
          recursive: true,
        });
        fs.writeFileSync(path.join(tmpDir, file), `# ${file}\n`, "utf-8");
      }

      const { stdout, exitCode } = await execaNode([
        "context",
        "--write-contract-assets",
        "--root",
        tmpDir,
        "--json",
      ]);
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.written).toContain("docs/RUNTIME_CONTRACT.md");
      expect(output.written).toContain("templates/COMPLETION_CARD.md");
      expect(output.written).toContain("adapters/claude-code/README.md");
      expect(
        fs.readFileSync(
          path.join(tmpDir, "docs", "RUNTIME_CONTRACT.md"),
          "utf-8"
        )
      ).toContain("BEGIN X-HARNESS MANAGED CONTRACT: runtime-contract");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("json mode reports stale when AGENTS.md missing managed block", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-context-"));
    try {
      await fs.writeFile(
        path.join(tmpDir, "AGENTS.md"),
        "# Agent Contract\n",
        "utf-8"
      );
      const { stdout, exitCode } = await execaNode([
        "context",
        "--json",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(0);
      const parsed = JSON.parse(stdout);
      expect(parsed.agents_fresh).toBe(false);
      expect(parsed.agents_note).toContain("missing");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("refresh updates AGENTS.md managed block", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-context-"));
    try {
      const agentsPath = path.join(tmpDir, "AGENTS.md");
      await fs.writeFile(agentsPath, "# Agent Contract\n", "utf-8");
      const { stdout, exitCode } = await execaNode([
        "context",
        "--refresh",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("AGENTS.md refreshed");
      expect(stdout).toContain("context-hash:");
      const content = await fs.readFile(agentsPath, "utf-8");
      expect(content).toContain("<!-- BEGIN X-HARNESS MANAGED CONTEXT -->");
      expect(content).toContain("<!-- END X-HARNESS MANAGED CONTEXT -->");
      expect(content).toContain("<!-- context-hash:");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("refresh fails when AGENTS.md is missing", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-context-"));
    try {
      const { exitCode } = await execaNode([
        "context",
        "--refresh",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(2);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("is registered in help", async () => {
    const { stdout, exitCode } = await execaNode(["--help-all"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("context");
  });
});
