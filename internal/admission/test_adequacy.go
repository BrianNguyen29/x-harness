package admission

import "strings"

// testAdequacyAdvisoryNotes holds the canonical advisory note text for
// test_adequacy. Centralized to keep wording parity-safe with the TS
// implementation in packages/cli/src/core/admission.ts and the policy
// documentation in policies/admission.yaml.
const (
	testAdequacyMissingNote         = "test_adequacy not declared (advisory-only; admission acceptance is not test adequacy)"
	testAdequacyBehaviorsMissingNote = "test_adequacy.impacted_behaviors not declared (advisory-only; consider listing behavior covered by tests)"
	testAdequacyTestsMissingNote     = "test_adequacy.tests_selected not declared (advisory-only; consider listing selected tests)"
	testAdequacyWhyMissingNote       = "test_adequacy.why_sufficient not declared (advisory-only; consider explaining why tests are sufficient)"
	testAdequacyGapsMissingNote      = "test_adequacy.known_gaps not declared (advisory-only; deep should list gaps or set [])"
)

// evaluateTestAdequacy emits advisory notes (never errors) for standard and
// deep tier cards when test_adequacy or its sub-properties are missing or
// blank. The light tier remains quiet. known_gaps == [] (explicit empty) is
// accepted for deep without emitting a note. This is the first vertical
// slice; it never blocks admission. Wording is parity-safe with the TS
// implementation in packages/cli/src/core/admission.ts and the policy
// documentation in policies/admission.yaml.
func evaluateTestAdequacy(doc map[string]any, tier string) []string {
	notes := make([]string, 0)
	if tier != "standard" && tier != "deep" {
		return notes
	}

	testAdequacy := mapValue(doc, "test_adequacy")
	if testAdequacy == nil {
		notes = append(notes, testAdequacyMissingNote)
		return notes
	}

	// impacted_behaviors: missing or non-array or empty -> note
	if !nonEmptyStringArray(testAdequacy, "impacted_behaviors") {
		notes = append(notes, testAdequacyBehaviorsMissingNote)
	}

	// tests_selected: missing or non-array or empty -> note
	if !nonEmptyStringArray(testAdequacy, "tests_selected") {
		notes = append(notes, testAdequacyTestsMissingNote)
	}

	// why_sufficient: missing or blank -> note
	why := strings.TrimSpace(stringInMap(testAdequacy, "why_sufficient"))
	if why == "" {
		notes = append(notes, testAdequacyWhyMissingNote)
	}

	// deep: known_gaps must be present (missing/blank key -> note;
	// explicit empty array is accepted and produces no note).
	if tier == "deep" {
		if !hasKey(testAdequacy, "known_gaps") || !nonEmptyStringArray(testAdequacy, "known_gaps") {
			// Distinguish "field is present but empty/blank" (already
			// filtered out by the prior nonEmptyStringArray) from
			// "field is missing entirely".
			if !hasKey(testAdequacy, "known_gaps") {
				notes = append(notes, testAdequacyGapsMissingNote)
			}
		}
	}

	return notes
}

// nonEmptyStringArray reports whether m[key] is a non-empty array of
// non-blank strings. Empty arrays, nil, missing keys, and non-string
// entries all return false.
func nonEmptyStringArray(m map[string]any, key string) bool {
	raw, ok := m[key]
	if !ok {
		return false
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return false
	}
	for _, v := range arr {
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return false
		}
	}
	return true
}

// hasKey reports whether m contains key (regardless of value).
func hasKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}
