package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunList(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--list"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "builtin:ci") {
		t.Fatalf("expected builtin:ci in list, got: %s", out)
	}
}

func TestRunListJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--list", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	recipes, ok := result["recipes"].([]any)
	if !ok || len(recipes) == 0 {
		t.Fatalf("expected recipes array, got: %+v", result)
	}
	found := false
	for _, r := range recipes {
		if r == "builtin:ci" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected builtin:ci in recipes, got: %+v", result)
	}
}

func TestRunDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "builtin:ci", "--dry-run"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "doctor") {
		t.Fatalf("expected doctor step, got: %s", out)
	}
	if !strings.Contains(out, "examples_verify") {
		t.Fatalf("expected examples_verify step, got: %s", out)
	}
	if !strings.Contains(out, "verify_ci_standard") {
		t.Fatalf("expected verify_ci_standard step, got: %s", out)
	}
}

func TestRunDryRunJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "builtin:ci", "--dry-run", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result RunResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Recipe != "builtin:ci" {
		t.Fatalf("expected recipe builtin:ci, got: %s", result.Recipe)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got: %+v", result)
	}
	if len(result.Steps) == 0 {
		t.Fatalf("expected steps, got: %+v", result)
	}
}

func TestRunUnknownRecipe(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown recipe") {
		t.Fatalf("expected unknown recipe error, got: %q", stderr.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"run", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %q", stderr.String())
	}
}

func TestRunInHelpListing(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "run") {
		t.Fatalf("expected help to include run, got: %s", stdout.String())
	}
}

func TestRunMaturityBeta(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help-maturity"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "run") {
		t.Fatalf("expected --help-maturity to include run, got: %s", out)
	}
	betaIdx := strings.Index(out, "beta:")
	expIdx := strings.Index(out, "experimental:")
	// Look for the line that starts with "  run" inside the beta block
	runIdx := strings.Index(out, "\n  run")
	if betaIdx == -1 || expIdx == -1 || runIdx == -1 {
		t.Fatalf("missing expected sections")
	}
	if runIdx < betaIdx || runIdx > expIdx {
		t.Fatalf("expected run to appear under beta section")
	}
}
