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

type metricsData struct {
	VerificationStrength verificationStrength `json:"verification_strength"`
	StateConsistency     stateConsistency     `json:"state_consistency"`
	RecoveryAbility      recoveryAbility      `json:"recovery_ability"`
	Replayability        replayability        `json:"replayability"`
	Cost                 costMetrics          `json:"cost"`
}

type admissionSummary struct {
	Outcome          string   `json:"outcome"`
	AcceptanceStatus string   `json:"acceptance_status"`
	Errors           []string `json:"errors"`
	Notes            []string `json:"notes"`
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

	if !metricsMode {
		WriteLine(stderr, "command %q is declared in the Go CLI skeleton but not implemented yet", "report")
		return ExitUsage
	}

	if format != "markdown" && format != "json" {
		fmt.Fprintf(stderr, "unsupported report format %q for --metrics (supported: markdown, json)\n", format)
		return ExitUsage
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

	admResult := admission.Run(doc)
	verifyRuntimeMs := int(time.Since(startTime).Milliseconds())

	metrics := computeMetrics(doc, inputCardHash, policyHash, verifyRuntimeMs)

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
		Admission: admissionSummary{
			Outcome:          admResult.Outcome,
			AcceptanceStatus: admResult.AcceptanceStatus,
			Errors:           admResult.Errors,
			Notes:            admResult.Notes,
		},
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
