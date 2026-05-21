import { Command } from "commander";
import * as path from "node:path";
import { readTrace } from "../core/trace.js";

interface ReportOptions {
  traceDir?: string;
  json?: boolean;
}

export function reportCommand(): Command {
  return new Command("report")
    .description("Summarize trace events")
    .option("--trace-dir <dir>", "Trace directory", ".x-harness/traces")
    .option("--json", "Output JSON instead of Markdown", false)
    .action(async (opts: ReportOptions) => {
      const events = await readTrace(path.resolve(opts.traceDir ?? ".x-harness/traces"));

      const total = events.length;
      const accepted = events.filter((e) => e.acceptance_status === "accepted").length;
      const withheld = events.filter((e) => e.acceptance_status === "withheld").length;
      const blocked = events.filter((e) => e.outcome === "blocked").length;
      const byOutcome: Record<string, number> = {};
      for (const e of events) {
        const o = String(e.outcome ?? "unknown");
        byOutcome[o] = (byOutcome[o] ?? 0) + 1;
      }

      if (opts.json) {
        const report = {
          total_events: total,
          accepted,
          withheld,
          by_outcome: byOutcome,
          latest: events.length > 0 ? events[events.length - 1] : null,
        };
        console.log(JSON.stringify(report, null, 2));
        return;
      }

      // Markdown output
      console.log("# x-harness Report");
      console.log("");
      console.log("## Installed mode");
      console.log("CLI-only (no daemon / no database / no MCP)");
      console.log("");
      console.log("## Templates");
      console.log("- COMPLETION_CARD.md");
      console.log("- SUBAGENT_TASK_light.md");
      console.log("- SUBAGENT_TASK_standard.md");
      console.log("- SUBAGENT_TASK_deep.md");
      console.log("- VERIFY_REPORT.md");
      console.log("");
      console.log("## Completion card");
      if (total === 0) {
        console.log("No completion cards found in trace.");
      } else {
        console.log(`${total} card(s) in trace.`);
      }
      console.log("");
      console.log("## Verification summary");
      if (total === 0) {
        console.log("No verification events recorded.");
      } else {
        for (const [outcome, count] of Object.entries(byOutcome)) {
          console.log(`- ${outcome}: ${count}/${total}`);
        }
      }
      console.log("");
      console.log("## Blocked items");
      if (blocked === 0) {
        console.log("None.");
      } else {
        console.log(`${blocked}/${total} blocked`);
      }
      console.log("");
      console.log("## Denominator warning");
      console.log("> Verify-event success must not be interpreted as task-level success without denominator review."
      );
      if (total === 0) {
        console.log("Denominator: NOT_COMPUTABLE (no events)");
      } else {
        console.log(`- accepted: ${accepted}/${total} cards`);
        console.log(`- blocked: ${blocked}/${total} cards`);
        console.log(`- withheld: ${withheld}/${total} cards`);
      }
    });
}
