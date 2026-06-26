import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const registryPath = join(repoRoot, "internal", "cli", "commands.json");
const docsPath = join(repoRoot, "docs", "CLI_COMMANDS.md");
const readmePath = join(repoRoot, "README.md");
const args = new Set(process.argv.slice(2));
const mode = args.has("--write") ? "write" : args.has("--check") ? "check" : "";

if (!mode) {
  console.error("usage: node scripts/generate-cli-docs.mjs --write|--check");
  process.exit(2);
}

const maturityOrder = ["stable", "beta", "experimental", "skeletal"];
const beginMarker = "<!-- BEGIN X-HARNESS MANAGED CLI CORE COMMANDS -->";
const endMarker = "<!-- END X-HARNESS MANAGED CLI CORE COMMANDS -->";

function loadRegistry() {
  const commands = JSON.parse(readFileSync(registryPath, "utf8"));
  const seen = new Set();
  for (const command of commands) {
    if (!command.name || !command.description || !command.maturity) {
      throw new Error(
        "CLI command registry entries require name, description, and maturity",
      );
    }
    if (!maturityOrder.includes(command.maturity)) {
      throw new Error(
        `unknown maturity for ${command.name}: ${command.maturity}`,
      );
    }
    if (seen.has(command.name)) {
      throw new Error(`duplicate CLI command registry entry: ${command.name}`);
    }
    seen.add(command.name);
  }
  return commands;
}

function onboardingCommands(commands) {
  return commands
    .filter((command) => command.onboarding)
    .sort((a, b) => {
      const left = a.onboarding_order ?? Number.MAX_SAFE_INTEGER;
      const right = b.onboarding_order ?? Number.MAX_SAFE_INTEGER;
      return left === right ? a.name.localeCompare(b.name) : left - right;
    });
}

function table(commands, columns = ["Command", "Maturity", "Description"]) {
  const lines = [
    `| ${columns.join(" | ")} |`,
    `| ${columns.map(() => ":--").join(" | ")} |`,
  ];
  for (const command of commands) {
    if (columns.length === 2) {
      lines.push(`| \`${command.name}\` | ${command.description} |`);
    } else {
      lines.push(
        `| \`${command.name}\` | ${command.maturity} | ${command.description} |`,
      );
    }
  }
  return lines.join("\n");
}

function renderReadmeBlock(commands) {
  return [
    beginMarker,
    table(onboardingCommands(commands), ["Action", "What it does"]),
    endMarker,
  ].join("\n");
}

function renderCliDocs(commands) {
  const lines = [
    "# CLI Commands",
    "",
    "<!-- generated-by: scripts/generate-cli-docs.mjs -->",
    "<!-- source: internal/cli/commands.json -->",
    "",
    "This file is generated from the canonical CLI command registry. Do not edit command tables manually; update `internal/cli/commands.json` and run `npm run cli-metadata:write`.",
    "",
    "## Onboarding Commands",
    "",
    "The default onboarding path is intentionally narrow: initialize the workspace, run the health check, then verify a completion card.",
    "",
    table(onboardingCommands(commands), ["Command", "Description"]),
    "",
    "## Full Command Matrix",
    "",
  ];
  for (const maturity of maturityOrder) {
    const group = commands.filter((command) => command.maturity === maturity);
    if (group.length === 0) continue;
    lines.push(`### ${maturity}`, "", table(group), "");
  }
  lines.push(
    "## Maturity Labels",
    "",
    "- `stable`: core command; tested and relied on in CI.",
    "- `beta`: functional but may change before 1.0.",
    "- `experimental`: advanced or exploratory; semantics may shift.",
    "- `skeletal`: declared but not yet implemented.",
    "",
  );
  return lines.join("\n");
}

function updateReadme(readme, block) {
  const start = readme.indexOf(beginMarker);
  const end = readme.indexOf(endMarker);
  if (start === -1 || end === -1 || end < start) {
    throw new Error("README.md is missing managed CLI core command markers");
  }
  return `${readme.slice(0, start)}${block}${readme.slice(end + endMarker.length)}`;
}

const commands = loadRegistry();
const nextDocs = renderCliDocs(commands);
const readme = readFileSync(readmePath, "utf8");
const nextReadme = updateReadme(readme, renderReadmeBlock(commands));

if (mode === "write") {
  writeFileSync(docsPath, nextDocs);
  writeFileSync(readmePath, nextReadme);
  console.error("generated CLI metadata docs");
} else {
  const currentDocs = readFileSync(docsPath, "utf8");
  const failures = [];
  if (currentDocs !== nextDocs) failures.push("docs/CLI_COMMANDS.md");
  if (readme !== nextReadme)
    failures.push("README.md managed CLI core commands block");
  if (failures.length > 0) {
    console.error(`CLI metadata docs are stale: ${failures.join(", ")}`);
    console.error("run: npm run cli-metadata:write");
    process.exit(1);
  }
  console.error("CLI metadata docs are current");
}
