package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/agentprofile"
)

func setupAgentProfileTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	schemaDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "agent-profile",
  "type": "object",
  "required": [
    "schema_version",
    "agent_id",
    "measured_on",
    "observed_failure_modes",
    "required_extra_checks",
    "benchmark_metrics",
    "advisory_only",
    "admission_authority"
  ],
  "properties": {
    "schema_version": { "const": "1" },
    "agent_id": { "type": "string", "minLength": 1 },
    "measured_on": { "type": "string", "minLength": 1 },
    "observed_failure_modes": { "type": "array", "items": { "type": "string" } },
    "required_extra_checks": { "type": "array", "items": { "type": "string" } },
    "benchmark_metrics": { "type": "object", "additionalProperties": true },
    "advisory_only": { "const": true },
    "admission_authority": { "const": false }
  },
  "additionalProperties": false
}`
	if err := os.WriteFile(filepath.Join(schemaDir, "agent-profile.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestAgentProfileUpdateTextOutput(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "update", "--agent", "test-agent", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# x-harness Agent Profile: test-agent") {
		t.Fatalf("expected profile header, got: %s", out)
	}
	if !strings.Contains(out, "- observed_failure_modes: 0") {
		t.Fatalf("expected 0 failure modes, got: %s", out)
	}
	if !strings.Contains(out, "- required_extra_checks: standard_verify_gate") {
		t.Fatalf("expected standard_verify_gate, got: %s", out)
	}
	expectedPath := agentprofile.DefaultAgentProfilePath(tmpDir, "test-agent")
	if !strings.Contains(out, expectedPath) {
		t.Fatalf("expected path %s, got: %s", expectedPath, out)
	}
}

func TestAgentProfileUpdateJSONOutput(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "update", "--agent", "test-agent", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	profile, ok := result["profile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected profile object, got: %v", result)
	}
	if profile["agent_id"] != "test-agent" {
		t.Fatalf("expected agent_id test-agent, got: %v", profile)
	}
}

func TestAgentProfileUpdateMissingAgent(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "update", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--agent <id> is required") {
		t.Fatalf("expected missing agent error, got: %s", stderr.String())
	}
}

func TestAgentProfileUpdateWithBenchmark(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	benchmarkPath := filepath.Join(tmpDir, "benchmark.json")
	benchmarkContent := `{"metrics":{"false_accept_count":1},"integration":{}}`
	if err := os.WriteFile(benchmarkPath, []byte(benchmarkContent), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "update", "--agent", "test-agent", "--from-benchmark", benchmarkPath, "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "observed_failure_modes: 1") {
		t.Fatalf("expected 1 failure mode, got: %s", out)
	}
}

func TestAgentProfileReportTextOutput(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	profilePath := filepath.Join(tmpDir, "profile.json")
	profile := &agentprofile.AgentProfile{
		SchemaVersion:        "1",
		AgentID:              "test-agent",
		MeasuredOn:           "2024-01-01T00:00:00Z",
		ObservedFailureModes: []string{},
		RequiredExtraChecks:  []string{"standard_verify_gate"},
		BenchmarkMetrics:     map[string]any{},
		AdvisoryOnly:         true,
		AdmissionAuthority:   false,
	}
	if err := agentprofile.WriteAgentProfile(profile, profilePath); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "report", "--profile", profilePath, "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# x-harness Agent Profile: test-agent") {
		t.Fatalf("expected profile header, got: %s", out)
	}
	if !strings.Contains(out, "- advisory_only: true") {
		t.Fatalf("expected advisory_only true, got: %s", out)
	}
}

func TestAgentProfileReportJSONOutput(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	profilePath := filepath.Join(tmpDir, "profile.json")
	profile := &agentprofile.AgentProfile{
		SchemaVersion:        "1",
		AgentID:              "test-agent",
		MeasuredOn:           "2024-01-01T00:00:00Z",
		ObservedFailureModes: []string{},
		RequiredExtraChecks:  []string{"standard_verify_gate"},
		BenchmarkMetrics:     map[string]any{},
		AdvisoryOnly:         true,
		AdmissionAuthority:   false,
	}
	if err := agentprofile.WriteAgentProfile(profile, profilePath); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "report", "--profile", profilePath, "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["agent_id"] != "test-agent" {
		t.Fatalf("expected agent_id test-agent, got: %v", result)
	}
}

func TestAgentProfileReportWithAgentID(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	profile := &agentprofile.AgentProfile{
		SchemaVersion:        "1",
		AgentID:              "test-agent",
		MeasuredOn:           "2024-01-01T00:00:00Z",
		ObservedFailureModes: []string{},
		RequiredExtraChecks:  []string{"standard_verify_gate"},
		BenchmarkMetrics:     map[string]any{},
		AdvisoryOnly:         true,
		AdmissionAuthority:   false,
	}
	defaultPath := agentprofile.DefaultAgentProfilePath(tmpDir, "test-agent")
	if err := agentprofile.WriteAgentProfile(profile, defaultPath); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "report", "--agent", "test-agent", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "test-agent") {
		t.Fatalf("expected test-agent in output, got: %s", out)
	}
}

func TestAgentProfileReportMissingProfileAndAgent(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "report", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires --profile or --agent") {
		t.Fatalf("expected missing profile/agent error, got: %s", stderr.String())
	}
}

func TestAgentProfileMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestAgentProfileUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown agent-profile subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestAgentProfileUnknownFlag(t *testing.T) {
	tmpDir := setupAgentProfileTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"agent-profile", "update", "--agent", "test-agent", "--root", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
