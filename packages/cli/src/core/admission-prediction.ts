import type { AdmissionInput } from "./admission.js";
import {
  getCommandEvidence,
  getDoneChecklist,
  getEvidenceRecord,
  getFilesChanged,
  getManualRationale,
  getPrediction,
  getRemainingRisks,
  getRollbackPolicy,
  getState,
  getUntestedRegions,
  getVerificationArtifacts,
} from "./admission-accessors.js";
import { hasScopeDeclared } from "./admission-evidence.js";
import type { AdmissionFinding } from "./admission-evidence.js";

export interface PredictionEvaluation {
  errors: Array<AdmissionFinding & { forcePredicate?: boolean }>;
}

function isNonEmptyArray(value: unknown): value is unknown[] {
  return Array.isArray(value) && value.length > 0;
}

function hasDoesNotVerify(artifacts: unknown[] | undefined): boolean {
  if (!artifacts) return false;
  return artifacts.some((artifact) => {
    const record = artifact as Record<string, unknown> | undefined;
    return isNonEmptyArray(record?.does_not_verify);
  });
}

function pushChecklistMismatch(
  errors: Array<AdmissionFinding & { forcePredicate?: boolean }>,
  message: string
): void {
  errors.push({
    message,
    predicate: "done_checklist_mismatch",
  });
}

function collectChecklistHonestyErrors(
  input: AdmissionInput,
  errors: Array<AdmissionFinding & { forcePredicate?: boolean }>
): void {
  const doneChecklist = getDoneChecklist(input);
  if (!doneChecklist || Object.keys(doneChecklist).length === 0) return;

  const evidenceRecord = getEvidenceRecord(input);
  const filesChanged = getFilesChanged(input);
  const commandEvidence = getCommandEvidence(input);
  const manualRationale = getManualRationale(input);
  const verificationArtifacts = getVerificationArtifacts(input);
  const state = getState(input);

  if (doneChecklist.evidence_attached === true) {
    if (!evidenceRecord) {
      pushChecklistMismatch(
        errors,
        "done_checklist.evidence_attached is true but evidence is missing"
      );
    } else if (!isNonEmptyArray(filesChanged)) {
      pushChecklistMismatch(
        errors,
        "done_checklist.evidence_attached is true but evidence.files_changed is missing"
      );
    } else if (
      !isNonEmptyArray(commandEvidence) &&
      !isNonEmptyArray(verificationArtifacts) &&
      !manualRationale
    ) {
      pushChecklistMismatch(
        errors,
        "done_checklist.evidence_attached is true but no command_evidence, verification_artifacts, or manual_rationale is present"
      );
    }
  }

  if (doneChecklist.evidence_attached === false && evidenceRecord) {
    pushChecklistMismatch(
      errors,
      "done_checklist.evidence_attached is false but evidence is present"
    );
  }

  if (
    doneChecklist.read_write_sets_declared === true &&
    !state &&
    (input.strict === true || input.tier === "deep")
  ) {
    pushChecklistMismatch(
      errors,
      "done_checklist.read_write_sets_declared is true but state is missing"
    );
  }

  if (doneChecklist.read_write_sets_declared === true && state) {
    if (!isNonEmptyArray(state.read_set)) {
      pushChecklistMismatch(
        errors,
        "done_checklist.read_write_sets_declared is true but state.read_set is missing"
      );
    }
    if (!isNonEmptyArray(state.write_set)) {
      pushChecklistMismatch(
        errors,
        "done_checklist.read_write_sets_declared is true but state.write_set is missing"
      );
    }
  }

  if (
    doneChecklist.read_write_sets_declared === false &&
    state &&
    (isNonEmptyArray(state.read_set) || isNonEmptyArray(state.write_set))
  ) {
    pushChecklistMismatch(
      errors,
      "done_checklist.read_write_sets_declared is false but state read/write sets are present"
    );
  }

  if (
    (input.strict === true || input.tier === "deep") &&
    doneChecklist.scope_explained === true &&
    isNonEmptyArray(verificationArtifacts) &&
    !hasScopeDeclared(verificationArtifacts)
  ) {
    pushChecklistMismatch(
      errors,
      "done_checklist.scope_explained is true but verification_artifacts lacks verifies/does_not_verify scope"
    );
  }

  if (
    (input.strict === true || input.tier === "deep") &&
    doneChecklist.coverage_gap_declared === true &&
    isNonEmptyArray(verificationArtifacts) &&
    !isNonEmptyArray(getUntestedRegions(input)) &&
    !hasDoesNotVerify(verificationArtifacts)
  ) {
    pushChecklistMismatch(
      errors,
      "done_checklist.coverage_gap_declared is true but no untested_regions or artifact does_not_verify scope is present"
    );
  }

  if (
    input.tier === "deep" &&
    doneChecklist.risk_and_rollback_declared === true
  ) {
    if (!isNonEmptyArray(getRemainingRisks(input))) {
      pushChecklistMismatch(
        errors,
        "done_checklist.risk_and_rollback_declared is true but evidence.remaining_risks is missing"
      );
    }
    if (!isNonEmptyArray(getRollbackPolicy(input))) {
      pushChecklistMismatch(
        errors,
        "done_checklist.risk_and_rollback_declared is true but evidence.rollback_policy is missing"
      );
    }
  }
}

export function evaluateDoneChecklistAndPrediction(
  input: AdmissionInput
): PredictionEvaluation {
  const errors: Array<AdmissionFinding & { forcePredicate?: boolean }> = [];
  const requiresChecklistAndPrediction =
    input.tier === "standard" || input.tier === "deep";

  const doneChecklist = getDoneChecklist(input);
  const prediction = getPrediction(input);

  if (
    requiresChecklistAndPrediction &&
    (!doneChecklist || Object.keys(doneChecklist).length === 0)
  ) {
    errors.push({
      message: `tier "${input.tier}" requires done_checklist`,
      predicate: "done_checklist_missing",
    });
  }

  if (
    requiresChecklistAndPrediction &&
    (!prediction || Object.keys(prediction).length === 0)
  ) {
    errors.push({
      message: `tier "${input.tier}" requires prediction`,
      predicate: "prediction_missing",
    });
  } else if (prediction && Object.keys(prediction).length > 0) {
    const predClaim = prediction.claim as string | undefined;
    const predExpectedEffect = prediction.expected_effect as string | undefined;
    const predFalsificationMethod = prediction.falsification_method as
      | string
      | undefined;
    const predHorizon = prediction.horizon as string | undefined;

    if (
      !predClaim ||
      typeof predClaim !== "string" ||
      predClaim.trim().length === 0
    ) {
      errors.push({
        message: "prediction.claim is required and must be non-empty string",
        predicate: "prediction_invalid",
      });
    }
    if (
      !predExpectedEffect ||
      typeof predExpectedEffect !== "string" ||
      predExpectedEffect.trim().length === 0
    ) {
      errors.push({
        message:
          "prediction.expected_effect is required and must be non-empty string",
        predicate: "prediction_invalid",
      });
    }
    if (
      !predFalsificationMethod ||
      typeof predFalsificationMethod !== "string" ||
      predFalsificationMethod.trim().length === 0
    ) {
      errors.push({
        message:
          "prediction.falsification_method is required and must be non-empty string",
        predicate: "prediction_invalid",
      });
    }
    if (!predHorizon) {
      errors.push({
        message: "prediction.horizon is required",
        predicate: "prediction_invalid",
      });
    }
  }

  if (
    doneChecklist?.prediction_declared === true &&
    (!prediction || Object.keys(prediction).length === 0)
  ) {
    errors.push({
      message:
        "done_checklist.prediction_declared is true but prediction is missing",
      predicate: "done_checklist_prediction_mismatch",
      forcePredicate: true,
    });
  }

  if (
    doneChecklist?.prediction_declared === false &&
    prediction &&
    Object.keys(prediction).length > 0
  ) {
    pushChecklistMismatch(
      errors,
      "done_checklist.prediction_declared is false but prediction is present"
    );
  }

  collectChecklistHonestyErrors(input, errors);

  return { errors };
}
