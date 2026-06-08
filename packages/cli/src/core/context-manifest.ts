import * as crypto from "node:crypto";
import * as fs from "node:fs";
import * as path from "node:path";
import YAML from "yaml";

export interface ManifestEntry {
  path: string;
  sha256: string;
  read_at?: string;
  reason?: string;
}

export interface Manifest {
  version: string;
  entries: ManifestEntry[];
}

export function generateManifest(
  filePaths: string[],
  baseDir: string = process.cwd(),
  reason: string = ""
): Manifest {
  const resolvedBase = path.resolve(baseDir);
  const now = new Date().toISOString();

  const entries: ManifestEntry[] = [];
  for (const fp of filePaths) {
    const trimmed = fp.trim();
    if (trimmed === "") continue;
    const absPath = path.resolve(trimmed);
    const data = fs.readFileSync(absPath);
    const hash = crypto.createHash("sha256").update(data).digest("hex");
    let relPath = path.relative(resolvedBase, absPath);
    // Normalize to forward slashes for cross-platform stability
    relPath = relPath.split(path.sep).join("/");
    entries.push({
      path: relPath,
      sha256: hash,
      read_at: now,
      reason,
    });
  }

  return {
    version: "1",
    entries,
  };
}

export function checkManifest(
  manifest: Manifest,
  baseDir: string = process.cwd()
): string[] {
  const resolvedBase = path.resolve(baseDir);
  const stale: string[] = [];

  for (const entry of manifest.entries) {
    // Normalize path separator for the current OS
    const entryPath = entry.path.split("/").join(path.sep);
    const resolved = path.join(resolvedBase, entryPath);

    if (!fs.existsSync(resolved)) {
      stale.push(entry.path);
      continue;
    }

    const data = fs.readFileSync(resolved);
    const hash = crypto.createHash("sha256").update(data).digest("hex");
    if (hash !== entry.sha256) {
      stale.push(entry.path);
    }
  }

  return stale;
}

export function writeManifest(manifest: Manifest, outPath: string): void {
  const data = YAML.stringify(manifest);
  fs.mkdirSync(path.dirname(outPath), { recursive: true });
  fs.writeFileSync(outPath, data, "utf-8");
}

export function readManifest(manifestPath: string): Manifest {
  const data = fs.readFileSync(manifestPath, "utf-8");
  const parsed = YAML.parse(data);
  if (typeof parsed !== "object" || parsed == null) {
    throw new Error("invalid manifest: not an object");
  }
  return parsed as Manifest;
}

export function validateManifest(manifest: Manifest): void {
  if (!manifest.version) {
    throw new Error("manifest version is required");
  }
  if (manifest.version !== "1") {
    throw new Error(`unsupported manifest version: ${manifest.version}`);
  }
  const seen = new Set<string>();
  for (let i = 0; i < manifest.entries.length; i++) {
    const entry = manifest.entries[i];
    if (!entry.path || entry.path.trim() === "") {
      throw new Error(`entry[${i}]: path is required`);
    }
    if (!entry.sha256 || entry.sha256.trim() === "") {
      throw new Error(`entry[${i}]: sha256 is required`);
    }
    if (seen.has(entry.path)) {
      throw new Error(`duplicate path in manifest: ${entry.path}`);
    }
    seen.add(entry.path);
  }
}
