import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import {
  generatePlaybook,
  generatePlaybookFromTrace,
  renderPlaybookMarkdown,
} from "../core/recovery.js";
import { readJsonl } from "../core/schema.js";

interface RecoverySuggestOptions {
  errors?: string;
  outcome?: string;
  json?: boolean;
  from?: string;
  write?: boolean;
  force?: boolean;
}

export async function recoverySuggestAction(
  opts: RecoverySuggestOptions
): Promise<void> {
  let suggestions;

  if (opts.from) {
    const tracePath = path.resolve(opts.from);
    if (!(await fs.pathExists(tracePath))) {
      console.error(`Error: Trace file not found: ${tracePath}`);
      process.exit(2);
    }
    const events = await readJsonl(tracePath);
    suggestions = generatePlaybookFromTrace(events);
  } else {
    const errors = opts.errors
      ? opts.errors
          .split(";")
          .map((e) => e.trim())
          .filter(Boolean)
      : [];
    suggestions = generatePlaybook(errors, opts.outcome ?? "failed");
  }

  const output = opts.json
    ? JSON.stringify({ suggestions }, null, 2)
    : renderPlaybookMarkdown(suggestions);

  if (opts.write) {
    const candidatesDir = path.resolve(".x-harness/candidates");
    await fs.ensureDir(candidatesDir);
    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    const fileName = `playbook-suggestion-${timestamp}.md`;
    const filePath = path.join(candidatesDir, fileName);

    if ((await fs.pathExists(filePath)) && !opts.force) {
      console.error(
        `Error: Candidate file already exists: ${filePath}\nUse --force to overwrite.`
      );
      process.exit(2);
    }

    await fs.writeFile(filePath, output);
    console.log(`candidate written: ${filePath}`);
  } else {
    console.log(output);
  }
}

export function recoveryCommand(): Command {
  const cmd = new Command("recovery").description(
    "Recovery playbook and routing utilities"
  );

  cmd
    .command("suggest")
    .description(
      "Generate a deterministic recovery playbook candidate from errors or trace"
    )
    .option("--errors <errors>", "Semicolon-separated error messages", "")
    .option(
      "--outcome <outcome>",
      "Outcome: success, failed, blocked, skipped, timeout, error",
      "failed"
    )
    .option("--from <trace-file>", "Path to trace JSONL file to analyze")
    .option(
      "--write",
      "Write candidate playbook to .x-harness/candidates/",
      false
    )
    .option("--force", "Allow overwrite of existing candidate file", false)
    .option("--json", "Output JSON instead of Markdown", false)
    .action(recoverySuggestAction);

  return cmd;
}
