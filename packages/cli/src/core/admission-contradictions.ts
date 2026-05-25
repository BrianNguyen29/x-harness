import type { AdmissionInput } from "./admission.js";
import {
  getAdmissionOutcome,
  getClaimFixStatus,
  getFixStatus,
  getHandoff,
  getSubagentFixStatus,
  getVerifyStatus,
  isNonSuccessStatus,
} from "./admission-accessors.js";
import type { AdmissionFinding } from "./admission-evidence.js";

export function collectFixStatusContradictions(
  input: AdmissionInput
): AdmissionFinding[] {
  const claimFixStatus = getClaimFixStatus(input);
  const subagentFixStatus = getSubagentFixStatus(input);
  if (
    claimFixStatus !== undefined &&
    subagentFixStatus !== undefined &&
    claimFixStatus !== subagentFixStatus
  ) {
    return [
      {
        message: `canonical contradiction: claim.fix_status is "${claimFixStatus}" but result.fix_status is "${subagentFixStatus}"`,
        predicate: "admission_failed",
      },
    ];
  }
  return [];
}

export function collectCanonicalStatusContradictions(
  input: AdmissionInput
): AdmissionFinding[] {
  const errors: AdmissionFinding[] = [];
  const fixStatus = getFixStatus(input);
  const verifyStatus = getVerifyStatus(input);
  const admissionOutcome = getAdmissionOutcome(input);
  const handoff = getHandoff(input);

  if (fixStatus === undefined) {
    errors.push({
      message: "claim.fix_status or result.fix_status is required",
      predicate: "admission_failed",
    });
  }

  if (verifyStatus === undefined) {
    errors.push({
      message: "verification.status is required",
      predicate: "admission_failed",
    });
  }

  if (verifyStatus !== undefined && fixStatus !== undefined) {
    if (verifyStatus === "passed" && fixStatus !== "fixed") {
      errors.push({
        message: `canonical contradiction: verification.status is "passed" but claim.fix_status is "${fixStatus}" (must be "fixed")`,
        predicate: "admission_failed",
      });
    }

    if (
      input.acceptance_status === "accepted" &&
      isNonSuccessStatus(verifyStatus)
    ) {
      errors.push({
        message: `canonical contradiction: acceptance_status is "accepted" but verification.status is "${verifyStatus}"`,
        predicate: "admission_failed",
      });
    }
  }

  if (admissionOutcome !== undefined) {
    if (
      admissionOutcome === "success" &&
      input.acceptance_status !== "accepted"
    ) {
      errors.push({
        message: `canonical contradiction: admission.outcome is "success" but acceptance_status is "${String(input.acceptance_status)}" (must be "accepted")`,
        predicate: "admission_failed",
      });
    }

    if (admissionOutcome === "success" && verifyStatus !== "passed") {
      errors.push({
        message: `success requires verification.status "passed" but found "${String(verifyStatus)}"`,
        predicate: "admission_failed",
      });
    }

    if (admissionOutcome === "success" && fixStatus !== "fixed") {
      errors.push({
        message: `success requires claim.fix_status "fixed" but found "${String(fixStatus)}"`,
        predicate: "admission_failed",
      });
    }

    if (
      input.acceptance_status === "accepted" &&
      admissionOutcome !== "success"
    ) {
      errors.push({
        message: `canonical contradiction: acceptance_status is "accepted" but admission.outcome is "${admissionOutcome}" (must be "success")`,
        predicate: "admission_failed",
      });
    }

    if (
      admissionOutcome !== "success" &&
      input.acceptance_status === "accepted"
    ) {
      errors.push({
        message: `canonical contradiction: non-success outcome "${admissionOutcome}" cannot have acceptance_status "accepted"`,
        predicate: "admission_failed",
      });
    }

    if (isNonSuccessStatus(admissionOutcome)) {
      if (
        !handoff ||
        !handoff.next_action ||
        String(handoff.next_action).trim().length === 0
      ) {
        errors.push({
          message: `admission.outcome "${admissionOutcome}" requires handoff.next_action`,
          predicate: "admission_failed",
        });
      }
      if (
        !handoff ||
        !handoff.owner ||
        String(handoff.owner).trim().length === 0
      ) {
        errors.push({
          message: `admission.outcome "${admissionOutcome}" requires handoff.owner`,
          predicate: "admission_failed",
        });
      }
    }
  }

  if (verifyStatus !== undefined && isNonSuccessStatus(verifyStatus)) {
    if (
      !handoff ||
      !handoff.next_action ||
      String(handoff.next_action).trim().length === 0
    ) {
      errors.push({
        message: `verification.status "${verifyStatus}" requires handoff.next_action`,
        predicate: "admission_failed",
      });
    }
    if (
      !handoff ||
      !handoff.owner ||
      String(handoff.owner).trim().length === 0
    ) {
      errors.push({
        message: `verification.status "${verifyStatus}" requires handoff.owner`,
        predicate: "admission_failed",
      });
    }
  }

  return errors;
}
