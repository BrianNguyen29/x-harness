import { execFile } from "node:child_process";
import { performance } from "node:perf_hooks";
import * as path from "node:path";
import fs from "fs-extra";
import { Command } from "commander";
import { checkPermission } from "../core/permissions.js";
import { runVerifyPipeline } from "../core/verify-pipeline.js";

type BenchmarkName = "verify" | "doctor" | "examples" | "test:fast" | "test";
type BenchmarkFilter = "all" | "latency" | "admission" | "adversarial";
type ExpectedAcceptance = "accepted" | "withheld";
type BenchmarkSuite = "golden" | "adversarial";

interface BenchmarkSample {
  duration_ms: number;
  exit_code: number;
  timed_out: boolean;
}

interface BenchmarkResult {
  command: BenchmarkName;
  iterations: number;
  ok: boolean;
  min_ms: number;
  avg_ms: number;
  max_ms: number;
  exit_codes: number[];
  samples: BenchmarkSample[];
}

interface BenchmarkOptions {
  commands?: string;
  iterations?: string;
  timeoutMs?: string;
  filter?: string;
  adversarial?: boolean;
  updateSnapshots?: boolean;
  json?: boolean;
}

interface BenchmarkCaseDefinition {
  suite: BenchmarkSuite;
  name: string;
  cardPath: string;
  expectedAcceptance: ExpectedAcceptance;
}

interface BenchmarkCaseResult {
  suite: BenchmarkSuite;
  name: string;
  card_path: string;
  expected_acceptance_status: ExpectedAcceptance;
  actual_acceptance_status: ExpectedAcceptance;
  outcome: string;
  accepted: boolean;
  false_accept: boolean;
  false_reject: boolean;
  blocking_predicate: string | null;
  schema_valid: boolean;
  policy_valid: boolean;
  permission_violation_expected: boolean;
  permission_violation_detected: boolean;
  authority_violation_expected: boolean;
  authority_violation_detected: boolean;
  mutation_guard_expected: boolean;
  mutation_guard_detected: boolean;
  runtime_ms: number;
  errors: string[];
  notes: string[];
}

interface BenchmarkSuiteReport {
  suite: BenchmarkSuite;
  cases_total: number;
  expected_pass_count: number;
  expected_block_count: number;
  false_accept_count: number;
  false_reject_count: number;
  runtime_ms: number;
  cases: BenchmarkCaseResult[];
}

interface BenchmarkMetrics {
  false_accept_count: number;
  false_reject_count: number;
  expected_pass_count: number;
  expected_block_count: number;
  schema_validation_pass_rate: number | null;
  policy_validation_pass_rate: number | null;
  episode_packaging_success_rate: number | null;
  mutation_guard_detection_rate: number | null;
  permission_violation_detection_rate: number | null;
  authority_violation_detection_rate: number | null;
  adversarial_false_accept_count: number;
  adversarial_block_rate: number | null;
  runtime_ms: number;
}

interface IntegrationBenchmarkReport {
  ok: boolean;
  runtime_ms: number;
  golden: BenchmarkSuiteReport | null;
  adversarial: BenchmarkSuiteReport | null;
}

const DEFAULT_COMMANDS: BenchmarkName[] = [
  "verify",
  "doctor",
  "examples",
  "test:fast",
];
const ALL_COMMANDS: BenchmarkName[] = [...DEFAULT_COMMANDS, "test"];

async function findRepoRoot(cwd: string): Promise<string> {
  let current = path.resolve(cwd);
  while (true) {
    if (
      (await fs.pathExists(path.join(current, "X_HARNESS.md"))) &&
      (await fs.pathExists(path.join(current, "packages", "cli")))
    ) {
      return current;
    }
    const parent = path.dirname(current);
    if (parent === current) {
      throw new Error(
        "benchmark must be run from an x-harness source checkout"
      );
    }
    current = parent;
  }
}

function parseCommands(value?: string): BenchmarkName[] {
  if (!value || value.trim().length === 0) return DEFAULT_COMMANDS;
  const names = value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  if (names.length === 0) {
    throw new Error(
      `--commands must include at least one command: ${DEFAULT_COMMANDS.join(", ")}`
    );
  }
  const valid = new Set(ALL_COMMANDS);
  const invalid = names.filter((name) => !valid.has(name as BenchmarkName));
  if (invalid.length > 0) {
    throw new Error(
      `Unknown benchmark command(s): ${invalid.join(", ")}. Use: ${ALL_COMMANDS.join(", ")}`
    );
  }
  return names as BenchmarkName[];
}

function parseIterations(value?: string): number {
  const normalized = value ?? "1";
  if (!/^[1-9]\d*$/.test(normalized)) {
    throw new Error("--iterations must be a positive integer");
  }
  return Number.parseInt(normalized, 10);
}

function parseTimeoutMs(value?: string): number {
  const normalized = value ?? "120000";
  if (!/^[1-9]\d*$/.test(normalized)) {
    throw new Error("--timeout-ms must be a positive integer");
  }
  return Number.parseInt(normalized, 10);
}

function parseFilter(value?: string): BenchmarkFilter {
  const normalized = value?.trim() || "all";
  if (
    normalized === "all" ||
    normalized === "latency" ||
    normalized === "admission" ||
    normalized === "adversarial"
  ) {
    return normalized;
  }
  throw new Error(
    "--filter must be one of: all, latency, admission, adversarial"
  );
}

function commandSpec(
  name: BenchmarkName,
  repoRoot: string
): { file: string; args: string[]; cwd: string } {
  const cliPath = path.join(repoRoot, "packages", "cli", "dist", "index.js");
  switch (name) {
    case "verify":
      return {
        file: process.execPath,
        args: [
          cliPath,
          "verify",
          "--card",
          path.join(repoRoot, "examples", "00-minimal", "completion-card.yaml"),
          "--json",
        ],
        cwd: repoRoot,
      };
    case "doctor":
      return {
        file: process.execPath,
        args: [cliPath, "doctor", "--root", repoRoot],
        cwd: repoRoot,
      };
    case "examples":
      return {
        file: process.execPath,
        args: [cliPath, "examples", "verify", "--json"],
        cwd: repoRoot,
      };
    case "test":
      return {
        file: process.platform === "win32" ? "npm.cmd" : "npm",
        args: ["-w", "packages/cli", "run", "test"],
        cwd: repoRoot,
      };
    case "test:fast":
      return {
        file: process.platform === "win32" ? "npm.cmd" : "npm",
        args: ["-w", "packages/cli", "run", "test:fast"],
        cwd: repoRoot,
      };
  }
}

async function runOnce(
  name: BenchmarkName,
  repoRoot: string,
  timeoutMs: number
): Promise<BenchmarkSample> {
  const spec = commandSpec(name, repoRoot);
  const started = performance.now();
  const sample = await new Promise<{ exitCode: number; timedOut: boolean }>(
    (resolve) => {
      execFile(
        spec.file,
        spec.args,
        {
          cwd: spec.cwd,
          maxBuffer: 20 * 1024 * 1024,
          timeout: timeoutMs,
        },
        (error) => {
          if (!error) {
            resolve({ exitCode: 0, timedOut: false });
            return;
          }
          const timedOut = Boolean(error.killed);
          resolve({
            exitCode:
              typeof error.code === "number" ? error.code : timedOut ? 124 : 1,
            timedOut,
          });
        }
      );
    }
  );
  return {
    duration_ms: roundMs(performance.now() - started),
    exit_code: sample.exitCode,
    timed_out: sample.timedOut,
  };
}

function roundMs(value: number): number {
  return Math.round(value * 10) / 10;
}

function roundRate(value: number): number {
  return Math.round(value * 1000) / 1000;
}

function summarize(
  name: BenchmarkName,
  samples: BenchmarkSample[]
): BenchmarkResult {
  const durations = samples.map((sample) => sample.duration_ms);
  const total = durations.reduce((sum, value) => sum + value, 0);
  return {
    command: name,
    iterations: samples.length,
    ok: samples.every((sample) => sample.exit_code === 0),
    min_ms: roundMs(Math.min(...durations)),
    avg_ms: roundMs(total / samples.length),
    max_ms: roundMs(Math.max(...durations)),
    exit_codes: samples.map((sample) => sample.exit_code),
    samples,
  };
}

async function discoverCaseDefinitions(
  repoRoot: string,
  suite: BenchmarkSuite
): Promise<BenchmarkCaseDefinition[]> {
  const suiteDir = path.join(repoRoot, "examples", suite);
  if (!(await fs.pathExists(suiteDir))) return [];

  const entries = await fs.readdir(suiteDir, { withFileTypes: true });
  const definitions: BenchmarkCaseDefinition[] = [];
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    const cardPath = path.join(suiteDir, entry.name, "completion-card.yaml");
    if (!(await fs.pathExists(cardPath))) continue;
    definitions.push({
      suite,
      name: entry.name,
      cardPath,
      expectedAcceptance:
        suite === "golden" ? expectedGoldenAcceptance(entry.name) : "withheld",
    });
  }
  return definitions.sort((a, b) => a.name.localeCompare(b.name));
}

function expectedGoldenAcceptance(name: string): ExpectedAcceptance {
  if (name.startsWith("success-") || name === "multi-agent-success") {
    return "accepted";
  }
  return "withheld";
}

function collectCommandsFromArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value
    .map((item) =>
      item && typeof item === "object"
        ? (item as Record<string, unknown>).command
        : null
    )
    .filter((command): command is string => typeof command === "string");
}

function collectCardCommands(
  card: Record<string, unknown> | undefined
): string[] {
  if (!card) return [];
  const evidence = card.evidence as Record<string, unknown> | undefined;
  const commands = [
    ...collectCommandsFromArray(evidence?.command_evidence),
    ...collectCommandsFromArray(evidence?.verification_artifacts),
  ];
  return [
    ...new Set(commands.map((command) => command.trim()).filter(Boolean)),
  ];
}

async function permissionBenchmarkErrors(
  card: Record<string, unknown> | undefined,
  repoRoot: string,
  tier: string
): Promise<string[]> {
  const errors: string[] = [];
  for (const command of collectCardCommands(card)) {
    const decision = await checkPermission({
      root: repoRoot,
      role: "worker",
      tier,
      command,
    });
    if (decision.status !== "allowed") {
      errors.push(
        `permission benchmark blocked command "${command}": ${decision.reason}`
      );
    }
  }
  return errors;
}

function evidenceConsistencyErrors(
  card: Record<string, unknown> | undefined
): string[] {
  if (!card) return [];
  const errors: string[] = [];
  const evidence = card.evidence as Record<string, unknown> | undefined;
  const commandEvidence = evidence?.command_evidence;
  if (Array.isArray(commandEvidence)) {
    for (const item of commandEvidence) {
      if (!item || typeof item !== "object") continue;
      const record = item as Record<string, unknown>;
      if (typeof record.exit_code === "number" && record.exit_code !== 0) {
        errors.push(
          `benchmark evidence check blocked non-zero command exit_code ${record.exit_code}`
        );
      }
    }
  }

  const artifacts = evidence?.verification_artifacts;
  if (Array.isArray(artifacts)) {
    for (const item of artifacts) {
      if (!item || typeof item !== "object") continue;
      const status = (item as Record<string, unknown>).status;
      if (typeof status === "string" && status !== "passed") {
        errors.push(
          `benchmark evidence check blocked verification artifact status "${status}"`
        );
      }
    }
  }

  const pgvAdvice = card.pgv_advice as Record<string, unknown> | undefined;
  if (pgvAdvice?.admission_authority === true) {
    errors.push("benchmark authority check blocked PGV admission authority");
  }

  return errors;
}

function includesSubstring(items: string[], value: string): boolean {
  return items.some((item) => item.includes(value));
}

function expectedPermissionViolation(
  definition: BenchmarkCaseDefinition
): boolean {
  return definition.name === "hidden-dangerous-command";
}

function expectedMutationGuardDetection(
  definition: BenchmarkCaseDefinition
): boolean {
  return definition.name === "verifier-mutates-source";
}

function expectedAuthorityViolation(
  definition: BenchmarkCaseDefinition
): boolean {
  return definition.name === "spoofed-protected-approval";
}

async function runBenchmarkCase(
  definition: BenchmarkCaseDefinition,
  repoRoot: string
): Promise<BenchmarkCaseResult> {
  const started = performance.now();
  const errors: string[] = [];
  const notes: string[] = [];
  let actualAcceptance: ExpectedAcceptance = "withheld";
  let outcome = "error";
  let accepted = false;
  let blockingPredicate: string | null = "benchmark_error";
  let schemaValid = false;
  let policyValid = false;
  let permissionDetected = false;
  let authorityDetected = false;
  let mutationGuardDetected = false;
  const mutationGuardExpected = expectedMutationGuardDetection(definition);
  const previousTestHooks = process.env.X_HARNESS_ENABLE_TEST_HOOKS;
  const previousInjectMutation = process.env.X_HARNESS_TEST_INJECT_MUTATION;
  const mutationProbePath = path.join(
    repoRoot,
    `.x-harness-mutation-guard-probe-${process.pid}-${Date.now()}.probe`
  );

  try {
    if (mutationGuardExpected) {
      process.env.X_HARNESS_ENABLE_TEST_HOOKS = "1";
      process.env.X_HARNESS_TEST_INJECT_MUTATION = mutationProbePath;
    }
    const result = await runVerifyPipeline(
      {
        card: definition.cardPath,
        governanceEnforced: definition.suite === "adversarial",
        strict: mutationGuardExpected,
      },
      repoRoot
    );
    errors.push(...result.errors);
    notes.push(...result.notes);

    const permissionErrors = await permissionBenchmarkErrors(
      result.card,
      repoRoot,
      result.tier
    );
    const consistencyErrors = evidenceConsistencyErrors(result.card);
    errors.push(...permissionErrors, ...consistencyErrors);

    permissionDetected = permissionErrors.length > 0;
    authorityDetected = includesSubstring(
      result.errors,
      "governance permission violation"
    );
    mutationGuardDetected =
      result.finalBlockingPredicate === "verifier_not_read_only" ||
      includesSubstring(result.errors, "mutation guard blocked");
    schemaValid = !includesSubstring(
      result.errors,
      "completion card validation failed"
    );
    policyValid = !includesSubstring(result.errors, "policy hash error");

    const benchmarkBlocked =
      permissionErrors.length > 0 || consistencyErrors.length > 0;
    actualAcceptance =
      result.finalAcceptance === "accepted" && !benchmarkBlocked
        ? "accepted"
        : "withheld";
    outcome = benchmarkBlocked ? "blocked" : result.finalOutcome;
    accepted = actualAcceptance === "accepted";
    blockingPredicate = benchmarkBlocked
      ? "benchmark_adversarial_guard"
      : result.finalBlockingPredicate;
  } catch (err) {
    errors.push(err instanceof Error ? err.message : String(err));
  } finally {
    if (previousTestHooks === undefined) {
      delete process.env.X_HARNESS_ENABLE_TEST_HOOKS;
    } else {
      process.env.X_HARNESS_ENABLE_TEST_HOOKS = previousTestHooks;
    }
    if (previousInjectMutation === undefined) {
      delete process.env.X_HARNESS_TEST_INJECT_MUTATION;
    } else {
      process.env.X_HARNESS_TEST_INJECT_MUTATION = previousInjectMutation;
    }
    await fs.remove(mutationProbePath);
  }

  const falseAccept =
    definition.expectedAcceptance === "withheld" &&
    actualAcceptance === "accepted";
  const falseReject =
    definition.expectedAcceptance === "accepted" &&
    actualAcceptance !== "accepted";

  return {
    suite: definition.suite,
    name: definition.name,
    card_path: path.relative(repoRoot, definition.cardPath),
    expected_acceptance_status: definition.expectedAcceptance,
    actual_acceptance_status: actualAcceptance,
    outcome,
    accepted,
    false_accept: falseAccept,
    false_reject: falseReject,
    blocking_predicate: blockingPredicate,
    schema_valid: schemaValid,
    policy_valid: policyValid,
    permission_violation_expected: expectedPermissionViolation(definition),
    permission_violation_detected: permissionDetected,
    authority_violation_expected: expectedAuthorityViolation(definition),
    authority_violation_detected: authorityDetected,
    mutation_guard_expected: mutationGuardExpected,
    mutation_guard_detected: mutationGuardDetected,
    runtime_ms: roundMs(performance.now() - started),
    errors,
    notes,
  };
}

function summarizeSuite(
  suite: BenchmarkSuite,
  cases: BenchmarkCaseResult[]
): BenchmarkSuiteReport {
  return {
    suite,
    cases_total: cases.length,
    expected_pass_count: cases.filter(
      (item) => item.expected_acceptance_status === "accepted"
    ).length,
    expected_block_count: cases.filter(
      (item) => item.expected_acceptance_status === "withheld"
    ).length,
    false_accept_count: cases.filter((item) => item.false_accept).length,
    false_reject_count: cases.filter((item) => item.false_reject).length,
    runtime_ms: roundMs(cases.reduce((sum, item) => sum + item.runtime_ms, 0)),
    cases,
  };
}

function suiteCases(
  report: IntegrationBenchmarkReport | null
): BenchmarkCaseResult[] {
  if (!report) return [];
  return [
    ...(report.golden?.cases ?? []),
    ...(report.adversarial?.cases ?? []),
  ];
}

function computeMetrics(
  results: BenchmarkResult[],
  integration: IntegrationBenchmarkReport | null
): BenchmarkMetrics {
  const cases = suiteCases(integration);
  const adversarialCases = integration?.adversarial?.cases ?? [];
  const permissionExpected = cases.filter(
    (item) => item.permission_violation_expected
  );
  const authorityExpected = cases.filter(
    (item) => item.authority_violation_expected
  );
  const mutationExpected = cases.filter((item) => item.mutation_guard_expected);
  const runtimeMs =
    roundMs(
      results.reduce(
        (sum, item) =>
          sum +
          item.samples.reduce((sampleSum, sample) => {
            return sampleSum + sample.duration_ms;
          }, 0),
        0
      )
    ) + (integration?.runtime_ms ?? 0);

  return {
    false_accept_count: cases.filter((item) => item.false_accept).length,
    false_reject_count: cases.filter((item) => item.false_reject).length,
    expected_pass_count: cases.filter(
      (item) => item.expected_acceptance_status === "accepted"
    ).length,
    expected_block_count: cases.filter(
      (item) => item.expected_acceptance_status === "withheld"
    ).length,
    schema_validation_pass_rate:
      cases.length === 0
        ? null
        : roundRate(
            cases.filter((item) => item.schema_valid).length / cases.length
          ),
    policy_validation_pass_rate:
      cases.length === 0
        ? null
        : roundRate(
            cases.filter((item) => item.policy_valid).length / cases.length
          ),
    episode_packaging_success_rate: null,
    mutation_guard_detection_rate:
      mutationExpected.length === 0
        ? null
        : roundRate(
            mutationExpected.filter((item) => item.mutation_guard_detected)
              .length / mutationExpected.length
          ),
    permission_violation_detection_rate:
      permissionExpected.length === 0
        ? null
        : roundRate(
            permissionExpected.filter(
              (item) => item.permission_violation_detected
            ).length / permissionExpected.length
          ),
    authority_violation_detection_rate:
      authorityExpected.length === 0
        ? null
        : roundRate(
            authorityExpected.filter(
              (item) => item.authority_violation_detected
            ).length / authorityExpected.length
          ),
    adversarial_false_accept_count: adversarialCases.filter(
      (item) => item.false_accept
    ).length,
    adversarial_block_rate:
      adversarialCases.length === 0
        ? null
        : roundRate(
            adversarialCases.filter(
              (item) => item.actual_acceptance_status === "withheld"
            ).length / adversarialCases.length
          ),
    runtime_ms: roundMs(runtimeMs),
  };
}

async function runIntegrationBenchmark(
  repoRoot: string,
  includeGolden: boolean,
  includeAdversarial: boolean
): Promise<IntegrationBenchmarkReport | null> {
  if (!includeGolden && !includeAdversarial) return null;
  const started = performance.now();
  const goldenDefinitions = includeGolden
    ? await discoverCaseDefinitions(repoRoot, "golden")
    : [];
  const adversarialDefinitions = includeAdversarial
    ? await discoverCaseDefinitions(repoRoot, "adversarial")
    : [];
  const goldenCases: BenchmarkCaseResult[] = [];
  for (const definition of goldenDefinitions) {
    goldenCases.push(await runBenchmarkCase(definition, repoRoot));
  }
  const adversarialCases: BenchmarkCaseResult[] = [];
  for (const definition of adversarialDefinitions) {
    adversarialCases.push(await runBenchmarkCase(definition, repoRoot));
  }

  const golden = includeGolden ? summarizeSuite("golden", goldenCases) : null;
  const adversarial = includeAdversarial
    ? summarizeSuite("adversarial", adversarialCases)
    : null;
  const falseAcceptCount =
    (golden?.false_accept_count ?? 0) + (adversarial?.false_accept_count ?? 0);
  const falseRejectCount =
    (golden?.false_reject_count ?? 0) + (adversarial?.false_reject_count ?? 0);
  return {
    ok: falseAcceptCount === 0 && falseRejectCount === 0,
    runtime_ms: roundMs(performance.now() - started),
    golden,
    adversarial,
  };
}

function detectionMetricsOk(metrics: BenchmarkMetrics): boolean {
  return [
    metrics.mutation_guard_detection_rate,
    metrics.permission_violation_detection_rate,
    metrics.authority_violation_detection_rate,
  ].every((rate) => rate === null || rate >= 1);
}

function renderMarkdown(
  results: BenchmarkResult[],
  integration: IntegrationBenchmarkReport | null,
  metrics: BenchmarkMetrics
): void {
  console.log("# x-harness Latency Report");
  console.log("");
  if (results.length === 0) {
    console.log("_Latency commands skipped by benchmark filter._");
  } else {
    console.log(
      "| command | iterations | min_ms | avg_ms | max_ms | exit_codes |"
    );
    console.log("| :-- | --: | --: | --: | --: | :-- |");
    for (const result of results) {
      console.log(
        `| ${result.command} | ${result.iterations} | ${result.min_ms} | ${result.avg_ms} | ${result.max_ms} | ${result.exit_codes.join(",")} |`
      );
    }
  }

  if (!integration) return;
  console.log("");
  console.log("# x-harness Integration Benchmark");
  console.log("");
  console.log(`- false_accept_count: ${metrics.false_accept_count}`);
  console.log(
    `- adversarial_false_accept_count: ${metrics.adversarial_false_accept_count}`
  );
  console.log(`- false_reject_count: ${metrics.false_reject_count}`);
  console.log(
    `- adversarial_block_rate: ${metrics.adversarial_block_rate ?? "n/a"}`
  );
  console.log(
    `- authority_violation_detection_rate: ${metrics.authority_violation_detection_rate ?? "n/a"}`
  );
  console.log("");
  console.log(
    "| suite | cases | expected_pass | expected_block | false_accept | false_reject | runtime_ms |"
  );
  console.log("| :-- | --: | --: | --: | --: | --: | --: |");
  for (const suite of [integration.golden, integration.adversarial]) {
    if (!suite) continue;
    console.log(
      `| ${suite.suite} | ${suite.cases_total} | ${suite.expected_pass_count} | ${suite.expected_block_count} | ${suite.false_accept_count} | ${suite.false_reject_count} | ${suite.runtime_ms} |`
    );
  }
}

export function benchmarkCommand(): Command {
  return new Command("benchmark")
    .description(
      "Measure latency and run golden/adversarial admission benchmarks"
    )
    .option(
      "--commands <list>",
      "Comma-separated command list: verify,doctor,examples,test:fast,test",
      DEFAULT_COMMANDS.join(",")
    )
    .option("--iterations <n>", "Iterations per command", "1")
    .option("--timeout-ms <n>", "Timeout per measured command", "120000")
    .option(
      "--filter <name>",
      "Benchmark filter: all, latency, admission, adversarial",
      "all"
    )
    .option("--adversarial", "Include adversarial examples corpus", false)
    .option(
      "--update-snapshots",
      "Reserved for human-approved benchmark boundary updates",
      false
    )
    .option("--json", "Output JSON instead of Markdown", false)
    .action(async (opts: BenchmarkOptions) => {
      let names: BenchmarkName[];
      let iterations: number;
      let timeoutMs: number;
      let filter: BenchmarkFilter;
      try {
        if (opts.updateSnapshots) {
          throw new Error(
            "--update-snapshots requires a human-approved boundary change; this command will not update snapshots automatically"
          );
        }
        names = parseCommands(opts.commands);
        iterations = parseIterations(opts.iterations);
        timeoutMs = parseTimeoutMs(opts.timeoutMs);
        filter = parseFilter(opts.filter);
      } catch (err) {
        console.error(err instanceof Error ? err.message : String(err));
        process.exit(2);
      }

      let repoRoot: string;
      try {
        repoRoot = await findRepoRoot(process.cwd());
      } catch (err) {
        console.error(err instanceof Error ? err.message : String(err));
        process.exit(2);
      }

      const includeLatency = filter === "all" || filter === "latency";
      const includeAdversarial =
        filter === "adversarial" || opts.adversarial === true;
      const includeGolden =
        filter === "all" || filter === "admission" || opts.adversarial === true;
      const results: BenchmarkResult[] = [];
      if (includeLatency) {
        for (const name of names) {
          const samples: BenchmarkSample[] = [];
          for (let i = 0; i < iterations; i += 1) {
            samples.push(await runOnce(name, repoRoot, timeoutMs));
          }
          results.push(summarize(name, samples));
        }
      }
      const integration = await runIntegrationBenchmark(
        repoRoot,
        includeGolden,
        includeAdversarial
      );
      const metrics = computeMetrics(results, integration);
      const ok =
        results.every((result) => result.ok) &&
        metrics.false_accept_count === 0 &&
        metrics.adversarial_false_accept_count === 0 &&
        metrics.false_reject_count === 0 &&
        detectionMetricsOk(metrics);

      if (opts.json) {
        console.log(
          JSON.stringify(
            {
              ok,
              generated_at: new Date().toISOString(),
              iterations,
              timeout_ms: timeoutMs,
              filter,
              results,
              integration,
              metrics,
            },
            null,
            2
          )
        );
      } else {
        renderMarkdown(results, integration, metrics);
      }

      process.exit(ok ? 0 : 1);
    });
}
