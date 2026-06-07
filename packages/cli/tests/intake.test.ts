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

describe("intake contract", () => {
  it("rejects missing --id with usage error", async () => {
    const { stderr, exitCode } = await execaNode([
      "intake",
      "contract",
      "--goal",
      "x",
      "--acceptance",
      "y",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--id is required");
  });

  it("rejects missing --goal with usage error", async () => {
    const { stderr, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "x",
      "--acceptance",
      "y",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--goal is required");
  });

  it("rejects missing --acceptance with usage error", async () => {
    const { stderr, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "x",
      "--goal",
      "y",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--acceptance is required");
  });

  it("emits YAML to stdout by default", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "intake-lite",
      "--goal",
      "ship the safe V1 slice",
      "--visible",
      "true",
      "--non-goal",
      "block admission",
      "--non-goal",
      "add new admission predicate",
      "--acceptance",
      "advisory note emitted on standard",
      "--acceptance",
      "no --from flag is added",
      "--protected-behavior",
      "intent_ref is never required",
      "--ambiguity",
      "none",
      "--note",
      "first vertical slice",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("id: intake-lite");
    expect(stdout).toContain("product_goal: ship the safe V1 slice");
    expect(stdout).toContain("user_visible_change: true");
    expect(stdout).toContain("- block admission");
    expect(stdout).toContain("- add new admission predicate");
    expect(stdout).toContain("- id: ac-1");
    expect(stdout).toContain("statement: advisory note emitted on standard");
    expect(stdout).toContain("- id: ac-2");
    expect(stdout).toContain("statement: no --from flag is added");
    expect(stdout).toContain("- intent_ref is never required");
    expect(stdout).toContain("status: none");
    expect(stdout).toContain("notes: first vertical slice");
  });

  it("emits JSON to stdout with --json", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "intake-lite",
      "--goal",
      "ship the safe V1 slice",
      "--visible",
      "false",
      "--acceptance",
      "advisory note emitted on standard",
      "--ambiguity",
      "partial",
      "--ambiguity-question",
      "Should intent_ref be deep-only?",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.schema_version).toBe("1");
    expect(parsed.id).toBe("intake-lite");
    expect(parsed.product_goal).toBe("ship the safe V1 slice");
    expect(parsed.user_visible_change).toBe(false);
    expect(parsed.ambiguity.status).toBe("partial");
    expect(parsed.ambiguity.questions).toContain(
      "Should intent_ref be deep-only?"
    );
    expect(parsed.acceptance_criteria[0].id).toBe("ac-1");
    expect(parsed.acceptance_criteria[0].statement).toBe(
      "advisory note emitted on standard"
    );
  });

  it("writes the record to --output and emits nothing on stdout", async () => {
    const tmpDir = path.join(
      repoRoot,
      ".x-harness",
      "tmp",
      "intake-contract-output"
    );
    fs.mkdirSync(tmpDir, { recursive: true });
    const out = path.join(tmpDir, "intent.yaml");
    const { stdout, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "intake-lite",
      "--goal",
      "ship the safe V1 slice",
      "--acceptance",
      "advisory note emitted on standard",
      "--output",
      out,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toBe("");
    const data = fs.readFileSync(out, "utf-8");
    expect(data).toContain("id: intake-lite");
    expect(data).toContain("product_goal: ship the safe V1 slice");
    expect(data).toContain("statement: advisory note emitted on standard");
  });

  it("fails when --output parent directory does not exist", async () => {
    const tmpDir = path.join(
      repoRoot,
      ".x-harness",
      "tmp",
      "intake-contract-missing-parent"
    );
    fs.mkdirSync(tmpDir, { recursive: true });
    const out = path.join(tmpDir, "does", "not", "exist", "intent.yaml");
    const { stderr, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "intake-lite",
      "--goal",
      "ship the safe V1 slice",
      "--acceptance",
      "advisory note emitted on standard",
      "--output",
      out,
    ]);
    expect(exitCode).toBe(1);
    expect(stderr).toContain("parent directory does not exist");
  });

  it("rejects invalid --visible", async () => {
    const { stderr, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "x",
      "--goal",
      "y",
      "--acceptance",
      "z",
      "--visible",
      "maybe",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--visible");
  });

  it("rejects invalid --ambiguity", async () => {
    const { stderr, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "x",
      "--goal",
      "y",
      "--acceptance",
      "z",
      "--ambiguity",
      "maybe",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--ambiguity");
  });

  it("accepts comma-delimited list values", async () => {
    const { stdout, exitCode } = await execaNode([
      "intake",
      "contract",
      "--id",
      "intake-lite",
      "--goal",
      "ship",
      "--non-goal",
      "a, b, c",
      "--acceptance",
      "x, y",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.non_goals).toEqual(["a", "b", "c"]);
    expect(parsed.acceptance_criteria).toHaveLength(2);
    expect(parsed.acceptance_criteria[0].statement).toBe("x");
    expect(parsed.acceptance_criteria[1].statement).toBe("y");
  });
});

describe("intake handoff", () => {
  function uniqueHandoffDir(label: string): string {
    const tmpDir = path.join(
      repoRoot,
      ".x-harness",
      "tmp",
      `intake-handoff-${label}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    );
    fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
    const policyContent = `version: 1
intake_labels:
  tiny:
    runtime_tier: light
    signals:
      - comment_only
  normal:
    runtime_tier: standard
    signals:
      - routine_implementation
  high_risk:
    runtime_tier: deep
    signals:
      - auth
high_risk_signals:
  auth:
    description: Auth changes
    examples:
      - login
runtime_tier_confirmation:
  tiers: [light, standard, deep]
  note: Tiers remain light, standard, deep.
`;
    fs.writeFileSync(
      path.join(tmpDir, "policies", "intake.yaml"),
      policyContent
    );
    return tmpDir;
  }

  it("rejects missing --tier with usage error", async () => {
    const { stderr, exitCode } = await execaNode([
      "intake",
      "handoff",
      "--task",
      "fix bug",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--tier is required");
  });

  it("rejects explicit tier with safe V1 message", async () => {
    const tmpDir = uniqueHandoffDir("explicit");
    const { stderr, exitCode } = await execaNode([
      "intake",
      "handoff",
      "--tier",
      "standard",
      "--root",
      tmpDir,
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("safe V1");
  });

  it("emits auto handoff text for a routine task", async () => {
    const tmpDir = uniqueHandoffDir("normal");
    const { stdout, exitCode } = await execaNode([
      "intake",
      "handoff",
      "--tier",
      "auto",
      "--task",
      "fix bug in formatter",
      "--file",
      "src/formatter.go",
      "--root",
      tmpDir,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Selected tier: standard");
    expect(stdout).toContain("Intake label: normal");
    expect(stdout).toContain("Suggested next: xh handoff standard");
  });

  it("emits auto handoff text for a high-risk task", async () => {
    const tmpDir = uniqueHandoffDir("high-risk");
    const { stdout, exitCode } = await execaNode([
      "intake",
      "handoff",
      "--tier",
      "auto",
      "--task",
      "update auth logic",
      "--root",
      tmpDir,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Selected tier: deep");
    expect(stdout).toContain("Intake label: high_risk");
    expect(stdout).toContain("Auto escalated: yes");
  });

  it("emits JSON for auto handoff", async () => {
    const tmpDir = uniqueHandoffDir("json");
    const { stdout, exitCode } = await execaNode([
      "intake",
      "handoff",
      "--tier",
      "auto",
      "--task",
      "fix bug in formatter",
      "--root",
      tmpDir,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.selected_tier).toBe("standard");
    expect(parsed.intake_label).toBe("normal");
    expect(parsed.command_suggestion).toContain("xh handoff standard");
    expect(parsed.command_suggestion).toContain("fix bug in formatter");
  });

  it("fails with usage error when intake policy is missing", async () => {
    const tmpDir = path.join(
      repoRoot,
      ".x-harness",
      "tmp",
      `intake-handoff-no-policy-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    );
    fs.mkdirSync(tmpDir, { recursive: true });
    const { stderr, exitCode } = await execaNode([
      "intake",
      "handoff",
      "--tier",
      "auto",
      "--root",
      tmpDir,
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("intake.yaml not found");
  });
});
