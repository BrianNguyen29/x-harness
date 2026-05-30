import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("profile recommend command", () => {
  it("recommends standard for PR verification goal", async () => {
    const { stdout, exitCode } = await execaNode([
      "profile",
      "recommend",
      "--goal",
      "AI PR verification",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Recommended profile: standard");
    expect(stdout).toContain("Reason:");
    expect(stdout).toContain("Required commands:");
    expect(stdout).toContain("Recommended checks:");
    expect(stdout).toContain("Not needed:");
  });

  it("recommends deep for release readiness goal", async () => {
    const { stdout, exitCode } = await execaNode([
      "profile",
      "recommend",
      "--goal",
      "release readiness",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Recommended profile: deep");
    expect(stdout).toContain("release readiness");
  });

  it("outputs JSON with --json", async () => {
    const { stdout, exitCode } = await execaNode([
      "profile",
      "recommend",
      "--goal",
      "AI PR verification",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const rec = JSON.parse(stdout);
    expect(rec.recommended_profile).toBe("standard");
    expect(rec.reason).toBeTruthy();
    expect(rec.required_commands.length).toBeGreaterThan(0);
    expect(rec.recommended_checks.length).toBeGreaterThan(0);
    expect(rec.not_needed.length).toBeGreaterThan(0);
  });

  it("outputs JSON for deep goal", async () => {
    const { stdout, exitCode } = await execaNode([
      "profile",
      "recommend",
      "--goal",
      "release readiness",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const rec = JSON.parse(stdout);
    expect(rec.recommended_profile).toBe("deep");
    expect(rec.not_needed.length).toBe(0);
  });

  it("errors without --goal", async () => {
    const { stderr, exitCode } = await execaNode(["profile", "recommend"]);
    expect(exitCode).toBe(1);
    expect(stderr).toContain("required option");
  });

  it("recommends minimal for local/quick goal", async () => {
    const { stdout, exitCode } = await execaNode([
      "profile",
      "recommend",
      "--goal",
      "local quick task",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Recommended profile: minimal");
  });

  it("defaults to standard for unknown goal", async () => {
    const { stdout, exitCode } = await execaNode([
      "profile",
      "recommend",
      "--goal",
      "something completely unrelated",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Recommended profile: standard");
  });

  it("recommends deep for security goal", async () => {
    const { stdout, exitCode } = await execaNode([
      "profile",
      "recommend",
      "--goal",
      "deep security-sensitive change",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Recommended profile: deep");
  });

  it("is registered in help", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("profile");
  });
});
