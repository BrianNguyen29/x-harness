package cli

import (
	"bytes"
	"encoding/json"
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
