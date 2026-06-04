package admission

import "strings"

// decisionRefsAdvisoryNotes holds the canonical advisory note text for
// context_alignment.decision_refs. Centralized to keep wording parity-safe
// with the TS implementation in packages/cli/src/core/admission.ts and the
// policy documentation in policies/admission.yaml.
const (
	// decisionRefsEmptyNote is emitted when the optional
	// context_alignment.decision_refs array is missing or contains no
	// non-blank string entries on a standard/deep tier card. Light tier
	// stays quiet. This is the first safe-V1 vertical slice; it never
	// blocks admission.
	decisionRefsEmptyNote = "context_alignment.decision_refs is empty (advisory-only; admission acceptance is not decision correctness)"
)

// evaluateDecisionRefs emits advisory notes (never errors) for standard and
// deep tier cards when the optional context_alignment.decision_refs array
// is missing or contains no non-blank string entries. A non-blank value
// (whether it is a slug id, a path, or a URI fragment) suppresses the note.
// The light tier remains quiet. This is the first safe-V1 vertical slice;
// it never blocks admission. Wording is parity-safe with the TS
// implementation in packages/cli/src/core/admission.ts and the policy
// documentation in policies/admission.yaml.
func evaluateDecisionRefs(doc map[string]any, tier string) []string {
	notes := make([]string, 0)
	if tier != "standard" && tier != "deep" {
		return notes
	}

	if !hasAnyDecisionRef(doc) {
		notes = append(notes, decisionRefsEmptyNote)
	}
	return notes
}

// hasAnyDecisionRef reports whether doc.context_alignment.decision_refs is
// a non-empty array containing at least one non-blank string. The check is
// performed purely for advisory purposes and intentionally tolerates
// nil/missing/non-array values, which all map to "empty" and trigger the
// note.
func hasAnyDecisionRef(doc map[string]any) bool {
	ctxAlign := mapValue(doc, "context_alignment")
	if ctxAlign == nil {
		return false
	}
	raw, ok := ctxAlign["decision_refs"]
	if !ok {
		return false
	}
	arr, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			continue
		}
		if strings.TrimSpace(s) != "" {
			return true
		}
	}
	return false
}
