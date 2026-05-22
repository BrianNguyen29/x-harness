import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { getCompactContextHeader } from "../core/context.js";

interface HandoffOptions {
  title?: string;
  task?: string;
  tier?: string;
  context?: boolean;
}

function isNonInteractive(): boolean {
  return (
    !process.stdin.isTTY ||
    process.env.CI === "true" ||
    process.env.NONINTERACTIVE === "true"
  );
}

async function askQuestion(question: string): Promise<boolean> {
  return new Promise((resolve) => {
    process.stdout.write(`${question} [y/N]: `);
    process.stdin.once("data", (data) => {
      const answer = data.toString().trim().toLowerCase();
      resolve(answer === "y" || answer === "yes");
    });
  });
}

async function checkReadiness(
  interactive: boolean,
  root: string
): Promise<{
  ready: boolean;
  checks: { name: string; passed: boolean; note: string }[];
}> {
  const checks: { name: string; passed: boolean; note: string }[] = [];

  // Check AGENTS.md
  const agentsPath = path.join(root, "AGENTS.md");
  const agentsExists = await fs.pathExists(agentsPath);
  checks.push({
    name: "agents_md_present",
    passed: agentsExists,
    note: agentsExists ? "AGENTS.md found" : "AGENTS.md missing",
  });

  // Check policies
  const policyPath = path.join(root, "policies", "admission.yaml");
  const policyExists = await fs.pathExists(policyPath);
  checks.push({
    name: "admission_policy_present",
    passed: policyExists,
    note: policyExists
      ? "policies/admission.yaml found"
      : "policies/admission.yaml missing",
  });

  // Check templates
  const templatesDir = path.join(root, "templates");
  const templatesExist = await fs.pathExists(templatesDir);
  checks.push({
    name: "templates_present",
    passed: templatesExist,
    note: templatesExist
      ? "templates/ directory found"
      : "templates/ directory missing",
  });

  // Check completion card template
  const completionCardPath = path.join(root, "templates", "COMPLETION_CARD.md");
  const completionCardExists = await fs.pathExists(completionCardPath);
  checks.push({
    name: "completion_card_template_present",
    passed: completionCardExists,
    note: completionCardExists
      ? "templates/COMPLETION_CARD.md found"
      : "templates/COMPLETION_CARD.md missing",
  });

  const allPassed = checks.every((c) => c.passed);

  if (interactive && !isNonInteractive()) {
    if (!allPassed) {
      console.log("Readiness checks failed:");
      for (const c of checks) {
        console.log(`  [${c.passed ? "PASS" : "FAIL"}] ${c.name}: ${c.note}`);
      }
      return { ready: false, checks };
    }

    console.log("Readiness checks passed. Answer the following prompts:");
    const scopeClear = await askQuestion(
      "Is the task scope clear and bounded?"
    );
    if (!scopeClear) {
      checks.push({
        name: "scope_clear",
        passed: false,
        note: "User indicated scope is not clear",
      });
      return { ready: false, checks };
    }

    const evidencePlan = await askQuestion(
      "Do you have an evidence plan (tests, lint, build)?"
    );
    if (!evidencePlan) {
      checks.push({
        name: "evidence_plan",
        passed: false,
        note: "User indicated no evidence plan",
      });
      return { ready: false, checks };
    }

    checks.push({
      name: "scope_clear",
      passed: true,
      note: "User confirmed scope is clear",
    });
    checks.push({
      name: "evidence_plan",
      passed: true,
      note: "User confirmed evidence plan exists",
    });
    return { ready: true, checks };
  }

  // Non-interactive mode: just report
  if (!allPassed) {
    return { ready: false, checks };
  }

  // In non-interactive mode, we can't ask questions, so we assume the
  // structural checks are sufficient and mark readiness as advisory.
  checks.push({
    name: "interactive_prompts",
    passed: true,
    note: "Non-interactive mode: skipping readiness prompts",
  });
  return { ready: true, checks };
}

export function handoffCommand(): Command {
  const cmd = new Command("handoff").description("Generate handoff templates");

  for (const tier of ["light", "standard", "deep"] as const) {
    cmd
      .command(tier)
      .description(`Generate ${tier} handoff template`)
      .option("--title <title>", "Task title")
      .option("--task <description>", "Task description")
      .option("--no-context", "Omit the compact context header")
      .action((opts: HandoffOptions) => {
        const title = opts.title ?? "Untitled";
        const task = opts.task ?? "Describe the task here.";
        const contextHeader =
          opts.context === false ? "" : getCompactContextHeader();
        const output = `# SUBAGENT_TASK ${tier}

${contextHeader}## Task: ${title}

${task}

## Constraints
- Do not self-admit completion.
- Return a completion candidate with result, evidence, verification, confidence, and handoff.

## Return format
Align with x-harness return schema:
\`\`\`yaml
result:
  summary: <one-line outcome>
  fix_status: <fixed|not_fixed|partial>
  key_findings: []
  recommendations: []
evidence:
  files_changed: []
  commands_ran: []
verification:
  status: <passed|failed|skipped|blocked>
  checks: []
confidence: <LOW|MED|HIGH>
handoff:
  next_action: <next step> (owner: <agent|user>)
\`\`\`
`;
        console.log(output);
      });
  }

  cmd
    .command("readiness")
    .description("Check handoff readiness with optional interactive prompts")
    .option("--interactive", "Enable interactive readiness prompts", false)
    .option("--json", "Output JSON instead of text", false)
    .option("--root <path>", "Repository root", process.cwd())
    .action(
      async (opts: {
        interactive?: boolean;
        json?: boolean;
        root?: string;
      }) => {
        const root = path.resolve(opts.root ?? process.cwd());
        const { ready, checks } = await checkReadiness(
          opts.interactive ?? false,
          root
        );

        if (opts.json) {
          console.log(JSON.stringify({ ready, checks }, null, 2));
        } else {
          console.log(`handoff readiness: ${ready ? "READY" : "NOT READY"}`);
          for (const c of checks) {
            console.log(
              `  [${c.passed ? "PASS" : "FAIL"}] ${c.name}: ${c.note}`
            );
          }
        }

        process.exit(ready ? 0 : 1);
      }
    );

  return cmd;
}
