import * as path from "node:path";
import fs from "fs-extra";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";

export interface AgentProfile {
  schema_version: "1";
  agent_id: string;
  measured_on: string;
  observed_failure_modes: string[];
  required_extra_checks: string[];
  benchmark_metrics: Record<string, unknown>;
  advisory_only: true;
  admission_authority: false;
}

function safeAgentId(agentId: string): string {
  return agentId.replace(/[^A-Za-z0-9._-]/g, "_");
}

export function defaultAgentProfilePath(root: string, agentId: string): string {
  return path.join(
    path.resolve(root),
    ".x-harness",
    "agent-profiles",
    `${safeAgentId(agentId)}.json`
  );
}

function metricsFromBenchmark(input: unknown): Record<string, unknown> {
  if (!input || typeof input !== "object") return {};
  const metrics = (input as Record<string, unknown>).metrics;
  return metrics && typeof metrics === "object"
    ? (metrics as Record<string, unknown>)
    : {};
}

function collectFailureModes(report: unknown): string[] {
  const modes = new Set<string>();
  const metrics = metricsFromBenchmark(report);
  if (Number(metrics.false_accept_count ?? 0) > 0) {
    modes.add("false_accept_regression");
  }
  if (Number(metrics.adversarial_false_accept_count ?? 0) > 0) {
    modes.add("adversarial_false_accept");
  }
  if (Number(metrics.false_reject_count ?? 0) > 0) {
    modes.add("false_reject_regression");
  }
  const integration = (report as Record<string, unknown> | null)?.integration;
  const text = JSON.stringify(integration ?? {});
  if (/stale/i.test(text)) modes.add("stale_context_reference");
  if (/evidence/i.test(text)) modes.add("evidence_scope_mismatch");
  return [...modes].sort();
}

function extraChecks(modes: string[]): string[] {
  const checks = new Set<string>(["standard_verify_gate"]);
  if (modes.includes("false_accept_regression")) {
    checks.add("adversarial_replay_required");
  }
  if (modes.includes("adversarial_false_accept")) {
    checks.add("permission_and_mutation_replay_required");
  }
  if (modes.includes("false_reject_regression")) {
    checks.add("fixture_review_required");
  }
  if (modes.includes("stale_context_reference")) {
    checks.add("context_check_required");
  }
  if (modes.includes("evidence_scope_mismatch")) {
    checks.add("evidence_digest_required");
  }
  return [...checks].sort();
}

async function validateProfile(profile: AgentProfile): Promise<void> {
  const schema = await loadSchema("agent-profile");
  const validate = compileSchema(schema);
  if (!validate(profile)) {
    throw new Error(
      `agent profile validation failed: ${(validate.errors ?? [])
        .map((err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`)
        .join("; ")}`
    );
  }
}

export async function buildAgentProfile(input: {
  agentId: string;
  benchmarkReportPath?: string;
  measuredOn?: string;
}): Promise<AgentProfile> {
  const benchmark = input.benchmarkReportPath
    ? await readYamlOrJson(path.resolve(input.benchmarkReportPath))
    : {};
  const modes = collectFailureModes(benchmark);
  const profile: AgentProfile = {
    schema_version: "1",
    agent_id: input.agentId,
    measured_on: input.measuredOn ?? new Date().toISOString(),
    observed_failure_modes: modes,
    required_extra_checks: extraChecks(modes),
    benchmark_metrics: metricsFromBenchmark(benchmark),
    advisory_only: true,
    admission_authority: false,
  };
  await validateProfile(profile);
  return profile;
}

export async function writeAgentProfile(
  profile: AgentProfile,
  outPath: string
): Promise<string> {
  const resolved = path.resolve(outPath);
  await fs.ensureDir(path.dirname(resolved));
  await fs.writeJson(resolved, profile, { spaces: 2 });
  return resolved;
}

export async function readAgentProfile(
  filePath: string
): Promise<AgentProfile> {
  const profile = (await readYamlOrJson(
    path.resolve(filePath)
  )) as AgentProfile;
  await validateProfile(profile);
  return profile;
}
