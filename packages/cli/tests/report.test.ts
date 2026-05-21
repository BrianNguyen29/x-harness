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
      JSON.stringify({ event_id: "E2", event_type: "verify_completed", outcome: "failed", acceptance_status: "withheld" }) + "\n"
    );
  });

  afterEach(async () => {
    await fs.remove(TEST_TRACE_DIR);
  });

  it("summarizes trace events", async () => {
    const { stdout, exitCode } = await execaNode(["report", "--trace-dir", TEST_TRACE_DIR]);
    expect(exitCode).toBe(0);
    const report = JSON.parse(stdout);
    expect(report.total_events).toBe(2);
    expect(report.accepted).toBe(1);
    expect(report.withheld).toBe(1);
    expect(report.by_outcome.success).toBe(1);
    expect(report.by_outcome.failed).toBe(1);
  });
});
