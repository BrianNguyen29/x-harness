import { describe, expect, it, afterEach } from "vitest";
import fs from "fs-extra";
import * as os from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";
import { redactText } from "../src/core/redaction.js";
import {
  buildEvidenceDigest,
  readEvidenceIndex,
  validateEvidenceIndex,
} from "../src/core/evidence-corpus.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const tempDirs: string[] = [];

function makeTempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-evidence-"));
  tempDirs.push(dir);
  return dir;
}

afterEach(async () => {
  for (const dir of tempDirs.splice(0)) {
    await fs.remove(dir);
  }
});

describe("evidence corpus", () => {
  it("redacts known secret and token fixtures", () => {
    const secretText = [
      "Authorization: Bearer abcDEF1234567890abcDEF",
      "GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz",
      "NPM_TOKEN=npm_1234567890abcdefghijklmnopqrstuv",
      "api_key=sk_test_1234567890abcdef",
      "password=hunter2-secret",
      "postgres://user:pass@example.com:5432/db",
    ].join("\n");

    const result = redactText(secretText);
    expect(result.replacements).toBeGreaterThanOrEqual(6);
    expect(result.text).not.toContain("abcDEF1234567890abcDEF");
    expect(result.text).not.toContain(
      "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
    );
    expect(result.text).not.toContain("npm_1234567890abcdefghijklmnopqrstuv");
    expect(result.text).toContain("[REDACTED:bearer_token]");
    expect(result.text).toContain("[REDACTED:github_token]");
    expect(result.text).toContain("[REDACTED:npm_token]");
    expect(result.text).toContain("[REDACTED:api_key]");
    expect(result.text).toContain("[REDACTED:password_assignment]");
    expect(result.text).toContain("[REDACTED:connection_string]");
  });

  it("indexes an episode and writes a separable redacted layer", async () => {
    const tmp = makeTempDir();
    const episodeDir = path.join(tmp, "episode");
    const indexPath = path.join(tmp, "evidence", "index.jsonl");
    const redactedDir = path.join(tmp, "evidence", "redacted", "TASK-EV-001");
    await fs.ensureDir(episodeDir);
    await fs.writeFile(
      path.join(episodeDir, "command.stdout.txt"),
      "npm test passed\nAuthorization: Bearer abcDEF1234567890abcDEF\n",
      "utf-8"
    );

    const { stdout, exitCode } = await execaNode([
      "evidence",
      "index",
      "--root",
      repoRoot,
      "--episode",
      episodeDir,
      "--task-id",
      "TASK-EV-001",
      "--out",
      indexPath,
      "--redact",
      "--redacted-dir",
      redactedDir,
      "--json",
    ]);

    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.task_id).toBe("TASK-EV-001");
    expect(output.entry_count).toBe(2);
    expect(await fs.pathExists(indexPath)).toBe(true);
    const redactedText = await fs.readFile(
      path.join(redactedDir, "command.stdout.txt"),
      "utf-8"
    );
    expect(redactedText).not.toContain("abcDEF1234567890abcDEF");
    expect(redactedText).toContain("[REDACTED:bearer_token]");

    const entries = await readEvidenceIndex(indexPath);
    const validation = await validateEvidenceIndex(entries);
    expect(validation.ok).toBe(true);
    expect(entries.some((entry) => entry.layer === "raw")).toBe(true);
    expect(entries.some((entry) => entry.layer === "redacted")).toBe(true);
  });

  it("builds replayable digest data from an evidence index", async () => {
    const tmp = makeTempDir();
    const episodeDir = path.join(tmp, "episode");
    const indexPath = path.join(tmp, "evidence", "index.jsonl");
    const redactedDir = path.join(tmp, "evidence", "redacted", "TASK-EV-002");
    await fs.ensureDir(episodeDir);
    await fs.writeFile(
      path.join(episodeDir, "command.stdout.txt"),
      "Authorization: Bearer abcDEF1234567890abcDEF\n",
      "utf-8"
    );

    await execaNode([
      "evidence",
      "index",
      "--root",
      repoRoot,
      "--episode",
      episodeDir,
      "--task-id",
      "TASK-EV-002",
      "--out",
      indexPath,
      "--redact",
      "--redacted-dir",
      redactedDir,
    ]);

    const entries = await readEvidenceIndex(indexPath);
    const digest = buildEvidenceDigest({
      taskId: "TASK-EV-002",
      entries,
      generatedAt: "2026-05-24T00:00:00Z",
    });
    expect(digest.generated_from_index).toBe(true);
    expect(digest.admission_authority).toBe(false);
    expect(digest.deterministic_summary.raw_redacted_separable).toBe(true);
    expect(digest.deterministic_summary.redaction_patterns).toContain(
      "bearer_token"
    );

    const { stdout, exitCode } = await execaNode([
      "report",
      "--digest",
      "--task-id",
      "TASK-EV-002",
      "--index",
      indexPath,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const reportDigest = JSON.parse(stdout);
    expect(reportDigest.index_hash).toBe(digest.index_hash);
    expect(reportDigest.admission_authority).toBe(false);
  });

  it("indexes completion-card command evidence and grep matches predicates", async () => {
    const tmp = makeTempDir();
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "regression",
      "success-standard-scoped-evidence",
      "completion-card.yaml"
    );
    const indexPath = path.join(tmp, "card-index.jsonl");

    const { exitCode } = await execaNode([
      "evidence",
      "index",
      "--root",
      repoRoot,
      "--card",
      cardPath,
      "--out",
      indexPath,
    ]);
    expect(exitCode).toBe(0);

    const entries = await readEvidenceIndex(indexPath);
    expect(entries.some((entry) => entry.kind === "command_evidence")).toBe(
      true
    );
    expect(
      entries.some((entry) => entry.kind === "verification_artifact")
    ).toBe(true);

    const { stdout } = await execaNode([
      "evidence",
      "grep",
      "--index",
      indexPath,
      "--predicate",
      "command_status_passed",
      "--json",
    ]);
    const result = JSON.parse(stdout);
    expect(result.count).toBeGreaterThan(0);
  });
});
