package admission

import (
	"fmt"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/classify"
)

// FailureTaxonomy provides minimal classification for withheld/failed/blocked outcomes.
type FailureTaxonomy struct {
	FailureClass   string `json:"failure_class"`
	FailureStage   string `json:"failure_stage"`
	Recoverability string `json:"recoverability"`
	NextAction     string `json:"next_action"`
}

// Result is the output of the admission decision engine.
type Result struct {
	Outcome           string           `json:"outcome"`
	AcceptanceStatus  string           `json:"acceptance_status"`
	Errors            []string         `json:"errors"`
	Notes             []string         `json:"notes"`
	BlockingPredicate string           `json:"blocking_predicate,omitempty"`
	WithheldReason    *FailureTaxonomy `json:"withheld_reason,omitempty"`
}

func isCompletionCardShape(doc map[string]any) bool {
	if _, ok := doc["schema_version"]; ok {
		return true
	}
	if _, ok := doc["task_id"]; ok {
		return true
	}
	if _, ok := doc["admission"]; ok {
		return true
	}
	if _, ok := doc["acceptance_status"]; ok {
		return true
	}
	if claim, ok := doc["claim"].(map[string]any); ok {
		if _, ok := claim["fix_status"]; ok {
			if _, ok := claim["summary"]; ok {
				return true
			}
		}
	}
	return false
}

// Run evaluates a parsed completion card and returns an admission result.
func Run(doc map[string]any, strict bool) Result {
	errors := make([]string, 0)
	notes := make([]string, 0)
	blockingPredicate := ""

	applyFinding := func(message, predicate string, force bool) {
		errors = append(errors, message)
		if force || (blockingPredicate == "" && predicate != "") {
			blockingPredicate = predicate
		}
	}

	// Stale-ground fail-closed
	if boolValue(doc, "stale_ground") {
		return Result{
			Outcome:           "blocked",
			AcceptanceStatus:  "withheld",
			Errors:            []string{"stale_ground detected: withholding pending refresh or ruling out"},
			Notes:             []string{"stale-ground policy: if_detected = withhold"},
			BlockingPredicate: "stale_ground",
			WithheldReason:    buildTaxonomy("stale_ground"),
		}
	}

	// Required fields check (only for completion card shape)
	if isCompletionCardShape(doc) {
		if owner := stringValue(doc, "owner"); strings.TrimSpace(owner) == "" {
			errors = append(errors, "missing owner: owner is required")
		}
		if accountable := stringValue(doc, "accountable"); strings.TrimSpace(accountable) == "" {
			errors = append(errors, "missing accountable: accountable is required")
		}
		if taskID := stringValue(doc, "task_id"); strings.TrimSpace(taskID) == "" {
			errors = append(errors, "missing task_id: task_id is required")
		}
	}

	// Tier validation
	tier := stringValue(doc, "tier")
	if tier != "" && tier != "light" && tier != "standard" && tier != "deep" {
		errors = append(errors, fmt.Sprintf(`invalid tier: %q must be one of light, standard, deep`, tier))
	}

	admissionOutcome := stringInMap(mapValue(doc, "admission"), "outcome")
	governance := mapValue(doc, "governance")
	intake := mapValue(doc, "intake")

	// Fix-status contradictions (claim vs result)
	claim := mapValue(doc, "claim")
	claimFixStatus := stringInMap(claim, "fix_status")
	subagentReturn := mapValue(doc, "subagent_return")
	subagentFixStatus := stringInMap(mapValue(subagentReturn, "result"), "fix_status")
	if claimFixStatus != "" && subagentFixStatus != "" && claimFixStatus != subagentFixStatus {
		applyFinding(
			fmt.Sprintf(`canonical contradiction: claim.fix_status is %q but result.fix_status is %q`, claimFixStatus, subagentFixStatus),
			"admission_failed",
			false,
		)
	}

	// Tier downgrade check
	if intake != nil {
		mappedTier := stringInMap(intake, "mapped_tier")
		if isRuntimeTier(tier) && isRuntimeTier(mappedTier) {
			if isTierDowngrade(tier, mappedTier) {
				if !hasApprovedTierDowngrade(governance) {
					applyFinding(
						fmt.Sprintf("intake tier downgrade requires governance intervention approval: declared %s, mapped %s", tier, mappedTier),
						"Fintervention",
						false,
					)
				} else {
					notes = append(notes, fmt.Sprintf("intake tier downgrade approved by governance intervention: declared %s, mapped %s", tier, mappedTier))
				}
			}
		} else if mappedTier != "" && !isRuntimeTier(mappedTier) {
			applyFinding(
				fmt.Sprintf(`intake.mapped_tier %q must be one of light, standard, deep`, mappedTier),
				"admission_failed",
				false,
			)
		}
	}

	// Evidence floor
	evResult := evaluateEvidenceFloor(doc, tier)
	notes = append(notes, evResult.notes...)
	for _, e := range evResult.errors {
		applyFinding(e.message, e.predicate, false)
	}

	// Command safety
	cmdResult := evaluateCommandSafety(doc)
	for _, e := range cmdResult.errors {
		applyFinding(e.message, e.predicate, false)
	}

	// Approval receipt enforcement for classified commands
	appResult := evaluateApprovalReceipt(doc, tier)
	for _, e := range appResult.errors {
		applyFinding(e.message, e.predicate, false)
	}
	notes = append(notes, appResult.notes...)

	// Artifact status consistency
	artResult := evaluateArtifactStatus(doc)
	for _, e := range artResult.errors {
		applyFinding(e.message, e.predicate, false)
	}

	// Strict provenance
	provResult := evaluateStrictProvenance(doc, tier, strict)
	for _, e := range provResult.errors {
		applyFinding(e.message, e.predicate, false)
	}

	// Done checklist + prediction for standard/deep
	if tier == "standard" || tier == "deep" {
		if doc["done_checklist"] == nil {
			applyFinding("done_checklist is required for standard/deep tier", "admission_failed", false)
		}
		if doc["prediction"] == nil {
			applyFinding("prediction is required for standard/deep tier", "admission_failed", false)
		}
	}

	// Deep approval check
	if tier == "deep" && governance != nil {
		if boolInMap(governance, "requires_human_approval") {
			approvalStatus := stringInMap(governance, "approval_status")
			if approvalStatus != "approved" {
				applyFinding("deep task requires human approval before admission", "approval_missing", false)
			}
		}
	}

	// Canonical status contradictions
	fixStatus := claimFixStatus
	if fixStatus == "" {
		fixStatus = subagentFixStatus
	}
	verifyStatus := stringInMap(mapValue(doc, "verification"), "status")
	acceptanceStatus := stringValue(doc, "acceptance_status")
	handoff := mapValue(doc, "handoff")

	if fixStatus == "" {
		applyFinding("claim.fix_status or result.fix_status is required", "admission_failed", false)
	}
	if verifyStatus == "" {
		applyFinding("verification.status is required", "admission_failed", false)
	}

	if verifyStatus == "passed" && fixStatus != "" && fixStatus != "fixed" {
		applyFinding(
			fmt.Sprintf(`canonical contradiction: verification.status is "passed" but claim.fix_status is %q (must be "fixed")`, fixStatus),
			"admission_failed",
			false,
		)
	}

	if acceptanceStatus == "accepted" && isNonSuccessStatus(verifyStatus) {
		applyFinding(
			fmt.Sprintf(`canonical contradiction: acceptance_status is "accepted" but verification.status is %q`, verifyStatus),
			"admission_failed",
			false,
		)
	}

	if admissionOutcome == "success" {
		if acceptanceStatus != "accepted" {
			applyFinding(
				fmt.Sprintf(`canonical contradiction: admission.outcome is "success" but acceptance_status is %q (must be "accepted")`, acceptanceStatus),
				"admission_failed",
				false,
			)
		}
		if verifyStatus != "passed" {
			applyFinding(
				fmt.Sprintf(`success requires verification.status "passed" but found %q`, verifyStatus),
				"admission_failed",
				false,
			)
		}
		if fixStatus != "" && fixStatus != "fixed" {
			applyFinding(
				fmt.Sprintf(`success requires claim.fix_status "fixed" but found %q`, fixStatus),
				"admission_failed",
				false,
			)
		}
	}

	if acceptanceStatus == "accepted" && admissionOutcome != "" && admissionOutcome != "success" {
		applyFinding(
			fmt.Sprintf(`canonical contradiction: acceptance_status is "accepted" but admission.outcome is %q (must be "success")`, admissionOutcome),
			"admission_failed",
			false,
		)
	}

	if admissionOutcome != "" && admissionOutcome != "success" && acceptanceStatus == "accepted" {
		applyFinding(
			fmt.Sprintf(`canonical contradiction: non-success outcome %q cannot have acceptance_status "accepted"`, admissionOutcome),
			"admission_failed",
			false,
		)
	}

	if isNonSuccessStatus(admissionOutcome) {
		nextAction := strings.TrimSpace(stringInMap(handoff, "next_action"))
		owner := strings.TrimSpace(stringInMap(handoff, "owner"))
		if nextAction == "" {
			applyFinding(fmt.Sprintf(`admission.outcome %q requires handoff.next_action`, admissionOutcome), "admission_failed", false)
		}
		if owner == "" {
			applyFinding(fmt.Sprintf(`admission.outcome %q requires handoff.owner`, admissionOutcome), "admission_failed", false)
		}
	}

	if isNonSuccessStatus(verifyStatus) {
		nextAction := strings.TrimSpace(stringInMap(handoff, "next_action"))
		owner := strings.TrimSpace(stringInMap(handoff, "owner"))
		if nextAction == "" {
			applyFinding(fmt.Sprintf(`verification.status %q requires handoff.next_action`, verifyStatus), "admission_failed", false)
		}
		if owner == "" {
			applyFinding(fmt.Sprintf(`verification.status %q requires handoff.owner`, verifyStatus), "admission_failed", false)
		}
	}

	// PGV authority check
	pgvAdvice := mapValue(doc, "pgv_advice")
	if boolInMap(pgvAdvice, "admission_authority") {
		applyFinding("pgv_advice cannot grant admission authority; PGV is advisory-only", "admission_failed", false)
	}

	if len(errors) > 0 {
		if blockingPredicate == "" {
			blockingPredicate = "admission_failed"
		}
		return Result{
			Outcome:           "failed",
			AcceptanceStatus:  "withheld",
			Errors:            errors,
			Notes:             notes,
			BlockingPredicate: blockingPredicate,
			WithheldReason:    buildTaxonomy(blockingPredicate),
		}
	}

	// Respect input admission outcome if valid
	finalOutcome := "success"
	if isValidOutcome(admissionOutcome) {
		finalOutcome = admissionOutcome
	}

	acceptance := "accepted"
	if finalOutcome != "success" {
		acceptance = "withheld"
	}

	if len(notes) == 0 {
		notes = append(notes, "admission checks passed")
	} else {
		notes = append(notes, "admission checks passed")
	}

	return Result{
		Outcome:           finalOutcome,
		AcceptanceStatus:  acceptance,
		Errors:            []string{},
		Notes:             notes,
		BlockingPredicate: "",
	}
}

type evidenceFinding struct {
	message   string
	predicate string
}

type evidenceResult struct {
	errors []evidenceFinding
	notes  []string
}

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

func isRuntimeTier(tier string) bool {
	return tier == "light" || tier == "standard" || tier == "deep"
}

func isTierDowngrade(declared, mapped string) bool {
	tierRank := map[string]int{"light": 1, "standard": 2, "deep": 3}
	return tierRank[declared] < tierRank[mapped]
}

func hasApprovedTierDowngrade(governance map[string]any) bool {
	if governance == nil {
		return false
	}
	approvalStatus := stringInMap(governance, "approval_status")
	return approvalStatus == "approved"
}

func isNonSuccessStatus(status string) bool {
	switch status {
	case "failed", "blocked", "skipped", "timeout", "error":
		return true
	}
	return false
}

func isValidOutcome(outcome string) bool {
	switch outcome {
	case "success", "failed", "blocked", "skipped", "timeout", "error":
		return true
	}
	return false
}

func stringValue(doc map[string]any, key string) string {
	if v, ok := doc[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func mapValue(doc map[string]any, key string) map[string]any {
	if v, ok := doc[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func boolValue(doc map[string]any, key string) bool {
	if v, ok := doc[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func boolInMap(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func stringInMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sliceInMap(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if s, ok := v.([]any); ok {
			return s
		}
	}
	return nil
}

func verificationArtifactsHaveScope(artifacts []any) bool {
	for _, item := range artifacts {
		artifact, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if len(sliceInMap(artifact, "verifies")) > 0 || len(sliceInMap(artifact, "does_not_verify")) > 0 {
			return true
		}
	}
	return false
}

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
	default:
		return &FailureTaxonomy{
			FailureClass:   "schema_or_policy_invalid",
			FailureStage:   "admission_gate",
			Recoverability: "retry_with_fixes",
			NextAction:     "review_and_resubmit",
		}
	}
}

func intLikeValue(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
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

func shellMetacharacter(command string) string {
	tokens := []string{"&&", "||", ";", "|", "`", "$(", ">", "<"}
	for _, token := range tokens {
		if strings.Contains(command, token) {
			return token
		}
	}
	return ""
}
