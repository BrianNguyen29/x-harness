#!/usr/bin/env node
import { Command } from "commander";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
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
import { learnCommand } from "./commands/learn.js";
import { quickCommand } from "./commands/quick.js";
import { runCommand } from "./commands/run.js";
import { ciCommand } from "./commands/ci.js";
import { CliError, handleCliError } from "./core/exit.js";
import {
  type Lang,
  getLang,
  withoutLang,
  resolveLang,
  startHereTitle,
  categoryGettingStarted,
  discoverMore,
  newToXHarness,
  usageLabel,
  forCommandSpecificHelp,
  advancedLabel,
  globalOptionsLabel,
  showHelpText,
  showAllCommandsText,
  showMaturityLabelsText,
  showVersionText,
  beginnerActionsTitle,
  invokeUsingEither,
  installedCLIText,
  localSourceText,
  forMoreInfo,
  getBeginnerCommandDesc,
  discoverHelpDesc,
  discoverHelpAllDesc,
  discoverHelpMaturityDesc,
  actionHeader,
  descriptionHeader,
} from "./i18n.js";

const packageJson = JSON.parse(
  readFileSync(
    join(dirname(fileURLToPath(import.meta.url)), "..", "package.json"),
    "utf8"
  )
) as { version?: string };
const CLI_VERSION = packageJson.version ?? "dev";

interface CliCommandMetadata {
  name: string;
  description: string;
  primary?: boolean;
  onboarding?: boolean;
  onboarding_order?: number;
  maturity: "stable" | "beta" | "experimental" | "skeletal";
}

function loadCliCommandMetadata(): CliCommandMetadata[] {
  const here = dirname(fileURLToPath(import.meta.url));
  const candidates = [
    join(here, "..", "..", "..", "internal", "cli", "commands.json"),
    join(here, "..", "internal", "cli", "commands.json"),
  ];
  for (const candidate of candidates) {
    if (!existsSync(candidate)) continue;
    return JSON.parse(readFileSync(candidate, "utf8")) as CliCommandMetadata[];
  }
  throw new Error("x-harness CLI command registry not found");
}

const CLI_COMMANDS = loadCliCommandMetadata();
const commandDescriptions: Record<string, string> = Object.fromEntries(
  CLI_COMMANDS.map((command) => [command.name, command.description])
);

const program = new Command();
program
  .name("xh")
  .description("A lightweight verify-gated harness for AI-agent workflows")
  .version(CLI_VERSION)
  .helpOption(false)
  .option("--lang <code>", "Language", "en");

const beginnerCommands = new Set(
  CLI_COMMANDS.filter((command) => command.onboarding).map(
    (command) => command.name
  )
);

function onboardingCommands(): CliCommandMetadata[] {
  return CLI_COMMANDS.filter((command) => command.onboarding).sort((a, b) => {
    const left = a.onboarding_order ?? Number.MAX_SAFE_INTEGER;
    const right = b.onboarding_order ?? Number.MAX_SAFE_INTEGER;
    return left === right ? a.name.localeCompare(b.name) : left - right;
  });
}

function commandDescription(name: string, lang: Lang): string {
  return getBeginnerCommandDesc(name, lang) || commandDescriptions[name] || "";
}

function hideAdvancedCommands() {
  for (const cmd of program.commands) {
    if (!beginnerCommands.has(cmd.name())) {
      (cmd as unknown as { _hidden: boolean })._hidden = true;
    }
  }
}

function printStartHere(lang: Lang) {
  console.log(`xh ${CLI_VERSION}`);
  console.log("");
  console.log("A lightweight verify-gated harness for AI-agent workflows.");
  console.log("");
  console.log(startHereTitle(lang));
  console.log("");
  console.log(categoryGettingStarted(lang));
  for (const command of onboardingCommands()) {
    console.log(
      `  xh ${command.name.padEnd(14)} ${commandDescription(command.name, lang)}`
    );
  }
  console.log("");
  console.log(discoverMore(lang));
  console.log(`  xh --help            ${discoverHelpDesc(lang)}`);
  console.log(`  xh --help-all        ${discoverHelpAllDesc(lang)}`);
  console.log(`  xh --help-maturity   ${discoverHelpMaturityDesc(lang)}`);
  console.log("");
  console.log(newToXHarness(lang));
}

function printHelp(lang: Lang) {
  console.log(`xh ${CLI_VERSION}`);
  console.log("");
  console.log("A lightweight verify-gated harness for AI-agent workflows.");
  console.log("");
  console.log(usageLabel(lang));
  console.log("  xh <command> [options]");
  console.log("");
  console.log(categoryGettingStarted(lang));
  for (const command of onboardingCommands()) {
    console.log(
      `  xh ${command.name.padEnd(14)} ${commandDescription(command.name, lang)}`
    );
  }
  console.log("");
  console.log(forCommandSpecificHelp(lang));
  console.log("  xh <command> --help");
  console.log("");
  console.log(advancedLabel(lang));
  console.log(`  xh --help-all          ${showAllCommandsText(lang)}`);
  console.log(`  xh --help-maturity     ${showMaturityLabelsText(lang)}`);
  console.log("");
  console.log(globalOptionsLabel(lang));
  console.log(`  -h, --help          ${showHelpText(lang)}`);
  console.log(`  --help-all          ${showAllCommandsText(lang)}`);
  console.log(`  --help-maturity     ${showMaturityLabelsText(lang)}`);
  console.log(`  -v, --version       ${showVersionText(lang)}`);
}

function printHelpMaturity() {
  console.log(`xh ${CLI_VERSION}`);
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

  const groups: Record<string, CliCommandMetadata[]> = {
    stable: [],
    beta: [],
    experimental: [],
    skeletal: [],
  };

  for (const command of CLI_COMMANDS) {
    groups[command.maturity].push(command);
  }

  for (const mat of ["stable", "beta", "experimental", "skeletal"]) {
    if (groups[mat].length > 0) {
      console.log(`${mat}:`);
      for (const command of groups[mat].sort((a, b) =>
        a.name.localeCompare(b.name)
      )) {
        console.log(`  ${command.name.padEnd(12)} ${command.description}`);
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
program.addCommand(learnCommand());
program.addCommand(quickCommand());
program.addCommand(runCommand());
program.addCommand(ciCommand());

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
actions.description("List beginner-friendly actions");
actions.option("--lang <code>", "Language", "en");
actions.action((opts: { lang: string }) => {
  const lang: Lang = resolveLang(opts, program.opts());
  console.log(beginnerActionsTitle(lang));
  console.log("");
  console.log(invokeUsingEither(lang));
  console.log(`  - ${installedCLIText(lang)}  xh <action>`);
  console.log(
    `  - ${localSourceText(lang)}   node packages/cli/dist/index.js <action>`
  );
  console.log("");
  console.log(`## ${categoryGettingStarted(lang)}`);
  console.log(`| ${actionHeader(lang)} | ${descriptionHeader(lang)} |`);
  console.log("| :-- | :-- |");
  for (const command of onboardingCommands()) {
    console.log(
      `| **${command.name}** | ${commandDescription(command.name, lang)} |`
    );
  }
  console.log("");
  console.log(forMoreInfo(lang));
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
const lang = getLang(args);
const cleanArgs = withoutLang(args);

if (cleanArgs.length === 0) {
  printStartHere(lang);
  process.exit(0);
}

if (
  cleanArgs.length === 1 &&
  (cleanArgs[0] === "--help" || cleanArgs[0] === "-h")
) {
  printHelp(lang);
  process.exit(0);
}

if (cleanArgs.includes("--help-all")) {
  program.commands.forEach(
    (cmd) => ((cmd as unknown as { _hidden: boolean })._hidden = false)
  );
  program.help();
}

if (cleanArgs.includes("--help-maturity")) {
  printHelpMaturity();
  process.exit(0);
}

program.parseAsync(process.argv).catch(handleCliError);
