import { Command } from "commander";
import * as path from "node:path";
import {
  appendTrace,
  readTrace,
  readTraceFromFile,
  verifyTraceChain,
} from "../core/trace.js";

interface TraceAddOptions {
  outcome?: string;
  acceptanceStatus?: string;
  taskId?: string;
  tier?: string;
  claimId?: string;
  evidenceId?: string;
}

export function traceCommand(): Command {
  const cmd = new Command("trace").description("Trace verify events");

  cmd
    .command("add")
    .description("Append a verify event to the trace log")
    .option(
      "--outcome <outcome>",
      "Outcome: success, failed, blocked, skipped, timeout, error",
      "success"
    )
    .option(
      "--acceptance-status <status>",
      "acceptance or withheld",
      "accepted"
    )
    .option("--task-id <id>", "Task ID", "TASK-UNKNOWN")
    .option("--tier <tier>", "Tier: light, standard, deep", "standard")
    .option("--claim-id <id>", "Claim ID")
    .option("--evidence-id <id>", "Evidence ID")
    .action(async (opts: TraceAddOptions) => {
      const event = {
        event_id: `VE-${Date.now()}`,
        event_type: "verify_completed",
        task_id: opts.taskId,
        tier: opts.tier,
        claim_id: opts.claimId ?? null,
        evidence_id: opts.evidenceId ?? null,
        verifier: "x-harness",
        verifier_mode: "read_only",
        outcome: opts.outcome,
        acceptance_status: opts.acceptanceStatus,
        blocking_predicate: null,
        blocked_reason_class: null,
        next_owner: null,
        next_action: null,
        created_at: new Date().toISOString(),
      };
      const enriched = await appendTrace(event);
      console.log("trace event appended");
      console.log(`event_id: ${enriched.event_id}`);
      console.log(`event_hash: ${enriched.event_hash}`);
      if (enriched.previous_hash) {
        console.log(`previous_hash: ${enriched.previous_hash}`);
      }
    });

  cmd
    .command("verify-chain")
    .description("Verify the integrity of the trace hash chain")
    .option("--trace-dir <dir>", "Trace directory", ".x-harness/traces")
    .option(
      "--from <file>",
      "Read trace events from a specific JSONL file path"
    )
    .action(async (opts: { traceDir?: string; from?: string }) => {
      let events: import("../core/trace.js").TraceEvent[];
      if (opts.from) {
        events = await readTraceFromFile(path.resolve(opts.from));
      } else {
        const traceDir = path.resolve(opts.traceDir ?? ".x-harness/traces");
        events = await readTrace(traceDir);
      }
      const result = verifyTraceChain(events);

      if (result.valid) {
        console.log(`chain valid: ${result.eventsChecked} event(s) checked`);
        process.exit(0);
      } else {
        console.error(
          `chain broken at index ${result.firstBrokenIndex} (event_id: ${result.firstBrokenEventId})`
        );
        console.error(`expected hash: ${result.expectedHash}`);
        console.error(`actual hash:   ${result.actualHash}`);
        process.exit(1);
      }
    });

  return cmd;
}
