import { describe, expect, it } from "vitest";
import { execFile } from "node:child_process";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const packageRoot = path.resolve(path.join(__dirname, ".."));
const strictFixturePath = path.join(
  repoRoot,
  "examples",
  "ci",
  "strict-verify",
  "completion-card.yaml"
);
const governedDeepFixturePath = path.join(
  repoRoot,
  "examples",
  "ci",
  "governed-deep-verify",
  "completion-card.yaml"
);
const strictVerifyCommand =
  "node packages/cli/dist/index.js verify --card examples/ci/strict-verify/completion-card.yaml --strict --json";
const governedDeepVerifyCommand =
  "node packages/cli/dist/index.js verify --card examples/ci/governed-deep-verify/completion-card.yaml --profile governed-deep --json";

function execFileAsync(
  file: string,
  args: string[],
  cwd: string
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    execFile(file, args, { cwd }, (error, stdout, stderr) => {
      resolve({
        stdout: stdout.trim(),
        stderr: stderr.trim(),
        exitCode: error?.code ? Number(error.code) : 0,
      });
    });
  });
}

describe("CI workflow", () => {
  it("runs strict verify against the representative fixture", () => {
    const workflowPath = path.join(
      repoRoot,
      ".github",
      "workflows",
      "x-harness-verify.yml"
    );
    const workflow = fs.readFileSync(workflowPath, "utf-8");
    expect(fs.existsSync(strictFixturePath)).toBe(true);
    expect(fs.existsSync(governedDeepFixturePath)).toBe(true);
    expect(workflow).toContain(strictVerifyCommand);
    expect(workflow).toContain(governedDeepVerifyCommand);
    expect(workflow).toContain("npm run build && npm run test");
    expect(workflow).not.toContain("npm run test:smoke");
    expect(workflow).not.toContain("npm run test:integration");
    expect(workflow).toContain("verify-gates");
    expect(workflow).toContain("go-quality");
    expect(workflow).toContain("go test ./...");
    expect(workflow).toContain("go test -race ./...");
    expect(workflow).toContain("go vet ./...");
    expect(workflow).toContain("go build ./cmd/x-harness");
    expect(workflow).toContain("npm run parity:check-go");
    expect(workflow).toContain("go-fuzz-smoke");
    expect(workflow).toContain("-fuzz=FuzzValidate");
  });

  it("strict verify fixture passes with mutation guard enabled", async () => {
    const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), "xh-ci-strict-"));
    try {
      fs.mkdirSync(path.join(tmpRoot, "examples", "ci", "strict-verify"), {
        recursive: true,
      });
      fs.mkdirSync(path.join(tmpRoot, "policies"), { recursive: true });
      fs.copyFileSync(
        strictFixturePath,
        path.join(
          tmpRoot,
          "examples",
          "ci",
          "strict-verify",
          "completion-card.yaml"
        )
      );
      fs.copyFileSync(
        path.join(repoRoot, "examples", "ci", "strict-verify", "README.md"),
        path.join(tmpRoot, "examples", "ci", "strict-verify", "README.md")
      );
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpRoot, "policies", "admission.yaml")
      );
      const gitInit = await execFileAsync("git", ["init"], tmpRoot);
      expect(gitInit.exitCode).toBe(0);

      const { stdout, exitCode } = await execFileAsync(
        process.execPath,
        [
          path.join(packageRoot, "dist", "index.js"),
          "verify",
          "--card",
          "examples/ci/strict-verify/completion-card.yaml",
          "--strict",
          "--json",
        ],
        tmpRoot
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.strict).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
      expect(
        output.checks.some((check: { note?: string }) =>
          check.note?.includes("mutation guard passed")
        )
      ).toBe(true);
      expect(
        output.checks.some((check: { note?: string }) =>
          check.note?.includes("context_acknowledged")
        )
      ).toBe(false);
    } finally {
      fs.rmSync(tmpRoot, { recursive: true, force: true });
    }
  });

  it("governed-deep verify fixture passes with mutation guard enabled", async () => {
    const tmpRoot = fs.mkdtempSync(
      path.join(os.tmpdir(), "xh-ci-governed-deep-")
    );
    try {
      fs.mkdirSync(
        path.join(tmpRoot, "examples", "ci", "governed-deep-verify"),
        {
          recursive: true,
        }
      );
      fs.mkdirSync(path.join(tmpRoot, "policies"), { recursive: true });
      fs.copyFileSync(
        governedDeepFixturePath,
        path.join(
          tmpRoot,
          "examples",
          "ci",
          "governed-deep-verify",
          "completion-card.yaml"
        )
      );
      fs.copyFileSync(
        path.join(
          repoRoot,
          "examples",
          "ci",
          "governed-deep-verify",
          "README.md"
        ),
        path.join(
          tmpRoot,
          "examples",
          "ci",
          "governed-deep-verify",
          "README.md"
        )
      );
      fs.copyFileSync(
        path.join(repoRoot, "policies", "admission.yaml"),
        path.join(tmpRoot, "policies", "admission.yaml")
      );
      const gitInit = await execFileAsync("git", ["init"], tmpRoot);
      expect(gitInit.exitCode).toBe(0);

      const { stdout, exitCode } = await execFileAsync(
        process.execPath,
        [
          path.join(packageRoot, "dist", "index.js"),
          "verify",
          "--card",
          "examples/ci/governed-deep-verify/completion-card.yaml",
          "--profile",
          "governed-deep",
          "--json",
        ],
        tmpRoot
      );
      expect(exitCode).toBe(0);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(true);
      expect(output.acceptance_status).toBe("accepted");
      expect(
        output.checks.some((check: { note?: string }) =>
          check.note?.includes("mutation guard passed")
        )
      ).toBe(true);
    } finally {
      fs.rmSync(tmpRoot, { recursive: true, force: true });
    }
  });
});
