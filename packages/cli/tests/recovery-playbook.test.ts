import { describe, it, expect } from "vitest";
import {
  generatePlaybook,
  renderPlaybookMarkdown,
} from "../src/core/recovery.js";
import { execaNode } from "../src/test-helpers.js";

describe("generatePlaybook", () => {
  it("returns empty array for success outcome", () => {
    const result = generatePlaybook(["some error"], "success");
    expect(result).toHaveLength(0);
  });

  it("returns empty array for skipped outcome", () => {
    const result = generatePlaybook(["some error"], "skipped");
    expect(result).toHaveLength(0);
  });

  it("suggests typecheck_failed for type errors", () => {
    const result = generatePlaybook(
      ["tsc --noEmit reported typecheck errors"],
      "failed"
    );
    expect(result).toHaveLength(1);
    expect(result[0].predicate).toBe("typecheck_failed");
    expect(result[0].review_required).toBe(true);
    expect(result[0].route.owner).toBe("implementation-worker");
  });

  it("suggests test_failed for test errors", () => {
    const result = generatePlaybook(["unit tests failed"], "failed");
    expect(result).toHaveLength(1);
    expect(result[0].predicate).toBe("test_failed");
    expect(result[0].review_required).toBe(true);
  });

  it("deduplicates multiple errors mapping to the same predicate", () => {
    const result = generatePlaybook(
      ["unit tests failed", "integration tests failed"],
      "failed"
    );
    expect(result).toHaveLength(1);
    expect(result[0].predicate).toBe("test_failed");
  });

  it("suggests multiple predicates for distinct errors", () => {
    const result = generatePlaybook(
      ["tsc --noEmit reported typecheck errors", "unit tests failed"],
      "failed"
    );
    expect(result).toHaveLength(2);
    const predicates = result.map((r) => r.predicate);
    expect(predicates).toContain("typecheck_failed");
    expect(predicates).toContain("test_failed");
  });

  it("does not mutate policies or completion cards", () => {
    // generatePlaybook is a pure function: it takes inputs and returns a new array.
    const errors = ["missing evidence packet"];
    const result1 = generatePlaybook(errors, "failed");
    const result2 = generatePlaybook(errors, "failed");
    expect(result1).toEqual(result2);
    // Ensure it returns new objects
    expect(result1).not.toBe(result2);
  });

  it("marks all suggestions as review_required", () => {
    const result = generatePlaybook(
      [
        "missing evidence packet",
        "tsc --noEmit reported typecheck errors",
        "eslint lint errors found",
      ],
      "failed"
    );
    for (const s of result) {
      expect(s.review_required).toBe(true);
    }
  });
});

describe("renderPlaybookMarkdown", () => {
  it("renders header and review notice", () => {
    const md = renderPlaybookMarkdown([]);
    expect(md).toContain("# Recovery Playbook (Review Required)");
    expect(md).toContain("Review before applying");
    expect(md).toContain("does NOT modify policies");
  });

  it("renders suggestions with correct fields", () => {
    const suggestions = generatePlaybook(["unit tests failed"], "failed");
    const md = renderPlaybookMarkdown(suggestions);
    expect(md).toContain("## test_failed");
    expect(md).toContain("Next action:");
    expect(md).toContain("Owner:");
    expect(md).toContain("**Review required:** yes");
    expect(md).toContain("Rationale:");
  });
});

describe("recovery suggest CLI", () => {
  it("outputs markdown playbook for errors", async () => {
    const { stdout, exitCode } = await execaNode([
      "recovery",
      "suggest",
      "--errors",
      "unit tests failed",
      "--outcome",
      "failed",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("# Recovery Playbook (Review Required)");
    expect(stdout).toContain("test_failed");
  });

  it("outputs JSON with --json", async () => {
    const { stdout, exitCode } = await execaNode([
      "recovery",
      "suggest",
      "--errors",
      "unit tests failed",
      "--outcome",
      "failed",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.suggestions).toBeDefined();
    expect(parsed.suggestions.length).toBeGreaterThan(0);
    expect(parsed.suggestions[0].predicate).toBe("test_failed");
  });

  it("returns empty playbook for success outcome", async () => {
    const { stdout, exitCode } = await execaNode([
      "recovery",
      "suggest",
      "--errors",
      "some error",
      "--outcome",
      "success",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("No recovery actions suggested");
  });
});
