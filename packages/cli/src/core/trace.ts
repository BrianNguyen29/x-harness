import fs from "fs-extra";
import * as path from "node:path";
import { createHash } from "node:crypto";

export interface TraceEvent {
  event_id: string;
  event_type: string;
  previous_hash?: string | null;
  event_hash?: string | null;
  [key: string]: unknown;
}

function sha256String(input: string): string {
  return createHash("sha256").update(input, "utf-8").digest("hex");
}

function computeEventHash(
  event: TraceEvent,
  previousHash: string | null
): string {
  // Hash canonical full event excluding previous_hash and event_hash itself.
  // previousHash (the prior event's event_hash) is included as chain linkage.
  const { previous_hash: _prev, event_hash: _h, ...rest } = event;
  const payload = {
    ...rest,
    previous_hash: previousHash,
  };
  return sha256String(JSON.stringify(payload));
}

export async function appendTrace(
  event: TraceEvent,
  traceDir = ".x-harness/traces"
): Promise<TraceEvent> {
  await fs.ensureDir(traceDir);
  const filePath = path.join(traceDir, "events.jsonl");

  // Read existing events to get the last hash
  const events = await readTrace(traceDir);
  const previousHash =
    events.length > 0 ? (events[events.length - 1].event_hash ?? null) : null;

  const enriched: TraceEvent = {
    ...event,
    previous_hash: previousHash,
    event_hash: computeEventHash(event, previousHash),
  };

  await fs.appendFile(filePath, JSON.stringify(enriched) + "\n");
  return enriched;
}

export async function readTrace(
  traceDir = ".x-harness/traces"
): Promise<TraceEvent[]> {
  const filePath = path.join(traceDir, "events.jsonl");
  return readTraceFromFile(filePath);
}

export async function readTraceFromFile(
  filePath: string
): Promise<TraceEvent[]> {
  if (!(await fs.pathExists(filePath))) return [];
  const content = await fs.readFile(filePath, "utf-8");
  return content
    .split("\n")
    .map((line: string) => line.trim())
    .filter((line: string) => line.length > 0)
    .map((line: string) => JSON.parse(line) as TraceEvent);
}

export interface ChainVerificationResult {
  valid: boolean;
  eventsChecked: number;
  firstBrokenIndex: number | null;
  firstBrokenEventId: string | null;
  expectedHash: string | null;
  actualHash: string | null;
}

export function verifyTraceChain(
  events: TraceEvent[]
): ChainVerificationResult {
  if (events.length === 0) {
    return {
      valid: true,
      eventsChecked: 0,
      firstBrokenIndex: null,
      firstBrokenEventId: null,
      expectedHash: null,
      actualHash: null,
    };
  }

  for (let i = 0; i < events.length; i++) {
    const event = events[i];
    const previousHash = i > 0 ? (events[i - 1].event_hash ?? null) : null;

    // Legacy events without event_hash are skipped in chain verification
    // but their presence is noted by the fact that we still check subsequent links
    if (!event.event_hash) {
      continue;
    }

    const expectedHash = computeEventHash(event, previousHash);
    if (event.event_hash !== expectedHash) {
      return {
        valid: false,
        eventsChecked: i + 1,
        firstBrokenIndex: i,
        firstBrokenEventId: event.event_id,
        expectedHash,
        actualHash: event.event_hash,
      };
    }

    // Also verify previous_hash linkage (for i > 0)
    if (i > 0) {
      const expectedPreviousHash = events[i - 1].event_hash ?? null;
      const actualPreviousHash = event.previous_hash ?? null;
      if (actualPreviousHash !== expectedPreviousHash) {
        return {
          valid: false,
          eventsChecked: i + 1,
          firstBrokenIndex: i,
          firstBrokenEventId: event.event_id,
          expectedHash: expectedPreviousHash,
          actualHash: actualPreviousHash,
        };
      }
    }
  }

  return {
    valid: true,
    eventsChecked: events.length,
    firstBrokenIndex: null,
    firstBrokenEventId: null,
    expectedHash: null,
    actualHash: null,
  };
}
