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

// Pure helper: returns the subset of `filesEntries` that are neither
// present in `covered` (entries with a dynamic/static pack-manifest
// assertion) nor in `excluded` (intentionally-ignored entries with
// justification). Extracted from the "every package.json files entry has
// pack-manifest coverage" meta-guard so the missing-entry detection
// logic can be exercised by a focused unit test.
function findMissingFilesCoverage(
  filesEntries: string[],
  covered: Set<string>,
  excluded: Set<string>
): string[] {
  return filesEntries.filter(
    (entry) => !covered.has(entry) && !excluded.has(entry)
  );
}

// Pure helper: returns the subset of `requiredDirs` (parsed from
// sync-package-assets.mjs) that are NOT present in `covered` (dirs with
// a dynamic pack-manifest coverage block in this test file). Extracted
// from the "every requiredDirs entry has a dynamic pack-manifest
// coverage group" meta-guard so the missing-entry detection logic can
// be exercised by a focused unit test.
function findMissingDirsCoverage(
  requiredDirs: string[],
  covered: Set<string>
): string[] {
  return requiredDirs.filter((dir) => !covered.has(dir));
}

// Pure helper: parses the `const requiredDirs = [...]` array literal
// from the canonical sync-package-assets.mjs source text. Extracted
// from the "every requiredDirs entry has a dynamic pack-manifest
// coverage group" meta-guard so the array-shape contract (and its
// failure modes) can be exercised by a focused unit test. Throws a
// clear error when the array literal is missing or when the parsed
// value is not a non-empty array of strings.
function parseRequiredDirsFromSyncScript(source: string): string[] {
  const match = source.match(/const requiredDirs\s*=\s*(\[[\s\S]*?\]);/m);
  if (!match) {
    throw new Error(
      "requiredDirs array literal not found in sync-package-assets.mjs"
    );
  }
  // The matched group is a plain JS array of double-quoted string
  // literals. Strip a trailing comma inside the array literal so the
  // text becomes valid JSON, then JSON.parse it. This keeps the
  // helper independent of any script execution and tolerant of
  // cosmetic formatting changes inside the array body.
  const arrayLiteral = match[1].replace(/,(\s*])/, "$1");
  const parsed = JSON.parse(arrayLiteral) as unknown;
  if (
    !Array.isArray(parsed) ||
    parsed.length === 0 ||
    !parsed.every((value) => typeof value === "string")
  ) {
    throw new Error(
      "requiredDirs must be a non-empty array of strings in sync-package-assets.mjs"
    );
  }
  return parsed;
}

// Pure helper: validates the package.json `files` entry shape used by
// the "every package.json files entry has pack-manifest coverage"
// meta-guard. Returns the entries as a `string[]` when `value` is a
// non-empty array of strings; throws a clear error otherwise. Extracted
// from the meta-guard so the array-shape contract (and its failure
// modes) can be exercised by a focused unit test. No I/O, no expect
// inside the helper.
function parsePackageFilesEntries(value: unknown): string[] {
  if (!Array.isArray(value)) {
    throw new Error(
      'package.json "files" must be a non-empty array of strings'
    );
  }
  if (value.length === 0) {
    throw new Error(
      'package.json "files" must be a non-empty array of strings'
    );
  }
  if (!value.every((entry) => typeof entry === "string")) {
    throw new Error(
      'package.json "files" must be a non-empty array of strings'
    );
  }
  return value as string[];
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
    // Every synced packaging/** file must appear in the pack manifest. The
    // sync script recursively copies the root packaging/ directory (which is
    // nested across platform subdirs and may include hidden files), so we
    // recursively collect every file under packages/cli/packaging/ and assert
    // each appears in the pack manifest using POSIX separators. This guards
    // against future packaging additions (e.g.
    // packaging/scoop/manifest.json) silently being dropped from the npm pack
    // manifest.
    const syncedPackagingDir = path.join(packageRoot, "packaging");
    expect(
      fs.existsSync(syncedPackagingDir),
      "synced packages/cli/packaging/ must exist after sync"
    ).toBe(true);
    const syncedPackagingFiles = collectFilesRecursive(syncedPackagingDir);
    expect(
      syncedPackagingFiles.length,
      "synced packages/cli/packaging/ must contain at least one file"
    ).toBeGreaterThan(0);
    for (const abs of syncedPackagingFiles) {
      const packPath = path.posix.join(
        "packaging",
        path.posix.relative(
          syncedPackagingDir.split(path.sep).join(path.posix.sep),
          abs.split(path.sep).join(path.posix.sep)
        )
      );
      expect(files.has(packPath), `packed file missing: ${packPath}`).toBe(
        true
      );
    }
    // Every synced skills/** file must appear in the pack manifest. The
    // sync script recursively copies the root skills/ directory (which is
    // nested across skill subdirs and may include hidden files), so we
    // recursively collect every file under packages/cli/skills/ and assert
    // each appears in the pack manifest using POSIX separators. This guards
    // against future skill additions (e.g.
    // skills/x-harness-admission/handbook.md) silently being dropped from
    // the npm pack manifest.
    const syncedSkillsDir = path.join(packageRoot, "skills");
    expect(
      fs.existsSync(syncedSkillsDir),
      "synced packages/cli/skills/ must exist after sync"
    ).toBe(true);
    const syncedSkillFiles = collectFilesRecursive(syncedSkillsDir);
    expect(
      syncedSkillFiles.length,
      "synced packages/cli/skills/ must contain at least one file"
    ).toBeGreaterThan(0);
    for (const abs of syncedSkillFiles) {
      const packPath = path.posix.join(
        "skills",
        path.posix.relative(
          syncedSkillsDir.split(path.sep).join(path.posix.sep),
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
    const copyIndex = releaseWorkflow.indexOf(
      "Copy Go binaries into npm package"
    );
    expect(goBuildIndex).toBeGreaterThan(-1);
    expect(npmPackIndex).toBeGreaterThan(-1);
    expect(goBuildIndex).toBeLessThan(npmPackIndex);
    expect(copyIndex).toBeGreaterThan(-1);
    expect(goBuildIndex).toBeLessThan(copyIndex);
    expect(copyIndex).toBeLessThan(npmPackIndex);
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

  it("every requiredDirs entry has a dynamic pack-manifest coverage group", () => {
    // Meta-guard: parse the canonical requiredDirs list from
    // packages/cli/scripts/sync-package-assets.mjs (without executing it)
    // and assert every entry has a corresponding dynamic pack-manifest
    // coverage block in this test file. Adding a new requiredDir in the
    // sync script without (a) adding a dynamic block for it in the
    // "npm pack dry run" test above and (b) registering it in coveredDirs
    // below will fail this guard.
    const scriptPath = path.join(
      packageRoot,
      "scripts",
      "sync-package-assets.mjs"
    );
    const scriptSource = fs.readFileSync(scriptPath, "utf-8");
    const requiredDirs = parseRequiredDirsFromSyncScript(scriptSource);
    // The coveredDirs set enumerates every requiredDir that has a dynamic
    // pack-manifest coverage block in this test file. Every entry in
    // requiredDirs above MUST appear in this set; the meta-guard fails
    // otherwise with a clear remediation message. Keep entries
    // alphabetised to minimise diff noise.
    const coveredDirs = new Set<string>([
      "adapters",
      "components",
      "docs",
      "examples",
      "packaging",
      "policies",
      "schemas",
      "skills",
      "templates",
      "tools",
    ]);
    const missing = findMissingDirsCoverage(requiredDirs, coveredDirs);
    expect(
      missing,
      `requiredDirs entries from sync-package-assets.mjs without pack-manifest coverage in release-pack.test.ts: ${missing.join(
        ", "
      )}; add a dynamic block (see packaging/skills guards above) and register the entry in coveredDirs`
    ).toEqual([]);
  });

  it("every package.json files entry has pack-manifest coverage", () => {
    // Meta-guard: read packages/cli/package.json directly and assert every
    // entry in its `files` array is either (a) covered by a dynamic or
    // static pack-manifest assertion in this test file, or (b) explicitly
    // listed in excludedFilesEntries with a justification. Adding a new
    // entry to package.json `files` without (a) adding a corresponding
    // dynamic/static coverage block AND registering it in
    // coveredFilesEntries, or (b) documenting it in excludedFilesEntries,
    // will fail this guard with a clear remediation message.
    const packageJsonPath = path.join(packageRoot, "package.json");
    const packageJson = JSON.parse(
      fs.readFileSync(packageJsonPath, "utf-8")
    ) as { files?: unknown };
    const filesEntries = parsePackageFilesEntries(packageJson.files);
    // The coveredFilesEntries set enumerates every package.json `files`
    // entry that is currently exercised by a dynamic or static
    // pack-manifest assertion in this test file:
    //   - schemas, policies, templates, adapters, examples, skills, docs,
    //     components, tools, packaging have dynamic iterated/recursive
    //     coverage blocks in the "npm pack dry run" test.
    //   - bin is covered by the static "bin/x-harness.js" required entry.
    //   - AGENTS.md, X_HARNESS.md, README.md, CHANGELOG.md, LICENSE,
    //     CODE_OF_CONDUCT.md, CONTRIBUTING.md, SECURITY.md, SUPPORT.md are
    //     covered by the static required root-file array.
    // Every entry in package.json `files` above MUST appear in this set
    // (unless listed in excludedFilesEntries below). Keep entries
    // alphabetised to minimise diff noise.
    const coveredFilesEntries = new Set<string>([
      "AGENTS.md",
      "CHANGELOG.md",
      "CODE_OF_CONDUCT.md",
      "CONTRIBUTING.md",
      "LICENSE",
      "README.md",
      "SECURITY.md",
      "SUPPORT.md",
      "X_HARNESS.md",
      "adapters",
      "bin",
      "components",
      "docs",
      "examples",
      "packaging",
      "policies",
      "schemas",
      "skills",
      "templates",
      "tools",
    ]);
    // The excludedFilesEntries set enumerates package.json `files` entries
    // that are intentionally NOT covered by a local pack-manifest
    // assertion. Each entry must be justified inline so reviewers can
    // audit intentional exclusions.
    const excludedFilesEntries = new Set<string>([
      // "go-binaries" is a CI-only release artifact directory populated
      // by .github/workflows/release.yml (Build Go release binaries +
      // Copy Go binaries into npm package steps). It does not exist in
      // the local working tree (no root go-binaries/ directory is
      // committed), so no local pack-manifest assertion can be written
      // for it. The release workflow gates (smoke test, Go binary smoke
      // test, Packed CLI Go smoke test) provide the upstream coverage
      // contract.
      "go-binaries",
    ]);
    const missing = findMissingFilesCoverage(
      filesEntries,
      coveredFilesEntries,
      excludedFilesEntries
    );
    expect(
      missing,
      `package.json "files" entries without pack-manifest coverage in release-pack.test.ts: ${missing.join(
        ", "
      )}; add a dynamic or static assertion and register the entry in coveredFilesEntries, or document an intentional exclusion in excludedFilesEntries with a comment`
    ).toEqual([]);
  });

  it("findMissingFilesCoverage detects uncovered entries", () => {
    // Negative fixture: prove the helper returns only the entries that are
    // neither covered nor excluded, while ignoring entries present in
    // either set. This guards against silent regressions in the
    // missing-entry detection logic shared with the package.json `files`
    // meta-guard above.
    expect(
      findMissingFilesCoverage(
        ["bin", "docs", "uncovered-entry"],
        new Set(["bin"]),
        new Set(["docs"])
      )
    ).toEqual(["uncovered-entry"]);
    // Order-preserving: uncovered entries appear in the order they are
    // declared in filesEntries, regardless of set insertion order.
    expect(
      findMissingFilesCoverage(
        ["alpha", "beta", "gamma", "delta"],
        new Set(["alpha"]),
        new Set(["gamma"])
      )
    ).toEqual(["beta", "delta"]);
    // Empty inputs short-circuit cleanly without throwing.
    expect(findMissingFilesCoverage([], new Set(), new Set())).toEqual([]);
    expect(
      findMissingFilesCoverage(["x", "y"], new Set(["x", "y"]), new Set())
    ).toEqual([]);
  });

  it("findMissingDirsCoverage detects uncovered requiredDirs", () => {
    // Negative fixture: prove the helper returns only the entries that
    // are not present in the covered set, while ignoring entries that
    // are present. This guards against silent regressions in the
    // missing-entry detection logic shared with the requiredDirs
    // meta-guard above.
    expect(
      findMissingDirsCoverage(
        ["adapters", "templates", "uncovered-dir"],
        new Set(["adapters", "templates"])
      )
    ).toEqual(["uncovered-dir"]);
    // Order-preserving: uncovered entries appear in the order they are
    // declared in requiredDirs, regardless of set insertion order.
    expect(
      findMissingDirsCoverage(
        ["alpha", "beta", "gamma", "delta"],
        new Set(["alpha"])
      )
    ).toEqual(["beta", "gamma", "delta"]);
    // Empty inputs short-circuit cleanly without throwing.
    expect(findMissingDirsCoverage([], new Set())).toEqual([]);
    // All-covered input returns an empty array.
    expect(findMissingDirsCoverage(["x", "y"], new Set(["x", "y"]))).toEqual(
      []
    );
  });

  it("parseRequiredDirsFromSyncScript parses the canonical array literal", () => {
    // Positive fixture: prove the helper accepts the current
    // sync-package-assets.mjs array shape, including a trailing
    // comma inside the literal. This guards against silent
    // regressions in the parser logic shared with the requiredDirs
    // meta-guard above.
    const source = `const requiredDirs = ["adapters", "docs",];`;
    expect(parseRequiredDirsFromSyncScript(source)).toEqual([
      "adapters",
      "docs",
    ]);
    // Single-entry array (no trailing comma) still parses.
    expect(
      parseRequiredDirsFromSyncScript(`const requiredDirs = ["schemas"];`)
    ).toEqual(["schemas"]);
    // Whitespace and newlines around the array literal are tolerated.
    const multiline = `const requiredDirs = [\n  "adapters",\n  "docs",\n];`;
    expect(parseRequiredDirsFromSyncScript(multiline)).toEqual([
      "adapters",
      "docs",
    ]);
  });

  it("parseRequiredDirsFromSyncScript fails clearly when requiredDirs is absent", () => {
    // Negative fixture: prove the helper throws a clear
    // "requiredDirs array literal not found" error when the source
    // does not declare a requiredDirs array. This guards against
    // silent regressions in the parser's missing-literal detection
    // shared with the requiredDirs meta-guard above.
    expect(() => parseRequiredDirsFromSyncScript("")).toThrow(
      /requiredDirs array literal not found/
    );
    expect(() =>
      parseRequiredDirsFromSyncScript('const otherDirs = ["adapters"];')
    ).toThrow(/requiredDirs array literal not found/);
  });

  it("parseRequiredDirsFromSyncScript fails clearly on empty or non-string arrays", () => {
    // Shape fixture: prove the helper throws a clear
    // "requiredDirs must be a non-empty array of strings" error
    // when the parsed value is empty or contains a non-string
    // element. This guards against silent regressions in the
    // parser's shape validation shared with the requiredDirs
    // meta-guard above.
    expect(() =>
      parseRequiredDirsFromSyncScript("const requiredDirs = [];")
    ).toThrow(/requiredDirs must be a non-empty array of strings/);
    expect(() =>
      parseRequiredDirsFromSyncScript("const requiredDirs = [1, 2];")
    ).toThrow(/requiredDirs must be a non-empty array of strings/);
    expect(() =>
      parseRequiredDirsFromSyncScript('const requiredDirs = ["adapters", 2];')
    ).toThrow(/requiredDirs must be a non-empty array of strings/);
  });

  it("parsePackageFilesEntries parses valid arrays of strings", () => {
    // Positive fixture: prove the helper accepts the canonical
    // package.json `files` array shape used by the npm pack meta-guard
    // above. Multi-entry arrays round-trip in declaration order, and a
    // single-entry array (the smallest legal shape) is accepted.
    expect(
      parsePackageFilesEntries([
        "bin",
        "schemas",
        "policies",
        "templates",
        "adapters",
      ])
    ).toEqual(["bin", "schemas", "policies", "templates", "adapters"]);
    // Single-entry array (no trailing entries) still parses.
    expect(parsePackageFilesEntries(["bin"])).toEqual(["bin"]);
    // Order is preserved exactly as declared in the input array,
    // matching the package.json `files` order in the meta-guard.
    expect(parsePackageFilesEntries(["z", "a", "m", "b"])).toEqual([
      "z",
      "a",
      "m",
      "b",
    ]);
  });

  it("parsePackageFilesEntries fails clearly on non-array inputs", () => {
    // Negative fixture: prove the helper throws a clear
    // "package.json \"files\" must be a non-empty array of strings"
    // error when the input is not an array (undefined, null, string,
    // object). This guards against silent regressions in the parser's
    // shape validation shared with the package.json `files`
    // meta-guard above.
    expect(() => parsePackageFilesEntries(undefined)).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
    expect(() => parsePackageFilesEntries(null)).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
    expect(() => parsePackageFilesEntries("bin")).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
    expect(() => parsePackageFilesEntries({ files: ["bin"] })).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
  });

  it("parsePackageFilesEntries fails clearly on an empty array", () => {
    // Negative fixture: prove the helper throws a clear
    // "package.json \"files\" must be a non-empty array of strings"
    // error when the input is an empty array. This guards against
    // silent regressions in the parser's shape validation shared with
    // the package.json `files` meta-guard above.
    expect(() => parsePackageFilesEntries([])).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
  });

  it("parsePackageFilesEntries fails clearly on non-string elements", () => {
    // Negative fixture: prove the helper throws a clear
    // "package.json \"files\" must be a non-empty array of strings"
    // error when the array contains any non-string element (number,
    // null, or object). This guards against silent regressions in the
    // parser's shape validation shared with the package.json `files`
    // meta-guard above.
    expect(() => parsePackageFilesEntries(["bin", 2])).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
    expect(() => parsePackageFilesEntries(["bin", null])).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
    expect(() => parsePackageFilesEntries(["bin", { name: "bin" }])).toThrow(
      /package\.json "files" must be a non-empty array of strings/
    );
  });
});
