import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import { readTrace } from "../src/core/trace.js";
import * as path from "node:path";
import * as fs from "node:fs";
import * as os from "node:os";
import { fileURLToPath } from "node:url";
import { spawn, execFile } from "node:child_process";

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

describe("verify command", () => {
  it("accepts a passing fixture via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim",
      "tests/fixtures/claim-pass.yaml",
      "--evidence",
      "tests/fixtures/evidence-pass.yaml",
      "--subagent-return",
      "tests/fixtures/subagent-pass.yaml",
      "--tier",
      "standard",
      "--task-id",
      "TASK-001",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(true);
    expect(event.acceptance_status).toBe("accepted");
  });

  it("withholds on canonical contradiction via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--subagent-return",
      "tests/fixtures/subagent-contradiction.yaml",
      "--tier",
      "light",
      "--task-id",
      "TASK-002",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
    expect(event.withheld_reason).toContain("canonical contradiction");
  });

  it("withholds when standard tier lacks evidence via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim",
      "tests/fixtures/claim-pass.yaml",
      "--tier",
      "standard",
      "--task-id",
      "TASK-003",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
  });

  it("accepts a completion card via --card", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "00-minimal",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.acceptance_status).toBe("accepted");
  });

  it("withholds a blocked completion card via --card", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "04-blocked-verification",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.acceptance_status).toBe("withheld");
  });

  it("prints quiet output by default", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "00-minimal",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
    ]);
    expect(exitCode).toBe(0);
    const lines = stdout.split("\n").filter((l) => l.trim().length > 0);
    expect(lines.length).toBeLessThanOrEqual(3);
    expect(stdout).toContain("outcome: success");
    expect(stdout).toContain("acceptance_status: accepted");
  });

  it("prints failed quiet output with error info", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim",
      "tests/fixtures/claim-pass.yaml",
      "--tier",
      "standard",
      "--task-id",
      "TASK-FAIL-QUIET",
    ]);
    expect(exitCode).toBe(1);
    const lines = stdout.split("\n").filter((l) => l.trim().length > 0);
    expect(lines.length).toBeLessThanOrEqual(4);
    expect(stdout).toContain("outcome:");
    expect(stdout).toContain("acceptance_status:");
    expect(stdout).toContain("checks:");
  });

  it("prints verbose output with --verbose", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "00-minimal",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--verbose",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("ACCEPTED");
    expect(stdout).toContain("Tier: light");
  });

  it("prints withheld verbose output for blocked card with --verbose", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "04-blocked-verification",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--verbose",
    ]);
    expect(exitCode).toBe(1);
    expect(stdout).toContain("WITHHELD");
    expect(stdout).toContain("Handoff:");
  });

  it("passes with --mutation-guard when no mutations occur", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-ok-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--mutation-guard", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("passes with --mutation-guard when pre-existing dirty file exists", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-dirty-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      // Create a dirty untracked file before verify
      fs.writeFileSync(path.join(tmpDir, "dirty.tmp"), "dirty content");

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--mutation-guard", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("allowlists trace writes with --mutation-guard --trace", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-tr-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--mutation-guard", "--trace", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("keeps prior behavior without --mutation-guard", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "00-minimal",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.acceptance_status).toBe("accepted");
    // Should not include mutation guard notes when flag is absent
    expect(stdout).not.toContain("mutation guard");
  });

  it("records blocked trace when mutation violation occurs", async () => {
    const script = path.join(cliDir, "dist", "index.js");
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-viol-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], {
        cwd: tmpDir,
      });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const injectPath = path.join(tmpDir, "unexpected.txt");
      const child = spawn(
        "node",
        [
          script,
          "verify",
          "--card",
          cardDst,
          "--mutation-guard",
          "--trace",
          "--json",
        ],
        {
          cwd: tmpDir,
          env: {
            ...process.env,
            X_HARNESS_ENABLE_TEST_HOOKS: "1",
            X_HARNESS_TEST_INJECT_MUTATION: injectPath,
          },
        }
      );

      let stdout = "";
      let childExited = false;
      let exitCode = 1;
      child.stdout.on("data", (chunk: Buffer) => {
        stdout += chunk.toString();
      });
      child.on("close", (code) => {
        childExited = true;
        exitCode = code ?? 1;
      });

      await new Promise<void>((resolve) => {
        const interval = setInterval(() => {
          if (childExited) {
            clearInterval(interval);
            resolve();
          }
        }, 50);
      });

      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.admission_outcome).toBe("blocked");
      expect(output.acceptance_status).toBe("withheld");
      expect(output.recovery).not.toBeNull();
      expect(output.recovery.predicate).toBe("verifier_not_read_only");
      expect(output.recovery.owner).toBe("admission-verifier");
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard blocked")
        )
      ).toBe(true);

      const traceDir = path.join(tmpDir, ".x-harness", "traces");
      const events = await readTrace(traceDir);
      expect(events.length).toBeGreaterThan(0);
      const lastEvent = events[events.length - 1];
      expect(lastEvent.outcome).toBe("blocked");
      expect(lastEvent.blocking_predicate).toBe("verifier_not_read_only");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("blocks admission when --stale-ground is set", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "00-minimal",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--stale-ground",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.admission_outcome).toBe("blocked");
    expect(event.acceptance_status).toBe("withheld");
    expect(event.withheld_reason).toContain("stale_ground");
  });

  it("blocks admission when completion card has stale_ground: true", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-stale-"));
    try {
      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      let cardContent = fs.readFileSync(cardSrc, "utf-8");
      // Add stale_ground: true at the top of the card
      cardContent = "stale_ground: true\n" + cardContent;
      fs.writeFileSync(cardDst, cardContent, "utf-8");

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--json"],
        tmpDir
      );
      expect(exitCode).toBe(1);
      const event = JSON.parse(stdout);
      expect(event.ok).toBe(false);
      expect(event.admission_outcome).toBe("blocked");
      expect(event.acceptance_status).toBe("withheld");
      expect(event.withheld_reason).toContain("stale_ground");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});
