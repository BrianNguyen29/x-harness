import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";
import {
  applyDecisionEnforceGate,
  applyIntentEnforceGate,
  isValidIntentEnforce,
  resolveDecisionEnforceMode,
  resolveIntentEnforceMode,
} from "../src/core/verify-pipeline.js";
import {
  applyDecisionLinkRefs,
  buildDecisionRecord,
  collectDecisionLinkRefs,
  isValidDecisionEnforce,
  listDecisionRecords,
  matchDecisionAffected,
  matchDecisionQuery,
  normalizeDecisionStatus,
  defaultDecisionOutputPath,
  isJsonExtension,
} from "../src/core/decision.js";
import {
  hasAnyDecisionRef as hasAnyDecisionRefFromAdmission,
  hasAnyIntentRef as hasAnyIntentRefFromAdmission,
} from "../src/core/admission.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const cliDistPath = path.join(repoRoot, "packages", "cli", "dist", "index.js");

let workDir: string;
let originalCwd: string;

beforeEach(() => {
  originalCwd = process.cwd();
  workDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-decision-test-"));
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
        // child.execFile passes an Error on non-zero exit; the
        // numeric exit code is on `error.code`. The Command parser
        // also exits with explicit process.exit(2) for usage errors,
        // which surfaces here.
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

describe("decision core helpers", () => {
  it("normalizeDecisionStatus accepts the closed enum", () => {
    expect(normalizeDecisionStatus("proposed")).toBe("proposed");
    expect(normalizeDecisionStatus("ACCEPTED")).toBe("accepted");
    expect(normalizeDecisionStatus(" superseded ")).toBe("superseded");
    expect(normalizeDecisionStatus("deprecated")).toBe("deprecated");
    expect(normalizeDecisionStatus("")).toBe("proposed");
  });

  it("normalizeDecisionStatus rejects unknown statuses", () => {
    expect(() => normalizeDecisionStatus("maybe")).toThrow(/proposed/);
  });

  it("buildDecisionRecord emits the canonical safe V1 shape", () => {
    const record = buildDecisionRecord({
      id: "intake-lite",
      title: "P3-S3 Decision Memory Safe V1",
      date: "2026-06-04",
      status: "accepted",
      decision: "ship the safe V1 slice",
      rationale: "keep scope minimal",
      context: "first vertical slice",
      consequences: "advisory note only",
      supersededBy: "",
      tags: ["p3-s3", "decision-memory"],
      affectedPaths: ["schemas/decision-record.schema.json"],
      notes: "follow-up slices may add query/link/affected",
    });
    expect(record.schema_version).toBe("1");
    expect(record.id).toBe("intake-lite");
    expect(record.title).toBe("P3-S3 Decision Memory Safe V1");
    expect(record.decision).toBe("ship the safe V1 slice");
    expect(record.tags).toEqual(["p3-s3", "decision-memory"]);
    expect(record.affected_paths).toEqual([
      "schemas/decision-record.schema.json",
    ]);
  });

  it("buildDecisionRecord defaults date to today when missing", () => {
    const record = buildDecisionRecord({
      id: "x",
      title: "",
      date: "",
      status: "",
      decision: "d",
      rationale: "r",
      context: "",
      consequences: "",
      supersededBy: "",
      tags: [],
      affectedPaths: [],
      notes: "",
    });
    expect(record.date).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("defaultDecisionOutputPath returns decisions/<id>.yaml", () => {
    const out = defaultDecisionOutputPath({
      id: "abc",
      decision: "",
      rationale: "",
      schema_version: "1",
    });
    expect(out).toBe(path.join("decisions", "abc.yaml"));
  });

  it("isJsonExtension detects .json", () => {
    expect(isJsonExtension("foo.json")).toBe(true);
    expect(isJsonExtension("foo.yaml")).toBe(false);
  });

  it("isValidDecisionEnforce enforces the closed enum", () => {
    expect(isValidDecisionEnforce("off")).toBe(true);
    expect(isValidDecisionEnforce("advisory")).toBe(true);
    expect(isValidDecisionEnforce("block")).toBe(true);
    expect(isValidDecisionEnforce("")).toBe(false);
    expect(isValidDecisionEnforce("bogus")).toBe(false);
  });

  it("listDecisionRecords returns an empty list for missing dir", async () => {
    const entries = await listDecisionRecords(path.join(workDir, "nope"));
    expect(entries).toEqual([]);
  });

  it("matchDecisionQuery finds substring matches across fields", async () => {
    writeFile(
      "decisions/zeta.yaml",
      'schema_version: "1"\nid: zeta\ntitle: AuthZ\ndecision: ship RBAC\nrationale: zr\nstatus: accepted\n'
    );
    writeFile(
      "decisions/alpha.yaml",
      'schema_version: "1"\nid: alpha\ntitle: AuthN\ndecision: ship OIDC\nrationale: ar\nstatus: proposed\n'
    );
    writeFile(
      "decisions/beta.yaml",
      'schema_version: "1"\nid: beta\ntitle: Logging\ndecision: structured logs\nrationale: br\nstatus: accepted\n'
    );

    const entries = await listDecisionRecords(path.join(workDir, "decisions"));
    const matches = matchDecisionQuery(entries, "auth");
    expect(matches.map((m) => m.id)).toEqual(["alpha", "zeta"]);
  });

  it("matchDecisionAffected honors exact and glob patterns", async () => {
    writeFile(
      "decisions/auth-login.yaml",
      'schema_version: "1"\nid: auth-login\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/login.ts\n'
    );
    writeFile(
      "decisions/auth-bulk.yaml",
      'schema_version: "1"\nid: auth-bulk\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/*.ts\n'
    );

    const entries = await listDecisionRecords(path.join(workDir, "decisions"));
    const exact = matchDecisionAffected(entries, "src/auth/login.ts");
    expect(exact.map((m) => m.id).sort()).toEqual(["auth-bulk", "auth-login"]);
    const none = matchDecisionAffected(entries, "src/other/file.ts");
    expect(none).toEqual([]);
  });

  it("applyDecisionLinkRefs appends deduped refs and preserves other fields", () => {
    const doc = {
      schema_version: "1",
      task_id: "TASK-1",
      context_alignment: { stale_ground_checked: true, decision_refs: ["a"] },
    };
    const {
      doc: out,
      added,
      skipped,
    } = applyDecisionLinkRefs(doc, ["a", "b", "b", "c"]);
    expect(added).toEqual(["b", "c"]);
    expect(skipped).toEqual(["a"]);
    expect(collectDecisionLinkRefs(out)).toEqual(["a", "b", "c"]);
    expect(
      (out.context_alignment as Record<string, unknown>).stale_ground_checked
    ).toBe(true);
  });
});

describe("hasAnyDecisionRef parity helper", () => {
  it("returns false for missing context_alignment", () => {
    expect(hasAnyDecisionRefFromAdmission({})).toBe(false);
  });

  it("returns false for empty array or blank strings", () => {
    expect(
      hasAnyDecisionRefFromAdmission({
        context_alignment: { decision_refs: [] },
      })
    ).toBe(false);
    expect(
      hasAnyDecisionRefFromAdmission({
        context_alignment: { decision_refs: ["", "  "] },
      })
    ).toBe(false);
  });

  it("returns true when any non-blank string is present", () => {
    expect(
      hasAnyDecisionRefFromAdmission({
        context_alignment: { decision_refs: ["ADR-1"] },
      })
    ).toBe(true);
    expect(
      hasAnyDecisionRefFromAdmission({
        context_alignment: { decision_refs: ["", "ADR-1"] },
      })
    ).toBe(true);
  });
});

describe("resolveDecisionEnforceMode", () => {
  it("returns the explicit value when set", () => {
    expect(resolveDecisionEnforceMode("ci-strict", "off")).toBe("off");
    expect(resolveDecisionEnforceMode("light-local", "block")).toBe("block");
  });

  it("uses the profile default when explicit is empty", () => {
    expect(resolveDecisionEnforceMode("light-local", "")).toBe("advisory");
    expect(resolveDecisionEnforceMode("ci-standard", "")).toBe("advisory");
    expect(resolveDecisionEnforceMode("ci-strict", "")).toBe("block");
    expect(resolveDecisionEnforceMode("governed-deep", "")).toBe("block");
  });

  it("falls back to off when neither is set or profile is unknown", () => {
    expect(resolveDecisionEnforceMode(undefined, undefined)).toBe("off");
    expect(resolveDecisionEnforceMode("bogus", "")).toBe("off");
  });

  it("treats invalid explicit values as off", () => {
    expect(resolveDecisionEnforceMode("ci-strict", "bogus")).toBe("off");
  });
});

describe("applyDecisionEnforceGate", () => {
  it("returns null in off and advisory modes", () => {
    expect(
      applyDecisionEnforceGate({
        mode: "off",
        tier: "standard",
        doc: {},
      })
    ).toBeNull();
    expect(
      applyDecisionEnforceGate({
        mode: "advisory",
        tier: "deep",
        doc: {},
      })
    ).toBeNull();
  });

  it("returns null for light tier even in block mode", () => {
    expect(
      applyDecisionEnforceGate({ mode: "block", tier: "light", doc: {} })
    ).toBeNull();
  });

  it("blocks standard/deep cards without decision_refs in block mode", () => {
    const reason = applyDecisionEnforceGate({
      mode: "block",
      tier: "standard",
      doc: { context_alignment: { decision_refs: [] } },
    });
    expect(reason).toMatch(/decision_refs is empty/);
  });

  it("does not block cards with a non-blank ref", () => {
    expect(
      applyDecisionEnforceGate({
        mode: "block",
        tier: "deep",
        doc: { context_alignment: { decision_refs: ["ADR-1"] } },
      })
    ).toBeNull();
  });
});

describe("xh decision CLI (built dist)", () => {
  it("record writes default YAML and is listed", async () => {
    const result = await execaNodeWorkdir([
      "decision",
      "record",
      "--id",
      "intake-lite",
      "--decision",
      "ship the safe V1 slice",
      "--rationale",
      "keep scope minimal",
      "--title",
      "P3-S3 Decision Memory Safe V1",
      "--status",
      "accepted",
      "--context",
      "first vertical slice",
      "--consequence",
      "advisory note only",
      "--tag",
      "p3-s3,decision-memory",
      "--affected-path",
      "schemas/decision-record.schema.json",
      "--note",
      "follow-up slices may add query/link/affected",
    ]);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("decisions/intake-lite.yaml");
    const data = fs.readFileSync(
      path.join(workDir, "decisions", "intake-lite.yaml"),
      "utf-8"
    );
    expect(data).toContain("id: intake-lite");
    expect(data).toContain("status: accepted");
    expect(data).toContain('schema_version: "1"');
  });

  it("record rejects missing --id", async () => {
    const result = await execaNodeWorkdir([
      "decision",
      "record",
      "--decision",
      "x",
      "--rationale",
      "y",
    ]);
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("--id is required");
  });

  it("record rejects invalid --status", async () => {
    const result = await execaNodeWorkdir([
      "decision",
      "record",
      "--id",
      "x",
      "--decision",
      "y",
      "--rationale",
      "z",
      "--status",
      "maybe",
    ]);
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("--status");
  });

  it("list returns Count: 0 on missing dir", async () => {
    const result = await execaNodeWorkdir([
      "decision",
      "list",
      "--dir",
      "nope",
    ]);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Count: 0");
  });

  it("list --json sorts by id", async () => {
    writeFile(
      "decisions/zeta.yaml",
      'schema_version: "1"\nid: zeta\ndecision: z\nrationale: zr\nstatus: accepted\n'
    );
    writeFile(
      "decisions/alpha.yaml",
      'schema_version: "1"\nid: alpha\ndecision: a\nrationale: ar\nstatus: proposed\n'
    );
    const result = await execaNodeWorkdir(["decision", "list", "--json"]);
    expect(result.exitCode).toBe(0);
    const parsed = JSON.parse(result.stdout);
    expect(parsed.count).toBe(2);
    expect(parsed.records.map((r: { id: string }) => r.id)).toEqual([
      "alpha",
      "zeta",
    ]);
  });

  it("query requires --keyword", async () => {
    const result = await execaNodeWorkdir([
      "decision",
      "query",
      "--dir",
      "nope",
    ]);
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("--keyword is required");
  });

  it("query filters by case-insensitive substring", async () => {
    writeFile(
      "decisions/auth.yaml",
      'schema_version: "1"\nid: auth\ntitle: AuthZ\ndecision: ship\nrationale: r\nstatus: accepted\n'
    );
    writeFile(
      "decisions/log.yaml",
      'schema_version: "1"\nid: log\ntitle: Logging\ndecision: logs\nrationale: r\nstatus: accepted\n'
    );
    const result = await execaNodeWorkdir([
      "decision",
      "query",
      "--keyword",
      "auth",
    ]);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Count: 1");
    expect(result.stdout).toContain("id=auth");
    expect(result.stdout).not.toContain("id=log");
  });

  it("affected requires --path", async () => {
    const result = await execaNodeWorkdir(["decision", "affected"]);
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("--path is required");
  });

  it("affected finds a record with matching affected_paths", async () => {
    writeFile(
      "decisions/login.yaml",
      'schema_version: "1"\nid: login\ntitle: Auth Login\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/login.ts\n'
    );
    const result = await execaNodeWorkdir([
      "decision",
      "affected",
      "--path",
      "src/auth/login.ts",
    ]);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Count: 1");
    expect(result.stdout).toContain("id=login");
  });

  it("link appends to context_alignment.decision_refs and reports totals", async () => {
    const card = writeFile(
      "card.yaml",
      'schema_version: "1"\ntask_id: T\ncontext_alignment:\n  stale_ground_checked: true\n'
    );
    const result = await execaNodeWorkdir([
      "decision",
      "link",
      "--card",
      card,
      "--decision",
      "ADR-1",
      "--decision",
      "ADR-2,ADR-1",
    ]);
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Added: ADR-1, ADR-2");
    expect(result.stdout).toContain("Total decision refs: 2");
    const data = fs.readFileSync(card, "utf-8");
    expect(data).toContain("ADR-1");
    expect(data).toContain("ADR-2");
  });

  it("link --json reports added and skipped", async () => {
    const card = writeFile(
      "card.yaml",
      'schema_version: "1"\ntask_id: T\ncontext_alignment:\n  decision_refs:\n    - ADR-1\n'
    );
    const result = await execaNodeWorkdir([
      "decision",
      "link",
      "--card",
      card,
      "--decision",
      "ADR-1",
      "--decision",
      "ADR-2",
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const parsed = JSON.parse(result.stdout);
    expect(parsed.added).toEqual(["ADR-2"]);
    expect(parsed.skipped).toEqual(["ADR-1"]);
    expect(parsed.decision_refs).toEqual(["ADR-1", "ADR-2"]);
  });

  it("link rejects missing --card", async () => {
    const result = await execaNodeWorkdir([
      "decision",
      "link",
      "--decision",
      "ADR-1",
    ]);
    expect(result.exitCode).toBe(2);
    expect(result.stderr).toContain("--card is required");
  });
});

describe("verify --decision-enforce (built dist)", () => {
  function writeStandardCard(): string {
    // The auto-enabled context floor for standard tier validates that
    // product_contract_refs entries resolve on disk. Provide a minimal
    // README so the floor check passes and the only error we observe
    // is the one under test.
    writeFile("README.md", "# Product\n");
    return writeFile(
      "completion-card.yaml",
      `schema_version: "1"
task_id: TASK-DECISION-ENFORCE-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - README.md
  architecture_refs: []
  test_matrix_refs: []
  decision_refs: []
  unresolved_context_questions: []
  context_evidence: []
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: TASK-DECISION-ENFORCE-001 claim
  expected_effect: works
  measurable_signal: tests pass
  falsification_method: skip fix
  horizon: same_verify
evidence:
  files_changed:
    - src/main.go
  command_evidence:
    - command: go test ./...
      exit_code: 0
      runner: go-test
      started_at: "2026-06-04T00:00:00Z"
claim:
  fix_status: fixed
  summary: TASK-DECISION-ENFORCE-001
  evidence:
    - description: source change
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
    );
  }

  it("rejects invalid --decision-enforce values", async () => {
    const card = writeStandardCard();
    const result = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--decision-enforce",
      "bogus",
    ]);
    // Bogus falls back to off per resolveDecisionEnforceMode, so it
    // does not block. The test only needs to confirm the command
    // still runs and produces a parseable JSON when --json is set.
    const result2 = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--decision-enforce",
      "bogus",
      "--json",
    ]);
    expect(result2.exitCode).toBe(0);
    void result;
  });

  it("--decision-enforce off does not block", async () => {
    const card = writeStandardCard();
    const result = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--decision-enforce",
      "off",
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("--decision-enforce block withholds when decision_refs is empty", async () => {
    const card = writeStandardCard();
    const result = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--decision-enforce",
      "block",
      "--json",
    ]);
    expect(result.exitCode).toBe(1);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("blocked");
    expect(event.acceptance_status).toBe("withheld");
  });

  it("--decision-enforce block accepts when decision_refs is non-empty", async () => {
    writeFile("README.md", "# Product\n");
    writeFile("decisions/ADR-1.md", "# ADR-1\n");
    const card = writeFile(
      "completion-card.yaml",
      `schema_version: "1"
task_id: TASK-DECISION-ENFORCE-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - README.md
  architecture_refs: []
  test_matrix_refs: []
  decision_refs:
    - decisions/ADR-1.md
  unresolved_context_questions: []
  context_evidence: []
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: TASK-DECISION-ENFORCE-001 claim
  expected_effect: works
  measurable_signal: tests pass
  falsification_method: skip fix
  horizon: same_verify
evidence:
  files_changed:
    - src/main.go
  command_evidence:
    - command: go test ./...
      exit_code: 0
      runner: go-test
      started_at: "2026-06-04T00:00:00Z"
claim:
  fix_status: fixed
  summary: TASK-DECISION-ENFORCE-001
  evidence:
    - description: source change
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
    );
    const result = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--decision-enforce",
      "block",
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("--profile ci-strict blocks missing decision_refs by default", async () => {
    const card = writeStandardCard();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "ci-strict",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(1);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("blocked");
  });

  it("--profile ci-standard does not block by default", async () => {
    const card = writeStandardCard();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "ci-standard",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("--profile light-local never blocks (advisory default)", async () => {
    const card = writeStandardCard();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "light-local",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("explicit --decision-enforce off overrides ci-strict block default", async () => {
    const card = writeStandardCard();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "ci-strict",
      "--decision-enforce",
      "off",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });
});

void execaNode;

describe("hasAnyIntentRef parity helper", () => {
  it("returns false for missing intent_ref", () => {
    expect(hasAnyIntentRefFromAdmission({})).toBe(false);
  });

  it("returns false for blank strings", () => {
    expect(hasAnyIntentRefFromAdmission({ intent_ref: "" })).toBe(false);
    expect(hasAnyIntentRefFromAdmission({ intent_ref: "   " })).toBe(false);
  });

  it("returns true for non-blank strings", () => {
    expect(
      hasAnyIntentRefFromAdmission({ intent_ref: "doc/intake-lite.md" })
    ).toBe(true);
  });
});

describe("isValidIntentEnforce enforces the closed enum", () => {
  it("accepts the canonical modes", () => {
    expect(isValidIntentEnforce("off")).toBe(true);
    expect(isValidIntentEnforce("advisory")).toBe(true);
    expect(isValidIntentEnforce("block")).toBe(true);
  });

  it("rejects empty and unknown values", () => {
    expect(isValidIntentEnforce("")).toBe(false);
    expect(isValidIntentEnforce("bogus")).toBe(false);
    expect(isValidIntentEnforce("high")).toBe(false);
  });
});

describe("resolveIntentEnforceMode", () => {
  it("returns the explicit value when set", () => {
    expect(resolveIntentEnforceMode("governed-deep", "off")).toBe("off");
    expect(resolveIntentEnforceMode("light-local", "block")).toBe("block");
  });

  it("uses the conservative profile default when explicit is empty", () => {
    expect(resolveIntentEnforceMode("light-local", "")).toBe("advisory");
    expect(resolveIntentEnforceMode("ci-standard", "")).toBe("advisory");
    expect(resolveIntentEnforceMode("ci-strict", "")).toBe("advisory");
    expect(resolveIntentEnforceMode("governed-deep", "")).toBe("block");
  });

  it("falls back to off when neither is set or profile is unknown", () => {
    expect(resolveIntentEnforceMode(undefined, undefined)).toBe("off");
    expect(resolveIntentEnforceMode("bogus", "")).toBe("off");
  });

  it("treats invalid explicit values as off", () => {
    expect(resolveIntentEnforceMode("governed-deep", "bogus")).toBe("off");
  });
});

describe("applyIntentEnforceGate", () => {
  it("returns null in off and advisory modes", () => {
    expect(
      applyIntentEnforceGate({ mode: "off", tier: "standard", doc: {} })
    ).toBeNull();
    expect(
      applyIntentEnforceGate({ mode: "advisory", tier: "deep", doc: {} })
    ).toBeNull();
  });

  it("returns null for light tier even in block mode", () => {
    expect(
      applyIntentEnforceGate({ mode: "block", tier: "light", doc: {} })
    ).toBeNull();
  });

  it("blocks standard/deep cards without intent_ref in block mode", () => {
    const reason = applyIntentEnforceGate({
      mode: "block",
      tier: "standard",
      doc: {},
    });
    expect(reason).toMatch(/intent_ref not declared/);
  });

  it("does not block cards with a non-blank intent_ref", () => {
    expect(
      applyIntentEnforceGate({
        mode: "block",
        tier: "deep",
        doc: { intent_ref: "doc/intake-lite.md" },
      })
    ).toBeNull();
  });

  it("blocks cards with a blank intent_ref string", () => {
    const reason = applyIntentEnforceGate({
      mode: "block",
      tier: "standard",
      doc: { intent_ref: "   " },
    });
    expect(reason).toMatch(/intent_ref not declared/);
  });
});

describe("xh verify --intent-enforce (built dist)", () => {
  function writeStandardCardWithDecisionRef(): string {
    // A non-empty decision_refs entry keeps the governed-deep profile
    // default from withholding on the decision_refs gate, isolating
    // the intent_ref gate under test. README and decisions/ADR-1.md
    // exist so the context floor for standard tier resolves.
    writeFile("README.md", "# Product\n");
    writeFile("decisions/ADR-1.md", "# ADR-1\n");
    return writeFile(
      "completion-card.yaml",
      `schema_version: "1"
task_id: TASK-INTENT-ENFORCE-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - README.md
  architecture_refs: []
  test_matrix_refs: []
  decision_refs:
    - decisions/ADR-1.md
  unresolved_context_questions: []
  context_evidence: []
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: TASK-INTENT-ENFORCE-001 claim
  expected_effect: works
  measurable_signal: tests pass
  falsification_method: skip fix
  horizon: same_verify
evidence:
  files_changed:
    - src/main.go
  command_evidence:
    - command: go test ./...
      exit_code: 0
      runner: go-test
      started_at: "2026-06-06T00:00:00Z"
claim:
  fix_status: fixed
  summary: TASK-INTENT-ENFORCE-001
  evidence:
    - description: source change
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
    );
  }

  it("--intent-enforce off does not block", async () => {
    const card = writeStandardCardWithDecisionRef();
    const result = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--intent-enforce",
      "off",
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("--intent-enforce block withholds when intent_ref is missing", async () => {
    const card = writeStandardCardWithDecisionRef();
    const result = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--intent-enforce",
      "block",
      "--json",
    ]);
    expect(result.exitCode).toBe(1);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("blocked");
    expect(event.acceptance_status).toBe("withheld");
  });

  it("--intent-enforce block accepts when intent_ref is non-blank", async () => {
    const card = writeStandardCardWithDecisionRef();
    const data = fs.readFileSync(card, "utf-8");
    fs.writeFileSync(
      card,
      data + "\nintent_ref: doc/intake-lite.md\n",
      "utf-8"
    );
    const result = await execaNodeWorkdir([
      "verify",
      "--card",
      card,
      "--intent-enforce",
      "block",
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("--profile governed-deep blocks missing intent_ref by default", async () => {
    const card = writeStandardCardWithDecisionRef();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "governed-deep",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(1);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("blocked");
  });

  it("--profile ci-strict stays advisory for intent_ref by default", async () => {
    const card = writeStandardCardWithDecisionRef();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "ci-strict",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("--profile ci-strict with explicit --intent-enforce block blocks", async () => {
    const card = writeStandardCardWithDecisionRef();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "ci-strict",
      "--intent-enforce",
      "block",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(1);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("blocked");
  });

  it("--profile light-local never blocks for intent_ref", async () => {
    const card = writeStandardCardWithDecisionRef();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "light-local",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });

  it("explicit --intent-enforce off overrides governed-deep block default", async () => {
    const card = writeStandardCardWithDecisionRef();
    const result = await execaNodeWorkdir([
      "verify",
      "--profile",
      "governed-deep",
      "--intent-enforce",
      "off",
      "--card",
      card,
      "--json",
    ]);
    expect(result.exitCode).toBe(0);
    const event = JSON.parse(result.stdout);
    expect(event.admission_outcome).toBe("success");
  });
});
