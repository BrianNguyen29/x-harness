package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xschema "github.com/BrianNguyen29/x-harness/internal/schema"
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
	validateReportJSON(t, stdout.Bytes())

	// Verify denominator contract fields are present in raw JSON
	var raw map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("expected valid JSON for raw check: %v", err)
	}
	metrics, ok := raw["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected metrics object")
	}
	vesr, ok := metrics["verify_event_success_rate"].(map[string]any)
	if !ok {
		t.Fatalf("expected verify_event_success_rate")
	}
	if vesr["unit"] != "verify_event" || vesr["not_task_level"] != true {
		t.Fatalf("unexpected verify_event_success_rate: %+v", vesr)
	}
	tcc, ok := metrics["task_completion_coverage"].(map[string]any)
	if !ok {
		t.Fatalf("expected task_completion_coverage")
	}
	if tcc["status"] != "not_computable" || tcc["reason"] != "missing_aligned_task_denominator" {
		t.Fatalf("unexpected task_completion_coverage: %+v", tcc)
	}
	wr, ok := metrics["withheld_rate"].(map[string]any)
	if !ok {
		t.Fatalf("expected withheld_rate")
	}
	if wr["unit"] != "verify_event" || wr["not_task_level"] != true {
		t.Fatalf("unexpected withheld_rate: %+v", wr)
	}
	if _, hasGeneric := metrics["success_rate"]; hasGeneric {
		t.Fatalf("generic success_rate must not be present")
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
	// Accepted card must omit withheld_reason
	var rawAdmission map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rawAdmission); err != nil {
		t.Fatalf("expected valid JSON for admission raw check: %v", err)
	}
	admissionRaw, ok := raw["admission"].(map[string]any)
	if !ok {
		t.Fatalf("expected admission object in raw JSON")
	}
	if _, hasWithheldReason := admissionRaw["withheld_reason"]; hasWithheldReason {
		t.Fatalf("accepted card must not include withheld_reason")
	}
}

func TestReportMetricsWithheldJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report", "--metrics", "--card", "../../examples/golden/blocked-missing-evidence/completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}

	var result struct {
		Admission struct {
			Outcome          string `json:"outcome"`
			AcceptanceStatus string `json:"acceptance_status"`
			WithheldReason   *struct {
				FailureClass      string `json:"failure_class"`
				FailureStage      string `json:"failure_stage"`
				Recoverability    string `json:"recoverability"`
				NextAction        string `json:"next_action"`
				BlockingPredicate string `json:"blocking_predicate"`
			} `json:"withheld_reason,omitempty"`
		} `json:"admission"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	validateReportJSON(t, stdout.Bytes())

	if result.Admission.Outcome != "failed" {
		t.Fatalf("expected admission outcome failed, got %+v", result.Admission)
	}
	if result.Admission.AcceptanceStatus != "withheld" {
		t.Fatalf("expected acceptance withheld, got %+v", result.Admission)
	}
	if result.Admission.WithheldReason == nil {
		t.Fatalf("expected withheld_reason for withheld card")
	}
	if result.Admission.WithheldReason.FailureClass == "" {
		t.Fatalf("expected failure_class in withheld_reason")
	}
	if result.Admission.WithheldReason.FailureStage == "" {
		t.Fatalf("expected failure_stage in withheld_reason")
	}
	if result.Admission.WithheldReason.Recoverability == "" {
		t.Fatalf("expected recoverability in withheld_reason")
	}
	if result.Admission.WithheldReason.NextAction == "" {
		t.Fatalf("expected next_action in withheld_reason")
	}
	if result.Admission.WithheldReason.BlockingPredicate == "" {
		t.Fatalf("expected blocking_predicate in withheld_reason")
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
		"## Rate metrics",
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

func TestReportTraceJSON(t *testing.T) {
	traceDir := t.TempDir()
	_, err := AppendTrace(TraceEvent{
		"event_id":          "VE-test-1",
		"event_type":        "verify_completed",
		"task_id":           "TASK-1",
		"tier":              "light",
		"outcome":           "success",
		"acceptance_status": "accepted",
		"created_at":        "2026-01-01T00:00:00Z",
	}, traceDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(traceDir, "events.jsonl")); err != nil {
		t.Fatalf("expected trace events file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report", "--trace-dir", traceDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		TotalEvents           int            `json:"total_events"`
		Accepted              int            `json:"accepted"`
		Withheld              int            `json:"withheld"`
		ByOutcome             map[string]int `json:"by_outcome"`
		Latest                map[string]any `json:"latest"`
		VerifyEventAccounting struct {
			TotalTraceEvents int `json:"total_trace_events"`
		} `json:"verify_event_accounting"`
		UnknownOrUnlinkedEvents struct {
			Count int `json:"count"`
		} `json:"unknown_or_unlinked_events"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	validateReportJSON(t, stdout.Bytes())
	if result.TotalEvents != 1 || result.Accepted != 1 || result.Withheld != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if result.ByOutcome["success"] != 1 {
		t.Fatalf("expected success outcome count, got %v", result.ByOutcome)
	}
	if result.VerifyEventAccounting.TotalTraceEvents != 1 {
		t.Fatalf("expected total_trace_events 1, got %d", result.VerifyEventAccounting.TotalTraceEvents)
	}
	if result.UnknownOrUnlinkedEvents.Count != 0 {
		t.Fatalf("expected unknown count 0, got %d", result.UnknownOrUnlinkedEvents.Count)
	}
	if result.Latest["event_type"] != "verify_completed" {
		t.Fatalf("expected latest verify event, got %v", result.Latest)
	}
}

func validateReportJSON(t *testing.T, data []byte) {
	t.Helper()
	validator, err := xschema.Compile(filepath.Join("..", "..", "schemas", "report.schema.json"))
	if err != nil {
		t.Fatalf("failed to compile report schema: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to unmarshal report JSON: %v", err)
	}
	if err := validator.Validate(doc); err != nil {
		t.Fatalf("report JSON failed schema validation: %v\n%s", err, string(data))
	}
}

func TestReportTraceMarkdown(t *testing.T) {
	traceDir := t.TempDir()
	if _, err := AppendTrace(TraceEvent{
		"event_id":          "VE-test-1",
		"event_type":        "verify_completed",
		"task_id":           "TASK-1",
		"tier":              "light",
		"outcome":           "blocked",
		"acceptance_status": "withheld",
		"created_at":        "2026-01-01T00:00:00Z",
	}, traceDir); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report", "--trace-dir", traceDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	for _, expected := range []string{
		"# x-harness Report",
		"## Verify event accounting",
		"- total_trace_events: 1",
		"- blocked: 1/1",
		"## Rate metrics",
		"- withheld_rate: 1/1 verify_event (not_task_level)",
		"## Withheld accounting",
		"> Verify-event success must not be interpreted as task-level success",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", expected, out)
		}
	}
}

func TestReportMetricsGoldenFixtures(t *testing.T) {
	cases := []struct {
		cardDir      string
		fixtureName  string
		expectReason bool
	}{
		{
			cardDir:      "success-standard-scoped-evidence",
			fixtureName:  "expected-report-metrics.json",
			expectReason: false,
		},
		{
			cardDir:      "blocked-missing-evidence",
			fixtureName:  "expected-report-metrics.json",
			expectReason: true,
		},
	}

	for _, c := range cases {
		t.Run(c.cardDir, func(t *testing.T) {
			cardPath := filepath.Join("..", "..", "examples", "golden", c.cardDir, "completion-card.yaml")
			fixturePath := filepath.Join("..", "..", "examples", "golden", c.cardDir, c.fixtureName)

			fixtureData, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("failed to read fixture %s: %v", fixturePath, err)
			}
			var expected map[string]any
			if err := json.Unmarshal(fixtureData, &expected); err != nil {
				t.Fatalf("failed to parse fixture %s: %v", fixturePath, err)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run([]string{"report", "--metrics", "--card", cardPath, "--json"}, &stdout, &stderr)
			if code != ExitOK {
				t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
			}

			var actual map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &actual); err != nil {
				t.Fatalf("failed to parse report JSON: %v\noutput: %s", err, stdout.String())
			}
			validateReportJSON(t, stdout.Bytes())

			// Validate denominator-safe metrics from fixture
			expMetrics, _ := expected["metrics"].(map[string]any)
			actMetrics, _ := actual["metrics"].(map[string]any)
			if actMetrics == nil {
				t.Fatalf("missing metrics in report output")
			}

			for _, key := range []string{"verify_event_success_rate", "task_completion_coverage", "withheld_rate"} {
				expVal, ok := expMetrics[key]
				if !ok {
					continue
				}
				actVal, ok := actMetrics[key].(map[string]any)
				if !ok {
					t.Fatalf("expected %s in metrics", key)
				}
				expMap, _ := expVal.(map[string]any)
				for k, v := range expMap {
					if actVal[k] != v {
						t.Fatalf("metrics.%s.%s mismatch: expected %v, got %v", key, k, v, actVal[k])
					}
				}
			}

			// Validate admission fields from fixture
			expAdmission, _ := expected["admission"].(map[string]any)
			actAdmission, _ := actual["admission"].(map[string]any)
			if actAdmission == nil {
				t.Fatalf("missing admission in report output")
			}
			for _, k := range []string{"outcome", "acceptance_status"} {
				if actAdmission[k] != expAdmission[k] {
					t.Fatalf("admission.%s mismatch: expected %v, got %v", k, expAdmission[k], actAdmission[k])
				}
			}

			if c.expectReason {
				actReason, ok := actAdmission["withheld_reason"].(map[string]any)
				if !ok {
					t.Fatalf("expected withheld_reason in admission")
				}
				expReason, _ := expAdmission["withheld_reason"].(map[string]any)
				for _, k := range []string{"failure_class", "failure_stage", "recoverability", "next_action", "blocking_predicate"} {
					if actReason[k] != expReason[k] {
						t.Fatalf("withheld_reason.%s mismatch: expected %v, got %v", k, expReason[k], actReason[k])
					}
				}
			} else {
				if _, has := actAdmission["withheld_reason"]; has {
					t.Fatalf("accepted card must not include withheld_reason")
				}
			}

			// Validate denominator_warning presence
			expWarn, _ := expected["denominator_warning"].(string)
			actWarn, _ := actual["denominator_warning"].(string)
			if !strings.Contains(actWarn, expWarn) {
				t.Fatalf("denominator_warning mismatch: expected to contain %q, got %q", expWarn, actWarn)
			}
		})
	}
}
