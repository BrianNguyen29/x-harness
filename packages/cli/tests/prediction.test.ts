import { afterEach, describe, expect, it } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import * as fs from "node:fs";
import * as os from "node:os";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const tempDirs: string[] = [];

function makeTempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-prediction-"));
  tempDirs.push(dir);
  return dir;
}

async function createVerifyEpisode(
  cardRelPath: string,
  episodesDir: string,
  expectedExitCode: number
): Promise<string> {
  const cardPath = path.join(repoRoot, cardRelPath);
  const { stdout, exitCode } = await execaNode([
    "verify",
    "--card",
    cardPath,
    "--episode",
    "--episodes-dir",
    episodesDir,
    "--json",
  ]);
  expect(exitCode).toBe(expectedExitCode);
  const output = JSON.parse(stdout) as { episode: { episode_dir: string } };
  return path.resolve(repoRoot, output.episode.episode_dir);
}

afterEach(() => {
  for (const dir of tempDirs.splice(0)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
});

describe("prediction command", () => {
  describe("prediction check", () => {
    it("returns error when no card found", async () => {
      const { stderr, exitCode } = await execaNode([
        "prediction",
        "check",
        "--card",
        "nonexistent.yaml",
      ]);
      expect(exitCode).toBe(1);
      expect(stderr).toContain("No completion card found");
    });

    it("returns error when card has no prediction", async () => {
      // Create a temp directory with a card that has no prediction
      const tmpDir = path.join(process.cwd(), ".x-harness", "tmp", "pred-test");
      fs.mkdirSync(tmpDir, { recursive: true });
      const cardPath = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(
        cardPath,
        `schema_version: "1"
task_id: TEST-001
tier: standard
owner: alice
accountable: bob
claim:
  fix_status: fixed
  summary: test
  evidence: []
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

      const { stderr, exitCode } = await execaNode([
        "prediction",
        "check",
        "--card",
        cardPath,
      ]);
      expect(exitCode).toBe(1);
      expect(stderr).toContain("No prediction found");
    });

    it("validates valid prediction successfully", async () => {
      const tmpDir = path.join(
        process.cwd(),
        ".x-harness",
        "tmp",
        "pred-valid-test"
      );
      fs.mkdirSync(tmpDir, { recursive: true });
      const cardPath = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(
        cardPath,
        `schema_version: "1"
task_id: TEST-001
tier: standard
owner: alice
accountable: bob
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: Task completes successfully
  expected_effect: Tests pass
  falsification_method: Run tests
  horizon: same_verify
claim:
  fix_status: fixed
  summary: test
  evidence: []
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
        "prediction",
        "check",
        "--card",
        cardPath,
      ]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("valid");
    });

    it("detects weak prediction with missing required fields", async () => {
      const tmpDir = path.join(
        process.cwd(),
        ".x-harness",
        "tmp",
        "pred-weak-test"
      );
      fs.mkdirSync(tmpDir, { recursive: true });
      const cardPath = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(
        cardPath,
        `schema_version: "1"
task_id: TEST-001
tier: standard
owner: alice
accountable: bob
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  measurable_signal: some metric
  confidence: high
claim:
  fix_status: fixed
  summary: test
  evidence: []
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
        "prediction",
        "check",
        "--card",
        cardPath,
        "--json",
      ]);
      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(false);
      expect(output.errors).toContain(
        "prediction.claim is required and must be a non-empty string"
      );
      expect(output.errors).toContain(
        "prediction.expected_effect is required and must be a non-empty string"
      );
      expect(output.errors).toContain(
        "prediction.falsification_method is required and must be a non-empty string"
      );
      expect(output.errors).toContain("prediction.horizon is required");
    });

    it("outputs JSON when --json flag is used", async () => {
      const tmpDir = path.join(
        process.cwd(),
        ".x-harness",
        "tmp",
        "pred-json-test"
      );
      fs.mkdirSync(tmpDir, { recursive: true });
      const cardPath = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(
        cardPath,
        `schema_version: "1"
task_id: TEST-001
tier: standard
owner: alice
accountable: bob
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: Task completes successfully
  expected_effect: Tests pass
  falsification_method: Run tests
  horizon: same_verify
claim:
  fix_status: fixed
  summary: test
  evidence: []
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
        "prediction",
        "check",
        "--card",
        cardPath,
        "--json",
      ]);
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.errors).toHaveLength(0);
    });
  });

  describe("prediction verify", () => {
    it("requires an episode", async () => {
      const { stdout, exitCode } = await execaNode([
        "prediction",
        "verify",
        "--json",
      ]);
      expect(exitCode).toBe(2);
      expect(stdout).toBe("");
    });

    it("confirms a same-verify prediction when the episode is accepted", async () => {
      const episodesDir = path.join(makeTempDir(), "episodes");
      const episodeDir = await createVerifyEpisode(
        "examples/golden/regression/success-standard-scoped-evidence/completion-card.yaml",
        episodesDir,
        0
      );

      const { stdout, exitCode } = await execaNode([
        "prediction",
        "verify",
        "--episode",
        episodeDir,
        "--json",
      ]);

      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.status).toBe("confirmed");
      expect(output.reason).toBe("same_verify_episode_accepted");
      expect(output.verdict.acceptance_status).toBe("accepted");
    });

    it("falsifies a same-verify prediction when the episode is withheld", async () => {
      const episodesDir = path.join(makeTempDir(), "episodes");
      const episodeDir = await createVerifyEpisode(
        "examples/golden/capability/failed-typecheck-recovery-route/completion-card.yaml",
        episodesDir,
        1
      );

      const { stdout, exitCode } = await execaNode([
        "prediction",
        "verify",
        "--episode",
        episodeDir,
        "--json",
      ]);

      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.status).toBe("falsified");
      expect(output.reason).toBe("same_verify_episode_withheld");
      expect(output.verdict.acceptance_status).toBe("withheld");
    });
  });

  describe("prediction report", () => {
    it("counts confirmed, falsified, and inconclusive episode predictions", async () => {
      const episodesDir = path.join(makeTempDir(), "episodes");
      await createVerifyEpisode(
        "examples/golden/regression/success-standard-scoped-evidence/completion-card.yaml",
        episodesDir,
        0
      );
      await createVerifyEpisode(
        "examples/golden/capability/failed-typecheck-recovery-route/completion-card.yaml",
        episodesDir,
        1
      );

      const { stdout, exitCode } = await execaNode([
        "prediction",
        "report",
        "--episodes-dir",
        episodesDir,
        "--json",
      ]);

      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.episodes_analyzed).toBe(2);
      expect(output.confirmed).toBe(1);
      expect(output.falsified).toBe(1);
      expect(output.inconclusive).toBe(0);
    });
  });
});
