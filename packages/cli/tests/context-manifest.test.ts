import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
// import { execaNode } from "../src/test-helpers.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const cliDistPath = path.join(repoRoot, "packages", "cli", "dist", "index.js");

let workDir: string;
let originalCwd: string;

beforeEach(() => {
  originalCwd = process.cwd();
  workDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-context-manifest-test-"));
  process.chdir(workDir);
});

afterEach(() => {
  process.chdir(originalCwd);
  fs.rmSync(workDir, { recursive: true, force: true });
});

function writeFile(rel: string, content: string): string {
  const abs = path.join(workDir, rel);
  fs.mkdirSync(path.dirname(abs), { recursive: true });
  fs.writeFileSync(abs, content, "utf-8");
  return abs;
}

async function execaNodeWorkdir(
  args: string[]
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  const { execFile } = await import("node:child_process");
  return new Promise((resolve) => {
    execFile(
      process.execPath,
      [cliDistPath, ...args],
      { cwd: workDir },
      (error: Error | null, stdout: string, stderr: string) => {
        const code = (error as { code?: unknown } | null)?.code;
        const exitCode = typeof code === "number" ? code : error ? 1 : 0;
        resolve({
          stdout: stdout.trim(),
          stderr: stderr.trim(),
          exitCode,
        });
      }
    );
  });
}

describe("context manifest write", () => {
  it("writes a manifest for given files", async () => {
    writeFile("a.txt", "hello");
    writeFile("b.txt", "world");
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "write",
      "--files",
      "a.txt,b.txt",
    ]);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("wrote manifest");
    expect(result.stdout).toContain(".x-harness/context-manifest.yaml");
    const manifestPath = path.join(
      workDir,
      ".x-harness",
      "context-manifest.yaml"
    );
    expect(fs.existsSync(manifestPath)).toBe(true);
    const content = fs.readFileSync(manifestPath, "utf-8");
    expect(content).toContain('version: "1"');
    expect(content).toContain("a.txt");
    expect(content).toContain("b.txt");
  });

  it("outputs JSON when requested", async () => {
    writeFile("a.txt", "hello");
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "write",
      "--files",
      "a.txt",
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const parsed = JSON.parse(result.stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.out).toBe(".x-harness/context-manifest.yaml");
    expect(parsed.entries.length).toBe(1);
    expect(parsed.entries[0].path).toBe("a.txt");
    expect(typeof parsed.entries[0].sha256).toBe("string");
  });

  it("accepts --reason", async () => {
    writeFile("a.txt", "hello");
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "write",
      "--files",
      "a.txt",
      "--reason",
      "test-reason",
    ]);
    expect(result.exitCode).toBe(0);
    const manifestPath = path.join(
      workDir,
      ".x-harness",
      "context-manifest.yaml"
    );
    const content = fs.readFileSync(manifestPath, "utf-8");
    expect(content).toContain("test-reason");
  });

  it("accepts custom --out path", async () => {
    writeFile("a.txt", "hello");
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "write",
      "--files",
      "a.txt",
      "--out",
      "custom-manifest.yaml",
    ]);
    expect(result.exitCode).toBe(0);
    expect(fs.existsSync(path.join(workDir, "custom-manifest.yaml"))).toBe(
      true
    );
  });
});

describe("context manifest check", () => {
  it("passes for fresh manifest", async () => {
    writeFile("a.txt", "hello");
    await execaNodeWorkdir([
      "context",
      "manifest",
      "write",
      "--files",
      "a.txt",
      "--out",
      "manifest.yaml",
    ]);
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "check",
      "--manifest",
      "manifest.yaml",
    ]);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("all entries fresh");
  });

  it("fails for stale manifest", async () => {
    writeFile("a.txt", "hello");
    await execaNodeWorkdir([
      "context",
      "manifest",
      "write",
      "--files",
      "a.txt",
      "--out",
      "manifest.yaml",
    ]);
    fs.writeFileSync(path.join(workDir, "a.txt"), "modified", "utf-8");
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "check",
      "--manifest",
      "manifest.yaml",
    ]);
    expect(result.exitCode).toBe(1);
    expect(result.stdout).toContain("stale");
  });

  it("outputs JSON when requested", async () => {
    writeFile("a.txt", "hello");
    await execaNodeWorkdir([
      "context",
      "manifest",
      "write",
      "--files",
      "a.txt",
      "--out",
      "manifest.yaml",
    ]);
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "check",
      "--manifest",
      "manifest.yaml",
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const parsed = JSON.parse(result.stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.stale).toEqual([]);
  });

  it("reports error for missing manifest", async () => {
    const result = await execaNodeWorkdir([
      "context",
      "manifest",
      "check",
      "--manifest",
      "missing.yaml",
      "--json",
    ]);
    expect(result.exitCode).toBe(1);
    // Output may be JSON or plain error depending on flush timing
    const combined = result.stdout + result.stderr;
    expect(combined).toMatch(/error|cannot read manifest|ENOENT/i);
  });
});
