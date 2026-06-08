import * as path from "node:path";
import fs from "fs-extra";
import yaml from "yaml";

export type IntakeLabel = "tiny" | "normal" | "high_risk";
export type RuntimeTier = "light" | "standard" | "deep";
export type AmbiguityStatus = "none" | "unresolved" | "partial";

// PRODUCT_INTENT_SCHEMA_VERSION is the schema_version emitted by
// `xh intake contract` for safe V1 product intent records. Fixed for the
// first slice to keep the contract deterministic.
export const PRODUCT_INTENT_SCHEMA_VERSION = "1";

export interface IntakePolicy {
  intake_labels: Record<
    IntakeLabel,
    { runtime_tier: RuntimeTier; signals: string[] }
  >;
  high_risk_signals: Record<
    string,
    { description: string; examples: string[] }
  >;
  runtime_tier_confirmation: { tiers: RuntimeTier[]; note: string };
}

export interface IntakeClassification {
  intake_label: IntakeLabel;
  runtime_tier: RuntimeTier;
  reasoning: string[];
  signals: string[];
  negative_signals_considered: string[];
  auto_escalated: boolean;
}

export interface IntakeExplanation {
  ok: boolean;
  source: "declared" | "inferred";
  declared_tier: RuntimeTier | null;
  intake_label: IntakeLabel;
  mapped_tier: RuntimeTier;
  tier_downgrade: boolean;
  intervention_required: boolean;
  intervention_approved: boolean;
  reasoning: string[];
  signals: string[];
  negative_signals_considered: string[];
  errors: string[];
  warnings: string[];
}

const TIER_RANK: Record<RuntimeTier, number> = {
  light: 0,
  standard: 1,
  deep: 2,
};

const HIGH_RISK_KEYWORDS = [
  "auth",
  "token",
  "session",
  "admission",
  "schema",
  "permissions",
  "permission",
  "ci",
  "release",
  "destroy",
  "delete",
];

const HIGH_RISK_FILE_PATTERNS = [
  { signal: "auth", pattern: /auth/i },
  { signal: "token", pattern: /token/i },
  { signal: "session", pattern: /session/i },
  { signal: "admission", pattern: /admission/i },
  { signal: "permissions", pattern: /permission/i },
  { signal: "schema", pattern: /schema/i },
  { signal: "release", pattern: /release/i },
];

const DESTRUCTIVE_PATTERNS = [/-rf/, /rm\s+-r/, /unlink/i, /del.*tree/i];

export function isRuntimeTier(value: unknown): value is RuntimeTier {
  return value === "light" || value === "standard" || value === "deep";
}

export function isIntakeLabel(value: unknown): value is IntakeLabel {
  return value === "tiny" || value === "normal" || value === "high_risk";
}

export function isTierDowngrade(
  declaredTier: RuntimeTier,
  mappedTier: RuntimeTier
): boolean {
  return TIER_RANK[declaredTier] < TIER_RANK[mappedTier];
}

export function hasApprovedTierDowngradeIntervention(
  governance: Record<string, unknown> | undefined
): boolean {
  if (!governance) return false;
  if (governance.approval_status !== "approved") return false;
  const approvalRequiredFor = Array.isArray(governance.approval_required_for)
    ? governance.approval_required_for
        .filter((item): item is string => typeof item === "string")
        .map((item) => item.toLowerCase().replace(/[-\s]+/g, "_"))
    : [];
  return (
    approvalRequiredFor.includes("tier_downgrade") ||
    approvalRequiredFor.includes("intake_tier_downgrade")
  );
}

export function loadIntakePolicy(root: string): IntakePolicy | null {
  const policyPath = path.join(root, "policies", "intake.yaml");
  if (!fs.existsSync(policyPath)) {
    return null;
  }
  const content = fs.readFileSync(policyPath, "utf-8");
  return yaml.parse(content) as IntakePolicy;
}

export function classifyTask(
  task: string,
  files: string[],
  change: string | undefined,
  _policy: IntakePolicy
): IntakeClassification {
  const reasoning: string[] = [];
  const signals: string[] = [];
  const negativeSignalsConsidered: string[] = [];
  const taskLower = task.toLowerCase();

  if (
    change === "comment-only" ||
    change === "comments" ||
    change === "comment"
  ) {
    signals.push("comment_only");
    reasoning.push("Change signal indicates comment-only modification");
    reasoning.push("Mapping to tiny/light");
    return {
      intake_label: "tiny",
      runtime_tier: "light",
      reasoning,
      signals,
      negative_signals_considered: ["behavior_change"],
      auto_escalated: false,
    };
  }
  negativeSignalsConsidered.push("comment_only");

  for (const keyword of HIGH_RISK_KEYWORDS) {
    if (taskLower.includes(keyword)) {
      signals.push(keyword);
      reasoning.push(`Task description contains high-risk keyword: ${keyword}`);
      reasoning.push("Mapping to high_risk/deep");
      return {
        intake_label: "high_risk",
        runtime_tier: "deep",
        reasoning,
        signals,
        negative_signals_considered: negativeSignalsConsidered,
        auto_escalated: true,
      };
    }
  }

  for (const file of files) {
    const fileLower = file.toLowerCase();
    for (const { signal, pattern } of HIGH_RISK_FILE_PATTERNS) {
      if (pattern.test(fileLower)) {
        signals.push(signal);
        reasoning.push(`File path matches high-risk pattern: ${pattern}`);
        reasoning.push("Mapping to high_risk/deep");
        return {
          intake_label: "high_risk",
          runtime_tier: "deep",
          reasoning,
          signals,
          negative_signals_considered: negativeSignalsConsidered,
          auto_escalated: true,
        };
      }
    }
  }

  if (files.some((f) => f.includes(".github/workflows"))) {
    signals.push("ci");
    reasoning.push("Files include CI/CD workflows");
    reasoning.push("Mapping to high_risk/deep");
    return {
      intake_label: "high_risk",
      runtime_tier: "deep",
      reasoning,
      signals,
      negative_signals_considered: negativeSignalsConsidered,
      auto_escalated: true,
    };
  }

  for (const file of files) {
    for (const pattern of DESTRUCTIVE_PATTERNS) {
      if (pattern.test(file)) {
        signals.push("destructive_filesystem");
        reasoning.push(`File path suggests destructive operation: ${pattern}`);
        reasoning.push("Mapping to high_risk/deep");
        return {
          intake_label: "high_risk",
          runtime_tier: "deep",
          reasoning,
          signals,
          negative_signals_considered: negativeSignalsConsidered,
          auto_escalated: true,
        };
      }
    }
  }

  signals.push("routine_implementation");
  negativeSignalsConsidered.push("auth", "token", "session", "ci", "release");
  reasoning.push("No high-risk signals detected");
  reasoning.push("Mapping to normal/standard");
  return {
    intake_label: "normal",
    runtime_tier: "standard",
    reasoning,
    signals,
    negative_signals_considered: negativeSignalsConsidered,
    auto_escalated: false,
  };
}

function getStringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : [];
}

function getCardFiles(card: Record<string, unknown>): string[] {
  const evidence = card.evidence as Record<string, unknown> | undefined;
  const state = card.state as Record<string, unknown> | undefined;
  return [
    ...getStringArray(evidence?.files_changed),
    ...getStringArray(state?.write_set),
  ];
}

function getCardTask(card: Record<string, unknown>): string {
  const claim = card.claim as Record<string, unknown> | undefined;
  if (typeof claim?.summary === "string" && claim.summary.trim().length > 0) {
    return claim.summary;
  }
  if (typeof card.task_id === "string" && card.task_id.trim().length > 0) {
    return card.task_id;
  }
  return "unknown";
}

export function explainCardIntake(
  card: Record<string, unknown>,
  policy: IntakePolicy
): IntakeExplanation {
  const errors: string[] = [];
  const warnings: string[] = [];
  const declaredTier = isRuntimeTier(card.tier) ? card.tier : null;
  const governance = card.governance as Record<string, unknown> | undefined;
  const intake = card.intake as Record<string, unknown> | undefined;

  let source: "declared" | "inferred" = "inferred";
  let classification: IntakeClassification;

  if (intake) {
    source = "declared";
    const intakeLabel = intake.classification;
    const mappedTier = intake.mapped_tier;
    if (!isIntakeLabel(intakeLabel)) {
      errors.push("intake.classification must be tiny, normal, or high_risk");
    }
    if (!isRuntimeTier(mappedTier)) {
      errors.push("intake.mapped_tier must be light, standard, or deep");
    }
    const normalizedLabel = isIntakeLabel(intakeLabel) ? intakeLabel : "normal";
    const normalizedTier = isRuntimeTier(mappedTier)
      ? mappedTier
      : policy.intake_labels[normalizedLabel].runtime_tier;
    const policyTier = policy.intake_labels[normalizedLabel].runtime_tier;
    if (normalizedTier !== policyTier) {
      errors.push(
        `intake.mapped_tier "${normalizedTier}" does not match policy tier "${policyTier}" for ${normalizedLabel}`
      );
    }
    classification = {
      intake_label: normalizedLabel,
      runtime_tier: normalizedTier,
      reasoning:
        typeof intake.rationale === "string" && intake.rationale.length > 0
          ? [intake.rationale]
          : ["Declared intake block has no rationale."],
      signals: getStringArray(intake.signals),
      negative_signals_considered: getStringArray(
        intake.negative_signals_considered
      ),
      auto_escalated: intake.auto_escalated === true,
    };
  } else {
    warnings.push(
      "completion card has no intake block; explanation is inferred from claim/evidence"
    );
    classification = classifyTask(
      getCardTask(card),
      getCardFiles(card),
      undefined,
      policy
    );
  }

  const downgrade =
    declaredTier !== null &&
    isTierDowngrade(declaredTier, classification.runtime_tier);
  const interventionApproved =
    downgrade && hasApprovedTierDowngradeIntervention(governance);
  if (downgrade && !interventionApproved) {
    errors.push(
      `intake tier downgrade requires governance intervention approval: declared ${declaredTier}, mapped ${classification.runtime_tier}`
    );
  }

  return {
    ok: errors.length === 0,
    source,
    declared_tier: declaredTier,
    intake_label: classification.intake_label,
    mapped_tier: classification.runtime_tier,
    tier_downgrade: downgrade,
    intervention_required: downgrade,
    intervention_approved: interventionApproved,
    reasoning: classification.reasoning,
    signals: classification.signals,
    negative_signals_considered: classification.negative_signals_considered,
    errors,
    warnings,
  };
}

// splitCsv splits a comma-delimited raw string into trimmed, non-empty
// entries. Used for repeatable flags that accept either repeated flag
// calls or a single comma-separated value.
export function splitCsv(raw: string): string[] {
  if (!raw) return [];
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

// parseBoolStrict parses a strict boolean. Returns null when the value is
// not one of true/yes/1/false/no/0 (case-insensitive). Mirrors the Go
// helper used by `xh intake contract --visible`.
export function parseBoolStrict(raw: string): boolean | null {
  switch (raw.toLowerCase().trim()) {
    case "true":
    case "yes":
    case "1":
      return true;
    case "false":
    case "no":
    case "0":
      return false;
    default:
      return null;
  }
}

// normalizeAmbiguityStatus coerces --ambiguity input into the safe V1
// enum (none/unresolved/partial). Empty input maps to "none". Returns
// null on unrecognized values.
export function normalizeAmbiguityStatus(raw: string): AmbiguityStatus | null {
  switch (raw.toLowerCase().trim()) {
    case "":
    case "none":
      return "none";
    case "unresolved":
      return "unresolved";
    case "partial":
      return "partial";
    default:
      return null;
  }
}

export interface ProductIntentSpec {
  id: string;
  product_goal: string;
  user_visible_change: boolean | null;
  non_goals: string[];
  acceptance: string[];
  protected_behavior: string[];
  ambiguity_status: AmbiguityStatus;
  ambiguity_questions: string[];
  notes: string;
}

export interface ProductIntentAcceptanceCriterion {
  id: string;
  statement: string;
  source_ref: string;
}

export interface ProductIntentRecord {
  schema_version: string;
  id: string;
  product_goal: string;
  user_visible_change: boolean | null;
  non_goals: string[];
  acceptance_criteria: ProductIntentAcceptanceCriterion[];
  protected_behaviors: string[];
  ambiguity: { status: AmbiguityStatus; questions: string[] };
  notes: string;
}

export interface ProductIntentBuildResult {
  record: ProductIntentRecord | null;
  error?: string;
}

// buildProductIntentRecord converts a structured spec into a record
// matching schemas/product-intent.schema.json (safe V1). Required
// fields: id, product_goal, at least one acceptance. Optional fields
// that are unset are emitted as null/empty values for stable round-trip.
export function buildProductIntentRecord(
  spec: ProductIntentSpec
): ProductIntentBuildResult {
  if (spec.id.trim() === "") {
    return { record: null, error: "--id is required" };
  }
  if (spec.product_goal.trim() === "") {
    return { record: null, error: "--goal is required" };
  }
  if (spec.acceptance.length === 0) {
    return {
      record: null,
      error:
        "at least one --acceptance is required (when using --from, include an ## Acceptance section with at least one item)",
    };
  }
  for (let i = 0; i < spec.acceptance.length; i++) {
    if (spec.acceptance[i].trim() === "") {
      return {
        record: null,
        error: `--acceptance entry ${i + 1} is blank`,
      };
    }
  }

  const acceptanceCriteria: ProductIntentAcceptanceCriterion[] =
    spec.acceptance.map((statement, i) => ({
      id: `ac-${i + 1}`,
      statement,
      source_ref: "",
    }));

  return {
    record: {
      schema_version: PRODUCT_INTENT_SCHEMA_VERSION,
      id: spec.id,
      product_goal: spec.product_goal,
      user_visible_change: spec.user_visible_change,
      non_goals: spec.non_goals,
      acceptance_criteria: acceptanceCriteria,
      protected_behaviors: spec.protected_behavior,
      ambiguity: {
        status: spec.ambiguity_status,
        questions: spec.ambiguity_questions,
      },
      notes: spec.notes,
    },
  };
}

// MarkdownIntentSpec is the result of parsing a markdown file for
// `xh intake contract --from <path>`. Field names mirror the
// product-intent schema. Unknown sections are skipped. Section
// validation is delegated to the canonical record builder.
export interface MarkdownIntentSpec {
  title: string;
  id: string;
  product_goal: string;
  user_visible_change: boolean | null;
  non_goals: string[];
  acceptance: string[];
  protected_behaviors: string[];
  ambiguity_questions: string[];
  notes: string;
  ambiguity_set: boolean;
}

const MARKDOWN_SECTION_ALIASES: Record<string, string> = {
  goal: "goal",
  "product goal": "goal",
  acceptance: "acceptance",
  "acceptance criteria": "acceptance",
  criteria: "acceptance",
  "non goals": "non-goals",
  "out of scope": "non-goals",
  "protected behavior": "protected-behavior",
  "protected behaviors": "protected-behavior",
  ambiguity: "ambiguity",
  "open questions": "ambiguity",
  "unresolved questions": "ambiguity",
  "user visible change": "user-visible-change",
  notes: "notes",
  context: "notes",
  problem: "notes",
};

const MARKDOWN_LIST_RE = /^\s*(?:[-*]|\d+\.)\s+(?:\[[ xX]\]\s+)?(.*)$/;

function canonicalSection(heading: string): string | null {
  const normalized = heading
    .toLowerCase()
    .trim()
    .replace(/[-_]+/g, " ")
    .split(/\s+/)
    .filter((part) => part.length > 0)
    .join(" ");
  return MARKDOWN_SECTION_ALIASES[normalized] ?? null;
}

function extractMarkdownListItems(lines: string[]): string[] {
  const items: string[] = [];
  for (const line of lines) {
    const match = MARKDOWN_LIST_RE.exec(line);
    if (!match) continue;
    const text = match[1].trim();
    if (text.length === 0) continue;
    items.push(text);
  }
  return items;
}

function firstMarkdownNonEmptyText(lines: string[]): string {
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.length === 0) continue;
    const match = MARKDOWN_LIST_RE.exec(line);
    if (match) {
      const text = match[1].trim();
      if (text.length > 0) return text;
      continue;
    }
    return trimmed;
  }
  return "";
}

function joinMarkdownNonEmptyText(lines: string[]): string {
  const parts: string[] = [];
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.length === 0) continue;
    const match = MARKDOWN_LIST_RE.exec(line);
    if (match) {
      const text = match[1].trim();
      if (text.length > 0) parts.push(text);
      continue;
    }
    parts.push(trimmed);
  }
  return parts.join("\n");
}

function slugifyMarkdownTitle(title: string): string {
  return title
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function parseMarkdownBoolStrict(raw: string): boolean | null {
  switch (raw.toLowerCase().trim()) {
    case "true":
    case "yes":
    case "1":
      return true;
    case "false":
    case "no":
    case "0":
      return false;
    default:
      return null;
  }
}

// ParseMarkdownIntent parses heading-based product intent content. Headings
// are matched case-insensitively; only `##` (level 2) sections are
// recognized for known section names. `###` headings are treated as
// content within the current section. The top-level `#` title provides
// default id and product_goal values unless overridden by sections.
export function ParseMarkdownIntent(md: string): {
  spec: MarkdownIntentSpec | null;
  error?: string;
} {
  const spec: MarkdownIntentSpec = {
    title: "",
    id: "",
    product_goal: "",
    user_visible_change: null,
    non_goals: [],
    acceptance: [],
    protected_behaviors: [],
    ambiguity_questions: [],
    notes: "",
    ambiguity_set: false,
  };

  let currentSection = "";
  let sectionLines: string[] = [];

  const flush = (): string | null => {
    if (currentSection === "") return null;
    return applyMarkdownSection(spec, currentSection, sectionLines);
  };

  for (const raw of md.split("\n")) {
    const line = raw.replace(/[ \t\r]+$/, "");
    if (line.startsWith("# ") && !line.startsWith("## ")) {
      const title = line.slice(2).trim();
      if (spec.title === "") spec.title = title;
      continue;
    }
    if (line.startsWith("## ")) {
      const err = flush();
      if (err) return { spec: null, error: err };
      currentSection = line.slice(3).trim();
      sectionLines = [];
      continue;
    }
    sectionLines.push(line);
  }
  const trailingErr = flush();
  if (trailingErr) return { spec: null, error: trailingErr };

  // Title-derived defaults: id always defaults to slugify(title);
  // product_goal defaults to title only when no Goal section set it.
  if (spec.id === "" && spec.title !== "") {
    spec.id = slugifyMarkdownTitle(spec.title);
  }
  if (spec.product_goal === "" && spec.title !== "") {
    spec.product_goal = spec.title;
  }
  return { spec };
}

function applyMarkdownSection(
  spec: MarkdownIntentSpec,
  heading: string,
  lines: string[]
): string | null {
  const canonical = canonicalSection(heading);
  if (!canonical) return null;
  switch (canonical) {
    case "goal":
      spec.product_goal = firstMarkdownNonEmptyText(lines);
      return null;
    case "acceptance":
      spec.acceptance = extractMarkdownListItems(lines);
      return null;
    case "non-goals":
      spec.non_goals = extractMarkdownListItems(lines);
      return null;
    case "protected-behavior":
      spec.protected_behaviors = extractMarkdownListItems(lines);
      return null;
    case "ambiguity": {
      const items = extractMarkdownListItems(lines);
      if (items.length > 0) {
        spec.ambiguity_set = true;
        spec.ambiguity_questions = items;
      }
      return null;
    }
    case "user-visible-change": {
      const text = firstMarkdownNonEmptyText(lines);
      if (text === "") return null;
      const parsed = parseMarkdownBoolStrict(text);
      if (parsed === null) {
        return `--from user-visible change: expected true or false, got "${text}"`;
      }
      spec.user_visible_change = parsed;
      return null;
    }
    case "notes":
      spec.notes = joinMarkdownNonEmptyText(lines);
      return null;
  }
  return null;
}

// writeProductIntentOutput writes the rendered product intent record
// to disk. To keep the slice safe V1, the parent directory must already
// exist; we do not auto-create intermediate directories because that
// would hide typos and is unnecessary for the typical use case where the
// user writes into a known workspace path.
export async function writeProductIntentOutput(
  outputPath: string,
  data: string
): Promise<{ error?: string }> {
  const parent = path.dirname(outputPath);
  try {
    const exists = await fs.pathExists(parent);
    if (!exists) {
      return { error: `parent directory does not exist: ${parent}` };
    }
  } catch (err) {
    return {
      error: err instanceof Error ? err.message : String(err),
    };
  }
  try {
    await fs.writeFile(outputPath, data);
    return {};
  } catch (err) {
    return {
      error: err instanceof Error ? err.message : String(err),
    };
  }
}

export interface HandoffAutoResult {
  selected_tier: RuntimeTier;
  intake_label: IntakeLabel;
  task: string;
  files: string[];
  signals: string[];
  reasoning: string[];
  auto_escalated: boolean;
  command_suggestion: string;
}

// formatHandoffAutoResult turns a classifier result into the minimal
// handoff suggestion payload emitted by `xh intake handoff --tier auto`.
export function formatHandoffAutoResult(
  classification: IntakeClassification,
  task: string,
  files: string[]
): HandoffAutoResult {
  return {
    selected_tier: classification.runtime_tier,
    intake_label: classification.intake_label,
    task,
    files,
    signals: classification.signals,
    reasoning: classification.reasoning,
    auto_escalated: classification.auto_escalated,
    command_suggestion: `xh handoff ${classification.runtime_tier} --task ${JSON.stringify(task)}`,
  };
}
