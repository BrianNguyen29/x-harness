package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/repo"
)

type verificationStrength struct {
	CommandEvidenceCount int      `json:"command_evidence_count"`
	OracleKinds          []string `json:"oracle_kinds"`
	UntestedRegionsCount int      `json:"untested_regions_count"`
	RemainingRisksCount  int      `json:"remaining_risks_count"`
}

type stateConsistency struct {
	OwnerPresent          bool `json:"owner_present"`
	AccountablePresent    bool `json:"accountable_present"`
	FilesChangedPresent   bool `json:"files_changed_present"`
	AdmissionMappingValid bool `json:"admission_mapping_valid"`
}

type recoveryAbility struct {
	BlockedHasNextAction bool `json:"blocked_has_next_action"`
	BlockedHasOwner      bool `json:"blocked_has_owner"`
	RecoveryRoutePresent bool `json:"recovery_route_present"`
}

type replayability struct {
	CompletionCardPresent bool `json:"completion_card_present"`
	InputCardHashPresent  bool `json:"input_card_hash_present"`
	PolicyHashPresent     bool `json:"policy_hash_present"`
}

type costMetrics struct {
	DefaultContextClass string `json:"default_context_class"`
	VerifyRuntimeMs     int    `json:"verify_runtime_ms"`
}

type rateMetric struct {
	Numerator    int    `json:"numerator"`
	Denominator  int    `json:"denominator"`
	Unit         string `json:"unit"`
	NotTaskLevel bool   `json:"not_task_level"`
}

type coverageMetric struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type metricsData struct {
	VerificationStrength   verificationStrength `json:"verification_strength"`
	StateConsistency       stateConsistency     `json:"state_consistency"`
	RecoveryAbility        recoveryAbility      `json:"recovery_ability"`
	Replayability          replayability        `json:"replayability"`
	Cost                   costMetrics          `json:"cost"`
	VerifyEventSuccessRate rateMetric           `json:"verify_event_success_rate"`
	TaskCompletionCoverage coverageMetric       `json:"task_completion_coverage"`
	WithheldRate           rateMetric           `json:"withheld_rate"`
}

type admissionSummary struct {
	Outcome          string          `json:"outcome"`
	AcceptanceStatus string          `json:"acceptance_status"`
	Errors           []string        `json:"errors"`
	Notes            []string        `json:"notes"`
	WithheldReason   *withheldReason `json:"withheld_reason,omitempty"`
}

type simpleAccounting struct {
	CardsAnalyzed int    `json:"cards_analyzed"`
	Note          string `json:"note"`
}

type lifecycleAccounting struct {
	Admitted int    `json:"admitted"`
	Withheld int    `json:"withheld"`
	Note     string `json:"note"`
}

type admissionAccounting struct {
	Accepted      int    `json:"accepted"`
	TotalAnalyzed int    `json:"total_analyzed"`
	Note          string `json:"note"`
}

type withheldAccounting struct {
	Failed  int    `json:"failed"`
	Blocked int    `json:"blocked"`
	Skipped int    `json:"skipped"`
	Timeout int    `json:"timeout"`
	Error   int    `json:"error"`
	Note    string `json:"note"`
}

type unknownEvents struct {
	Count int    `json:"count"`
	Note  string `json:"note"`
}

type traceEventAccounting struct {
	TotalTraceEvents int    `json:"total_trace_events"`
	Note             string `json:"note"`
}

type traceAdmissionAccounting struct {
	Accepted         int    `json:"accepted"`
	TotalTraceEvents int    `json:"total_trace_events"`
	Note             string `json:"note"`
}

type traceReportOutput struct {
	TotalEvents             int                      `json:"total_events"`
	Accepted                int                      `json:"accepted"`
	Withheld                int                      `json:"withheld"`
	ByOutcome               map[string]int           `json:"by_outcome"`
	VerifyEventAccounting   traceEventAccounting     `json:"verify_event_accounting"`
	TaskLifecycleAccounting lifecycleAccounting      `json:"task_lifecycle_accounting"`
	AdmissionAccounting     traceAdmissionAccounting `json:"admission_accounting"`
	WithheldAccounting      withheldAccounting       `json:"withheld_accounting"`
	UnknownOrUnlinkedEvents unknownEvents            `json:"unknown_or_unlinked_events"`
	Latest                  TraceEvent               `json:"latest"`
	VerifyEventSuccessRate  rateMetric               `json:"verify_event_success_rate"`
	TaskCompletionCoverage  coverageMetric           `json:"task_completion_coverage"`
	WithheldRate            rateMetric               `json:"withheld_rate"`
}

type reportMetricsOutput struct {
	CardID                  any                 `json:"card_id"`
	TaskID                  any                 `json:"task_id"`
	Tier                    string              `json:"tier"`
	Metrics                 metricsData         `json:"metrics"`
	Admission               admissionSummary    `json:"admission"`
	VerifyEventAccounting   simpleAccounting    `json:"verify_event_accounting"`
	TaskLifecycleAccounting lifecycleAccounting `json:"task_lifecycle_accounting"`
	AdmissionAccounting     admissionAccounting `json:"admission_accounting"`
	WithheldAccounting      withheldAccounting  `json:"withheld_accounting"`
	UnknownOrUnlinkedEvents unknownEvents       `json:"unknown_or_unlinked_events"`
	DenominatorWarning      string              `json:"denominator_warning"`
}

func handleReport(args []string, stdout io.Writer, stderr io.Writer) int {
	metricsMode := false
	cardPath := "completion-card.yaml"
	traceDir := ".x-harness/traces"
	jsonMode := false
	format := "markdown"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--metrics":
			metricsMode = true
		case "--card":
			if i+1 < len(args) {
				cardPath = args[i+1]
				i++
			}
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
			format = "json"
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	if format != "markdown" && format != "json" {
		fmt.Fprintf(stderr, "unsupported report format %q (supported: markdown, json)\n", format)
		return ExitUsage
	}

	if !metricsMode {
		return renderTraceReport(traceDir, jsonMode || format == "json", stdout, stderr)
	}

	if _, err := os.Stat(cardPath); os.IsNotExist(err) {
		fmt.Fprintf(stderr, "Error: Completion card not found at %s\n", cardPath)
		return ExitUsage
	}

	startTime := time.Now()
	var doc map[string]any
	if err := loader.LoadDocument(cardPath, &doc); err != nil {
		fmt.Fprintf(stderr, "error: cannot load card: %v\n", err)
		return ExitError
	}

	docBytes, _ := json.Marshal(doc)
	hash := sha256.Sum256(docBytes)
	inputCardHash := hex.EncodeToString(hash[:])

	root, _ := repo.FindRoot("")
	policyPath := filepath.Join(root, "policies", "admission.yaml")
	var policyHash string
	if policyData, err := os.ReadFile(policyPath); err == nil {
		hash := sha256.Sum256(policyData)
		policyHash = hex.EncodeToString(hash[:])
	} else {
		fmt.Fprintf(stderr, "warning: could not compute policy hash for %s: %v\n", policyPath, err)
	}

	admResult := admission.Run(doc, false)
	verifyRuntimeMs := int(time.Since(startTime).Milliseconds())

	metrics := computeMetrics(doc, inputCardHash, policyHash, verifyRuntimeMs)

	admittedCount := 0
	withheldCount := 0
	if admResult.AcceptanceStatus == "accepted" {
		admittedCount = 1
	} else {
		withheldCount = 1
	}
	metrics.VerifyEventSuccessRate = rateMetric{
		Numerator:    admittedCount,
		Denominator:  1,
		Unit:         "verify_event",
		NotTaskLevel: true,
	}
	metrics.TaskCompletionCoverage = coverageMetric{
		Status: "not_computable",
		Reason: "missing_aligned_task_denominator",
	}
	metrics.WithheldRate = rateMetric{
		Numerator:    withheldCount,
		Denominator:  1,
		Unit:         "verify_event",
		NotTaskLevel: true,
	}

	taskID := stringValue(doc, "task_id")
	tier := stringValue(doc, "tier")
	if tier == "" {
		tier = "standard"
	}

	admitted := 0
	withheld := 0
	if admResult.AcceptanceStatus == "accepted" {
		admitted = 1
	} else {
		withheld = 1
	}

	report := reportMetricsOutput{
		CardID:  doc["id"],
		TaskID:  taskID,
		Tier:    tier,
		Metrics: metrics,
		Admission: func() admissionSummary {
			summary := admissionSummary{
				Outcome:          admResult.Outcome,
				AcceptanceStatus: admResult.AcceptanceStatus,
				Errors:           admResult.Errors,
				Notes:            admResult.Notes,
			}
			if admResult.WithheldReason != nil {
				summary.WithheldReason = &withheldReason{
					FailureClass:      admResult.WithheldReason.FailureClass,
					FailureStage:      admResult.WithheldReason.FailureStage,
					Recoverability:    admResult.WithheldReason.Recoverability,
					NextAction:        admResult.WithheldReason.NextAction,
					BlockingPredicate: admResult.BlockingPredicate,
				}
			}
			return summary
		}(),
		VerifyEventAccounting: simpleAccounting{
			CardsAnalyzed: 1,
			Note:          "Single-card analysis; aggregate task denominator is not inferred.",
		},
		TaskLifecycleAccounting: lifecycleAccounting{
			Admitted: admitted,
			Withheld: withheld,
			Note:     "Lifecycle state reflects only the analyzed completion card.",
		},
		AdmissionAccounting: admissionAccounting{
			Accepted:      admitted,
			TotalAnalyzed: 1,
			Note:          "Admission requires outcome=success; non-success outcomes are withheld.",
		},
		WithheldAccounting: withheldAccounting{
			Failed:  boolToInt(admResult.Outcome == "failed"),
			Blocked: boolToInt(admResult.Outcome == "blocked"),
			Skipped: boolToInt(admResult.Outcome == "skipped"),
			Timeout: boolToInt(admResult.Outcome == "timeout"),
			Error:   boolToInt(admResult.Outcome == "error"),
			Note:    "Withheld breakdown reflects only the analyzed completion card.",
		},
		UnknownOrUnlinkedEvents: unknownEvents{
			Count: 0,
			Note:  "Not applicable for single-card metrics analysis.",
		},
		DenominatorWarning: "Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.",
	}

	if jsonMode || format == "json" {
		if err := WriteJSON(stdout, report); err != nil {
			return ExitError
		}
		return ExitOK
	}

	renderMetricsMarkdown(stdout, report)
	return ExitOK
}

func renderTraceReport(traceDir string, jsonMode bool, stdout io.Writer, stderr io.Writer) int {
	events, err := ReadTrace(traceDir)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read trace: %v\n", err)
		return ExitError
	}
	report := buildTraceReport(events)
	if jsonMode {
		if err := WriteJSON(stdout, report); err != nil {
			return ExitError
		}
		return ExitOK
	}
	renderTraceReportMarkdown(stdout, report)
	return ExitOK
}

func buildTraceReport(events []TraceEvent) traceReportOutput {
	total := len(events)
	accepted := 0
	withheld := 0
	failed := 0
	blocked := 0
	skipped := 0
	timeoutCount := 0
	errorCount := 0
	unknownCount := 0
	byOutcome := make(map[string]int)

	validOutcomes := map[string]bool{"success": true, "failed": true, "blocked": true, "skipped": true, "timeout": true, "error": true}
	for _, event := range events {
		outcome := traceString(event, "outcome")
		acceptance := traceString(event, "acceptance_status")
		if outcome == "" {
			outcome = "unknown"
		}
		byOutcome[outcome]++
		switch acceptance {
		case "accepted":
			accepted++
		case "withheld":
			withheld++
		}
		switch outcome {
		case "failed":
			failed++
		case "blocked":
			blocked++
		case "skipped":
			skipped++
		case "timeout":
			timeoutCount++
		case "error":
			errorCount++
		}
		if !validOutcomes[outcome] || (acceptance != "accepted" && acceptance != "withheld") {
			unknownCount++
		}
	}

	var latest TraceEvent
	if len(events) > 0 {
		latest = events[len(events)-1]
	}

	return traceReportOutput{
		TotalEvents: total,
		Accepted:    accepted,
		Withheld:    withheld,
		ByOutcome:   byOutcome,
		VerifyEventAccounting: traceEventAccounting{
			TotalTraceEvents: total,
			Note:             "Counts are based only on traced verify events; total task denominator may differ.",
		},
		TaskLifecycleAccounting: lifecycleAccounting{
			Admitted: accepted,
			Withheld: withheld,
			Note:     "Lifecycle accounting covers only events present in the trace log.",
		},
		AdmissionAccounting: traceAdmissionAccounting{
			Accepted:         accepted,
			TotalTraceEvents: total,
			Note:             "Admission requires outcome=success; non-success outcomes are withheld.",
		},
		WithheldAccounting: withheldAccounting{
			Failed:  failed,
			Blocked: blocked,
			Skipped: skipped,
			Timeout: timeoutCount,
			Error:   errorCount,
			Note:    "Withheld breakdown is only as complete as the trace event set.",
		},
		UnknownOrUnlinkedEvents: unknownEvents{
			Count: unknownCount,
			Note:  "Events with missing or unrecognized outcome/acceptance_status.",
		},
		Latest: latest,
		VerifyEventSuccessRate: rateMetric{
			Numerator:    accepted,
			Denominator:  total,
			Unit:         "verify_event",
			NotTaskLevel: true,
		},
		TaskCompletionCoverage: coverageMetric{
			Status: "not_computable",
			Reason: "missing_aligned_task_denominator",
		},
		WithheldRate: rateMetric{
			Numerator:    withheld,
			Denominator:  total,
			Unit:         "verify_event",
			NotTaskLevel: true,
		},
	}
}

func traceString(event TraceEvent, key string) string {
	if event == nil {
		return ""
	}
	if value, ok := event[key].(string); ok {
		return value
	}
	return ""
}

type worktreeInfo struct {
	Root   string `json:"root"`
	Branch string `json:"branch"`
	Commit string `json:"commit"`
}

func extractWorktree(event TraceEvent) *worktreeInfo {
	if event == nil {
		return nil
	}
	raw, ok := event["worktree"]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	wt := &worktreeInfo{}
	if v, ok := m["root"].(string); ok {
		wt.Root = v
	}
	if v, ok := m["branch"].(string); ok {
		wt.Branch = v
	}
	if v, ok := m["commit"].(string); ok {
		wt.Commit = v
	}
	if wt.Root == "" && wt.Branch == "" && wt.Commit == "" {
		return nil
	}
	return wt
}

func renderTraceReportMarkdown(w io.Writer, report traceReportOutput) {
	WriteLine(w, "# x-harness Report")
	WriteLine(w, "")
	WriteLine(w, "## Installed mode")
	WriteLine(w, "CLI-only (no daemon / no database / no MCP)")
	WriteLine(w, "")
	WriteLine(w, "## Completion card")
	if report.TotalEvents == 0 {
		WriteLine(w, "No completion cards found in trace.")
	} else {
		WriteLine(w, "%d card(s) in trace.", report.TotalEvents)
	}
	WriteLine(w, "")
	if wt := extractWorktree(report.Latest); wt != nil {
		WriteLine(w, "## Worktree")
		if wt.Root != "" {
			WriteLine(w, "- root: %s", wt.Root)
		}
		if wt.Branch != "" {
			WriteLine(w, "- branch: %s", wt.Branch)
		}
		if wt.Commit != "" {
			WriteLine(w, "- commit: %s", wt.Commit)
		}
		WriteLine(w, "")
	}
	WriteLine(w, "## Verify event accounting")
	if report.TotalEvents == 0 {
		WriteLine(w, "No verify events recorded.")
		WriteLine(w, "Denominator: NOT_COMPUTABLE (no events)")
	} else {
		WriteLine(w, "- total_trace_events: %d", report.TotalEvents)
		outcomes := make([]string, 0, len(report.ByOutcome))
		for outcome := range report.ByOutcome {
			outcomes = append(outcomes, outcome)
		}
		sort.Strings(outcomes)
		for _, outcome := range outcomes {
			WriteLine(w, "- %s: %d/%d", outcome, report.ByOutcome[outcome], report.TotalEvents)
		}
		WriteLine(w, "> Counts are based only on traced verify events; total task denominator may differ.")
	}
	WriteLine(w, "")
	WriteLine(w, "## Task lifecycle accounting")
	if report.TotalEvents == 0 {
		WriteLine(w, "No lifecycle data available.")
	} else {
		WriteLine(w, "- admitted: %d/%d", report.Accepted, report.TotalEvents)
		WriteLine(w, "- withheld: %d/%d", report.Withheld, report.TotalEvents)
		WriteLine(w, "> Lifecycle accounting covers only events present in the trace log.")
	}
	WriteLine(w, "")
	WriteLine(w, "## Admission accounting")
	if report.TotalEvents == 0 {
		WriteLine(w, "No admission data available.")
	} else {
		WriteLine(w, "- accepted: %d/%d", report.Accepted, report.TotalEvents)
		WriteLine(w, "> Admission requires outcome=success; non-success outcomes are withheld.")
	}
	WriteLine(w, "")
	WriteLine(w, "## Withheld accounting")
	if report.TotalEvents == 0 {
		WriteLine(w, "No withheld data available.")
	} else if report.Withheld == 0 {
		WriteLine(w, "None.")
	} else {
		if report.WithheldAccounting.Failed > 0 {
			WriteLine(w, "- failed: %d/%d", report.WithheldAccounting.Failed, report.TotalEvents)
		}
		if report.WithheldAccounting.Blocked > 0 {
			WriteLine(w, "- blocked: %d/%d", report.WithheldAccounting.Blocked, report.TotalEvents)
		}
		if report.WithheldAccounting.Skipped > 0 {
			WriteLine(w, "- skipped: %d/%d", report.WithheldAccounting.Skipped, report.TotalEvents)
		}
		if report.WithheldAccounting.Timeout > 0 {
			WriteLine(w, "- timeout: %d/%d", report.WithheldAccounting.Timeout, report.TotalEvents)
		}
		if report.WithheldAccounting.Error > 0 {
			WriteLine(w, "- error: %d/%d", report.WithheldAccounting.Error, report.TotalEvents)
		}
		WriteLine(w, "> Withheld breakdown is only as complete as the trace event set.")
	}
	WriteLine(w, "")
	WriteLine(w, "## Unknown or unlinked events")
	if report.UnknownOrUnlinkedEvents.Count == 0 {
		WriteLine(w, "None.")
	} else {
		WriteLine(w, "%d/%d events with missing or unrecognized outcome/acceptance_status.", report.UnknownOrUnlinkedEvents.Count, report.TotalEvents)
	}
	WriteLine(w, "")
	WriteLine(w, "## Rate metrics")
	if report.TotalEvents == 0 {
		WriteLine(w, "No rate metrics available (no events).")
	} else {
		WriteLine(w, "- verify_event_success_rate: %d/%d verify_event (not_task_level)", report.Accepted, report.TotalEvents)
		WriteLine(w, "- task_completion_coverage: not_computable (missing_aligned_task_denominator)")
		WriteLine(w, "- withheld_rate: %d/%d verify_event (not_task_level)", report.Withheld, report.TotalEvents)
	}
	WriteLine(w, "")
	WriteLine(w, "## Denominator warning")
	WriteLine(w, "> Verify-event success must not be interpreted as task-level success without denominator review.")
	if report.TotalEvents == 0 {
		WriteLine(w, "Denominator: NOT_COMPUTABLE (no events)")
	} else {
		WriteLine(w, "- accepted: %d/%d cards", report.Accepted, report.TotalEvents)
	}
}

func computeMetrics(doc map[string]any, inputCardHash, policyHash string, verifyRuntimeMs int) metricsData {
	evidence := mapValue(doc, "evidence")
	claim := mapValue(doc, "claim")
	var cardEvidence []any
	if claim != nil {
		if ce, ok := claim["evidence"].([]any); ok {
			cardEvidence = ce
		}
	}
	verificationArtifacts := sliceValue(evidence, "verification_artifacts")
	untestedRegions := sliceValue(evidence, "untested_regions")
	remainingRisks := sliceValue(evidence, "remaining_risks")

	oracleKindsSet := make(map[string]struct{})
	for _, item := range verificationArtifacts {
		artifact, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if kind, ok := artifact["kind"].(string); ok && kind != "" {
			oracleKindsSet[kind] = struct{}{}
		}
	}
	oracleKinds := make([]string, 0, len(oracleKindsSet))
	for k := range oracleKindsSet {
		oracleKinds = append(oracleKinds, k)
	}
	sort.Strings(oracleKinds)

	filesChanged := sliceValue(evidence, "files_changed")
	if len(filesChanged) == 0 {
		filesChanged = cardEvidence
	}

	admissionMap := mapValue(doc, "admission")
	outcome := ""
	if admissionMap != nil {
		if s, ok := admissionMap["outcome"].(string); ok {
			outcome = s
		}
	}
	blocked := outcome == "blocked" || outcome == "failed"

	handoff := mapValue(doc, "handoff")
	nextAction := ""
	if handoff != nil {
		if s, ok := handoff["next_action"].(string); ok {
			nextAction = strings.TrimSpace(s)
		}
	}
	hasNextAction := nextAction != "" && nextAction != "none"

	ownerHandoff := ""
	if handoff != nil {
		if s, ok := handoff["owner"].(string); ok {
			ownerHandoff = strings.TrimSpace(s)
		}
	}
	hasOwner := ownerHandoff != ""

	tier := stringValue(doc, "tier")
	if tier == "" {
		tier = "standard"
	}
	contextClass := "medium"
	switch tier {
	case "light":
		contextClass = "low"
	case "deep":
		contextClass = "high"
	}

	owner := stringValue(doc, "owner")
	accountable := stringValue(doc, "accountable")
	taskID := stringValue(doc, "task_id")

	admissionMappingValid := true
	if outcome == "success" {
		admissionMappingValid = stringValue(doc, "acceptance_status") == "accepted"
	} else {
		admissionMappingValid = stringValue(doc, "acceptance_status") != "accepted"
	}

	return metricsData{
		VerificationStrength: verificationStrength{
			CommandEvidenceCount: len(verificationArtifacts),
			OracleKinds:          oracleKinds,
			UntestedRegionsCount: len(untestedRegions),
			RemainingRisksCount:  len(remainingRisks),
		},
		StateConsistency: stateConsistency{
			OwnerPresent:          strings.TrimSpace(owner) != "",
			AccountablePresent:    strings.TrimSpace(accountable) != "",
			FilesChangedPresent:   len(filesChanged) > 0,
			AdmissionMappingValid: admissionMappingValid,
		},
		RecoveryAbility: recoveryAbility{
			BlockedHasNextAction: func() bool {
				if blocked {
					return hasNextAction
				}
				return true
			}(),
			BlockedHasOwner: func() bool {
				if blocked {
					return hasOwner
				}
				return true
			}(),
			RecoveryRoutePresent: func() bool {
				if blocked {
					return hasNextAction && hasOwner
				}
				return true
			}(),
		},
		Replayability: replayability{
			CompletionCardPresent: strings.TrimSpace(taskID) != "",
			InputCardHashPresent:  inputCardHash != "",
			PolicyHashPresent:     policyHash != "",
		},
		Cost: costMetrics{
			DefaultContextClass: contextClass,
			VerifyRuntimeMs:     verifyRuntimeMs,
		},
	}
}

func renderMetricsMarkdown(w io.Writer, report reportMetricsOutput) {
	WriteLine(w, "# x-harness Metrics Report")
	WriteLine(w, "")
	WriteLine(w, "## Verification strength")
	WriteLine(w, "- command_evidence_count: %d", report.Metrics.VerificationStrength.CommandEvidenceCount)
	kinds := strings.Join(report.Metrics.VerificationStrength.OracleKinds, ", ")
	if kinds == "" {
		kinds = "none"
	}
	WriteLine(w, "- oracle_kinds: %s", kinds)
	WriteLine(w, "- untested_regions_count: %d", report.Metrics.VerificationStrength.UntestedRegionsCount)
	WriteLine(w, "- remaining_risks_count: %d", report.Metrics.VerificationStrength.RemainingRisksCount)
	WriteLine(w, "")
	WriteLine(w, "## State consistency")
	WriteLine(w, "- owner_present: %v", report.Metrics.StateConsistency.OwnerPresent)
	WriteLine(w, "- accountable_present: %v", report.Metrics.StateConsistency.AccountablePresent)
	WriteLine(w, "- files_changed_present: %v", report.Metrics.StateConsistency.FilesChangedPresent)
	WriteLine(w, "- admission_mapping_valid: %v", report.Metrics.StateConsistency.AdmissionMappingValid)
	WriteLine(w, "")
	WriteLine(w, "## Recovery ability")
	WriteLine(w, "- blocked_has_next_action: %v", report.Metrics.RecoveryAbility.BlockedHasNextAction)
	WriteLine(w, "- blocked_has_owner: %v", report.Metrics.RecoveryAbility.BlockedHasOwner)
	WriteLine(w, "- recovery_route_present: %v", report.Metrics.RecoveryAbility.RecoveryRoutePresent)
	WriteLine(w, "")
	WriteLine(w, "## Replayability")
	WriteLine(w, "- completion_card_present: %v", report.Metrics.Replayability.CompletionCardPresent)
	WriteLine(w, "- input_card_hash_present: %v", report.Metrics.Replayability.InputCardHashPresent)
	WriteLine(w, "- policy_hash_present: %v", report.Metrics.Replayability.PolicyHashPresent)
	WriteLine(w, "")
	WriteLine(w, "## Cost")
	WriteLine(w, "- default_context_class: %s", report.Metrics.Cost.DefaultContextClass)
	WriteLine(w, "- verify_runtime_ms: %d", report.Metrics.Cost.VerifyRuntimeMs)
	WriteLine(w, "")
	WriteLine(w, "## Rate metrics")
	WriteLine(w, "- verify_event_success_rate: %d/%d verify_event (not_task_level)", report.Metrics.VerifyEventSuccessRate.Numerator, report.Metrics.VerifyEventSuccessRate.Denominator)
	WriteLine(w, "- task_completion_coverage: %s (%s)", report.Metrics.TaskCompletionCoverage.Status, report.Metrics.TaskCompletionCoverage.Reason)
	WriteLine(w, "- withheld_rate: %d/%d verify_event (not_task_level)", report.Metrics.WithheldRate.Numerator, report.Metrics.WithheldRate.Denominator)
	WriteLine(w, "")
	WriteLine(w, "## Verify event accounting")
	WriteLine(w, "- cards_analyzed: 1")
	WriteLine(w, "> Single-card analysis; aggregate task denominator is not inferred.")
	WriteLine(w, "")
	WriteLine(w, "## Task lifecycle accounting")
	WriteLine(w, "- admitted: %d/1", report.TaskLifecycleAccounting.Admitted)
	WriteLine(w, "- withheld: %d/1", report.TaskLifecycleAccounting.Withheld)
	WriteLine(w, "> Lifecycle state reflects only the analyzed completion card.")
	WriteLine(w, "")
	WriteLine(w, "## Admission accounting")
	WriteLine(w, "- accepted: %d/1", report.AdmissionAccounting.Accepted)
	WriteLine(w, "> Admission requires outcome=success; non-success outcomes are withheld.")
	WriteLine(w, "")
	WriteLine(w, "## Withheld accounting")
	if report.Admission.AcceptanceStatus == "accepted" {
		WriteLine(w, "None.")
	} else {
		if report.WithheldAccounting.Failed > 0 {
			WriteLine(w, "- failed: 1/1")
		}
		if report.WithheldAccounting.Blocked > 0 {
			WriteLine(w, "- blocked: 1/1")
		}
		if report.WithheldAccounting.Skipped > 0 {
			WriteLine(w, "- skipped: 1/1")
		}
		if report.WithheldAccounting.Timeout > 0 {
			WriteLine(w, "- timeout: 1/1")
		}
		if report.WithheldAccounting.Error > 0 {
			WriteLine(w, "- error: 1/1")
		}
		WriteLine(w, "> Withheld breakdown reflects only the analyzed completion card.")
	}
	WriteLine(w, "")
	WriteLine(w, "## Unknown or unlinked events")
	WriteLine(w, "Not applicable for single-card metrics analysis.")
	WriteLine(w, "")
	WriteLine(w, "## Denominator warning")
	WriteLine(w, "> Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.")
}

func sliceValue(m map[string]any, key string) []any {
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
