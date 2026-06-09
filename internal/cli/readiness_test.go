package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestReadinessTaskValidCard(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"readiness", "task", "--card", "../../examples/golden/regression/success-light/completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		ReadinessLevel   string `json:"readiness_level"`
		OK               bool   `json:"ok"`
		AdmissionOutcome string `json:"admission_outcome"`
		AcceptanceStatus string `json:"acceptance_status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.ReadinessLevel != "task" {
		t.Fatalf("expected readiness_level task, got %s", result.ReadinessLevel)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got: %+v", result)
	}
	if result.AdmissionOutcome != "success" {
		t.Fatalf("expected success outcome, got %s", result.AdmissionOutcome)
	}
	if result.AcceptanceStatus != "accepted" {
		t.Fatalf("expected accepted, got %s", result.AcceptanceStatus)
	}
}

func TestReadinessTaskInvalidCard(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"readiness", "task", "--card", "../../examples/golden/regression/blocked-missing-evidence/completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result struct {
		OK               bool   `json:"ok"`
		AcceptanceStatus string `json:"acceptance_status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected ok=false for invalid card")
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
}

func TestReadinessTaskMissingCard(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "task"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}

func TestReadinessTaskCardFlagMissingValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "task", "--card", "--json"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}

func TestReadinessPRValidCard(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"readiness", "pr", "--card", "../../examples/golden/regression/success-light/completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		ReadinessLevel   string `json:"readiness_level"`
		OK               bool   `json:"ok"`
		AdmissionOutcome string `json:"admission_outcome"`
		AcceptanceStatus string `json:"acceptance_status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.ReadinessLevel != "pr" {
		t.Fatalf("expected readiness_level pr, got %s", result.ReadinessLevel)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got: %+v", result)
	}
}

func TestReadinessPRMissingCard(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "pr"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}

func TestReadinessReleaseJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"readiness", "release", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		ReadinessLevel   string `json:"readiness_level"`
		OK               bool   `json:"ok"`
		AdmissionOutcome string `json:"admission_outcome"`
		AcceptanceStatus string `json:"acceptance_status"`
		Note             string `json:"note"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.ReadinessLevel != "release" {
		t.Fatalf("expected readiness_level release, got %s", result.ReadinessLevel)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got: %+v", result)
	}
	if !strings.Contains(result.Note, "local evidence generation/verification available") {
		t.Fatalf("expected note about local evidence generation/verification available, got: %s", result.Note)
	}
}

func TestReadinessReleaseText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"readiness", "release"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "readiness_level: release") {
		t.Fatalf("expected readiness_level in output, got: %s", out)
	}
	if !strings.Contains(out, "local evidence generation/verification available") {
		t.Fatalf("expected note about local evidence generation/verification available, got: %s", out)
	}
}

func TestReadinessUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown readiness subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestPrepareStillWorks(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prepare", "--root", "../.."}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "handoff readiness:") {
		t.Fatalf("expected handoff readiness output, got: %s", out)
	}
}
