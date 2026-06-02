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

function collectFilesRecursive(dir: string): string[] {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const out: string[] = [];
  for (const entry of entries) {
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      out.push(...collectFilesRecursive(abs));
    } else if (entry.isFile()) {
      out.push(abs);
    }
  }
  return out;
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
    expect(
      files.has("dist/index.js"),
      "packed file must not include dist/index.js"
    ).toBe(false);
    // Every synced policies/*.yaml file must appear in the pack manifest.
    // The sync script recursively copies the root policies/ directory, so we
    // discover the runtime set from packages/cli/policies/ and assert each
    // shallow *.yaml is present. This guards against future root policy
    // additions (e.g. policies/evidence.yaml) silently being dropped.
    const syncedPoliciesDir = path.join(packageRoot, "policies");
    const syncedPolicyFiles = fs
      .readdirSync(syncedPoliciesDir)
      .filter((name) => name.endsWith(".yaml"));
    expect(
      syncedPolicyFiles.length,
      "synced packages/cli/policies/ must contain at least one .yaml file"
    ).toBeGreaterThan(0);
    for (const name of syncedPolicyFiles) {
      const packPath = `policies/${name}`;
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced schemas/*.schema.json file must appear in the pack
    // manifest. The sync script recursively copies the root schemas/
    // directory, so we discover the runtime set from packages/cli/schemas/
    // and assert each shallow *.schema.json is present. This guards against
    // future root schema additions (e.g. schemas/foo.schema.json) silently
    // being dropped from the npm pack manifest.
    const syncedSchemasDir = path.join(packageRoot, "schemas");
    const syncedSchemaFiles = fs
      .readdirSync(syncedSchemasDir)
      .filter((name) => name.endsWith(".schema.json"));
    expect(
      syncedSchemaFiles.length,
      "synced packages/cli/schemas/ must contain at least one .schema.json file"
    ).toBeGreaterThan(0);
    for (const name of syncedSchemaFiles) {
      const packPath = `schemas/${name}`;
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced templates/*.md file must appear in the pack manifest.
    // The sync script recursively copies the root templates/ directory, so
    // we discover the runtime set from packages/cli/templates/ and assert
    // each shallow *.md is present. This guards against future root template
    // additions (e.g. templates/NEW_CONTRACT.md) silently being dropped from
    // the npm pack manifest.
    const syncedTemplatesDir = path.join(packageRoot, "templates");
    const syncedTemplateFiles = fs
      .readdirSync(syncedTemplatesDir)
      .filter((name) => name.endsWith(".md"));
    expect(
      syncedTemplateFiles.length,
      "synced packages/cli/templates/ must contain at least one .md file"
    ).toBeGreaterThan(0);
    for (const name of syncedTemplateFiles) {
      const packPath = `templates/${name}`;
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced docs/*.md file must appear in the pack manifest. The
    // sync script recursively copies the root docs/ directory, so we discover
    // the runtime set from packages/cli/docs/ and assert each shallow *.md
    // is present. This guards against future root doc additions (e.g.
    // docs/NEW_GUIDE.md) silently being dropped from the npm pack manifest.
    const syncedDocsDir = path.join(packageRoot, "docs");
    const syncedDocFiles = fs
      .readdirSync(syncedDocsDir)
      .filter((name) => name.endsWith(".md"));
    expect(
      syncedDocFiles.length,
      "synced packages/cli/docs/ must contain at least one .md file"
    ).toBeGreaterThan(0);
    for (const name of syncedDocFiles) {
      const packPath = `docs/${name}`;
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced components/*.yaml file must appear in the pack manifest.
    // The sync script recursively copies the root components/ directory, so
    // we discover the runtime set from packages/cli/components/ and assert
    // each shallow *.yaml is present. This guards against future root
    // component additions (e.g. components/REGISTRY.yaml) silently being
    // dropped from the npm pack manifest.
    const syncedComponentsDir = path.join(packageRoot, "components");
    const syncedComponentFiles = fs
      .readdirSync(syncedComponentsDir)
      .filter((name) => name.endsWith(".yaml"));
    expect(
      syncedComponentFiles.length,
      "synced packages/cli/components/ must contain at least one .yaml file"
    ).toBeGreaterThan(0);
    for (const name of syncedComponentFiles) {
      const packPath = `components/${name}`;
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced adapters/** file must appear in the pack manifest. The
    // sync script recursively copies the root adapters/ directory (which is
    // nested across platform subdirs), so we recursively collect every file
    // under packages/cli/adapters/ and assert each appears in the pack
    // manifest using POSIX separators. This guards against future adapter
    // additions (e.g. adapters/opencode/agents/fixer.md) silently being
    // dropped from the npm pack manifest.
    const syncedAdaptersDir = path.join(packageRoot, "adapters");
    expect(
      fs.existsSync(syncedAdaptersDir),
      "synced packages/cli/adapters/ must exist after sync"
    ).toBe(true);
    const syncedAdapterFiles = collectFilesRecursive(syncedAdaptersDir);
    expect(
      syncedAdapterFiles.length,
      "synced packages/cli/adapters/ must contain at least one file"
    ).toBeGreaterThan(0);
    for (const abs of syncedAdapterFiles) {
      const packPath = path.posix.join(
        "adapters",
        path.posix.relative(
          syncedAdaptersDir.split(path.sep).join(path.posix.sep),
          abs.split(path.sep).join(path.posix.sep)
        )
      );
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced examples/** file must appear in the pack manifest. The
    // sync script recursively copies the root examples/ directory (which is
    // nested across topic subdirs and golden regression cases), so we
    // recursively collect every file under packages/cli/examples/ and assert
    // each appears in the pack manifest using POSIX separators. This guards
    // against future example additions (e.g.
    // examples/golden/regression/success-light/completion-card.yaml) silently
    // being dropped from the npm pack manifest.
    const syncedExamplesDir = path.join(packageRoot, "examples");
    expect(
      fs.existsSync(syncedExamplesDir),
      "synced packages/cli/examples/ must exist after sync"
    ).toBe(true);
    const syncedExampleFiles = collectFilesRecursive(syncedExamplesDir);
    expect(
      syncedExampleFiles.length,
      "synced packages/cli/examples/ must contain at least one file"
    ).toBeGreaterThan(0);
    for (const abs of syncedExampleFiles) {
      const packPath = path.posix.join(
        "examples",
        path.posix.relative(
          syncedExamplesDir.split(path.sep).join(path.posix.sep),
          abs.split(path.sep).join(path.posix.sep)
        )
      );
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced tools/** file must appear in the pack manifest. The
    // sync script recursively copies the root tools/ directory (which is
    // nested across experimental subdirs and may include hidden .gitkeep
    // files), so we recursively collect every file under packages/cli/tools/
    // and assert each appears in the pack manifest using POSIX separators.
    // This guards against future tool additions (e.g.
    // tools/experimental/evolve/runs/.gitkeep) silently being dropped from
    // the npm pack manifest.
    const syncedToolsDir = path.join(packageRoot, "tools");
    expect(
      fs.existsSync(syncedToolsDir),
      "synced packages/cli/tools/ must exist after sync"
    ).toBe(true);
    const syncedToolFiles = collectFilesRecursive(syncedToolsDir);
    expect(
      syncedToolFiles.length,
      "synced packages/cli/tools/ must contain at least one file"
    ).toBeGreaterThan(0);
    for (const abs of syncedToolFiles) {
      const packPath = path.posix.join(
        "tools",
        path.posix.relative(
          syncedToolsDir.split(path.sep).join(path.posix.sep),
          abs.split(path.sep).join(path.posix.sep)
        )
      );
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    for (const required of [
      "bin/x-harness.js",
      "schemas/agent-profile.schema.json",
      "schemas/approval-risk.schema.json",
      "schemas/claim.schema.json",
      "schemas/completion-card.schema.json",
      "schemas/cost-budget.schema.json",
      "schemas/evidence.schema.json",
      "schemas/intervention.schema.json",
      "schemas/packet.schema.json",
      "schemas/report.schema.json",
      "policies/admission.yaml",
      "policies/authority.yaml",
      "policies/approval-risk.yaml",
      "policies/cost-budget.yaml",
      "policies/intake.yaml",
      "policies/recovery.yaml",
      "adapters/generic/AGENTS.md",
      "examples/00-minimal/completion-card.yaml",
      "examples/golden/regression/success-light/completion-card.yaml",
      "policies/federation.yaml",
      "schemas/federation-pattern.schema.json",
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

  it("npm bin points to compatibility wrapper", () => {
    const packageJson = JSON.parse(
      fs.readFileSync(path.join(packageRoot, "package.json"), "utf-8")
    ) as { bin: Record<string, string>; files: string[] };
    expect(packageJson.bin["x-harness"]).toBe("./bin/x-harness.js");
    expect(packageJson.bin.xh).toBe("./bin/x-harness.js");
    expect(packageJson.files).toContain("bin");
    expect(packageJson.files).toContain("go-binaries");

    const wrapper = fs.readFileSync(
      path.join(packageRoot, "bin", "x-harness.js"),
      "utf-8"
    );
    expect(wrapper).toContain("X_HARNESS_GO");
    expect(wrapper).toContain('process.env.X_HARNESS_GO === "0"');
    expect(wrapper).not.toContain('process.env.X_HARNESS_GO !== "1"');
    expect(wrapper).toContain("go-binaries");
    expect(wrapper).toContain("existsSync(nodeEntrypoint)");
    expect(wrapper).toContain(
      "No Go binary found for your platform and Node fallback is not available"
    );
  });

  it("runtime policy copies are byte-identical to root contracts", async () => {
    // Explicit named parity guard for every root policies/*.yaml. The sync
    // script (packages/cli/scripts/sync-package-assets.mjs) recursively
    // regenerates packages/cli/policies/ from the canonical root copy; every
    // runtime copy must be byte-identical. The packages/cli/policies/
    // directory is gitignored, so this test cleans the generated path so it
    // does not depend on stale local artifact state, then re-syncs from root.
    const gitignorePath = path.join(packageRoot, ".gitignore");
    const rootPoliciesDir = path.join(repoRoot, "policies");
    const syncedPoliciesDir = path.join(packageRoot, "policies");
    const gitignore = fs.readFileSync(gitignorePath, "utf-8");
    expect(
      gitignore.includes("/policies/"),
      "packages/cli/.gitignore must exclude /policies/ so no committed pre-sync package copy is ever published"
    ).toBe(true);
    fs.rmSync(syncedPoliciesDir, { recursive: true, force: true });
    await syncPackageAssets();
    const rootPolicyFiles = fs
      .readdirSync(rootPoliciesDir)
      .filter((name) => name.endsWith(".yaml"))
      .sort();
    expect(
      rootPolicyFiles.length,
      "root policies/ must contain at least one .yaml file"
    ).toBeGreaterThan(0);
    for (const name of rootPolicyFiles) {
      const rootPolicy = path.join(rootPoliciesDir, name);
      const syncedPolicy = path.join(syncedPoliciesDir, name);
      expect(
        fs.existsSync(syncedPolicy),
        `synced packages/cli/policies/${name} must exist after sync`
      ).toBe(true);
      const rootBuffer = fs.readFileSync(rootPolicy);
      const syncedBuffer = fs.readFileSync(syncedPolicy);
      expect(
        syncedBuffer.equals(rootBuffer),
        `packages/cli/policies/${name} must be byte-identical to root policies/${name} after sync`
      ).toBe(true);
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
    expect(releaseWorkflow).toContain(
      "benchmark --filter adversarial --gate --json"
    );
    expect(releaseWorkflow).toContain("npm -w packages/cli run pack:dry-run");
    expect(releaseWorkflow).toContain("npm sbom --workspace x-harness");
    expect(releaseWorkflow).toContain(
      "npm publish --workspace x-harness --provenance"
    );
    expect(releaseWorkflow).toContain("Packed CLI smoke test");
    expect(releaseWorkflow).toContain("Frozen transfer compatibility");
    expect(releaseWorkflow).toContain("frozen verify");
    expect(releaseWorkflow).toContain("--frozen --target");
    const goBuildIndex = releaseWorkflow.indexOf("Build Go release binaries");
    const npmPackIndex = releaseWorkflow.indexOf("Build release package");
    expect(goBuildIndex).toBeGreaterThan(-1);
    expect(npmPackIndex).toBeGreaterThan(-1);
    expect(goBuildIndex).toBeLessThan(npmPackIndex);
    expect(releaseWorkflow).toContain("Copy Go binaries into npm package");
    expect(releaseWorkflow).toContain("Generate Go binary checksums");
    expect(releaseWorkflow).toContain("Go binary smoke test");
    expect(releaseWorkflow).toContain("tests/smoke/go-binary-smoke.sh");
    expect(releaseWorkflow).toContain("go-binaries");
    expect(releaseWorkflow).toContain("cosign-installer");
    expect(releaseWorkflow).toContain("cosign sign-blob");
    expect(releaseWorkflow).toContain("Packed CLI Go smoke test");
    expect(releaseWorkflow).toContain("X_HARNESS_GO=1");
    expect(releaseWorkflow).not.toContain("X_HARNESS_GO=0");
    expect(releaseWorkflow).toContain("cross-platform-smoke");
    expect(releaseWorkflow).toContain("ubuntu-latest");
    expect(releaseWorkflow).toContain("macos-latest");
    expect(releaseWorkflow).toContain("windows-latest");
    expect(sbomWorkflow).toContain("npm sbom --workspace x-harness");
  });
});
