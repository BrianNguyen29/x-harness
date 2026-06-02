package admission

import "strings"

// intentContractAdvisoryNotes holds the canonical advisory note text for
// intent_contract. Centralized to keep wording parity-safe with the TS
// implementation in packages/cli/src/core/admission.ts and the policy
// documentation in policies/admission.yaml.
const (
	intentContractMissingNote      = "intent_contract not declared (advisory-only; admission acceptance is not intent correctness)"
	intentContractGoalMissingNote   = "intent_contract.product_goal not declared (advisory-only; consider documenting the intended change goal)"
	intentContractUvChangeNote      = "intent_contract.user_visible_change not declared (advisory-only; consider declaring whether the change is user-visible)"
)

// evaluateIntentContract emits advisory notes (never errors) for standard
// and deep tier cards when intent_contract is missing, when its product_goal
// is missing/blank, or when its user_visible_change key is absent. The
// light tier remains quiet. user_visible_change == false (an explicit
// non-user-visible declaration) is accepted and produces no uvchange note.
// This is the first vertical slice; it never blocks admission. Wording is
// parity-safe with the TS implementation in packages/cli/src/core/admission.ts
// and the policy documentation in policies/admission.yaml.
func evaluateIntentContract(doc map[string]any, tier string) []string {
	notes := make([]string, 0)
	if tier != "standard" && tier != "deep" {
		return notes
	}

	intentContract := mapValue(doc, "intent_contract")
	if intentContract == nil {
		notes = append(notes, intentContractMissingNote)
		return notes
	}

	// product_goal: missing or blank -> note
	goal := strings.TrimSpace(stringInMap(intentContract, "product_goal"))
	if goal == "" {
		notes = append(notes, intentContractGoalMissingNote)
	}

	// user_visible_change: key absent -> note. An explicit `false` value
	// is a valid "not user-visible" declaration and produces no note.
	if !hasKey(intentContract, "user_visible_change") {
		notes = append(notes, intentContractUvChangeNote)
	}

	return notes
}
