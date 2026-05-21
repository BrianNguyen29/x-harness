import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs-extra";
import * as path from "node:path";
import { execaNode } from "../src/test-helpers.js";

const TEST_TRACE_DIR = path.join(process.cwd(), ".x-harness-test-traces-report");

describe("report command", () => {
  beforeEach(async () => {
    await fs.ensureDir(TEST_TRACE_DIR);
    await fs.writeFile(
      path.join(TEST_TRACE_DIR, "events.jsonl"),
      JSON.stringify({ event_id: "E1", event_type: "verify_completed", outcome: "success", acceptance_status: "accepted" }) + "\n" +
      JSON.stringify({ event_id: "E2", event_type: "verify_completed", outcome: "failed", acceptance_status: "withheld" }) + "\n" +
      JSON.stringify({ event_id: "E3", event_type: "verify_completed", outcome: "blocked", acceptance_status: "withheld" }) + "\n"
    );
  });

  afterEach(async () => {
    await fs.remove(TEST_TRACE_DIR);
  });

  it("outputs Markdown by default", async () => {
    const { stdout, exitCode } = await execaNode(["report", "--trace-dir", TEST_TRACE_DIR]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("# x-harness Report");
    expect(stdout).toContain("## Verification summary");
    expect(stdout).toContain("## Denominator warning");
    expect(stdout).toContain("accepted: 1/3 cards");
    expect(stdout).toContain("blocked: 1/3 cards");
    expect(stdout).toContain("withheld: 2/3 cards");
  });

  it("outputs JSON with --json", async () => {
    const { stdout, exitCode } = await execaNode(["report", "--trace-dir", TEST_TRACE_DIR, "--json"]);
    expect(exitCode).toBe(0);
    const report = JSON.parse(stdout);
    expect(report.total_events).toBe(3);
    expect(report.accepted).toBe(1);
    expect(report.withheld).toBe(2);
    expect(report.by_outcome.success).toBe(1);
    expect(report.by_outcome.failed).toBe(1);
    expect(report.by_outcome.blocked).toBe(1);
  });

  it("shows denominator warning", async () => {
    const { stdout } = await execaNode(["report", "--trace-dir", TEST_TRACE_DIR]);
    expect(stdout).toContain("Verify-event success must not be interpreted as task-level success without denominator review.");
  });

  it("shows NOT_COMPUTABLE when no events", async () => {
    const emptyDir = path.join(process.cwd(), ".x-harness-test-traces-empty");
    await fs.ensureDir(emptyDir);
    const { stdout, exitCode } = await execaNode(["report", "--trace-dir", emptyDir]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("NOT_COMPUTABLE");
    await fs.remove(emptyDir);
  });
});
