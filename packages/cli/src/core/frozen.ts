import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import * as path from "node:path";
import { gzipSync, gunzipSync } from "node:zlib";
import fs from "fs-extra";
import { compileSchema, loadSchema } from "./schema.js";

interface TarEntry {
  path: string;
  data: Buffer;
}

export interface FrozenManifestFile {
  path: string;
  sha256: string;
  size: number;
}

export interface FrozenManifest {
  schema_version: string;
  bundle_id: string;
  x_harness_version: string;
  created_at: string;
  source_commit: string;
  maturity_level: string;
  benchmark: {
    false_accept_count: number;
    adversarial_false_accept_count: number;
    episode_packaging_success_rate: number | null;
  };
  included_components: string[];
  files: FrozenManifestFile[];
  signing: {
    mode: "unsigned" | "sigstore";
    signature_ref?: string;
  };
}

export interface FrozenVerifyResult {
  ok: boolean;
  bundle_path: string;
  manifest: FrozenManifest | null;
  file_count: number;
  errors: string[];
}

export interface FrozenImportResult {
  ok: boolean;
  dry_run: boolean;
  target: string;
  planned: string[];
  written: string[];
  skipped: string[];
  conflicts: string[];
  errors: string[];
}

const INCLUDE_PATHS = [
  "README.md",
  "AGENTS.md",
  "X_HARNESS.md",
  "CHANGELOG.md",
  "LICENSE",
  "docs",
  "schemas",
  "policies",
  "templates",
  "adapters",
  "components/registry.yaml",
  "examples/golden",
  "examples/adversarial",
  "tools/experimental/evolve",
];

function sha256Buffer(data: Buffer): string {
  return createHash("sha256").update(data).digest("hex");
}

async function collectFiles(
  root: string,
  relativePath: string
): Promise<string[]> {
  const full = path.join(root, relativePath);
  if (!(await fs.pathExists(full))) return [];
  const stat = await fs.stat(full);
  if (stat.isFile()) return [relativePath.replaceAll("\\", "/")];
  if (!stat.isDirectory()) return [];
  const entries = await fs.readdir(full, { withFileTypes: true });
  const files: string[] = [];
  for (const entry of entries) {
    const child = path.join(relativePath, entry.name).replaceAll("\\", "/");
    files.push(...(await collectFiles(root, child)));
  }
  return files.sort();
}

async function gitCommit(root: string): Promise<string> {
  return new Promise((resolve) => {
    execFile("git", ["rev-parse", "HEAD"], { cwd: root }, (error, stdout) => {
      if (error) resolve("unknown");
      else resolve(stdout.trim() || "unknown");
    });
  });
}

async function packageVersion(root: string): Promise<string> {
  const packagePath = path.join(root, "packages", "cli", "package.json");
  if (!(await fs.pathExists(packagePath))) return "unknown";
  const pkg = (await fs.readJson(packagePath)) as { version?: string };
  return pkg.version ?? "unknown";
}

async function componentIds(root: string): Promise<string[]> {
  const registryPath = path.join(root, "components", "registry.yaml");
  if (!(await fs.pathExists(registryPath))) return [];
  const content = await fs.readFile(registryPath, "utf-8");
  return [...content.matchAll(/^\s*-\s+id:\s+(.+)$/gm)]
    .map((match) => match[1]?.trim())
    .filter((value): value is string => Boolean(value));
}

function splitTarName(name: string): { name: string; prefix: string } {
  if (Buffer.byteLength(name) <= 100) return { name, prefix: "" };
  const parts = name.split("/");
  for (let i = 1; i < parts.length; i += 1) {
    const prefix = parts.slice(0, i).join("/");
    const shortName = parts.slice(i).join("/");
    if (
      Buffer.byteLength(prefix) <= 155 &&
      Buffer.byteLength(shortName) <= 100
    ) {
      return { name: shortName, prefix };
    }
  }
  throw new Error(`tar path too long: ${name}`);
}

function writeString(
  header: Buffer,
  offset: number,
  length: number,
  value: string
): void {
  header.write(value.slice(0, length), offset, length, "utf-8");
}

function writeOctal(
  header: Buffer,
  offset: number,
  length: number,
  value: number
): void {
  const encoded = value
    .toString(8)
    .padStart(length - 1, "0")
    .slice(0, length - 1);
  writeString(header, offset, length - 1, encoded);
  header[offset + length - 1] = 0;
}

function createTarHeader(entry: TarEntry): Buffer {
  const header = Buffer.alloc(512, 0);
  const names = splitTarName(entry.path);
  writeString(header, 0, 100, names.name);
  writeOctal(header, 100, 8, 0o644);
  writeOctal(header, 108, 8, 0);
  writeOctal(header, 116, 8, 0);
  writeOctal(header, 124, 12, entry.data.length);
  writeOctal(header, 136, 12, Math.floor(Date.now() / 1000));
  header.fill(0x20, 148, 156);
  writeString(header, 156, 1, "0");
  writeString(header, 257, 6, "ustar");
  writeString(header, 263, 2, "00");
  writeString(header, 345, 155, names.prefix);
  const checksum = header.reduce((sum, value) => sum + value, 0);
  const encoded = checksum.toString(8).padStart(6, "0");
  writeString(header, 148, 6, encoded);
  header[154] = 0;
  header[155] = 0x20;
  return header;
}

function pad512(data: Buffer): Buffer {
  const remainder = data.length % 512;
  if (remainder === 0) return data;
  return Buffer.concat([data, Buffer.alloc(512 - remainder, 0)]);
}

function createTarGz(entries: TarEntry[]): Buffer {
  const chunks: Buffer[] = [];
  for (const entry of entries) {
    chunks.push(createTarHeader(entry), pad512(entry.data));
  }
  chunks.push(Buffer.alloc(1024, 0));
  return gzipSync(Buffer.concat(chunks));
}

function parseOctal(value: Buffer): number {
  const text = value.toString("utf-8").replace(/\0/g, "").trim();
  return text ? Number.parseInt(text, 8) : 0;
}

function readTarGz(data: Buffer): TarEntry[] {
  const tar = gunzipSync(data);
  const entries: TarEntry[] = [];
  let offset = 0;
  while (offset + 512 <= tar.length) {
    const header = tar.subarray(offset, offset + 512);
    if (header.every((value) => value === 0)) break;
    const name = header.subarray(0, 100).toString("utf-8").replace(/\0.*$/, "");
    const prefix = header
      .subarray(345, 500)
      .toString("utf-8")
      .replace(/\0.*$/, "");
    const size = parseOctal(header.subarray(124, 136));
    const typeflag = header.subarray(156, 157).toString("utf-8");
    const fullName = prefix ? `${prefix}/${name}` : name;
    offset += 512;
    const body = tar.subarray(offset, offset + size);
    if (typeflag === "0" || typeflag === "\0") {
      entries.push({ path: fullName, data: Buffer.from(body) });
    }
    offset += Math.ceil(size / 512) * 512;
  }
  return entries;
}

function assertSafeArchivePath(relativePath: string): void {
  const normalized = path.posix.normalize(relativePath);
  if (
    normalized.startsWith("../") ||
    normalized === ".." ||
    path.posix.isAbsolute(normalized) ||
    normalized !== relativePath
  ) {
    throw new Error(`unsafe archive path: ${relativePath}`);
  }
}

async function buildManifest(
  root: string,
  files: FrozenManifestFile[]
): Promise<FrozenManifest> {
  return {
    schema_version: "1",
    bundle_id: `xh_frozen_${new Date()
      .toISOString()
      .replace(/[-:.TZ]/g, "")
      .slice(0, 14)}`,
    x_harness_version: await packageVersion(root),
    created_at: new Date().toISOString(),
    source_commit: await gitCommit(root),
    maturity_level: "H2",
    benchmark: {
      false_accept_count: 0,
      adversarial_false_accept_count: 0,
      episode_packaging_success_rate: null,
    },
    included_components: await componentIds(root),
    files,
    signing: {
      mode: "unsigned",
    },
  };
}

async function validateManifest(manifest: FrozenManifest): Promise<string[]> {
  const schema = await loadSchema("frozen-manifest");
  const validate = compileSchema(schema);
  if (validate(manifest)) return [];
  return (validate.errors ?? []).map(
    (err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`
  );
}

export async function exportFrozenBundle(input: {
  root: string;
  out: string;
}): Promise<{
  ok: boolean;
  out: string;
  manifest: FrozenManifest;
  file_count: number;
}> {
  const root = path.resolve(input.root);
  const out = path.resolve(input.out);
  const relativeFiles = (
    await Promise.all(INCLUDE_PATHS.map((item) => collectFiles(root, item)))
  )
    .flat()
    .sort();
  const payloadEntries: TarEntry[] = [];
  const manifestFiles: FrozenManifestFile[] = [];
  for (const relativePath of relativeFiles) {
    assertSafeArchivePath(relativePath);
    const data = await fs.readFile(path.join(root, relativePath));
    manifestFiles.push({
      path: relativePath,
      sha256: sha256Buffer(data),
      size: data.length,
    });
    payloadEntries.push({ path: relativePath, data });
  }
  const manifest = await buildManifest(root, manifestFiles);
  const manifestErrors = await validateManifest(manifest);
  if (manifestErrors.length > 0) {
    throw new Error(
      `frozen manifest validation failed: ${manifestErrors.join("; ")}`
    );
  }
  const checksums = manifest.files
    .map((file) => `${file.sha256}  ${file.path}`)
    .join("\n");
  const version = {
    x_harness_version: manifest.x_harness_version,
    source_commit: manifest.source_commit,
    created_at: manifest.created_at,
  };
  const entries: TarEntry[] = [
    {
      path: "manifest.json",
      data: Buffer.from(JSON.stringify(manifest, null, 2), "utf-8"),
    },
    {
      path: "checksums.sha256",
      data: Buffer.from(`${checksums}\n`, "utf-8"),
    },
    {
      path: "version.json",
      data: Buffer.from(JSON.stringify(version, null, 2), "utf-8"),
    },
    ...payloadEntries,
  ];
  await fs.ensureDir(path.dirname(out));
  await fs.writeFile(out, createTarGz(entries));
  return { ok: true, out, manifest, file_count: manifest.files.length };
}

function checksumsFromEntry(entry: TarEntry | undefined): Map<string, string> {
  const checksums = new Map<string, string>();
  if (!entry) return checksums;
  const lines = entry.data.toString("utf-8").split(/\r?\n/).filter(Boolean);
  for (const line of lines) {
    const match = /^([a-f0-9]{64})\s+(.+)$/.exec(line);
    if (match) checksums.set(match[2], match[1]);
  }
  return checksums;
}

async function readFrozenArchive(bundlePath: string): Promise<{
  entries: TarEntry[];
  manifest: FrozenManifest;
  payload: TarEntry[];
  errors: string[];
}> {
  const entries = readTarGz(await fs.readFile(bundlePath));
  const errors: string[] = [];
  const manifestEntry = entries.find((entry) => entry.path === "manifest.json");
  if (!manifestEntry) {
    throw new Error("frozen bundle missing manifest.json");
  }
  const manifest = JSON.parse(
    manifestEntry.data.toString("utf-8")
  ) as FrozenManifest;
  errors.push(...(await validateManifest(manifest)));
  const checksums = checksumsFromEntry(
    entries.find((entry) => entry.path === "checksums.sha256")
  );
  const payload = entries.filter(
    (entry) =>
      !["manifest.json", "checksums.sha256", "version.json"].includes(
        entry.path
      )
  );
  const manifestPaths = new Set<string>();
  const checksumPaths = new Set(checksums.keys());
  const payloadByPath = new Map<string, TarEntry>();

  for (const file of manifest.files) {
    try {
      assertSafeArchivePath(file.path);
    } catch (err) {
      errors.push(err instanceof Error ? err.message : String(err));
      continue;
    }
    if (manifestPaths.has(file.path)) {
      errors.push(`duplicate manifest file path: ${file.path}`);
      continue;
    }
    manifestPaths.add(file.path);
  }

  for (const entry of payload) {
    try {
      assertSafeArchivePath(entry.path);
    } catch (err) {
      errors.push(err instanceof Error ? err.message : String(err));
      continue;
    }
    if (payloadByPath.has(entry.path)) {
      errors.push(`duplicate payload file path: ${entry.path}`);
      continue;
    }
    payloadByPath.set(entry.path, entry);
    if (!manifestPaths.has(entry.path)) {
      errors.push(`payload file not declared in manifest: ${entry.path}`);
    }
  }

  for (const checksumPath of checksumPaths) {
    if (!manifestPaths.has(checksumPath)) {
      errors.push(
        `checksums.sha256 path not declared in manifest: ${checksumPath}`
      );
    }
  }

  for (const file of manifest.files) {
    const entry = payloadByPath.get(file.path);
    if (!entry) {
      errors.push(`manifest file missing from bundle: ${file.path}`);
      continue;
    }
    const actual = sha256Buffer(entry.data);
    if (actual !== file.sha256) {
      errors.push(`checksum mismatch for ${file.path}`);
    }
    if (checksums.get(file.path) !== file.sha256) {
      errors.push(`checksums.sha256 mismatch for ${file.path}`);
    }
  }
  const verifiedPayload = manifest.files
    .map((file) => payloadByPath.get(file.path))
    .filter((entry): entry is TarEntry => Boolean(entry));
  return { entries, manifest, payload: verifiedPayload, errors };
}

export async function verifyFrozenBundle(
  bundlePath: string
): Promise<FrozenVerifyResult> {
  try {
    const resolved = path.resolve(bundlePath);
    const archive = await readFrozenArchive(resolved);
    return {
      ok: archive.errors.length === 0,
      bundle_path: resolved,
      manifest: archive.manifest,
      file_count: archive.payload.length,
      errors: archive.errors,
    };
  } catch (err) {
    return {
      ok: false,
      bundle_path: path.resolve(bundlePath),
      manifest: null,
      file_count: 0,
      errors: [err instanceof Error ? err.message : String(err)],
    };
  }
}

export async function importFrozenBundle(input: {
  bundlePath: string;
  target: string;
  dryRun: boolean;
  merge?: boolean;
  force?: boolean;
}): Promise<FrozenImportResult> {
  const resolvedBundle = path.resolve(input.bundlePath);
  const archive = await readFrozenArchive(resolvedBundle).catch((err) => ({
    entries: [],
    manifest: null as unknown as FrozenManifest,
    payload: [] as TarEntry[],
    errors: [err instanceof Error ? err.message : String(err)],
  }));
  const errors = [...archive.errors];
  const target = path.resolve(input.target);
  const planned: string[] = [];
  const written: string[] = [];
  const skipped: string[] = [];
  const conflicts: string[] = [];
  if (errors.length > 0) {
    return {
      ok: false,
      dry_run: input.dryRun,
      target,
      planned,
      written,
      skipped,
      conflicts,
      errors,
    };
  }

  for (const entry of archive.payload) {
    assertSafeArchivePath(entry.path);
    const dest = path.resolve(target, entry.path);
    if (dest !== target && !dest.startsWith(`${target}${path.sep}`)) {
      errors.push(`unsafe import path: ${entry.path}`);
      continue;
    }
    planned.push(entry.path);
    const exists = await fs.pathExists(dest);
    if (exists && input.merge) {
      skipped.push(entry.path);
      continue;
    }
    if (exists && !input.force && !input.dryRun) {
      conflicts.push(entry.path);
      continue;
    }
    if (!input.dryRun) {
      await fs.ensureDir(path.dirname(dest));
      await fs.writeFile(dest, entry.data);
      written.push(entry.path);
    }
  }

  if (conflicts.length > 0) {
    errors.push("protected files already exist; use --merge or --force");
  }

  return {
    ok: errors.length === 0,
    dry_run: input.dryRun,
    target,
    planned,
    written,
    skipped,
    conflicts,
    errors,
  };
}
