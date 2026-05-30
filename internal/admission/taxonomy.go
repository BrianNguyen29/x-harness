package admission

func buildTaxonomy(predicate string) *FailureTaxonomy {
	switch predicate {
	case "stale_ground":
		return &FailureTaxonomy{
			FailureClass:   "stale_context",
			FailureStage:   "admission_gate",
			Recoverability: "retry_after_refresh",
			NextAction:     "review_and_resubmit",
		}
	case "approval_missing", "Fintervention":
		return &FailureTaxonomy{
			FailureClass:   "governance_missing",
			FailureStage:   "admission_gate",
			Recoverability: "human_intervention",
			NextAction:     "escalate",
		}
	case "classifier_approval_required":
		return &FailureTaxonomy{
			FailureClass:   "command_risky",
			FailureStage:   "admission_gate",
			Recoverability: "human_intervention",
			NextAction:     "request_approval",
		}
	case "evidence_provenance_missing":
		return &FailureTaxonomy{
			FailureClass:   "evidence_provenance_invalid",
			FailureStage:   "admission_gate",
			Recoverability: "retry_with_fixes",
			NextAction:     "review_and_resubmit",
		}
	case "context_floor_blocked":
		return &FailureTaxonomy{
			FailureClass:   "context_missing",
			FailureStage:   "context_floor",
			Recoverability: "retry_with_fixes",
			NextAction:     "review_and_resubmit",
		}
	case "contract_oracle_blocked":
		return &FailureTaxonomy{
			FailureClass:   "contract_mismatch",
			FailureStage:   "verify_pipeline",
			Recoverability: "retry_with_fixes",
			NextAction:     "review_and_resubmit",
		}
	default:
		return &FailureTaxonomy{
			FailureClass:   "schema_or_policy_invalid",
			FailureStage:   "admission_gate",
			Recoverability: "retry_with_fixes",
			NextAction:     "review_and_resubmit",
		}
	}
}
