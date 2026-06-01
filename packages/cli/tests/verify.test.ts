import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import { readTrace } from "../src/core/trace.js";
import * as path from "node:path";
import * as fs from "node:fs";
import * as os from "node:os";
import { fileURLToPath } from "node:url";
import { spawn, execFile } from "node:child_process";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const cliDir = path.join(repoRoot, "packages", "cli");

function execaNodeCwd(
  args: string[],
  cwd: string
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    const script = path.join(cliDir, "dist", "index.js");
    execFile("node", [script, ...args], { cwd }, (error, stdout, stderr) => {
      resolve({
        stdout: stdout.trim(),
        stderr: stderr.trim(),
        exitCode: error?.code ? Number(error.code) : 0,
      });
    });
  });
}

describe("verify command", () => {
  it("accepts a passing fixture via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim",
      "tests/fixtures/claim-pass.yaml",
      "--evidence",
      "tests/fixtures/evidence-pass.yaml",
      "--subagent-return",
      "tests/fixtures/subagent-pass.yaml",
      "--tier",
      "standard",
      "--task-id",
      "TASK-001",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(true);
    expect(event.acceptance_status).toBe("accepted");
  });

  it("maps subagent-return evidence and handoff in compatibility mode", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--subagent-return",
      "tests/fixtures/subagent-pass.yaml",
      "--tier",
      "standard",
      "--task-id",
      "TASK-SUBAGENT-ONLY",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(true);
    expect(event.acceptance_status).toBe("accepted");
  });

  it("withholds legacy evidence-only input without fix and verification status", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--evidence",
      "tests/fixtures/evidence-pass.yaml",
      "--tier",
      "light",
      "--task-id",
      "TASK-EVIDENCE-ONLY",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
    expect(event.withheld_reason).toContain(
      "claim.fix_status or result.fix_status is required"
    );
    expect(event.withheld_reason).toContain("verification.status is required");
  });

  it("withholds claim and evidence legacy input without verification status", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim",
      "tests/fixtures/claim-pass.yaml",
      "--evidence",
      "tests/fixtures/evidence-pass.yaml",
      "--tier",
      "light",
      "--task-id",
      "TASK-CLAIM-EVIDENCE-ONLY",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
    expect(event.withheld_reason).toContain("verification.status is required");
  });

  it("withholds on canonical contradiction via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--subagent-return",
      "tests/fixtures/subagent-contradiction.yaml",
      "--tier",
      "light",
      "--task-id",
      "TASK-002",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
    expect(event.withheld_reason).toContain("canonical contradiction");
  });

  it("withholds when standard tier lacks evidence via legacy flags", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim",
      "tests/fixtures/claim-pass.yaml",
      "--tier",
      "standard",
      "--task-id",
      "TASK-003",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.acceptance_status).toBe("withheld");
  });

  it("accepts a completion card via --card", async () => {
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
    expect(output.acceptance_status).toBe("accepted");
  });

  it("withholds a blocked completion card via --card", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "04-blocked-verification",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.acceptance_status).toBe("withheld");
  });

  it("withholds intake tier downgrade via --card", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "regression",
      "blocked-tier-downgrade",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.acceptance_status).toBe("withheld");
    expect(output.recovery.predicate).toBe("Fintervention");
    expect(output.recovery.owner).toBe("implementation-worker");
    expect(output.withheld_reason).toContain("tier downgrade");
  });

  it("withholds adversarial evidence with non-zero command exit code via verify gate", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "adversarial",
      "lying-command-exit-code",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--strict",
      "--governance-enforced",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.acceptance_status).toBe("withheld");
    expect(output.withheld_reason).toContain("non-zero exit_code");
  });

  it("withholds dangerous evidence commands via verify gate", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "adversarial",
      "hidden-dangerous-command",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--strict",
      "--governance-enforced",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.acceptance_status).toBe("withheld");
    expect(output.recovery.predicate).toBe("Fpermission");
    expect(output.withheld_reason).toContain("shell metacharacter");
  });

  it("withholds strict standard cards missing evidence provenance", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-prov-"));
    try {
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );
      fs.writeFileSync(
        path.join(tmpDir, "completion-card.yaml"),
        `schema_version: "1"
task_id: TASK-STRICT-PROVENANCE-001
tier: standard
owner: alice
accountable: bob
context_acknowledged: true
state:
  read_set:
    - completion-card.yaml
  write_set:
    - src/example.ts
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  evidence_attached: true
prediction:
  claim: Strict provenance missing runner should withhold.
  expected_effect: verify --strict exits non-zero.
  falsification_method: Run the strict verify command.
  horizon: same_verify
evidence:
  files_changed:
    - src/example.ts
  command_evidence:
    - command: npm run test
      exit_code: 0
  verification_artifacts:
    - kind: unit_test
      command: npm run test
      status: passed
      exit_code: 0
      runner: local-vitest
      started_at: "2026-05-25T00:00:00.000Z"
claim:
  fix_status: fixed
  summary: Strict provenance test card.
  evidence:
    - description: unit tests passed
verification:
  status: passed
  checks:
    - name: unit
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`,
        "utf-8"
      );
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });
      execFileSync("git", ["add", "."], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", "completion-card.yaml", "--strict", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(false);
      expect(output.recovery.predicate).toBe("evidence_provenance_missing");
      expect(output.withheld_reason).toContain(
        "strict evidence provenance requires evidence.command_evidence[0].runner"
      );
      expect(output.withheld_reason).toContain(
        "strict evidence provenance requires evidence.command_evidence[0].started_at"
      );
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("withholds governance-enforced verification when strict diff files are missing from evidence", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-diff-gov-"));
    try {
      const { execFileSync } = await import("node:child_process");
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.mkdirSync(path.join(tmpDir, "schemas"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );
      fs.copyFileSync(
        path.join(repoRoot, "policies", "authority.yaml"),
        path.join(tmpDir, "policies", "authority.yaml")
      );
      fs.writeFileSync(
        path.join(tmpDir, "schemas", "completion-card.schema.json"),
        "{}\n",
        "utf-8"
      );
      fs.writeFileSync(
        path.join(tmpDir, "completion-card.yaml"),
        `schema_version: "1"
task_id: TASK-DIFF-GOV-001
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
  checks:
    - name: basic
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`,
        "utf-8"
      );
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });
      execFileSync("git", ["add", "."], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });
      fs.writeFileSync(
        path.join(tmpDir, "schemas", "completion-card.schema.json"),
        '{ "changed": true }\n',
        "utf-8"
      );

      const { stdout, exitCode } = await execaNodeCwd(
        [
          "verify",
          "--card",
          "completion-card.yaml",
          "--governance-enforced",
          "--diff",
          "HEAD",
          "--changed-files-source",
          "strict",
          "--json",
        ],
        tmpDir
      );
      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(false);
      expect(output.changed_files.source).toBe("strict");
      expect(output.changed_files.git_files).toContain(
        "schemas/completion-card.schema.json"
      );
      expect(output.withheld_reason).toContain(
        "evidence.files_changed missing git diff file"
      );
      expect(output.withheld_reason).toContain(
        "governance permission violation"
      );
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("prints quiet output by default", async () => {
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
    ]);
    expect(exitCode).toBe(0);
    const lines = stdout.split("\n").filter((l) => l.trim().length > 0);
    expect(lines.length).toBeLessThanOrEqual(3);
    expect(stdout).toContain("outcome: success");
    expect(stdout).toContain("acceptance_status: accepted");
  });

  it("prints failed quiet output with error info", async () => {
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--claim",
      "tests/fixtures/claim-pass.yaml",
      "--tier",
      "standard",
      "--task-id",
      "TASK-FAIL-QUIET",
    ]);
    expect(exitCode).toBe(1);
    const lines = stdout.split("\n").filter((l) => l.trim().length > 0);
    expect(lines.length).toBeLessThanOrEqual(4);
    expect(stdout).toContain("outcome:");
    expect(stdout).toContain("acceptance_status:");
    expect(stdout).toContain("checks:");
  });

  it("prints verbose output with --verbose", async () => {
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
      "--verbose",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("ACCEPTED");
    expect(stdout).toContain("Tier: light");
  });

  it("prints withheld verbose output for blocked card with --verbose", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "04-blocked-verification",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--verbose",
    ]);
    expect(exitCode).toBe(1);
    expect(stdout).toContain("WITHHELD");
    expect(stdout).toContain("Handoff:");
  });

  it("passes with --mutation-guard when no mutations occur", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-ok-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--mutation-guard", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("enables mutation guard through --strict", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-strict-ok-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--strict", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.strict).toBe(true);
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("uses mutation guard fallback for strict verification in non-git workspaces", async () => {
    const tmpDir = fs.mkdtempSync(
      path.join(os.tmpdir(), "verify-strict-nogit-")
    );
    try {
      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--strict", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.strict).toBe(true);
      expect(output.admission_outcome).toBe("success");
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard passed")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("blocks unexpected mutations through --strict in non-git workspaces", async () => {
    const script = path.join(cliDir, "dist", "index.js");
    const tmpDir = fs.mkdtempSync(
      path.join(os.tmpdir(), "verify-strict-nongit-mg-")
    );
    try {
      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );

      const injectPath = path.join(tmpDir, "unexpected.txt");
      const child = spawn(
        "node",
        [script, "verify", "--card", cardDst, "--strict", "--json"],
        {
          cwd: tmpDir,
          env: {
            ...process.env,
            X_HARNESS_ENABLE_TEST_HOOKS: "1",
            X_HARNESS_TEST_INJECT_MUTATION: injectPath,
          },
        }
      );

      let stdout = "";
      let childExited = false;
      let exitCode = 1;
      child.stdout.on("data", (chunk: Buffer) => {
        stdout += chunk.toString();
      });
      child.on("close", (code) => {
        childExited = true;
        exitCode = code ?? 1;
      });

      await new Promise<void>((resolve) => {
        const interval = setInterval(() => {
          if (childExited) {
            clearInterval(interval);
            resolve();
          }
        }, 50);
      });

      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.admission_outcome).toBe("blocked");
      expect(output.recovery.predicate).toBe("verifier_not_read_only");
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard blocked")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("blocks non-allowlisted trace dirs with --strict", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-strict-tr-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const traceDir = path.join(tmpDir, "trace-out");
      const { stdout, exitCode } = await execaNodeCwd(
        [
          "verify",
          "--card",
          cardDst,
          "--strict",
          "--trace",
          "--trace-dir",
          traceDir,
          "--json",
        ],
        tmpDir
      );
      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.admission_outcome).toBe("blocked");
      expect(output.recovery.predicate).toBe("verifier_not_read_only");
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("trace directory is not allowlisted")
        )
      ).toBe(true);
      expect(fs.existsSync(path.join(traceDir, "events.jsonl"))).toBe(false);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("blocks unexpected mutations through --strict", async () => {
    const script = path.join(cliDir, "dist", "index.js");
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-strict-mg-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], {
        cwd: tmpDir,
      });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const injectPath = path.join(tmpDir, "unexpected.txt");
      const child = spawn(
        "node",
        [script, "verify", "--card", cardDst, "--strict", "--json"],
        {
          cwd: tmpDir,
          env: {
            ...process.env,
            X_HARNESS_ENABLE_TEST_HOOKS: "1",
            X_HARNESS_TEST_INJECT_MUTATION: injectPath,
          },
        }
      );

      let stdout = "";
      let childExited = false;
      let exitCode = 1;
      child.stdout.on("data", (chunk: Buffer) => {
        stdout += chunk.toString();
      });
      child.on("close", (code) => {
        childExited = true;
        exitCode = code ?? 1;
      });

      await new Promise<void>((resolve) => {
        const interval = setInterval(() => {
          if (childExited) {
            clearInterval(interval);
            resolve();
          }
        }, 50);
      });

      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.admission_outcome).toBe("blocked");
      expect(output.recovery.predicate).toBe("verifier_not_read_only");
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard blocked")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("passes with --mutation-guard when pre-existing dirty file exists", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-dirty-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      // Create a dirty untracked file before verify
      fs.writeFileSync(path.join(tmpDir, "dirty.tmp"), "dirty content");

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--mutation-guard", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("allowlists trace writes with --mutation-guard --trace", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-tr-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--mutation-guard", "--trace", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("keeps prior behavior without --mutation-guard", async () => {
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
    expect(output.acceptance_status).toBe("accepted");
    // Should not include mutation guard notes when flag is absent
    expect(stdout).not.toContain("mutation guard");
  });

  it("records blocked trace when mutation violation occurs", async () => {
    const script = path.join(cliDir, "dist", "index.js");
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-viol-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], {
        cwd: tmpDir,
      });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const injectPath = path.join(tmpDir, "unexpected.txt");
      const child = spawn(
        "node",
        [
          script,
          "verify",
          "--card",
          cardDst,
          "--mutation-guard",
          "--trace",
          "--json",
        ],
        {
          cwd: tmpDir,
          env: {
            ...process.env,
            X_HARNESS_ENABLE_TEST_HOOKS: "1",
            X_HARNESS_TEST_INJECT_MUTATION: injectPath,
          },
        }
      );

      let stdout = "";
      let childExited = false;
      let exitCode = 1;
      child.stdout.on("data", (chunk: Buffer) => {
        stdout += chunk.toString();
      });
      child.on("close", (code) => {
        childExited = true;
        exitCode = code ?? 1;
      });

      await new Promise<void>((resolve) => {
        const interval = setInterval(() => {
          if (childExited) {
            clearInterval(interval);
            resolve();
          }
        }, 50);
      });

      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.admission_outcome).toBe("blocked");
      expect(output.acceptance_status).toBe("withheld");
      expect(output.recovery).not.toBeNull();
      expect(output.recovery.predicate).toBe("verifier_not_read_only");
      expect(output.recovery.owner).toBe("admission-verifier");
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard blocked")
        )
      ).toBe(true);

      const traceDir = path.join(tmpDir, ".x-harness", "traces");
      const events = await readTrace(traceDir);
      expect(events.length).toBeGreaterThan(0);
      const lastEvent = events[events.length - 1];
      expect(lastEvent.outcome).toBe("blocked");
      expect(lastEvent.blocking_predicate).toBe("verifier_not_read_only");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("blocks admission when --stale-ground is set", async () => {
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
      "--stale-ground",
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const event = JSON.parse(stdout);
    expect(event.ok).toBe(false);
    expect(event.admission_outcome).toBe("blocked");
    expect(event.acceptance_status).toBe("withheld");
    expect(event.withheld_reason).toContain("stale_ground");
  });

  it("blocks admission when completion card has stale_ground: true", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-stale-"));
    try {
      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      let cardContent = fs.readFileSync(cardSrc, "utf-8");
      // Add stale_ground: true at the top of the card
      cardContent = "stale_ground: true\n" + cardContent;
      fs.writeFileSync(cardDst, cardContent, "utf-8");

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--json"],
        tmpDir
      );
      expect(exitCode).toBe(1);
      const event = JSON.parse(stdout);
      expect(event.ok).toBe(false);
      expect(event.admission_outcome).toBe("blocked");
      expect(event.acceptance_status).toBe("withheld");
      expect(event.withheld_reason).toContain("stale_ground");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("rejects mutation injection path outside cwd", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-mg-safety-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], {
        cwd: tmpDir,
      });

      const cardSrc = path.join(
        repoRoot,
        "examples",
        "00-minimal",
        "completion-card.yaml"
      );
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.copyFileSync(cardSrc, cardDst);
      execFileSync("git", ["add", "completion-card.yaml"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      // Inject to a path OUTSIDE cwd - should be rejected
      const outsidePath = path.join(os.tmpdir(), "should-not-be-created.txt");
      if (fs.existsSync(outsidePath)) fs.unlinkSync(outsidePath);

      const script = path.join(cliDir, "dist", "index.js");
      const child = spawn(
        "node",
        [
          script,
          "verify",
          "--card",
          cardDst,
          "--mutation-guard",
          "--trace",
          "--json",
        ],
        {
          cwd: tmpDir,
          env: {
            ...process.env,
            X_HARNESS_ENABLE_TEST_HOOKS: "1",
            X_HARNESS_TEST_INJECT_MUTATION: outsidePath,
          },
        }
      );

      let stdout = "";
      let exitCode = 1;
      child.stdout.on("data", (chunk) => {
        stdout += chunk.toString();
      });
      await new Promise<void>((resolve) => {
        child.on("close", (code) => {
          exitCode = code ?? 1;
          resolve();
        });
      });

      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      // Verify the outside path was NOT created
      expect(fs.existsSync(outsidePath)).toBe(false);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("auto-enables mutation guard for standard tier without flag", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-std-mg-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardYAML = `schema_version: "1"
task_id: TASK-TS-STD-001
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
  claim: Standard tier TS test
  expected_effect: Works
  measurable_signal: npm test
  falsification_method: Skip fix
  horizon: same_verify
evidence:
  files_changed:
    - src/main.ts
  command_evidence:
    - command: npm test
      exit_code: 0
claim:
  fix_status: fixed
  summary: Standard tier TS test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`;
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(cardDst, cardYAML, "utf-8");
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );
      execFileSync("git", ["add", "."], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard passed")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("auto-enables mutation guard for deep tier without flag", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-deep-mg-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardYAML = `schema_version: "1"
task_id: TASK-TS-DEEP-001
tier: deep
owner: alice
accountable: bob
state:
  read_set:
    - src/main.ts
  write_set:
    - src/main.ts
evidence:
  files_changed:
    - src/main.ts
  command_evidence:
    - command: npm test
      exit_code: 0
  verification_artifacts:
    - kind: unit_test
      command: npm test
      status: passed
      verifies:
        - "basic functionality"
      does_not_verify:
        - "edge cases"
      confidence: medium
  untested_regions:
    - "No integration tests."
  remaining_risks:
    - "May fail in production."
  rollback_policy:
    - "Revert commit."
  execution_controls:
    - "Deploy behind feature flag."
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: Deep tier TS test
  expected_effect: Works
  measurable_signal: npm test
  falsification_method: Skip fix
  horizon: same_verify
claim:
  fix_status: fixed
  summary: Deep tier TS test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`;
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(cardDst, cardYAML, "utf-8");
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );
      execFileSync("git", ["add", "."], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard passed")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("does not auto-enable mutation guard for light tier without flag", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-light-mg-"));
    try {
      const cardYAML = `schema_version: "1"
task_id: TASK-TS-LIGHT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.ts
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: Light tier TS test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`;
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(cardDst, cardYAML, "utf-8");
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(stdout).not.toContain("mutation guard");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("explicit --mutation-guard still enables for light tier", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "verify-light-ex-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const cardYAML = `schema_version: "1"
task_id: TASK-TS-LIGHT-EX-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.ts
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: Light tier TS test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`;
      const cardDst = path.join(tmpDir, "completion-card.yaml");
      fs.writeFileSync(cardDst, cardYAML, "utf-8");
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpDir, "policies", "admission.yaml")
      );
      execFileSync("git", ["add", "."], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const { stdout, exitCode } = await execaNodeCwd(
        ["verify", "--card", cardDst, "--mutation-guard", "--json"],
        tmpDir
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(
        output.checks.some((c: { note: string }) =>
          c.note.includes("mutation guard passed")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});
