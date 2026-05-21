import { Command } from "commander";
import { appendTrace } from "../core/trace.js";

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
    .option("--outcome <outcome>", "Outcome: success, failed, blocked, skipped, timeout, error", "success")
    .option("--acceptance-status <status>", "acceptance or withheld", "accepted")
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
      await appendTrace(event);
      console.log("trace event appended");
    });

  return cmd;
}
