import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs-extra";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const TEST_TRACE_DIR = path.join(
  process.cwd(),
  ".x-harness-test-traces-report-html"
);

describe("report --format html", () => {
  beforeEach(async () => {
    await fs.ensureDir(TEST_TRACE_DIR);
    await fs.writeFile(
      path.join(TEST_TRACE_DIR, "events.jsonl"),
      JSON.stringify({
        event_id: "E1",
        event_type: "verify_completed",
        outcome: "success",
        acceptance_status: "accepted",
      }) +
        "\n" +
        JSON.stringify({
          event_id: "E2",
          event_type: "verify_completed",
          outcome: "failed",
          acceptance_status: "withheld",
        }) +
        "\n" +
        JSON.stringify({
          event_id: "E3",
          event_type: "verify_completed",
          outcome: "blocked",
          acceptance_status: "withheld",
        }) +
        "\n"
    );
  });

  afterEach(async () => {
    await fs.remove(TEST_TRACE_DIR);
  });

  it("produces a valid HTML document", async () => {
    const { stdout, exitCode } = await execaNode([
      "report",
      "--trace-dir",
      TEST_TRACE_DIR,
      "--format",
      "html",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("<!DOCTYPE html>");
    expect(stdout).toContain('<html lang="en">');
    expect(stdout).toContain("<title>x-harness Audit Report</title>");
    expect(stdout).toContain("</html>");
  });

  it("escapes malicious payload in event data", async () => {
    const maliciousDir = path.join(
      process.cwd(),
      ".x-harness-test-traces-malicious"
    );
    await fs.ensureDir(maliciousDir);
    await fs.writeFile(
      path.join(maliciousDir, "events.jsonl"),
      JSON.stringify({
        event_id: "E1",
        event_type: "verify_completed",
        outcome: "success",
        acceptance_status: "accepted",
        task_id: "<script>alert('xss')</script>",
      }) + "\n"
    );

    const { stdout, exitCode } = await execaNode([
      "report",
      "--trace-dir",
      maliciousDir,
      "--format",
      "html",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).not.toContain("<script>alert('xss')</script>");
    // JSON.stringify inside <pre><code> escapes ' as &#39;, then escapeHtml wraps it
    expect(stdout).toContain(
      "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
    );
    await fs.remove(maliciousDir);
  });

  it("renders summary counts correctly", async () => {
    const { stdout, exitCode } = await execaNode([
      "report",
      "--trace-dir",
      TEST_TRACE_DIR,
      "--format",
      "html",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Total events");
    expect(stdout).toContain(">3<");
    expect(stdout).toContain("Accepted");
    expect(stdout).toContain("Withheld");
  });

  it("includes denominator warning", async () => {
    const { stdout, exitCode } = await execaNode([
      "report",
      "--trace-dir",
      TEST_TRACE_DIR,
      "--format",
      "html",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Denominator warning");
    expect(stdout).toContain("must not be interpreted");
  });

  it("supports html format for metrics report", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "success-standard-scoped-evidence",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "report",
      "--metrics",
      "--card",
      cardPath,
      "--format",
      "html",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("<!DOCTYPE html>");
    expect(stdout).toContain("x-harness Metrics Report");
  });

  it("renders structured metrics sections with tables", async () => {
    const cardPath = path.join(
      repoRoot,
      "examples",
      "golden",
      "success-standard-scoped-evidence",
      "completion-card.yaml"
    );
    const { stdout, exitCode } = await execaNode([
      "report",
      "--metrics",
      "--card",
      cardPath,
      "--format",
      "html",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("<table>");
    expect(stdout).toContain("Verification Strength");
    expect(stdout).toContain("State Consistency");
    expect(stdout).toContain("Recovery Ability");
    expect(stdout).toContain("Replayability");
    expect(stdout).toContain("Cost");
    expect(stdout).toContain("Admission Outcome");
    expect(stdout).toContain("Denominator warning");
  });

  it("preserves escaping in metrics html report", async () => {
    const maliciousCardPath = path.join(
      repoRoot,
      ".x-harness-test-malicious-card.yaml"
    );
    await fs.writeFile(
      maliciousCardPath,
      `schema_version: "1.0"
task_id: "TASK-1"
tier: standard
owner: "Alice"
accountable: "Bob"
claim:
  fix_status: fixed
evidence:
  files_changed:
    - "<script>alert('xss')</script>.ts"
verification:
  status: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: verify
  owner: verifier
`
    );
    const { stdout, exitCode } = await execaNode([
      "report",
      "--metrics",
      "--card",
      maliciousCardPath,
      "--format",
      "html",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).not.toContain("<script>alert('xss')</script>");
    expect(stdout).toContain("Raw JSON (for debugging)");
    expect(stdout).toContain("&quot;outcome&quot;");
    await fs.remove(maliciousCardPath);
  });
});
