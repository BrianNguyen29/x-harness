import { describe, it, expect, beforeEach } from "vitest";
import * as path from "node:path";
import * as fs from "node:fs";
import * as os from "node:os";
import { fileURLToPath } from "node:url";
import {
  loadAuthorityPolicy,
  classifyPath,
  getProtectedPaths,
  checkGovernance,
  explainPath,
  isReportOnly,
  type AuthorityPolicy,
} from "../src/core/authority.js";
import { sha256File } from "../src/core/hash.js";
import { execaNode } from "../src/test-helpers.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("authority", () => {
  let policy: AuthorityPolicy;

  beforeEach(async () => {
    policy = await loadAuthorityPolicy(repoRoot);
  });

  describe("loadAuthorityPolicy", () => {
    it("loads authority policy from policies/authority.yaml", async () => {
      expect(policy).not.toBeNull();
      expect(policy.version).toBe(1);
    });

    it("has required authority classes", () => {
      expect(policy.authority_classes).toHaveProperty("agent_editable");
      expect(policy.authority_classes).toHaveProperty(
        "agent_proposable_human_approved"
      );
      expect(policy.authority_classes).toHaveProperty("human_only");
    });

    it("has protected_paths array", () => {
      expect(Array.isArray(policy.protected_paths)).toBe(true);
      expect(policy.protected_paths.length).toBeGreaterThan(0);
    });

    it("has report_only true (PR2 deferred enforcement)", () => {
      expect(policy.report_only).toBe(true);
    });

    it("governance_check is warn-only", () => {
      expect(policy.governance_check).toBeDefined();
      expect(policy.governance_check.behavior).toBe("warn");
      expect(policy.governance_check.exit_on_warnings).toBe(false);
      expect(policy.governance_check.block_on_violations).toBe(false);
    });
  });

  describe("classifyPath", () => {
    it("classifies schemas/** as human_only", () => {
      const result = classifyPath(
        "schemas/completion-card.schema.json",
        policy
      );
      expect(result.authority).toBe("human_only");
    });

    it("classifies policies/admission.yaml as human_only", () => {
      const result = classifyPath("policies/admission.yaml", policy);
      expect(result.authority).toBe("human_only");
    });

    it("classifies policies/authority.yaml as human_only", () => {
      const result = classifyPath("policies/authority.yaml", policy);
      expect(result.authority).toBe("human_only");
    });

    it("classifies policies/permissions.yaml as human_only", () => {
      const result = classifyPath("policies/permissions.yaml", policy);
      expect(result.authority).toBe("human_only");
    });

    it("classifies packages/cli/src/core/admission.ts as human_only", () => {
      const result = classifyPath("packages/cli/src/core/admission.ts", policy);
      expect(result.authority).toBe("human_only");
    });

    it("classifies packages/cli/src/core/mutation-guard.ts as human_only", () => {
      const result = classifyPath(
        "packages/cli/src/core/mutation-guard.ts",
        policy
      );
      expect(result.authority).toBe("human_only");
    });

    it("classifies .github/workflows/x-harness-verify.yml as human_only", () => {
      const result = classifyPath(
        ".github/workflows/x-harness-verify.yml",
        policy
      );
      expect(result.authority).toBe("human_only");
    });

    it("classifies package.json as human_only", () => {
      const result = classifyPath("package.json", policy);
      expect(result.authority).toBe("human_only");
    });

    it("classifies policies/recovery.yaml as agent_proposable_human_approved", () => {
      const result = classifyPath("policies/recovery.yaml", policy);
      expect(result.authority).toBe("agent_proposable_human_approved");
    });

    it("classifies packages/cli/src/**/*.ts as agent_editable", () => {
      const result = classifyPath(
        "packages/cli/src/commands/my-command.ts",
        policy
      );
      expect(result.authority).toBe("agent_editable");
    });

    it("classifies unknown paths as agent_editable by default", () => {
      const result = classifyPath("some/random/path.txt", policy);
      expect(result.authority).toBe("agent_editable");
    });

    it("handles backslash paths on Windows", () => {
      const result = classifyPath(
        "packages\\cli\\src\\core\\authority.ts",
        policy
      );
      expect(result.authority).toBe("human_only");
    });

    it("handles paths with regex metacharacters correctly", () => {
      // The old globMatch only escaped '.' so characters like '+' were treated
      // as regex quantifiers. This test ensures metacharacters in paths are
      // treated as literals after the fix.
      const result = classifyPath(
        "packages/cli/src/core/test+foo.ts",
        policy
      );
      // Should match agent_editable pattern packages/cli/src/**/*.ts
      expect(result.authority).toBe("agent_editable");
    });

    it("handles paths with dots as literal characters", () => {
      // Dots in path segments should be literal after escaping
      const result = classifyPath(
        "packages/cli/src/core/test.file.ts",
        policy
      );
      expect(result.authority).toBe("agent_editable");
    });
  });

  describe("getProtectedPaths", () => {
    it("returns all protected paths", () => {
      const paths = getProtectedPaths(policy);
      expect(Array.isArray(paths)).toBe(true);
      expect(paths.length).toBeGreaterThan(0);
    });

    it("each protected path has required fields", () => {
      const paths = getProtectedPaths(policy);
      for (const pp of paths) {
        expect(pp).toHaveProperty("path");
        expect(pp).toHaveProperty("authority");
        expect(pp).toHaveProperty("rationale");
      }
    });
  });

  describe("checkGovernance", () => {
    it("returns no warnings for agent_editable files", async () => {
      const result = await checkGovernance(
        ["packages/cli/src/commands/foo.ts"],
        repoRoot
      );
      expect(result.total_warnings).toBe(0);
    });

    it("returns warnings for human_only files", async () => {
      const result = await checkGovernance(
        ["schemas/completion-card.schema.json"],
        repoRoot
      );
      expect(result.total_warnings).toBe(1);
      expect(result.warnings[0].authority).toBe("human_only");
    });

    it("returns warnings for agent_proposable_human_approved files", async () => {
      const result = await checkGovernance(
        ["policies/recovery.yaml"],
        repoRoot
      );
      expect(result.total_warnings).toBe(1);
      expect(result.warnings[0].authority).toBe(
        "agent_proposable_human_approved"
      );
    });

    it("handles multiple files", async () => {
      const result = await checkGovernance(
        [
          "packages/cli/src/commands/foo.ts",
          "schemas/completion-card.schema.json",
          "policies/recovery.yaml",
        ],
        repoRoot
      );
      expect(result.total_warnings).toBe(2); // schemas and recovery (admission is not in test path)
    });

    it("is report_only (violations are warnings, not blocks)", async () => {
      const result = await checkGovernance(
        ["schemas/completion-card.schema.json"],
        repoRoot
      );
      expect(result.report_only).toBe(true);
      expect(result.total_violations).toBe(0);
    });

    it("enforced mode blocks protected paths without verified approval artifacts", async () => {
      const result = await checkGovernance(
        ["schemas/completion-card.schema.json"],
        repoRoot,
        { enforce: true, governance: { approval_status: "approved" } }
      );
      expect(result.enforced).toBe(true);
      expect(result.report_only).toBe(false);
      expect(result.total_violations).toBe(1);
      expect(result.violations[0].approval_verified).toBe(false);
      expect(result.violations[0].approval_note).toContain(
        "approval_artifact is missing"
      );
    });

    it("enforced mode accepts protected paths with scoped hashed approval artifacts", async () => {
      const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), "xh-authority-"));
      try {
        fs.mkdirSync(path.join(tmpRoot, "policies"), { recursive: true });
        fs.mkdirSync(path.join(tmpRoot, ".x-harness", "approvals"), {
          recursive: true,
        });
        fs.copyFileSync(
          path.join(repoRoot, "policies", "authority.yaml"),
          path.join(tmpRoot, "policies", "authority.yaml")
        );
        const approvalPath = path.join(
          tmpRoot,
          ".x-harness",
          "approvals",
          "schema.yaml"
        );
        fs.writeFileSync(
          approvalPath,
          `approval_id: APPROVAL-SCHEMA-001
decision: approved
approved_by: alice
approved_at: 2026-05-25T00:00:00.000Z
scope:
  paths:
    - schemas/completion-card.schema.json
`,
          "utf-8"
        );
        const hash = await sha256File(approvalPath);
        fs.writeFileSync(
          path.join(tmpRoot, ".x-harness", "approvals", "registry.json"),
          JSON.stringify(
            {
              approvals: [
                {
                  path: ".x-harness/approvals/schema.yaml",
                  sha256: `sha256:${hash}`,
                  status: "approved",
                  approved_by: "alice",
                  scope: {
                    paths: ["schemas/completion-card.schema.json"],
                  },
                },
              ],
            },
            null,
            2
          ),
          "utf-8"
        );
        const result = await checkGovernance(
          ["schemas/completion-card.schema.json"],
          tmpRoot,
          {
            enforce: true,
            governance: {
              approval_status: "approved",
              approval_artifact: {
                path: ".x-harness/approvals/schema.yaml",
                sha256: `sha256:${hash}`,
              },
            },
          }
        );
        expect(result.total_violations).toBe(0);
        expect(result.total_warnings).toBe(0);
      } finally {
        fs.rmSync(tmpRoot, { recursive: true, force: true });
      }
    });
  });

  describe("explainPath", () => {
    it("explains authority for a protected path", async () => {
      const result = await explainPath("policies/admission.yaml", repoRoot);
      expect(result.authority).toBe("human_only");
      expect(result.path).toBe("policies/admission.yaml");
    });

    it("explains authority for an editable path", async () => {
      const result = await explainPath("packages/cli/src/foo.ts", repoRoot);
      expect(result.authority).toBe("agent_editable");
    });
  });

  describe("isReportOnly", () => {
    it("returns true when report_only is set", () => {
      expect(isReportOnly(policy)).toBe(true);
    });
  });
});

describe("governance report-only behavior", () => {
  it("PR2 governance check never blocks admission", async () => {
    const policy = await loadAuthorityPolicy(repoRoot);
    expect(policy.report_only).toBe(true);
    expect(policy.governance_check.block_on_violations).toBe(false);
    expect(policy.governance_check.exit_on_warnings).toBe(false);
  });

  it("warnings do not cause non-zero exit in report-only mode", async () => {
    const result = await checkGovernance(
      ["schemas/completion-card.schema.json"],
      repoRoot
    );
    // In report-only mode, violations go to warnings, not blocks
    expect(result.warnings.length).toBeGreaterThan(0);
    expect(result.violations.length).toBe(0);
    expect(result.total_violations).toBe(0);
  });

  it("governance check --enforce exits non-zero for spoofed protected approval", async () => {
    const { stdout, exitCode } = await execaNode([
      "governance",
      "check",
      "--card",
      path.join(
        repoRoot,
        "examples",
        "adversarial",
        "spoofed-protected-approval",
        "completion-card.yaml"
      ),
      "--root",
      repoRoot,
      "--enforce",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.enforced).toBe(true);
    expect(output.total_violations).toBe(1);
    expect(output.violations[0].approval_note).toContain(
      "approval_artifact is missing"
    );
  });

  it("governance check --card --diff strict reports evidence/diff mismatch", async () => {
    const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), "xh-gov-diff-"));
    try {
      const { execFileSync } = await import("node:child_process");
      fs.mkdirSync(path.join(tmpRoot, "policies"), { recursive: true });
      fs.mkdirSync(path.join(tmpRoot, "schemas"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "authority.yaml"),
        path.join(tmpRoot, "policies", "authority.yaml")
      );
      fs.writeFileSync(
        path.join(tmpRoot, "schemas", "completion-card.schema.json"),
        "{}\n",
        "utf-8"
      );
      fs.writeFileSync(
        path.join(tmpRoot, "completion-card.yaml"),
        `schema_version: "1"
task_id: TASK-GOV-DIFF-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/minimal.ts
  manual_rationale: Minimal task completed successfully
claim:
  fix_status: fixed
  summary: Minimal task completed successfully
  evidence:
    - description: verified manually
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`,
        "utf-8"
      );
      execFileSync("git", ["init"], { cwd: tmpRoot });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpRoot,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpRoot });
      execFileSync("git", ["add", "."], { cwd: tmpRoot });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpRoot });
      fs.writeFileSync(
        path.join(tmpRoot, "schemas", "completion-card.schema.json"),
        '{ "changed": true }\n',
        "utf-8"
      );

      const { stdout, exitCode } = await execaNode([
        "governance",
        "check",
        "--card",
        path.join(tmpRoot, "completion-card.yaml"),
        "--root",
        tmpRoot,
        "--diff",
        "HEAD",
        "--changed-files-source",
        "strict",
        "--json",
      ]);
      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.changed_files.source).toBe("strict");
      expect(output.errors.join("; ")).toContain(
        "evidence.files_changed missing git diff file"
      );
    } finally {
      fs.rmSync(tmpRoot, { recursive: true, force: true });
    }
  });
});
