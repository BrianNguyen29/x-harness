package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestReportMissingCardReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report", "--metrics"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "Completion card not found") {
		t.Fatalf("expected missing card message, got %q", stderr.String())
	}
}

func TestReportMetricsJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report", "--metrics", "--card", "../../examples/golden/success-standard-scoped-evidence/completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}

	var result struct {
		Metrics struct {
			VerificationStrength struct {
				CommandEvidenceCount int      `json:"command_evidence_count"`
				OracleKinds          []string `json:"oracle_kinds"`
			} `json:"verification_strength"`
		} `json:"metrics"`
		Admission struct {
			Outcome          string `json:"outcome"`
			AcceptanceStatus string `json:"acceptance_status"`
		} `json:"admission"`
		VerifyEventAccounting struct {
			CardsAnalyzed int `json:"cards_analyzed"`
		} `json:"verify_event_accounting"`
		TaskLifecycleAccounting struct {
			Admitted int `json:"admitted"`
			Withheld int `json:"withheld"`
		} `json:"task_lifecycle_accounting"`
		AdmissionAccounting struct {
			Accepted      int `json:"accepted"`
			TotalAnalyzed int `json:"total_analyzed"`
		} `json:"admission_accounting"`
		WithheldAccounting struct {
			Failed  int `json:"failed"`
			Blocked int `json:"blocked"`
			Skipped int `json:"skipped"`
			Timeout int `json:"timeout"`
			Error   int `json:"error"`
		} `json:"withheld_accounting"`
		UnknownOrUnlinkedEvents struct {
			Count int `json:"count"`
		} `json:"unknown_or_unlinked_events"`
		DenominatorWarning string `json:"denominator_warning"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result.Admission.Outcome != "success" {
		t.Fatalf("expected admission outcome success, got %+v", result.Admission)
	}
	if result.Admission.AcceptanceStatus != "accepted" {
		t.Fatalf("expected acceptance accepted, got %+v", result.Admission)
	}
	if result.VerifyEventAccounting.CardsAnalyzed != 1 {
		t.Fatalf("expected cards_analyzed 1, got %d", result.VerifyEventAccounting.CardsAnalyzed)
	}
	if result.TaskLifecycleAccounting.Admitted != 1 {
		t.Fatalf("expected admitted 1, got %d", result.TaskLifecycleAccounting.Admitted)
	}
	if result.AdmissionAccounting.Accepted != 1 {
		t.Fatalf("expected accepted 1, got %d", result.AdmissionAccounting.Accepted)
	}
	if result.Metrics.VerificationStrength.CommandEvidenceCount != 2 {
		t.Fatalf("expected command_evidence_count 2, got %d", result.Metrics.VerificationStrength.CommandEvidenceCount)
	}
	if len(result.Metrics.VerificationStrength.OracleKinds) != 2 {
		t.Fatalf("expected 2 oracle kinds, got %d", len(result.Metrics.VerificationStrength.OracleKinds))
	}
	hasUnitTest := false
	hasTypecheck := false
	for _, k := range result.Metrics.VerificationStrength.OracleKinds {
		if k == "unit_test" {
			hasUnitTest = true
		}
		if k == "typecheck" {
			hasTypecheck = true
		}
	}
	if !hasUnitTest || !hasTypecheck {
		t.Fatalf("expected oracle kinds unit_test and typecheck, got %v", result.Metrics.VerificationStrength.OracleKinds)
	}
	if result.UnknownOrUnlinkedEvents.Count != 0 {
		t.Fatalf("expected unknown count 0, got %d", result.UnknownOrUnlinkedEvents.Count)
	}
	if result.DenominatorWarning == "" {
		t.Fatalf("expected denominator warning, got empty")
	}
}

func TestReportMetricsMarkdown(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report", "--metrics", "--card", "../../examples/golden/success-standard-scoped-evidence/completion-card.yaml"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()

	headings := []string{
		"# x-harness Metrics Report",
		"## Verification strength",
		"## State consistency",
		"## Recovery ability",
		"## Replayability",
		"## Cost",
		"## Verify event accounting",
		"## Task lifecycle accounting",
		"## Admission accounting",
		"## Withheld accounting",
		"## Unknown or unlinked events",
		"## Denominator warning",
	}
	for _, h := range headings {
		if !strings.Contains(out, h) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", h, out)
		}
	}

	if !strings.Contains(out, "command_evidence_count: 2") {
		t.Fatalf("expected command_evidence_count 2, got:\n%s", out)
	}
	if !strings.Contains(out, "unit_test") || !strings.Contains(out, "typecheck") {
		t.Fatalf("expected oracle_kinds to contain unit_test and typecheck, got:\n%s", out)
	}
	if !strings.Contains(out, "admitted: 1/1") {
		t.Fatalf("expected admitted 1/1, got:\n%s", out)
	}
	if !strings.Contains(out, "> Verify-event success must not be interpreted as task-level success") {
		t.Fatalf("expected denominator warning, got:\n%s", out)
	}
}
