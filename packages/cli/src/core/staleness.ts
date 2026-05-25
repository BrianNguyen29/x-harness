import fs from "fs-extra";
import * as path from "node:path";
import { validateManagedBlock } from "./context.js";

export interface StalenessResult {
  stale: boolean;
  findings: StalenessFinding[];
}

export interface StalenessFinding {
  type: StalenessFindingType;
  severity: "error" | "warn" | "info";
  message: string;
  details?: Record<string, unknown>;
}

export type StalenessFindingType =
  | "missing_managed_block"
  | "stale_hash"
  | "missing_linked_file"
  | "deleted_command_reference"
  | "deleted_policy_reference";

const LINKED_FILES = [
  "X_HARNESS.md",
  "policies/admission.yaml",
  "policies/recovery.yaml",
  "schemas/completion-card.schema.json",
];

export async function checkStaleness(root: string): Promise<StalenessResult> {
  const findings: StalenessFinding[] = [];
  const agentsPath = path.join(root, "AGENTS.md");

  const agentsContent = await fs.readFile(agentsPath, "utf-8");
  const validation = validateManagedBlock(agentsContent);

  if (!validation.valid) {
    if (validation.note.includes("missing managed context block")) {
      findings.push({
        type: "missing_managed_block",
        severity: "error",
        message: "AGENTS.md is missing the managed context block",
      });
    } else if (validation.note.includes("stale")) {
      findings.push({
        type: "stale_hash",
        severity: "error",
        message: validation.note,
      });
    } else if (validation.note.includes("missing context-hash")) {
      findings.push({
        type: "stale_hash",
        severity: "error",
        message: "AGENTS.md managed block is missing context-hash",
      });
    } else {
      findings.push({
        type: "stale_hash",
        severity: "error",
        message: validation.note,
      });
    }
  }

  for (const linkedFile of LINKED_FILES) {
    const filePath = path.join(root, linkedFile);
    if (!(await fs.pathExists(filePath))) {
      findings.push({
        type: "missing_linked_file",
        severity: "warn",
        message: `Linked file does not exist: ${linkedFile}`,
        details: { file: linkedFile },
      });
    }
  }

  return {
    stale: findings.some((f) => f.severity === "error"),
    findings,
  };
}

export function getSourceOfTruthFiles(): string[] {
  return [
    "X_HARNESS.md",
    "policies/admission.yaml",
    "policies/recovery.yaml",
    "policies/intake.yaml",
    "schemas/completion-card.schema.json",
    "templates/SUBAGENT_TASK_light.md",
    "templates/SUBAGENT_TASK_standard.md",
    "templates/SUBAGENT_TASK_deep.md",
  ];
}

export function getManagedBlockReadingOrder(): string[] {
  return [
    "AGENTS.md (managed block)",
    "X_HARNESS.md",
    "policies/admission.yaml",
    "policies/recovery.yaml",
    "policies/intake.yaml",
    "schemas/completion-card.schema.json",
  ];
}
