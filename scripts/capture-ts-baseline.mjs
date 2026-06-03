#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import {
  existsSync,
  mkdirSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const defaultBaselineRoot = path.join(
  repoRoot,
  "tests",
  "parity",
  "baseline",
  "typescript",
);
let baselineRoot = defaultBaselineRoot;
const cliPath = path.join(repoRoot, "packages", "cli", "dist", "index.js");

const dynamicKeyPatterns = [
  /(^|_)duration_ms$/,
  /(^|_)runtime_ms$/,
  /^min_ms$/,
  /^avg_ms$/,
  /^max_ms$/,
  /^started_at$/,
  /^created_at$/,
  /^generated_at$/,
  /^event_id$/,
  /^timestamp$/,
];

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: repoRoot,
    encoding: "utf8",
    maxBuffer: 128 * 1024 * 1024,
    ...options,
  });
  return {
    command,
    args,
    exitCode: result.status ?? 1,
    signal: result.signal,
    stdout: result.stdout ?? "",
    stderr: result.stderr ?? "",
  };
}

function ensureSuccess(result, label) {
  if (result.exitCode !== 0) {
    process.stderr.write(result.stdout);
    process.stderr.write(result.stderr);
    throw new Error(`${label} failed with exit code ${result.exitCode}`);
  }
}

function listCardCases(group) {
  const root = path.join(repoRoot, "examples", group);
  const cases = [];

  function scan(dir, prefix) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const subDir = path.join(dir, entry.name);
      const cardPath = path.join(subDir, "completion-card.yaml");
      if (existsSync(cardPath)) {
        cases.push({
          group,
          name: entry.name,
          cardPath: path.join("examples", group, prefix, entry.name, "completion-card.yaml"),
        });
      } else {
        scan(subDir, path.join(prefix, entry.name));
      }
    }
  }

  scan(root, "");
  return cases.sort((a, b) => a.name.localeCompare(b.name));
}

function isDynamicKey(key) {
  return dynamicKeyPatterns.some((pattern) => pattern.test(key));
}

function normalizeString(value) {
  return value.replace(
    /\.x-harness-mutation-guard-probe-\d+-\d+\.probe/g,
    ".x-harness-mutation-guard-probe-<dynamic>.probe",
  );
}

function normalizeJson(value, key = "") {
  if (isDynamicKey(key) && typeof value === "number") return "<dynamic-number>";
  if (isDynamicKey(key) && typeof value === "string") return "<dynamic-string>";
  if (typeof value === "string") return normalizeString(value);
  if (Array.isArray(value)) {
    return value.map((item) => normalizeJson(item));
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([entryKey, entryValue]) => [
          entryKey,
          normalizeJson(entryValue, entryKey),
        ]),
    );
  }
  return value;
}

function writeText(relativePath, content) {
  const target = path.join(baselineRoot, relativePath);
  mkdirSync(path.dirname(target), { recursive: true });
  writeFileSync(target, content, "utf8");
}

function writeJson(relativePath, value) {
  writeText(relativePath, `${JSON.stringify(value, null, 2)}\n`);
}

function captureJsonCase(manifest, id, relativePath, commandArgs) {
  const result = run(process.execPath, [cliPath, ...commandArgs]);
  let parsed;
  try {
    parsed = JSON.parse(result.stdout);
  } catch (error) {
    writeText(`${relativePath}.stdout.txt`, result.stdout);
    writeText(`${relativePath}.stderr.txt`, result.stderr);
    throw new Error(
      `${id} did not produce parseable JSON: ${error instanceof Error ? error.message : String(error)}`,
    );
  }

  const outputFile = `${relativePath}.json`;
  writeJson(outputFile, normalizeJson(parsed));
  if (result.stderr.trim()) {
    writeText(`${relativePath}.stderr.txt`, result.stderr);
  }
  manifest.cases.push({
    id,
    command: ["node", "packages/cli/dist/index.js", ...commandArgs],
    exit_code: result.exitCode,
    signal: result.signal,
    output: outputFile,
  });
}

function captureTextCase(manifest, id, relativePath, commandArgs) {
  const result = run(process.execPath, [cliPath, ...commandArgs]);
  const outputFile = `${relativePath}.txt`;
  writeText(outputFile, result.stdout);
  if (result.stderr.trim()) {
    writeText(`${relativePath}.stderr.txt`, result.stderr);
  }
  manifest.cases.push({
    id,
    command: ["node", "packages/cli/dist/index.js", ...commandArgs],
    exit_code: result.exitCode,
    signal: result.signal,
    output: outputFile,
  });
}

function parseArgs(argv) {
  const parsed = {
    skipBuild: false,
    out: defaultBaselineRoot,
  };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--skip-build") {
      parsed.skipBuild = true;
    } else if (arg === "--out") {
      const value = argv[index + 1];
      if (!value) throw new Error("--out requires a directory path");
      parsed.out = path.resolve(repoRoot, value);
      index += 1;
    } else {
      throw new Error(`unknown argument: ${arg}`);
    }
  }
  return parsed;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  baselineRoot = args.out;
  if (!args.skipBuild) {
    ensureSuccess(run("npm", ["run", "build"]), "npm run build");
  }

  rmSync(baselineRoot, { recursive: true, force: true });
  mkdirSync(baselineRoot, { recursive: true });

  const manifest = {
    schema_version: 1,
    source_implementation: "typescript",
    source_command: "node packages/cli/dist/index.js",
    normalization: {
      dynamic_fields: dynamicKeyPatterns.map((pattern) => pattern.source),
    },
    cases: [],
  };

  for (const cardCase of [
    ...listCardCases("golden"),
    ...listCardCases("adversarial"),
  ]) {
    captureJsonCase(
      manifest,
      `verify:${cardCase.group}:${cardCase.name}`,
      path.join("verify", cardCase.group, cardCase.name),
      ["verify", "--card", cardCase.cardPath, "--json"],
    );
  }

  captureJsonCase(manifest, "doctor:json", path.join("doctor", "root-json"), [
    "doctor",
    "--root",
    ".",
    "--json",
  ]);

  captureJsonCase(
    manifest,
    "examples:verify:json",
    path.join("examples", "verify-json"),
    ["examples", "verify", "--json"],
  );

  captureTextCase(
    manifest,
    "context:contract",
    path.join("context", "contract"),
    ["context", "--contract"],
  );

  captureTextCase(
    manifest,
    "help:maturity",
    path.join("help", "maturity"),
    ["--help-maturity"],
  );

  captureJsonCase(
    manifest,
    "benchmark:adversarial",
    path.join("benchmark", "adversarial-json"),
    ["benchmark", "--filter", "adversarial", "--json"],
  );

  captureJsonCase(
    manifest,
    "benchmark:mutation-guard",
    path.join("benchmark", "mutation-guard-json"),
    [
      "benchmark",
      "--filter",
      "mutation-guard",
      "--mutation-files",
      "100,1000,5000",
      "--mutation-concurrency",
      "1,4,16,64",
      "--json",
    ],
  );

  writeJson("manifest.json", manifest);
  process.stdout.write(
    `Captured ${manifest.cases.length} TypeScript baseline case(s) in ${path.relative(repoRoot, baselineRoot)}\n`,
  );
}

main();
