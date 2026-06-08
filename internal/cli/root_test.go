package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHelpListsBeginnerCommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	output := stdout.String()
	beginner := []string{"check", "prepare", "recover", "doctor", "actions", "status", "reset", "init", "add", "start"}
	for _, name := range beginner {
		if !strings.Contains(output, name) {
			t.Fatalf("help output does not include beginner command %q:\n%s", name, output)
		}
	}
	advanced := []string{"benchmark", "packet", "intake", "governance", "prediction", "components", "federation", "contract"}
	for _, name := range advanced {
		if strings.Contains(output, name) {
			t.Fatalf("help output should not include advanced command %q:\n%s", name, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestHelpAllListsAllCommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help-all"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	output := stdout.String()
	if !strings.Contains(output, "check") {
		t.Fatalf("--help-all missing beginner command check:\n%s", output)
	}
	if !strings.Contains(output, "verify") {
		t.Fatalf("--help-all missing advanced command verify:\n%s", output)
	}
	if !strings.Contains(output, "packet") {
		t.Fatalf("--help-all missing advanced command packet:\n%s", output)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestNoArgsShowsStartHere(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	output := stdout.String()
	if !strings.Contains(output, "Start here") {
		t.Fatalf("no-args output missing 'Start here':\n%s", output)
	}
	if !strings.Contains(output, "check") {
		t.Fatalf("no-args output missing 'check':\n%s", output)
	}
	if !strings.Contains(output, "--help-all") {
		t.Fatalf("no-args output missing '--help-all':\n%s", output)
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

func TestReportCommandRendersTraceSummary(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"report"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "# x-harness Report") {
		t.Fatalf("expected report output, got %q", stdout.String())
	}
}

func TestVerifyValidCardReturnsOK(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "../../examples/golden/regression/success-light/completion-card.yaml"}, &stdout, &stderr)
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
	code := Run([]string{"verify", "--card", "../../examples/golden/regression/failed-invalid-status/completion-card.yaml"}, &stdout, &stderr)
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
	code := Run([]string{"check", "--card", "../../examples/golden/regression/success-light/completion-card.yaml"}, &stdout, &stderr)
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
	code := Run([]string{"verify", "--card", "../../examples/golden/regression/success-light/completion-card.yaml", "--json"}, &stdout, &stderr)
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
	code := Run([]string{"verify", "--strict", "--card", "../../examples/golden/regression/success-light/completion-card.yaml"}, &stdout, &stderr)
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
	code := Run([]string{"verify", "--mutation-guard", "--card", "../../examples/golden/regression/success-light/completion-card.yaml"}, &stdout, &stderr)
	// This repo IS a git repo, so guard should run and be clean
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "mutation_guard: clean") && !strings.Contains(out, "mutation_guard: skipped") {
		t.Fatalf("expected mutation_guard info, got:\n%s", out)
	}
}

func TestActionsListsBeginnerActions(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"actions"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	expected := []string{"prepare", "check", "recover", "doctor", "actions", "status", "reset", "init", "add", "start"}
	for _, name := range expected {
		if !strings.Contains(out, name) {
			t.Fatalf("actions output missing %q:\n%s", name, out)
		}
	}
}

func TestHelpMaturityOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help-maturity"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Maturity labels:") {
		t.Fatalf("--help-maturity missing maturity labels header:\n%s", out)
	}
	if !strings.Contains(out, "stable:") {
		t.Fatalf("--help-maturity missing stable group:\n%s", out)
	}
	if !strings.Contains(out, "check") {
		t.Fatalf("--help-maturity missing command check:\n%s", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
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
