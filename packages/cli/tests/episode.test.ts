import { describe, expect, it, afterEach } from "vitest";
import fs from "fs-extra";
import * as os from "node:os";
import * as path from "node:path";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";
import { inspectEpisode, verifyEpisodeChain } from "../src/core/episode.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const tempDirs: string[] = [];

function makeTempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-episode-"));
  tempDirs.push(dir);
  return dir;
}

function sha256Ref(text: string): string {
  return `sha256:${createHash("sha256").update(text, "utf-8").digest("hex")}`;
}

afterEach(async () => {
  for (const dir of tempDirs.splice(0)) {
    await fs.remove(dir);
  }
});

describe("episode package", () => {
  it("creates an episode package from verify --episode --bundle", async () => {
    const tmp = makeTempDir();
    const episodesDir = path.join(tmp, "episodes");
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "success-light",
      "completion-card.yaml"
    );

    const { stdout, exitCode } = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--episode",
      "--bundle",
      "--episodes-dir",
      episodesDir,
      "--json",
    ]);

    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.episode).toBeDefined();
    expect(output.episode.task_id).toBe("TASK-GOLDEN-001");
    expect(output.episode.redacted_bundle).toContain(".redacted.tar.gz");
    expect(output.episode.raw_bundle).toContain(".raw.tar.gz");

    const episodeDir = path.join(
      repoRoot,
      output.episode.episode_dir as string
    );
    const validation = await inspectEpisode(episodeDir);
    expect(validation.ok).toBe(true);
    expect(validation.manifest?.admission_authority).toBe(false);
    expect(validation.manifest?.signing.mode).toBe("unsigned");

    const redactedBundle = path.join(
      repoRoot,
      output.episode.redacted_bundle as string
    );
    const bundleValidation = await inspectEpisode(redactedBundle);
    expect(bundleValidation.ok).toBe(true);
  });

  it("episode inspect command validates a directory and bundle", async () => {
    const tmp = makeTempDir();
    const episodesDir = path.join(tmp, "episodes");
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "success-light",
      "completion-card.yaml"
    );
    const verifyResult = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--episode",
      "--bundle",
      "--episodes-dir",
      episodesDir,
      "--json",
    ]);
    const output = JSON.parse(verifyResult.stdout);
    const episodeDir = path.join(
      repoRoot,
      output.episode.episode_dir as string
    );

    const { stdout, exitCode } = await execaNode([
      "episode",
      "inspect",
      episodeDir,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const inspect = JSON.parse(stdout);
    expect(inspect.ok).toBe(true);
    expect(inspect.task_id).toBe("TASK-GOLDEN-001");

    const bundlePath = path.join(
      repoRoot,
      output.episode.redacted_bundle as string
    );
    const bundleInspect = await execaNode([
      "episode",
      "inspect",
      bundlePath,
      "--json",
    ]);
    expect(bundleInspect.exitCode).toBe(0);
    expect(JSON.parse(bundleInspect.stdout).ok).toBe(true);
  });

  it("verifies an episode chain for repeated task runs", async () => {
    const tmp = makeTempDir();
    const episodesDir = path.join(tmp, "episodes");
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "success-light",
      "completion-card.yaml"
    );

    await execaNode([
      "verify",
      "--card",
      cardPath,
      "--episode",
      "--episodes-dir",
      episodesDir,
      "--json",
    ]);
    await execaNode([
      "verify",
      "--card",
      cardPath,
      "--episode",
      "--episodes-dir",
      episodesDir,
      "--json",
    ]);

    const result = await verifyEpisodeChain("TASK-GOLDEN-001", episodesDir);
    expect(result.ok).toBe(true);
    expect(result.episodes_checked).toBe(2);

    const { stdout, exitCode } = await execaNode([
      "episode",
      "verify-chain",
      "--task-id",
      "TASK-GOLDEN-001",
      "--episodes-dir",
      episodesDir,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    expect(JSON.parse(stdout).episodes_checked).toBe(2);
  });

  it("detects tampered episode files", async () => {
    const tmp = makeTempDir();
    const episodesDir = path.join(tmp, "episodes");
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "success-light",
      "completion-card.yaml"
    );

    const verifyResult = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--episode",
      "--episodes-dir",
      episodesDir,
      "--json",
    ]);
    const output = JSON.parse(verifyResult.stdout);
    const episodeDir = path.join(
      repoRoot,
      output.episode.episode_dir as string
    );
    await fs.writeFile(
      path.join(episodeDir, "verdict.json"),
      JSON.stringify({ tampered: true }),
      "utf-8"
    );

    const validation = await inspectEpisode(episodeDir);
    expect(validation.ok).toBe(false);
    expect(validation.errors.join("; ")).toContain("hash mismatch");
  });

  it("detects tampered episode files even when hashes.json is recomputed", async () => {
    const tmp = makeTempDir();
    const episodesDir = path.join(tmp, "episodes");
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "success-light",
      "completion-card.yaml"
    );

    const verifyResult = await execaNode([
      "verify",
      "--card",
      cardPath,
      "--episode",
      "--episodes-dir",
      episodesDir,
      "--json",
    ]);
    const output = JSON.parse(verifyResult.stdout);
    const episodeDir = path.join(
      repoRoot,
      output.episode.episode_dir as string
    );
    const verdictPath = path.join(episodeDir, "verdict.json");
    const tamperedVerdict = JSON.stringify({ tampered: true });
    await fs.writeFile(verdictPath, tamperedVerdict, "utf-8");

    const hashesPath = path.join(episodeDir, "hashes.json");
    const hashes = await fs.readJson(hashesPath);
    const stat = await fs.stat(verdictPath);
    hashes.files = hashes.files.map(
      (file: { path: string; sha256: string; size_bytes: number }) =>
        file.path === "verdict.json"
          ? {
              ...file,
              sha256: sha256Ref(tamperedVerdict),
              size_bytes: stat.size,
            }
          : file
    );
    await fs.writeJson(hashesPath, hashes, { spaces: 2 });

    const validation = await inspectEpisode(episodeDir);
    expect(validation.ok).toBe(false);
    expect(validation.errors.join("; ")).toContain("hashes_hash mismatch");
  });
});
