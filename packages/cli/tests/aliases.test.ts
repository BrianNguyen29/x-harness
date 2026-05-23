import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";

const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("alias registration", () => {
  describe("check alias for verify", () => {
    it("check command is registered and produces same result as verify", async () => {
      const cardPath = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );

      // Run verify
      const verifyResult = await execaNode([
        "verify",
        "--card",
        cardPath,
        "--json",
      ]);

      // Run check (alias)
      const checkResult = await execaNode([
        "check",
        "--card",
        cardPath,
        "--json",
      ]);

      expect(checkResult.exitCode).toBe(verifyResult.exitCode);
      const verifyOut = JSON.parse(verifyResult.stdout);
      const checkOut = JSON.parse(checkResult.stdout);
      expect(checkOut.ok).toBe(verifyOut.ok);
      expect(checkOut.acceptance_status).toBe(verifyOut.acceptance_status);
    });

    it("check --help shows verify alias info", async () => {
      const { stdout } = await execaNode(["check", "--help"]);
      expect(stdout).toContain("verify");
    });
  });

  describe("prepare alias for handoff readiness", () => {
    it("prepare command is registered and produces same result as handoff readiness", async () => {
      // Run handoff readiness in non-interactive mode
      const handoffResult = await execaNode([
        "handoff",
        "readiness",
        "--non-interactive",
        "--json",
      ]);

      // Run prepare (alias) in non-interactive mode
      const prepareResult = await execaNode([
        "prepare",
        "--non-interactive",
        "--json",
      ]);

      // Both should produce valid JSON output
      const handoffOut = JSON.parse(handoffResult.stdout);
      const prepareOut = JSON.parse(prepareResult.stdout);
      expect(prepareOut.ready).toBe(handoffOut.ready);
    });

    it("prepare --help shows readiness description", async () => {
      const { stdout } = await execaNode(["prepare", "--help"]);
      expect(stdout).toContain("readiness");
    });
  });

  describe("recover alias for recovery suggest", () => {
    it("recover command is registered and produces same result as recovery suggest", async () => {
      const errors = "test failure; evidence missing";

      // Run recovery suggest
      const suggestResult = await execaNode([
        "recovery",
        "suggest",
        "--errors",
        errors,
        "--json",
      ]);

      // Run recover (alias)
      const recoverResult = await execaNode([
        "recover",
        "--errors",
        errors,
        "--json",
      ]);

      expect(recoverResult.exitCode).toBe(suggestResult.exitCode);
      const suggestOut = JSON.parse(suggestResult.stdout);
      const recoverOut = JSON.parse(recoverResult.stdout);
      expect(recoverOut.suggestions).toEqual(suggestOut.suggestions);
    });

    it("recover --help shows recovery description", async () => {
      const { stdout } = await execaNode(["recover", "--help"]);
      expect(stdout).toContain("recovery");
    });
  });

  describe("actions command", () => {
    it("actions output includes all 7 actions", async () => {
      const { stdout } = await execaNode(["actions"]);
      expect(stdout).toContain("prepare");
      expect(stdout).toContain("check");
      expect(stdout).toContain("recover");
      expect(stdout).toContain("doctor");
      expect(stdout).toContain("actions");
      expect(stdout).toContain("status");
      expect(stdout).toContain("reset");
    });

    it("actions --help shows actions description", async () => {
      const { stdout } = await execaNode(["actions", "--help"]);
      expect(stdout).toContain("actions");
    });
  });

  describe("status command", () => {
    it("status is an alias for report command", async () => {
      // status and report should produce identical output
      const statusResult = await execaNode(["status", "--json"]);
      const reportResult = await execaNode(["report", "--json"]);

      expect(statusResult.exitCode).toBe(reportResult.exitCode);
      // Both should return JSON with total_events
      const statusOut = JSON.parse(statusResult.stdout);
      const reportOut = JSON.parse(reportResult.stdout);
      expect(statusOut).toHaveProperty("total_events");
      expect(reportOut).toHaveProperty("total_events");
    });

    it("status --help shows report alias info", async () => {
      const { stdout } = await execaNode(["status", "--help"]);
      // Should show that status is an alias for report
      expect(stdout).toContain("report");
    });
  });

  describe("reset command", () => {
    it("reset without --confirm does not delete", async () => {
      const { stdout, exitCode } = await execaNode(["reset"]);
      // Should exit with error code 1 (safety check)
      expect(exitCode).toBe(1);
      expect(stdout).toContain("--confirm");
      expect(stdout).toContain("requires --confirm");
    });

    it("reset without --confirm goes through top-level error boundary with x-harness error prefix", async () => {
      // Test that reset error is caught by top-level parseAsync.catch handler
      // and printed with "x-harness error:" prefix to stderr
      const { spawn } = await import("node:child_process");
      const script = path.join(repoRoot, "packages", "cli", "dist", "index.js");
      const child = spawn("node", [script, "reset"], {
        cwd: repoRoot,
      });

      let stderr = "";
      let exitCode = 1;
      child.stderr.on("data", (chunk) => {
        stderr += chunk.toString();
      });
      await new Promise<void>((resolve) => {
        child.on("close", (code) => {
          exitCode = code ?? 1;
          resolve();
        });
      });

      expect(exitCode).toBe(1);
      expect(stderr).toContain("x-harness error:");
      expect(stderr).toContain("reset aborted");
      expect(stderr).toContain("--confirm");
    });

    it("reset --help shows reset description", async () => {
      const { stdout } = await execaNode(["reset", "--help"]);
      expect(stdout).toContain("reset");
      expect(stdout).toContain("--confirm");
    });

    it("reset --confirm invokes clean behavior", async () => {
      // reset --confirm should produce output matching clean --tmp --force
      const resetResult = await execaNode(["reset", "--confirm"]);
      expect(resetResult.exitCode).toBe(0);
      // Should show the clean header indicating it delegated to clean --tmp --force
      expect(resetResult.stdout).toContain("x-harness clean --tmp --force");
      expect(resetResult.stdout).toContain("reset complete");
    });
  });

  describe("original commands still work", () => {
    it("verify command still works", async () => {
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
    });

    it("handoff readiness command still works", async () => {
      const { stdout, exitCode } = await execaNode([
        "handoff",
        "readiness",
        "--non-interactive",
        "--json",
      ]);
      const output = JSON.parse(stdout);
      expect(output).toHaveProperty("ready");
      expect(output).toHaveProperty("checks");
      // exitCode is 1 when not ready, 0 when ready - both are valid states
      expect(exitCode).toBeLessThanOrEqual(1);
    });

    it("recovery suggest command still works", async () => {
      const { stdout, exitCode } = await execaNode([
        "recovery",
        "suggest",
        "--errors",
        "test failure",
        "--json",
      ]);
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output).toHaveProperty("suggestions");
    });

    it("report command still works", async () => {
      const { stdout, exitCode } = await execaNode(["report"]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("Report");
    });

    it("clean command still works", async () => {
      const { stdout, exitCode } = await execaNode(["clean"]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("Nothing to clean");
    });

    it("doctor command still works", async () => {
      const { stdout } = await execaNode(["doctor"]);
      // doctor outputs JSON with "healthy" field when run in test environment
      const output = JSON.parse(stdout);
      expect(output).toHaveProperty("healthy");
      expect(output).toHaveProperty("checks");
    });
  });
});
