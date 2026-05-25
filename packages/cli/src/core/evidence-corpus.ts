import * as path from "node:path";
import fs from "fs-extra";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";
import { sha256File, sha256String } from "./hash.js";
import { redactText, type RedactionResult } from "./redaction.js";

export type EvidenceLayer = "raw" | "redacted" | "digest" | "index";
export type EvidenceKind =
  | "completion_card"
  | "command_evidence"
  | "episode_file"
  | "verification_artifact"
  | "trace_event"
  | "digest"
  | "other";

export interface EvidenceIndexEntry {
  schema_version: "1";
  task_id: string;
  evidence_id: string;
  layer: EvidenceLayer;
  kind: EvidenceKind;
  path: string;
  source_path?: string;
  sha256: string;
  size_bytes: number;
  redacted: boolean;
  redaction?: {
    mode: "none" | "secret-redaction";
    patterns: string[];
    replacements: number;
  };
  predicate?: string;
  command?: string;
  exit_code?: number | null;
  verifies?: string[];
  does_not_verify?: string[];
  created_at: string;
  admission_authority: false;
  metadata?: Record<string, unknown>;
}

export interface EvidenceIndexEnvelope {
  schema_version: "1";
  task_id: string;
  created_at: string;
  entry_count: number;
  index_hash: string;
  entries: EvidenceIndexEntry[];
}

export interface EvidenceIndexResult extends EvidenceIndexEnvelope {
  out_path: string | null;
  redacted_dir: string | null;
  warnings: string[];
}

export interface EvidenceDigest {
  schema_version: "1";
  task_id: string;
  generated_at: string;
  generated_from_index: true;
  index_hash: string;
  deterministic_summary: {
    evidence_entries: number;
    raw_entries: number;
    redacted_entries: number;
    command_evidence_entries: number;
    verification_artifact_entries: number;
    trace_event_entries: number;
    completion_card_entries: number;
    admission_outcome: string | null;
    acceptance_status: string | null;
    primary_failure: string | null;
    missing_evidence: string[];
    redaction_patterns: string[];
    raw_redacted_separable: boolean;
  };
  narrative_summary: {
    generated_by: "none";
    admission_authority: false;
    text: null;
  };
  admission_authority: false;
}

interface IndexOptions {
  root: string;
  episodeDir?: string;
  cardPath?: string;
  taskId?: string;
  outPath?: string;
  redact?: boolean;
  redactedDir?: string;
  now?: string;
}

const TEXT_EXTENSIONS = new Set([
  ".txt",
  ".md",
  ".json",
  ".jsonl",
  ".yaml",
  ".yml",
  ".log",
  ".out",
  ".err",
  ".stdout",
  ".stderr",
  ".env",
  ".ts",
  ".js",
  ".tsx",
  ".jsx",
]);

function normalizePath(input: string): string {
  return input.split(path.sep).join("/");
}

function relativeTo(root: string, filePath: string): string {
  return normalizePath(path.relative(root, filePath));
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

function evidenceId(parts: string[]): string {
  return sha256String(parts.join(":")).slice(0, 16);
}

async function isTextFile(filePath: string): Promise<boolean> {
  const ext = path.extname(filePath).toLowerCase();
  if (TEXT_EXTENSIONS.has(ext)) return true;
  const buffer = await fs.readFile(filePath);
  return !buffer.includes(0);
}

async function collectFiles(rootDir: string): Promise<string[]> {
  const files: string[] = [];
  const walk = async (dir: string) => {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    for (const entry of entries) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        if ([".git", "node_modules", "dist"].includes(entry.name)) continue;
        await walk(full);
      } else if (entry.isFile()) {
        files.push(full);
      }
    }
  };
  await walk(rootDir);
  return files.sort();
}

function inferKind(filePath: string): EvidenceKind {
  const name = path.basename(filePath).toLowerCase();
  if (name === "completion-card.yaml" || name === "completion-card.json") {
    return "completion_card";
  }
  if (name.endsWith(".jsonl") && name.includes("trace")) {
    return "trace_event";
  }
  if (
    name.includes("stdout") ||
    name.includes("stderr") ||
    name.includes("test") ||
    name.includes("verify")
  ) {
    return "verification_artifact";
  }
  return "episode_file";
}

function extractArray(input: unknown): string[] | undefined {
  if (!Array.isArray(input)) return undefined;
  return input.filter((item): item is string => typeof item === "string");
}

function commandEvidenceMetadata(item: unknown): {
  command?: string;
  exit_code?: number | null;
  verifies?: string[];
  does_not_verify?: string[];
  predicate?: string;
  metadata?: Record<string, unknown>;
} {
  if (!item || typeof item !== "object") return {};
  const record = item as Record<string, unknown>;
  const result: ReturnType<typeof commandEvidenceMetadata> = {};
  if (typeof record.command === "string") result.command = record.command;
  if (typeof record.exit_code === "number") result.exit_code = record.exit_code;
  if (record.exit_code === null) result.exit_code = null;
  const verifies = extractArray(record.verifies);
  if (verifies) result.verifies = verifies;
  const doesNotVerify = extractArray(record.does_not_verify);
  if (doesNotVerify) result.does_not_verify = doesNotVerify;
  if (typeof record.predicate === "string") result.predicate = record.predicate;
  if (typeof record.status === "string" && !result.predicate) {
    result.predicate = `command_status_${record.status}`;
  }
  result.metadata = { indexed_from: "completion_card" };
  return result;
}

async function loadCardTask(cardPath: string): Promise<{
  taskId?: string;
  metadata?: Record<string, unknown>;
  commandEvidenceEntries: unknown[];
  verificationArtifactEntries: unknown[];
}> {
  const card = (await readYamlOrJson(cardPath)) as Record<string, unknown>;
  const evidence = card.evidence as Record<string, unknown> | undefined;
  return {
    taskId: typeof card.task_id === "string" ? card.task_id : undefined,
    metadata: {
      tier: card.tier ?? null,
      verification_status:
        (card.verification as Record<string, unknown> | undefined)?.status ??
        null,
      admission_outcome:
        (card.admission as Record<string, unknown> | undefined)?.outcome ??
        null,
      acceptance_status: card.acceptance_status ?? null,
    },
    commandEvidenceEntries: Array.isArray(evidence?.command_evidence)
      ? evidence.command_evidence
      : [],
    verificationArtifactEntries: Array.isArray(evidence?.verification_artifacts)
      ? evidence.verification_artifacts
      : [],
  };
}

async function makeFileEntry(input: {
  root: string;
  taskId: string;
  filePath: string;
  layer: EvidenceLayer;
  kind: EvidenceKind;
  createdAt: string;
  sourcePath?: string;
  redacted?: boolean;
  redaction?: EvidenceIndexEntry["redaction"];
  metadata?: Record<string, unknown>;
}): Promise<EvidenceIndexEntry> {
  const rel = relativeTo(input.root, input.filePath);
  const hash = await sha256File(input.filePath);
  const stat = await fs.stat(input.filePath);
  return {
    schema_version: "1",
    task_id: input.taskId,
    evidence_id: evidenceId([input.taskId, input.layer, input.kind, rel]),
    layer: input.layer,
    kind: input.kind,
    path: rel,
    ...(input.sourcePath ? { source_path: input.sourcePath } : {}),
    sha256: hash ?? sha256String(""),
    size_bytes: stat.size,
    redacted: Boolean(input.redacted),
    ...(input.redaction ? { redaction: input.redaction } : {}),
    created_at: input.createdAt,
    admission_authority: false,
    ...(input.metadata ? { metadata: input.metadata } : {}),
  };
}

function makeVirtualEntry(input: {
  taskId: string;
  path: string;
  kind: EvidenceKind;
  createdAt: string;
  value: unknown;
  metadata?: Record<string, unknown>;
  command?: string;
  exit_code?: number | null;
  verifies?: string[];
  does_not_verify?: string[];
  predicate?: string;
}): EvidenceIndexEntry {
  const serialized = stableStringify(input.value);
  return {
    schema_version: "1",
    task_id: input.taskId,
    evidence_id: evidenceId([input.taskId, "raw", input.kind, input.path]),
    layer: "raw",
    kind: input.kind,
    path: input.path,
    sha256: sha256String(serialized),
    size_bytes: Buffer.byteLength(serialized, "utf-8"),
    redacted: false,
    ...(input.predicate ? { predicate: input.predicate } : {}),
    ...(input.command ? { command: input.command } : {}),
    ...(input.exit_code !== undefined ? { exit_code: input.exit_code } : {}),
    ...(input.verifies ? { verifies: input.verifies } : {}),
    ...(input.does_not_verify
      ? { does_not_verify: input.does_not_verify }
      : {}),
    created_at: input.createdAt,
    admission_authority: false,
    ...(input.metadata ? { metadata: input.metadata } : {}),
  };
}

function redactionSummary(
  result: RedactionResult
): EvidenceIndexEntry["redaction"] {
  return {
    mode: "secret-redaction",
    patterns: result.findings.map((finding) => finding.pattern).sort(),
    replacements: result.replacements,
  };
}

export async function createEvidenceIndex(
  options: IndexOptions
): Promise<EvidenceIndexResult> {
  const root = path.resolve(options.root);
  const createdAt = options.now ?? new Date().toISOString();
  const entries: EvidenceIndexEntry[] = [];
  const warnings: string[] = [];
  let taskId = options.taskId;

  if (options.cardPath) {
    const cardPath = path.resolve(root, options.cardPath);
    if (!(await fs.pathExists(cardPath))) {
      throw new Error(`completion card not found: ${cardPath}`);
    }
    const cardInfo = await loadCardTask(cardPath);
    taskId = taskId ?? cardInfo.taskId;
    if (!taskId)
      throw new Error("task id is required when card has no task_id");
    entries.push(
      await makeFileEntry({
        root,
        taskId,
        filePath: cardPath,
        layer: "raw",
        kind: "completion_card",
        createdAt,
        metadata: cardInfo.metadata,
      })
    );
    cardInfo.commandEvidenceEntries.forEach((item, index) => {
      const metadata = commandEvidenceMetadata(item);
      entries.push(
        makeVirtualEntry({
          taskId: taskId as string,
          path: `completion-card.yaml#/evidence/command_evidence/${index}`,
          kind: "command_evidence",
          createdAt,
          value: item,
          metadata: metadata.metadata,
          command: metadata.command,
          exit_code: metadata.exit_code,
          verifies: metadata.verifies,
          does_not_verify: metadata.does_not_verify,
          predicate: metadata.predicate,
        })
      );
    });
    cardInfo.verificationArtifactEntries.forEach((item, index) => {
      const metadata = commandEvidenceMetadata(item);
      entries.push(
        makeVirtualEntry({
          taskId: taskId as string,
          path: `completion-card.yaml#/evidence/verification_artifacts/${index}`,
          kind: "verification_artifact",
          createdAt,
          value: item,
          metadata: metadata.metadata,
          command: metadata.command,
          exit_code: metadata.exit_code,
          verifies: metadata.verifies,
          does_not_verify: metadata.does_not_verify,
          predicate: metadata.predicate,
        })
      );
    });
  }

  if (options.episodeDir) {
    const episodeDir = path.resolve(root, options.episodeDir);
    if (!(await fs.pathExists(episodeDir))) {
      throw new Error(`episode directory not found: ${episodeDir}`);
    }
    const cardPath = path.join(episodeDir, "completion-card.yaml");
    if (!taskId && (await fs.pathExists(cardPath))) {
      const cardInfo = await loadCardTask(cardPath);
      taskId = cardInfo.taskId;
    }
    if (!taskId) {
      const manifestPath = path.join(episodeDir, "manifest.json");
      if (await fs.pathExists(manifestPath)) {
        const manifest = (await fs.readJson(manifestPath)) as Record<
          string,
          unknown
        >;
        if (typeof manifest.task_id === "string") taskId = manifest.task_id;
      }
    }
    if (!taskId) throw new Error("--task-id is required for this episode");

    const files = await collectFiles(episodeDir);
    const redactedDir = options.redactedDir
      ? path.resolve(root, options.redactedDir)
      : path.resolve(root, "evidence", "redacted", taskId);
    for (const filePath of files) {
      const kind = inferKind(filePath);
      const rawEntry = await makeFileEntry({
        root,
        taskId,
        filePath,
        layer: "raw",
        kind,
        createdAt,
      });
      entries.push(rawEntry);

      if (!options.redact) continue;
      if (!(await isTextFile(filePath))) {
        warnings.push(
          `skipped binary redaction for ${relativeTo(root, filePath)}`
        );
        continue;
      }
      const text = await fs.readFile(filePath, "utf-8");
      const result = redactText(text);
      if (result.replacements === 0) continue;
      const relFromEpisode = path.relative(episodeDir, filePath);
      const redactedPath = path.join(redactedDir, relFromEpisode);
      await fs.ensureDir(path.dirname(redactedPath));
      await fs.writeFile(redactedPath, result.text, "utf-8");
      entries.push(
        await makeFileEntry({
          root,
          taskId,
          filePath: redactedPath,
          layer: "redacted",
          kind,
          createdAt,
          sourcePath: rawEntry.path,
          redacted: true,
          redaction: redactionSummary(result),
        })
      );
    }
  }

  if (!taskId) throw new Error("task id could not be inferred");
  const sortedEntries = entries.sort((a, b) =>
    `${a.task_id}:${a.layer}:${a.kind}:${a.path}`.localeCompare(
      `${b.task_id}:${b.layer}:${b.kind}:${b.path}`
    )
  );
  const indexHash = hashEvidenceEntries(sortedEntries);

  const outPath = options.outPath ? path.resolve(root, options.outPath) : null;
  if (outPath) {
    await fs.ensureDir(path.dirname(outPath));
    await fs.writeFile(outPath, toJsonl(sortedEntries), "utf-8");
  }

  return {
    schema_version: "1",
    task_id: taskId,
    created_at: createdAt,
    entry_count: sortedEntries.length,
    index_hash: indexHash,
    entries: sortedEntries,
    out_path: outPath ? relativeTo(root, outPath) : null,
    redacted_dir: options.redactedDir
      ? relativeTo(root, path.resolve(root, options.redactedDir))
      : options.redact
        ? normalizePath(path.join("evidence", "redacted", taskId))
        : null,
    warnings,
  };
}

export function toJsonl(entries: EvidenceIndexEntry[]): string {
  return entries.map((entry) => JSON.stringify(entry)).join("\n") + "\n";
}

export function hashEvidenceEntries(entries: EvidenceIndexEntry[]): string {
  return sha256String(
    entries.map((entry) => stableStringify(entry)).join("\n")
  );
}

export async function readEvidenceIndex(
  indexPath: string
): Promise<EvidenceIndexEntry[]> {
  const resolved = path.resolve(indexPath);
  if (!(await fs.pathExists(resolved))) {
    throw new Error(`evidence index not found: ${resolved}`);
  }
  const content = await fs.readFile(resolved, "utf-8");
  const trimmed = content.trim();
  if (!trimmed) return [];
  if (trimmed.startsWith("{")) {
    try {
      const parsed = JSON.parse(trimmed) as EvidenceIndexEnvelope;
      if (Array.isArray(parsed.entries)) return parsed.entries;
    } catch {
      // JSONL starts with "{" too; fall through to line-by-line parsing.
    }
  }
  return trimmed
    .split(/\r?\n/)
    .filter((line) => line.trim().length > 0)
    .map((line) => JSON.parse(line) as EvidenceIndexEntry);
}

export async function validateEvidenceIndex(
  entries: EvidenceIndexEntry[] | EvidenceIndexEnvelope
): Promise<{ ok: boolean; errors: string[] }> {
  const schema = await loadSchema("evidence-index");
  const validate = compileSchema(schema);
  const values = Array.isArray(entries) ? entries : [entries];
  const errors: string[] = [];
  for (const value of values) {
    if (!validate(value)) {
      errors.push(
        ...(validate.errors ?? []).map(
          (err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`
        )
      );
    }
  }
  return { ok: errors.length === 0, errors };
}

export function buildEvidenceDigest(input: {
  taskId: string;
  entries: EvidenceIndexEntry[];
  generatedAt?: string;
}): EvidenceDigest {
  const taskEntries = input.entries.filter(
    (entry) => entry.task_id === input.taskId
  );
  const redactionPatterns = new Set<string>();
  for (const entry of taskEntries) {
    for (const pattern of entry.redaction?.patterns ?? []) {
      redactionPatterns.add(pattern);
    }
  }

  const completionCard = taskEntries.find(
    (entry) => entry.kind === "completion_card" && entry.metadata
  );
  const metadata = completionCard?.metadata ?? {};
  const missingEvidence = taskEntries
    .filter((entry) => entry.metadata?.artifact_missing === true)
    .map((entry) => entry.path)
    .sort();

  return {
    schema_version: "1",
    task_id: input.taskId,
    generated_at: input.generatedAt ?? new Date().toISOString(),
    generated_from_index: true,
    index_hash: hashEvidenceEntries(taskEntries),
    deterministic_summary: {
      evidence_entries: taskEntries.length,
      raw_entries: taskEntries.filter((entry) => entry.layer === "raw").length,
      redacted_entries: taskEntries.filter(
        (entry) => entry.layer === "redacted"
      ).length,
      command_evidence_entries: taskEntries.filter(
        (entry) => entry.kind === "command_evidence"
      ).length,
      verification_artifact_entries: taskEntries.filter(
        (entry) => entry.kind === "verification_artifact"
      ).length,
      trace_event_entries: taskEntries.filter(
        (entry) => entry.kind === "trace_event"
      ).length,
      completion_card_entries: taskEntries.filter(
        (entry) => entry.kind === "completion_card"
      ).length,
      admission_outcome:
        typeof metadata.admission_outcome === "string"
          ? metadata.admission_outcome
          : null,
      acceptance_status:
        typeof metadata.acceptance_status === "string"
          ? metadata.acceptance_status
          : null,
      primary_failure: missingEvidence.length > 0 ? "Fverification" : null,
      missing_evidence: missingEvidence,
      redaction_patterns: [...redactionPatterns].sort(),
      raw_redacted_separable:
        taskEntries.some((entry) => entry.layer === "raw") &&
        taskEntries.some((entry) => entry.layer === "redacted"),
    },
    narrative_summary: {
      generated_by: "none",
      admission_authority: false,
      text: null,
    },
    admission_authority: false,
  };
}

export function renderEvidenceDigestMarkdown(digest: EvidenceDigest): string {
  const summary = digest.deterministic_summary;
  const lines = [
    "# x-harness Evidence Digest",
    "",
    `- task_id: ${digest.task_id}`,
    `- generated_from_index: ${digest.generated_from_index}`,
    `- index_hash: ${digest.index_hash}`,
    `- admission_authority: ${digest.admission_authority}`,
    "",
    "## Deterministic summary",
    `- evidence_entries: ${summary.evidence_entries}`,
    `- raw_entries: ${summary.raw_entries}`,
    `- redacted_entries: ${summary.redacted_entries}`,
    `- command_evidence_entries: ${summary.command_evidence_entries}`,
    `- verification_artifact_entries: ${summary.verification_artifact_entries}`,
    `- trace_event_entries: ${summary.trace_event_entries}`,
    `- completion_card_entries: ${summary.completion_card_entries}`,
    `- admission_outcome: ${summary.admission_outcome ?? "unknown"}`,
    `- acceptance_status: ${summary.acceptance_status ?? "unknown"}`,
    `- primary_failure: ${summary.primary_failure ?? "none"}`,
    `- raw_redacted_separable: ${summary.raw_redacted_separable}`,
    "",
    "## Redaction",
    summary.redaction_patterns.length > 0
      ? summary.redaction_patterns.map((pattern) => `- ${pattern}`).join("\n")
      : "No redaction patterns recorded.",
    "",
    "## Missing evidence",
    summary.missing_evidence.length > 0
      ? summary.missing_evidence.map((item) => `- ${item}`).join("\n")
      : "None.",
    "",
    "## Advisory boundary",
    "Digest output is replayed from indexed evidence. It is not admission evidence and cannot accept completion.",
  ];
  return lines.join("\n") + "\n";
}
