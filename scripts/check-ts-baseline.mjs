#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import {
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  statSync,
} from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const expectedRoot = path.join(
  repoRoot,
  "tests",
  "parity",
  "baseline",
  "typescript",
);

function run(command, args) {
  const result = spawnSync(command, args, {
    cwd: repoRoot,
    encoding: "utf8",
    maxBuffer: 128 * 1024 * 1024,
  });
  if (result.status !== 0) {
    process.stderr.write(result.stdout ?? "");
    process.stderr.write(result.stderr ?? "");
    throw new Error(`${command} ${args.join(" ")} failed`);
  }
  return result;
}

function listFiles(root) {
  const files = [];
  function walk(current) {
    for (const entry of readdirSync(current)) {
      const absolute = path.join(current, entry);
      const stat = statSync(absolute);
      if (stat.isDirectory()) {
        walk(absolute);
      } else if (stat.isFile()) {
        files.push(path.relative(root, absolute).replace(/\\/g, "/"));
      }
    }
  }
  walk(root);
  return files.sort();
}

function compareTrees(expected, actual) {
  const expectedFiles = listFiles(expected);
  const actualFiles = listFiles(actual);
  const expectedSet = new Set(expectedFiles);
  const actualSet = new Set(actualFiles);
  const missing = expectedFiles.filter((file) => !actualSet.has(file));
  const extra = actualFiles.filter((file) => !expectedSet.has(file));
  const changed = expectedFiles.filter((file) => {
    if (!actualSet.has(file)) return false;
    return (
      readFileSync(path.join(expected, file), "utf8") !==
      readFileSync(path.join(actual, file), "utf8")
    );
  });
  return { missing, extra, changed };
}

function main() {
  const tempRoot = mkdtempSync(path.join(os.tmpdir(), "xh-ts-baseline-"));
  try {
    run("npm", ["run", "build"]);
    run(process.execPath, [
      "scripts/capture-ts-baseline.mjs",
      "--skip-build",
      "--out",
      tempRoot,
    ]);
    const diff = compareTrees(expectedRoot, tempRoot);
    if (diff.missing.length || diff.extra.length || diff.changed.length) {
      process.stderr.write("TypeScript parity baseline drift detected.\n");
      if (diff.missing.length) {
        process.stderr.write(`missing: ${diff.missing.join(", ")}\n`);
      }
      if (diff.extra.length) {
        process.stderr.write(`extra: ${diff.extra.join(", ")}\n`);
      }
      if (diff.changed.length) {
        process.stderr.write(`changed: ${diff.changed.join(", ")}\n`);
      }
      process.exit(1);
    }
    process.stdout.write("TypeScript parity baseline is up to date.\n");
  } finally {
    rmSync(tempRoot, { recursive: true, force: true });
  }
}

main();
