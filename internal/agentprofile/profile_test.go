package agentprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) string {
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

func TestBuildAgentProfileWithoutBenchmark(t *testing.T) {
	profile, err := BuildAgentProfile("test-agent", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.AgentID != "test-agent" {
		t.Fatalf("expected agent_id test-agent, got %s", profile.AgentID)
	}
	if profile.SchemaVersion != "1" {
		t.Fatalf("expected schema_version 1, got %s", profile.SchemaVersion)
	}
	if len(profile.ObservedFailureModes) != 0 {
		t.Fatalf("expected no failure modes, got %v", profile.ObservedFailureModes)
	}
	if len(profile.RequiredExtraChecks) != 1 || profile.RequiredExtraChecks[0] != "standard_verify_gate" {
		t.Fatalf("expected standard_verify_gate, got %v", profile.RequiredExtraChecks)
	}
}

func TestBuildAgentProfileWithBenchmark(t *testing.T) {
	tmpDir := t.TempDir()
	benchmarkPath := filepath.Join(tmpDir, "benchmark.json")
	benchmarkContent := `{
  "metrics": {
    "false_accept_count": 2,
    "adversarial_false_accept_count": 1,
    "false_reject_count": 0
  },
  "integration": {
    "stale": true,
    "evidence": "missing"
  }
}`
	if err := os.WriteFile(benchmarkPath, []byte(benchmarkContent), 0644); err != nil {
		t.Fatal(err)
	}

	profile, err := BuildAgentProfile("test-agent", benchmarkPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedModes := []string{"adversarial_false_accept", "false_accept_regression", "stale_context_reference", "evidence_scope_mismatch"}
	if len(profile.ObservedFailureModes) != len(expectedModes) {
		t.Fatalf("expected %d failure modes, got %v", len(expectedModes), profile.ObservedFailureModes)
	}
	for _, mode := range expectedModes {
		found := false
		for _, m := range profile.ObservedFailureModes {
			if m == mode {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected failure mode %s, got %v", mode, profile.ObservedFailureModes)
		}
	}
}

func TestReadWriteAgentProfile(t *testing.T) {
	tmpDir := setupTestDir(t)
	profile := &AgentProfile{
		SchemaVersion:        "1",
		AgentID:              "agent-1",
		MeasuredOn:           "2024-01-01T00:00:00Z",
		ObservedFailureModes: []string{"false_accept_regression"},
		RequiredExtraChecks:  []string{"adversarial_replay_required", "standard_verify_gate"},
		BenchmarkMetrics:     map[string]any{"runtime_ms": 100},
		AdvisoryOnly:         true,
		AdmissionAuthority:   false,
	}
	path := filepath.Join(tmpDir, "profile.json")
	if err := WriteAgentProfile(profile, path); err != nil {
		t.Fatalf("unexpected error writing profile: %v", err)
	}
	read, err := ReadAgentProfile(path)
	if err != nil {
		t.Fatalf("unexpected error reading profile: %v", err)
	}
	if read.AgentID != profile.AgentID {
		t.Fatalf("expected agent_id %s, got %s", profile.AgentID, read.AgentID)
	}
}

func TestValidateAgentProfile(t *testing.T) {
	tmpDir := setupTestDir(t)
	profile := &AgentProfile{
		SchemaVersion:        "1",
		AgentID:              "agent-1",
		MeasuredOn:           "2024-01-01T00:00:00Z",
		ObservedFailureModes: []string{},
		RequiredExtraChecks:  []string{"standard_verify_gate"},
		BenchmarkMetrics:     map[string]any{},
		AdvisoryOnly:         true,
		AdmissionAuthority:   false,
	}
	if err := ValidateAgentProfile(profile, tmpDir); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidateAgentProfileInvalid(t *testing.T) {
	tmpDir := setupTestDir(t)
	profile := &AgentProfile{
		SchemaVersion: "2",
		AgentID:       "agent-1",
	}
	err := ValidateAgentProfile(profile, tmpDir)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("expected validation failed message, got %v", err)
	}
}

func TestDefaultAgentProfilePath(t *testing.T) {
	path := DefaultAgentProfilePath("/tmp/root", "agent-1")
	expected := filepath.Join("/tmp/root", ".x-harness", "agent-profiles", "agent-1.json")
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestSafeAgentID(t *testing.T) {
	if SafeAgentID("a@b#c") != "a_b_c" {
		t.Fatalf("expected a_b_c, got %s", SafeAgentID("a@b#c"))
	}
}
