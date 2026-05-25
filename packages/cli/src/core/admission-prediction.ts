import type { AdmissionInput } from "./admission.js";
import { getDoneChecklist, getPrediction } from "./admission-accessors.js";
import type { AdmissionFinding } from "./admission-evidence.js";

export interface PredictionEvaluation {
  errors: Array<AdmissionFinding & { forcePredicate?: boolean }>;
}

export function evaluateDoneChecklistAndPrediction(
  input: AdmissionInput
): PredictionEvaluation {
  const errors: Array<AdmissionFinding & { forcePredicate?: boolean }> = [];
  if (input.tier !== "standard" && input.tier !== "deep") {
    return { errors };
  }

  const doneChecklist = getDoneChecklist(input);
  const prediction = getPrediction(input);

  if (!doneChecklist || Object.keys(doneChecklist).length === 0) {
    errors.push({
      message: `tier "${input.tier}" requires done_checklist`,
      predicate: "done_checklist_missing",
    });
  }

  if (!prediction || Object.keys(prediction).length === 0) {
    errors.push({
      message: `tier "${input.tier}" requires prediction`,
      predicate: "prediction_missing",
    });
  } else {
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

  return { errors };
}
