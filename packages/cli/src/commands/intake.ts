import { Command } from "commander";
import * as path from "node:path";
import { readYamlOrJson } from "../core/schema.js";
import {
  classifyTask,
  explainCardIntake,
  loadIntakePolicy,
} from "../core/intake.js";

export interface IntakeOptions {
  task?: string;
  files?: string;
  change?: string;
  json?: boolean;
  root?: string;
}

export interface IntakeExplainOptions {
  card?: string;
  json?: boolean;
  root?: string;
}

export async function intakeClassifyAction(opts: IntakeOptions): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const policy = loadIntakePolicy(root);

  if (!policy) {
    console.error("Error: policies/intake.yaml not found");
    process.exit(2);
  }

  const task = opts.task ?? "unknown";
  const files = opts.files ? opts.files.split(",").map((f) => f.trim()) : [];
  const change = opts.change;

  const result = classifyTask(task, files, change, policy);

  if (opts.json) {
    console.log(
      JSON.stringify(
        {
          intake_label: result.intake_label,
          runtime_tier: result.runtime_tier,
          task,
          files,
          change,
          reasoning: result.reasoning,
          signals: result.signals,
          negative_signals_considered: result.negative_signals_considered,
          auto_escalated: result.auto_escalated,
          policy_valid: true,
        },
        null,
        2
      )
    );
    return;
  }

  console.log(`Task: ${task}`);
  console.log(`Files: ${files.join(", ") || "(none)"}`);
  if (change) console.log(`Change type: ${change}`);
  console.log("");
  console.log(`Intake label: ${result.intake_label}`);
  console.log(`Runtime tier: ${result.runtime_tier}`);
  console.log("");
  console.log("Reasoning:");
  for (const r of result.reasoning) {
    console.log(`  - ${r}`);
  }
}

export async function intakeExplainAction(
  opts: IntakeExplainOptions
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const policy = loadIntakePolicy(root);

  if (!policy) {
    console.error("Error: policies/intake.yaml not found");
    process.exit(2);
  }

  if (!opts.card) {
    console.error("Error: --card <path> is required");
    process.exit(2);
  }

  const cardPath = path.resolve(opts.card);
  let card: Record<string, unknown>;
  try {
    card = (await readYamlOrJson(cardPath)) as Record<string, unknown>;
  } catch (err) {
    console.error(
      `Error loading card: ${err instanceof Error ? err.message : String(err)}`
    );
    process.exit(1);
  }

  const explanation = explainCardIntake(card, policy);

  if (opts.json) {
    console.log(JSON.stringify(explanation, null, 2));
    process.exit(explanation.ok ? 0 : 1);
  }

  console.log(`Card: ${path.relative(root, cardPath)}`);
  console.log(`Source: ${explanation.source}`);
  console.log(`Declared tier: ${explanation.declared_tier ?? "(missing)"}`);
  console.log(`Intake label: ${explanation.intake_label}`);
  console.log(`Mapped tier: ${explanation.mapped_tier}`);
  console.log(`Tier downgrade: ${explanation.tier_downgrade ? "yes" : "no"}`);
  if (explanation.intervention_required) {
    console.log(
      `Intervention approved: ${
        explanation.intervention_approved ? "yes" : "no"
      }`
    );
  }
  console.log("");
  console.log("Reasoning:");
  for (const r of explanation.reasoning) {
    console.log(`  - ${r}`);
  }
  if (explanation.warnings.length > 0) {
    console.log("");
    console.log("Warnings:");
    for (const warning of explanation.warnings) {
      console.log(`  - ${warning}`);
    }
  }
  if (explanation.errors.length > 0) {
    console.log("");
    console.log("Errors:");
    for (const err of explanation.errors) {
      console.log(`  - ${err}`);
    }
  }

  process.exit(explanation.ok ? 0 : 1);
}

export function intakeCommand(): Command {
  return new Command("intake")
    .description("Intake classification for x-harness tasks")
    .addCommand(
      new Command("classify")
        .description("Classify a task based on signals and file paths")
        .option("--task <text>", "Task description", "")
        .option("--files <paths>", "Comma-separated file paths", "")
        .option("--change <type>", "Change type (e.g., comment-only)", "")
        .option("--json", "Output JSON", false)
        .option("--root <path>", "Repository root", process.cwd())
        .action(intakeClassifyAction)
    )
    .addCommand(
      new Command("explain")
        .description("Explain intake classification for a completion card")
        .option("--card <path>", "Path to completion card YAML/JSON")
        .option("--json", "Output JSON", false)
        .option("--root <path>", "Repository root", process.cwd())
        .action(intakeExplainAction)
    );
}
