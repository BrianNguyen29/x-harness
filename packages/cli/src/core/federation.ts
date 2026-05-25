import * as path from "node:path";
import fs from "fs-extra";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";
import {
  readEvidenceIndex,
  validateEvidenceIndex,
  type EvidenceIndexEntry,
} from "./evidence-corpus.js";
import { sha256String } from "./hash.js";
import { redactText } from "./redaction.js";

const ADMISSION_OUTCOMES = new Set([
  "success",
  "failed",
  "blocked",
  "skipped",
  "timeout",
  "error",
]);
const ACCEPTANCE_STATUSES = new Set(["accepted", "withheld"]);

export interface FederationPolicy {
  version: number;
  federation: {
    enabled: boolean;
    default_enabled: boolean;
    require_opt_in: boolean;
    require_redaction: boolean;
    tenant_boundary: "required";
    retention_days: number;
    data_sent: string[];
    data_never_sent: string[];
    import: {
      default_dry_run: boolean;
      affects_admission: false;
    };
  };
}

export interface FederationPattern {
  schema_version: "1";
  pattern_id: string;
  tenant_hash: string;
  source_hash: string;
  pattern_class: "failure" | "observation";
  signal: {
    predicate_hash: string | null;
    predicate_present: boolean;
    admission_outcome?: string | null;
    acceptance_status?: string | null;
    evidence_layer: EvidenceIndexEntry["layer"];
  };
  evidence_kind: EvidenceIndexEntry["kind"];
  component_hashes: string[];
  benchmark_metrics: BenchmarkMetrics | null;
  created_at: string;
  retention_expires_at: string;
  redaction: {
    mode: "anonymized-pattern";
    redacted_required: true;
    raw_content_included: false;
    secret_scan_replacements: number;
  };
  admission_authority: false;
}

interface BenchmarkMetrics {
  false_accept_count?: number;
  adversarial_false_accept_count?: number;
  false_reject_count?: number;
  runtime_ms?: number;
}

export interface FederationExportResult {
  ok: boolean;
  out_path: string;
  record_count: number;
  policy_enabled: boolean;
  opt_in: boolean;
  redacted: boolean;
  tenant_hash: string;
  source_hash: string;
  admission_authority: false;
}

export interface FederationImportResult {
  ok: boolean;
  dry_run: boolean;
  target: string;
  planned_count: number;
  written_count: number;
  errors: string[];
  admission_authority: false;
}

function stableStringify(value: unknown): string {
  if (value === null || typeof value !== "object") {
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) {
    return `[${value.map((item) => stableStringify(item)).join(",")}]`;
  }
  const record = value as Record<string, unknown>;
  return `{${Object.keys(record)
    .sort()
    .map((key) => `${JSON.stringify(key)}:${stableStringify(record[key])}`)
    .join(",")}}`;
}

function scopedHash(tenant: string, value: string): string {
  return sha256String(`${tenant}:${value}`);
}

function getStringMetadata(
  entry: EvidenceIndexEntry,
  key: string
): string | null {
  const value = entry.metadata?.[key];
  return typeof value === "string" ? value : null;
}

function getCanonicalSignalMetadata(
  entry: EvidenceIndexEntry,
  key: "admission_outcome" | "acceptance_status"
): string | null {
  const value = getStringMetadata(entry, key);
  if (value === null) return null;
  const allowed =
    key === "admission_outcome" ? ADMISSION_OUTCOMES : ACCEPTANCE_STATUSES;
  if (allowed.has(value)) return value;
  throw new Error(
    `invalid federation ${key} metadata for ${entry.evidence_id}: ${value}`
  );
}

function componentHints(entry: EvidenceIndexEntry): string[] {
  const componentIds = entry.metadata?.component_ids;
  if (
    Array.isArray(componentIds) &&
    componentIds.every((item) => typeof item === "string")
  ) {
    return componentIds as string[];
  }
  const firstSegment = entry.path.split("/")[0];
  return firstSegment ? [firstSegment] : [];
}

function isFailureSignal(input: {
  predicate: string | null;
  outcome: string | null;
  acceptance: string | null;
}): boolean {
  if (input.outcome && input.outcome !== "success") return true;
  if (input.acceptance === "withheld") return true;
  return Boolean(
    input.predicate &&
    /(blocked|failed|withheld|missing|error|timeout|false_accept)/i.test(
      input.predicate
    )
  );
}

function retentionExpiry(createdAt: string, days: number): string {
  const date = new Date(createdAt);
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString();
}

async function validatePatterns(patterns: FederationPattern[]): Promise<void> {
  const schema = await loadSchema("federation-pattern");
  const validate = compileSchema(schema);
  const errors: string[] = [];
  for (const pattern of patterns) {
    if (!validate(pattern)) {
      errors.push(
        ...(validate.errors ?? []).map(
          (err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`
        )
      );
    }
    if (pattern.redaction.secret_scan_replacements > 0) {
      errors.push(`secret-like value detected in ${pattern.pattern_id}`);
    }
  }
  if (errors.length > 0) {
    throw new Error(
      `federation pattern validation failed: ${errors.join("; ")}`
    );
  }
}

function toJsonl(patterns: FederationPattern[]): string {
  return patterns.map((pattern) => JSON.stringify(pattern)).join("\n") + "\n";
}

function extractMetrics(report: unknown): BenchmarkMetrics | null {
  if (!report || typeof report !== "object") return null;
  const metrics = (report as Record<string, unknown>).metrics;
  if (!metrics || typeof metrics !== "object") return null;
  const record = metrics as Record<string, unknown>;
  const output: BenchmarkMetrics = {};
  for (const key of [
    "false_accept_count",
    "adversarial_false_accept_count",
    "false_reject_count",
    "runtime_ms",
  ] as const) {
    if (typeof record[key] === "number") output[key] = record[key];
  }
  return Object.keys(output).length > 0 ? output : null;
}

async function loadBenchmarkMetrics(
  benchmarkReportPath?: string
): Promise<BenchmarkMetrics | null> {
  if (!benchmarkReportPath) return null;
  const report = await readYamlOrJson(path.resolve(benchmarkReportPath));
  return extractMetrics(report);
}

export async function loadFederationPolicy(
  root: string,
  policyPath?: string
): Promise<{ path: string; policy: FederationPolicy }> {
  const resolved = path.resolve(
    root,
    policyPath ?? path.join("policies", "federation.yaml")
  );
  const policy = (await readYamlOrJson(resolved)) as FederationPolicy;
  return { path: resolved, policy };
}

export async function buildFederationPatterns(input: {
  root: string;
  indexPath: string;
  tenant: string;
  source: string;
  benchmarkReportPath?: string;
  policyPath?: string;
  now?: string;
}): Promise<{
  patterns: FederationPattern[];
  policy: FederationPolicy;
  tenant_hash: string;
  source_hash: string;
}> {
  const root = path.resolve(input.root);
  const { policy } = await loadFederationPolicy(root, input.policyPath);
  const entries = await readEvidenceIndex(path.resolve(root, input.indexPath));
  const validation = await validateEvidenceIndex(entries);
  if (!validation.ok) {
    throw new Error(
      `evidence index validation failed: ${validation.errors.join("; ")}`
    );
  }

  const tenantHash = scopedHash(input.tenant, "tenant");
  const sourceHash = scopedHash(input.tenant, input.source);
  const createdAt = input.now ?? new Date().toISOString();
  const benchmarkMetrics = await loadBenchmarkMetrics(
    input.benchmarkReportPath
  );
  const candidates = entries.filter((entry) => {
    const outcome = getCanonicalSignalMetadata(entry, "admission_outcome");
    const acceptance = getCanonicalSignalMetadata(entry, "acceptance_status");
    return Boolean(entry.predicate || outcome || acceptance);
  });

  const patterns: FederationPattern[] = candidates.map((entry) => {
    const outcome = getCanonicalSignalMetadata(entry, "admission_outcome");
    const acceptance = getCanonicalSignalMetadata(entry, "acceptance_status");
    const predicate = entry.predicate ?? null;
    const basePattern: Omit<FederationPattern, "redaction"> = {
      schema_version: "1",
      pattern_id: scopedHash(
        input.tenant,
        stableStringify({
          evidence_id: entry.evidence_id,
          kind: entry.kind,
          predicate,
          outcome,
          acceptance,
        })
      ),
      tenant_hash: tenantHash,
      source_hash: sourceHash,
      pattern_class: isFailureSignal({ predicate, outcome, acceptance })
        ? "failure"
        : "observation",
      signal: {
        predicate_hash: predicate ? scopedHash(input.tenant, predicate) : null,
        predicate_present: Boolean(predicate),
        admission_outcome: outcome,
        acceptance_status: acceptance,
        evidence_layer: entry.layer,
      },
      evidence_kind: entry.kind,
      component_hashes: componentHints(entry)
        .map((hint) => scopedHash(input.tenant, hint))
        .sort(),
      benchmark_metrics: benchmarkMetrics,
      created_at: createdAt,
      retention_expires_at: retentionExpiry(
        createdAt,
        policy.federation.retention_days
      ),
      admission_authority: false,
    };
    const secretScan = redactText(stableStringify(basePattern));
    return {
      ...basePattern,
      redaction: {
        mode: "anonymized-pattern",
        redacted_required: true,
        raw_content_included: false,
        secret_scan_replacements: secretScan.replacements,
      },
    };
  });

  await validatePatterns(patterns);
  return { patterns, policy, tenant_hash: tenantHash, source_hash: sourceHash };
}

export async function exportFederationPatterns(input: {
  root: string;
  indexPath: string;
  outPath: string;
  tenant: string;
  source: string;
  optIn: boolean;
  redacted: boolean;
  benchmarkReportPath?: string;
  policyPath?: string;
}): Promise<FederationExportResult> {
  const root = path.resolve(input.root);
  const { policy } = await loadFederationPolicy(root, input.policyPath);
  if (policy.federation.require_opt_in && !input.optIn) {
    throw new Error("federation export requires explicit --opt-in");
  }
  if (policy.federation.require_redaction && !input.redacted) {
    throw new Error("federation export requires --redacted");
  }
  if (!input.tenant.trim()) {
    throw new Error("federation export requires a non-empty --tenant");
  }

  const { patterns, tenant_hash, source_hash } = await buildFederationPatterns({
    root,
    indexPath: input.indexPath,
    tenant: input.tenant,
    source: input.source,
    benchmarkReportPath: input.benchmarkReportPath,
    policyPath: input.policyPath,
  });
  const outPath = path.resolve(root, input.outPath);
  await fs.ensureDir(path.dirname(outPath));
  await fs.writeFile(outPath, toJsonl(patterns), "utf-8");
  return {
    ok: true,
    out_path: outPath,
    record_count: patterns.length,
    policy_enabled: policy.federation.enabled,
    opt_in: true,
    redacted: true,
    tenant_hash,
    source_hash,
    admission_authority: false,
  };
}

export async function readFederationPatterns(
  filePath: string
): Promise<FederationPattern[]> {
  const content = await fs.readFile(path.resolve(filePath), "utf-8");
  const trimmed = content.trim();
  if (!trimmed) return [];
  if (trimmed.startsWith("{")) {
    const parsed = JSON.parse(trimmed) as Record<string, unknown>;
    if (Array.isArray(parsed.patterns)) {
      return parsed.patterns as FederationPattern[];
    }
    return [parsed as unknown as FederationPattern];
  }
  return trimmed
    .split(/\r?\n/)
    .filter(Boolean)
    .map((line) => JSON.parse(line) as FederationPattern);
}

export async function validateFederationPatternFile(
  filePath: string
): Promise<{ ok: boolean; patterns: FederationPattern[]; errors: string[] }> {
  try {
    const patterns = await readFederationPatterns(filePath);
    const schema = await loadSchema("federation-pattern");
    const validate = compileSchema(schema);
    const errors: string[] = [];
    for (const pattern of patterns) {
      if (!validate(pattern)) {
        errors.push(
          ...(validate.errors ?? []).map(
            (err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`
          )
        );
      }
      const secretScan = redactText(stableStringify(pattern));
      if (secretScan.replacements > 0) {
        errors.push(
          `secret-like value detected in ${pattern.pattern_id ?? "record"}`
        );
      }
    }
    return { ok: errors.length === 0, patterns, errors };
  } catch (err) {
    return {
      ok: false,
      patterns: [],
      errors: [err instanceof Error ? err.message : String(err)],
    };
  }
}

export async function importFederationPatterns(input: {
  root: string;
  patternsPath: string;
  targetPath: string;
  dryRun: boolean;
  merge?: boolean;
  force?: boolean;
}): Promise<FederationImportResult> {
  const root = path.resolve(input.root);
  const target = path.resolve(root, input.targetPath);
  const rootPrefix = root.endsWith(path.sep) ? root : `${root}${path.sep}`;
  if (target !== root && !target.startsWith(rootPrefix)) {
    return {
      ok: false,
      dry_run: input.dryRun,
      target,
      planned_count: 0,
      written_count: 0,
      errors: ["federation import target must stay inside --root"],
      admission_authority: false,
    };
  }

  const validation = await validateFederationPatternFile(input.patternsPath);
  if (!validation.ok) {
    return {
      ok: false,
      dry_run: input.dryRun,
      target,
      planned_count: 0,
      written_count: 0,
      errors: validation.errors,
      admission_authority: false,
    };
  }

  if (input.dryRun) {
    return {
      ok: true,
      dry_run: true,
      target,
      planned_count: validation.patterns.length,
      written_count: 0,
      errors: [],
      admission_authority: false,
    };
  }

  if ((await fs.pathExists(target)) && !input.merge && !input.force) {
    return {
      ok: false,
      dry_run: false,
      target,
      planned_count: validation.patterns.length,
      written_count: 0,
      errors: ["target exists; use --merge or --force"],
      admission_authority: false,
    };
  }

  let patterns = validation.patterns;
  if (input.merge && (await fs.pathExists(target))) {
    const existing = await readFederationPatterns(target);
    const byId = new Map(
      existing.map((pattern) => [pattern.pattern_id, pattern])
    );
    for (const pattern of validation.patterns) {
      byId.set(pattern.pattern_id, pattern);
    }
    patterns = [...byId.values()].sort((a, b) =>
      a.pattern_id.localeCompare(b.pattern_id)
    );
  }

  await validatePatterns(patterns);
  await fs.ensureDir(path.dirname(target));
  await fs.writeFile(target, toJsonl(patterns), "utf-8");
  return {
    ok: true,
    dry_run: false,
    target,
    planned_count: validation.patterns.length,
    written_count: patterns.length,
    errors: [],
    admission_authority: false,
  };
}
