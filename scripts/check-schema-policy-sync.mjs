#!/usr/bin/env node
/**
 * Schema / Policy Sync Check
 *
 * Compares root schemas/ and policies/ against their packages/cli/ counterparts.
 * Reports missing files, content mismatches, and extra files.
 *
 * Usage:
 *   node scripts/check-schema-policy-sync.mjs
 *   node scripts/check-schema-policy-sync.mjs --json
 *
 * Exit code: 0 if all tracked files match, 1 otherwise.
 */

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const syncIgnoreFile = join(repoRoot, "scripts", "sync-ignore.json");

function loadIgnoreSet() {
  try {
    const raw = readFileSync(syncIgnoreFile, "utf8");
    const list = JSON.parse(raw);
    return new Set(list.map((s) => String(s)));
  } catch {
    return new Set();
  }
}

function walkDir(dir, base = dir) {
  const results = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const rel = relative(base, full);
    const st = statSync(full);
    if (st.isDirectory()) {
      results.push(...walkDir(full, base));
    } else {
      results.push(rel);
    }
  }
  return results;
}

function comparePair(name, rootDir, cliDir) {
  const rootFiles = new Set(walkDir(rootDir));
  const cliFiles = new Set(walkDir(cliDir));
  const ignoreSet = loadIgnoreSet();

  const missing = [];
  const mismatched = [];
  const extra = [];
  const ignored = [];
  const matched = [];

  for (const f of rootFiles) {
    if (ignoreSet.has(f)) {
      ignored.push(f);
      continue;
    }
    if (!cliFiles.has(f)) {
      missing.push(f);
      continue;
    }
    const rootData = readFileSync(join(rootDir, f), "utf8");
    const cliData = readFileSync(join(cliDir, f), "utf8");
    if (rootData !== cliData) {
      mismatched.push(f);
    } else {
      matched.push(f);
    }
  }

  for (const f of cliFiles) {
    if (ignoreSet.has(f)) continue;
    if (!rootFiles.has(f)) {
      extra.push(f);
    }
  }

  return { name, missing, mismatched, extra, ignored, matched };
}

function main() {
  const args = process.argv.slice(2);
  const jsonMode = args.includes("--json");

  const schemas = comparePair(
    "schemas",
    join(repoRoot, "schemas"),
    join(repoRoot, "packages", "cli", "schemas")
  );
  const policies = comparePair(
    "policies",
    join(repoRoot, "policies"),
    join(repoRoot, "packages", "cli", "policies")
  );

  const results = [schemas, policies];
  let ok = true;

  for (const r of results) {
    if (r.missing.length || r.mismatched.length) {
      ok = false;
    }
  }

  if (jsonMode) {
    console.log(JSON.stringify({ ok, results }, null, 2));
    process.exit(ok ? 0 : 1);
  }

  for (const r of results) {
    console.log(`--- ${r.name} ---`);
    console.log(`  matched:   ${r.matched.length}`);
    if (r.ignored.length) {
      console.log(`  ignored:   ${r.ignored.length} (${r.ignored.join(", ")})`);
    }
    if (r.missing.length) {
      console.log(`  MISSING:   ${r.missing.length}`);
      for (const f of r.missing) console.log(`    - ${f}`);
    }
    if (r.mismatched.length) {
      console.log(`  MISMATCHED: ${r.mismatched.length}`);
      for (const f of r.mismatched) console.log(`    - ${f}`);
    }
    if (r.extra.length) {
      console.log(`  extra:     ${r.extra.length} (informational)`);
      for (const f of r.extra) console.log(`    - ${f}`);
    }
  }

  console.log();
  if (ok) {
    console.log("OK: all tracked schema and policy files are synchronized.");
  } else {
    console.log("FAIL: schema or policy drift detected.");
  }
  process.exit(ok ? 0 : 1);
}

main();
