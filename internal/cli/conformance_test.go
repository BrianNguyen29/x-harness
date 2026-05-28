package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestConformanceMinimalPasses(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"conformance", "run", "--profile", "minimal", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Profile string `json:"profile"`
		OK      bool   `json:"ok"`
		Checks  []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Note   string `json:"note"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Profile != "minimal" {
		t.Fatalf("expected profile minimal, got %s", result.Profile)
	}
	if !result.OK {
		t.Fatalf("expected conformance to pass, got: %+v", result)
	}

	expectedChecks := []string{
		"critical_files_exist",
		"schemas_compile",
		"policies_parse",
		"agents_managed_context",
		"golden_success_light",
		"golden_blocked_missing_evidence",
	}
	found := map[string]bool{}
	for _, c := range result.Checks {
		found[c.Name] = true
	}
	for _, name := range expectedChecks {
		if !found[name] {
			t.Fatalf("expected check %s to be present", name)
		}
	}
}

func TestConformanceMinimalTextOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"conformance", "run", "--profile", "minimal"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "profile: minimal") {
		t.Fatalf("expected profile in output, got: %s", out)
	}
	if !strings.Contains(out, "ok: true") {
		t.Fatalf("expected ok=true in output, got: %s", out)
	}
}

func TestConformanceMissingProfile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"conformance", "run"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}

func TestConformanceUnknownProfile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"conformance", "run", "--profile", "strict"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown profile") {
		t.Fatalf("expected unknown profile error, got: %s", stderr.String())
	}
}

func TestConformanceUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"conformance", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown conformance subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}
