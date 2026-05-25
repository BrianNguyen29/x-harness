import { describe, it, expect } from "vitest";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import {
  isGitAvailable,
  getRepoRoot,
  snapshotGitStatus,
  compareSnapshots,
  filterUnexpectedDeltas,
  runMutationGuard,
} from "../src/core/mutation-guard.js";

describe("mutation-guard module", () => {
  it("detects git availability", async () => {
    const available = await isGitAvailable();
    expect(available).toBe(true);
  });

  it("finds repo root for this project", async () => {
    const root = await getRepoRoot(process.cwd());
    expect(root).toBeTruthy();
    expect(fs.existsSync(path.join(root!, ".git"))).toBe(true);
  });

  it("returns null repo root for non-git directory", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "mg-test-"));
    try {
      const root = await getRepoRoot(tmpDir);
      expect(root).toBeNull();
    } finally {
      fs.rmSync(tmpDir, { recursive: true });
    }
  });

  it("snapshotGitStatus captures untracked and modified files", async () => {
    const repoRoot = await getRepoRoot(process.cwd());
    expect(repoRoot).toBeTruthy();
    const snapshot = await snapshotGitStatus(repoRoot!);
    expect(snapshot.statusMap).toBeInstanceOf(Map);
    expect(snapshot.repoRoot).toBe(repoRoot);
  });

  it("compareSnapshots finds no delta for identical snapshots", () => {
    const before = {
      statusMap: new Map([["a.txt", "M "]]),
      contentHashMap: new Map([["a.txt", "hash-a"]]),
      repoRoot: "/repo",
    };
    const after = {
      statusMap: new Map([["a.txt", "M "]]),
      contentHashMap: new Map([["a.txt", "hash-a"]]),
      repoRoot: "/repo",
    };
    const deltas = compareSnapshots(before, after);
    expect(deltas).toHaveLength(0);
  });

  it("compareSnapshots detects dirty content changes without status changes", () => {
    const before = {
      statusMap: new Map([["a.txt", " M"]]),
      contentHashMap: new Map([["a.txt", "before"]]),
      repoRoot: "/repo",
    };
    const after = {
      statusMap: new Map([["a.txt", " M"]]),
      contentHashMap: new Map([["a.txt", "after"]]),
      repoRoot: "/repo",
    };
    const deltas = compareSnapshots(before, after);
    expect(deltas).toHaveLength(1);
    expect(deltas[0].path).toBe("a.txt");
    expect(deltas[0].beforeHash).toBe("before");
    expect(deltas[0].afterHash).toBe("after");
  });

  it("compareSnapshots detects new untracked file", () => {
    const before = {
      statusMap: new Map(),
      repoRoot: "/repo",
    };
    const after = {
      statusMap: new Map([["b.txt", "??"]]),
      repoRoot: "/repo",
    };
    const deltas = compareSnapshots(before, after);
    expect(deltas).toHaveLength(1);
    expect(deltas[0].path).toBe("b.txt");
    expect(deltas[0].beforeStatus).toBeNull();
    expect(deltas[0].afterStatus).toBe("??");
  });

  it("compareSnapshots detects changed status", () => {
    const before = {
      statusMap: new Map([["c.txt", " M"]]),
      repoRoot: "/repo",
    };
    const after = {
      statusMap: new Map([["c.txt", "M "]]),
      repoRoot: "/repo",
    };
    const deltas = compareSnapshots(before, after);
    expect(deltas).toHaveLength(1);
    expect(deltas[0].path).toBe("c.txt");
    expect(deltas[0].beforeStatus).toBe(" M");
    expect(deltas[0].afterStatus).toBe("M ");
  });

  it("filterUnexpectedDeltas allowlists .x-harness/traces/ paths", () => {
    const deltas = [
      {
        path: "foo.txt",
        beforeStatus: null as string | null,
        afterStatus: "??",
      },
      {
        path: "packages/cli/.x-harness/traces/events.jsonl",
        beforeStatus: null,
        afterStatus: "??",
      },
      {
        path: ".x-harness/traces/other.jsonl",
        beforeStatus: null,
        afterStatus: "??",
      },
    ];
    const unexpected = filterUnexpectedDeltas(deltas);
    expect(unexpected).toHaveLength(1);
    expect(unexpected[0].path).toBe("foo.txt");
  });

  it("runMutationGuard disabled returns no violation", async () => {
    const guard = await runMutationGuard(false, process.cwd());
    const snap = await guard.takeSnapshot();
    expect(snap.statusMap.size).toBe(0);
    const result = await guard.evaluate();
    expect(result.enabled).toBe(false);
    expect(result.violated).toBe(false);
  });

  it("runMutationGuard detects unexpected mutations in temp repo", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "mg-guard-"));
    try {
      // Init git repo
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      // Create and commit a file
      fs.writeFileSync(path.join(tmpDir, "tracked.txt"), "hello");
      execFileSync("git", ["add", "tracked.txt"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const guard = await runMutationGuard(true, tmpDir);
      await guard.takeSnapshot();

      // Create unexpected file
      fs.writeFileSync(path.join(tmpDir, "unexpected.txt"), "surprise");

      const result = await guard.evaluate();
      expect(result.enabled).toBe(true);
      expect(result.violated).toBe(true);
      expect(result.unexpectedDeltas).toHaveLength(1);
      expect(result.unexpectedDeltas![0].path).toBe("unexpected.txt");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("runMutationGuard ignores allowlisted trace writes", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "mg-allow-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      fs.writeFileSync(path.join(tmpDir, "tracked.txt"), "hello");
      execFileSync("git", ["add", "tracked.txt"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      const guard = await runMutationGuard(true, tmpDir);
      await guard.takeSnapshot();

      // Create allowlisted trace file
      fs.mkdirSync(path.join(tmpDir, ".x-harness", "traces"), {
        recursive: true,
      });
      fs.writeFileSync(
        path.join(tmpDir, ".x-harness", "traces", "events.jsonl"),
        "{}\n"
      );

      const result = await guard.evaluate();
      expect(result.violated).toBe(false);
      expect(result.deltas).toHaveLength(1);
      expect(result.deltas![0].path).toMatch(/\.x-harness/);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("runMutationGuard handles pre-existing dirty file without delta", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "mg-dirty-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      fs.writeFileSync(path.join(tmpDir, "tracked.txt"), "hello");
      execFileSync("git", ["add", "tracked.txt"], { cwd: tmpDir });
      execFileSync("git", ["commit", "-m", "init"], { cwd: tmpDir });

      // Create dirty file before snapshot
      fs.writeFileSync(path.join(tmpDir, "dirty.txt"), "dirty");

      const guard = await runMutationGuard(true, tmpDir);
      await guard.takeSnapshot();

      // No new changes
      const result = await guard.evaluate();
      expect(result.violated).toBe(false);
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("runMutationGuard detects edits to pre-existing dirty files", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "mg-dirty-edit-"));
    try {
      const { execFileSync } = await import("node:child_process");
      execFileSync("git", ["init"], { cwd: tmpDir });
      execFileSync("git", ["config", "user.email", "test@test.com"], {
        cwd: tmpDir,
      });
      execFileSync("git", ["config", "user.name", "Test"], { cwd: tmpDir });

      const dirtyPath = path.join(tmpDir, "dirty.txt");
      fs.writeFileSync(dirtyPath, "dirty-before");

      const guard = await runMutationGuard(true, tmpDir);
      await guard.takeSnapshot();

      fs.writeFileSync(dirtyPath, "dirty-after");

      const result = await guard.evaluate();
      expect(result.violated).toBe(true);
      expect(result.unexpectedDeltas?.[0].path).toBe("dirty.txt");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});
