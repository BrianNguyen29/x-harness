package admission

import "strings"

// intentRefAdvisoryNotes holds the canonical advisory note text for
// intent_ref. Centralized to keep wording parity-safe with the TS
// implementation in packages/cli/src/core/admission.ts and the policy
// documentation in policies/admission.yaml.
const (
	// intentRefMissingNote is emitted when the optional top-level
	// intent_ref field is missing or blank on a standard/deep tier card.
	// Light tier stays quiet. This is the first safe-V1 vertical slice;
	// it never blocks admission.
	intentRefMissingNote = "intent_ref not declared (advisory-only; admission acceptance is not intent correctness)"
)

// evaluateIntentRef emits advisory notes (never errors) for standard and
// deep tier cards when the optional top-level intent_ref field is missing
// or blank. The light tier remains quiet. A non-blank intent_ref (whether
// it is a slug id, a path, or a URI fragment) suppresses the note. This
// is the first safe-V1 vertical slice; it never blocks admission. Wording
// is parity-safe with the TS implementation in
// packages/cli/src/core/admission.ts and the policy documentation in
// policies/admission.yaml.
func evaluateIntentRef(doc map[string]any, tier string) []string {
	notes := make([]string, 0)
	if tier != "standard" && tier != "deep" {
		return notes
	}

	ref := strings.TrimSpace(stringValue(doc, "intent_ref"))
	if ref == "" {
		notes = append(notes, intentRefMissingNote)
	}
	return notes
}
