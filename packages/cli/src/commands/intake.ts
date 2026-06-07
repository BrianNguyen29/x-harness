import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import yaml from "yaml";
import { readYamlOrJson } from "../core/schema.js";
import {
  buildProductIntentRecord,
  classifyTask,
  explainCardIntake,
  formatHandoffAutoResult,
  loadIntakePolicy,
  normalizeAmbiguityStatus,
  ParseMarkdownIntent,
  parseBoolStrict,
  splitCsv,
  writeProductIntentOutput,
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

export interface IntakeContractOptions {
  id?: string;
  goal?: string;
  visible?: string;
  nonGoal?: string[];
  acceptance?: string[];
  protectedBehavior?: string[];
  ambiguity?: string;
  ambiguityQuestion?: string[];
  note?: string;
  from?: string;
  output?: string;
  json?: boolean;
}

export interface IntakeHandoffOptions {
  tier?: string;
  task?: string;
  file?: string[];
  root?: string;
  json?: boolean;
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

export async function intakeContractAction(
  opts: IntakeContractOptions
): Promise<void> {
  let userVisibleChange: boolean | null = null;
  if (opts.visible !== undefined && opts.visible !== "") {
    const parsed = parseBoolStrict(opts.visible);
    if (parsed === null) {
      console.error(
        `error: --visible expected true or false, got ${JSON.stringify(opts.visible)}`
      );
      process.exit(2);
    }
    userVisibleChange = parsed;
  }

  let ambiguity = normalizeAmbiguityStatus(opts.ambiguity ?? "");
  if (ambiguity === null) {
    console.error(
      `error: --ambiguity expected none, unresolved, or partial, got ${JSON.stringify(opts.ambiguity)}`
    );
    process.exit(2);
  }

  let id = opts.id ?? "";
  let productGoal = opts.goal ?? "";
  let nonGoals = flattenList(opts.nonGoal);
  let acceptance = flattenList(opts.acceptance);
  let protectedBehavior = flattenList(opts.protectedBehavior);
  let ambiguityQuestions = flattenList(opts.ambiguityQuestion);
  let notes = opts.note ?? "";

  if (opts.from) {
    const contentFlagSet =
      id !== "" ||
      productGoal !== "" ||
      userVisibleChange !== null ||
      nonGoals.length > 0 ||
      acceptance.length > 0 ||
      protectedBehavior.length > 0 ||
      opts.ambiguity !== undefined ||
      ambiguityQuestions.length > 0 ||
      notes !== "";
    if (contentFlagSet) {
      console.error(
        "error: --from is mutually exclusive with --id/--goal/--visible/--non-goal/--acceptance/--protected-behavior/--ambiguity/--ambiguity-question/--note"
      );
      process.exit(2);
    }
    let data: string;
    try {
      data = await fs.readFile(opts.from, "utf-8");
    } catch (err) {
      console.error(
        `error: --from ${err instanceof Error ? err.message : String(err)}`
      );
      process.exit(1);
    }
    const parsed = ParseMarkdownIntent(data);
    if (parsed.error || !parsed.spec) {
      console.error(`error: ${parsed.error ?? "unknown error"}`);
      process.exit(2);
    }
    const mdSpec = parsed.spec;
    id = mdSpec.id;
    productGoal = mdSpec.product_goal;
    userVisibleChange = mdSpec.user_visible_change;
    nonGoals = mdSpec.non_goals;
    acceptance = mdSpec.acceptance;
    protectedBehavior = mdSpec.protected_behaviors;
    if (mdSpec.ambiguity_set) ambiguity = "partial";
    ambiguityQuestions = mdSpec.ambiguity_questions;
    notes = mdSpec.notes;
  }

  const result = buildProductIntentRecord({
    id,
    product_goal: productGoal,
    user_visible_change: userVisibleChange,
    non_goals: nonGoals,
    acceptance,
    protected_behavior: protectedBehavior,
    ambiguity_status: ambiguity,
    ambiguity_questions: ambiguityQuestions,
    notes,
  });
  if (result.error || !result.record) {
    console.error(`error: ${result.error ?? "unknown error"}`);
    process.exit(2);
  }
  const record = result.record;

  const rendered = opts.json
    ? `${JSON.stringify(record, null, 2)}\n`
    : yaml.stringify(record);

  if (opts.output) {
    const writeResult = await writeProductIntentOutput(opts.output, rendered);
    if (writeResult.error) {
      console.error(`error: ${writeResult.error}`);
      process.exit(1);
    }
    return;
  }

  process.stdout.write(rendered);
}

export async function intakeHandoffAction(
  opts: IntakeHandoffOptions
): Promise<void> {
  if (opts.tier === undefined || opts.tier === "") {
    console.error(
      "error: --tier is required (safe V1 supports only --tier auto)"
    );
    process.exit(2);
  }
  if (opts.tier !== "auto") {
    console.error(
      `error: --tier ${JSON.stringify(opts.tier)} is not supported in safe V1; use \`xh handoff ${opts.tier}\` for explicit tiers, or pass --tier auto`
    );
    process.exit(2);
  }

  const root = path.resolve(opts.root ?? process.cwd());
  const policy = loadIntakePolicy(root);
  if (!policy) {
    console.error("Error: policies/intake.yaml not found");
    process.exit(2);
  }

  const task = opts.task ?? "unknown";
  const files = flattenList(opts.file);

  const classification = classifyTask(task, files, "", policy);
  const result = formatHandoffAutoResult(classification, task, files);

  if (opts.json) {
    process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
    return;
  }

  console.log(`Task: ${task}`);
  if (files.length > 0) {
    console.log(`Files: ${files.join(", ")}`);
  }
  console.log(`Selected tier: ${classification.runtime_tier}`);
  console.log(`Intake label: ${classification.intake_label}`);
  if (classification.auto_escalated) {
    console.log("Auto escalated: yes");
  }
  console.log("Reasoning:");
  for (const r of classification.reasoning) {
    console.log(`  - ${r}`);
  }
  console.log("");
  console.log(`Suggested next: ${result.command_suggestion}`);
}

// flattenList takes repeated `--flag a --flag b` values (collected by
// Commander as string[]) and, for each value, splits it by comma. This
// mirrors the Go `appendList` helper used by `xh intake contract/handoff`
// so callers can pass either repeatable flags or comma-delimited values.
function flattenList(values: string[] | undefined): string[] {
  if (!values) return [];
  const out: string[] = [];
  for (const value of values) {
    for (const part of splitCsv(value)) {
      out.push(part);
    }
  }
  return out;
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
    )
    .addCommand(
      new Command("contract")
        .description(
          "Build a safe V1 product intent record from structured flags or --from <markdown>"
        )
        .option("--id <id>", "Stable identifier for the product intent")
        .option("--goal <text>", "Plain-language product goal")
        .option("--visible <value>", "true or false (user-visible change)")
        .option(
          "--non-goal <text>",
          "Non-goal entry (repeatable or comma-delimited)",
          collectStrings,
          [] as string[]
        )
        .option(
          "--acceptance <text>",
          "Acceptance criterion (repeatable or comma-delimited)",
          collectStrings,
          [] as string[]
        )
        .option(
          "--protected-behavior <text>",
          "Protected behavior (repeatable or comma-delimited)",
          collectStrings,
          [] as string[]
        )
        .option(
          "--ambiguity <status>",
          "Ambiguity status: none, unresolved, or partial"
        )
        .option(
          "--ambiguity-question <text>",
          "Ambiguity question (repeatable or comma-delimited)",
          collectStrings,
          [] as string[]
        )
        .option("--note <text>", "Free-form notes")
        .option(
          "--from <path>",
          "Read product intent fields from a markdown file (mutually exclusive with content flags)"
        )
        .option("--output <path>", "Write the record to a file")
        .option("--json", "Output JSON instead of YAML", false)
        .action(intakeContractAction)
    )
    .addCommand(
      new Command("handoff")
        .description(
          "Suggest a handoff command after running the intake classifier (safe V1: --tier auto only)"
        )
        .option("--tier <tier>", "Tier selector (safe V1 supports only 'auto')")
        .option("--task <text>", "Task description", "")
        .option(
          "--file <path>",
          "File path (repeatable or comma-delimited)",
          collectStrings,
          [] as string[]
        )
        .option("--root <path>", "Repository root", process.cwd())
        .option("--json", "Output JSON instead of text", false)
        .action(intakeHandoffAction)
    );
}

// collectStrings is a Commander option parser that accumulates repeated
// flag values into an array. The action layer then splits each value
// by comma so callers can pass either repeatable flags or
// comma-delimited values.
const collectStrings = (value: string, previous: string[] = []): string[] => [
  ...previous,
  value,
];
