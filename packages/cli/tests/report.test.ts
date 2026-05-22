import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs-extra";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const TEST_TRACE_DIR = path.join(
  process.cwd(),
  ".x-harness-test-traces-report"
);

describe("report command", () => {
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

  it("outputs Markdown by default", async () => {
    const { stdout, exitCode } = await execaNode([
      "report",
      "--trace-dir",
      TEST_TRACE_DIR,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("# x-harness Report");
    expect(stdout).toContain("## Verify event accounting");
    expect(stdout).toContain("## Task lifecycle accounting");
    expect(stdout).toContain("## Admission accounting");
    expect(stdout).toContain("## Withheld accounting");
    expect(stdout).toContain("## Unknown or unlinked events");
    expect(stdout).toContain("## Denominator warning");
    expect(stdout).toContain("accepted: 1/3 cards");
    expect(stdout).toContain("blocked: 1/3 cards");
    expect(stdout).toContain("withheld: 2/3 cards");
  });

  it("outputs JSON with --json", async () => {
    const { stdout, exitCode } = await execaNode([
      "report",
      "--trace-dir",
      TEST_TRACE_DIR,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const report = JSON.parse(stdout);
    expect(report.total_events).toBe(3);
    expect(report.accepted).toBe(1);
    expect(report.withheld).toBe(2);
    expect(report.by_outcome.success).toBe(1);
    expect(report.by_outcome.failed).toBe(1);
    expect(report.by_outcome.blocked).toBe(1);
    expect(report.verify_event_accounting).toBeDefined();
    expect(report.verify_event_accounting.total_trace_events).toBe(3);
    expect(report.task_lifecycle_accounting).toBeDefined();
    expect(report.task_lifecycle_accounting.admitted).toBe(1);
    expect(report.admission_accounting).toBeDefined();
    expect(report.admission_accounting.accepted).toBe(1);
    expect(report.withheld_accounting).toBeDefined();
    expect(report.withheld_accounting.blocked).toBe(1);
    expect(report.unknown_or_unlinked_events).toBeDefined();
    expect(report.unknown_or_unlinked_events.count).toBe(0);
  });

  it("shows denominator warning", async () => {
    const { stdout } = await execaNode([
      "report",
      "--trace-dir",
      TEST_TRACE_DIR,
    ]);
    expect(stdout).toContain(
      "Verify-event success must not be interpreted as task-level success without denominator review."
    );
  });

  it("shows NOT_COMPUTABLE when no events", async () => {
    const emptyDir = path.join(process.cwd(), ".x-harness-test-traces-empty");
    await fs.ensureDir(emptyDir);
    const { stdout, exitCode } = await execaNode([
      "report",
      "--trace-dir",
      emptyDir,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("NOT_COMPUTABLE");
    await fs.remove(emptyDir);
  });

  it("outputs metrics in JSON with --metrics --json", async () => {
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
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const report = JSON.parse(stdout);
    expect(report.metrics).toBeDefined();
    expect(
      report.metrics.verification_strength.command_evidence_count
    ).toBeGreaterThanOrEqual(0);
    expect(report.metrics.state_consistency.owner_present).toBe(true);
    expect(report.metrics.replayability.completion_card_present).toBe(true);
    expect(report.metrics.cost.default_context_class).toBe("medium");
    expect(report.denominator_warning).toContain("must not be interpreted");
    expect(report.verify_event_accounting).toBeDefined();
    expect(report.verify_event_accounting.cards_analyzed).toBe(1);
    expect(report.task_lifecycle_accounting).toBeDefined();
    expect(report.admission_accounting).toBeDefined();
    expect(report.withheld_accounting).toBeDefined();
    expect(report.unknown_or_unlinked_events).toBeDefined();
  });

  it("outputs metrics in Markdown with --metrics", async () => {
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
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("# x-harness Metrics Report");
    expect(stdout).toContain("## Verification strength");
    expect(stdout).toContain("## Verify event accounting");
    expect(stdout).toContain("## Task lifecycle accounting");
    expect(stdout).toContain("## Admission accounting");
    expect(stdout).toContain("## Withheld accounting");
    expect(stdout).toContain("## Unknown or unlinked events");
    expect(stdout).toContain("## Denominator warning");
  });

  it("metrics fail when card not found", async () => {
    const { exitCode } = await execaNode([
      "report",
      "--metrics",
      "--card",
      "/nonexistent/card.yaml",
    ]);
    expect(exitCode).toBe(2);
  });
});
