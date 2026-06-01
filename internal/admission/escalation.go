package admission

import (
	"fmt"
	"strings"
)

// escalationHighRiskPathPatterns is the hardcoded v1 list of high-risk
// path patterns. The values mirror the canonical
// verify_stage_escalation.v1.high_risk_path_patterns list declared in
// policies/escalation.yaml; they are inlined here rather than loaded at
// runtime, so this slice must be kept in lockstep with that policy file.
// A declared `claim.evidence.files_changed` entry matching any of these
// patterns requires a `deep` tier unless an approved governance
// intervention has been recorded.
//
// Wording is parity-safe with policies/escalation.yaml and the TS
// evaluator in packages/cli/src/core/admission-evidence.ts.
var escalationHighRiskPathPatterns = []string{
	"schemas/",
	"policies/",
	".github/workflows/",
	"authority",
	"auth",
	"migrations/",
}

// escalationBypassApprovedDowngrade mirrors the existing intake tier
// downgrade bypass: when governance.approval_status == "approved", the
// escalation guard is skipped for the affected card. This stays in
// lockstep with hasApprovedTierDowngrade() to avoid creating a new
// bypass surface beyond existing mechanisms.
func escalationBypassApprovedDowngrade(governance map[string]any) bool {
	return hasApprovedTierDowngrade(governance)
}

// isEscalationHighRiskPath returns true if path matches any v1 high-risk
// pattern. The check is case-insensitive and substring-based, matching
// the existing tier-guard helper style for predictable semantics.
func isEscalationHighRiskPath(path string) bool {
	lower := strings.ToLower(path)
	for _, pattern := range escalationHighRiskPathPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// collectEscalationHighRiskFiles returns the subset of files_changed
// entries that match the v1 high-risk path patterns. Non-string entries
// are silently skipped, matching the existing tier-guard behavior.
func collectEscalationHighRiskFiles(files []any) []string {
	var matched []string
	for _, item := range files {
		path, ok := item.(string)
		if !ok {
			continue
		}
		if isEscalationHighRiskPath(path) {
			matched = append(matched, path)
		}
	}
	return matched
}

// evaluateEscalation enforces the v1 verify-stage auto-escalation guard.
// A card declared `light` or `standard` whose `claim.evidence.files_changed`
// matches any high-risk path pattern is withheld with predicate
// `tier_escalation_required` unless an approved governance intervention
// has been recorded (same bypass as the intake tier-downgrade check).
//
// `deep` cards are never blocked by this guard, regardless of which paths
// are declared.
func evaluateEscalation(doc map[string]any, tier string) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}

	if tier != "light" && tier != "standard" {
		return result
	}

	evidence := mapValue(doc, "evidence")
	files := sliceInMap(evidence, "files_changed")
	if len(files) == 0 {
		return result
	}

	highRiskFiles := collectEscalationHighRiskFiles(files)
	if len(highRiskFiles) == 0 {
		return result
	}

	governance := mapValue(doc, "governance")
	if escalationBypassApprovedDowngrade(governance) {
		result.notes = append(result.notes, fmt.Sprintf(
			"tier escalation bypassed by approved governance intervention for high-risk files (%v)",
			highRiskFiles,
		))
		return result
	}

	result.errors = append(result.errors, evidenceFinding{
		message: fmt.Sprintf(
			"tier escalation required: %s tier declared with high-risk files %v; required tier is deep (or an approved governance intervention must be recorded)",
			tier, highRiskFiles,
		),
		predicate: "tier_escalation_required",
	})
	return result
}
