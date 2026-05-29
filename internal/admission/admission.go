package admission

import (
	"fmt"
	"strings"
)

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

	// Tier guard: warn/block suspicious low-tier declarations for high-risk content
	tgResult := evaluateTierGuard(doc, tier)
	notes = append(notes, tgResult.notes...)
	for _, e := range tgResult.errors {
		applyFinding(e.message, e.predicate, false)
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
