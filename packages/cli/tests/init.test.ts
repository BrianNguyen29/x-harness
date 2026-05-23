import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import fs from "fs-extra";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";

describe("init command", () => {
  it("defaults to minimal mode", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-init-"));
    try {
      const { stdout, exitCode } = await execaNode(["init", tmpDir]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("init (minimal) complete");
      expect(await fs.pathExists(path.join(tmpDir, "AGENTS.md"))).toBe(true);
      expect(
        await fs.pathExists(
          path.join(tmpDir, "templates", "SUBAGENT_TASK_light.md")
        )
      ).toBe(true);
      expect(
        await fs.pathExists(path.join(tmpDir, "policies", "admission.yaml"))
      ).toBe(true);
      // Standard-only files should NOT be present
      expect(await fs.pathExists(path.join(tmpDir, "schemas"))).toBe(false);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("standard mode installs schemas and examples", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-init-"));
    try {
      const { stdout, exitCode } = await execaNode([
        "init",
        tmpDir,
        "--standard",
      ]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("init (standard) complete");
      expect(await fs.pathExists(path.join(tmpDir, "schemas"))).toBe(true);
      expect(await fs.pathExists(path.join(tmpDir, "01-solo-agent"))).toBe(
        true
      );
      expect(await fs.pathExists(path.join(tmpDir, "02-assisted-agent"))).toBe(
        true
      );
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("dry-run shows plan without copying", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-init-"));
    try {
      const { stdout, exitCode } = await execaNode([
        "init",
        tmpDir,
        "--dry-run",
      ]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("dry run");
      expect(await fs.pathExists(path.join(tmpDir, "AGENTS.md"))).toBe(false);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("conflict without force exits with summary", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-init-"));
    try {
      // First init
      await execaNode(["init", tmpDir]);
      // Second init without force should fail
      const { stdout, stderr, exitCode } = await execaNode(["init", tmpDir]);
      expect(exitCode).toBe(1);
      const output = stdout + stderr;
      expect(output).toContain("blocked");
      expect(output).toContain("conflict");
      expect(output).toContain("Use --force to overwrite");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("force overwrites existing files", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-init-"));
    try {
      await execaNode(["init", tmpDir]);
      const { stdout, exitCode } = await execaNode(["init", tmpDir, "--force"]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("init (minimal) complete");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("merge preserves existing files and copies missing ones", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-init-"));
    try {
      // First init
      await execaNode(["init", tmpDir]);
      const agentsPath = path.join(tmpDir, "AGENTS.md");

      // Modify a file that will be overwritten if not for --merge
      await fs.writeFile(agentsPath, "custom content", "utf-8");

      // Second init with --merge should not overwrite the modified file
      const { stdout, exitCode } = await execaNode(["init", tmpDir, "--merge"]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("init (minimal) complete");

      // File should still have custom content (not overwritten)
      const agentsContentAfter = await fs.readFile(agentsPath, "utf-8");
      expect(agentsContentAfter).toBe("custom content");

      // But missing files should be copied (e.g., X_HARNESS.md)
      expect(await fs.pathExists(path.join(tmpDir, "X_HARNESS.md"))).toBe(true);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("is registered in help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("init");
  });
});
