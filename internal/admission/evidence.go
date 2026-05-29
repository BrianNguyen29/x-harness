package admission

import (
	"fmt"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/classify"
)

func evaluateEvidenceFloor(doc map[string]any, tier string) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}

	evidence := mapValue(doc, "evidence")
	state := mapValue(doc, "state")

	filesChanged := sliceInMap(evidence, "files_changed")
	commandEvidence := sliceInMap(evidence, "command_evidence")
	manualRationale := stringInMap(evidence, "manual_rationale")
	verificationArtifacts := sliceInMap(evidence, "verification_artifacts")
	untestedRegions := sliceInMap(evidence, "untested_regions")
	remainingRisks := sliceInMap(evidence, "remaining_risks")
	rollbackPolicy := sliceInMap(evidence, "rollback_policy")
	executionControls := sliceInMap(evidence, "execution_controls")
	readSet := sliceInMap(state, "read_set")
	writeSet := sliceInMap(state, "write_set")

	hasFilesChanged := len(filesChanged) > 0
	hasCommandEvidence := len(commandEvidence) > 0
	hasManualRationale := strings.TrimSpace(manualRationale) != ""
	hasVerificationArtifacts := len(verificationArtifacts) > 0
	hasUntestedRegions := len(untestedRegions) > 0
	hasRemainingRisks := len(remainingRisks) > 0
	hasRollbackPolicy := len(rollbackPolicy) > 0
	hasExecutionControls := len(executionControls) > 0
	hasReadSet := len(readSet) > 0
	hasWriteSet := len(writeSet) > 0
	hasScopeDeclared := verificationArtifactsHaveScope(verificationArtifacts)

	for _, item := range commandEvidence {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		exitCode, ok := intLikeValue(record["exit_code"])
		if !ok || exitCode == 0 {
			continue
		}
		message := fmt.Sprintf("evidence.command_evidence has non-zero exit_code %d", exitCode)
		if command := stringInMap(record, "command"); strings.TrimSpace(command) != "" {
			message += fmt.Sprintf(" for command %q", command)
		}
		result.errors = append(result.errors, evidenceFinding{message: message, predicate: "admission_failed"})
	}

	switch tier {
	case "light":
		if !hasFilesChanged {
			result.errors = append(result.errors, evidenceFinding{message: "light tier evidence floor requires files_changed", predicate: "admission_failed"})
		}
		if !hasCommandEvidence && !hasManualRationale {
			result.errors = append(result.errors, evidenceFinding{message: "light tier evidence floor requires command_evidence or manual_rationale", predicate: "admission_failed"})
		}
	case "standard":
		if !hasFilesChanged {
			result.errors = append(result.errors, evidenceFinding{message: "standard tier evidence floor requires files_changed", predicate: "admission_failed"})
		}
		if !hasCommandEvidence {
			result.errors = append(result.errors, evidenceFinding{message: "standard tier evidence floor requires command_evidence", predicate: "admission_failed"})
		}
	case "deep":
		if !hasFilesChanged {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires files_changed", predicate: "admission_failed"})
		}
		if !hasCommandEvidence {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires command_evidence", predicate: "admission_failed"})
		}
		if !hasVerificationArtifacts {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires verification_artifacts", predicate: "admission_failed"})
		}
		if !hasScopeDeclared {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires evidence scope declared (verifies/does_not_verify)", predicate: "admission_failed"})
		}
		if !hasUntestedRegions {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires untested_regions", predicate: "admission_failed"})
		}
		if !hasRemainingRisks {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires remaining_risks", predicate: "admission_failed"})
		}
		if !hasRollbackPolicy {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires rollback_policy", predicate: "admission_failed"})
		}
		if !hasExecutionControls {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires execution_controls", predicate: "admission_failed"})
		}
		if !hasReadSet {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires state.read_set", predicate: "admission_failed"})
		}
		if !hasWriteSet {
			result.errors = append(result.errors, evidenceFinding{message: "deep tier evidence floor requires state.write_set", predicate: "admission_failed"})
		}
	}

	return result
}

func evaluateTierGuard(doc map[string]any, tier string) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}
	if tier == "" {
		return result
	}

	evidence := mapValue(doc, "evidence")
	files := sliceInMap(evidence, "files_changed")

	var highRiskFiles []string
	for _, item := range files {
		path, ok := item.(string)
		if !ok {
			continue
		}
		lower := strings.ToLower(path)
		if strings.Contains(lower, "schema") ||
			strings.Contains(lower, "policy") ||
			strings.Contains(lower, "admission") ||
			strings.Contains(lower, "permission") ||
			strings.Contains(lower, ".github/workflows") ||
			strings.Contains(lower, "ci/") ||
			strings.Contains(lower, "/ci/") {
			highRiskFiles = append(highRiskFiles, path)
		}
	}

	var highRiskCommands []string
	for _, item := range sliceInMap(evidence, "command_evidence") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cmd := stringInMap(record, "command")
		if cmd != "" {
			classification := classify.ClassifyCommand(cmd)
			if classification.Risk == "high" || classification.Unknown {
				highRiskCommands = append(highRiskCommands, cmd)
			}
		}
	}
	for _, item := range sliceInMap(evidence, "verification_artifacts") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cmd := stringInMap(record, "command")
		if cmd != "" {
			classification := classify.ClassifyCommand(cmd)
			if classification.Risk == "high" || classification.Unknown {
				highRiskCommands = append(highRiskCommands, cmd)
			}
		}
	}

	if tier == "light" {
		if len(highRiskFiles) > 0 {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("tier guard: light tier declared but high-risk files detected (%v); consider standard or deep", highRiskFiles),
				predicate: "admission_failed",
			})
		}
		if len(highRiskCommands) > 0 {
			result.notes = append(result.notes, fmt.Sprintf("tier guard warning: light tier with high-risk command(s) (%v); consider raising tier", highRiskCommands))
		}
	}

	if tier == "standard" && len(highRiskFiles) > 0 && len(highRiskCommands) > 0 {
		result.notes = append(result.notes, fmt.Sprintf("tier guard warning: standard tier with both high-risk files (%v) and high-risk commands (%v); consider deep", highRiskFiles, highRiskCommands))
	}

	return result
}

func evaluateCommandSafety(doc map[string]any) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}
	evidence := mapValue(doc, "evidence")

	for _, item := range sliceInMap(evidence, "command_evidence") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		command := stringInMap(record, "command")
		if token := shellMetacharacter(command); token != "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("evidence.command_evidence command contains denied shell metacharacter %s: %q", token, command),
				predicate: "admission_failed",
			})
		}
	}

	for _, item := range sliceInMap(evidence, "verification_artifacts") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		command := stringInMap(record, "command")
		if token := shellMetacharacter(command); token != "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("evidence.verification_artifacts command contains denied shell metacharacter %s: %q", token, command),
				predicate: "admission_failed",
			})
		}
	}

	return result
}

func evaluateApprovalReceipt(doc map[string]any, tier string) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}
	if tier != "standard" && tier != "deep" {
		return result
	}

	evidence := mapValue(doc, "evidence")
	var commands []string
	for _, item := range sliceInMap(evidence, "command_evidence") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if cmd := stringInMap(record, "command"); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	for _, item := range sliceInMap(evidence, "verification_artifacts") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if cmd := stringInMap(record, "command"); cmd != "" {
			commands = append(commands, cmd)
		}
	}

	var requiringApproval []classify.CommandClassification
	var maxRequiredRisk string
	for _, cmd := range commands {
		classification := classify.ClassifyCommand(cmd)
		needsApproval := false
		switch tier {
		case "standard":
			if classification.Risk == "high" || classification.Unknown {
				needsApproval = true
			}
		case "deep":
			if classification.Risk == "medium" || classification.Risk == "high" || classification.Unknown {
				needsApproval = true
			}
		}
		if needsApproval {
			requiringApproval = append(requiringApproval, classification)
			if classify.RiskMeetsThreshold(classification.Risk, maxRequiredRisk) {
				maxRequiredRisk = classification.Risk
			}
		}
	}

	if len(requiringApproval) == 0 {
		return result
	}

	receipt := mapValue(doc, "approval_receipt")
	if receipt == nil {
		result.errors = append(result.errors, evidenceFinding{
			message:   fmt.Sprintf("tier %s requires approval receipt for %d high-risk command(s)", tier, len(requiringApproval)),
			predicate: "classifier_approval_required",
		})
		return result
	}

	decision := stringInMap(receipt, "decision")
	approver := stringInMap(receipt, "approver")
	aggregateRisk := stringInMap(receipt, "aggregate_risk")
	classifiedCmds := sliceInMap(receipt, "classified_commands")

	if decision != "approved" {
		result.errors = append(result.errors, evidenceFinding{
			message:   fmt.Sprintf("approval_receipt decision is %q; must be 'approved'", decision),
			predicate: "classifier_approval_required",
		})
	}
	if strings.TrimSpace(approver) == "" {
		result.errors = append(result.errors, evidenceFinding{
			message:   "approval_receipt approver is required",
			predicate: "classifier_approval_required",
		})
	}
	if len(classifiedCmds) == 0 {
		result.errors = append(result.errors, evidenceFinding{
			message:   "approval_receipt classified_commands is required",
			predicate: "classifier_approval_required",
		})
	}
	if maxRequiredRisk != "" && !classify.RiskMeetsThreshold(aggregateRisk, maxRequiredRisk) {
		result.errors = append(result.errors, evidenceFinding{
			message:   fmt.Sprintf("approval_receipt aggregate_risk %q is below required threshold %q", aggregateRisk, maxRequiredRisk),
			predicate: "classifier_approval_required",
		})
	}

	// Build coverage map from receipt classified commands
	covered := make(map[string]struct{})
	for _, item := range classifiedCmds {
		rec, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cmd := stringInMap(rec, "command")
		if cmd != "" {
			covered[cmd] = struct{}{}
		}
	}

	for _, classification := range requiringApproval {
		if _, ok := covered[classification.Command]; !ok {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("approval_receipt does not cover command %q (risk: %s)", classification.Command, classification.Risk),
				predicate: "classifier_approval_required",
			})
		}
	}

	if len(result.errors) == 0 {
		result.notes = append(result.notes, fmt.Sprintf("approval_receipt validated for %d high-risk command(s)", len(requiringApproval)))
	}

	return result
}

func evaluateArtifactStatus(doc map[string]any) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}
	evidence := mapValue(doc, "evidence")

	for _, item := range sliceInMap(evidence, "verification_artifacts") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		status := stringInMap(record, "status")
		if status == "" || status == "passed" {
			continue
		}
		command := stringInMap(record, "command")
		msg := fmt.Sprintf("evidence.verification_artifacts status %q is not passed", status)
		if command != "" {
			msg += fmt.Sprintf(" for command %q", command)
		}
		result.errors = append(result.errors, evidenceFinding{
			message:   msg,
			predicate: "admission_failed",
		})
	}

	return result
}

func evaluateStrictProvenance(doc map[string]any, tier string, strict bool) evidenceResult {
	result := evidenceResult{errors: []evidenceFinding{}, notes: []string{}}
	if !strict {
		return result
	}
	if tier != "standard" && tier != "deep" {
		return result
	}

	evidence := mapValue(doc, "evidence")
	for i, item := range sliceInMap(evidence, "command_evidence") {
		record, ok := item.(map[string]any)
		if !ok {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.command_evidence[%d] to be an object", i),
				predicate: "evidence_provenance_missing",
			})
			continue
		}
		if strings.TrimSpace(stringInMap(record, "command")) == "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.command_evidence[%d].command", i),
				predicate: "evidence_provenance_missing",
			})
		}
		if _, ok := intLikeValue(record["exit_code"]); !ok {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.command_evidence[%d].exit_code", i),
				predicate: "evidence_provenance_missing",
			})
		}
		if strings.TrimSpace(stringInMap(record, "runner")) == "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.command_evidence[%d].runner", i),
				predicate: "evidence_provenance_missing",
			})
		}
		if strings.TrimSpace(stringInMap(record, "started_at")) == "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.command_evidence[%d].started_at", i),
				predicate: "evidence_provenance_missing",
			})
		}
	}

	for i, item := range sliceInMap(evidence, "verification_artifacts") {
		record, ok := item.(map[string]any)
		if !ok {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.verification_artifacts[%d] to be an object", i),
				predicate: "evidence_provenance_missing",
			})
			continue
		}
		if strings.TrimSpace(stringInMap(record, "command")) == "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.verification_artifacts[%d].command", i),
				predicate: "evidence_provenance_missing",
			})
		}
		if _, ok := intLikeValue(record["exit_code"]); !ok {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.verification_artifacts[%d].exit_code", i),
				predicate: "evidence_provenance_missing",
			})
		}
		if strings.TrimSpace(stringInMap(record, "runner")) == "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.verification_artifacts[%d].runner", i),
				predicate: "evidence_provenance_missing",
			})
		}
		if strings.TrimSpace(stringInMap(record, "started_at")) == "" {
			result.errors = append(result.errors, evidenceFinding{
				message:   fmt.Sprintf("strict evidence provenance requires evidence.verification_artifacts[%d].started_at", i),
				predicate: "evidence_provenance_missing",
			})
		}
	}

	return result
}
