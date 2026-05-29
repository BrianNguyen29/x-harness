package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRecoverMarkdownOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--errors", "verification.status failed; missing evidence"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# Recovery Playbook (Review Required)") {
		t.Fatalf("expected playbook header, got:\n%s", out)
	}
	if !strings.Contains(out, "evidence_missing") {
		t.Fatalf("expected evidence_missing predicate, got:\n%s", out)
	}
	if !strings.Contains(out, "admission_failed") {
		t.Fatalf("expected admission_failed predicate, got:\n%s", out)
	}
}

func TestRecoverJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--errors", "verification.status failed; missing evidence", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Suggestions []struct {
			Predicate string `json:"predicate"`
			Route     struct {
				NextAction string `json:"next_action"`
				Owner      string `json:"owner"`
			} `json:"route"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d: %+v", len(result.Suggestions), result)
	}
	found := make(map[string]bool)
	for _, s := range result.Suggestions {
		found[s.Predicate] = true
	}
	if !found["evidence_missing"] {
		t.Fatalf("expected evidence_missing in suggestions: %+v", result)
	}
	if !found["admission_failed"] {
		t.Fatalf("expected admission_failed in suggestions: %+v", result)
	}
}

func TestRecoverUnknownFlagReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--force"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestRecoverSuccessOutcomeReturnsEmpty(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--errors", "something", "--outcome", "success"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "No recovery actions suggested.") {
		t.Fatalf("expected no suggestions for success outcome, got:\n%s", stdout.String())
	}
}

func TestRecoverySuggestJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recovery", "suggest", "--errors", "missing evidence", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Suggestions []struct {
			Predicate string `json:"predicate"`
			Route     struct {
				NextAction string `json:"next_action"`
				Owner      string `json:"owner"`
			} `json:"route"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d: %+v", len(result.Suggestions), result)
	}
	if result.Suggestions[0].Predicate != "evidence_missing" {
		t.Fatalf("expected evidence_missing, got %s", result.Suggestions[0].Predicate)
	}
}

func TestRecoveryUnsupportedSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recovery", "plan"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestRecoveryMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recovery"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestRecoverAutoNoTrace(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	traceDir := t.TempDir()
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "No trace events found") {
		t.Fatalf("expected no-trace message, got:\n%s", out)
	}
	if strings.Contains(out, "## ") {
		t.Fatalf("expected no suggestion sections, got:\n%s", out)
	}
}

func TestRecoverAutoAllSuccess(t *testing.T) {
	traceDir := t.TempDir()
	event := map[string]interface{}{
		"event_id":   "VE-1",
		"event_type": "verify_completed",
		"task_id":    "T1",
		"outcome":    "success",
		"created_at": "2026-05-29T00:00:00Z",
	}
	if _, err := AppendTrace(event, traceDir); err != nil {
		t.Fatalf("failed to append trace: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "No failures detected in trace") {
		t.Fatalf("expected no-failure message, got:\n%s", out)
	}
}

func TestRecoverAutoWithTrace(t *testing.T) {
	traceDir := t.TempDir()
	event := map[string]interface{}{
		"event_id":             "VE-1",
		"event_type":           "verify_completed",
		"task_id":              "T1",
		"outcome":              "failed",
		"blocking_predicate":   "admission_failed",
		"blocked_reason_class": "schema_or_policy_invalid",
		"notes":                []interface{}{"missing evidence", "typecheck failed"},
		"created_at":           "2026-05-29T00:00:00Z",
	}
	if _, err := AppendTrace(event, traceDir); err != nil {
		t.Fatalf("failed to append trace: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Suggestions []struct {
			Predicate string `json:"predicate"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Suggestions) == 0 {
		t.Fatalf("expected suggestions, got none")
	}
	foundEvidence := false
	foundTypecheck := false
	for _, s := range result.Suggestions {
		if s.Predicate == "evidence_missing" {
			foundEvidence = true
		}
		if s.Predicate == "typecheck_failed" {
			foundTypecheck = true
		}
	}
	if !foundEvidence {
		t.Fatalf("expected evidence_missing suggestion, got: %+v", result.Suggestions)
	}
	if !foundTypecheck {
		t.Fatalf("expected typecheck_failed suggestion, got: %+v", result.Suggestions)
	}
}

func TestRecoverAutoReadOnlyDoesNotMutate(t *testing.T) {
	traceDir := t.TempDir()
	event := map[string]interface{}{
		"event_id":   "VE-1",
		"event_type": "verify_completed",
		"task_id":    "T1",
		"outcome":    "failed",
		"notes":      []interface{}{"missing evidence"},
		"created_at": "2026-05-29T00:00:00Z",
	}
	if _, err := AppendTrace(event, traceDir); err != nil {
		t.Fatalf("failed to append trace: %v", err)
	}

	// Capture pre-run file listing
	entriesBefore, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace dir: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	entriesAfter, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace dir after: %v", err)
	}
	if len(entriesAfter) != len(entriesBefore) {
		t.Fatalf("recover --auto mutated trace directory: before=%d after=%d", len(entriesBefore), len(entriesAfter))
	}
	out := stdout.String()
	if !strings.Contains(out, "Read-Only") {
		t.Fatalf("expected read-only label in output, got:\n%s", out)
	}
}

func TestRecoverDeterministicHeuristic(t *testing.T) {
	var stdout1 bytes.Buffer
	var stdout2 bytes.Buffer
	var stderr bytes.Buffer
	code1 := Run([]string{"recover", "--errors", "typecheck failed", "--json"}, &stdout1, &stderr)
	code2 := Run([]string{"recover", "--errors", "typecheck failed", "--json"}, &stdout2, &stderr)
	if code1 != ExitOK || code2 != ExitOK {
		t.Fatalf("expected both to succeed")
	}
	if stdout1.String() != stdout2.String() {
		t.Fatalf("expected deterministic output, got diff:\n%s\nvs\n%s", stdout1.String(), stdout2.String())
	}
}
