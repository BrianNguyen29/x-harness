package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffLightTier(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "light", "--title", "Fix bug"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "SUBAGENT_TASK light") {
		t.Fatalf("expected SUBAGENT_TASK light, got:\n%s", out)
	}
	if !strings.Contains(out, "Fix bug") {
		t.Fatalf("expected title Fix bug, got:\n%s", out)
	}
	if !strings.Contains(out, "next_action") {
		t.Fatalf("expected next_action in output, got:\n%s", out)
	}
}

func TestHandoffStandardTierWithTask(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "standard", "--title", "Refactor auth", "--task", "Split auth module into services."}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "SUBAGENT_TASK standard") {
		t.Fatalf("expected SUBAGENT_TASK standard, got:\n%s", out)
	}
	if !strings.Contains(out, "Refactor auth") {
		t.Fatalf("expected title Refactor auth, got:\n%s", out)
	}
	if !strings.Contains(out, "Split auth module into services.") {
		t.Fatalf("expected task description, got:\n%s", out)
	}
}

func TestHandoffDeepTierDefaultTitle(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "deep"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "SUBAGENT_TASK deep") {
		t.Fatalf("expected SUBAGENT_TASK deep, got:\n%s", out)
	}
	if !strings.Contains(out, "Untitled") {
		t.Fatalf("expected default title Untitled, got:\n%s", out)
	}
}

func TestHandoffStandardIncludesContextByDefault(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "standard", "--title", "Test"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "## Context") {
		t.Fatalf("expected ## Context header, got:\n%s", out)
	}
	if !strings.Contains(out, "Completion is admitted, not claimed.") {
		t.Fatalf("expected context fact, got:\n%s", out)
	}
}

func TestHandoffStandardOmitsContextWithFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "standard", "--title", "Test", "--no-context"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if strings.Contains(out, "## Context") {
		t.Fatalf("expected no ## Context header, got:\n%s", out)
	}
	if strings.Contains(out, "Completion is admitted, not claimed.") {
		t.Fatalf("expected no context fact, got:\n%s", out)
	}
}

func TestHandoffUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "invalid"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown handoff subcommand") {
		t.Fatalf("expected unknown subcommand error, got:\n%s", stderr.String())
	}
}

func TestHandoffMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got:\n%s", stderr.String())
	}
}

func TestHandoffReadinessJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "readiness", "--root", "../..", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}

	var result struct {
		Ready  bool `json:"ready"`
		Checks []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
			Note   string `json:"note"`
		} `json:"checks"`
		Readiness struct {
			Proceed       bool   `json:"proceed"`
			SuggestedTier string `json:"suggested_tier"`
		} `json:"readiness"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.Ready {
		t.Fatalf("expected ready repo, got: %s", stdout.String())
	}
	if result.Readiness.SuggestedTier != "standard" {
		t.Fatalf("expected suggested tier standard, got: %s", result.Readiness.SuggestedTier)
	}
	foundInteractive := false
	for _, c := range result.Checks {
		if c.Name == "interactive_prompts" {
			foundInteractive = true
		}
	}
	if !foundInteractive {
		t.Fatalf("expected interactive_prompts check in JSON")
	}
}

func TestHandoffReadinessText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "readiness", "--root", "../.."}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "handoff readiness: READY") {
		t.Fatalf("expected READY status, got:\n%s", out)
	}
	if !strings.Contains(out, "suggested_tier: standard") {
		t.Fatalf("expected suggested tier standard, got:\n%s", out)
	}
}

func TestHandoffReadinessUnhealthyRoot(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "readiness", "--root", "/tmp/nonexistent-x-harness-12345"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestPrepareAliasWorks(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prepare", "--root", "../..", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}

	var result struct {
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if !result.Ready {
		t.Fatal("expected healthy repo from prepare alias")
	}
}

func TestPrepareAliasText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prepare", "--root", "../.."}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "handoff readiness: READY") {
		t.Fatalf("expected READY from prepare alias, got:\n%s", stdout.String())
	}
}

func TestReadinessNonInteractiveMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "readiness", "--root", "../..", "--non-interactive"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Non-interactive mode: skipping readiness prompts") {
		t.Fatalf("expected non-interactive note, got:\n%s", out)
	}
}

func TestHandoffTierNoContextFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "light", "--no-context", "--title", "T"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if strings.Contains(out, "## Context") {
		t.Fatalf("expected no context header with --no-context, got:\n%s", out)
	}
}

func TestReadinessWithMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "readiness", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result struct {
		Ready     bool `json:"ready"`
		Readiness struct {
			SuggestedTier string `json:"suggested_tier"`
		} `json:"readiness"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if result.Ready {
		t.Fatal("expected not ready for empty tmp dir")
	}
	if result.Readiness.SuggestedTier != "light" {
		t.Fatalf("expected suggested tier light, got: %s", result.Readiness.SuggestedTier)
	}
}

func TestReadinessPartialFiles(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# agents\n"), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "policies"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "policies", "admission.yaml"), []byte("policy: true\n"), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "templates"), 0755)
	// Missing COMPLETION_CARD.md

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "readiness", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result struct {
		Ready  bool `json:"ready"`
		Checks []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if result.Ready {
		t.Fatal("expected not ready")
	}
	var completionCardCheck struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
	}
	for _, c := range result.Checks {
		if c.Name == "completion_card_template_present" {
			completionCardCheck = c
		}
	}
	if completionCardCheck.Passed {
		t.Fatal("expected completion_card_template_present to fail")
	}
}

func TestHandoffHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"handoff", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}
