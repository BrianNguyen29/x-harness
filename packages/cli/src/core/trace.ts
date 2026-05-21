import fs from "fs-extra";
import * as path from "node:path";

export interface TraceEvent {
  event_id: string;
  event_type: string;
  [key: string]: unknown;
}

export async function appendTrace(event: TraceEvent, traceDir = ".x-harness/traces"): Promise<void> {
  await fs.ensureDir(traceDir);
  const filePath = path.join(traceDir, "events.jsonl");
  await fs.appendFile(filePath, JSON.stringify(event) + "\n");
}

export async function readTrace(traceDir = ".x-harness/traces"): Promise<TraceEvent[]> {
  const filePath = path.join(traceDir, "events.jsonl");
  if (!(await fs.pathExists(filePath))) return [];
  const content = await fs.readFile(filePath, "utf-8");
  return content
    .split("\n")
    .map((line: string) => line.trim())
    .filter((line: string) => line.length > 0)
    .map((line: string) => JSON.parse(line) as TraceEvent);
}
