import { cpSync, existsSync, mkdirSync, rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const packageRoot = resolve(scriptDir, "..");
const repoRoot = resolve(packageRoot, "..", "..");

const requiredFiles = [
  "AGENTS.md",
  "X_HARNESS.md",
  "README.md",
  "CHANGELOG.md",
  "LICENSE",
  "CODE_OF_CONDUCT.md",
  "CONTRIBUTING.md",
  "SECURITY.md",
  "SUPPORT.md",
];

const requiredDirs = [
  "adapters",
  "components",
  "docs",
  "examples",
  "policies",
  "schemas",
  "skills",
  "templates",
  "tools",
];

function copyFile(relativePath) {
  const source = join(repoRoot, relativePath);
  if (!existsSync(source)) {
    throw new Error(`required package asset missing: ${relativePath}`);
  }
  cpSync(source, join(packageRoot, relativePath));
}

function copyDir(relativePath) {
  const source = join(repoRoot, relativePath);
  if (!existsSync(source)) {
    throw new Error(
      `required package asset directory missing: ${relativePath}`
    );
  }
  const target = join(packageRoot, relativePath);
  rmSync(target, { recursive: true, force: true });
  mkdirSync(join(packageRoot, relativePath, ".."), { recursive: true });
  cpSync(source, target, { recursive: true });
}

for (const file of requiredFiles) {
  copyFile(file);
}

for (const dir of requiredDirs) {
  copyDir(dir);
}

console.error(
  `synced ${requiredFiles.length + requiredDirs.length} package asset group(s)`
);
