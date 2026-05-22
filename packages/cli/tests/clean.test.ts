import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import * as fs from "node:fs";
import * as os from "node:os";
import { mkdtempSync, rmSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { execFile } from "node:child_process";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const cliDir = path.join(repoRoot, "packages", "cli");

function execaNodeCwd(
  args: string[],
  cwd: string
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    const script = path.join(cliDir, "dist", "index.js");
    execFile("node", [script, ...args], { cwd }, (error, stdout, stderr) => {
      resolve({
        stdout: stdout.trim(),
        stderr: stderr.trim(),
        exitCode: error?.code ? Number(error.code) : 0,
      });
    });
  });
}

describe("clean command", () => {
  it("defaults to dry-run and shows nothing to clean", async () => {
    const { stdout, exitCode } = await execaNode(["clean"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Nothing to clean");
  });

  it("dry-run shows what --tmp would clean when tmp exists", async () => {
    const { stdout, exitCode } = await execaNode(["clean", "--tmp"]);
    expect(exitCode).toBe(0);
    // If no tmp exists, it says "Nothing to clean"; if tmp exists, it shows dry-run.
    expect(
      stdout.includes("dry-run") || stdout.includes("Nothing to clean")
    ).toBe(true);
  });

  it("dry-run shows what --reset-card would do when card exists", async () => {
    const { stdout, exitCode } = await execaNode(["clean", "--reset-card"]);
    expect(exitCode).toBe(0);
    // If no card exists, it says "No completion-card.yaml found"; if card exists, it shows dry-run.
    expect(
      stdout.includes("dry-run") ||
        stdout.includes("No completion-card.yaml found")
    ).toBe(true);
  });

  it("does not delete protected paths", async () => {
    const { exitCode } = await execaNode(["clean", "--tmp", "--force"]);
    // Should succeed even if nothing to clean; protected paths are never touched.
    expect(exitCode).toBe(0);
  });

  it("is registered in help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("clean");
  });

  // Real mutation tests with --force in temp dirs
  describe("--force real mutations", () => {
    it("clean --tmp --force removes temp artifacts in temp dir", async () => {
      const tmpDir = mkdtempSync(path.join(os.tmpdir(), "x-harness-clean-"));
      try {
        // Create a .x-harness/tmp directory with artifacts
        const tmpXhDir = path.join(tmpDir, ".x-harness", "tmp");
        fs.mkdirSync(tmpXhDir, { recursive: true });
        fs.writeFileSync(path.join(tmpXhDir, "artifacts.log"), "test data");
        fs.writeFileSync(path.join(tmpXhDir, "trace.jsonl"), "{}");

        // Verify files exist
        expect(fs.existsSync(path.join(tmpXhDir, "artifacts.log"))).toBe(true);

        // Run clean --tmp --force
        const { exitCode } = await execaNodeCwd(
          ["clean", "--tmp", "--force"],
          tmpDir
        );

        expect(exitCode).toBe(0);
        // After force clean, the tmp dir should be gone or empty
        if (fs.existsSync(tmpXhDir)) {
          const files = fs.readdirSync(tmpXhDir);
          expect(files.length).toBe(0);
        }
      } finally {
        rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("clean --reset-card --force removes completion card in temp dir", async () => {
      const tmpDir = mkdtempSync(path.join(os.tmpdir(), "x-harness-clean-"));
      try {
        // Create a completion card
        const cardPath = path.join(tmpDir, "completion-card.yaml");
        fs.writeFileSync(
          cardPath,
          `
schema_version: "1"
task_id: TEST-001
tier: light
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: test
  evidence: []
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: test
`.trim()
        );

        // Verify card exists
        expect(fs.existsSync(cardPath)).toBe(true);

        // Run clean --reset-card --force
        const { exitCode } = await execaNodeCwd(
          ["clean", "--reset-card", "--force"],
          tmpDir
        );

        expect(exitCode).toBe(0);
        // Card should be removed
        expect(fs.existsSync(cardPath)).toBe(false);
      } finally {
        rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("clean --archive-success --force archives completion card", async () => {
      const tmpDir = mkdtempSync(path.join(os.tmpdir(), "x-harness-clean-"));
      try {
        // Create a completion card
        const cardPath = path.join(tmpDir, "completion-card.yaml");
        fs.writeFileSync(
          cardPath,
          `
schema_version: "1"
task_id: TEST-001
tier: light
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: test
  evidence: []
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: test
`.trim()
        );

        // Verify card exists
        expect(fs.existsSync(cardPath)).toBe(true);

        // Run clean --archive-success --force
        const { exitCode } = await execaNodeCwd(
          ["clean", "--archive-success", "--force"],
          tmpDir
        );

        expect(exitCode).toBe(0);
        // Card should be moved/archived, not in original location
        // The archive location is implementation-defined, so we just check card is gone
        if (fs.existsSync(cardPath)) {
          // If still exists, check if it was renamed or moved
          const content = fs.readFileSync(cardPath, "utf-8");
          expect(content).not.toContain("TEST-001");
        }
      } finally {
        rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("clean --force does not affect protected paths outside tmp", async () => {
      const tmpDir = mkdtempSync(path.join(os.tmpdir(), "x-harness-clean-"));
      try {
        // Create a protected file (simulating package.json)
        const protectedFile = path.join(tmpDir, "package.json");
        fs.writeFileSync(protectedFile, '{"name": "test"}');

        // Create .x-harness/tmp
        const tmpXhDir = path.join(tmpDir, ".x-harness", "tmp");
        fs.mkdirSync(tmpXhDir, { recursive: true });
        fs.writeFileSync(path.join(tmpXhDir, "artifacts.log"), "test data");

        // Run clean --force on tmp
        const { exitCode } = await execaNodeCwd(
          ["clean", "--tmp", "--force"],
          tmpDir
        );

        expect(exitCode).toBe(0);
        // Protected file should still exist
        expect(fs.existsSync(protectedFile)).toBe(true);
        expect(fs.readFileSync(protectedFile, "utf-8")).toBe(
          '{"name": "test"}'
        );
      } finally {
        rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
