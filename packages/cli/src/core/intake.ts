import * as path from "node:path";
import fs from "fs-extra";
import yaml from "yaml";

export type IntakeLabel = "tiny" | "normal" | "high_risk";
export type RuntimeTier = "light" | "standard" | "deep";

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
