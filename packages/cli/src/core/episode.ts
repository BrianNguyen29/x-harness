import { execFile } from "node:child_process";
import * as path from "node:path";
import * as os from "node:os";
import fs from "fs-extra";
import { compileSchema, loadSchema } from "./schema.js";
import { sha256File, sha256String } from "./hash.js";
import {
  buildEvidenceDigest,
  createEvidenceIndex,
  readEvidenceIndex,
  renderEvidenceDigestMarkdown,
  validateEvidenceIndex,
} from "./evidence-corpus.js";
import { redactText } from "./redaction.js";
import { readTraceFromFile, verifyTraceChain, appendTrace } from "./trace.js";
import { createFailureAttribution, writeAttribution } from "./attribution.js";
import type { VerifyPipelineResult } from "./verify-pipeline.js";

export interface EpisodeManifest {
  schema_version: "1";
  episode_id: string;
  task_id: string;
  created_at: string;
  x_harness_version: string;
  previous_episode_id: string | null;
  git: {
    base_sha: string | null;
    head_sha: string | null;
    dirty_before_verify: boolean;
    dirty_after_verify: boolean;
  };
  policy_hashes: Record<string, string | null>;
  schema_hashes: Record<string, string | null>;
  verdict: {
    admission_outcome: string;
    acceptance_status: string;
    blocking_predicate: string | null;
  };
  mutation_guard: {
    enabled: boolean;
    violated: boolean;
    skipped_reason?: string;
    unexpected_delta_count: number;
  };
  signing: {
    mode: "unsigned";
    signature_ref: null;
  };
  bundle_refs: {
    raw: string | null;
    redacted: string | null;
  };
  admission_authority: false;
  hashes_hash: string;
  manifest_hash: string;
}

export interface EpisodeFileHash {
  path: string;
  sha256: string;
  size_bytes: number;
}

export interface EpisodeHashes {
  schema_version: "1";
  files: EpisodeFileHash[];
}

export interface EpisodeCreateOptions {
  root?: string;
  episodesDir?: string;
  bundle?: boolean;
}

export interface EpisodeCreateResult {
  episode_id: string;
  task_id: string;
  episode_dir: string;
  manifest_path: string;
  raw_bundle: string | null;
  redacted_bundle: string | null;
  manifest_hash: string;
}

export interface EpisodeValidationResult {
  ok: boolean;
  episode_id: string | null;
  task_id: string | null;
  errors: string[];
  warnings: string[];
  manifest?: EpisodeManifest;
  file_count: number;
}

export interface EpisodeChainResult {
  ok: boolean;
  task_id: string;
  episodes_checked: number;
  errors: string[];
  episode_ids: string[];
}

const POLICY_SNAPSHOTS = [
  ["admission", "policies/admission.yaml"],
  ["authority", "policies/authority.yaml"],
  ["permissions", "policies/permissions.yaml"],
  ["recovery", "policies/recovery.yaml"],
  ["intake", "policies/intake.yaml"],
];

const SCHEMA_SNAPSHOTS = [
  ["attribution", "schemas/attribution.schema.json"],
  ["completion_card", "schemas/completion-card.schema.json"],
  ["evidence_index", "schemas/evidence-index.schema.json"],
  ["episode_manifest", "schemas/episode-manifest.schema.json"],
];

const TEXT_EXTENSIONS = new Set([
  ".json",
  ".jsonl",
  ".yaml",
  ".yml",
  ".md",
  ".txt",
  ".log",
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

function manifestHash(
  manifest: Omit<EpisodeManifest, "manifest_hash">
): string {
  return `sha256:${sha256String(stableStringify(manifest))}`;
}

function sanitizeId(input: string): string {
  return input.replace(/[^A-Za-z0-9_.-]/g, "_").slice(0, 80) || "TASK_UNKNOWN";
}

function generateEpisodeId(taskId: string): string {
  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  return `ep_${stamp}_${sanitizeId(taskId)}`;
}

async function execGit(root: string, args: string[]): Promise<string | null> {
  return new Promise((resolve) => {
    execFile("git", args, { cwd: root }, (err, stdout) => {
      if (err) {
        resolve(null);
        return;
      }
      resolve(stdout.trim() || null);
    });
  });
}

async function getGitInfo(root: string): Promise<EpisodeManifest["git"]> {
  const head = await execGit(root, ["rev-parse", "HEAD"]);
  const base = await execGit(root, ["rev-parse", "HEAD~1"]);
  const status = await execGit(root, ["status", "--porcelain"]);
  const dirty = Boolean(status);
  return {
    base_sha: base,
    head_sha: head,
    dirty_before_verify: dirty,
    dirty_after_verify: dirty,
  };
}

async function readPackageVersion(root: string): Promise<string> {
  const packagePath = path.join(root, "packages", "cli", "package.json");
  if (!(await fs.pathExists(packagePath))) return "0.1.0";
  const pkg = (await fs.readJson(packagePath)) as Record<string, unknown>;
  return typeof pkg.version === "string" ? pkg.version : "0.1.0";
}

async function sha256Ref(filePath: string): Promise<string | null> {
  const hash = await sha256File(filePath);
  return hash ? `sha256:${hash}` : null;
}

async function copySnapshotFiles(
  root: string,
  episodeDir: string,
  snapshots: string[][],
  targetDir: string
): Promise<Record<string, string | null>> {
  const hashes: Record<string, string | null> = {};
  for (const [name, rel] of snapshots) {
    const source = path.join(root, rel);
    const target = path.join(episodeDir, targetDir, path.basename(rel));
    if (await fs.pathExists(source)) {
      await fs.ensureDir(path.dirname(target));
      await fs.copyFile(source, target);
      hashes[name] = await sha256Ref(source);
    } else {
      hashes[name] = null;
    }
  }
  return hashes;
}

async function collectFiles(dir: string): Promise<string[]> {
  const files: string[] = [];
  const walk = async (current: string) => {
    const entries = await fs.readdir(current, { withFileTypes: true });
    for (const entry of entries) {
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) {
        await walk(full);
      } else if (entry.isFile()) {
        files.push(full);
      }
    }
  };
  await walk(dir);
  return files.sort();
}

async function writeHashes(episodeDir: string): Promise<EpisodeHashes> {
  const files = await collectFiles(episodeDir);
  const hashed: EpisodeFileHash[] = [];
  for (const file of files) {
    const rel = relativeTo(episodeDir, file);
    if (rel === "hashes.json" || rel === "manifest.json") continue;
    const hash = await sha256File(file);
    const stat = await fs.stat(file);
    hashed.push({
      path: rel,
      sha256: `sha256:${hash ?? sha256String("")}`,
      size_bytes: stat.size,
    });
  }
  const hashes: EpisodeHashes = {
    schema_version: "1",
    files: hashed.sort((a, b) => a.path.localeCompare(b.path)),
  };
  await fs.writeJson(path.join(episodeDir, "hashes.json"), hashes, {
    spaces: 2,
  });
  return hashes;
}

function isSafeEpisodeRelativePath(filePath: string): boolean {
  if (!filePath || path.isAbsolute(filePath)) return false;
  const normalized = normalizePath(path.normalize(filePath));
  return (
    normalized === filePath &&
    normalized !== "." &&
    !normalized.startsWith("../") &&
    !normalized.includes("/../")
  );
}

async function writeEpisodeTrace(
  episodeDir: string,
  event: VerifyPipelineResult["event"]
): Promise<void> {
  const tempTraceDir = path.join(episodeDir, ".trace-tmp");
  await fs.ensureDir(tempTraceDir);
  await appendTrace(event, tempTraceDir);
  await fs.copyFile(
    path.join(tempTraceDir, "events.jsonl"),
    path.join(episodeDir, "trace.jsonl")
  );
  await fs.remove(tempTraceDir);
}

async function listEpisodeManifests(
  episodesDir: string,
  taskId?: string
): Promise<Array<{ dir: string; manifest: EpisodeManifest }>> {
  if (!(await fs.pathExists(episodesDir))) return [];
  const entries = await fs.readdir(episodesDir, { withFileTypes: true });
  const manifests: Array<{ dir: string; manifest: EpisodeManifest }> = [];
  for (const entry of entries) {
    if (!entry.isDirectory() || !entry.name.startsWith("ep_")) continue;
    const dir = path.join(episodesDir, entry.name);
    const manifestPath = path.join(dir, "manifest.json");
    if (!(await fs.pathExists(manifestPath))) continue;
    const manifest = (await fs.readJson(manifestPath)) as EpisodeManifest;
    if (!taskId || manifest.task_id === taskId) {
      manifests.push({ dir, manifest });
    }
  }
  return manifests.sort(
    (a, b) =>
      new Date(a.manifest.created_at).getTime() -
      new Date(b.manifest.created_at).getTime()
  );
}

export async function listEpisodeDirectories(
  episodesDir = ".x-harness/episodes",
  taskId?: string
): Promise<Array<{ dir: string; manifest: EpisodeManifest }>> {
  return listEpisodeManifests(path.resolve(episodesDir), taskId);
}

function mutationGuardSummary(
  result: VerifyPipelineResult
): EpisodeManifest["mutation_guard"] {
  const guard = result.mutationGuardResult;
  return {
    enabled: guard?.enabled ?? false,
    violated: guard?.violated ?? false,
    ...(guard?.skippedReason ? { skipped_reason: guard.skippedReason } : {}),
    unexpected_delta_count: guard?.unexpectedDeltas?.length ?? 0,
  };
}

async function isTextFile(filePath: string): Promise<boolean> {
  const ext = path.extname(filePath).toLowerCase();
  if (TEXT_EXTENSIONS.has(ext)) return true;
  const buffer = await fs.readFile(filePath);
  return !buffer.includes(0);
}

async function redactEpisodeCopy(
  sourceDir: string,
  targetDir: string
): Promise<void> {
  await fs.copy(sourceDir, targetDir);
  const files = await collectFiles(targetDir);
  for (const file of files) {
    if (!(await isTextFile(file))) continue;
    const rel = relativeTo(targetDir, file);
    if (rel === "hashes.json") continue;
    const text = await fs.readFile(file, "utf-8");
    const redacted = redactText(text);
    if (redacted.replacements > 0) {
      await fs.writeFile(file, redacted.text, "utf-8");
    }
  }
  await writeHashes(targetDir);
}

async function createTarball(
  sourceDir: string,
  outPath: string
): Promise<void> {
  await fs.ensureDir(path.dirname(outPath));
  await new Promise<void>((resolve, reject) => {
    execFile(
      "tar",
      [
        "-czf",
        outPath,
        "-C",
        path.dirname(sourceDir),
        path.basename(sourceDir),
      ],
      (err) => {
        if (err) reject(err);
        else resolve();
      }
    );
  });
}

async function extractTarball(bundlePath: string): Promise<string> {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "xh-episode-"));
  await new Promise<void>((resolve, reject) => {
    execFile("tar", ["-xzf", bundlePath, "-C", tempDir], (err) => {
      if (err) reject(err);
      else resolve();
    });
  });
  const files = await collectFiles(tempDir);
  const manifest = files.find(
    (file) => path.basename(file) === "manifest.json"
  );
  if (!manifest)
    throw new Error("episode bundle does not contain manifest.json");
  return path.dirname(manifest);
}

export async function withEpisodeDirectory<T>(
  inputPath: string,
  fn: (episodeDir: string) => Promise<T>
): Promise<T> {
  const resolved = path.resolve(inputPath);
  if (!(await fs.pathExists(resolved))) {
    throw new Error(`episode not found: ${resolved}`);
  }
  const stat = await fs.stat(resolved);
  if (stat.isDirectory()) {
    return fn(resolved);
  }
  if (resolved.endsWith(".tar.gz") || resolved.endsWith(".tgz")) {
    const extracted = await extractTarball(resolved);
    try {
      return await fn(extracted);
    } finally {
      await fs.remove(path.dirname(extracted));
    }
  }
  throw new Error("episode path must be a directory or .tar.gz bundle");
}

export async function createEpisodeFromVerifyResult(
  result: VerifyPipelineResult,
  options: EpisodeCreateOptions = {}
): Promise<EpisodeCreateResult> {
  const inputRoot = path.resolve(options.root ?? process.cwd());
  const gitRoot = await execGit(inputRoot, ["rev-parse", "--show-toplevel"]);
  const root = path.resolve(gitRoot ?? inputRoot);
  const episodesDir = path.resolve(
    root,
    options.episodesDir ?? ".x-harness/episodes"
  );
  await fs.ensureDir(episodesDir);

  const taskId = result.taskId;
  const episodeId = generateEpisodeId(taskId);
  const episodeDir = path.join(episodesDir, episodeId);
  await fs.ensureDir(episodeDir);

  const existing = await listEpisodeManifests(episodesDir, taskId);
  const previousEpisodeId =
    existing.length > 0
      ? existing[existing.length - 1].manifest.episode_id
      : null;

  if (result.cardPath && (await fs.pathExists(result.cardPath))) {
    await fs.copyFile(
      result.cardPath,
      path.join(episodeDir, "completion-card.yaml")
    );
  } else if (result.card) {
    await fs.writeJson(
      path.join(episodeDir, "completion-card.json"),
      result.card,
      {
        spaces: 2,
      }
    );
  }

  await fs.writeJson(
    path.join(episodeDir, "verdict.json"),
    {
      schema_version: "1",
      task_id: taskId,
      admission_outcome: result.finalOutcome,
      acceptance_status: result.finalAcceptance,
      blocking_predicate: result.finalBlockingPredicate,
      accepted: result.accepted,
      errors: result.errors,
      notes: result.notes,
      checks: result.checks,
    },
    { spaces: 2 }
  );

  await writeEpisodeTrace(episodeDir, result.event);
  await writeAttribution(
    episodeDir,
    createFailureAttribution({
      episodeId,
      taskId,
      createdAt: result.event.created_at,
      admissionOutcome: result.finalOutcome,
      acceptanceStatus: result.finalAcceptance,
      blockingPredicate: result.finalBlockingPredicate,
      errors: result.errors,
      notes: result.notes,
    })
  );

  const policyHashes = await copySnapshotFiles(
    root,
    episodeDir,
    POLICY_SNAPSHOTS,
    "policy-snapshot"
  );
  const schemaHashes = await copySnapshotFiles(
    root,
    episodeDir,
    SCHEMA_SNAPSHOTS,
    "schema-snapshot"
  );

  const evidenceIndexPath = path.join(episodeDir, "evidence-index.jsonl");
  if (result.cardPath) {
    await createEvidenceIndex({
      root,
      cardPath: result.cardPath,
      taskId,
      outPath: evidenceIndexPath,
    });
  } else {
    await fs.writeFile(evidenceIndexPath, "", "utf-8");
  }
  const evidenceEntries = await readEvidenceIndex(evidenceIndexPath);
  const digest = buildEvidenceDigest({ taskId, entries: evidenceEntries });
  await fs.writeFile(
    path.join(episodeDir, "digest.md"),
    renderEvidenceDigestMarkdown(digest),
    "utf-8"
  );
  await fs.writeJson(path.join(episodeDir, "digest.json"), digest, {
    spaces: 2,
  });

  await fs.writeJson(
    path.join(episodeDir, "mutation-guard.json"),
    {
      schema_version: "1",
      strict: result.strict,
      result: result.mutationGuardResult ?? {
        enabled: false,
        violated: false,
      },
    },
    { spaces: 2 }
  );
  await fs.writeJson(
    path.join(episodeDir, "git.json"),
    await getGitInfo(root),
    {
      spaces: 2,
    }
  );
  await fs.writeFile(path.join(episodeDir, "interventions.jsonl"), "", "utf-8");
  await fs.ensureDir(path.join(episodeDir, "signatures"));

  const bundleRefs = {
    raw: options.bundle ? `${episodeId}.raw.tar.gz` : null,
    redacted: options.bundle ? `${episodeId}.redacted.tar.gz` : null,
  };
  await writeHashes(episodeDir);
  const hashesHash =
    (await sha256Ref(path.join(episodeDir, "hashes.json"))) ??
    `sha256:${sha256String("")}`;

  const unsignedManifest: Omit<EpisodeManifest, "manifest_hash"> = {
    schema_version: "1",
    episode_id: episodeId,
    task_id: taskId,
    created_at: result.event.created_at,
    x_harness_version: await readPackageVersion(root),
    previous_episode_id: previousEpisodeId,
    git: await getGitInfo(root),
    policy_hashes: policyHashes,
    schema_hashes: schemaHashes,
    verdict: {
      admission_outcome: result.finalOutcome,
      acceptance_status: result.finalAcceptance,
      blocking_predicate: result.finalBlockingPredicate,
    },
    mutation_guard: mutationGuardSummary(result),
    signing: {
      mode: "unsigned",
      signature_ref: null,
    },
    bundle_refs: bundleRefs,
    admission_authority: false,
    hashes_hash: hashesHash,
  };
  const manifest: EpisodeManifest = {
    ...unsignedManifest,
    manifest_hash: manifestHash(unsignedManifest),
  };
  await fs.writeJson(path.join(episodeDir, "manifest.json"), manifest, {
    spaces: 2,
  });

  let rawBundle: string | null = null;
  let redactedBundle: string | null = null;
  if (options.bundle) {
    rawBundle = path.join(episodesDir, `${episodeId}.raw.tar.gz`);
    redactedBundle = path.join(episodesDir, `${episodeId}.redacted.tar.gz`);
    await createTarball(episodeDir, rawBundle);

    const redactedStage = path.join(
      root,
      ".x-harness",
      "tmp",
      "episodes",
      `${episodeId}.redacted`
    );
    await fs.remove(redactedStage);
    await redactEpisodeCopy(episodeDir, redactedStage);
    await createTarball(redactedStage, redactedBundle);
    await fs.remove(redactedStage);
  }

  return {
    episode_id: episodeId,
    task_id: taskId,
    episode_dir: relativeTo(root, episodeDir),
    manifest_path: relativeTo(root, path.join(episodeDir, "manifest.json")),
    raw_bundle: rawBundle ? relativeTo(root, rawBundle) : null,
    redacted_bundle: redactedBundle ? relativeTo(root, redactedBundle) : null,
    manifest_hash: manifest.manifest_hash,
  };
}

export async function validateEpisodeDirectory(
  episodeDir: string
): Promise<EpisodeValidationResult> {
  const errors: string[] = [];
  const warnings: string[] = [];
  const manifestPath = path.join(episodeDir, "manifest.json");
  if (!(await fs.pathExists(manifestPath))) {
    return {
      ok: false,
      episode_id: null,
      task_id: null,
      errors: ["manifest.json not found"],
      warnings,
      file_count: 0,
    };
  }

  const manifest = (await fs.readJson(manifestPath)) as EpisodeManifest;
  const schema = await loadSchema("episode-manifest");
  const validate = compileSchema(schema);
  if (!validate(manifest)) {
    errors.push(
      ...(validate.errors ?? []).map(
        (err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`
      )
    );
  }

  const { manifest_hash: _hash, ...withoutHash } = manifest;
  const expectedManifestHash = manifestHash(withoutHash);
  if (manifest.manifest_hash !== expectedManifestHash) {
    errors.push(
      `manifest_hash mismatch: expected ${expectedManifestHash}, got ${manifest.manifest_hash}`
    );
  }

  const hashesPath = path.join(episodeDir, "hashes.json");
  let fileCount = 0;
  if (await fs.pathExists(hashesPath)) {
    const hashes = (await fs.readJson(hashesPath)) as EpisodeHashes;
    const actualHashesHash = await sha256Ref(hashesPath);
    if (manifest.hashes_hash !== actualHashesHash) {
      errors.push(
        `hashes_hash mismatch: expected ${manifest.hashes_hash}, got ${actualHashesHash}`
      );
    }
    fileCount = hashes.files.length;
    const seen = new Set<string>();
    const declared = new Set<string>();
    for (const file of hashes.files) {
      if (!isSafeEpisodeRelativePath(file.path)) {
        errors.push(`unsafe hashed file path: ${file.path}`);
        continue;
      }
      if (seen.has(file.path)) {
        errors.push(`duplicate hashed file path: ${file.path}`);
        continue;
      }
      seen.add(file.path);
      declared.add(file.path);
      const full = path.join(episodeDir, file.path);
      if (!(await fs.pathExists(full))) {
        errors.push(`hashed file missing: ${file.path}`);
        continue;
      }
      const actual = await sha256Ref(full);
      if (actual !== file.sha256) {
        errors.push(
          `hash mismatch for ${file.path}: expected ${file.sha256}, got ${actual}`
        );
      }
    }
    const actualFiles = (await collectFiles(episodeDir))
      .map((file) => relativeTo(episodeDir, file))
      .filter((file) => file !== "hashes.json" && file !== "manifest.json");
    for (const file of actualFiles) {
      if (!declared.has(file)) {
        errors.push(`unhashed episode file: ${file}`);
      }
    }
  } else {
    errors.push("hashes.json not found");
  }

  const tracePath = path.join(episodeDir, "trace.jsonl");
  if (await fs.pathExists(tracePath)) {
    const traceResult = verifyTraceChain(await readTraceFromFile(tracePath));
    if (!traceResult.valid) {
      errors.push(
        `trace chain broken at index ${traceResult.firstBrokenIndex}`
      );
    }
  } else {
    errors.push("trace.jsonl not found");
  }

  const evidenceIndexPath = path.join(episodeDir, "evidence-index.jsonl");
  if (await fs.pathExists(evidenceIndexPath)) {
    const evidenceValidation = await validateEvidenceIndex(
      await readEvidenceIndex(evidenceIndexPath)
    );
    if (!evidenceValidation.ok) {
      errors.push(
        `evidence index invalid: ${evidenceValidation.errors.join("; ")}`
      );
    }
  } else {
    warnings.push("evidence-index.jsonl not found");
  }

  return {
    ok: errors.length === 0,
    episode_id: manifest.episode_id,
    task_id: manifest.task_id,
    errors,
    warnings,
    manifest,
    file_count: fileCount,
  };
}

export async function inspectEpisode(
  inputPath: string
): Promise<EpisodeValidationResult> {
  const resolved = path.resolve(inputPath);
  if (!(await fs.pathExists(resolved))) {
    throw new Error(`episode not found: ${resolved}`);
  }
  const stat = await fs.stat(resolved);
  if (stat.isDirectory()) {
    return validateEpisodeDirectory(resolved);
  }
  if (resolved.endsWith(".tar.gz") || resolved.endsWith(".tgz")) {
    return withEpisodeDirectory(resolved, validateEpisodeDirectory);
  }
  throw new Error("episode inspect expects a directory or .tar.gz bundle");
}

export async function verifyEpisodeChain(
  taskId: string,
  episodesDir = ".x-harness/episodes"
): Promise<EpisodeChainResult> {
  const resolved = path.resolve(episodesDir);
  const episodes = await listEpisodeManifests(resolved, taskId);
  const errors: string[] = [];
  const ids = episodes.map((episode) => episode.manifest.episode_id);
  const idSet = new Set(ids);

  for (const episode of episodes) {
    const validation = await validateEpisodeDirectory(episode.dir);
    if (!validation.ok) {
      errors.push(
        `${episode.manifest.episode_id}: ${validation.errors.join("; ")}`
      );
    }
    const previous = episode.manifest.previous_episode_id;
    if (previous && !idSet.has(previous)) {
      errors.push(
        `${episode.manifest.episode_id}: missing previous episode ${previous}`
      );
    }
  }

  for (let i = 1; i < episodes.length; i++) {
    const expectedPrevious = episodes[i - 1].manifest.episode_id;
    const actualPrevious = episodes[i].manifest.previous_episode_id;
    if (actualPrevious !== expectedPrevious) {
      errors.push(
        `${episodes[i].manifest.episode_id}: previous_episode_id expected ${expectedPrevious}, got ${actualPrevious}`
      );
    }
  }

  return {
    ok: errors.length === 0,
    task_id: taskId,
    episodes_checked: episodes.length,
    errors,
    episode_ids: ids,
  };
}
