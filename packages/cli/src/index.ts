#!/usr/bin/env node
import { Command } from "commander";
import { initCommand } from "./commands/init.js";
import { addCommand } from "./commands/add.js";
import { verifyCommand } from "./commands/verify.js";
import { doctorCommand } from "./commands/doctor.js";
import { reportCommand } from "./commands/report.js";
import { traceCommand } from "./commands/trace.js";
import { handoffCommand, checkReadinessAction } from "./commands/handoff.js";
import { cleanCommand, cleanTmpAction } from "./commands/clean.js";
import { examplesCommand } from "./commands/examples.js";
import { contextCommand } from "./commands/context.js";
import { recoveryCommand, recoverySuggestAction } from "./commands/recovery.js";
import { packetCommand } from "./commands/packet.js";
import { CliError, handleCliError } from "./core/exit.js";

const program = new Command();
program
  .name("x-harness")
  .description("A lightweight verify-gated harness for AI-agent workflows")
  .version("0.1.0");

// Base commands (no aliases)
program.addCommand(initCommand());
program.addCommand(addCommand());
program.addCommand(traceCommand());
// report is added below with status alias
program.addCommand(cleanCommand());
program.addCommand(examplesCommand());
program.addCommand(contextCommand());
program.addCommand(packetCommand());
program.addCommand(doctorCommand());

// Commands with beginner-friendly aliases
const verify = verifyCommand();
verify.alias("check");
program.addCommand(verify);

const handoff = handoffCommand();
program.addCommand(handoff);

// prepare is an alias for handoff readiness
const prepare = new Command("prepare");
prepare.description(
  "Check handoff readiness with optional interactive prompts"
);
prepare.option("--interactive", "Enable interactive readiness prompts", false);
prepare.option(
  "--non-interactive",
  "Explicitly skip interactive prompts",
  false
);
prepare.option("--json", "Output JSON instead of text", false);
prepare.option("--root <path>", "Repository root", process.cwd());
prepare.action(checkReadinessAction);
program.addCommand(prepare);

const recovery = recoveryCommand();
program.addCommand(recovery);

// recover is an alias for recovery suggest
const recover = new Command("recover");
recover.description(
  "Generate a deterministic recovery playbook candidate from errors or trace"
);
recover.option("--errors <errors>", "Semicolon-separated error messages", "");
recover.option(
  "--outcome <outcome>",
  "Outcome: success, failed, blocked, skipped, timeout, error",
  "failed"
);
recover.option("--from <trace-file>", "Path to trace JSONL file to analyze");
recover.option(
  "--write",
  "Write candidate playbook to .x-harness/candidates/",
  false
);
recover.option("--force", "Allow overwrite of existing candidate file", false);
recover.option("--json", "Output JSON instead of Markdown", false);
recover.action(recoverySuggestAction);
program.addCommand(recover);

// status is an alias for report (shows trace summary by default)
const report = reportCommand();
report.alias("status");
program.addCommand(report);

// actions lists all beginner-friendly actions
const actions = new Command("actions");
actions.description("List all beginner-friendly actions");
actions.action(() => {
  console.log("# x-harness Beginner Actions");
  console.log("");
  console.log("Invoke using either:");
  console.log("  - Installed CLI:  xh <action>");
  console.log("  - Local source:   node packages/cli/dist/index.js <action>");
  console.log("");
  console.log(
    "| Action       | Description                                              |"
  );
  console.log(
    "| :----------- | :------------------------------------------------------- |"
  );
  console.log(
    "| **prepare**  | Check if workspace is ready for agent task handoff        |"
  );
  console.log(
    "| **check**    | Run read-only verification against a completion card       |"
  );
  console.log(
    "| **recover**  | Get recovery playbook suggestions from errors or trace     |"
  );
  console.log(
    "| **doctor**   | Validate workspace health and configuration               |"
  );
  console.log(
    "| **actions** | Show this list of actions                                |"
  );
  console.log(
    "| **status**  | Show trace summary (alias for report without --metrics)  |"
  );
  console.log(
    "| **reset**    | Clean generated harness state (requires --confirm)       |"
  );
  console.log("");
  console.log("For more info: xh <command> --help");
});
program.addCommand(actions);

// reset cleans generated harness state safely (requires --confirm)
// reset --confirm invokes cleanTmpAction which is the same as clean --tmp --force
const reset = new Command("reset");
reset.description(
  "Clean generated harness state (tmp, cache). Requires --confirm for safety."
);
reset.option(
  "--confirm",
  "Confirm reset (deletes .x-harness/tmp and .x-harness/cache)",
  false
);
reset.action(async (opts: { confirm?: boolean }) => {
  if (!opts.confirm) {
    console.log("x-harness reset requires --confirm for safety.");
    console.log("");
    console.log("To reset harness state:");
    console.log("  xh reset --confirm");
    console.log("");
    console.log("This will delete:");
    console.log("  - .x-harness/tmp/");
    console.log("  - .x-harness/cache/");
    throw new CliError("reset aborted: --confirm required", 1);
  }

  // Delegate to clean --tmp --force behavior
  await cleanTmpAction();
});
program.addCommand(reset);

program.parseAsync(process.argv).catch(handleCliError);
