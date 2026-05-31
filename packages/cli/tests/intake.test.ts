import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import * as fs from "node:fs";

// Use three levels up to get to workspace root from packages/cli/tests
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("intake classify", () => {
  it("classifies auth/token work as high_risk/deep", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "classify",
      "--task",
      "Fix refresh token race",
      "--files",
      "src/auth/session.ts",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.intake_label).toBe("high_risk");
    expect(parsed.runtime_tier).toBe("deep");
    expect(parsed.reasoning).toContainEqual(
      expect.stringContaining("high-risk keyword: token")
    );
  });

  it("classifies comment-only changes as tiny/light", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "classify",
      "--task",
      "Update comments in auth file",
      "--files",
      "src/auth/session.ts",
      "--change",
      "comment-only",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.intake_label).toBe("tiny");
    expect(parsed.runtime_tier).toBe("light");
    expect(parsed.reasoning).toContainEqual(
      "Change signal indicates comment-only modification"
    );
  });

  it("classifies routine work as normal/standard", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "classify",
      "--task",
      "Refactor utils helper functions",
      "--files",
      "src/utils/helpers.ts",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.intake_label).toBe("normal");
    expect(parsed.runtime_tier).toBe("standard");
  });

  it("classifies CI workflow changes as high_risk/deep", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "classify",
      "--task",
      "Update CI pipeline",
      "--files",
      ".github/workflows/test.yml",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.intake_label).toBe("high_risk");
    expect(parsed.runtime_tier).toBe("deep");
    // Either the task description contains "ci" or the files include CI/CD workflows
    const hasExpectedReasoning = parsed.reasoning.some(
      (r: string) =>
        r.includes("high-risk keyword: ci") || r.includes("CI/CD workflows")
    );
    expect(hasExpectedReasoning).toBe(true);
  });

  it("is registered in help", async () => {
    const { stdout, exitCode } = await execaNode(["--help-all"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("intake");
  });

  it("intake classify --help shows options", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "classify",
      "--help",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("--task");
    expect(stdout).toContain("--files");
    expect(stdout).toContain("--change");
    expect(stdout).toContain("--json");
  });

  it("explains a declared intake tier downgrade from a card", async () => {
    const tmpDir = path.join(
      repoRoot,
      ".x-harness",
      "tmp",
      "intake-explain-downgrade"
    );
    fs.mkdirSync(tmpDir, { recursive: true });
    const cardPath = path.join(tmpDir, "completion-card.yaml");
    fs.writeFileSync(
      cardPath,
      `schema_version: "1"
task_id: TEST-INTAKE-DOWNGRADE
tier: light
owner: alice
accountable: bob
intake:
  classification: high_risk
  mapped_tier: deep
  rationale: Auth/session work is high risk
  signals:
    - auth
  auto_escalated: true
evidence:
  files_changed:
    - src/auth/session.ts
  manual_rationale: "Fixture"
claim:
  fix_status: fixed
  summary: "Fix auth session handling"
  evidence:
    - "fixture"
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
    );

    const { stdout, exitCode } = await execaNode([
      "intake",
      "explain",
      "--card",
      cardPath,
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(1);
    const parsed = JSON.parse(stdout);
    expect(parsed.ok).toBe(false);
    expect(parsed.source).toBe("declared");
    expect(parsed.tier_downgrade).toBe(true);
    expect(parsed.intervention_required).toBe(true);
    expect(parsed.errors[0]).toContain("tier downgrade");
  });

  it("explains inferred intake when a card has no intake block", async () => {
    const tmpDir = path.join(
      repoRoot,
      ".x-harness",
      "tmp",
      "intake-explain-inferred"
    );
    fs.mkdirSync(tmpDir, { recursive: true });
    const cardPath = path.join(tmpDir, "completion-card.yaml");
    fs.writeFileSync(
      cardPath,
      `schema_version: "1"
task_id: TEST-INTAKE-INFERRED
tier: deep
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/auth/session.ts
  command_evidence:
    - command: npm test
      exit_code: 0
  verification_artifacts:
    - kind: unit_test
      command: npm test
      status: passed
      verifies:
        - auth session handling
      does_not_verify:
        - production OAuth
  untested_regions:
    - production OAuth
  remaining_risks:
    - no live auth integration
  rollback_policy:
    - revert commit
  execution_controls:
    - feature flag
state:
  read_set:
    - src/auth/session.ts
  write_set:
    - src/auth/session.ts
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: "Auth session fix passes tests"
  expected_effect: "Unit tests pass"
  falsification_method: "npm test"
  horizon: same_verify
claim:
  fix_status: fixed
  summary: "Fix auth session handling"
  evidence:
    - "fixture"
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
    );

    const { stdout, exitCode } = await execaNode([
      "intake",
      "explain",
      "--card",
      cardPath,
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.source).toBe("inferred");
    expect(parsed.intake_label).toBe("high_risk");
    expect(parsed.mapped_tier).toBe("deep");
    expect(parsed.warnings[0]).toContain("no intake block");
  });
});
