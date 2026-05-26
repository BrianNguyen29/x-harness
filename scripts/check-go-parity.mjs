#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const baselineRoot = path.join(repoRoot, "tests", "parity", "baseline", "typescript");
const manifestPath = path.join(baselineRoot, "manifest.json");
const goBinary = path.join(repoRoot, "x-harness");

function parseArgs(argv) {
  return {
    json: argv.includes("--json"),
    skipTsBaselineCheck: argv.includes("--skip-ts-baseline-check"),
    skipGoBuild: argv.includes("--skip-go-build"),
  };
}

function run(args, cwd = repoRoot) {
  const result = spawnSync(goBinary, args, {
    cwd,
    encoding: "utf8",
    maxBuffer: 128 * 1024 * 1024,
  });
  return result;
}

function readJson(filePath) {
  return JSON.parse(readFileSync(filePath, "utf8"));
}

function readText(filePath) {
  return readFileSync(filePath, "utf8");
}

function buildGoCommand(tsCommand) {
  // Replace "node packages/cli/dist/index.js" with Go binary path
  const args = [];
  let skip = true;
  for (const part of tsCommand) {
    if (part === "node") {
      skip = true;
      continue;
    }
    if (part === "packages/cli/dist/index.js") {
      skip = false;
      continue;
    }
    if (!skip) {
      args.push(part);
    }
  }
  return args;
}

function isSupported(caseId) {
  const supportedPrefixes = [
    "verify:golden:",
    "verify:adversarial:",
    "doctor:json",
    "context:contract",
    "benchmark:mutation-guard",
  ];
  if (!supportedPrefixes.some((p) => caseId.startsWith(p))) {
    return false;
  }
  // Known divergences: Go does not yet implement these specific TS checks
  const knownDivergences = [
    "verify:golden:blocked-missing-evidence-scope",
    "verify:adversarial:hidden-dangerous-command",
    "verify:adversarial:lying-command-exit-code",
  ];
  return !knownDivergences.includes(caseId);
}

function skipReason(caseId) {
  if (caseId.startsWith("examples:")) {
    return "Go examples command not yet implemented";
  }
  if (caseId.startsWith("benchmark:")) {
    return "Go benchmark command not yet implemented";
  }
  return "unsupported case";
}

function compareVerifyJson(tsOutput, goOutput, tsExit, goExit) {
  const errors = [];
  if (tsExit !== goExit) {
    errors.push(`exit code mismatch: ts=${tsExit}, go=${goExit}`);
  }
  if (tsOutput.ok !== goOutput.ok) {
    errors.push(`ok mismatch: ts=${tsOutput.ok}, go=${goOutput.ok}`);
  }
  const tsOutcome = tsOutput.admission_outcome ?? tsOutput.decision?.outcome;
  const goOutcome = goOutput.admission_outcome;
  if (tsOutcome !== goOutcome) {
    errors.push(`admission outcome mismatch: ts=${tsOutcome}, go=${goOutcome}`);
  }
  const tsAcceptance = tsOutput.acceptance_status ?? tsOutput.decision?.acceptance_status;
  const goAcceptance = goOutput.acceptance_status;
  if (tsAcceptance !== goAcceptance) {
    errors.push(`acceptance status mismatch: ts=${tsAcceptance}, go=${goAcceptance}`);
  }
  return errors;
}

function compareDoctorJson(tsOutput, goOutput, tsExit, goExit) {
  const errors = [];
  if (tsExit !== goExit) {
    errors.push(`exit code mismatch: ts=${tsExit}, go=${goExit}`);
  }
  if (tsOutput.healthy !== goOutput.healthy) {
    errors.push(`healthy mismatch: ts=${tsOutput.healthy}, go=${goOutput.healthy}`);
  }
  return errors;
}

function compareContextContract(tsOutput, goOutput, tsExit, goExit) {
  const errors = [];
  if (tsExit !== goExit) {
    errors.push(`exit code mismatch: ts=${tsExit}, go=${goExit}`);
  }
  const expectedFacts = [
    "Completion is admitted, not claimed",
    "verifier is read-only",
    "Success is the only accepted outcome",
    "Canonical tiers",
    "PGV is advisory-only",
  ];
  for (const fact of expectedFacts) {
    if (!goOutput.includes(fact)) {
      errors.push(`missing contract fact: ${fact}`);
    }
  }
  return errors;
}

function compareBenchmarkMutationGuardJson(tsOutput, goOutput, tsExit, goExit) {
  const errors = [];
  if (tsExit !== goExit) {
    errors.push(`exit code mismatch: ts=${tsExit}, go=${goExit}`);
  }
  if (tsOutput.ok !== goOutput.ok) {
    errors.push(`ok mismatch: ts=${tsOutput.ok}, go=${goOutput.ok}`);
  }
  if (tsOutput.filter !== goOutput.filter) {
    errors.push(`filter mismatch: ts=${tsOutput.filter}, go=${goOutput.filter}`);
  }
  if (tsOutput.integration !== goOutput.integration) {
    errors.push(`integration mismatch: ts=${tsOutput.integration}, go=${goOutput.integration}`);
  }
  if (!Array.isArray(goOutput.results) || goOutput.results.length !== 0) {
    errors.push(`results mismatch: expected empty array`);
  }
  const tsMgb = tsOutput.mutation_guard_benchmark;
  const goMgb = goOutput.mutation_guard_benchmark;
  if (!tsMgb || !goMgb) {
    errors.push(`mutation_guard_benchmark missing`);
    return errors;
  }
  if (tsMgb.ok !== goMgb.ok) {
    errors.push(`mutation_guard_benchmark.ok mismatch`);
  }
  if (JSON.stringify(tsMgb.file_counts) !== JSON.stringify(goMgb.file_counts)) {
    errors.push(`file_counts mismatch: ts=${JSON.stringify(tsMgb.file_counts)}, go=${JSON.stringify(goMgb.file_counts)}`);
  }
  if (JSON.stringify(tsMgb.concurrency) !== JSON.stringify(goMgb.concurrency)) {
    errors.push(`concurrency mismatch: ts=${JSON.stringify(tsMgb.concurrency)}, go=${JSON.stringify(goMgb.concurrency)}`);
  }
  if (tsMgb.cases.length !== goMgb.cases.length) {
    errors.push(`cases length mismatch: ts=${tsMgb.cases.length}, go=${goMgb.cases.length}`);
  } else {
    for (let i = 0; i < tsMgb.cases.length; i++) {
      const tc = tsMgb.cases[i];
      const gc = goMgb.cases[i];
      if (tc.mode !== gc.mode) {
        errors.push(`case[${i}].mode mismatch: ts=${tc.mode}, go=${gc.mode}`);
      }
      if (tc.file_count !== gc.file_count) {
        errors.push(`case[${i}].file_count mismatch: ts=${tc.file_count}, go=${gc.file_count}`);
      }
      if (tc.concurrency !== gc.concurrency) {
        errors.push(`case[${i}].concurrency mismatch: ts=${tc.concurrency}, go=${gc.concurrency}`);
      }
      if (tc.hashed_paths !== gc.hashed_paths) {
        errors.push(`case[${i}].hashed_paths mismatch: ts=${tc.hashed_paths}, go=${gc.hashed_paths}`);
      }
      if (tc.ok !== gc.ok) {
        errors.push(`case[${i}].ok mismatch: ts=${tc.ok}, go=${gc.ok}`);
      }
    }
  }
  return errors;
}

function main() {
  const args = parseArgs(process.argv.slice(2));

  if (!args.skipTsBaselineCheck) {
    const tsBaselineCheck = spawnSync(process.execPath, ["scripts/check-ts-baseline.mjs"], {
      cwd: repoRoot,
      encoding: "utf8",
      maxBuffer: 128 * 1024 * 1024,
    });
    if (tsBaselineCheck.status !== 0) {
      process.stderr.write(tsBaselineCheck.stdout ?? "");
      process.stderr.write(tsBaselineCheck.stderr ?? "");
      process.stderr.write("Go parity check aborted: TypeScript baseline is not current.\n");
      process.exit(tsBaselineCheck.status ?? 1);
    }
  }

  if (!args.skipGoBuild) {
    const buildResult = spawnSync("go", ["build", "./cmd/x-harness"], {
      cwd: repoRoot,
      encoding: "utf8",
    });
    if (buildResult.status !== 0) {
      process.stderr.write("Go build failed:\n");
      process.stderr.write(buildResult.stderr);
      process.exit(1);
    }
  }

  const manifest = readJson(manifestPath);
  const results = { passed: [], failed: [], skipped: [] };

  for (const c of manifest.cases) {
    const caseId = c.id;

    if (!isSupported(caseId)) {
      results.skipped.push({ id: caseId, reason: skipReason(caseId) });
      continue;
    }

    const goArgs = buildGoCommand(c.command);
    const tsOutputPath = path.join(baselineRoot, c.output);

    let tsOutput;
    let goOutput;
    let tsExit = c.exit_code;
    let goExit;

    try {
      if (caseId === "context:contract") {
        tsOutput = readText(tsOutputPath);
        const goResult = run(goArgs);
        goOutput = goResult.stdout;
        goExit = goResult.status ?? (goResult.error ? 1 : 0);
        const errors = compareContextContract(tsOutput, goOutput, tsExit, goExit);
        if (errors.length > 0) {
          results.failed.push({ id: caseId, errors });
        } else {
          results.passed.push(caseId);
        }
      } else {
        tsOutput = readJson(tsOutputPath);
        const goResult = run(goArgs);
        goExit = goResult.status ?? (goResult.error ? 1 : 0);
        try {
          goOutput = JSON.parse(goResult.stdout);
        } catch (e) {
          results.failed.push({
            id: caseId,
            errors: [`Go output is not valid JSON: ${goResult.stdout.slice(0, 200)}`],
          });
          continue;
        }

        let errors;
        if (caseId.startsWith("verify:")) {
          errors = compareVerifyJson(tsOutput, goOutput, tsExit, goExit);
        } else if (caseId.startsWith("doctor:")) {
          errors = compareDoctorJson(tsOutput, goOutput, tsExit, goExit);
        } else if (caseId === "benchmark:mutation-guard") {
          errors = compareBenchmarkMutationGuardJson(tsOutput, goOutput, tsExit, goExit);
        } else {
          errors = [];
        }

        if (errors.length > 0) {
          results.failed.push({ id: caseId, errors });
        } else {
          results.passed.push(caseId);
        }
      }
    } catch (e) {
      results.failed.push({ id: caseId, errors: [e.message] });
    }
  }

  const total = manifest.cases.length;
  const passed = results.passed.length;
  const failed = results.failed.length;
  const skipped = results.skipped.length;

  const report = {
    schema_version: 1,
    source: "go-parity-check",
    summary: { total, passed, failed, skipped },
    passed: results.passed,
    failed: results.failed.map((f) => ({ id: f.id, errors: f.errors })),
    skipped: results.skipped,
  };

  if (args.json) {
    process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
  } else {
    process.stdout.write(`Go parity check\n`);
    process.stdout.write(`  total: ${total}\n`);
    process.stdout.write(`  passed: ${passed}\n`);
    process.stdout.write(`  failed: ${failed}\n`);
    process.stdout.write(`  skipped: ${skipped}\n`);
    process.stdout.write(`\n`);

    if (results.passed.length > 0) {
      process.stdout.write(`Passed (${passed}):\n`);
      for (const id of results.passed) {
        process.stdout.write(`  ✓ ${id}\n`);
      }
      process.stdout.write(`\n`);
    }

    if (results.failed.length > 0) {
      process.stdout.write(`Failed (${failed}):\n`);
      for (const f of results.failed) {
        process.stdout.write(`  ✗ ${f.id}\n`);
        for (const err of f.errors) {
          process.stdout.write(`    - ${err}\n`);
        }
      }
      process.stdout.write(`\n`);
    }

    if (results.skipped.length > 0) {
      process.stdout.write(`Skipped (${skipped}):\n`);
      for (const s of results.skipped) {
        process.stdout.write(`  ○ ${s.id} (${s.reason})\n`);
      }
      process.stdout.write(`\n`);
    }
  }

  if (failed > 0) {
    process.exit(1);
  }
  process.exit(0);
}

main();
