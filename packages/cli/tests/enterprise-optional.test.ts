import { describe, expect, it } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { execaNode } from "../src/test-helpers.js";

const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

function tmpDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "xh-enterprise-"));
}

describe("optional enterprise controls", () => {
  it("evaluates approval risk without admission authority or personal scoring", async () => {
    const dir = tmpDir();
    try {
      const cardPath = path.join(dir, "completion-card.yaml");
      fs.writeFileSync(
        cardPath,
        `task_id: TASK-APPROVAL-RISK-001
tier: deep
evidence:
  files_changed:
    - schemas/completion-card.schema.json
`,
        "utf-8"
      );

      const { stdout, exitCode } = await execaNode([
        "approval-risk",
        "evaluate",
        "--card",
        cardPath,
        "--root",
        repoRoot,
        "--json",
      ]);

      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.task_id).toBe("TASK-APPROVAL-RISK-001");
      expect(output.risk_class).toBe("critical");
      expect(output.personal_scoring).toBe(false);
      expect(output.admission_authority).toBe(false);
      expect(output.signals).toContain("human_only_path");
      expect(output.signals).toContain("missing_governance_approval");
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  it("builds and reports advisory agent profiles from benchmark output", async () => {
    const dir = tmpDir();
    try {
      const benchmarkPath = path.join(dir, "benchmark-report.json");
      fs.writeFileSync(
        benchmarkPath,
        JSON.stringify(
          {
            metrics: {
              false_accept_count: 1,
              adversarial_false_accept_count: 1,
              false_reject_count: 0,
            },
            integration: {
              note: "stale context and evidence scope mismatch observed",
            },
          },
          null,
          2
        ),
        "utf-8"
      );

      const update = await execaNode([
        "agent-profile",
        "update",
        "--agent",
        "agent@test",
        "--from-benchmark",
        benchmarkPath,
        "--root",
        dir,
        "--json",
      ]);

      expect(update.exitCode).toBe(0);
      const updateOutput = JSON.parse(update.stdout);
      expect(updateOutput.ok).toBe(true);
      expect(updateOutput.profile.advisory_only).toBe(true);
      expect(updateOutput.profile.admission_authority).toBe(false);
      expect(updateOutput.profile.observed_failure_modes).toContain(
        "false_accept_regression"
      );
      expect(updateOutput.profile.observed_failure_modes).toContain(
        "adversarial_false_accept"
      );
      expect(updateOutput.profile.required_extra_checks).toContain(
        "adversarial_replay_required"
      );

      const report = await execaNode([
        "agent-profile",
        "report",
        "--agent",
        "agent@test",
        "--root",
        dir,
        "--json",
      ]);

      expect(report.exitCode).toBe(0);
      const reportOutput = JSON.parse(report.stdout);
      expect(reportOutput.agent_id).toBe("agent@test");
      expect(reportOutput.admission_authority).toBe(false);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  it("reports cost budget overages as advisory when policy is disabled", async () => {
    const { stdout, exitCode } = await execaNode([
      "cost",
      "check",
      "--actual-usd",
      "99",
      "--input-tokens",
      "1",
      "--output-tokens",
      "1",
      "--root",
      repoRoot,
      "--json",
    ]);

    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.over_budget).toBe(true);
    expect(output.policy_enabled).toBe(false);
    expect(output.enforcement_enabled).toBe(false);
    expect(output.admission_authority).toBe(false);
  });

  it("can fail an external budget check only when policy enforcement is enabled", async () => {
    const dir = tmpDir();
    try {
      fs.mkdirSync(path.join(dir, "policies"), { recursive: true });
      fs.writeFileSync(
        path.join(dir, "policies", "cost-budget.yaml"),
        `version: 1
cost_budget:
  enabled: true
  max_usd: 1
  max_input_tokens: 10
  max_output_tokens: 10
  over_budget_recovery: escalate_to_human
  affects_admission: false
`,
        "utf-8"
      );

      const { stdout, exitCode } = await execaNode([
        "cost",
        "check",
        "--actual-usd",
        "2",
        "--input-tokens",
        "1",
        "--output-tokens",
        "1",
        "--root",
        dir,
        "--enforce",
        "--json",
      ]);

      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.over_budget).toBe(true);
      expect(output.enforcement_enabled).toBe(true);
      expect(output.admission_authority).toBe(false);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});
