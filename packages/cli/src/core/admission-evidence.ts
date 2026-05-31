import type { AdmissionInput } from "./admission.js";
import {
  getCommandEvidence,
  getEvidenceArray,
  getEvidenceRecord,
  getExecutionControls,
  getFilesChanged,
  getManualRationale,
  getRemainingRisks,
  getRollbackPolicy,
  getState,
  getUntestedRegions,
  getVerificationArtifacts,
} from "./admission-accessors.js";
import { classifyCommand } from "./classify.js";

export interface AdmissionFinding {
  message: string;
  predicate: string;
}

export interface AdmissionEvidenceEvaluation {
  errors: AdmissionFinding[];
  notes: string[];
}

function shellMetacharacter(command: string): string | null {
  const checks: Array<[string, RegExp]> = [
    ["&&", /&&/],
    ["||", /\|\|/],
    [";", /;/],
    ["|", /\|/],
    ["`", /`/],
    ["$(", /\$\(/],
    [">", />/],
    ["<", /</],
  ];
  for (const [token, pattern] of checks) {
    if (pattern.test(command)) return token;
  }
  return null;
}

function evidenceFailurePredicate(command: unknown, kind?: unknown): string {
  const signal = `${String(command ?? "")} ${String(kind ?? "")}`.toLowerCase();
  if (signal.includes("typecheck") || signal.includes("type check")) {
    return "typecheck_failed";
  }
  if (signal.includes("lint")) return "lint_failed";
  if (signal.includes("build")) return "build_failed";
  if (signal.includes("test")) return "test_failed";
  return "admission_failed";
}

function hasNonEmptyString(value: unknown): boolean {
  return typeof value === "string" && value.trim().length > 0;
}

export function hasScopeDeclared(artifacts: unknown[] | undefined): boolean {
  if (!artifacts || artifacts.length === 0) return false;
  for (const artifact of artifacts) {
    const a = artifact as Record<string, unknown> | undefined;
    if (!a) continue;
    const verifies = a.verifies as unknown[] | undefined;
    const doesNotVerify = a.does_not_verify as unknown[] | undefined;
    if (verifies && verifies.length > 0) return true;
    if (doesNotVerify && doesNotVerify.length > 0) return true;
  }
  return false;
}

function hasArtifactQuality(artifact: unknown): boolean {
  const a = artifact as Record<string, unknown> | undefined;
  if (!a) return false;
  const hasCommand = typeof a.command === "string" && a.command.length > 0;
  const hasQualityField =
    typeof a.exit_code === "number" ||
    typeof a.started_at === "string" ||
    typeof a.ended_at === "string" ||
    typeof a.stdout_hash === "string" ||
    typeof a.stderr_hash === "string" ||
    typeof a.artifact_path === "string" ||
    typeof a.artifact_hash === "string" ||
    typeof a.ci_run_url === "string";
  return hasCommand && hasQualityField;
}

function collectEvidenceQualityErrors(
  input: AdmissionInput
): AdmissionFinding[] {
  const errors: AdmissionFinding[] = [];
  const evidence = input.evidence;
  if (!evidence) return errors;

  const commandEvidence = evidence.command_evidence;
  if (Array.isArray(commandEvidence)) {
    for (const item of commandEvidence) {
      if (!item || typeof item !== "object") continue;
      const record = item as Record<string, unknown>;
      const command =
        typeof record.command === "string" ? record.command.trim() : "";
      const exitCode = record.exit_code;
      if (typeof exitCode === "number" && exitCode !== 0) {
        errors.push({
          message: `evidence.command_evidence has non-zero exit_code ${exitCode}${command ? ` for command "${command}"` : ""}`,
          predicate: evidenceFailurePredicate(command),
        });
      }
      if (command) {
        const token = shellMetacharacter(command);
        if (token) {
          errors.push({
            message: `evidence.command_evidence command contains denied shell metacharacter ${token}: "${command}"`,
            predicate: "Fpermission",
          });
        }
      }
    }
  }

  const verificationArtifacts = evidence.verification_artifacts;
  if (Array.isArray(verificationArtifacts)) {
    for (const item of verificationArtifacts) {
      if (!item || typeof item !== "object") continue;
      const artifact = item as Record<string, unknown>;
      const command =
        typeof artifact.command === "string" ? artifact.command.trim() : "";
      const kind = artifact.kind;
      const status = artifact.status;
      if (typeof status === "string" && status !== "passed") {
        errors.push({
          message: `evidence.verification_artifacts status "${status}" is not passed${command ? ` for command "${command}"` : ""}`,
          predicate: evidenceFailurePredicate(command, kind),
        });
      }
      const exitCode = artifact.exit_code;
      if (typeof exitCode === "number" && exitCode !== 0) {
        errors.push({
          message: `evidence.verification_artifacts has non-zero exit_code ${exitCode}${command ? ` for command "${command}"` : ""}`,
          predicate: evidenceFailurePredicate(command, kind),
        });
      }
      if (command) {
        const token = shellMetacharacter(command);
        if (token) {
          errors.push({
            message: `evidence.verification_artifacts command contains denied shell metacharacter ${token}: "${command}"`,
            predicate: "Fpermission",
          });
        }
      }
    }
  }

  return errors;
}

function collectStrictProvenanceErrors(
  input: AdmissionInput
): AdmissionFinding[] {
  const errors: AdmissionFinding[] = [];
  if (input.strict !== true) return errors;
  if (input.tier !== "standard" && input.tier !== "deep") return errors;

  const evidence = getEvidenceRecord(input);
  if (!evidence) return errors;

  const commandEvidence = evidence.command_evidence;
  if (Array.isArray(commandEvidence)) {
    commandEvidence.forEach((item, index) => {
      if (!item || typeof item !== "object") {
        errors.push({
          message: `strict evidence provenance requires evidence.command_evidence[${index}] to be an object`,
          predicate: "evidence_provenance_missing",
        });
        return;
      }
      const record = item as Record<string, unknown>;
      if (!hasNonEmptyString(record.command)) {
        errors.push({
          message: `strict evidence provenance requires evidence.command_evidence[${index}].command`,
          predicate: "evidence_provenance_missing",
        });
      }
      if (typeof record.exit_code !== "number") {
        errors.push({
          message: `strict evidence provenance requires evidence.command_evidence[${index}].exit_code`,
          predicate: "evidence_provenance_missing",
        });
      }
      if (!hasNonEmptyString(record.runner)) {
        errors.push({
          message: `strict evidence provenance requires evidence.command_evidence[${index}].runner`,
          predicate: "evidence_provenance_missing",
        });
      }
      if (!hasNonEmptyString(record.started_at)) {
        errors.push({
          message: `strict evidence provenance requires evidence.command_evidence[${index}].started_at`,
          predicate: "evidence_provenance_missing",
        });
      }
    });
  }

  const verificationArtifacts = evidence.verification_artifacts;
  if (Array.isArray(verificationArtifacts)) {
    verificationArtifacts.forEach((item, index) => {
      if (!item || typeof item !== "object") {
        errors.push({
          message: `strict evidence provenance requires evidence.verification_artifacts[${index}] to be an object`,
          predicate: "evidence_provenance_missing",
        });
        return;
      }
      const artifact = item as Record<string, unknown>;
      if (!hasNonEmptyString(artifact.command)) {
        errors.push({
          message: `strict evidence provenance requires evidence.verification_artifacts[${index}].command`,
          predicate: "evidence_provenance_missing",
        });
      }
      if (typeof artifact.exit_code !== "number") {
        errors.push({
          message: `strict evidence provenance requires evidence.verification_artifacts[${index}].exit_code`,
          predicate: "evidence_provenance_missing",
        });
      }
      if (!hasNonEmptyString(artifact.runner)) {
        errors.push({
          message: `strict evidence provenance requires evidence.verification_artifacts[${index}].runner`,
          predicate: "evidence_provenance_missing",
        });
      }
      if (!hasNonEmptyString(artifact.started_at)) {
        errors.push({
          message: `strict evidence provenance requires evidence.verification_artifacts[${index}].started_at`,
          predicate: "evidence_provenance_missing",
        });
      }
    });
  }

  return errors;
}

function isHighRiskFilePath(path: string): boolean {
  const lower = path.toLowerCase();
  return (
    lower.includes("schema") ||
    lower.includes("policy") ||
    lower.includes("admission") ||
    lower.includes("permission") ||
    lower.includes("authority") ||
    lower.includes(".github/workflows") ||
    lower.includes("ci/") ||
    lower.includes("/ci/")
  );
}

export function evaluateTierGuard(
  input: AdmissionInput
): AdmissionEvidenceEvaluation {
  const errors: AdmissionFinding[] = [];
  const notes: string[] = [];
  const tier = input.tier;
  if (!tier) {
    return { errors, notes };
  }

  const filesChanged = getFilesChanged(input) ?? [];
  const highRiskFiles: string[] = [];
  for (const item of filesChanged) {
    if (typeof item === "string" && isHighRiskFilePath(item)) {
      highRiskFiles.push(item);
    }
  }

  const highRiskCommands: string[] = [];
  const commandEvidence = getCommandEvidence(input) ?? [];
  for (const item of commandEvidence) {
    if (!item || typeof item !== "object") continue;
    const record = item as Record<string, unknown>;
    const cmd =
      typeof record.command === "string" ? record.command.trim() : "";
    if (cmd) {
      const classification = classifyCommand(cmd);
      if (classification.risk === "high" || classification.unknown) {
        highRiskCommands.push(cmd);
      }
    }
  }
  const verificationArtifacts = getVerificationArtifacts(input) ?? [];
  for (const item of verificationArtifacts) {
    if (!item || typeof item !== "object") continue;
    const record = item as Record<string, unknown>;
    const cmd =
      typeof record.command === "string" ? record.command.trim() : "";
    if (cmd) {
      const classification = classifyCommand(cmd);
      if (classification.risk === "high" || classification.unknown) {
        highRiskCommands.push(cmd);
      }
    }
  }

  if (tier === "light") {
    if (highRiskFiles.length > 0) {
      errors.push({
        message: `tier guard: light tier declared but high-risk files detected (${highRiskFiles.join(", ")}); consider standard or deep`,
        predicate: "admission_failed",
      });
    }
    if (highRiskCommands.length > 0) {
      notes.push(
        `tier guard warning: light tier with high-risk command(s) (${highRiskCommands.join(", ")}); consider raising tier`
      );
    }
  }

  if (
    tier === "standard" &&
    highRiskFiles.length > 0 &&
    highRiskCommands.length > 0
  ) {
    notes.push(
      `tier guard warning: standard tier with both high-risk files (${highRiskFiles.join(", ")}) and high-risk commands (${highRiskCommands.join(", ")}); consider deep`
    );
  }

  return { errors, notes };
}

export function evaluateEvidenceRules(
  input: AdmissionInput
): AdmissionEvidenceEvaluation {
  const errors: AdmissionFinding[] = [];
  const notes: string[] = [];
  const evidenceArray = getEvidenceArray(input);
  const evidenceRecord = getEvidenceRecord(input);
  const filesChanged = getFilesChanged(input);
  const commandEvidence = getCommandEvidence(input);
  const manualRationale = getManualRationale(input);
  const verificationArtifacts = getVerificationArtifacts(input);
  const untestedRegions = getUntestedRegions(input);
  const remainingRisks = getRemainingRisks(input);
  const rollbackPolicy = getRollbackPolicy(input);
  const executionControls = getExecutionControls(input);
  const state = getState(input);

  if (evidenceArray !== undefined) {
    if (evidenceArray.length === 0) {
      errors.push({
        message: "claim.evidence is required and must be non-empty",
        predicate: "evidence_missing",
      });
    }
  } else {
    if (evidenceRecord) {
      const ev = evidenceRecord;
      if (!ev.owner && !ev.accountable) {
        notes.push("evidence packet lacks owner/accountable fields");
      }
    }
    if (input.tier && input.tier !== "light" && !evidenceRecord) {
      errors.push({
        message: `tier "${input.tier}" requires evidence packet`,
        predicate: "evidence_missing",
      });
    }
  }

  if (!filesChanged || filesChanged.length === 0) {
    errors.push({
      message: "evidence.files_changed is required and must be non-empty",
      predicate: "evidence_missing",
    });
  }

  if (input.tier === "light") {
    const hasCommandEvidence = commandEvidence && commandEvidence.length > 0;
    const hasManualRationale =
      manualRationale && manualRationale.trim().length > 0;
    if (!hasCommandEvidence && !hasManualRationale) {
      errors.push({
        message:
          'tier "light" requires evidence.command_evidence or evidence.manual_rationale',
        predicate: "evidence_floor_not_met",
      });
    }
  }

  if (input.tier === "deep") {
    if (!commandEvidence || commandEvidence.length === 0) {
      errors.push({
        message: 'tier "deep" requires evidence.command_evidence',
        predicate: "evidence_floor_not_met",
      });
    }
    if (!verificationArtifacts || verificationArtifacts.length === 0) {
      errors.push({
        message: 'tier "deep" requires verification_artifacts',
        predicate: "evidence_scope_missing",
      });
    }
    if (!hasScopeDeclared(verificationArtifacts)) {
      errors.push({
        message:
          'tier "deep" requires evidence scope declared (verifies/does_not_verify)',
        predicate: "evidence_scope_missing",
      });
    }
    if (!untestedRegions || untestedRegions.length === 0) {
      errors.push({
        message: 'tier "deep" requires untested_regions',
        predicate: "evidence_scope_missing",
      });
    }
    if (!remainingRisks || remainingRisks.length === 0) {
      errors.push({
        message: 'tier "deep" requires remaining_risks',
        predicate: "evidence_scope_missing",
      });
    }
    if (!rollbackPolicy || rollbackPolicy.length === 0) {
      errors.push({
        message: 'tier "deep" requires rollback_policy',
        predicate: "evidence_scope_missing",
      });
    }
    if (!executionControls || executionControls.length === 0) {
      errors.push({
        message: 'tier "deep" requires execution_controls',
        predicate: "evidence_scope_missing",
      });
    }
    if (
      !state ||
      !Array.isArray(state.write_set) ||
      (state.write_set as unknown[]).length === 0
    ) {
      errors.push({
        message: 'tier "deep" requires state.write_set',
        predicate: "state_read_write_missing",
      });
    }
    if (
      !state ||
      !Array.isArray(state.read_set) ||
      (state.read_set as unknown[]).length === 0
    ) {
      errors.push({
        message: 'tier "deep" requires state.read_set',
        predicate: "state_read_write_missing",
      });
    }
    if (verificationArtifacts && verificationArtifacts.length > 0) {
      const lowQuality = verificationArtifacts.filter(
        (a) => !hasArtifactQuality(a)
      );
      if (lowQuality.length > 0) {
        notes.push(
          'tier "deep" recommends artifact metadata (command, timestamps, hashes, ci_run_url) for stronger traceability'
        );
      }
    }
  }

  if (input.tier === "standard") {
    if (!commandEvidence || commandEvidence.length === 0) {
      errors.push({
        message: 'tier "standard" requires evidence.command_evidence',
        predicate: "evidence_floor_not_met",
      });
    }
    if (!verificationArtifacts || verificationArtifacts.length === 0) {
      notes.push('tier "standard" recommends verification_artifacts');
    } else {
      const lowQuality = verificationArtifacts.filter(
        (a) => !hasArtifactQuality(a)
      );
      if (lowQuality.length > 0) {
        notes.push(
          'tier "standard" recommends artifact metadata (command, timestamps, hashes, ci_run_url) for stronger traceability'
        );
      }
    }
    if (!hasScopeDeclared(verificationArtifacts)) {
      notes.push(
        'tier "standard" recommends evidence scope (verifies/does_not_verify)'
      );
    }
    if (!untestedRegions || untestedRegions.length === 0) {
      notes.push('tier "standard" recommends untested_regions');
    }
  }

  errors.push(...collectEvidenceQualityErrors(input));
  errors.push(...collectStrictProvenanceErrors(input));
  return { errors, notes };
}
