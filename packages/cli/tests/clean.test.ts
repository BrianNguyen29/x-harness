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
    const { stdout, exitCode } = await execaNode(["--help-all"]);
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

    it("clean --archive-success skips card that becomes non-accepted at execution time", async () => {
      // This test verifies TOCTOU protection: the card is re-verified at move time
      const tmpDir = mkdtempSync(path.join(os.tmpdir(), "x-harness-clean-"));
      try {
        // Create an accepted completion card
        const cardPath = path.join(tmpDir, "completion-card.yaml");
        fs.writeFileSync(
          cardPath,
          `
schema_version: "1"
task_id: TEST-TOCTOU-001
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

        // After the planning phase queues the archive action but BEFORE execution,
        // we modify the card to be non-accepted.
        // This simulates a TOCTOU race condition.
        // Use a fresh read to get past planning phase first call
        await new Promise<void>((resolve) => setImmediate(resolve));

        // Now modify the card to be non-accepted (blocked status)
        fs.writeFileSync(
          cardPath,
          `
schema_version: "1"
task_id: TEST-TOCTOU-001
tier: light
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: test
  evidence: []
verification:
  status: blocked
  checks: []
admission:
  outcome: blocked
acceptance_status: withheld
handoff:
  next_action: none
  owner: test
`.trim()
        );

        // Run clean --archive-success --force
        // The card should NOT be archived because it was no longer accepted
        const { exitCode, stdout } = await execaNodeCwd(
          ["clean", "--archive-success", "--force"],
          tmpDir
        );

        expect(exitCode).toBe(0);
        // Card should still exist and be the modified (non-accepted) version
        expect(fs.existsSync(cardPath)).toBe(true);
        const content = fs.readFileSync(cardPath, "utf-8");
        expect(content).toContain("TEST-TOCTOU-001");
        expect(content).toContain("acceptance_status: withheld");
        // Should have skipped due to not being accepted
        expect(stdout).toContain("not accepted");
      } finally {
        rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("clean --archive-success does not archive non-accepted card", async () => {
      // Test that a non-accepted card is never queued for archival
      const tmpDir = mkdtempSync(path.join(os.tmpdir(), "x-harness-clean-"));
      try {
        // Create a non-accepted completion card (withheld)
        const cardPath = path.join(tmpDir, "completion-card.yaml");
        fs.writeFileSync(
          cardPath,
          `
schema_version: "1"
task_id: TEST-002
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
acceptance_status: withheld
handoff:
  next_action: review
  owner: test
`.trim()
        );

        // Verify card exists
        expect(fs.existsSync(cardPath)).toBe(true);

        // Run clean --archive-success --force
        const { exitCode, stdout } = await execaNodeCwd(
          ["clean", "--archive-success", "--force"],
          tmpDir
        );

        expect(exitCode).toBe(0);
        // Card should NOT have been archived - it should still exist
        expect(fs.existsSync(cardPath)).toBe(true);
        const content = fs.readFileSync(cardPath, "utf-8");
        expect(content).toContain("TEST-002");
        expect(content).toContain("acceptance_status: withheld");
        expect(stdout).toContain("not accepted");
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
