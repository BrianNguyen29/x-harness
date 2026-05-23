import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs-extra";
import * as path from "node:path";
import { appendTrace, readTrace, verifyTraceChain } from "../src/core/trace.js";
import { execaNode } from "../src/test-helpers.js";

const TEST_TRACE_DIR = path.join(process.cwd(), ".x-harness-test-traces-hash");

describe("trace hash chain", () => {
  beforeEach(async () => {
    await fs.ensureDir(TEST_TRACE_DIR);
    await fs.emptyDir(TEST_TRACE_DIR);
  });

  afterEach(async () => {
    await fs.remove(TEST_TRACE_DIR);
  });

  it("enriches appended events with previous_hash and event_hash", async () => {
    const event1 = {
      event_id: "E1",
      event_type: "verify_completed",
      outcome: "success",
    };
    const enriched1 = await appendTrace(event1, TEST_TRACE_DIR);
    expect(enriched1.event_hash).toBeDefined();
    expect(enriched1.previous_hash).toBeNull();

    const event2 = {
      event_id: "E2",
      event_type: "verify_completed",
      outcome: "failed",
    };
    const enriched2 = await appendTrace(event2, TEST_TRACE_DIR);
    expect(enriched2.event_hash).toBeDefined();
    expect(enriched2.previous_hash).toBe(enriched1.event_hash);
  });

  it("verifies a valid chain", async () => {
    await appendTrace(
      { event_id: "E1", event_type: "verify_completed", outcome: "success" },
      TEST_TRACE_DIR
    );
    await appendTrace(
      { event_id: "E2", event_type: "verify_completed", outcome: "failed" },
      TEST_TRACE_DIR
    );

    const events = await readTrace(TEST_TRACE_DIR);
    const result = verifyTraceChain(events);
    expect(result.valid).toBe(true);
    expect(result.eventsChecked).toBe(2);
  });

  it("detects tampered event_hash", async () => {
    await appendTrace(
      { event_id: "E1", event_type: "verify_completed", outcome: "success" },
      TEST_TRACE_DIR
    );
    await appendTrace(
      { event_id: "E2", event_type: "verify_completed", outcome: "failed" },
      TEST_TRACE_DIR
    );

    const events = await readTrace(TEST_TRACE_DIR);
    events[1].event_hash = "tampered";
    const result = verifyTraceChain(events);
    expect(result.valid).toBe(false);
    expect(result.firstBrokenIndex).toBe(1);
    expect(result.firstBrokenEventId).toBe("E2");
  });

  it("detects tampered previous_hash linkage", async () => {
    await appendTrace(
      { event_id: "E1", event_type: "verify_completed", outcome: "success" },
      TEST_TRACE_DIR
    );
    await appendTrace(
      { event_id: "E2", event_type: "verify_completed", outcome: "failed" },
      TEST_TRACE_DIR
    );

    const events = await readTrace(TEST_TRACE_DIR);
    events[1].previous_hash = "tampered";
    const result = verifyTraceChain(events);
    expect(result.valid).toBe(false);
    expect(result.firstBrokenIndex).toBe(1);
  });

  it("handles legacy events without event_hash gracefully", async () => {
    const filePath = path.join(TEST_TRACE_DIR, "events.jsonl");
    await fs.writeFile(
      filePath,
      JSON.stringify({
        event_id: "LEGACY1",
        event_type: "verify_completed",
        outcome: "success",
      }) + "\n"
    );

    const events = await readTrace(TEST_TRACE_DIR);
    const result = verifyTraceChain(events);
    expect(result.valid).toBe(true);
    expect(result.eventsChecked).toBe(1);
  });

  it("handles mixed legacy and hashed events", async () => {
    const filePath = path.join(TEST_TRACE_DIR, "events.jsonl");
    await fs.writeFile(
      filePath,
      JSON.stringify({
        event_id: "LEGACY1",
        event_type: "verify_completed",
        outcome: "success",
      }) + "\n"
    );

    await appendTrace(
      { event_id: "E2", event_type: "verify_completed", outcome: "failed" },
      TEST_TRACE_DIR
    );

    const events = await readTrace(TEST_TRACE_DIR);
    const result = verifyTraceChain(events);
    expect(result.valid).toBe(true);
    expect(result.eventsChecked).toBe(2);
  });

  it("returns empty chain as valid", () => {
    const result = verifyTraceChain([]);
    expect(result.valid).toBe(true);
    expect(result.eventsChecked).toBe(0);
  });

  it("hash includes all event fields except previous_hash and event_hash", async () => {
    const event1 = {
      event_id: "E1",
      event_type: "verify_completed",
      outcome: "success",
      tier: "standard",
      task_id: "TASK-1",
    };
    await appendTrace(event1, TEST_TRACE_DIR);

    // Tamper with an extra field — should break chain
    const events = await readTrace(TEST_TRACE_DIR);
    const tamperedEvent = { ...events[0], tier: "deep" };
    const result = verifyTraceChain([tamperedEvent]);
    expect(result.valid).toBe(false);
    expect(result.firstBrokenIndex).toBe(0);
  });
});

describe("trace verify-chain CLI", () => {
  const CLI_TRACE_DIR = path.join(
    process.cwd(),
    ".x-harness-test-traces-cli-chain"
  );

  beforeEach(async () => {
    await fs.ensureDir(CLI_TRACE_DIR);
    await fs.emptyDir(CLI_TRACE_DIR);
  });

  afterEach(async () => {
    await fs.remove(CLI_TRACE_DIR);
  });

  it("passes for a valid chain", async () => {
    await appendTrace(
      { event_id: "E1", event_type: "verify_completed", outcome: "success" },
      CLI_TRACE_DIR
    );
    await appendTrace(
      { event_id: "E2", event_type: "verify_completed", outcome: "success" },
      CLI_TRACE_DIR
    );

    const { stdout, exitCode } = await execaNode([
      "trace",
      "verify-chain",
      "--trace-dir",
      CLI_TRACE_DIR,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("chain valid");
    expect(stdout).toContain("2 event(s)");
  });

  it("passes for a valid chain from an explicit trace file", async () => {
    await appendTrace(
      { event_id: "E1", event_type: "verify_completed", outcome: "success" },
      CLI_TRACE_DIR
    );
    await appendTrace(
      { event_id: "E2", event_type: "verify_completed", outcome: "success" },
      CLI_TRACE_DIR
    );

    const traceFile = path.join(CLI_TRACE_DIR, "events.jsonl");
    const { stdout, exitCode } = await execaNode([
      "trace",
      "verify-chain",
      "--from",
      traceFile,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("chain valid");
    expect(stdout).toContain("2 event(s)");
  });

  it("fails for a tampered chain", async () => {
    await appendTrace(
      { event_id: "E1", event_type: "verify_completed", outcome: "success" },
      CLI_TRACE_DIR
    );
    await appendTrace(
      { event_id: "E2", event_type: "verify_completed", outcome: "success" },
      CLI_TRACE_DIR
    );

    const events = await readTrace(CLI_TRACE_DIR);
    events[1].event_hash = "tampered";
    await fs.writeFile(
      path.join(CLI_TRACE_DIR, "events.jsonl"),
      events.map((e) => JSON.stringify(e)).join("\n") + "\n"
    );

    const { stderr, exitCode } = await execaNode([
      "trace",
      "verify-chain",
      "--trace-dir",
      CLI_TRACE_DIR,
    ]);
    expect(exitCode).toBe(1);
    expect(stderr).toContain("chain broken");
  });
});
