package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartDefaultDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"start", "--root", "../.."}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "doctor") {
		t.Fatalf("expected doctor step, got: %s", out)
	}
	if !strings.Contains(out, "examples verify") {
		t.Fatalf("expected examples verify step, got: %s", out)
	}
	if !strings.Contains(out, "init wizard") {
		t.Fatalf("expected init wizard step, got: %s", out)
	}
	if !strings.Contains(out, "Next steps") {
		t.Fatalf("expected next steps, got: %s", out)
	}
}

func TestStartJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"start", "--root", "../..", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result StartResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got: %+v", result)
	}
	if len(result.Steps) != 3 {
		t.Fatalf("expected 3 steps, got: %d", len(result.Steps))
	}
	if len(result.NextSteps) == 0 {
		t.Fatalf("expected next steps, got: %+v", result)
	}
}

func TestStartInvalidProfile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"start", "--root", "../..", "--profile", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid profile") {
		t.Fatalf("expected invalid profile error, got: %q", stderr.String())
	}
}

func TestStartSkipDoctor(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"start", "--root", "../..", "--skip-doctor", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result StartResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	for _, step := range result.Steps {
		if step.Name == "doctor" {
			t.Fatal("expected doctor to be skipped")
		}
	}
}

func TestStartSkipExamples(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"start", "--root", "../..", "--skip-examples", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result StartResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	for _, step := range result.Steps {
		if step.Name == "examples_verify" {
			t.Fatal("expected examples_verify to be skipped")
		}
	}
}

func TestStartApplyInTempDir(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"start", "--root", tmpDir, "--apply", "--profile", "minimal", "--skip-doctor", "--skip-examples", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result StartResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	found := false
	for _, step := range result.Steps {
		if step.Name == "init_wizard" {
			found = true
			if step.Status != "passed" {
				t.Fatalf("expected init_wizard passed, got: %s", step.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected init_wizard step")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to exist after apply: %v", err)
	}
}

func TestStartNoApplyDoesNotMutate(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"start", "--root", tmpDir, "--profile", "minimal", "--skip-doctor", "--skip-examples", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err == nil {
		t.Fatal("expected AGENTS.md to NOT exist in dry-run")
	}
}

func TestStartHelpIncludesStart(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "start") {
		t.Fatalf("expected help to include start, got: %s", stdout.String())
	}
}

func TestStartMaturityBeta(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help-maturity"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "start") {
		t.Fatalf("expected --help-maturity to include start, got: %s", out)
	}
	// Verify it appears under beta section (before experimental)
	betaIdx := strings.Index(out, "beta:")
	expIdx := strings.Index(out, "experimental:")
	startIdx := strings.Index(out, "start")
	if betaIdx == -1 || expIdx == -1 || startIdx == -1 {
		t.Fatalf("missing expected sections")
	}
	if startIdx < betaIdx || startIdx > expIdx {
		t.Fatalf("expected start to appear under beta section")
	}
}
