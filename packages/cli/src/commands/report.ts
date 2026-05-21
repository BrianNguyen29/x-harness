import { Command } from "commander";
import * as path from "node:path";
import { readTrace } from "../core/trace.js";

interface ReportOptions {
  traceDir?: string;
}

export function reportCommand(): Command {
  return new Command("report")
    .description("Summarize trace events")
    .option("--trace-dir <dir>", "Trace directory", ".claimgate/traces")
    .action(async (opts: ReportOptions) => {
      const events = await readTrace(path.resolve(opts.traceDir ?? ".claimgate/traces"));

      const total = events.length;
      const accepted = events.filter((e) => e.acceptance_status === "accepted").length;
      const withheld = events.filter((e) => e.acceptance_status === "withheld").length;
      const byOutcome: Record<string, number> = {};
      for (const e of events) {
        const o = String(e.outcome ?? "unknown");
        byOutcome[o] = (byOutcome[o] ?? 0) + 1;
      }

      const report = {
        total_events: total,
        accepted,
        withheld,
        by_outcome: byOutcome,
        latest: events.length > 0 ? events[events.length - 1] : null,
      };

      console.log(JSON.stringify(report, null, 2));
    });
}
