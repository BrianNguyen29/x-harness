import * as path from "node:path";
import { classifyPath, loadAuthorityPolicy } from "./authority.js";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";

interface ApprovalRiskPolicy {
  version: number;
  approval_risk: {
    enabled: boolean;
    personal_scoring: false;
    thresholds: Record<"moderate" | "elevated" | "critical", number>;
    required_approvals: Record<
      "low" | "moderate" | "elevated" | "critical",
      number
    >;
    signals: Record<string, number>;
  };
}

export interface ApprovalRiskReport {
  schema_version: "1";
  task_id: string;
  tier?: "light" | "standard" | "deep";
  risk_class: "low" | "moderate" | "elevated" | "critical";
  score: number;
  signals: string[];
  required_approvals: number;
  personal_scoring: false;
  policy_enabled: boolean;
  admission_authority: false;
}

function evidenceFiles(card: Record<string, unknown>): string[] {
  const evidence = card.evidence as Record<string, unknown> | undefined;
  const files = evidence?.files_changed;
  return Array.isArray(files)
    ? files.filter((item): item is string => typeof item === "string")
    : [];
}

function riskClass(
  score: number,
  thresholds: ApprovalRiskPolicy["approval_risk"]["thresholds"]
): ApprovalRiskReport["risk_class"] {
  if (score >= thresholds.critical) return "critical";
  if (score >= thresholds.elevated) return "elevated";
  if (score >= thresholds.moderate) return "moderate";
  return "low";
}

async function validateReport(report: ApprovalRiskReport): Promise<void> {
  const schema = await loadSchema("approval-risk");
  const validate = compileSchema(schema);
  if (!validate(report)) {
    throw new Error(
      `approval risk report validation failed: ${(validate.errors ?? [])
        .map((err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`)
        .join("; ")}`
    );
  }
}

export async function loadApprovalRiskPolicy(
  root: string
): Promise<ApprovalRiskPolicy> {
  return (await readYamlOrJson(
    path.join(root, "policies", "approval-risk.yaml")
  )) as ApprovalRiskPolicy;
}

export async function evaluateApprovalRisk(input: {
  root: string;
  cardPath: string;
}): Promise<ApprovalRiskReport> {
  const root = path.resolve(input.root);
  const card = (await readYamlOrJson(
    path.resolve(root, input.cardPath)
  )) as Record<string, unknown>;
  const policy = await loadApprovalRiskPolicy(root);
  const authority = await loadAuthorityPolicy(root);
  const signals = new Set<string>();
  let score = 0;

  const tier = card.tier as "light" | "standard" | "deep" | undefined;
  if (tier === "deep") {
    signals.add("deep_tier");
    score += policy.approval_risk.signals.deep_tier;
  }

  for (const file of evidenceFiles(card)) {
    const classified = classifyPath(file, authority);
    if (classified.authority === "human_only") {
      signals.add("human_only_path");
      score += policy.approval_risk.signals.human_only_path;
    }
    if (classified.authority === "agent_proposable_human_approved") {
      signals.add("human_approved_path");
      score += policy.approval_risk.signals.human_approved_path;
    }
    if (/(auth|token|secret|session|permission|policy)/i.test(file)) {
      signals.add("security_sensitive_path");
      score += policy.approval_risk.signals.security_sensitive_path;
    }
  }

  const governance = card.governance as Record<string, unknown> | undefined;
  if (
    signals.has("human_only_path") &&
    governance?.approval_status !== "approved"
  ) {
    signals.add("missing_governance_approval");
    score += policy.approval_risk.signals.missing_governance_approval;
  }

  const classified = riskClass(score, policy.approval_risk.thresholds);
  const report: ApprovalRiskReport = {
    schema_version: "1",
    task_id: String(card.task_id ?? "unknown"),
    ...(tier ? { tier } : {}),
    risk_class: classified,
    score,
    signals: [...signals].sort(),
    required_approvals: policy.approval_risk.required_approvals[classified],
    personal_scoring: false,
    policy_enabled: policy.approval_risk.enabled,
    admission_authority: false,
  };
  await validateReport(report);
  return report;
}
