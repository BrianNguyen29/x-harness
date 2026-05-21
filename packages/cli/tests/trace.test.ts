import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs-extra";
import * as path from "node:path";
import { appendTrace, readTrace } from "../src/core/trace.js";

const TEST_TRACE_DIR = path.join(process.cwd(), ".claimgate-test-traces");

describe("trace", () => {
  beforeEach(async () => {
    await fs.ensureDir(TEST_TRACE_DIR);
    await fs.emptyDir(TEST_TRACE_DIR);
  });

  afterEach(async () => {
    await fs.remove(TEST_TRACE_DIR);
  });

  it("appends and reads trace events", async () => {
    const event1 = { event_id: "E1", event_type: "verify_completed", outcome: "success" };
    const event2 = { event_id: "E2", event_type: "verify_completed", outcome: "failed" };

    await appendTrace(event1, TEST_TRACE_DIR);
    await appendTrace(event2, TEST_TRACE_DIR);

    const events = await readTrace(TEST_TRACE_DIR);
    expect(events).toHaveLength(2);
    expect(events[0].event_id).toBe("E1");
    expect(events[1].event_id).toBe("E2");
  });

  it("returns empty array when trace file does not exist", async () => {
    const events = await readTrace(path.join(TEST_TRACE_DIR, "nonexistent"));
    expect(events).toHaveLength(0);
  });
});
