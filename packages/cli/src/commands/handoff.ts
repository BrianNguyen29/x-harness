import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { getCompactContextHeader } from "../core/context.js";
import { renderFixStatusGuidance } from "../core/contract.js";

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

async function askQuestion(
  question: string,
  timeoutMs = 30000
): Promise<boolean> {
  return new Promise((resolve) => {
    const timeout = setTimeout(() => {
      process.stdout.write(" (timeout - assuming no)\n");
      resolve(false);
    }, timeoutMs);

    const cleanup = () => {
      clearTimeout(timeout);
      process.stdin.removeListener("data", onData);
      process.stdin.removeListener("end", onEnd);
      process.stdin.removeListener("error", onError);
    };

    const onData = (data: Buffer) => {
      cleanup();
      const answer = data.toString().trim().toLowerCase();
      resolve(answer === "y" || answer === "yes");
    };

    const onEnd = () => {
      cleanup();
      process.stdout.write(" (stdin closed - assuming no)\n");
      resolve(false);
    };

    const onError = () => {
      cleanup();
      process.stdout.write(" (stdin error - assuming no)\n");
      resolve(false);
    };

    process.stdout.write(`${question} [y/N]: `);
    process.stdin.once("data", onData);
    process.stdin.once("end", onEnd);
    process.stdin.once("error", onError);
  });
}

interface RiskSurvey {
  touches_security: boolean;
  touches_payment: boolean;
  touches_database: boolean;
  touches_deploy: boolean;
  risk_level: "low" | "normal" | "high";
}

function suggestTier(survey: RiskSurvey): "light" | "standard" | "deep" {
  if (survey.risk_level === "high") return "deep";
  if (
    survey.touches_security ||
    survey.touches_payment ||
    survey.touches_deploy
  )
    return "deep";
  if (survey.risk_level === "normal") return "standard";
  return "light";
}

export async function checkReadinessAction(opts: {
  interactive?: boolean;
  nonInteractive?: boolean;
  json?: boolean;
  root?: string;
}): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const result = await checkReadiness(
    opts.interactive ?? false,
    opts.nonInteractive ?? false,
    root
  );
  const { ready, checks } = result;

  if (opts.json) {
    console.log(
      JSON.stringify(
        {
          ready,
          checks,
          readiness: result.readiness,
        },
        null,
        2
      )
    );
  } else {
    console.log(`handoff readiness: ${ready ? "READY" : "NOT READY"}`);
    for (const c of checks) {
      console.log(`  [${c.passed ? "PASS" : "FAIL"}] ${c.name}: ${c.note}`);
    }
    if (result.readiness) {
      console.log(`  suggested_tier: ${result.readiness.suggested_tier}`);
      if (Object.keys(result.readiness.risk_flags).length > 0) {
        console.log(
          `  risk_flags: ${
            Object.entries(result.readiness.risk_flags)
              .filter(([, v]) => v)
              .map(([k]) => k)
              .join(", ") || "none"
          }`
        );
      }
    }
  }

  process.exit(ready ? 0 : 1);
}

async function askRiskSurvey(): Promise<RiskSurvey> {
  const touches_security = await askQuestion(
    "Does the task touch authentication, authorization, or security boundaries?"
  );
  const touches_payment = await askQuestion(
    "Does the task touch payment, billing, or financial data?"
  );
  const touches_database = await askQuestion(
    "Does the task modify database schema or migration logic?"
  );
  const touches_deploy = await askQuestion(
    "Does the task affect deployment, infrastructure, or release pipelines?"
  );

  // Determine risk level from answers
  const risk_level: "low" | "normal" | "high" =
    touches_security || touches_payment || touches_deploy
      ? "high"
      : touches_database
        ? "normal"
        : "low";

  return {
    touches_security,
    touches_payment,
    touches_database,
    touches_deploy,
    risk_level,
  };
}

async function checkReadiness(
  interactive: boolean,
  nonInteractive: boolean,
  root: string
): Promise<{
  ready: boolean;
  checks: { name: string; passed: boolean; note: string }[];
  readiness?: {
    proceed: boolean;
    suggested_tier: "light" | "standard" | "deep";
    risk_flags: Record<string, boolean>;
    missing_information: string[];
    evidence_expected: string[];
  };
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

  if (interactive && !nonInteractive && !isNonInteractive()) {
    if (!allPassed) {
      console.log("Readiness checks failed:");
      for (const c of checks) {
        console.log(`  [${c.passed ? "PASS" : "FAIL"}] ${c.name}: ${c.note}`);
      }
      return {
        ready: false,
        checks,
        readiness: {
          proceed: false,
          suggested_tier: "light",
          risk_flags: {},
          missing_information: ["structural checks failed"],
          evidence_expected: [],
        },
      };
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
      return {
        ready: false,
        checks,
        readiness: {
          proceed: false,
          suggested_tier: "light",
          risk_flags: {},
          missing_information: ["scope unclear"],
          evidence_expected: [],
        },
      };
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
      return {
        ready: false,
        checks,
        readiness: {
          proceed: false,
          suggested_tier: "light",
          risk_flags: {},
          missing_information: ["no evidence plan"],
          evidence_expected: ["tests", "lint", "build"],
        },
      };
    }

    // Risk survey
    const survey = await askRiskSurvey();
    const suggested_tier = suggestTier(survey);

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
    checks.push({
      name: "risk_survey",
      passed: true,
      note: `risk_level=${survey.risk_level}, suggested_tier=${suggested_tier}`,
    });

    return {
      ready: true,
      checks,
      readiness: {
        proceed: true,
        suggested_tier,
        risk_flags: {
          touches_security: survey.touches_security,
          touches_payment: survey.touches_payment,
          touches_database: survey.touches_database,
          touches_deploy: survey.touches_deploy,
        },
        missing_information: [],
        evidence_expected: ["tests", "lint", "build"],
      },
    };
  }

  // Non-interactive mode: just report
  if (!allPassed) {
    return {
      ready: false,
      checks,
      readiness: {
        proceed: false,
        suggested_tier: "light",
        risk_flags: {},
        missing_information: ["structural checks failed"],
        evidence_expected: [],
      },
    };
  }

  // In non-interactive mode, we can't ask questions, so we assume the
  // structural checks are sufficient and mark readiness as advisory.
  checks.push({
    name: "interactive_prompts",
    passed: true,
    note: "Non-interactive mode: skipping readiness prompts",
  });
  return {
    ready: true,
    checks,
    readiness: {
      proceed: true,
      suggested_tier: "standard",
      risk_flags: {},
      missing_information: [],
      evidence_expected: ["tests", "lint", "build"],
    },
  };
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
- ${renderFixStatusGuidance()}

## Return format
Align with the compatibility subagent return schema:
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
    .option("--non-interactive", "Explicitly skip interactive prompts", false)
    .option("--json", "Output JSON instead of text", false)
    .option("--root <path>", "Repository root", process.cwd())
    .action(checkReadinessAction);

  return cmd;
}
