package admission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/loader"
)

func loadGolden(t *testing.T, name string) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "examples", "golden", name, "completion-card.yaml")
	if _, err := os.Stat(path); err != nil {
		for _, suite := range []string{"regression", "capability", "adversarial"} {
			alt := filepath.Join("..", "..", "examples", "golden", suite, name, "completion-card.yaml")
			if _, err := os.Stat(alt); err == nil {
				path = alt
				break
			}
		}
	}
	var doc map[string]any
	if err := loader.LoadDocument(path, &doc); err != nil {
		t.Fatalf("failed to load %s: %v", path, err)
	}
	return doc
}

func TestSuccessLight(t *testing.T) {
	doc := loadGolden(t, "success-light")
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
}

func TestWithheldPartialFix(t *testing.T) {
	doc := loadGolden(t, "withheld-partial-fix")
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
}

func TestSuccessStandardScopedEvidence(t *testing.T) {
	doc := loadGolden(t, "success-standard-scoped-evidence")
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
}

func TestDeepApprovalRequired(t *testing.T) {
	doc := loadGolden(t, "deep-approval-required")
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	found := false
	for _, e := range result.Errors {
		if e == "deep task requires human approval before admission" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected deep approval error, got %v", result.Errors)
	}
}

func TestBlockedTierDowngrade(t *testing.T) {
	doc := loadGolden(t, "blocked-tier-downgrade")
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	found := false
	for _, e := range result.Errors {
		if e == "intake tier downgrade requires governance intervention approval: declared light, mapped deep" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tier downgrade error, got %v", result.Errors)
	}
}

func TestMultiAgentSuccess(t *testing.T) {
	doc := loadGolden(t, "multi-agent-success")
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
}

func TestBlockedMissingEvidence(t *testing.T) {
	doc := loadGolden(t, "blocked-missing-evidence")
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
}

func TestBlockedMissingEvidenceScope(t *testing.T) {
	doc := loadGolden(t, "blocked-missing-evidence-scope")
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	found := false
	for _, e := range result.Errors {
		if e == "deep tier evidence floor requires evidence scope declared (verifies/does_not_verify)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected evidence scope error, got %v", result.Errors)
	}
}

func TestFailedTypecheckRecoveryRoute(t *testing.T) {
	doc := loadGolden(t, "failed-typecheck-recovery-route")
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	found := false
	for _, e := range result.Errors {
		if e == "evidence.command_evidence has non-zero exit_code 1 for command \"npm run typecheck\"" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected command evidence exit_code error, got %v", result.Errors)
	}
}

func TestHiddenDangerousCommand(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "adversarial", "hidden-dangerous-command", "completion-card.yaml")
	var doc map[string]any
	if err := loader.LoadDocument(path, &doc); err != nil {
		t.Fatalf("failed to load card: %v", err)
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "shell metacharacter") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected shell metacharacter error, got %v", result.Errors)
	}
}

func TestLyingCommandExitCode(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "adversarial", "lying-command-exit-code", "completion-card.yaml")
	var doc map[string]any
	if err := loader.LoadDocument(path, &doc); err != nil {
		t.Fatalf("failed to load card: %v", err)
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
}

func TestStaleGroundTaxonomy(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
		"stale_ground": true,
	}
	result := Run(doc, false, false)
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason for stale_ground")
	}
	if result.WithheldReason.FailureClass != "stale_context" {
		t.Fatalf("expected failure_class stale_context, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.FailureStage != "admission_gate" {
		t.Fatalf("expected failure_stage admission_gate, got %s", result.WithheldReason.FailureStage)
	}
	if result.WithheldReason.Recoverability != "retry_after_refresh" {
		t.Fatalf("expected recoverability retry_after_refresh, got %s", result.WithheldReason.Recoverability)
	}
	if result.WithheldReason.NextAction != "review_and_resubmit" {
		t.Fatalf("expected next_action review_and_resubmit, got %s", result.WithheldReason.NextAction)
	}
}

func TestMissingEvidenceTaxonomy(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
		"evidence": map[string]any{
			"files_changed": []any{},
		},
	}
	result := Run(doc, false, false)
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason for missing evidence")
	}
	if result.WithheldReason.FailureClass != "schema_or_policy_invalid" {
		t.Fatalf("expected failure_class schema_or_policy_invalid, got %s", result.WithheldReason.FailureClass)
	}
	if result.BlockingPredicate != "admission_failed" {
		t.Fatalf("expected blocking_predicate admission_failed, got %s", result.BlockingPredicate)
	}
}

func TestDeepApprovalTaxonomy(t *testing.T) {
	doc := loadGolden(t, "deep-approval-required")
	result := Run(doc, false, false)
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason for deep approval missing")
	}
	if result.WithheldReason.FailureClass != "governance_missing" {
		t.Fatalf("expected failure_class governance_missing, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.Recoverability != "human_intervention" {
		t.Fatalf("expected recoverability human_intervention, got %s", result.WithheldReason.Recoverability)
	}
	if result.WithheldReason.NextAction != "escalate" {
		t.Fatalf("expected next_action escalate, got %s", result.WithheldReason.NextAction)
	}
}

func TestStaleGroundBlocks(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
		"stale_ground": true,
	}
	result := Run(doc, false, false)
	if result.Outcome != "blocked" {
		t.Fatalf("expected blocked, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	if result.BlockingPredicate != "stale_ground" {
		t.Fatalf("expected stale_ground predicate, got %s", result.BlockingPredicate)
	}
}

func TestAcceptanceStatusMapping(t *testing.T) {
	tests := []struct {
		outcome  string
		expected string
	}{
		{"success", "accepted"},
		{"failed", "withheld"},
		{"blocked", "withheld"},
		{"skipped", "withheld"},
		{"timeout", "withheld"},
		{"error", "withheld"},
	}
	for _, tt := range tests {
		t.Run(tt.outcome, func(t *testing.T) {
			doc := map[string]any{
				"schema_version": "1",
				"task_id":        "T",
				"tier":           "light",
				"owner":          "a",
				"accountable":    "b",
				"evidence": map[string]any{
					"files_changed":    []any{"f"},
					"manual_rationale": "rationale",
				},
				"claim": map[string]any{
					"fix_status": "fixed",
					"summary":    "s",
					"evidence":   []any{"e"},
				},
				"verification": map[string]any{
					"status": "passed",
					"checks": []any{},
				},
				"admission": map[string]any{
					"outcome": tt.outcome,
				},
				"acceptance_status": tt.expected,
				"handoff": map[string]any{
					"next_action": "n",
					"owner":       "o",
				},
			}
			result := Run(doc, false, false)
			if result.Outcome != tt.outcome {
				t.Fatalf("expected outcome %s, got %s", tt.outcome, result.Outcome)
			}
			if result.AcceptanceStatus != tt.expected {
				t.Fatalf("expected acceptance %s, got %s", tt.expected, result.AcceptanceStatus)
			}
		})
	}
}

func TestStrictStandardMissingFieldsFails(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "npm test",
					"exit_code": 0,
					// missing runner and started_at
				},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, true, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	if result.BlockingPredicate != "evidence_provenance_missing" {
		t.Fatalf("expected evidence_provenance_missing predicate, got %s", result.BlockingPredicate)
	}
	foundRunner := false
	foundStartedAt := false
	for _, e := range result.Errors {
		if strings.Contains(e, ".runner") {
			foundRunner = true
		}
		if strings.Contains(e, ".started_at") {
			foundStartedAt = true
		}
	}
	if !foundRunner {
		t.Fatalf("expected runner error, got %v", result.Errors)
	}
	if !foundStartedAt {
		t.Fatalf("expected started_at error, got %v", result.Errors)
	}
}

func TestStrictStandardFullFieldsPasses(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":    "npm test",
					"exit_code":  0,
					"runner":     "local",
					"started_at": "2026-05-25T00:00:00Z",
				},
			},
			"verification_artifacts": []any{
				map[string]any{
					"command":    "npm test",
					"exit_code":  0,
					"runner":     "local",
					"started_at": "2026-05-25T00:00:00Z",
					"status":     "passed",
				},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, true, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
}

func TestStrictLightMissingFieldsExempt(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "npm test",
					"exit_code": 0,
					// missing runner and started_at
				},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, true, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
}

func TestStandardHighRiskMissingReceipt(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "rm -rf dist",
					"exit_code": 0,
				},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "requires approval receipt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected approval receipt error, got %v", result.Errors)
	}
	if result.BlockingPredicate != "classifier_approval_required" {
		t.Fatalf("expected classifier_approval_required predicate, got %s", result.BlockingPredicate)
	}
}

func TestStandardHighRiskWithReceipt(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "rm -rf dist",
					"exit_code": 0,
				},
			},
		},
		"approval_receipt": map[string]any{
			"decision": "approved",
			"approver": "user",
			"classified_commands": []any{
				map[string]any{
					"command": "rm -rf dist",
					"risk":    "high",
				},
			},
			"aggregate_risk": "high",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
}

func TestDeepMediumRiskMissingReceipt(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "deep",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"state": map[string]any{
			"read_set":  []any{"r"},
			"write_set": []any{"w"},
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "go build ./...",
					"exit_code": 0,
				},
			},
			"verification_artifacts": []any{
				map[string]any{
					"kind":     "build",
					"command":  "go build ./...",
					"status":   "passed",
					"verifies": []any{"v"},
				},
			},
			"untested_regions":   []any{"u"},
			"remaining_risks":    []any{"r"},
			"rollback_policy":    []any{"rp"},
			"execution_controls": []any{"ec"},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "requires approval receipt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected approval receipt error, got %v", result.Errors)
	}
}

func TestDeepMediumRiskWithReceipt(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "deep",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"state": map[string]any{
			"read_set":  []any{"r"},
			"write_set": []any{"w"},
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "go build ./...",
					"exit_code": 0,
				},
			},
			"verification_artifacts": []any{
				map[string]any{
					"kind":     "build",
					"command":  "go build ./...",
					"status":   "passed",
					"verifies": []any{"v"},
				},
			},
			"untested_regions":   []any{"u"},
			"remaining_risks":    []any{"r"},
			"rollback_policy":    []any{"rp"},
			"execution_controls": []any{"ec"},
		},
		"approval_receipt": map[string]any{
			"decision": "approved",
			"approver": "user",
			"classified_commands": []any{
				map[string]any{
					"command": "go build ./...",
					"risk":    "medium",
				},
			},
			"aggregate_risk": "medium",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
}

func TestLightHighRiskNoReceiptAllowed(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "rm -rf dist",
					"exit_code": 0,
				},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
}

func TestApprovalReceiptTaxonomy(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "rm -rf dist",
					"exit_code": 0,
				},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason for missing approval receipt")
	}
	if result.WithheldReason.FailureClass != "command_risky" {
		t.Fatalf("expected failure_class command_risky, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.Recoverability != "human_intervention" {
		t.Fatalf("expected recoverability human_intervention, got %s", result.WithheldReason.Recoverability)
	}
	if result.WithheldReason.NextAction != "request_approval" {
		t.Fatalf("expected next_action request_approval, got %s", result.WithheldReason.NextAction)
	}
}

func TestApprovalReceiptInvalidDecision(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "rm -rf dist",
					"exit_code": 0,
				},
			},
		},
		"approval_receipt": map[string]any{
			"decision": "rejected",
			"approver": "user",
			"classified_commands": []any{
				map[string]any{
					"command": "rm -rf dist",
					"risk":    "high",
				},
			},
			"aggregate_risk": "high",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "decision is \"rejected\"") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected rejected decision error, got %v", result.Errors)
	}
}

func TestApprovalReceiptInsufficientAggregateRisk(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "deep",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"state": map[string]any{
			"read_set":  []any{"r"},
			"write_set": []any{"w"},
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "go build ./...",
					"exit_code": 0,
				},
			},
			"verification_artifacts": []any{
				map[string]any{
					"kind":     "build",
					"command":  "go build ./...",
					"status":   "passed",
					"verifies": []any{"v"},
				},
			},
			"untested_regions":   []any{"u"},
			"remaining_risks":    []any{"r"},
			"rollback_policy":    []any{"rp"},
			"execution_controls": []any{"ec"},
		},
		"approval_receipt": map[string]any{
			"decision": "approved",
			"approver": "user",
			"classified_commands": []any{
				map[string]any{
					"command": "go build ./...",
					"risk":    "medium",
				},
			},
			"aggregate_risk": "low",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "aggregate_risk \"low\" is below required threshold \"medium\"") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected insufficient aggregate risk error, got %v", result.Errors)
	}
}

func TestApprovalReceiptMissingCommandCoverage(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":   "rm -rf dist",
					"exit_code": 0,
				},
			},
		},
		"approval_receipt": map[string]any{
			"decision": "approved",
			"approver": "user",
			"classified_commands": []any{
				map[string]any{
					"command": "npm test",
					"risk":    "low",
				},
			},
			"aggregate_risk": "high",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "does not cover command \"rm -rf dist\"") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing command coverage error, got %v", result.Errors)
	}
}

func TestStrictDeepMissingArtifactsProvenanceFails(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "deep",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"state": map[string]any{
			"read_set":  []any{"r"},
			"write_set": []any{"w"},
		},
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{
					"command":    "npm test",
					"exit_code":  0,
					"runner":     "local",
					"started_at": "2026-05-25T00:00:00Z",
				},
			},
			"verification_artifacts": []any{
				map[string]any{
					"command":   "npm test",
					"exit_code": 0,
					// missing runner and started_at
					"status":   "passed",
					"verifies": []any{"v"},
				},
			},
			"untested_regions":   []any{"u"},
			"remaining_risks":    []any{"r"},
			"rollback_policy":    []any{"rp"},
			"execution_controls": []any{"ec"},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, true, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	if result.BlockingPredicate != "evidence_provenance_missing" {
		t.Fatalf("expected evidence_provenance_missing predicate, got %s", result.BlockingPredicate)
	}
	foundArtifact := false
	for _, e := range result.Errors {
		if strings.Contains(e, "verification_artifacts") {
			foundArtifact = true
			break
		}
	}
	if !foundArtifact {
		t.Fatalf("expected verification_artifacts error, got %v", result.Errors)
	}
}

// TestEvidenceFloorDriftGuard compares policies/admission.yaml evidence-floor
// declarations against the Go runtime admission behavior. It fails if either
// side changes without a matching change on the other.
func TestEvidenceFloorDriftGuard(t *testing.T) {
	var policyDoc map[string]any
	if err := loader.LoadDocument("../../policies/admission.yaml", &policyDoc); err != nil {
		t.Fatalf("failed to load policies/admission.yaml: %v", err)
	}

	stringSlice := func(m map[string]any, key string) []string {
		raw := sliceInMap(m, key)
		out := make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}

	evidenceFloor := mapValue(policyDoc, "evidence_floor")
	lightRequired := stringSlice(mapValue(evidenceFloor, "light"), "required")
	lightOneOf := stringSlice(mapValue(evidenceFloor, "light"), "one_of")
	standardRequired := stringSlice(mapValue(evidenceFloor, "standard"), "required")
	deepRequired := stringSlice(mapValue(evidenceFloor, "deep"), "required")
	deepRuntime := stringSlice(mapValue(evidenceFloor, "deep"), "runtime_enforced")

	minimalCard := func(tier string) map[string]any {
		card := map[string]any{
			"schema_version": "1",
			"task_id":        "T",
			"tier":           tier,
			"owner":          "a",
			"accountable":    "b",
			"claim": map[string]any{
				"fix_status": "fixed",
				"summary":    "s",
				"evidence":   []any{"e"},
			},
			"verification": map[string]any{
				"status": "passed",
				"checks": []any{},
			},
			"admission": map[string]any{
				"outcome": "success",
			},
			"acceptance_status": "accepted",
			"handoff": map[string]any{
				"next_action": "n",
				"owner":       "o",
			},
		}
		switch tier {
		case "light":
			card["evidence"] = map[string]any{
				"files_changed": []any{"f.go"},
				"command_evidence": []any{
					map[string]any{"command": "go test", "exit_code": 0},
				},
			}
		case "standard":
			card["done_checklist"] = map[string]any{"item": true}
			card["prediction"] = map[string]any{"claim": "p"}
			card["evidence"] = map[string]any{
				"files_changed": []any{"f.go"},
				"command_evidence": []any{
					map[string]any{"command": "go test", "exit_code": 0},
				},
			}
		case "deep":
			card["done_checklist"] = map[string]any{"item": true}
			card["prediction"] = map[string]any{"claim": "p"}
			card["state"] = map[string]any{
				"read_set":  []any{"r"},
				"write_set": []any{"w"},
			}
			card["evidence"] = map[string]any{
				"files_changed": []any{"f.go"},
				"command_evidence": []any{
					map[string]any{"command": "go test", "exit_code": 0},
				},
				"verification_artifacts": []any{
					map[string]any{
						"kind":     "k",
						"command":  "go test",
						"status":   "passed",
						"verifies": []any{"v"},
					},
				},
				"untested_regions":   []any{"u"},
				"remaining_risks":    []any{"r"},
				"rollback_policy":    []any{"rp"},
				"execution_controls": []any{"ec"},
			}
		}
		return card
	}

	type fieldCheck struct {
		remove    func(card map[string]any)
		expectErr string
	}
	fieldChecks := map[string]fieldCheck{
		"files_changed": {
			remove:    func(c map[string]any) { c["evidence"].(map[string]any)["files_changed"] = []any{} },
			expectErr: "files_changed",
		},
		"command_evidence": {
			remove:    func(c map[string]any) { c["evidence"].(map[string]any)["command_evidence"] = []any{} },
			expectErr: "command_evidence",
		},
		"manual_rationale": {
			remove:    func(c map[string]any) { delete(c["evidence"].(map[string]any), "manual_rationale") },
			expectErr: "manual_rationale",
		},
		"done_checklist": {
			remove:    func(c map[string]any) { delete(c, "done_checklist") },
			expectErr: "done_checklist",
		},
		"prediction": {
			remove:    func(c map[string]any) { delete(c, "prediction") },
			expectErr: "prediction",
		},
		"evidence_scope_declared": {
			remove: func(c map[string]any) {
				c["evidence"].(map[string]any)["verification_artifacts"] = []any{
					map[string]any{"kind": "k", "command": "go test", "status": "passed"},
				}
			},
			expectErr: "evidence scope declared",
		},
		"untested_regions_declared": {
			remove:    func(c map[string]any) { c["evidence"].(map[string]any)["untested_regions"] = []any{} },
			expectErr: "untested_regions",
		},
		"remaining_risks_declared": {
			remove:    func(c map[string]any) { c["evidence"].(map[string]any)["remaining_risks"] = []any{} },
			expectErr: "remaining_risks",
		},
		"execution_controls_present": {
			remove:    func(c map[string]any) { c["evidence"].(map[string]any)["execution_controls"] = []any{} },
			expectErr: "execution_controls",
		},
		"rollback_policy_present": {
			remove:    func(c map[string]any) { c["evidence"].(map[string]any)["rollback_policy"] = []any{} },
			expectErr: "rollback_policy",
		},
		"verification_artifacts": {
			remove:    func(c map[string]any) { c["evidence"].(map[string]any)["verification_artifacts"] = []any{} },
			expectErr: "verification_artifacts",
		},
		"state.read_set": {
			remove:    func(c map[string]any) { c["state"].(map[string]any)["read_set"] = []any{} },
			expectErr: "state.read_set",
		},
		"state.write_set": {
			remove:    func(c map[string]any) { c["state"].(map[string]any)["write_set"] = []any{} },
			expectErr: "state.write_set",
		},
	}

	type tierConfig struct {
		name            string
		required        []string
		oneOf           []string
		runtimeEnforced []string
	}
	tiers := []tierConfig{
		{name: "light", required: lightRequired, oneOf: lightOneOf},
		{name: "standard", required: standardRequired},
		{name: "deep", required: deepRequired, runtimeEnforced: deepRuntime},
	}

	expectedFields := map[string]map[string]struct{}{
		"light": {
			"files_changed":    {},
			"command_evidence": {},
			"manual_rationale": {},
		},
		"standard": {
			"files_changed":    {},
			"command_evidence": {},
			"done_checklist":   {},
			"prediction":       {},
		},
		"deep": {
			"files_changed":              {},
			"command_evidence":           {},
			"evidence_scope_declared":    {},
			"untested_regions_declared":  {},
			"remaining_risks_declared":   {},
			"execution_controls_present": {},
			"rollback_policy_present":    {},
			"done_checklist":             {},
			"prediction":                 {},
			"verification_artifacts":     {},
			"state.read_set":             {},
			"state.write_set":            {},
		},
	}

	for _, tc := range tiers {
		t.Run(tc.name, func(t *testing.T) {
			allDeclared := make(map[string]struct{})
			for _, f := range tc.required {
				allDeclared[f] = struct{}{}
			}
			for _, f := range tc.oneOf {
				allDeclared[f] = struct{}{}
			}
			for _, f := range tc.runtimeEnforced {
				allDeclared[f] = struct{}{}
			}
			expected := expectedFields[tc.name]
			for f := range expected {
				if _, ok := allDeclared[f]; !ok {
					t.Fatalf("runtime expects field %q but policy does not declare it for tier %s", f, tc.name)
				}
			}
			for f := range allDeclared {
				if _, ok := expected[f]; !ok {
					t.Fatalf("policy declares field %q but runtime drift guard does not expect it for tier %s", f, tc.name)
				}
			}

			card := minimalCard(tc.name)
			result := Run(card, false, false)
			if result.Outcome != "success" {
				t.Fatalf("minimal %s card should pass; got outcome=%s errors=%v", tc.name, result.Outcome, result.Errors)
			}

			checkFields := append([]string{}, tc.required...)
			checkFields = append(checkFields, tc.runtimeEnforced...)
			for _, field := range checkFields {
				check, ok := fieldChecks[field]
				if !ok {
					t.Fatalf("drift guard missing check for field %q in tier %s", field, tc.name)
				}
				t.Run("missing_"+field, func(t *testing.T) {
					c := minimalCard(tc.name)
					check.remove(c)
					r := Run(c, false, false)
					if r.Outcome != "failed" && r.Outcome != "blocked" {
						t.Fatalf("expected failed/blocked when %s missing, got %s", field, r.Outcome)
					}
					found := false
					for _, e := range r.Errors {
						if strings.Contains(strings.ToLower(e), strings.ToLower(check.expectErr)) {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected error mentioning %q when %s missing, got %v", check.expectErr, field, r.Errors)
					}
				})
			}

			if tc.name == "light" {
				t.Run("one_of_manual_rationale_only", func(t *testing.T) {
					c := minimalCard("light")
					ev := c["evidence"].(map[string]any)
					delete(ev, "command_evidence")
					ev["manual_rationale"] = "rationale text"
					r := Run(c, false, false)
					if r.Outcome != "success" {
						t.Fatalf("light should pass with manual_rationale only; got outcome=%s errors=%v", r.Outcome, r.Errors)
					}
				})
				t.Run("one_of_neither", func(t *testing.T) {
					c := minimalCard("light")
					ev := c["evidence"].(map[string]any)
					delete(ev, "command_evidence")
					delete(ev, "manual_rationale")
					r := Run(c, false, false)
					if r.Outcome != "failed" && r.Outcome != "blocked" {
						t.Fatalf("light should fail without command_evidence or manual_rationale, got %s", r.Outcome)
					}
					found := false
					for _, e := range r.Errors {
						if strings.Contains(e, "command_evidence or manual_rationale") {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected one_of error, got %v", r.Errors)
					}
				})
			}
		})
	}
}

func TestTierGuardBlocksLightWithSchemaPath(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed": []any{"schemas/completion-card.schema.json"},
			"command_evidence": []any{
				map[string]any{"command": "go test", "exit_code": 0},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" && result.Outcome != "blocked" {
		t.Fatalf("expected failed/blocked, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "tier guard: light tier declared but high-risk files detected") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tier guard error, got %v", result.Errors)
	}
}

func TestTierGuardWarnsLightWithHighRiskCommand(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed": []any{"f"},
			"command_evidence": []any{
				map[string]any{"command": "rm -rf dist", "exit_code": 0},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	// Should still pass (warning as note, not error) for conservative false-positive control
	if result.Outcome != "success" {
		t.Fatalf("expected success for conservative light-tier command warning, got %s errors=%v", result.Outcome, result.Errors)
	}
	found := false
	for _, n := range result.Notes {
		if strings.Contains(n, "tier guard warning: light tier with high-risk command(s)") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tier guard warning in notes, got %v", result.Notes)
	}
}

func TestTierGuardWarnsStandardWithBoth(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":                "p",
			"expected_effect":      "e",
			"falsification_method": "f",
			"measurable_signal":    "m",
			"horizon":              "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"internal/admission/roles.go"},
			"command_evidence": []any{
				map[string]any{"command": "rm -rf dist", "exit_code": 0},
			},
		},
		"approval_receipt": map[string]any{
			"decision": "approved",
			"approver": "user",
			"classified_commands": []any{
				map[string]any{
					"command": "rm -rf dist",
					"risk":    "high",
				},
			},
			"aggregate_risk": "high",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	// Should still pass because it's only a warning for standard tier
	if result.Outcome != "success" {
		t.Fatalf("expected success for standard-tier warning, got %s errors=%v", result.Outcome, result.Errors)
	}
	found := false
	for _, n := range result.Notes {
		if strings.Contains(n, "tier guard warning: standard tier with both high-risk files") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tier guard warning in notes, got %v", result.Notes)
	}
}

// Verify-stage v1 auto-escalation guard tests. The guard requires
// `deep` tier for cards whose `claim.evidence.files_changed` matches
// any high-risk path pattern (schemas/, policies/, .github/workflows/,
// authority, auth, migrations/). Lower tiers are withheld with
// predicate `tier_escalation_required` unless an approved governance
// intervention is recorded. Wording and predicate must stay parity-safe
// with the TS implementation in packages/cli/src/core/admission-evidence.ts
// and the policy in policies/escalation.yaml.

func TestEscalationBlocksLightWithPolicyPath(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed":    []any{"policies/admission.yaml"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s errors=%v", result.Outcome, result.Errors)
	}
	if result.BlockingPredicate != "tier_escalation_required" {
		t.Fatalf("expected blocking_predicate tier_escalation_required, got %q", result.BlockingPredicate)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "tier escalation required") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tier escalation error, got %v", result.Errors)
	}
}

func TestEscalationBlocksStandardWithAuthPath(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist":  map[string]any{"source_of_truth_read": true},
		"prediction":      map[string]any{"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify"},
		"evidence": map[string]any{
			"files_changed":    []any{"src/auth/session.ts"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s errors=%v", result.Outcome, result.Errors)
	}
	if result.BlockingPredicate != "tier_escalation_required" {
		t.Fatalf("expected blocking_predicate tier_escalation_required, got %q", result.BlockingPredicate)
	}
}

func TestEscalationAllowsDeepWithSchemaPath(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "deep",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist":  map[string]any{"source_of_truth_read": true},
		"prediction":      map[string]any{"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify"},
		"state":           map[string]any{"read_set": []any{"schemas/x.json"}, "write_set": []any{"schemas/x.json"}},
		"evidence": map[string]any{
			"files_changed":    []any{"schemas/completion-card.schema.json"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
			"verification_artifacts": []any{
				map[string]any{
					"kind": "unit_test", "command": "go test", "status": "passed",
					"verifies": []any{"x"}, "does_not_verify": []any{"y"},
				},
			},
			"untested_regions":   []any{"no e2e"},
			"remaining_risks":    []any{"prod untested"},
			"rollback_policy":    []any{"revert commit"},
			"execution_controls": []any{"feature flag"},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success for deep + schema path, got %s errors=%v", result.Outcome, result.Errors)
	}
	if result.BlockingPredicate != "" {
		t.Fatalf("expected no blocking predicate, got %q", result.BlockingPredicate)
	}
}

func TestEscalationAllowsSafeDocsPath(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed":    []any{"docs/readme.md"},
			"manual_rationale": "doc tweak",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success for light + safe docs path, got %s errors=%v", result.Outcome, result.Errors)
	}
}

func TestEscalationBypassedByApprovedGovernance(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist":  map[string]any{"source_of_truth_read": true},
		"prediction":      map[string]any{"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify"},
		"evidence": map[string]any{
			"files_changed":    []any{"policies/admission.yaml"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"governance": map[string]any{
			"requires_human_approval": true,
			"approval_status":         "approved",
			"approver":                "maintainer",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success for standard + escalation files with approved governance, got %s errors=%v", result.Outcome, result.Errors)
	}
	found := false
	for _, n := range result.Notes {
		if strings.Contains(n, "tier escalation bypassed by approved governance intervention") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected escalation bypass note, got %v", result.Notes)
	}
}

func TestEscalationBlocksStandardWithWorkflowPath(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist":  map[string]any{"source_of_truth_read": true},
		"prediction":      map[string]any{"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify"},
		"evidence": map[string]any{
			"files_changed":    []any{".github/workflows/x-harness-verify.yml"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s errors=%v", result.Outcome, result.Errors)
	}
	if result.BlockingPredicate != "tier_escalation_required" {
		t.Fatalf("expected blocking_predicate tier_escalation_required, got %q", result.BlockingPredicate)
	}
}

func TestEscalationTaxonomy(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed":    []any{"migrations/0001_init.sql"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "n", "owner": "o"},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason for tier escalation failure")
	}
	if result.WithheldReason.FailureClass != "tier_under_escalated" {
		t.Fatalf("expected failure_class tier_under_escalated, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.FailureStage != "verify_pipeline" {
		t.Fatalf("expected failure_stage verify_pipeline, got %s", result.WithheldReason.FailureStage)
	}
	if result.WithheldReason.Recoverability != "raise_tier" {
		t.Fatalf("expected recoverability raise_tier, got %s", result.WithheldReason.Recoverability)
	}
	if result.WithheldReason.NextAction != "raise_tier_to_deep" {
		t.Fatalf("expected next_action raise_tier_to_deep, got %s", result.WithheldReason.NextAction)
	}
}

// TestContextFloorStandardMissingContextAlignment tests that standard tier fails when
// context_alignment is missing and --context-floor is enabled.
func TestContextFloorStandardMissingContextAlignment(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed for missing context_alignment, got %s errors=%v", result.Outcome, result.Errors)
	}
	found := false
	for _, e := range result.Errors {
		if e == "context_alignment is required for standard/deep tier when --context-floor is enabled" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected context_alignment missing error, got %v", result.Errors)
	}
}

// TestContextFloorStandardMissingStaleGroundChecked tests that standard tier fails when
// stale_ground_checked is false.
func TestContextFloorStandardMissingStaleGroundChecked(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"context_alignment": map[string]any{
			"stale_ground_checked":  false,
			"product_contract_refs": []any{"src/context/contract.md"},
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed for stale_ground_checked=false, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if e == "context_alignment.stale_ground_checked must be true for standard/deep tier" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected stale_ground_checked error, got %v", result.Errors)
	}
}

// TestContextFloorStandardMissingRefs tests that standard tier fails when no ref arrays are populated.
func TestContextFloorStandardMissingRefs(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"context_alignment": map[string]any{
			"stale_ground_checked":  true,
			"product_contract_refs": []any{},
			"architecture_refs":     []any{},
			"decision_refs":         []any{},
			"test_matrix_refs":      []any{},
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed for missing refs, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if e == "context_alignment must have at least one non-empty ref array (product_contract_refs, architecture_refs, decision_refs, or test_matrix_refs)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ref array error, got %v", result.Errors)
	}
}

// TestContextFloorDeepMissingContextPackID tests that deep tier fails when context_pack_id is missing.
func TestContextFloorDeepMissingContextPackID(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "deep",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"state": map[string]any{
			"read_set":  []any{"r"},
			"write_set": []any{"w"},
		},
		"context_alignment": map[string]any{
			"stale_ground_checked":         true,
			"product_contract_refs":        []any{"src/context/contract.md"},
			"unresolved_context_questions": []any{},
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
			"verification_artifacts": []any{
				map[string]any{"kind": "k", "command": "go test", "status": "passed", "verifies": []any{"v"}},
			},
			"untested_regions":   []any{"u"},
			"remaining_risks":    []any{"r"},
			"rollback_policy":    []any{"rp"},
			"execution_controls": []any{"ec"},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed for missing context_pack_id, got %s errors=%v", result.Outcome, result.Errors)
	}
	found := false
	for _, e := range result.Errors {
		if e == "context_alignment.context_pack_id is required for deep tier" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected context_pack_id error, got %v", result.Errors)
	}
}

// TestContextFloorDeepUnresolvedQuestions tests that deep tier fails when unresolved_context_questions is non-empty.
func TestContextFloorDeepUnresolvedQuestions(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "deep",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"state": map[string]any{
			"read_set":  []any{"r"},
			"write_set": []any{"w"},
		},
		"context_alignment": map[string]any{
			"stale_ground_checked":         true,
			"context_pack_id":              "ctx-123",
			"product_contract_refs":        []any{"src/context/contract.md"},
			"unresolved_context_questions": []any{"Is this the right approach?"},
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
			"verification_artifacts": []any{
				map[string]any{"kind": "k", "command": "go test", "status": "passed", "verifies": []any{"v"}},
			},
			"untested_regions":   []any{"u"},
			"remaining_risks":    []any{"r"},
			"rollback_policy":    []any{"rp"},
			"execution_controls": []any{"ec"},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed for unresolved questions, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if e == "context_alignment.unresolved_context_questions must be empty for deep tier" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unresolved_context_questions error, got %v", result.Errors)
	}
}

// TestContextFloorLightAdvisoryOnly tests that light tier does not block on context floor.
func TestContextFloorLightAdvisoryOnly(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.Outcome != "success" {
		t.Fatalf("expected success for light tier (advisory only), got %s errors=%v", result.Outcome, result.Errors)
	}
	foundNote := false
	for _, n := range result.Notes {
		if n == "context floor advisory only for light tier" {
			foundNote = true
			break
		}
	}
	if !foundNote {
		t.Fatalf("expected advisory note for light tier, got %v", result.Notes)
	}
}

// TestContextFloorNoFlagNoBlock tests that without the flag, context_alignment is not required.
func TestContextFloorNoFlagNoBlock(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success without context floor flag, got %s errors=%v", result.Outcome, result.Errors)
	}
}

// TestContextFloorMissingReferencedFile tests that standard tier fails when a referenced
// file in context_alignment does not exist.
func TestContextFloorMissingReferencedFile(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"context_alignment": map[string]any{
			"stale_ground_checked":  true,
			"product_contract_refs": []any{"nonexistent/path/contract.md"},
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed for missing referenced file, got %s errors=%v", result.Outcome, result.Errors)
	}
	found := false
	for _, e := range result.Errors {
		if e == "referenced file does not exist: nonexistent/path/contract.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing file error, got %v", result.Errors)
	}
}

// TestContextFloorTaxonomy tests that context floor failures have correct taxonomy.
func TestContextFloorTaxonomy(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"item": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"evidence": map[string]any{
			"files_changed":    []any{"f.go"},
			"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, true)
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason for context floor failure")
	}
	if result.WithheldReason.FailureClass != "context_missing" {
		t.Fatalf("expected failure_class context_missing, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.FailureStage != "context_floor" {
		t.Fatalf("expected failure_stage context_floor, got %s", result.WithheldReason.FailureStage)
	}
	if result.BlockingPredicate != "context_floor_blocked" {
		t.Fatalf("expected blocking_predicate context_floor_blocked, got %s", result.BlockingPredicate)
	}
}

func TestTierGuardBlocksLightWithAuthorityPath(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "light",
		"owner":          "a",
		"accountable":    "b",
		"evidence": map[string]any{
			"files_changed": []any{"internal/authority/roles.yaml"},
			"command_evidence": []any{
				map[string]any{"command": "go test", "exit_code": 0},
			},
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "failed" && result.Outcome != "blocked" {
		t.Fatalf("expected failed/blocked, got %s", result.Outcome)
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "tier guard: light tier declared but high-risk files detected") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tier guard error, got %v", result.Errors)
	}
}

func TestTierGuardWarnsStandardWithAuthorityPathAndHighRiskCommand(t *testing.T) {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "T",
		"tier":           "standard",
		"owner":          "a",
		"accountable":    "b",
		"done_checklist": map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim": "p", "expected_effect": "e", "falsification_method": "f", "measurable_signal": "m", "horizon": "same_verify",
		},
		"evidence": map[string]any{
			"files_changed": []any{"internal/permission/roles.go"},
			"command_evidence": []any{
				map[string]any{"command": "rm -rf dist", "exit_code": 0},
			},
		},
		"approval_receipt": map[string]any{
			"decision": "approved",
			"approver": "user",
			"classified_commands": []any{
				map[string]any{
					"command": "rm -rf dist",
					"risk":    "high",
				},
			},
			"aggregate_risk": "high",
		},
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "s",
			"evidence":   []any{"e"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission": map[string]any{
			"outcome": "success",
		},
		"acceptance_status": "accepted",
		"handoff": map[string]any{
			"next_action": "n",
			"owner":       "o",
		},
	}
	result := Run(doc, false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success for standard-tier warning, got %s errors=%v", result.Outcome, result.Errors)
	}
	found := false
	for _, n := range result.Notes {
		if strings.Contains(n, "tier guard warning: standard tier with both high-risk files") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tier guard warning in notes, got %v", result.Notes)
	}
}

// Product intent status advisory tests (advisory-only; never blocks admission).
func standardProductIntentFixture(intent map[string]any) map[string]any {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "TASK-PRODUCT-INTENT",
		"tier":           "standard",
		"owner":          "alice",
		"accountable":    "bob",
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "ok",
			"evidence":   []any{"e1"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "none", "owner": "alice"},
		"done_checklist":    map[string]any{"source_of_truth_read": true},
		"prediction": map[string]any{
			"claim":               "p",
			"expected_effect":     "e",
			"falsification_method": "f",
			"horizon":             "same_verify",
		},
		"evidence": map[string]any{
			"files_changed":    []any{"src/x.ts"},
			"command_evidence": []any{map[string]any{"command": "npm test", "exit_code": 0}},
		},
	}
	if intent != nil {
		doc["product_intent"] = intent
	}
	return doc
}

func deepProductIntentFixture(intent map[string]any) map[string]any {
	doc := standardProductIntentFixture(intent)
	doc["tier"] = "deep"
	doc["state"] = map[string]any{
		"read_set":  []any{"src/x.ts"},
		"write_set": []any{"src/x.ts"},
	}
	evidence := map[string]any{
		"files_changed":    []any{"src/x.ts"},
		"command_evidence": []any{map[string]any{"command": "npm test", "exit_code": 0}},
		"verification_artifacts": []any{
			map[string]any{
				"kind":              "unit_test",
				"command":           "npm test",
				"status":            "passed",
				"verifies":          []any{"x"},
				"does_not_verify":   []any{"y"},
			},
		},
		"untested_regions":   []any{"no e2e"},
		"remaining_risks":    []any{"prod untested"},
		"rollback_policy":    []any{"revert commit"},
		"execution_controls": []any{"feature flag"},
	}
	doc["evidence"] = evidence
	return doc
}

func lightProductIntentFixture() map[string]any {
	doc := map[string]any{
		"schema_version": "1",
		"task_id":        "TASK-PRODUCT-INTENT-LIGHT",
		"tier":           "light",
		"owner":          "alice",
		"accountable":    "bob",
		"claim": map[string]any{
			"fix_status": "fixed",
			"summary":    "ok",
			"evidence":   []any{"e1"},
		},
		"verification": map[string]any{
			"status": "passed",
			"checks": []any{},
		},
		"admission":         map[string]any{"outcome": "success"},
		"acceptance_status": "accepted",
		"handoff":           map[string]any{"next_action": "none", "owner": "alice"},
		"evidence": map[string]any{
			"files_changed":    []any{"src/x.ts"},
			"manual_rationale": "doc tweak",
		},
	}
	return doc
}

func containsNote(notes []string, fragment string) bool {
	for _, n := range notes {
		if strings.Contains(n, fragment) {
			return true
		}
	}
	return false
}

func TestProductIntentStandardMissingEmitsAdvisory(t *testing.T) {
	result := Run(standardProductIntentFixture(nil), false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success (advisory-only), got %s errors=%v", result.Outcome, result.Errors)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted (advisory-only), got %s", result.AcceptanceStatus)
	}
	if !containsNote(result.Notes, "product_intent.status not declared") {
		t.Fatalf("expected product_intent missing advisory note, got %v", result.Notes)
	}
}

func TestProductIntentStandardUnknownEmitsAdvisory(t *testing.T) {
	intent := map[string]any{"status": "unknown"}
	result := Run(standardProductIntentFixture(intent), false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success (advisory-only), got %s errors=%v", result.Outcome, result.Errors)
	}
	if !containsNote(result.Notes, "product_intent.status is unknown") {
		t.Fatalf("expected unknown advisory note, got %v", result.Notes)
	}
}

func TestProductIntentStandardAlignedNoAdvisory(t *testing.T) {
	intent := map[string]any{"status": "aligned"}
	result := Run(standardProductIntentFixture(intent), false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s errors=%v", result.Outcome, result.Errors)
	}
	if containsNote(result.Notes, "product_intent.status") {
		t.Fatalf("did not expect product_intent advisory for aligned, got %v", result.Notes)
	}
}

func TestProductIntentStandardOtherStatusesNoAdvisory(t *testing.T) {
	for _, status := range []string{"unreviewed", "disputed", "not_applicable"} {
		t.Run(status, func(t *testing.T) {
			intent := map[string]any{"status": status}
			result := Run(standardProductIntentFixture(intent), false, false)
			if result.Outcome != "success" {
				t.Fatalf("expected success for status %s, got %s errors=%v", status, result.Outcome, result.Errors)
			}
			if containsNote(result.Notes, "product_intent.status") {
				t.Fatalf("did not expect product_intent advisory for status %s, got %v", status, result.Notes)
			}
		})
	}
}

func TestProductIntentDeepMissingEmitsAdvisory(t *testing.T) {
	result := Run(deepProductIntentFixture(nil), false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success (advisory-only), got %s errors=%v", result.Outcome, result.Errors)
	}
	if !containsNote(result.Notes, "product_intent.status not declared") {
		t.Fatalf("expected product_intent missing advisory note on deep, got %v", result.Notes)
	}
}

func TestProductIntentDeepUnknownEmitsAdvisory(t *testing.T) {
	intent := map[string]any{"status": "unknown"}
	result := Run(deepProductIntentFixture(intent), false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success (advisory-only), got %s errors=%v", result.Outcome, result.Errors)
	}
	if !containsNote(result.Notes, "product_intent.status is unknown") {
		t.Fatalf("expected unknown advisory note on deep, got %v", result.Notes)
	}
}

func TestProductIntentLightNoAdvisory(t *testing.T) {
	result := Run(lightProductIntentFixture(), false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success for light, got %s errors=%v", result.Outcome, result.Errors)
	}
	if containsNote(result.Notes, "product_intent.status") {
		t.Fatalf("did not expect product_intent advisory for light, got %v", result.Notes)
	}
}

func TestProductIntentEmptyStatusTreatedAsMissing(t *testing.T) {
	intent := map[string]any{"status": "   "}
	result := Run(standardProductIntentFixture(intent), false, false)
	if result.Outcome != "success" {
		t.Fatalf("expected success, got %s errors=%v", result.Outcome, result.Errors)
	}
	if !containsNote(result.Notes, "product_intent.status not declared") {
		t.Fatalf("expected missing advisory for empty status, got %v", result.Notes)
	}
}

// TestEscalationDriftGuard compares policies/escalation.yaml
// verify_stage_escalation.v1 declarations against the Go runtime
// hardcoded copy in escalationHighRiskPathPatterns. It also verifies the
// v1 high-risk pattern list bidirectionally matches the runtime slice and
// the policy scalar fields (required_tier, blocked_predicate, bypass)
// align with the guard's expected behavior. The test fails if either the
// YAML or the runtime hardcoded copy changes without a matching change
// on the other side.
func TestEscalationDriftGuard(t *testing.T) {
	var policyDoc map[string]any
	if err := loader.LoadDocument("../../policies/escalation.yaml", &policyDoc); err != nil {
		t.Fatalf("failed to load policies/escalation.yaml: %v", err)
	}

	stringSlice := func(m map[string]any, key string) []string {
		raw := sliceInMap(m, key)
		out := make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}

	verifyStage := mapValue(mapValue(policyDoc, "verify_stage_escalation"), "v1")
	if verifyStage == nil {
		t.Fatalf("policies/escalation.yaml missing verify_stage_escalation.v1 block")
	}

	requiredTier := stringInMap(verifyStage, "required_tier")
	if requiredTier != "deep" {
		t.Fatalf("policy required_tier drift: expected \"deep\", got %q", requiredTier)
	}

	blockedPredicate := stringInMap(verifyStage, "blocked_predicate")
	if blockedPredicate != "tier_escalation_required" {
		t.Fatalf("policy blocked_predicate drift: expected \"tier_escalation_required\", got %q", blockedPredicate)
	}

	bypass := stringSlice(verifyStage, "bypass")
	bypassHasApprovedDowngrade := false
	for _, b := range bypass {
		if b == "approved_tier_downgrade" {
			bypassHasApprovedDowngrade = true
			break
		}
	}
	if !bypassHasApprovedDowngrade {
		t.Fatalf("policy bypass drift: expected bypass to include \"approved_tier_downgrade\", got %v", bypass)
	}

	yamlPatterns := stringSlice(verifyStage, "high_risk_path_patterns")
	if len(yamlPatterns) == 0 {
		t.Fatalf("policy high_risk_path_patterns is empty")
	}

	if len(yamlPatterns) != len(escalationHighRiskPathPatterns) {
		t.Fatalf(
			"pattern count drift: policy declares %d patterns (%v), runtime has %d (%v)",
			len(yamlPatterns), yamlPatterns,
			len(escalationHighRiskPathPatterns), escalationHighRiskPathPatterns,
		)
	}

	yamlSet := make(map[string]struct{}, len(yamlPatterns))
	for _, p := range yamlPatterns {
		yamlSet[p] = struct{}{}
	}
	runtimeSet := make(map[string]struct{}, len(escalationHighRiskPathPatterns))
	for _, p := range escalationHighRiskPathPatterns {
		runtimeSet[p] = struct{}{}
	}
	for _, p := range yamlPatterns {
		if _, ok := runtimeSet[p]; !ok {
			t.Fatalf("policy pattern %q is missing from runtime hardcoded list %v", p, escalationHighRiskPathPatterns)
		}
	}
	for _, p := range escalationHighRiskPathPatterns {
		if _, ok := yamlSet[p]; !ok {
			t.Fatalf("runtime hardcoded pattern %q is missing from policy list %v", p, yamlPatterns)
		}
	}

	// Behavioral check: each YAML pattern must trigger the guard for a
	// light-tier card whose files_changed contains the pattern, and a
	// safe path must not trigger it.
	for _, pattern := range yamlPatterns {
		t.Run("pattern_triggers_"+pattern, func(t *testing.T) {
			lower := strings.ToLower(pattern)
			var sample string
			if strings.HasSuffix(lower, "/") {
				sample = pattern + "drift-guard.txt"
			} else {
				// Substring patterns like "auth" or "authority" need a
				// separator after the substring to be a meaningful
				// sample file path.
				sample = pattern + "/drift-guard.txt"
			}
			doc := map[string]any{
				"schema_version": "1",
				"task_id":        "T",
				"tier":           "light",
				"owner":          "a",
				"accountable":    "b",
				"evidence": map[string]any{
					"files_changed":    []any{sample},
					"command_evidence": []any{map[string]any{"command": "go test", "exit_code": 0}},
				},
				"claim": map[string]any{
					"fix_status": "fixed",
					"summary":    "s",
					"evidence":   []any{"e"},
				},
				"verification": map[string]any{
					"status": "passed",
					"checks": []any{},
				},
				"admission":         map[string]any{"outcome": "success"},
				"acceptance_status": "accepted",
				"handoff":           map[string]any{"next_action": "n", "owner": "o"},
			}
			result := Run(doc, false, false)
			if result.BlockingPredicate != "tier_escalation_required" {
				t.Fatalf("expected blocking_predicate tier_escalation_required for pattern %q, got %q (errors=%v)",
					pattern, result.BlockingPredicate, result.Errors)
			}
		})
	}

	t.Run("safe_docs_path_does_not_trigger", func(t *testing.T) {
		doc := map[string]any{
			"schema_version": "1",
			"task_id":        "T",
			"tier":           "light",
			"owner":          "a",
			"accountable":    "b",
			"evidence": map[string]any{
				"files_changed":    []any{"docs/readme.md"},
				"manual_rationale": "doc tweak",
			},
			"claim": map[string]any{
				"fix_status": "fixed",
				"summary":    "s",
				"evidence":   []any{"e"},
			},
			"verification": map[string]any{
				"status": "passed",
				"checks": []any{},
			},
			"admission":         map[string]any{"outcome": "success"},
			"acceptance_status": "accepted",
			"handoff":           map[string]any{"next_action": "n", "owner": "o"},
		}
		result := Run(doc, false, false)
		if result.BlockingPredicate == "tier_escalation_required" {
			t.Fatalf("safe docs path should not trigger escalation, got errors=%v", result.Errors)
		}
	})
}
