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
  });
});
