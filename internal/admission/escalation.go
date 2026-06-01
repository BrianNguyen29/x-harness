package admission

import (
	"fmt"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/classify"
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

// escalationOperationBlockedIntents is the hardcoded v1 list of
// intents returned by internal/classify.ClassifyCommand that, when
// observed in a declared command_evidence or verification_artifacts
// entry, require a `deep` tier. The set mirrors
// `operation_rules.v1.blocked_intents` in policies/escalation.yaml; it is
// inlined here rather than loaded at runtime, so this map must be kept
// in lockstep with that policy file. Names are exactly the values
// ClassifyCommand returns (e.g. `delete_files`, `git_mutation`).
//
// Wording is parity-safe with policies/escalation.yaml and the TS
// evaluator in packages/cli/src/core/admission-evidence.ts.
var escalationOperationBlockedIntents = map[string]struct{}{
	"delete_files":       {},
	"network_outbound":   {},
	"package_publish":    {},
	"secret_access":      {},
	"git_mutation":       {},
	"database_mutation":  {},
	"deploy_or_publish":  {},
	"permission_change":  {},
}

// escalationOperationEscalateUnknown mirrors
// `operation_rules.v1.escalate_unknown` in policies/escalation.yaml.
// When true, a card whose command_evidence or verification_artifacts
// contains a command that ClassifyCommand marks as unknown is treated as
// requiring a `deep` tier under the same v1 operation-based rule.
const escalationOperationEscalateUnknown = true

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

// classifyEvidenceCommand safely classifies a command string by trimming
// whitespace first. Empty commands are reported as unknown to mirror
// internal/classify.ClassifyCommand's behavior. The string is passed
// through classify.ClassifyCommand.
func classifyEvidenceCommand(command string) classify.CommandClassification {
	return classify.ClassifyCommand(command)
}

// collectOperationEscalationTriggers returns the subset of declared
// command entries (from command_evidence or verification_artifacts) that
// trigger v1 operation-based escalation. Triggers are: (a) a command
// whose classified intents intersect the v1 blocked-intent set, or (b) a
// command whose classification is unknown when
// escalationOperationEscalateUnknown is true. Non-object and empty
// command entries are silently skipped.
//
// Returned entries are formatted as "<source>:<command>" for use in
// error messages, where <source> is "command_evidence" or
// "verification_artifacts".
func collectOperationEscalationTriggers(items []any, source string) []string {
	if len(items) == 0 {
		return nil
	}
	var matched []string
	for _, raw := range items {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		command, ok := entry["command"].(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(command)
		if trimmed == "" {
			continue
		}
		classification := classifyEvidenceCommand(trimmed)
		triggered := false
		for _, intent := range classification.Intents {
			if _, blocked := escalationOperationBlockedIntents[intent]; blocked {
				triggered = true
				break
			}
		}
		if !triggered && escalationOperationEscalateUnknown && classification.Unknown {
			triggered = true
		}
		if triggered {
			matched = append(matched, fmt.Sprintf("%s:%s", source, trimmed))
		}
	}
	return matched
}

// collectAllOperationEscalationTriggers returns every triggered command
// across command_evidence and verification_artifacts, in that order. It
// is the union used by evaluateOperationEscalation to keep error
// messages predictable and deterministic.
func collectAllOperationEscalationTriggers(evidence map[string]any) []string {
	if evidence == nil {
		return nil
	}
	var triggered []string
	triggered = append(triggered, collectOperationEscalationTriggers(
		sliceInMap(evidence, "command_evidence"),
		"command_evidence",
	)...)
	triggered = append(triggered, collectOperationEscalationTriggers(
		sliceInMap(evidence, "verification_artifacts"),
		"verification_artifacts",
	)...)
	return triggered
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

// evaluateOperationEscalation enforces the v1 operation-based
// auto-escalation guard. A card declared `light` or `standard` whose
// declared commands (in `evidence.command_evidence` or
// `evidence.verification_artifacts`) carry a blocked intent — or, when
// `escalate_unknown: true` is in effect, an unknown command — is
// withheld with predicate `tier_escalation_required` unless an approved
// governance intervention has been recorded.
//
// `deep` cards are never blocked by this guard, regardless of which
// commands are declared. Safe build commands (e.g. `go build`, `tsc`)
// that classify as low risk or moderate risk and have no blocked intent
// never trigger the guard. Wording is parity-safe with
// policies/escalation.yaml and the TS evaluator in
// packages/cli/src/core/admission-evidence.ts.
func evaluateOperationEscalation(doc map[string]any, tier string) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}

	if tier != "light" && tier != "standard" {
		return result
	}

	evidence := mapValue(doc, "evidence")
	triggered := collectAllOperationEscalationTriggers(evidence)
	if len(triggered) == 0 {
		return result
	}

	governance := mapValue(doc, "governance")
	if escalationBypassApprovedDowngrade(governance) {
		result.notes = append(result.notes, fmt.Sprintf(
			"tier escalation bypassed by approved governance intervention for blocked-operation commands (%v)",
			triggered,
		))
		return result
	}

	result.errors = append(result.errors, evidenceFinding{
		message: fmt.Sprintf(
			"tier escalation required: %s tier declared with blocked-operation commands %v; required tier is deep (or an approved governance intervention must be recorded)",
			tier, triggered,
		),
		predicate: "tier_escalation_required",
	})
	return result
}
