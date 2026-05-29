package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/trace"
)

type recoveryRoute struct {
	NextAction string `json:"next_action"`
	Owner      string `json:"owner"`
}

type playbookSuggestion struct {
	Predicate      string        `json:"predicate"`
	Route          recoveryRoute `json:"route"`
	ReviewRequired bool          `json:"review_required"`
	Rationale      string        `json:"rationale"`
}

type recoverOutput struct {
	Suggestions []playbookSuggestion `json:"suggestions"`
}

var defaultRoutes = map[string]recoveryRoute{
	"evidence_missing": {
		NextAction: "Attach validation evidence or explain why unavailable.",
		Owner:      "implementation-worker",
	},
	"evidence_floor_not_met": {
		NextAction: "Attach the tier-required evidence floor and rerun verification.",
		Owner:      "implementation-worker",
	},
	"evidence_scope_missing": {
		NextAction: "Declare what each validation artifact verifies and does not verify.",
		Owner:      "implementation-worker",
	},
	"evidence_provenance_missing": {
		NextAction: "Attach strict evidence provenance fields and rerun verification.",
		Owner:      "implementation-worker",
	},
	"typecheck_failed": {
		NextAction: "Return to implementation-worker for type repair.",
		Owner:      "implementation-worker",
	},
	"test_failed": {
		NextAction: "Diagnose failing behavior and update implementation or tests.",
		Owner:      "implementation-worker",
	},
	"lint_failed": {
		NextAction: "Fix lint issues or justify why the lint rule is not applicable.",
		Owner:      "implementation-worker",
	},
	"build_failed": {
		NextAction: "Fix build failure before requesting admission.",
		Owner:      "implementation-worker",
	},
	"approval_missing": {
		NextAction: "Request human approval before admission.",
		Owner:      "user",
	},
	"conflicting_scope": {
		NextAction: "Ask user to clarify task scope.",
		Owner:      "user",
	},
	"verifier_not_read_only": {
		NextAction: "Rerun verification with a read-only verifier.",
		Owner:      "admission-verifier",
	},
	"admission_failed": {
		NextAction: "Resolve admission validation errors and rerun verification.",
		Owner:      "implementation-worker",
	},
	"state_read_write_missing": {
		NextAction: "Declare state.read_set and state.write_set for the task.",
		Owner:      "implementation-worker",
	},
	"done_checklist_missing": {
		NextAction: "Declare the done_checklist required for standard or deep admission.",
		Owner:      "implementation-worker",
	},
	"done_checklist_mismatch": {
		NextAction: "Align done_checklist claims with state, evidence, artifacts, and prediction.",
		Owner:      "implementation-worker",
	},
	"prediction_missing": {
		NextAction: "Declare the falsifiable prediction required for standard or deep admission.",
		Owner:      "implementation-worker",
	},
	"prediction_invalid": {
		NextAction: "Complete the required prediction fields and rerun verification.",
		Owner:      "implementation-worker",
	},
	"done_checklist_prediction_mismatch": {
		NextAction: "Align done_checklist.prediction_declared with the prediction block.",
		Owner:      "implementation-worker",
	},
	"stale_ground": {
		NextAction: "Refresh stale context or rule it out before requesting admission.",
		Owner:      "implementation-worker",
	},
	"Fpermission": {
		NextAction: "Request human approval for this protected path change before admission.",
		Owner:      "user",
	},
	"Fintervention": {
		NextAction: "Review intervention artifact for authority boundary violation and resolve.",
		Owner:      "implementation-worker",
	},
}

func handleRecover(args []string, stdout io.Writer, stderr io.Writer) int {
	errorsStr := ""
	outcome := "failed"
	jsonMode := false
	autoMode := false
	traceDir := ".x-harness/traces"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--errors":
			if i+1 < len(args) {
				errorsStr = args[i+1]
				i++
			}
		case "--outcome":
			if i+1 < len(args) {
				outcome = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		case "--auto":
			autoMode = true
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
		}
	}

	if autoMode {
		return handleRecoverAuto(traceDir, jsonMode, stdout, stderr)
	}

	var errors []string
	if errorsStr != "" {
		for _, e := range strings.Split(errorsStr, ";") {
			e = strings.TrimSpace(e)
			if e != "" {
				errors = append(errors, e)
			}
		}
	}

	suggestions := generatePlaybook(errors, outcome)

	if jsonMode {
		output := recoverOutput{Suggestions: suggestions}
		if err := WriteJSON(stdout, output); err != nil {
			return ExitError
		}
		return ExitOK
	}

	renderPlaybookMarkdown(stdout, suggestions)
	return ExitOK
}

func handleRecoverAuto(traceDir string, jsonMode bool, stdout io.Writer, stderr io.Writer) int {
	events, err := trace.ReadTrace(traceDir)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read trace: %v\n", err)
		return ExitError
	}

	if len(events) == 0 {
		if jsonMode {
			output := recoverOutput{Suggestions: []playbookSuggestion{}}
			_ = WriteJSON(stdout, output)
		} else {
			WriteLine(stdout, "# Recovery Playbook Suggestion (Read-Only)")
			WriteLine(stdout, "")
			WriteLine(stdout, "> No trace events found. Run with trace events or use --errors to provide failures manually.")
			WriteLine(stdout, "")
			WriteLine(stdout, "Suggested command:")
			WriteLine(stdout, "  x-harness recovery suggest --errors '<error-message>'")
		}
		return ExitOK
	}

	// Find the latest non-success event.
	var latestEvent trace.TraceEvent
	found := false
	for i := len(events) - 1; i >= 0; i-- {
		outcome := getTraceString(events[i], "outcome")
		if outcome != "success" && outcome != "" {
			latestEvent = events[i]
			found = true
			break
		}
	}

	if !found {
		if jsonMode {
			output := recoverOutput{Suggestions: []playbookSuggestion{}}
			_ = WriteJSON(stdout, output)
		} else {
			WriteLine(stdout, "# Recovery Playbook Suggestion (Read-Only)")
			WriteLine(stdout, "")
			WriteLine(stdout, "> No failures detected in trace. All recent events report success.")
			WriteLine(stdout, "")
			WriteLine(stdout, "If you still need recovery guidance, run:")
			WriteLine(stdout, "  x-harness recovery suggest --errors '<error-message>'")
		}
		return ExitOK
	}

	outcome := getTraceString(latestEvent, "outcome")
	if outcome == "" {
		outcome = "failed"
	}

	var errors []string
	if notes, ok := latestEvent["notes"].([]interface{}); ok {
		for _, n := range notes {
			if s, ok := n.(string); ok && s != "" {
				errors = append(errors, s)
			}
		}
	}
	if pred := getTraceString(latestEvent, "blocking_predicate"); pred != "" {
		errors = append(errors, fmt.Sprintf("blocking_predicate: %s", pred))
	}
	if reason := getTraceString(latestEvent, "blocked_reason_class"); reason != "" {
		errors = append(errors, fmt.Sprintf("blocked_reason_class: %s", reason))
	}

	suggestions := generatePlaybook(errors, outcome)

	if jsonMode {
		output := recoverOutput{Suggestions: suggestions}
		if err := WriteJSON(stdout, output); err != nil {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "# Recovery Playbook Suggestion (Read-Only)")
	WriteLine(stdout, "")
	WriteLine(stdout, "> This playbook is a suggestion generated from the latest trace failure. Review before applying.")
	WriteLine(stdout, "> It does NOT modify policies, completion cards, or trace events.")
	WriteLine(stdout, "> Source trace event: %s", getTraceString(latestEvent, "event_id"))
	WriteLine(stdout, "")

	for _, s := range suggestions {
		WriteLine(stdout, "## %s", s.Predicate)
		WriteLine(stdout, "")
		WriteLine(stdout, "- **Next action:** %s", s.Route.NextAction)
		WriteLine(stdout, "- **Owner:** %s", s.Route.Owner)
		WriteLine(stdout, "- **Review required:** %s", map[bool]string{true: "yes", false: "no"}[s.ReviewRequired])
		WriteLine(stdout, "- **Rationale:** %s", s.Rationale)
		WriteLine(stdout, "")
	}

	if len(suggestions) == 0 {
		WriteLine(stdout, "No recovery actions suggested.")
		WriteLine(stdout, "")
	}

	WriteLine(stdout, "---")
	WriteLine(stdout, "Generated by x-harness recovery playbook generator (auto mode).")
	WriteLine(stdout, "")
	return ExitOK
}

func getTraceString(event trace.TraceEvent, key string) string {
	if v, ok := event[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func generatePlaybook(errors []string, outcome string) []playbookSuggestion {
	if outcome != "blocked" && outcome != "failed" {
		return nil
	}

	var suggestions []playbookSuggestion
	seen := make(map[string]bool)

	for _, err := range errors {
		predicate, route := suggestRecovery(err)
		if predicate != "" && route.NextAction != "" && !seen[predicate] {
			seen[predicate] = true
			suggestions = append(suggestions, playbookSuggestion{
				Predicate:      predicate,
				Route:          route,
				ReviewRequired: true,
				Rationale:      fmt.Sprintf(`Detected from error: "%s"`, err),
			})
		}
	}

	return suggestions
}

func suggestRecovery(errorText string) (string, recoveryRoute) {
	lower := strings.ToLower(errorText)

	if strings.Contains(lower, "stale_ground") {
		return "stale_ground", defaultRoutes["stale_ground"]
	}
	if strings.Contains(lower, "done_checklist.prediction_declared") {
		return "done_checklist_prediction_mismatch", defaultRoutes["done_checklist_prediction_mismatch"]
	}
	if strings.Contains(lower, "done_checklist") && strings.Contains(lower, "but") {
		return "done_checklist_mismatch", defaultRoutes["done_checklist_mismatch"]
	}
	if strings.Contains(lower, "done_checklist") {
		return "done_checklist_missing", defaultRoutes["done_checklist_missing"]
	}
	if strings.Contains(lower, "prediction.") {
		return "prediction_invalid", defaultRoutes["prediction_invalid"]
	}
	if strings.Contains(lower, "prediction") {
		return "prediction_missing", defaultRoutes["prediction_missing"]
	}
	if strings.Contains(lower, "governance") && strings.Contains(lower, "permission") {
		return "Fpermission", defaultRoutes["Fpermission"]
	}
	if strings.Contains(lower, "governance") && strings.Contains(lower, "intervention") {
		return "Fintervention", defaultRoutes["Fintervention"]
	}
	if strings.Contains(lower, "approval") {
		return "approval_missing", defaultRoutes["approval_missing"]
	}
	if strings.Contains(lower, "typecheck") || strings.Contains(lower, "type check") {
		return "typecheck_failed", defaultRoutes["typecheck_failed"]
	}
	if strings.Contains(lower, "test") && !strings.Contains(lower, "typecheck") {
		return "test_failed", defaultRoutes["test_failed"]
	}
	if strings.Contains(lower, "lint") {
		return "lint_failed", defaultRoutes["lint_failed"]
	}
	if strings.Contains(lower, "build") {
		return "build_failed", defaultRoutes["build_failed"]
	}
	if strings.Contains(lower, "state") || strings.Contains(lower, "read_set") || strings.Contains(lower, "write_set") {
		return "state_read_write_missing", defaultRoutes["state_read_write_missing"]
	}
	if strings.Contains(lower, "scope") || strings.Contains(lower, "untested") || strings.Contains(lower, "does_not_verify") {
		return "evidence_scope_missing", defaultRoutes["evidence_scope_missing"]
	}
	if strings.Contains(lower, "evidence floor") {
		return "evidence_floor_not_met", defaultRoutes["evidence_floor_not_met"]
	}
	if strings.Contains(lower, "evidence provenance") {
		return "evidence_provenance_missing", defaultRoutes["evidence_provenance_missing"]
	}
	if strings.Contains(lower, "evidence") {
		return "evidence_missing", defaultRoutes["evidence_missing"]
	}
	return "admission_failed", defaultRoutes["admission_failed"]
}

func handleRecovery(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "suggest" {
		WriteLine(stderr, "usage: recovery suggest [options]")
		return ExitUsage
	}
	return handleRecover(args[1:], stdout, stderr)
}

func renderPlaybookMarkdown(w io.Writer, suggestions []playbookSuggestion) {
	WriteLine(w, "# Recovery Playbook (Review Required)")
	WriteLine(w, "")
	WriteLine(w, "> This playbook is a candidate generated from verification failures. Review before applying.")
	WriteLine(w, "> It does NOT modify policies or completion cards.")
	WriteLine(w, "")

	for _, s := range suggestions {
		WriteLine(w, "## %s", s.Predicate)
		WriteLine(w, "")
		WriteLine(w, "- **Next action:** %s", s.Route.NextAction)
		WriteLine(w, "- **Owner:** %s", s.Route.Owner)
		WriteLine(w, "- **Review required:** %s", map[bool]string{true: "yes", false: "no"}[s.ReviewRequired])
		WriteLine(w, "- **Rationale:** %s", s.Rationale)
		WriteLine(w, "")
	}

	if len(suggestions) == 0 {
		WriteLine(w, "No recovery actions suggested.")
		WriteLine(w, "")
	}

	WriteLine(w, "---")
	WriteLine(w, "Generated by x-harness recovery playbook generator.")
	WriteLine(w, "")
}
