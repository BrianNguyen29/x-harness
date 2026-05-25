import { Command } from "commander";
import {
  evaluateCostBudget,
  readCostBudgetReport,
} from "../core/cost-budget.js";
import { CliError } from "../core/exit.js";

interface CostOptions {
  root?: string;
  actualUsd?: string;
  inputTokens?: string;
  outputTokens?: string;
  enforce?: boolean;
  from?: string;
  json?: boolean;
}

function parseNumber(value: string | undefined, label: string): number {
  if (!value || !/^\d+(?:\.\d+)?$/.test(value)) {
    throw new CliError(`${label} must be a non-negative number`, 2);
  }
  return Number(value);
}

function parseInteger(value: string | undefined, label: string): number {
  if (!value || !/^\d+$/.test(value)) {
    throw new CliError(`${label} must be a non-negative integer`, 2);
  }
  return Number.parseInt(value, 10);
}

export function costCommand(): Command {
  const cmd = new Command("cost").description(
    "Evaluate advisory cost and token budgets"
  );

  cmd
    .command("check")
    .description("Check observed cost metadata against the local policy")
    .requiredOption("--actual-usd <amount>", "Observed spend")
    .requiredOption("--input-tokens <count>", "Observed input token count")
    .requiredOption("--output-tokens <count>", "Observed output token count")
    .option("--root <path>", "Repository root", process.cwd())
    .option(
      "--enforce",
      "Exit non-zero only when policy enables cost enforcement",
      false
    )
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: CostOptions) => {
      const report = await evaluateCostBudget({
        root: opts.root ?? process.cwd(),
        actualUsd: parseNumber(opts.actualUsd, "--actual-usd"),
        inputTokens: parseInteger(opts.inputTokens, "--input-tokens"),
        outputTokens: parseInteger(opts.outputTokens, "--output-tokens"),
        enforce: Boolean(opts.enforce),
      });
      if (opts.json) console.log(JSON.stringify(report, null, 2));
      else {
        console.log("# x-harness Cost Budget");
        console.log(`- status: ${report.status}`);
        console.log(`- over_budget: ${report.over_budget}`);
        console.log(`- enforcement_enabled: ${report.enforcement_enabled}`);
      }
      if (report.over_budget && report.enforcement_enabled) {
        throw new CliError("cost budget exceeded", 1);
      }
    });

  cmd
    .command("report")
    .description("Read and validate a cost budget report")
    .requiredOption("--from <path>", "Report JSON/YAML path")
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: CostOptions) => {
      const report = await readCostBudgetReport(opts.from as string);
      if (opts.json) console.log(JSON.stringify(report, null, 2));
      else console.log(`cost budget: ${report.status}`);
    });

  return cmd;
}
