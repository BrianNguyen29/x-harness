package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeValidEvidenceEntry() map[string]any {
	return map[string]any{
		"schema_version":      "1",
		"task_id":             "task-001",
		"evidence_id":         "ev-001",
		"layer":               "raw",
		"kind":                "other",
		"path":                "test.txt",
		"sha256":              "0000000000000000000000000000000000000000000000000000000000000000",
		"size_bytes":          0,
		"redacted":            false,
		"created_at":          "2024-01-01T00:00:00Z",
		"admission_authority": false,
	}
}

func TestEvidenceValidateTextOutput(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "index.jsonl")
	entry := makeValidEvidenceEntry()
	b, _ := json.Marshal(entry)
	if err := os.WriteFile(indexPath, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "validate", "--index", indexPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Evidence index valid (1 entries).") {
		t.Fatalf("expected valid message, got: %s", out)
	}
}

func TestEvidenceValidateJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "index.jsonl")
	entry := makeValidEvidenceEntry()
	b, _ := json.Marshal(entry)
	if err := os.WriteFile(indexPath, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "validate", "--index", indexPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %s", out)
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["entry_count"] != float64(1) {
		t.Fatalf("expected entry_count=1, got: %v", result)
	}
}

func TestEvidenceValidateMissingIndex(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "validate", "--index", "/nonexistent/path.jsonl"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("expected not found error, got: %s", stderr.String())
	}
}

func TestEvidenceValidateUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "validate", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestEvidenceUnsupportedSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "unsupported"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown evidence subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}
