import { Command } from "commander";
import { generatePlaybook, renderPlaybookMarkdown } from "../core/recovery.js";

interface RecoverySuggestOptions {
  errors?: string;
  outcome?: string;
  json?: boolean;
}

export function recoveryCommand(): Command {
  const cmd = new Command("recovery").description(
    "Recovery playbook and routing utilities"
  );

  cmd
    .command("suggest")
    .description(
      "Generate a deterministic recovery playbook candidate from errors"
    )
    .option("--errors <errors>", "Semicolon-separated error messages", "")
    .option(
      "--outcome <outcome>",
      "Outcome: success, failed, blocked, skipped, timeout, error",
      "failed"
    )
    .option("--json", "Output JSON instead of Markdown", false)
    .action((opts: RecoverySuggestOptions) => {
      const errors = opts.errors
        ? opts.errors
            .split(";")
            .map((e) => e.trim())
            .filter(Boolean)
        : [];
      const suggestions = generatePlaybook(errors, opts.outcome ?? "failed");

      if (opts.json) {
        console.log(JSON.stringify({ suggestions }, null, 2));
        return;
      }

      console.log(renderPlaybookMarkdown(suggestions));
    });

  return cmd;
}
