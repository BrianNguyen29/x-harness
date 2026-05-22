import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import fs from "fs-extra";
import { fileURLToPath } from "node:url";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";

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
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("context");
  });
});
