import * as path from "node:path";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";

interface CostPolicy {
  version: number;
  cost_budget: {
    enabled: boolean;
    max_usd: number;
    max_input_tokens: number;
    max_output_tokens: number;
    over_budget_recovery: string;
    affects_admission: false;
  };
}

export interface CostBudgetReport {
  schema_version: "1";
  max_usd: number;
  actual_usd: number;
  token_usage: {
    input: number;
    output: number;
  };
  over_budget: boolean;
  status: "within_budget" | "over_budget";
  recovery: string;
  policy_enabled: boolean;
  enforcement_enabled: boolean;
  admission_authority: false;
}

async function validateReport(report: CostBudgetReport): Promise<void> {
  const schema = await loadSchema("cost-budget");
  const validate = compileSchema(schema);
  if (!validate(report)) {
    throw new Error(
      `cost budget report validation failed: ${(validate.errors ?? [])
        .map((err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`)
        .join("; ")}`
    );
  }
}

export async function loadCostPolicy(root: string): Promise<CostPolicy> {
  return (await readYamlOrJson(
    path.join(root, "policies", "cost-budget.yaml")
  )) as CostPolicy;
}

export async function evaluateCostBudget(input: {
  root: string;
  actualUsd: number;
  inputTokens: number;
  outputTokens: number;
  enforce?: boolean;
}): Promise<CostBudgetReport> {
  const policy = await loadCostPolicy(path.resolve(input.root));
  const overBudget =
    input.actualUsd > policy.cost_budget.max_usd ||
    input.inputTokens > policy.cost_budget.max_input_tokens ||
    input.outputTokens > policy.cost_budget.max_output_tokens;
  const report: CostBudgetReport = {
    schema_version: "1",
    max_usd: policy.cost_budget.max_usd,
    actual_usd: input.actualUsd,
    token_usage: {
      input: input.inputTokens,
      output: input.outputTokens,
    },
    over_budget: overBudget,
    status: overBudget ? "over_budget" : "within_budget",
    recovery: overBudget ? policy.cost_budget.over_budget_recovery : "none",
    policy_enabled: policy.cost_budget.enabled,
    enforcement_enabled: Boolean(input.enforce && policy.cost_budget.enabled),
    admission_authority: false,
  };
  await validateReport(report);
  return report;
}

export async function readCostBudgetReport(
  filePath: string
): Promise<CostBudgetReport> {
  const report = (await readYamlOrJson(
    path.resolve(filePath)
  )) as CostBudgetReport;
  await validateReport(report);
  return report;
}
