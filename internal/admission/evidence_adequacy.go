package admission

import "strings"

// evidenceAdequacyAdvisoryNotes holds the canonical advisory note text for
// evidence_adequacy. Centralized to keep wording parity-safe with the TS
// implementation in packages/cli/src/core/admission.ts and the policy
// documentation in policies/admission.yaml.
const (
	evidenceAdequacyMissingNote   = "evidence_adequacy not declared (advisory-only; admission acceptance is not evidence adequacy)"
	evidenceAdequacySummaryMissing = "evidence_adequacy.summary not declared (advisory-only; consider explaining how evidence covers the change)"
)

// evaluateEvidenceAdequacy emits advisory notes (never errors) for standard
// and deep tier cards when evidence_adequacy is missing or when its summary
// is missing/blank. The light tier remains quiet. A non-blank summary
// suppresses the summary note but does not gate the missing-object note.
// This is the first vertical slice; it never blocks admission. Wording is
// parity-safe with the TS implementation in packages/cli/src/core/admission.ts
// and the policy documentation in policies/admission.yaml.
func evaluateEvidenceAdequacy(doc map[string]any, tier string) []string {
	notes := make([]string, 0)
	if tier != "standard" && tier != "deep" {
		return notes
	}

	evidenceAdequacy := mapValue(doc, "evidence_adequacy")
	if evidenceAdequacy == nil {
		notes = append(notes, evidenceAdequacyMissingNote)
		return notes
	}

	// summary: missing or blank -> note
	summary := strings.TrimSpace(stringInMap(evidenceAdequacy, "summary"))
	if summary == "" {
		notes = append(notes, evidenceAdequacySummaryMissing)
	}

	return notes
}
