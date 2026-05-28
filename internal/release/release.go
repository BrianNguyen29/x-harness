package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// Artifact describes a release artifact with its hash and size.
type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// ConformanceStatus holds the conformance check results.
type ConformanceStatus struct {
	Minimal string `json:"minimal"`
}

// DoctorStatus holds the doctor health status.
type DoctorStatus struct {
	Status string `json:"status"`
}

// ContextSyncStatus holds the context sync status.
type ContextSyncStatus struct {
	Status string `json:"status"`
}

// Evidence is the release evidence bundle.
type Evidence struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	Version       string            `json:"version"`
	Commit        string            `json:"commit,omitempty"`
	GoVersion     string            `json:"go_version,omitempty"`
	Artifacts     []Artifact        `json:"artifacts"`
	Conformance   ConformanceStatus `json:"conformance"`
	Doctor        *DoctorStatus     `json:"doctor,omitempty"`
	ContextSync   *ContextSyncStatus `json:"context_sync,omitempty"`
}

// ComputeArtifactHash computes the SHA-256 hash and size of the file at path.
func ComputeArtifactHash(path string) (string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), int64(len(data)), nil
}

// VerifyEvidence checks the evidence bundle for required fields, artifact
// presence, checksums, and conformance status.
func VerifyEvidence(ev *Evidence) error {
	if len(ev.Artifacts) == 0 {
		return fmt.Errorf("no artifacts in evidence")
	}
	if ev.Conformance.Minimal != "passed" {
		return fmt.Errorf("conformance minimal status is %q, expected passed", ev.Conformance.Minimal)
	}
	for _, art := range ev.Artifacts {
		hash, _, err := ComputeArtifactHash(art.Path)
		if err != nil {
			return fmt.Errorf("artifact %s: missing: %w", art.Path, err)
		}
		if hash != art.SHA256 {
			return fmt.Errorf("artifact %s: checksum mismatch: expected %s, got %s", art.Path, art.SHA256, hash)
		}
	}
	return nil
}
