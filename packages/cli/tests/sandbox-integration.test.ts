import { describe, expect, it } from "vitest";
import { execFile } from "node:child_process";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const packageRoot = path.resolve(path.join(__dirname, ".."));
const repoRoot = path.resolve(path.join(packageRoot, "..", ".."));
const cliPath = path.join(packageRoot, "dist", "index.js");

type RunResult = {
  stdout: string;
  stderr: string;
  exitCode: number;
};

function run(
  file: string,
  args: string[],
  cwd: string,
  env: NodeJS.ProcessEnv = {}
): Promise<RunResult> {
  return new Promise((resolve) => {
    execFile(
      file,
      args,
      {
        cwd,
        env: { ...process.env, ...env },
        maxBuffer: 20 * 1024 * 1024,
      },
      (error, stdout, stderr) => {
        resolve({
          stdout: stdout.trim(),
          stderr: stderr.trim(),
          exitCode: error?.code ? Number(error.code) : 0,
        });
      }
    );
  });
}

function xh(
  args: string[],
  cwd: string,
  env?: NodeJS.ProcessEnv
): Promise<RunResult> {
  return run(process.execPath, [cliPath, ...args], cwd, env);
}

function makeSandbox(prefix = "xh-real-sandbox-"): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

function writeFile(root: string, relativePath: string, content: string): void {
  const target = path.join(root, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, content, "utf-8");
}

function readJSON<T = Record<string, unknown>>(text: string): T {
  return JSON.parse(text) as T;
}

async function initGit(root: string): Promise<void> {
  expect((await run("git", ["init"], root)).exitCode).toBe(0);
  expect(
    (await run("git", ["config", "user.email", "sandbox@example.test"], root))
      .exitCode
  ).toBe(0);
  expect(
    (await run("git", ["config", "user.name", "Sandbox Tester"], root)).exitCode
  ).toBe(0);
}

async function commitAll(
  root: string,
  message = "sandbox baseline"
): Promise<void> {
  expect((await run("git", ["add", "."], root)).exitCode).toBe(0);
  const commit = await run("git", ["commit", "-m", message], root);
  if (commit.exitCode !== 0) {
    expect(commit.stderr + commit.stdout).toContain("nothing to commit");
  }
}

async function initHarness(
  root: string,
  profile: "standard" | "deep" = "deep"
): Promise<void> {
  const init = await xh(
    ["init", root, "--profile", profile, "--merge"],
    repoRoot
  );
  expect(init.exitCode, init.stderr || init.stdout).toBe(0);
}

function writePackageProject(root: string): void {
  writeFile(
    root,
    "package.json",
    JSON.stringify(
      {
        name: "xh-sandbox-app",
        version: "0.0.0",
        private: true,
        type: "module",
        scripts: {
          test: "node test/smoke.test.mjs",
          typecheck: "node -c src/ui/app.js",
        },
      },
      null,
      2
    ) + "\n"
  );
  writeFile(
    root,
    "src/ui/app.js",
    `import { publicQuery } from "../internal/db/public/query.js";

export function renderUser(id) {
  return publicQuery(id).name;
}
`
  );
  writeFile(
    root,
    "src/internal/db/public/query.js",
    `export function publicQuery(id) {
  return { id, name: "Ada" };
}
`
  );
  writeFile(
    root,
    "src/internal/db/secret.js",
    `export function secretQuery() {
  return { token: "secret" };
}
`
  );
  writeFile(
    root,
    "test/smoke.test.mjs",
    `import { renderUser } from "../src/ui/app.js";

if (renderUser("u-1") !== "Ada") {
  throw new Error("unexpected render output");
}
`
  );
  writeFile(
    root,
    "README.md",
    "# x-harness sandbox app\n\nSmall throwaway app used by integration tests.\n"
  );
}

function writeStandardCard(
  root: string,
  relativePath: string,
  options: {
    taskId?: string;
    filesChanged?: string[];
    boundaryApprovals?: string;
    command?: string;
  } = {}
): void {
  const taskId = options.taskId ?? "TASK-SANDBOX-STANDARD";
  const command = options.command ?? "npm test";
  const filesChanged = options.filesChanged ?? [
    "src/ui/app.js",
    "test/smoke.test.mjs",
  ];
  const boundaryApprovals = options.boundaryApprovals
    ? `\nboundary_approvals:\n${options.boundaryApprovals}`
    : "";
  writeFile(
    root,
    relativePath,
    `schema_version: "1"
task_id: ${taskId}
tier: standard
owner: sandbox-agent
accountable: sandbox-maintainer
context_acknowledged: true
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - README.md
  architecture_refs: []
  decision_refs: []
  test_matrix_refs:
    - test/smoke.test.mjs
  unresolved_context_questions: []
  context_evidence: []
intake:
  classification: normal
  mapped_tier: standard
  rationale: Sandbox integration work changes application code and tests.
  signals:
    - routine_implementation
  negative_signals_considered:
    - auth
    - token
    - release
  auto_escalated: false
state:
  read_set:
    - README.md
    - ${relativePath}
  write_set:
${filesChanged.map((file) => `    - ${file}`).join("\n")}
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
  notes:
    - Sandbox app smoke command ran successfully.
prediction:
  claim: Sandbox changes should pass strict verify without mutating the workspace.
  expected_effect: x-harness strict verify exits 0 for this card.
  measurable_signal: ${command}
  falsification_method: Run the command and strict verify in the sandbox.
  horizon: same_verify
  confidence: high
evidence:
  files_changed:
${filesChanged.map((file) => `    - ${file}`).join("\n")}
  command_evidence:
    - command: ${command}
      exit_code: 0
      runner: local-sandbox
      started_at: "2026-06-01T00:00:00.000Z"
      ended_at: "2026-06-01T00:00:01.000Z"
  verification_artifacts:
    - kind: unit_test
      command: ${command}
      status: passed
      exit_code: 0
      runner: local-sandbox
      started_at: "2026-06-01T00:00:00.000Z"
      ended_at: "2026-06-01T00:00:01.000Z"
      verifies:
        - sandbox app smoke path works
        - changed files are covered by command evidence
      does_not_verify:
        - external services
      confidence: high
  untested_regions:
    - external services
claim:
  fix_status: fixed
  summary: Sandbox implementation is complete.
  evidence:
    - description: Sandbox smoke command passed.
      command: ${command}
verification:
  status: passed
  checks:
    - name: smoke
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: sandbox-agent${boundaryApprovals}
`
  );
}

function writeIntervention(
  root: string,
  relativePath: string,
  authorizer = "maintainer"
): void {
  writeFile(
    root,
    relativePath,
    `actor: sandbox-worker
task: TASK-SANDBOX-PERMISSION
scope: path
paths:
  - capability:dependency_install
decision: allow
reason: Sandbox validates approval-gated capability handling.
expiration: "2099-01-01T00:00:00.000Z"
authorizer: ${authorizer}
created_at: "2026-06-01T00:00:00.000Z"
`
  );
}

describe("real sandbox command flows", () => {
  it("runs init, doctor, context, permissions, and strict verify in a git sandbox", async () => {
    const root = makeSandbox();
    try {
      writePackageProject(root);
      await initGit(root);
      await initHarness(root, "deep");
      writeStandardCard(root, ".x-harness/cards/success.yaml");
      writeIntervention(
        root,
        ".x-harness/interventions/dependency-install.yaml"
      );
      await commitAll(root);

      expect((await run("npm", ["test"], root)).exitCode).toBe(0);

      const doctor = await xh(["doctor", "--root", root, "--json"], root);
      expect(doctor.exitCode, doctor.stderr || doctor.stdout).toBe(0);
      expect(readJSON<{ healthy: boolean }>(doctor.stdout).healthy).toBe(true);

      const context = await xh(["context", "--json", "--root", root], root);
      expect(context.exitCode, context.stderr || context.stdout).toBe(0);
      const contextJSON = readJSON<{ agents_fresh: boolean; hash: string }>(
        context.stdout
      );
      expect(contextJSON.agents_fresh).toBe(true);
      expect(contextJSON.hash).toMatch(/^[a-f0-9]{16}$/);

      const permission = await xh(
        [
          "permissions",
          "check",
          "--role",
          "worker",
          "--tier",
          "deep",
          "--capability",
          "dependency_install",
          "--intervention",
          ".x-harness/interventions/dependency-install.yaml",
          "--root",
          root,
          "--json",
        ],
        root
      );
      expect(permission.exitCode, permission.stderr || permission.stdout).toBe(
        0
      );
      const permissionJSON = readJSON<{
        status: string;
        intervention: { valid: boolean };
      }>(permission.stdout);
      expect(permissionJSON.status).toBe("allowed");
      expect(permissionJSON.intervention.valid).toBe(true);

      const verify = await xh(
        [
          "verify",
          "--card",
          ".x-harness/cards/success.yaml",
          "--strict",
          "--json",
        ],
        root
      );
      expect(verify.exitCode, verify.stderr || verify.stdout).toBe(0);
      const verifyJSON = readJSON<{
        ok: boolean;
        schema_version: string | number;
        acceptance_status: string;
      }>(verify.stdout);
      expect(verifyJSON.ok).toBe(true);
      expect(String(verifyJSON.schema_version)).toBe("1");
      expect(verifyJSON.acceptance_status).toBe("accepted");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }, 20000);

  it("fails closed for invalid intervention authority in a sandbox permission check", async () => {
    const root = makeSandbox();
    try {
      await initHarness(root, "standard");
      writeIntervention(
        root,
        ".x-harness/interventions/missing-authorizer.yaml",
        ""
      );

      const permission = await xh(
        [
          "permissions",
          "check",
          "--role",
          "worker",
          "--tier",
          "deep",
          "--capability",
          "dependency_install",
          "--intervention",
          ".x-harness/interventions/missing-authorizer.yaml",
          "--root",
          root,
          "--json",
        ],
        root
      );
      expect(permission.exitCode).toBe(1);
      const output = readJSON<{
        status: string;
        intervention: { valid: boolean; reason: string };
      }>(permission.stdout);
      expect(output.status).toBe("requires_intervention");
      expect(output.intervention.valid).toBe(false);
      expect(output.intervention.reason).toContain("authorizer");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }, 20000);

  it("uses strict mutation guard fallback in a non-git sandbox", async () => {
    const root = makeSandbox("xh-real-nongit-");
    try {
      writePackageProject(root);
      await initHarness(root, "standard");
      writeStandardCard(root, ".x-harness/cards/non-git.yaml", {
        taskId: "TASK-SANDBOX-NON-GIT",
      });

      const verify = await xh(
        [
          "verify",
          "--card",
          ".x-harness/cards/non-git.yaml",
          "--strict",
          "--json",
        ],
        root,
        { X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY: "4" }
      );
      expect(verify.exitCode, verify.stderr || verify.stdout).toBe(0);
      const output = readJSON<{
        ok: boolean;
        checks: Array<{ name: string; note?: string }>;
      }>(verify.stdout);
      expect(output.ok).toBe(true);
      expect(
        output.checks.some((check) =>
          check.note?.includes("mutation guard passed")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }, 20000);

  it("preserves user-managed files during deep init merge and keeps benchmark guarded outside the source checkout", async () => {
    const root = makeSandbox("xh-real-merge-qa-");
    try {
      writePackageProject(root);
      writeFile(
        root,
        ".github/workflows/x-harness-verify.yml",
        "name: user-owned workflow\n"
      );

      await initGit(root);
      await initHarness(root, "deep");

      const workflow = fs.readFileSync(
        path.join(root, ".github/workflows/x-harness-verify.yml"),
        "utf-8"
      );
      expect(workflow).toBe("name: user-owned workflow\n");

      const doctor = await xh(["doctor", "--root", root, "--json"], root);
      expect(doctor.exitCode, doctor.stderr || doctor.stdout).toBe(0);
      const doctorJSON = readJSON<{ healthy: boolean }>(doctor.stdout);
      expect(doctorJSON.healthy).toBe(true);

      const context = await xh(
        ["context", "--check", "--root", root, "--json"],
        root
      );
      expect(context.exitCode, context.stderr || context.stdout).toBe(0);
      const contextJSON = readJSON<{ valid: boolean }>(context.stdout);
      expect(contextJSON.valid).toBe(true);

      const benchmark = await run(
        process.execPath,
        [cliPath, "benchmark", "--commands", "verify"],
        root
      );
      expect(benchmark.exitCode).not.toBe(0);
      expect(benchmark.stderr + benchmark.stdout).toContain(
        "benchmark must be run from an x-harness source checkout"
      );
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }, 30000);

  it("detects verifier-side source mutations in a sandbox", async () => {
    const root = makeSandbox();
    try {
      writePackageProject(root);
      await initGit(root);
      await initHarness(root, "standard");
      writeStandardCard(root, ".x-harness/cards/mutation.yaml", {
        taskId: "TASK-SANDBOX-MUTATION",
      });
      await commitAll(root);

      const injectedMutation = path.join(
        root,
        "src",
        "ui",
        "mutated-by-verify.js"
      );
      const verify = await xh(
        [
          "verify",
          "--card",
          ".x-harness/cards/mutation.yaml",
          "--strict",
          "--json",
        ],
        root,
        {
          X_HARNESS_ENABLE_TEST_HOOKS: "1",
          X_HARNESS_TEST_INJECT_MUTATION: injectedMutation,
        }
      );
      expect(verify.exitCode).toBe(1);
      const output = readJSON<{
        ok: boolean;
        acceptance_status: string;
        recovery: { predicate: string };
      }>(verify.stdout);
      expect(output.ok).toBe(false);
      expect(output.acceptance_status).toBe("withheld");
      expect(output.recovery.predicate).toBe("verifier_not_read_only");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }, 20000);
});
