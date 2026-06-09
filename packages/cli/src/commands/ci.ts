import { Command } from "commander";
import { executeRun } from "./run.js";

export function ciCommand(): Command {
  return new Command("ci")
    .description("Run the built-in CI workflow")
    .option("--dry-run", "Print planned steps without executing", false)
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: { dryRun: boolean; json: boolean }) => {
      await executeRun("builtin:ci", {
        list: false,
        dryRun: opts.dryRun,
        json: opts.json,
      });
    });
}
