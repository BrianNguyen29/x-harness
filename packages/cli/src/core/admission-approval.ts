import type { AdmissionInput } from "./admission.js";
import { classifyCommand, riskMeetsThreshold, type CommandClassification } from "./classify.js";
import { getEvidenceRecord } from "./admission-accessors.js";

export interface ApprovalFinding {
  message: string;
  predicate: string;
}

export interface ApprovalEvaluation {
  errors: ApprovalFinding[];
  notes: string[];
}

function stringInMap(m: Record<string, unknown> | undefined, key: string): string {
  if (!m) return "";
  const v = m[key];
  if (typeof v === "string") return v;
  return "";
}

function mapValue(doc: Record<string, unknown> | undefined, key: string): Record<string, unknown> | undefined {
  if (!doc) return undefined;
  const v = doc[key];
  if (v && typeof v === "object" && !Array.isArray(v)) return v as Record<string, unknown>;
  return undefined;
}

function sliceInMap(m: Record<string, unknown> | undefined, key: string): unknown[] {
  if (!m) return [];
  const v = m[key];
  if (Array.isArray(v)) return v;
  return [];
}

export function evaluateApprovalReceipt(
  input: AdmissionInput,
  tier: string | undefined
): ApprovalEvaluation {
  const result: ApprovalEvaluation = { errors: [], notes: [] };
  if (tier !== "standard" && tier !== "deep") {
    return result;
  }

  const evidence = getEvidenceRecord(input);
  if (!evidence) {
    return result;
  }

  const commands: string[] = [];
  for (const item of sliceInMap(evidence, "command_evidence")) {
    if (!item || typeof item !== "object") continue;
    const record = item as Record<string, unknown>;
    const cmd = stringInMap(record, "command");
    if (cmd) commands.push(cmd);
  }
  for (const item of sliceInMap(evidence, "verification_artifacts")) {
    if (!item || typeof item !== "object") continue;
    const record = item as Record<string, unknown>;
    const cmd = stringInMap(record, "command");
    if (cmd) commands.push(cmd);
  }

  const requiringApproval: CommandClassification[] = [];
  let maxRequiredRisk = "";
  for (const cmd of commands) {
    const classification = classifyCommand(cmd);
    let needsApproval = false;
    switch (tier) {
      case "standard":
        if (classification.risk === "high" || classification.unknown) {
          needsApproval = true;
        }
        break;
      case "deep":
        if (classification.risk === "medium" || classification.risk === "high" || classification.unknown) {
          needsApproval = true;
        }
        break;
    }
    if (needsApproval) {
      requiringApproval.push(classification);
      if (riskMeetsThreshold(classification.risk, maxRequiredRisk)) {
        maxRequiredRisk = classification.risk;
      }
    }
  }

  if (requiringApproval.length === 0) {
    return result;
  }

  const receipt = mapValue(input as unknown as Record<string, unknown>, "approval_receipt");
  if (receipt === undefined) {
    result.errors.push({
      message: `tier ${tier} requires approval receipt for ${requiringApproval.length} high-risk command(s)`,
      predicate: "classifier_approval_required",
    });
    return result;
  }

  const decision = stringInMap(receipt, "decision");
  const approver = stringInMap(receipt, "approver");
  const aggregateRisk = stringInMap(receipt, "aggregate_risk");
  const classifiedCmds = sliceInMap(receipt, "classified_commands");

  if (decision !== "approved") {
    result.errors.push({
      message: `approval_receipt decision is "${decision}"; must be 'approved'`,
      predicate: "classifier_approval_required",
    });
  }
  if (approver.trim() === "") {
    result.errors.push({
      message: "approval_receipt approver is required",
      predicate: "classifier_approval_required",
    });
  }
  if (classifiedCmds.length === 0) {
    result.errors.push({
      message: "approval_receipt classified_commands is required",
      predicate: "classifier_approval_required",
    });
  }
  if (maxRequiredRisk !== "" && !riskMeetsThreshold(aggregateRisk, maxRequiredRisk)) {
    result.errors.push({
      message: `approval_receipt aggregate_risk "${aggregateRisk}" is below required threshold "${maxRequiredRisk}"`,
      predicate: "classifier_approval_required",
    });
  }

  // Build coverage map from receipt classified commands
  const covered = new Set<string>();
  for (const item of classifiedCmds) {
    if (!item || typeof item !== "object") continue;
    const rec = item as Record<string, unknown>;
    const cmd = stringInMap(rec, "command");
    if (cmd) covered.add(cmd);
  }

  for (const classification of requiringApproval) {
    if (!covered.has(classification.command)) {
      result.errors.push({
        message: `approval_receipt does not cover command "${classification.command}" (risk: ${classification.risk})`,
        predicate: "classifier_approval_required",
      });
    }
  }

  if (result.errors.length === 0) {
    result.notes.push(`approval_receipt validated for ${requiringApproval.length} high-risk command(s)`);
  }

  return result;
}
