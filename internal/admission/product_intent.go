package admission

import "strings"

// productIntentAdvisoryNotes holds the canonical advisory note text for
// product_intent.status. Centralized to keep wording parity-safe with the TS
// implementation in packages/cli/src/core/admission.ts and the policy
// documentation in policies/admission.yaml.
const (
	productIntentMissingNote = "product_intent.status not declared (advisory-only; admission acceptance is not product correctness)"
	productIntentUnknownNote = "product_intent.status is unknown (advisory-only; admission acceptance is not product correctness)"
)

// evaluateProductIntent emits advisory notes (never errors) for standard and
// deep tier cards when product_intent.status is missing or set to "unknown".
// The light tier remains quiet. aligned/unreviewed/disputed/not_applicable do
// not produce any advisory note. This is the first vertical slice; it never
// blocks admission.
func evaluateProductIntent(doc map[string]any, tier string) []string {
	notes := make([]string, 0)
	if tier != "standard" && tier != "deep" {
		return notes
	}

	productIntent := mapValue(doc, "product_intent")
	if productIntent == nil {
		notes = append(notes, productIntentMissingNote)
		return notes
	}

	status := strings.TrimSpace(stringInMap(productIntent, "status"))
	if status == "" {
		notes = append(notes, productIntentMissingNote)
		return notes
	}
	if status == "unknown" {
		notes = append(notes, productIntentUnknownNote)
	}
	return notes
}
