package admission

import "fmt"

func evaluateDoneChecklistAndPrediction(doc map[string]any, strict bool, tier string) evidenceResult {
	var result evidenceResult
	requires := tier == "standard" || tier == "deep"

	doneChecklist := mapValue(doc, "done_checklist")
	prediction := mapValue(doc, "prediction")

	if requires {
		if doneChecklist == nil || len(doneChecklist) == 0 {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf(`tier "%s" requires done_checklist`, tier),
				predicate: "done_checklist_missing",
			})
		}
		if prediction == nil || len(prediction) == 0 {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf(`tier "%s" requires prediction`, tier),
				predicate: "prediction_missing",
			})
		} else {
			if s := stringInMap(prediction, "claim"); s == "" {
				result.errors = append(result.errors, evidenceFinding{
					message:   "prediction.claim is required and must be non-empty string",
					predicate: "prediction_invalid",
				})
			}
			if s := stringInMap(prediction, "expected_effect"); s == "" {
				result.errors = append(result.errors, evidenceFinding{
					message:   "prediction.expected_effect is required and must be non-empty string",
					predicate: "prediction_invalid",
				})
			}
			if s := stringInMap(prediction, "falsification_method"); s == "" {
				result.errors = append(result.errors, evidenceFinding{
					message:   "prediction.falsification_method is required and must be non-empty string",
					predicate: "prediction_invalid",
				})
			}
			if _, ok := prediction["horizon"]; !ok {
				result.errors = append(result.errors, evidenceFinding{
					message:   "prediction.horizon is required",
					predicate: "prediction_invalid",
				})
			} else if s := stringInMap(prediction, "horizon"); s == "" {
				result.errors = append(result.errors, evidenceFinding{
					message:   "prediction.horizon is required",
					predicate: "prediction_invalid",
				})
			}
		}
	}

	if doneChecklist != nil && len(doneChecklist) > 0 {
		// prediction_declared mismatch
		predDeclared := boolInMap(doneChecklist, "prediction_declared")
		if predDeclared && (prediction == nil || len(prediction) == 0) {
			result.errors = append(result.errors, evidenceFinding{
				message:   "done_checklist.prediction_declared is true but prediction is missing",
				predicate: "done_checklist_prediction_mismatch",
			})
		}
		if !predDeclared && prediction != nil && len(prediction) > 0 {
			result.errors = append(result.errors, evidenceFinding{
				message:   "done_checklist.prediction_declared is false but prediction is present",
				predicate: "done_checklist_mismatch",
			})
		}

		// evidence_attached honesty
		if boolInMap(doneChecklist, "evidence_attached") {
			evidence := evidenceRecord(doc)
			if evidence == nil {
				result.errors = append(result.errors, evidenceFinding{
					message:   "done_checklist.evidence_attached is true but evidence is missing",
					predicate: "done_checklist_mismatch",
				})
			} else {
				filesChanged := sliceInMap(evidence, "files_changed")
				if len(filesChanged) == 0 {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.evidence_attached is true but evidence.files_changed is missing",
						predicate: "done_checklist_mismatch",
					})
				} else {
					commandEvidence := sliceInMap(evidence, "command_evidence")
					verificationArtifacts := sliceInMap(evidence, "verification_artifacts")
					manualRationale := stringInMap(evidence, "manual_rationale")
					if len(commandEvidence) == 0 && len(verificationArtifacts) == 0 && manualRationale == "" {
						result.errors = append(result.errors, evidenceFinding{
							message:   "done_checklist.evidence_attached is true but no command_evidence, verification_artifacts, or manual_rationale is present",
							predicate: "done_checklist_mismatch",
						})
					}
				}
			}
		}
		if boolExplicitlyFalse(doneChecklist, "evidence_attached") {
			if evidence := evidenceRecord(doc); evidence != nil {
				result.errors = append(result.errors, evidenceFinding{
					message:   "done_checklist.evidence_attached is false but evidence is present",
					predicate: "done_checklist_mismatch",
				})
			}
		}

		// read_write_sets_declared honesty
		if boolInMap(doneChecklist, "read_write_sets_declared") {
			state := mapValue(doc, "state")
			if state == nil && (strict || tier == "deep") {
				result.errors = append(result.errors, evidenceFinding{
					message:   "done_checklist.read_write_sets_declared is true but state is missing",
					predicate: "done_checklist_mismatch",
				})
			}
			if state != nil {
				if len(sliceInMap(state, "read_set")) == 0 {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.read_write_sets_declared is true but state.read_set is missing",
						predicate: "done_checklist_mismatch",
					})
				}
				if len(sliceInMap(state, "write_set")) == 0 {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.read_write_sets_declared is true but state.write_set is missing",
						predicate: "done_checklist_mismatch",
					})
				}
			}
		}
		if boolExplicitlyFalse(doneChecklist, "read_write_sets_declared") {
			state := mapValue(doc, "state")
			if state != nil {
				if len(sliceInMap(state, "read_set")) > 0 || len(sliceInMap(state, "write_set")) > 0 {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.read_write_sets_declared is false but state read/write sets are present",
						predicate: "done_checklist_mismatch",
					})
				}
			}
		}

		// scope_explained honesty (strict or deep)
		if (strict || tier == "deep") && boolInMap(doneChecklist, "scope_explained") {
			evidence := evidenceRecord(doc)
			if evidence != nil {
				artifacts := sliceInMap(evidence, "verification_artifacts")
				if len(artifacts) > 0 && !verificationArtifactsHaveScope(artifacts) {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.scope_explained is true but verification_artifacts lacks verifies/does_not_verify scope",
						predicate: "done_checklist_mismatch",
					})
				}
			}
		}

		// coverage_gap_declared honesty (strict or deep)
		if (strict || tier == "deep") && boolInMap(doneChecklist, "coverage_gap_declared") {
			evidence := evidenceRecord(doc)
			if evidence != nil {
				artifacts := sliceInMap(evidence, "verification_artifacts")
				untestedRegions := sliceInMap(evidence, "untested_regions")
				hasDoesNotVerify := false
				if len(artifacts) > 0 {
					for _, item := range artifacts {
						artifact, ok := item.(map[string]any)
						if !ok {
							continue
						}
						if len(sliceInMap(artifact, "does_not_verify")) > 0 {
							hasDoesNotVerify = true
							break
						}
					}
				}
				if len(artifacts) > 0 && len(untestedRegions) == 0 && !hasDoesNotVerify {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.coverage_gap_declared is true but no untested_regions or artifact does_not_verify scope is present",
						predicate: "done_checklist_mismatch",
					})
				}
			}
		}

		// risk_and_rollback_declared honesty (deep only)
		if tier == "deep" && boolInMap(doneChecklist, "risk_and_rollback_declared") {
			evidence := evidenceRecord(doc)
			if evidence != nil {
				if len(sliceInMap(evidence, "remaining_risks")) == 0 {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.risk_and_rollback_declared is true but evidence.remaining_risks is missing",
						predicate: "done_checklist_mismatch",
					})
				}
				if len(sliceInMap(evidence, "rollback_policy")) == 0 {
					result.errors = append(result.errors, evidenceFinding{
						message:   "done_checklist.risk_and_rollback_declared is true but evidence.rollback_policy is missing",
						predicate: "done_checklist_mismatch",
					})
				}
			}
		}
	}

	return result
}
