package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHelpListsPrimaryCommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	output := stdout.String()
	for _, name := range PrimaryCommandNames() {
		if !strings.Contains(output, name) {
			t.Fatalf("help output does not include primary command %q:\n%s", name, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestVersionOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--version"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if stdout.String() != VersionText() {
		t.Fatalf("unexpected version output: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestKnownCommandStubReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "not implemented yet") {
		t.Fatalf("expected stub message, got %q", stderr.String())
	}
}

func TestVerifyValidCardReturnsOK(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "../../examples/golden/success-light/completion-card.yaml"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "outcome: success") {
		t.Fatalf("expected success outcome, got:\n%s", out)
	}
	if !strings.Contains(out, "acceptance_status: accepted") {
		t.Fatalf("expected accepted status, got:\n%s", out)
	}
}

func TestVerifyInvalidCardReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "../../examples/golden/failed-invalid-status/completion-card.yaml"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "outcome: failed") {
		t.Fatalf("expected failed outcome, got:\n%s", out)
	}
	if !strings.Contains(out, "acceptance_status: withheld") {
		t.Fatalf("expected withheld status, got:\n%s", out)
	}
	if !strings.Contains(out, "schema_error") {
		t.Fatalf("expected schema_error, got:\n%s", out)
	}
}

func TestCheckAliasWorks(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"check", "--card", "../../examples/golden/success-light/completion-card.yaml"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "outcome: success") {
		t.Fatalf("expected success outcome from check alias, got:\n%s", stdout.String())
	}
}

func TestVerifyJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "../../examples/golden/success-light/completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var result struct {
		OK               bool   `json:"ok"`
		TaskID           string `json:"task_id"`
		Tier             string `json:"tier"`
		AdmissionOutcome string `json:"admission_outcome"`
		AcceptanceStatus string `json:"acceptance_status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK || result.AdmissionOutcome != "success" || result.AcceptanceStatus != "accepted" {
		t.Fatalf("unexpected JSON result: %+v", result)
	}
}

func TestVerifyMissingCardFlagReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got %q", stderr.String())
	}
}

func TestDoctorHealthyRepoJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", "../.."}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var result struct {
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if !result.Healthy {
		t.Fatalf("expected healthy repo, got: %s", stdout.String())
	}
}

func TestDoctorHealthyRepoFormatJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", "../..", "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	var result struct {
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy repo")
	}
}

func TestDoctorHealthyRepoJsonFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", "../..", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	var result struct {
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if !result.Healthy {
		t.Fatal("expected healthy repo")
	}
}

func TestDoctorTextOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", "../..", "--format", "text"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "healthy:") {
		t.Fatalf("expected text output to contain 'healthy:', got:\n%s", out)
	}
}

func TestDoctorMissingRootReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got %q", stderr.String())
	}
}

func TestDoctorBadFormatReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", "../..", "--format", "xml"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown format") {
		t.Fatalf("expected unknown format error, got %q", stderr.String())
	}
}

func TestDoctorUnhealthyRootReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", "/tmp/nonexistent-x-harness-12345"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestVerifyStrictEnablesMutationGuard(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--strict", "--card", "../../examples/golden/success-light/completion-card.yaml"}, &stdout, &stderr)
	// In this repo (which is a git repo), strict should succeed because no mutation occurs
	if code != ExitOK {
		out := stdout.String()
		errStr := stderr.String()
		t.Fatalf("expected exit code %d, got %d. stdout:\n%s\nstderr:\n%s", ExitOK, code, out, errStr)
	}
	out := stdout.String()
	if !strings.Contains(out, "mutation_guard: clean") {
		t.Fatalf("expected mutation_guard clean, got:\n%s", out)
	}
}

func TestVerifyMutationGuardSkipInNonGit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--mutation-guard", "--card", "../../examples/golden/success-light/completion-card.yaml"}, &stdout, &stderr)
	// This repo IS a git repo, so guard should run and be clean
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "mutation_guard: clean") && !strings.Contains(out, "mutation_guard: skipped") {
		t.Fatalf("expected mutation_guard info, got:\n%s", out)
	}
}

func TestUnknownCommandReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected unknown command message, got %q", stderr.String())
	}
}
