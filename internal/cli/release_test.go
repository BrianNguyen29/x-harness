package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/doctor"
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

func TestReleaseReportMarkdown(t *testing.T) {
	tmpDir := t.TempDir()

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
		Commit:        "abc123",
		GoVersion:     "go1.22.0",
		Artifacts: []release.Artifact{
			{Path: artifactPath, SHA256: hash, Size: size},
		},
		Conformance: release.ConformanceStatus{Minimal: "passed"},
		Doctor:      &release.DoctorStatus{Status: "healthy"},
		ContextSync: &release.ContextSyncStatus{Status: "no_drift"},
	}

	data, _ := json.MarshalIndent(ev, "", "  ")
	evidencePath := filepath.Join(tmpDir, "evidence.json")
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "report", "--evidence", evidencePath, "--format", "markdown"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "# Release Report") {
		t.Fatalf("expected markdown header, got: %s", out)
	}
	if !strings.Contains(out, artifactPath) {
		t.Fatalf("expected artifact path, got: %s", out)
	}
	if !strings.Contains(out, hash) {
		t.Fatalf("expected artifact hash, got: %s", out)
	}
	if !strings.Contains(out, "healthy") {
		t.Fatalf("expected doctor status, got: %s", out)
	}
	if !strings.Contains(out, "no_drift") {
		t.Fatalf("expected context sync status, got: %s", out)
	}
	if !strings.Contains(out, "not declared in minimal evidence") {
		t.Fatalf("expected non-overclaiming note, got: %s", out)
	}
}

func TestReleaseReportJSON(t *testing.T) {
	tmpDir := t.TempDir()

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
		Commit:        "abc123",
		GoVersion:     "go1.22.0",
		Artifacts: []release.Artifact{
			{Path: artifactPath, SHA256: hash, Size: size},
		},
		Conformance: release.ConformanceStatus{Minimal: "passed"},
		Doctor:      &release.DoctorStatus{Status: "healthy"},
		ContextSync: &release.ContextSyncStatus{Status: "no_drift"},
	}

	data, _ := json.MarshalIndent(ev, "", "  ")
	evidencePath := filepath.Join(tmpDir, "evidence.json")
	if err := os.WriteFile(evidencePath, data, 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"release", "report", "--evidence", evidencePath, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var report map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if report["schema_version"] != "x-harness.release-evidence.v1" {
		t.Fatalf("expected schema_version, got: %+v", report)
	}
	if report["doctor"] != "healthy" {
		t.Fatalf("expected doctor healthy, got: %+v", report)
	}
	if report["context_sync"] != "no_drift" {
		t.Fatalf("expected context_sync no_drift, got: %+v", report)
	}
	if report["platform_matrix"] != "not declared in minimal evidence" {
		t.Fatalf("expected platform_matrix note, got: %+v", report)
	}
}

func TestReleaseReportMissingEvidence(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "missing.json")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "report", "--evidence", missingPath, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "cannot read evidence file") {
		t.Fatalf("expected missing file error, got: %s", stderr.String())
	}
}

func TestReleaseReportInvalidEvidence(t *testing.T) {
	tmpDir := t.TempDir()
	evidencePath := filepath.Join(tmpDir, "bad-evidence.json")
	if err := os.WriteFile(evidencePath, []byte(`{"artifacts":[]}`), 0644); err != nil {
		t.Fatalf("failed to write evidence: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "report", "--evidence", evidencePath, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "invalid or incomplete evidence") {
		t.Fatalf("expected invalid evidence error, got: %s", stderr.String())
	}
}

func TestReleaseReportMissingEvidenceFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "report"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
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

func TestReleaseVerifyDocsHealthyJSON(t *testing.T) {
	tmpDir := writeHealthyDocsFixture(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "verify-docs", "--root", tmpDir, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var report doctor.DocsDriftReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !report.Healthy {
		t.Fatalf("expected healthy report, got %+v", report)
	}
	if report.Root != tmpDir {
		t.Fatalf("expected root=%s, got %s", tmpDir, report.Root)
	}
}

func TestReleaseVerifyDocsUnhealthyJSON(t *testing.T) {
	tmpDir := writeUnhealthyDocsFixture(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "verify-docs", "--root", tmpDir, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}

	var report doctor.DocsDriftReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if report.Healthy {
		t.Fatalf("expected unhealthy report, got %+v", report)
	}
	found := false
	for _, tag := range report.DriftTags {
		if tag == "workflow_missing_verify" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected drift_tag=workflow_missing_verify, got %+v", report.DriftTags)
	}
}

func TestReleaseVerifyDocsTextFormat(t *testing.T) {
	tmpDir := writeHealthyDocsFixture(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "verify-docs", "--root", tmpDir, "--format", "text"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("expected text output, got JSON: %s", out)
	}
	if !strings.Contains(out, "healthy: true") {
		t.Fatalf("expected text output to contain 'healthy: true', got: %s", out)
	}
}

func TestReleaseVerifyDocsUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "verify-docs", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitUsage, code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
