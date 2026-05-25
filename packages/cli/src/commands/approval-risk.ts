import { Command } from "commander";
import { evaluateApprovalRisk } from "../core/approval-risk.js";
import { CliError } from "../core/exit.js";

interface ApprovalRiskOptions {
  root?: string;
  card?: string;
  json?: boolean;
}

export function approvalRiskCommand(): Command {
  const cmd = new Command("approval-risk").description(
    "Evaluate advisory approval risk without personal scoring"
  );

  cmd
    .command("evaluate")
    .description("Evaluate approval risk for a completion card")
    .requiredOption("--card <path>", "Completion card path")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: ApprovalRiskOptions) => {
      const report = await evaluateApprovalRisk({
        root: opts.root ?? process.cwd(),
        cardPath: opts.card as string,
      });
      if (opts.json) {
        console.log(JSON.stringify(report, null, 2));
      } else {
        console.log("# x-harness Approval Risk");
        console.log(`- task_id: ${report.task_id}`);
        console.log(`- risk_class: ${report.risk_class}`);
        console.log(`- score: ${report.score}`);
        console.log(`- required_approvals: ${report.required_approvals}`);
        console.log(`- admission_authority: ${report.admission_authority}`);
      }
    });

  cmd
    .command("check")
    .description("Alias for evaluate")
    .requiredOption("--card <path>", "Completion card path")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: ApprovalRiskOptions) => {
      if (!opts.card) throw new CliError("--card is required", 2);
      const report = await evaluateApprovalRisk({
        root: opts.root ?? process.cwd(),
        cardPath: opts.card,
      });
      if (opts.json) console.log(JSON.stringify(report, null, 2));
      else console.log(`approval risk: ${report.risk_class}`);
    });

  return cmd;
}
