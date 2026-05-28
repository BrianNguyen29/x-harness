package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeArtifactHash(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-file")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hash, size, err := ComputeArtifactHash(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), size)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	// Idempotency: same file should produce same hash
	hash2, size2, err := ComputeArtifactHash(path)
	if err != nil {
		t.Fatalf("expected no error on second call, got: %v", err)
	}
	if hash != hash2 {
		t.Fatalf("expected same hash, got %s vs %s", hash, hash2)
	}
	if size != size2 {
		t.Fatalf("expected same size, got %d vs %d", size, size2)
	}
}

func TestComputeArtifactHashMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	_, _, err := ComputeArtifactHash(path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestVerifyEvidenceValid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "artifact")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	hash, size, err := ComputeArtifactHash(path)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}

	ev := &Evidence{
		SchemaVersion: "x-harness.release-evidence.v1",
		Artifacts: []Artifact{
			{Path: path, SHA256: hash, Size: size},
		},
		Conformance: ConformanceStatus{Minimal: "passed"},
	}

	if err := VerifyEvidence(ev); err != nil {
		t.Fatalf("expected verification to pass, got: %v", err)
	}
}

func TestVerifyEvidenceMissingArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	ev := &Evidence{
		Artifacts: []Artifact{
			{Path: filepath.Join(tmpDir, "missing"), SHA256: "abcd", Size: 0},
		},
		Conformance: ConformanceStatus{Minimal: "passed"},
	}

	if err := VerifyEvidence(ev); err == nil {
		t.Fatal("expected verification to fail for missing artifact")
	}
}

func TestVerifyEvidenceChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "artifact")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	size, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}

	ev := &Evidence{
		Artifacts: []Artifact{
			{Path: path, SHA256: "0000000000000000000000000000000000000000000000000000000000000000", Size: size.Size()},
		},
		Conformance: ConformanceStatus{Minimal: "passed"},
	}

	if err := VerifyEvidence(ev); err == nil {
		t.Fatal("expected verification to fail for checksum mismatch")
	}
}

func TestVerifyEvidenceMissingConformance(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "artifact")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write artifact: %v", err)
	}

	hash, size, err := ComputeArtifactHash(path)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}

	ev := &Evidence{
		Artifacts: []Artifact{
			{Path: path, SHA256: hash, Size: size},
		},
		Conformance: ConformanceStatus{Minimal: "failed"},
	}

	if err := VerifyEvidence(ev); err == nil {
		t.Fatal("expected verification to fail for missing conformance")
	}
}

func TestVerifyEvidenceNoArtifacts(t *testing.T) {
	ev := &Evidence{
		Artifacts:   []Artifact{},
		Conformance: ConformanceStatus{Minimal: "passed"},
	}

	if err := VerifyEvidence(ev); err == nil {
		t.Fatal("expected verification to fail for no artifacts")
	}
}
