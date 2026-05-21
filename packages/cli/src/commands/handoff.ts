import { Command } from "commander";

interface HandoffOptions {
  title?: string;
  task?: string;
  tier?: string;
}

export function handoffCommand(): Command {
  const cmd = new Command("handoff").description("Generate handoff templates");

  for (const tier of ["light", "standard", "deep"] as const) {
    cmd
      .command(tier)
      .description(`Generate ${tier} handoff template`)
      .option("--title <title>", "Task title")
      .option("--task <description>", "Task description")
      .action((opts: HandoffOptions) => {
        const title = opts.title ?? "Untitled";
        const task = opts.task ?? "Describe the task here.";
        const output = `# SUBAGENT_TASK ${tier}

## Task: ${title}

${task}

## Constraints
- Do not self-admit completion.
- Return a completion candidate with result, evidence, verification, confidence, and handoff.

## Return format
Align with ClaimGate return schema:
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

  return cmd;
}
