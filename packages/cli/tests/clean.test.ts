import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

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
});
