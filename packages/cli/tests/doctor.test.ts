import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import fs from "fs-extra";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

interface Check {
  name: string;
  status: string;
  note?: string;
}

describe("doctor command", () => {
  it("passes when all critical assets are present", async () => {
    const { stdout, exitCode } = await execaNode([
      "doctor",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const report = JSON.parse(stdout);
    expect(report.healthy).toBe(true);
    expect(report.checks.length).toBeGreaterThan(0);
    for (const check of report.checks) {
      expect(check.status).toBe("pass");
    }
  });

  it("fails when critical assets are missing", async () => {
    const { exitCode } = await execaNode([
      "doctor",
      "--root",
      "/tmp/nonexistent-x-harness",
    ]);
    expect(exitCode).toBe(1);
  });

  it("includes schema compile check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const schemaCheck = report.checks.find(
      (c: Check) => c.name === "schema_compile"
    );
    expect(schemaCheck).toBeDefined();
    expect(schemaCheck.status).toBe("pass");
  });

  it("includes policy key check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const policyCheck = report.checks.find(
      (c: Check) => c.name === "policy_keys"
    );
    expect(policyCheck).toBeDefined();
    expect(policyCheck.status).toBe("pass");
  });

  it("includes no-python-core check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const pyCheck = report.checks.find(
      (c: Check) => c.name === "no_python_core"
    );
    expect(pyCheck).toBeDefined();
    expect(pyCheck.status).toBe("pass");
  });

  it("includes PGV authority wording check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const pgvCheck = report.checks.find(
      (c: Check) => c.name === "pgv_authority_wording"
    );
    expect(pgvCheck).toBeDefined();
    expect(pgvCheck.status).toBe("pass");
  });

  it("includes tier label check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const tierCheck = report.checks.find(
      (c: Check) => c.name === "tier_labels"
    );
    expect(tierCheck).toBeDefined();
    expect(tierCheck.status).toBe("pass");
  });

  it("includes AGENTS size check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const agentsCheck = report.checks.find(
      (c: Check) => c.name === "agents_size"
    );
    expect(agentsCheck).toBeDefined();
    expect(agentsCheck.status).toBe("pass");
  });

  it("includes adapter presence check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const adapterCheck = report.checks.find(
      (c: Check) => c.name === "adapters_present"
    );
    expect(adapterCheck).toBeDefined();
    expect(adapterCheck.status).toBe("pass");
  });

  it("includes evidence scope support check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find(
      (c: Check) => c.name === "evidence_scope_support"
    );
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
  });

  it("includes component registry coverage check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find(
      (c: Check) => c.name === "component_registry"
    );
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
    expect(check.note).toContain("protected paths");
    expect(check.note).toContain("covered");
  });

  it("fails when component registry does not cover protected paths", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-doctor-"));
    try {
      for (const dir of [
        "docs",
        "templates",
        "schemas",
        "policies",
        "adapters",
        "components",
      ]) {
        fs.copySync(path.join(repoRoot, dir), path.join(tmpDir, dir));
      }
      for (const file of ["README.md", "AGENTS.md", "X_HARNESS.md"]) {
        fs.copyFileSync(path.join(repoRoot, file), path.join(tmpDir, file));
      }
      await fs.writeFile(
        path.join(tmpDir, "components", "registry.yaml"),
        `version: 1
components:
  - id: docs_only
    kind: docs
    paths:
      - docs/**
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - npm test
`,
        "utf-8"
      );

      const { stdout, exitCode } = await execaNode([
        "doctor",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(1);
      const report = JSON.parse(stdout);
      const check = report.checks.find(
        (c: Check) => c.name === "component_registry"
      );
      expect(check).toBeDefined();
      expect(check.status).toBe("fail");
      expect(check.note).toContain(
        "protected path is not registered to any component"
      );
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("includes read-only verifier check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find(
      (c: Check) => c.name === "read_only_verifier"
    );
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
  });

  it("includes no heavy runtime check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find(
      (c: Check) => c.name === "no_heavy_runtime"
    );
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
  });

  it("includes templates inventory check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find(
      (c: Check) => c.name === "templates_inventory"
    );
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
    expect(check.note).toContain("SUBAGENT_TASK_light.md");
    expect(check.note).toContain("COMPLETION_CARD.md");
  });

  it("includes context freshness check and passes when fresh", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find(
      (c: Check) => c.name === "context_freshness"
    );
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
    expect(check.note).toContain("fresh");
  });

  it("includes managed contract block freshness check", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find(
      (c: Check) => c.name === "managed_contract_blocks"
    );
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
    expect(check.note).toContain("runtime-contract");
  });

  it("fails context freshness when AGENTS.md is missing", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-doctor-"));
    try {
      const { stdout, exitCode } = await execaNode([
        "doctor",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(1);
      const report = JSON.parse(stdout);
      const check = report.checks.find(
        (c: Check) => c.name === "context_freshness"
      );
      expect(check).toBeDefined();
      expect(check.status).toBe("fail");
      expect(check.note).toContain("not found");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("fails context freshness when managed block is stale", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-doctor-"));
    try {
      await fs.writeFile(
        path.join(tmpDir, "AGENTS.md"),
        `<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- context-hash: deadbeef -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n`,
        "utf-8"
      );
      const { stdout, exitCode } = await execaNode([
        "doctor",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(1);
      const report = JSON.parse(stdout);
      const check = report.checks.find(
        (c: Check) => c.name === "context_freshness"
      );
      expect(check).toBeDefined();
      expect(check.status).toBe("fail");
      expect(check.note).toContain("stale");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("includes policy drift check and passes for current repo", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find((c: Check) => c.name === "policy_drift");
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
    expect(check.note).toContain("policy section candidate_completion present");
    expect(check.note).toContain("success_requires predicate known");
    expect(check.note).toContain(
      "evidence_floor.standard.required includes done_checklist"
    );
    expect(check.note).toContain(
      "evidence_floor.standard.required includes prediction"
    );
  });

  it("fails policy drift when admission.yaml has unknown predicates", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-doctor-"));
    try {
      const badPolicy = {
        version: 1,
        candidate_completion: {
          required: ["schema_version", "unknown_field_xyz"],
        },
        success_requires: [
          "claim.fix_status == fixed",
          "unknown.predicate == true",
        ],
        reject_success_if: {
          fix_status: ["partial"],
          unknown_reject_key: true,
        },
        outcome_mapping: {
          success: { acceptance_status: "accepted" },
          unknown_outcome: { acceptance_status: "withheld" },
        },
        evidence_floor: {
          light: {
            required: ["files_changed"],
            one_of: ["unknown_evidence_label"],
          },
          standard: {
            required: ["files_changed", "command_evidence"],
          },
          deep: {
            required: [
              "files_changed",
              "command_evidence",
              "evidence_scope_declared",
              "untested_regions_declared",
              "remaining_risks_declared",
              "execution_controls_present",
              "rollback_policy_present",
            ],
          },
        },
      };
      await fs.ensureDir(path.join(tmpDir, "policies"));
      await fs.writeFile(
        path.join(tmpDir, "policies", "admission.yaml"),
        JSON.stringify(badPolicy, null, 2),
        "utf-8"
      );
      const { stdout, exitCode } = await execaNode([
        "doctor",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(1);
      const report = JSON.parse(stdout);
      const check = report.checks.find((c: Check) => c.name === "policy_drift");
      expect(check).toBeDefined();
      expect(check.status).toBe("fail");
      expect(check.note).toContain(
        "candidate_completion.required field unknown: unknown_field_xyz"
      );
      expect(check.note).toContain(
        "success_requires predicate unknown: unknown.predicate == true"
      );
      expect(check.note).toContain(
        "reject_success_if key unknown: unknown_reject_key"
      );
      expect(check.note).toContain(
        "outcome_mapping key unknown: unknown_outcome"
      );
      expect(check.note).toContain(
        "evidence_floor.light label unknown: unknown_evidence_label"
      );
      expect(check.note).toContain(
        "evidence_floor.standard.required missing done_checklist"
      );
      expect(check.note).toContain(
        "evidence_floor.deep.required missing prediction"
      );
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("supports --policy-drift flag explicitly", async () => {
    const { stdout, exitCode } = await execaNode([
      "doctor",
      "--root",
      repoRoot,
      "--policy-drift",
    ]);
    expect(exitCode).toBe(0);
    const report = JSON.parse(stdout);
    const check = report.checks.find((c: Check) => c.name === "policy_drift");
    expect(check).toBeDefined();
    expect(check.status).toBe("pass");
    expect(check.note).toContain("[explicit]");
  });

  it("policy-drift note lacks [explicit] without the flag", async () => {
    const { stdout } = await execaNode(["doctor", "--root", repoRoot]);
    const report = JSON.parse(stdout);
    const check = report.checks.find((c: Check) => c.name === "policy_drift");
    expect(check).toBeDefined();
    expect(check.note).not.toContain("[explicit]");
  });
});
