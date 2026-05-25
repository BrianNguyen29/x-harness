import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import * as path from "node:path";
import { promises as fs } from "node:fs";

export interface GitSnapshot {
  statusMap: Map<string, string>;
  contentHashMap?: Map<string, string | null>;
  repoRoot: string;
}

const DEFAULT_HASH_CONCURRENCY = 16;
const FALLBACK_IGNORED_DIRS = new Set([".git", "node_modules", ".x-harness"]);

export function mutationGuardHashConcurrency(): number {
  const raw = process.env.X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY;
  if (!raw) return DEFAULT_HASH_CONCURRENCY;
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed < 1) return DEFAULT_HASH_CONCURRENCY;
  return Math.min(parsed, 64);
}

async function mapWithConcurrency<T, R>(
  items: T[],
  concurrency: number,
  mapper: (item: T) => Promise<R>
): Promise<R[]> {
  const results = new Array<R>(items.length);
  let nextIndex = 0;
  const workerCount = Math.min(Math.max(concurrency, 1), items.length);
  const workers = Array.from({ length: workerCount }, async () => {
    while (nextIndex < items.length) {
      const index = nextIndex;
      nextIndex += 1;
      results[index] = await mapper(items[index]);
    }
  });
  await Promise.all(workers);
  return results;
}

export async function isGitAvailable(): Promise<boolean> {
  return new Promise((resolve) => {
    execFile("git", ["--version"], (err) => {
      resolve(!err);
    });
  });
}

export async function getRepoRoot(cwd: string): Promise<string | null> {
  return new Promise((resolve) => {
    execFile(
      "git",
      ["rev-parse", "--show-toplevel"],
      { cwd },
      (err, stdout) => {
        if (err) {
          resolve(null);
          return;
        }
        resolve(stdout.trim());
      }
    );
  });
}

export async function snapshotGitStatus(
  repoRoot: string
): Promise<GitSnapshot> {
  return new Promise((resolve, reject) => {
    execFile(
      "git",
      ["status", "--porcelain=v1", "-z", "--untracked-files=all"],
      { cwd: repoRoot },
      async (err, stdout) => {
        if (err) {
          reject(err);
          return;
        }
        const statusMap = new Map<string, string>();
        const entries = stdout.split("\0").filter(Boolean);
        for (let i = 0; i < entries.length; i += 1) {
          const entry = entries[i] ?? "";
          if (entry.length < 4) continue;
          // Porcelain v1 -z format: "XY path\0"; rename/copy records include
          // a second NUL-delimited source path that we skip.
          const status = entry.slice(0, 2);
          const filePath = entry.slice(3);
          statusMap.set(filePath, status);
          if (status.includes("R") || status.includes("C")) i += 1;
        }
        try {
          const contentHashMap = await contentHashesForPaths(repoRoot, [
            ...statusMap.keys(),
          ]);
          resolve({ statusMap, contentHashMap, repoRoot });
        } catch (hashErr) {
          reject(hashErr);
        }
      }
    );
  });
}

async function contentHashesForPaths(
  repoRoot: string,
  filePaths: string[]
): Promise<Map<string, string | null>> {
  const entries = await mapWithConcurrency(
    filePaths,
    mutationGuardHashConcurrency(),
    async (filePath) =>
      [filePath, await workingTreeContentHash(repoRoot, filePath)] as const
  );
  return new Map(entries);
}

async function workingTreeContentHash(
  repoRoot: string,
  filePath: string
): Promise<string | null> {
  const resolved = path.resolve(repoRoot, filePath);
  const rootPrefix = repoRoot.endsWith(path.sep)
    ? repoRoot
    : `${repoRoot}${path.sep}`;
  if (resolved !== repoRoot && !resolved.startsWith(rootPrefix)) return null;

  try {
    const stat = await fs.lstat(resolved);
    if (stat.isSymbolicLink()) {
      const target = await fs.readlink(resolved);
      return createHash("sha256").update(`symlink:${target}`).digest("hex");
    }
    if (!stat.isFile()) return null;
    const data = await fs.readFile(resolved);
    return createHash("sha256").update(data).digest("hex");
  } catch {
    return null;
  }
}

export function isMutationGuardAllowlistedPath(filePath: string): boolean {
  // Normalize path separators
  const normalized = filePath.replace(/\\/g, "/");
  // Allow generated harness state under .x-harness/, including trace output
  // and the directory itself when git reports it as a single untracked entry.
  return (
    normalized === ".x-harness" ||
    normalized.startsWith(".x-harness/") ||
    normalized.includes("/.x-harness/") ||
    normalized.endsWith(".x-harness") ||
    normalized.endsWith(".x-harness/")
  );
}

export interface MutationDelta {
  path: string;
  beforeStatus: string | null;
  afterStatus: string | null;
  beforeHash?: string | null;
  afterHash?: string | null;
}

export function compareSnapshots(
  before: GitSnapshot,
  after: GitSnapshot
): MutationDelta[] {
  const deltas: MutationDelta[] = [];
  const allPaths = new Set([
    ...before.statusMap.keys(),
    ...after.statusMap.keys(),
  ]);
  for (const filePath of allPaths) {
    const beforeStatus = before.statusMap.get(filePath) ?? null;
    const afterStatus = after.statusMap.get(filePath) ?? null;
    const beforeHash = before.contentHashMap?.get(filePath) ?? null;
    const afterHash = after.contentHashMap?.get(filePath) ?? null;
    if (beforeStatus !== afterStatus || beforeHash !== afterHash) {
      deltas.push({
        path: filePath,
        beforeStatus,
        afterStatus,
        beforeHash,
        afterHash,
      });
    }
  }
  return deltas;
}

export function filterUnexpectedDeltas(
  deltas: MutationDelta[]
): MutationDelta[] {
  return deltas.filter((d) => !isMutationGuardAllowlistedPath(d.path));
}

function fallbackStatusPath(root: string, filePath: string): string {
  return path.relative(root, filePath).replace(/\\/g, "/");
}

async function collectFallbackSnapshotPaths(
  root: string,
  current = root
): Promise<string[]> {
  const entries = await fs.readdir(current, { withFileTypes: true });
  const paths: string[] = [];
  for (const entry of entries) {
    if (entry.isDirectory() && FALLBACK_IGNORED_DIRS.has(entry.name)) {
      continue;
    }
    const fullPath = path.join(current, entry.name);
    if (entry.isDirectory()) {
      paths.push(...(await collectFallbackSnapshotPaths(root, fullPath)));
    } else if (entry.isFile() || entry.isSymbolicLink()) {
      paths.push(fallbackStatusPath(root, fullPath));
    }
  }
  return paths.sort();
}

export async function snapshotDirectoryTree(
  root: string
): Promise<GitSnapshot> {
  const resolvedRoot = path.resolve(root);
  const paths = await collectFallbackSnapshotPaths(resolvedRoot);
  const statusMap = new Map(paths.map((filePath) => [filePath, "F "]));
  const contentHashMap = await contentHashesForPaths(resolvedRoot, paths);
  return { statusMap, contentHashMap, repoRoot: resolvedRoot };
}

export interface GuardResult {
  enabled: boolean;
  skippedReason?: string;
  deltas?: MutationDelta[];
  unexpectedDeltas?: MutationDelta[];
  violated: boolean;
}

export async function runMutationGuard(
  enabled: boolean,
  cwd: string
): Promise<{
  takeSnapshot: () => Promise<GitSnapshot>;
  evaluate: () => Promise<GuardResult>;
}> {
  if (!enabled) {
    return {
      takeSnapshot: async () => ({ statusMap: new Map(), repoRoot: cwd }),
      evaluate: async () => ({ enabled: false, violated: false }),
    };
  }

  let snapshotFn: () => Promise<GitSnapshot>;
  const gitAvailable = await isGitAvailable();
  if (gitAvailable) {
    const repoRoot = await getRepoRoot(cwd);
    if (repoRoot) {
      snapshotFn = () => snapshotGitStatus(repoRoot);
    } else {
      const fallbackRoot = path.resolve(cwd);
      snapshotFn = () => snapshotDirectoryTree(fallbackRoot);
    }
  } else {
    const fallbackRoot = path.resolve(cwd);
    snapshotFn = () => snapshotDirectoryTree(fallbackRoot);
  }

  let beforeSnapshot: GitSnapshot | undefined;
  let snapshotError: string | undefined;

  return {
    takeSnapshot: async () => {
      try {
        beforeSnapshot = await snapshotFn();
        snapshotError = undefined;
        return beforeSnapshot;
      } catch (err) {
        snapshotError = err instanceof Error ? err.message : String(err);
        return { statusMap: new Map(), repoRoot: path.resolve(cwd) };
      }
    },
    evaluate: async () => {
      if (snapshotError) {
        return {
          enabled: true,
          skippedReason: `snapshot failed: ${snapshotError}`,
          violated: false,
        };
      }
      if (!beforeSnapshot) {
        return {
          enabled: true,
          skippedReason: "before snapshot missing",
          violated: false,
        };
      }
      let afterSnapshot: GitSnapshot;
      try {
        afterSnapshot = await snapshotFn();
      } catch (err) {
        return {
          enabled: true,
          skippedReason: `snapshot failed: ${err instanceof Error ? err.message : String(err)}`,
          violated: false,
        };
      }
      const deltas = compareSnapshots(beforeSnapshot, afterSnapshot);
      const unexpectedDeltas = filterUnexpectedDeltas(deltas);
      return {
        enabled: true,
        deltas,
        unexpectedDeltas,
        violated: unexpectedDeltas.length > 0,
      };
    },
  };
}
