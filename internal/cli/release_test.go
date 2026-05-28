package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/release"
)

func TestReleaseEvidenceGeneratesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "evidence.json")

	// Create a dummy artifact
	artifactPath := filepath.Join(tmpDir, "dummy-artifact")
	if err := os.WriteFile(artifactPath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "evidence", "--out", outPath, "--artifact", artifactPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %+v", result)
	}

	// Verify file was written
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected evidence file to exist: %v", err)
	}

	var ev release.Evidence
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatalf("expected valid evidence JSON: %v", err)
	}
	if ev.SchemaVersion != "x-harness.release-evidence.v1" {
		t.Fatalf("expected schema version x-harness.release-evidence.v1, got %s", ev.SchemaVersion)
	}
	if len(ev.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(ev.Artifacts))
	}
	if ev.Artifacts[0].Path != artifactPath {
		t.Fatalf("expected artifact path %s, got %s", artifactPath, ev.Artifacts[0].Path)
	}
	if ev.Conformance.Minimal != "passed" {
		t.Fatalf("expected conformance passed, got %s", ev.Conformance.Minimal)
	}
	if ev.Doctor == nil || ev.Doctor.Status == "" {
		t.Fatalf("expected doctor status")
	}
	if ev.ContextSync == nil || ev.ContextSync.Status == "" {
		t.Fatalf("expected context sync status")
	}
}

func TestReleaseEvidenceMissingOut(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "evidence"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}

func TestReleaseVerifyEvidencePasses(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a dummy artifact
	artifactPath := filepath.Join(tmpDir, "dummy-artifact")
	if err := os.WriteFile(artifactPath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	hash, size, err := release.ComputeArtifactHash(artifactPath)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	ev := release.Evidence{
		SchemaVersion: "x-harness.release-evidence.v1",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Version:       "0.1.0",
		Artifacts: []release.Artifact{
			{Path: artifactPath, SHA256: hash, Size: size},
		},
		Conformance: release.ConformanceStatus{Minimal: "passed"},
	}

	data, _ := json.MarshalIndent(ev, "", "  ")
	evidencePath := filepath.Join(tmpDir, "evidence.json")
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "verify-evidence", evidencePath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %+v", result)
	}
}

func TestReleaseVerifyEvidenceFailsMissingArtifact(t *testing.T) {
	tmpDir := t.TempDir()

	ev := release.Evidence{
		SchemaVersion: "x-harness.release-evidence.v1",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Version:       "0.1.0",
		Artifacts: []release.Artifact{
			{Path: filepath.Join(tmpDir, "missing-artifact"), SHA256: "abcd", Size: 0},
		},
		Conformance: release.ConformanceStatus{Minimal: "passed"},
	}

	data, _ := json.MarshalIndent(ev, "", "  ")
	evidencePath := filepath.Join(tmpDir, "evidence.json")
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "verify-evidence", evidencePath, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false, got: %+v", result)
	}
	if result["error"] == nil {
		t.Fatalf("expected error field")
	}
}

func TestReleaseVerifyEvidenceFailsChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a dummy artifact
	artifactPath := filepath.Join(tmpDir, "dummy-artifact")
	if err := os.WriteFile(artifactPath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	size, err := os.Stat(artifactPath)
	if err != nil {
		t.Fatalf("failed to stat artifact: %v", err)
	}

	ev := release.Evidence{
		SchemaVersion: "x-harness.release-evidence.v1",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Version:       "0.1.0",
		Artifacts: []release.Artifact{
			{Path: artifactPath, SHA256: "0000000000000000000000000000000000000000000000000000000000000000", Size: size.Size()},
		},
		Conformance: release.ConformanceStatus{Minimal: "passed"},
	}

	data, _ := json.MarshalIndent(ev, "", "  ")
	evidencePath := filepath.Join(tmpDir, "evidence.json")
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "verify-evidence", evidencePath, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false, got: %+v", result)
	}
	if result["error"] == nil {
		t.Fatalf("expected error field")
	}
}

func TestReleaseVerifyEvidenceFailsMissingConformance(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a dummy artifact
	artifactPath := filepath.Join(tmpDir, "dummy-artifact")
	if err := os.WriteFile(artifactPath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	hash, size, err := release.ComputeArtifactHash(artifactPath)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	ev := release.Evidence{
		SchemaVersion: "x-harness.release-evidence.v1",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Version:       "0.1.0",
		Artifacts: []release.Artifact{
			{Path: artifactPath, SHA256: hash, Size: size},
		},
		Conformance: release.ConformanceStatus{Minimal: "failed"},
	}

	data, _ := json.MarshalIndent(ev, "", "  ")
	evidencePath := filepath.Join(tmpDir, "evidence.json")
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "verify-evidence", evidencePath, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false, got: %+v", result)
	}
	if result["error"] == nil {
		t.Fatalf("expected error field")
	}
}

func TestReleaseVerifyEvidenceFailsNoArtifacts(t *testing.T) {
	tmpDir := t.TempDir()

	ev := release.Evidence{
		SchemaVersion: "x-harness.release-evidence.v1",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Version:       "0.1.0",
		Artifacts:     []release.Artifact{},
		Conformance:   release.ConformanceStatus{Minimal: "passed"},
	}

	data, _ := json.MarshalIndent(ev, "", "  ")
	evidencePath := filepath.Join(tmpDir, "evidence.json")
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "verify-evidence", evidencePath, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false, got: %+v", result)
	}
}

func TestReleaseUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown release subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}
