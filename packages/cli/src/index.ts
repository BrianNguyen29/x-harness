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
import { intakeCommand } from "./commands/intake.js";
import {
  governanceCommand,
  interventionCommand,
} from "./commands/governance.js";
import { predictionCommand } from "./commands/prediction.js";
import { benchmarkCommand } from "./commands/benchmark.js";
import { componentsCommand } from "./commands/components.js";
import { evidenceCommand } from "./commands/evidence.js";
import { episodeCommand } from "./commands/episode.js";
import { attributionCommand } from "./commands/attribution.js";
import { permissionsCommand } from "./commands/permissions.js";
import { evolveCommand } from "./commands/evolve.js";
import {
  frozenCommand,
  frozenExportCommand,
  frozenImportCommand,
} from "./commands/frozen.js";
import { federationCommand } from "./commands/federation.js";
import { approvalRiskCommand } from "./commands/approval-risk.js";
import { agentProfileCommand } from "./commands/agent-profile.js";
import { costCommand } from "./commands/cost.js";
import { profileCommand } from "./commands/profile.js";
import { decisionCommand } from "./commands/decision.js";
import { startCommand } from "./commands/start.js";
import { CliError, handleCliError } from "./core/exit.js";

const program = new Command();
program
  .name("xh")
  .description("A lightweight verify-gated harness for AI-agent workflows")
  .version("0.1.0");

const beginnerCommands = new Set([
  "check",
  "prepare",
  "recover",
  "doctor",
  "actions",
  "status",
  "reset",
  "init",
  "add",
  "start",
]);

const commandMaturity: Record<string, string> = {
  check: "stable",
  prepare: "stable",
  recover: "stable",
  doctor: "stable",
  actions: "beta",
  status: "stable",
  reset: "stable",
  init: "stable",
  add: "stable",
  start: "beta",
  verify: "stable",
  handoff: "stable",
  report: "stable",
  trace: "stable",
  clean: "stable",
  examples: "stable",
  context: "stable",
  recovery: "stable",
  packet: "beta",
  profile: "beta",
  intake: "experimental",
  governance: "experimental",
  intervention: "experimental",
  prediction: "experimental",
  benchmark: "stable",
  components: "experimental",
  evidence: "experimental",
  episode: "experimental",
  attribution: "experimental",
  permissions: "experimental",
  evolve: "experimental",
  frozen: "experimental",
  "frozen-export": "experimental",
  "frozen-import": "experimental",
  federation: "experimental",
  "approval-risk": "experimental",
  "agent-profile": "experimental",
  cost: "experimental",
  decision: "experimental",
};

function hideAdvancedCommands() {
  for (const cmd of program.commands) {
    if (!beginnerCommands.has(cmd.name())) {
      (cmd as unknown as { _hidden: boolean })._hidden = true;
    }
  }
}

function printStartHere() {
  console.log("xh 0.1.0");
  console.log("");
  console.log("A lightweight verify-gated harness for AI-agent workflows.");
  console.log("");
  console.log("Start here — a few commands to get you going:");
  console.log("");
  console.log(
    "  xh start           Guided onboarding: doctor, examples verify, init wizard, next steps"
  );
  console.log(
    "  xh check (verify)  Run read-only verification against a completion card"
  );
  console.log(
    "  xh prepare         Check if workspace is ready for agent task handoff"
  );
  console.log(
    "  xh recover         Get recovery playbook suggestions from errors or trace"
  );
  console.log(
    "  xh doctor          Validate workspace health and configuration"
  );
  console.log("  xh actions         Show this list of actions");
  console.log("  xh status          Show trace summary");
  console.log("  xh reset           Clean generated harness state");
  console.log("");
  console.log("Discover more:");
  console.log("  xh --help            Common commands and usage");
  console.log("  xh --help-all        All commands");
  console.log("  xh --help-maturity   Commands grouped by stability");
  console.log("");
  console.log("New to x-harness? See docs/GETTING_STARTED.md");
}

function printHelpMaturity() {
  console.log("xh 0.1.0");
  console.log("");
  console.log("A lightweight verify-gated harness for AI-agent workflows.");
  console.log("");
  console.log("Maturity labels:");
  console.log("  stable       Core command; tested and relied on in CI");
  console.log("  beta         Functional but may change; feedback welcome");
  console.log("  experimental New or advanced; semantics may shift");
  console.log("  skeletal     Declared but not yet implemented");
  console.log("");
  console.log("Usage:");
  console.log("  xh <command> [options]");
  console.log("");

  const groups: Record<string, string[]> = {
    stable: [],
    beta: [],
    experimental: [],
    skeletal: [],
  };

  for (const cmd of program.commands) {
    const mat = commandMaturity[cmd.name()] || "experimental";
    const names = [cmd.name(), ...cmd.aliases()];
    for (const n of names) {
      if (!groups[mat].includes(n)) {
        groups[mat].push(n);
      }
    }
  }

  for (const mat of ["stable", "beta", "experimental", "skeletal"]) {
    if (groups[mat].length > 0) {
      console.log(`${mat}:`);
      for (const name of groups[mat].sort()) {
        console.log(`  ${name}`);
      }
      console.log("");
    }
  }

  console.log("Global options:");
  console.log("  -h, --help          Show help");
  console.log("  --help-all          Show all commands");
  console.log(
    "  --help-maturity     Show help with maturity labels for all commands"
  );
  console.log("  -v, --version       Show version");
}

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
program.addCommand(intakeCommand());
program.addCommand(governanceCommand());
program.addCommand(interventionCommand());
program.addCommand(predictionCommand());
program.addCommand(benchmarkCommand());
program.addCommand(componentsCommand());
program.addCommand(evidenceCommand());
program.addCommand(episodeCommand());
program.addCommand(attributionCommand());
program.addCommand(permissionsCommand());
program.addCommand(evolveCommand());
program.addCommand(frozenCommand());
program.addCommand(frozenExportCommand());
program.addCommand(frozenImportCommand());
program.addCommand(federationCommand());
program.addCommand(approvalRiskCommand());
program.addCommand(agentProfileCommand());
program.addCommand(costCommand());
program.addCommand(profileCommand());
program.addCommand(decisionCommand());
program.addCommand(startCommand());

// Commands with beginner-friendly aliases
const verify = verifyCommand();
verify.name("check");
verify.alias("verify");
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
report.name("status");
report.alias("report");
program.addCommand(report);

// actions lists all beginner-friendly actions
const actions = new Command("actions");
actions.description("List all beginner-friendly actions");
actions.action(() => {
  console.log("# xh Beginner Actions");
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
    "| **start**    | Guided onboarding: doctor, examples verify, init wizard, next steps |"
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
    console.log("xh reset requires --confirm for safety.");
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

hideAdvancedCommands();

program.addHelpText(
  "after",
  `
Common commands shown above. For all commands:
  xh --help-all          Show all commands
  xh --help-maturity     Show commands grouped by maturity

For command-specific help: xh <command> --help
`
);

const args = process.argv.slice(2);

if (args.length === 0) {
  printStartHere();
  process.exit(0);
}

if (args.includes("--help-all")) {
  program.commands.forEach(
    (cmd) => ((cmd as unknown as { _hidden: boolean })._hidden = false)
  );
  program.help();
}

if (args.includes("--help-maturity")) {
  printHelpMaturity();
  process.exit(0);
}

program.parseAsync(process.argv).catch(handleCliError);
