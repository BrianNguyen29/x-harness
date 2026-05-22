import { execFile } from "node:child_process";

export interface GitSnapshot {
  statusMap: Map<string, string>;
  repoRoot: string;
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
      ["status", "--porcelain"],
      { cwd: repoRoot },
      (err, stdout) => {
        if (err) {
          reject(err);
          return;
        }
        const statusMap = new Map<string, string>();
        for (const line of stdout.trim().split("\n")) {
          const trimmed = line.trim();
          if (!trimmed) continue;
          // Porcelain format: XY path or XY path -> origPath
          const status = trimmed.slice(0, 2);
          const rest = trimmed.slice(3);
          const filePath = rest.split(" -> ")[0] || rest;
          statusMap.set(filePath, status);
        }
        resolve({ statusMap, repoRoot });
      }
    );
  });
}

function isAllowlisted(filePath: string): boolean {
  // Normalize path separators
  const normalized = filePath.replace(/\\/g, "/");
  // Allow .x-harness/traces/ writes anywhere in repo,
  // and allow the .x-harness/ directory itself when git reports it
  // as a single untracked directory entry.
  return (
    normalized.includes(".x-harness/traces/") ||
    normalized.endsWith(".x-harness") ||
    normalized.endsWith(".x-harness/")
  );
}

export interface MutationDelta {
  path: string;
  beforeStatus: string | null;
  afterStatus: string | null;
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
    if (beforeStatus !== afterStatus) {
      deltas.push({ path: filePath, beforeStatus, afterStatus });
    }
  }
  return deltas;
}

export function filterUnexpectedDeltas(
  deltas: MutationDelta[]
): MutationDelta[] {
  return deltas.filter((d) => !isAllowlisted(d.path));
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

  const gitAvailable = await isGitAvailable();
  if (!gitAvailable) {
    return {
      takeSnapshot: async () => ({ statusMap: new Map(), repoRoot: cwd }),
      evaluate: async () => ({
        enabled: true,
        skippedReason: "git not available",
        violated: false,
      }),
    };
  }

  const repoRoot = await getRepoRoot(cwd);
  if (!repoRoot) {
    return {
      takeSnapshot: async () => ({ statusMap: new Map(), repoRoot: cwd }),
      evaluate: async () => ({
        enabled: true,
        skippedReason: "not a git repository",
        violated: false,
      }),
    };
  }

  let beforeSnapshot: GitSnapshot | undefined;

  return {
    takeSnapshot: async () => {
      beforeSnapshot = await snapshotGitStatus(repoRoot);
      return beforeSnapshot;
    },
    evaluate: async () => {
      if (!beforeSnapshot) {
        return {
          enabled: true,
          skippedReason: "before snapshot missing",
          violated: false,
        };
      }
      const afterSnapshot = await snapshotGitStatus(repoRoot);
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
