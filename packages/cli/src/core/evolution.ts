import * as path from "node:path";
import fs from "fs-extra";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";

export interface EvolutionInvariant {
  id: string;
  statement: string;
  protected_paths?: string[];
  forbidden_changes?: string[];
  benchmark_required?: boolean;
}

export interface EvolutionConstitution {
  version: number;
  invariants: EvolutionInvariant[];
}

export interface EvolutionCandidate {
  schema_version: number;
  candidate_id: string;
  base_commit?: string;
  component_ids?: string[];
  change_summary?: string;
  prediction?: Record<string, unknown>;
  metrics_before?: Record<string, unknown>;
  metrics_after?: Record<string, unknown>;
  regression_budget?: Record<string, unknown>;
  forbidden_changes?: string[];
  touched_paths?: string[];
  constitution?: Record<string, unknown>;
  promotion_status?: string;
  rollback?: Record<string, unknown>;
}

export interface EvolutionBudget {
  evolution_budget: {
    enabled: boolean;
    max_candidates_per_day: number;
    max_runtime_minutes_per_run: number;
    max_cost_usd_per_run: number;
    min_failure_pattern_count: number;
    require_h2_maturity: boolean;
    require_adversarial_suite: boolean;
  };
}

export interface ConstitutionCheckResult {
  ok: boolean;
  status: "passed" | "failed";
  candidate_id: string;
  constitution_path: string;
  candidate_path: string;
  violations: string[];
  checked_invariants: string[];
  admission_authority: false;
}

export interface EvolutionRequestResult {
  ok: boolean;
  status: "disabled" | "proposed" | "written" | "failed";
  path: string | null;
  message: string;
  admission_authority: false;
}

function evolutionRoot(root: string): string {
  return path.join(root, "tools", "experimental", "evolve");
}

function defaultConstitutionPath(root: string): string {
  return path.join(evolutionRoot(root), "constitution.yaml");
}

function defaultBudgetPath(root: string): string {
  return path.join(evolutionRoot(root), "evolution-budget.yaml");
}

export async function loadEvolutionConstitution(
  root: string,
  constitutionPath?: string
): Promise<{ path: string; constitution: EvolutionConstitution }> {
  const resolved = path.resolve(
    root,
    constitutionPath ?? defaultConstitutionPath(root)
  );
  const data = (await readYamlOrJson(resolved)) as EvolutionConstitution;
  const schema = await loadSchema("evolution-constitution");
  const validate = compileSchema(schema);
  if (!validate(data)) {
    throw new Error(
      `constitution schema validation failed: ${(validate.errors ?? [])
        .map((err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`)
        .join("; ")}`
    );
  }
  return { path: resolved, constitution: data };
}

export async function loadEvolutionBudget(
  root: string
): Promise<EvolutionBudget> {
  const budgetPath = defaultBudgetPath(root);
  return (await readYamlOrJson(budgetPath)) as EvolutionBudget;
}

async function resolveCandidatePath(
  root: string,
  candidate: string
): Promise<string> {
  const direct = path.resolve(root, candidate);
  if (await fs.pathExists(direct)) return direct;

  const candidatesDir = path.join(evolutionRoot(root), "candidates");
  for (const suffix of [
    "candidate.yaml",
    "candidate.yml",
    `${candidate}.yaml`,
  ]) {
    const p = path.join(candidatesDir, candidate, suffix);
    if (await fs.pathExists(p)) return p;
  }
  const flat = path.join(candidatesDir, `${candidate}.yaml`);
  if (await fs.pathExists(flat)) return flat;

  throw new Error(`candidate not found: ${candidate}`);
}

export async function loadEvolutionCandidate(
  root: string,
  candidate: string
): Promise<{ path: string; candidate: EvolutionCandidate }> {
  const candidatePath = await resolveCandidatePath(root, candidate);
  const data = (await readYamlOrJson(candidatePath)) as EvolutionCandidate;
  if (!data.candidate_id) {
    throw new Error("candidate manifest missing candidate_id");
  }
  return { path: candidatePath, candidate: data };
}

function listIncludes(items: string[] | undefined, item: string): boolean {
  return (items ?? []).includes(item);
}

function pathMatches(pattern: string, filePath: string): boolean {
  if (pattern.endsWith("/**")) {
    return (
      filePath === pattern.slice(0, -3) ||
      filePath.startsWith(pattern.slice(0, -2))
    );
  }
  return pattern === filePath;
}

function numberMetric(
  record: Record<string, unknown> | undefined,
  key: string
): number {
  const value = record?.[key];
  return typeof value === "number" ? value : 0;
}

export async function checkEvolutionConstitution(input: {
  root: string;
  candidate: string;
  constitutionPath?: string;
}): Promise<ConstitutionCheckResult> {
  const { path: constitutionPath, constitution } =
    await loadEvolutionConstitution(input.root, input.constitutionPath);
  const { path: candidatePath, candidate } = await loadEvolutionCandidate(
    input.root,
    input.candidate
  );
  const violations: string[] = [];
  const forbiddenChanges = candidate.forbidden_changes ?? [];
  const touchedPaths = candidate.touched_paths ?? [];

  for (const invariant of constitution.invariants) {
    for (const forbidden of invariant.forbidden_changes ?? []) {
      if (listIncludes(forbiddenChanges, forbidden)) {
        violations.push(
          `${invariant.id}: candidate declares forbidden change ${forbidden}`
        );
      }
    }
    for (const protectedPath of invariant.protected_paths ?? []) {
      if (touchedPaths.some((item) => pathMatches(protectedPath, item))) {
        violations.push(
          `${invariant.id}: candidate touches protected path ${protectedPath}`
        );
      }
    }
    if (invariant.benchmark_required) {
      const afterFalseAccept = numberMetric(
        candidate.metrics_after,
        "false_accept_count"
      );
      const afterAdversarialFalseAccept = numberMetric(
        candidate.metrics_after,
        "adversarial_false_accept_count"
      );
      if (afterFalseAccept > 0 || afterAdversarialFalseAccept > 0) {
        violations.push(
          `${invariant.id}: benchmark false accepts must remain zero`
        );
      }
    }
  }

  const beforeFalseAccept = numberMetric(
    candidate.metrics_before,
    "false_accept_count"
  );
  const afterFalseAccept = numberMetric(
    candidate.metrics_after,
    "false_accept_count"
  );
  if (afterFalseAccept > beforeFalseAccept) {
    violations.push("false_accept_count increased from baseline");
  }

  return {
    ok: violations.length === 0,
    status: violations.length === 0 ? "passed" : "failed",
    candidate_id: candidate.candidate_id,
    constitution_path: constitutionPath,
    candidate_path: candidatePath,
    violations,
    checked_invariants: constitution.invariants.map((item) => item.id),
    admission_authority: false,
  };
}

export async function evaluateEvolutionBudget(
  root: string
): Promise<EvolutionRequestResult> {
  const budget = await loadEvolutionBudget(root);
  if (!budget.evolution_budget.enabled) {
    return {
      ok: true,
      status: "disabled",
      path: null,
      message: "evolution budget is disabled; no candidate loop will run",
      admission_authority: false,
    };
  }
  return {
    ok: true,
    status: "proposed",
    path: null,
    message:
      "evolution budget enabled; external model loop is not implemented in local MVP",
    admission_authority: false,
  };
}

export function renderChangeRequest(input: {
  kind: "proposal" | "promotion" | "rollback" | "analysis";
  candidateId?: string;
  component?: string;
  summary: string;
  constitution?: ConstitutionCheckResult;
}): string {
  const lines = [
    `# x-harness Evolution ${input.kind}`,
    "",
    `summary: ${input.summary}`,
    `admission_authority: false`,
  ];
  if (input.component) lines.push(`component: ${input.component}`);
  if (input.candidateId) lines.push(`candidate_id: ${input.candidateId}`);
  if (input.constitution) {
    lines.push(`constitution_status: ${input.constitution.status}`);
    if (input.constitution.violations.length > 0) {
      lines.push("", "## Violations");
      for (const violation of input.constitution.violations) {
        lines.push(`- ${violation}`);
      }
    }
  }
  lines.push(
    "",
    "## Boundary",
    "",
    "This file is a change request only. It does not promote, merge, or mutate harness policy."
  );
  return `${lines.join("\n")}\n`;
}

export async function writeChangeRequest(
  root: string,
  content: string,
  outPath?: string
): Promise<string> {
  const baseDir = path.resolve(
    root,
    ".x-harness",
    "evolution",
    "change-requests"
  );
  const target = outPath
    ? path.resolve(root, outPath)
    : path.join(baseDir, `request-${Date.now()}.md`);
  const basePrefix = baseDir.endsWith(path.sep)
    ? baseDir
    : `${baseDir}${path.sep}`;
  if (target === baseDir || !target.startsWith(basePrefix)) {
    throw new Error(
      "evolution change requests must be written under .x-harness/evolution/change-requests"
    );
  }
  if (await fs.pathExists(target)) {
    throw new Error(
      `evolution change request already exists; refusing to overwrite: ${target}`
    );
  }
  await fs.ensureDir(path.dirname(target));
  await fs.writeFile(target, content, "utf-8");
  return target;
}
