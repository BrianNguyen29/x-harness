import { describe, expect, it } from "vitest";
import { execFile } from "node:child_process";
import * as path from "node:path";
import * as fs from "node:fs";
import { resolveAssetPath, resolveAssetRoot } from "../src/core/assets.js";

const packageRoot = path.resolve(path.join(__dirname, ".."));
const repoRoot = path.resolve(path.join(packageRoot, "..", ".."));

function execFileAsync(
  file: string,
  args: string[],
  cwd: string
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    execFile(
      file,
      args,
      { cwd, maxBuffer: 20 * 1024 * 1024 },
      (error, stdout, stderr) => {
        resolve({
          stdout,
          stderr,
          exitCode: error?.code ? Number(error.code) : 0,
        });
      }
    );
  });
}

async function syncPackageAssets(): Promise<void> {
  const result = await execFileAsync(
    process.execPath,
    [path.join(packageRoot, "scripts", "sync-package-assets.mjs")],
    repoRoot
  );
  expect(result.stderr).toContain("synced");
  expect(result.exitCode).toBe(0);
}

describe("release packaging", () => {
  it("resolves packaged assets from the runtime asset root", async () => {
    await syncPackageAssets();
    const assetRoot = await resolveAssetRoot();
    expect(
      fs.existsSync(path.join(assetRoot, "templates", "COMPLETION_CARD.md"))
    ).toBe(true);
    expect(
      fs.existsSync(await resolveAssetPath("policies/admission.yaml"))
    ).toBe(true);
  });

  it("npm pack dry run includes required runtime assets", async () => {
    await syncPackageAssets();
    const npmBin = process.platform === "win32" ? "npm.cmd" : "npm";
    const result = await execFileAsync(
      npmBin,
      ["pack", "--dry-run", "--json", "--ignore-scripts"],
      packageRoot
    );
    expect(result.stderr).toBe("");
    expect(result.exitCode).toBe(0);

    const pack = JSON.parse(result.stdout)[0] as {
      files: Array<{ path: string }>;
    };
    const files = new Set(pack.files.map((file) => file.path));
    for (const required of [
      "dist/index.js",
      "schemas/agent-profile.schema.json",
      "schemas/approval-risk.schema.json",
      "schemas/claim.schema.json",
      "schemas/completion-card.schema.json",
      "schemas/cost-budget.schema.json",
      "schemas/evidence.schema.json",
      "schemas/intervention.schema.json",
      "schemas/packet.schema.json",
      "policies/admission.yaml",
      "policies/authority.yaml",
      "policies/approval-risk.yaml",
      "policies/cost-budget.yaml",
      "policies/intake.yaml",
      "policies/recovery.yaml",
      "templates/COMPLETION_CARD.md",
      "adapters/generic/AGENTS.md",
      "examples/00-minimal/completion-card.yaml",
      "examples/golden/success-light/completion-card.yaml",
      "docs/README.md",
      "docs/RELEASE_SECURITY.md",
      "components/registry.yaml",
      "policies/federation.yaml",
      "schemas/federation-pattern.schema.json",
      "tools/experimental/evolve/constitution.yaml",
      "AGENTS.md",
      "X_HARNESS.md",
      "README.md",
      "CHANGELOG.md",
      "LICENSE",
      "CODE_OF_CONDUCT.md",
      "CONTRIBUTING.md",
      "SECURITY.md",
      "SUPPORT.md",
    ]) {
      expect(files.has(required), `packed file missing: ${required}`).toBe(
        true
      );
    }
  });

  it("release workflows include benchmark, pack, SBOM, and provenance gates", () => {
    const releaseWorkflow = fs.readFileSync(
      path.join(repoRoot, ".github", "workflows", "release.yml"),
      "utf-8"
    );
    const sbomWorkflow = fs.readFileSync(
      path.join(repoRoot, ".github", "workflows", "sbom.yml"),
      "utf-8"
    );
    expect(releaseWorkflow).toContain("benchmark --filter adversarial --json");
    expect(releaseWorkflow).toContain("npm -w packages/cli run pack:dry-run");
    expect(releaseWorkflow).toContain("npm sbom --workspace x-harness");
    expect(releaseWorkflow).toContain(
      "npm publish --workspace x-harness --provenance"
    );
    expect(releaseWorkflow).toContain("Packed CLI smoke test");
    expect(releaseWorkflow).toContain("Frozen transfer compatibility");
    expect(releaseWorkflow).toContain("frozen verify");
    expect(releaseWorkflow).toContain("--frozen --target");
    expect(releaseWorkflow).toContain("Build Go release binaries");
    expect(releaseWorkflow).toContain("Generate Go binary checksums");
    expect(releaseWorkflow).toContain("Go binary smoke test");
    expect(releaseWorkflow).toContain("tests/smoke/go-binary-smoke.sh");
    expect(releaseWorkflow).toContain("go-binaries");
    expect(sbomWorkflow).toContain("npm sbom --workspace x-harness");
  });
});
