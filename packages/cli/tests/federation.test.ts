import { afterEach, describe, expect, it } from "vitest";
import fs from "fs-extra";
import * as os from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";
import {
  toJsonl,
  type EvidenceIndexEntry,
} from "../src/core/evidence-corpus.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const tempDirs: string[] = [];

function makeTempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-federation-"));
  tempDirs.push(dir);
  return dir;
}

async function writeEvidenceIndex(root: string): Promise<string> {
  const indexPath = path.join(root, "evidence", "index.jsonl");
  const entry: EvidenceIndexEntry = {
    schema_version: "1",
    task_id: "TASK-FED-001",
    evidence_id: "evidence-fed-001",
    layer: "raw",
    kind: "completion_card",
    path: "completion-card.yaml",
    sha256: "a".repeat(64),
    size_bytes: 123,
    redacted: false,
    predicate: "missing_evidence",
    created_at: "2026-05-24T00:00:00.000Z",
    admission_authority: false,
    metadata: {
      admission_outcome: "failed",
      acceptance_status: "withheld",
      component_ids: ["admission_policy"],
    },
  };
  await fs.ensureDir(path.dirname(indexPath));
  await fs.writeFile(indexPath, toJsonl([entry]), "utf-8");
  return indexPath;
}

async function writeInvalidSignalEvidenceIndex(root: string): Promise<string> {
  const indexPath = path.join(root, "evidence", "invalid-index.jsonl");
  const entry: EvidenceIndexEntry = {
    schema_version: "1",
    task_id: "TASK-FED-INVALID",
    evidence_id: "evidence-fed-invalid",
    layer: "raw",
    kind: "completion_card",
    path: "completion-card.yaml",
    sha256: "a".repeat(64),
    size_bytes: 123,
    redacted: false,
    predicate: "bad_signal",
    created_at: "2026-05-24T00:00:00.000Z",
    admission_authority: false,
    metadata: {
      admission_outcome: "approved",
      acceptance_status: "accepted",
    },
  };
  await fs.ensureDir(path.dirname(indexPath));
  await fs.writeFile(indexPath, toJsonl([entry]), "utf-8");
  return indexPath;
}

afterEach(async () => {
  for (const dir of tempDirs.splice(0)) {
    await fs.remove(dir);
  }
});

describe("federation command", () => {
  it("requires explicit opt-in for export", async () => {
    const tmp = makeTempDir();
    const index = await writeEvidenceIndex(tmp);
    const { stderr, exitCode } = await execaNode([
      "federation",
      "export-patterns",
      "--root",
      tmp,
      "--policy",
      path.join(repoRoot, "policies", "federation.yaml"),
      "--index",
      index,
      "--out",
      path.join(tmp, "patterns.jsonl"),
      "--tenant",
      "tenant-a",
      "--redacted",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("requires explicit --opt-in");
  });

  it("exports anonymized patterns without raw task, predicate, or component labels", async () => {
    const tmp = makeTempDir();
    const index = await writeEvidenceIndex(tmp);
    const out = path.join(tmp, "patterns.jsonl");

    const { stdout, exitCode } = await execaNode([
      "federation",
      "export-patterns",
      "--root",
      tmp,
      "--policy",
      path.join(repoRoot, "policies", "federation.yaml"),
      "--index",
      index,
      "--out",
      out,
      "--tenant",
      "tenant-a",
      "--opt-in",
      "--redacted",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const summary = JSON.parse(stdout);
    expect(summary.ok).toBe(true);
    expect(summary.record_count).toBe(1);
    expect(summary.policy_enabled).toBe(false);
    expect(summary.admission_authority).toBe(false);

    const exported = await fs.readFile(out, "utf-8");
    expect(exported).not.toContain("TASK-FED-001");
    expect(exported).not.toContain("missing_evidence");
    expect(exported).not.toContain("admission_policy");
    const pattern = JSON.parse(exported.trim());
    expect(pattern.pattern_class).toBe("failure");
    expect(pattern.signal.predicate_present).toBe(true);
    expect(pattern.redaction.raw_content_included).toBe(false);
    expect(pattern.admission_authority).toBe(false);
  });

  it("rejects non-canonical admission signal metadata on export", async () => {
    const tmp = makeTempDir();
    const index = await writeInvalidSignalEvidenceIndex(tmp);
    const out = path.join(tmp, "patterns.jsonl");

    const { stderr, exitCode } = await execaNode([
      "federation",
      "export-patterns",
      "--root",
      tmp,
      "--policy",
      path.join(repoRoot, "policies", "federation.yaml"),
      "--index",
      index,
      "--out",
      out,
      "--tenant",
      "tenant-a",
      "--opt-in",
      "--redacted",
      "--json",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("invalid federation admission_outcome metadata");
  });

  it("defaults import to dry-run and writes nothing", async () => {
    const tmp = makeTempDir();
    const index = await writeEvidenceIndex(tmp);
    const out = path.join(tmp, "patterns.jsonl");
    await execaNode([
      "federation",
      "export-patterns",
      "--root",
      tmp,
      "--policy",
      path.join(repoRoot, "policies", "federation.yaml"),
      "--index",
      index,
      "--out",
      out,
      "--tenant",
      "tenant-a",
      "--opt-in",
      "--redacted",
    ]);

    const { stdout, exitCode } = await execaNode([
      "federation",
      "import-patterns",
      out,
      "--root",
      tmp,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const result = JSON.parse(stdout);
    expect(result.ok).toBe(true);
    expect(result.dry_run).toBe(true);
    expect(result.planned_count).toBe(1);
    expect(result.written_count).toBe(0);
    expect(
      await fs.pathExists(
        path.join(tmp, ".x-harness", "federation", "imported-patterns.jsonl")
      )
    ).toBe(false);
  });

  it("imports with merge and validates the stored patterns", async () => {
    const tmp = makeTempDir();
    const index = await writeEvidenceIndex(tmp);
    const out = path.join(tmp, "patterns.jsonl");
    await execaNode([
      "federation",
      "export-patterns",
      "--root",
      tmp,
      "--policy",
      path.join(repoRoot, "policies", "federation.yaml"),
      "--index",
      index,
      "--out",
      out,
      "--tenant",
      "tenant-a",
      "--opt-in",
      "--redacted",
    ]);

    const imported = await execaNode([
      "federation",
      "import-patterns",
      out,
      "--root",
      tmp,
      "--merge",
      "--json",
    ]);
    expect(imported.exitCode).toBe(0);
    const result = JSON.parse(imported.stdout);
    expect(result.ok).toBe(true);
    expect(result.dry_run).toBe(false);
    expect(result.written_count).toBe(1);

    const validate = await execaNode([
      "federation",
      "validate",
      result.target,
      "--json",
    ]);
    expect(validate.exitCode).toBe(0);
    expect(JSON.parse(validate.stdout).ok).toBe(true);
  });

  it("rejects imported records that contain raw fields", async () => {
    const tmp = makeTempDir();
    const invalid = path.join(tmp, "invalid-patterns.jsonl");
    await fs.writeFile(
      invalid,
      `${JSON.stringify({
        schema_version: "1",
        pattern_id: "b".repeat(64),
        tenant_hash: "c".repeat(64),
        source_hash: "d".repeat(64),
        pattern_class: "failure",
        signal: {
          predicate_hash: "e".repeat(64),
          predicate_present: true,
          evidence_layer: "raw",
        },
        evidence_kind: "completion_card",
        component_hashes: [],
        benchmark_metrics: null,
        created_at: "2026-05-24T00:00:00.000Z",
        retention_expires_at: "2026-06-23T00:00:00.000Z",
        redaction: {
          mode: "anonymized-pattern",
          redacted_required: true,
          raw_content_included: false,
          secret_scan_replacements: 0,
        },
        admission_authority: false,
        raw_content: "do not import this",
      })}\n`,
      "utf-8"
    );

    const { stdout, exitCode } = await execaNode([
      "federation",
      "import-patterns",
      invalid,
      "--root",
      tmp,
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const result = JSON.parse(stdout);
    expect(result.ok).toBe(false);
    expect(result.errors.join("\n")).toContain(
      "must NOT have additional properties"
    );
  });

  it("rejects imported records with non-canonical admission signal enums", async () => {
    const tmp = makeTempDir();
    const invalid = path.join(tmp, "invalid-signal-patterns.jsonl");
    await fs.writeFile(
      invalid,
      `${JSON.stringify({
        schema_version: "1",
        pattern_id: "b".repeat(64),
        tenant_hash: "c".repeat(64),
        source_hash: "d".repeat(64),
        pattern_class: "failure",
        signal: {
          predicate_hash: "e".repeat(64),
          predicate_present: true,
          admission_outcome: "approved",
          acceptance_status: "withheld",
          evidence_layer: "raw",
        },
        evidence_kind: "completion_card",
        component_hashes: [],
        benchmark_metrics: null,
        created_at: "2026-05-24T00:00:00.000Z",
        retention_expires_at: "2026-06-23T00:00:00.000Z",
        redaction: {
          mode: "anonymized-pattern",
          redacted_required: true,
          raw_content_included: false,
          secret_scan_replacements: 0,
        },
        admission_authority: false,
      })}\n`,
      "utf-8"
    );

    const { stdout, exitCode } = await execaNode([
      "federation",
      "import-patterns",
      invalid,
      "--root",
      tmp,
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const result = JSON.parse(stdout);
    expect(result.ok).toBe(false);
    expect(result.errors.join("\n")).toContain("must be equal to one of");
  });
});
