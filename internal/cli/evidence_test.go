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

func TestEvidenceIndexMissingFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "index"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "evidence index requires --episode or --card") {
		t.Fatalf("expected required flag error, got: %s", stderr.String())
	}
}

func TestEvidenceIndexEpisode(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "index", "--episode", tmpDir, "--task-id", "task-001"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Evidence index written.") {
		t.Fatalf("expected written message, got: %s", out)
	}
	if !strings.Contains(out, "task_id: task-001") {
		t.Fatalf("expected task_id, got: %s", out)
	}
	if !strings.Contains(out, "entries: 1") {
		t.Fatalf("expected entries count, got: %s", out)
	}
}

func TestEvidenceIndexCard(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `task_id: card-task
evidence:
  command_evidence: []
  verification_artifacts: []
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "index", "--card", cardPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "task_id: card-task") {
		t.Fatalf("expected card task_id, got: %s", out)
	}
}

func TestEvidenceIndexJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "index", "--episode", tmpDir, "--task-id", "task-001", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["task_id"] != "task-001" {
		t.Fatalf("expected task_id=task-001, got: %v", result)
	}
}

func TestEvidenceIndexWithRedact(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "secrets.txt"), []byte("password=secret123"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "index", "--episode", tmpDir, "--task-id", "task-001", "--redact"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "redacted_dir:") {
		t.Fatalf("expected redacted_dir in output, got: %s", out)
	}
}

func TestEvidenceIndexUnknownFlag(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "index", "--episode", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestEvidenceClassifyCommandText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "classify", "--command", "npm publish"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "risk: high") {
		t.Fatalf("expected high risk, got: %s", out)
	}
	if !strings.Contains(out, "package_publish") {
		t.Fatalf("expected package_publish intent, got: %s", out)
	}
}

func TestEvidenceClassifyCommandJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "classify", "--command", "custom-tool --flag", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["risk"] != "high" {
		t.Fatalf("expected high risk, got: %v", result)
	}
	if result["unknown"] != true {
		t.Fatalf("expected unknown=true, got: %v", result)
	}
}

func TestEvidenceClassifyCardJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `task_id: classify-task
evidence:
  command_evidence:
    - command: "go test ./..."
      exit_code: 0
  verification_artifacts:
    - command: "npm publish"
      exit_code: 0
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "classify", "--card", cardPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["card"] != cardPath {
		t.Fatalf("expected card path, got: %v", result)
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 results, got: %v", result)
	}
}

func TestEvidenceClassifyMissingFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "classify"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires --command or --card") {
		t.Fatalf("expected required flag error, got: %s", stderr.String())
	}
}

func TestEvidenceClassifyBothFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "classify", "--command", "ls", "--card", "card.yaml"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "provide only one of --command or --card") {
		t.Fatalf("expected mutual exclusion error, got: %s", stderr.String())
	}
}

func TestEvidenceClassifyUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "classify", "--command", "ls", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
