import { cpSync, existsSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const packageRoot = resolve(scriptDir, "..");
const repoRoot = resolve(packageRoot, "..", "..");

const requiredFiles = [
  "X_HARNESS.md",
  "README.md",
  "README.vi.md",
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
  "packaging",
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

function writeAgentsStub() {
  const stub = `# x-harness Agent Contract\n\nSee the root agent contract at \`../../AGENTS.md\`.\n`;
  writeFileSync(join(packageRoot, "AGENTS.md"), stub, "utf-8");
}

for (const file of requiredFiles) {
  copyFile(file);
}

writeAgentsStub();

for (const dir of requiredDirs) {
  copyDir(dir);
}

console.error(
  `synced ${requiredFiles.length + requiredDirs.length + 1} package asset group(s)`
);
