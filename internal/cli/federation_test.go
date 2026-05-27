package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupFederationTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	policyContent := `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: false
  require_redaction: false
  tenant_boundary: required
  retention_days: 30
  data_sent:
    - anonymized_failure_predicates
  data_never_sent:
    - raw_source_code
  import:
    default_dry_run: true
    affects_admission: false
`
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "federation.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	entry := map[string]any{
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
		"predicate":           "test-failure",
		"metadata": map[string]any{
			"admission_outcome": "failed",
			"acceptance_status": "withheld",
		},
	}
	b, _ := json.Marshal(entry)
	evidenceDir := filepath.Join(tmpDir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "index.jsonl"), []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestFederationExportTextOutput(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "federation patterns written:") {
		t.Fatalf("expected written message, got: %s", outStr)
	}
	if !strings.Contains(outStr, "records:") {
		t.Fatalf("expected record count, got: %s", outStr)
	}
}

func TestFederationExportJSONOutput(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
}

func TestFederationExportMissingOutFlag(t *testing.T) {
	root := setupFederationTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--out") {
		t.Fatalf("expected --out error, got: %s", stderr.String())
	}
}

func TestFederationExportMissingTenantFlag(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--tenant") {
		t.Fatalf("expected --tenant error, got: %s", stderr.String())
	}
}

func TestFederationExportMissingOptIn(t *testing.T) {
	tmpDir := t.TempDir()
	policyContent := `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: true
  require_redaction: true
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "federation.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{
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
		"predicate":           "fail",
	}
	b, _ := json.Marshal(entry)
	evidenceDir := filepath.Join(tmpDir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "index.jsonl"), []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", tmpDir, "--redacted"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--opt-in") {
		t.Fatalf("expected --opt-in error, got: %s", stderr.String())
	}
}

func TestFederationExportMissingRedacted(t *testing.T) {
	tmpDir := t.TempDir()
	policyContent := `
version: 1
federation:
  enabled: true
  default_enabled: false
  require_opt_in: true
  require_redaction: true
  tenant_boundary: required
  retention_days: 30
  data_sent: []
  data_never_sent: []
  import:
    default_dry_run: true
    affects_admission: false
`
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "federation.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{
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
		"predicate":           "fail",
	}
	b, _ := json.Marshal(entry)
	evidenceDir := filepath.Join(tmpDir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "index.jsonl"), []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", tmpDir, "--opt-in"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--redacted") {
		t.Fatalf("expected --redacted error, got: %s", stderr.String())
	}
}

func TestFederationValidateTextOutput(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"federation", "validate", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "federation patterns valid:") {
		t.Fatalf("expected valid message, got: %s", outStr)
	}
}

func TestFederationValidateJSONOutput(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"federation", "validate", out, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
}

func TestFederationValidateInvalidRecord(t *testing.T) {
	tmpDir := t.TempDir()
	badPattern := map[string]any{
		"schema_version": "1",
		"pattern_id":     "bad-id",
	}
	b, _ := json.Marshal(badPattern)
	path := filepath.Join(tmpDir, "bad.jsonl")
	if err := os.WriteFile(path, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "validate", path}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stdout.String(), "federation patterns invalid:") {
		t.Fatalf("expected invalid message, got: %s", stdout.String())
	}
}

func TestFederationImportDryRunTextOutput(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	targetDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"federation", "import-patterns", out, "--root", targetDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "federation import dry-run:") {
		t.Fatalf("expected dry-run message, got: %s", outStr)
	}
}

func TestFederationImportDryRunJSONOutput(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	targetDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"federation", "import-patterns", out, "--root", targetDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got: %v", result)
	}
}

func TestFederationImportWrite(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	targetDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"federation", "import-patterns", out, "--root", targetDir, "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "federation import wrote") {
		t.Fatalf("expected wrote message, got: %s", outStr)
	}
}

func TestFederationImportConflict(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, ".x-harness", "federation", "imported-patterns.jsonl")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetFile, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"federation", "import-patterns", out, "--root", targetDir, "--no-dry-run"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stdout.String(), "federation import failed:") {
		t.Fatalf("expected failed message, got: %s", stdout.String())
	}
}

func TestFederationImportMerge(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	targetDir := t.TempDir()
	targetFile := filepath.Join(targetDir, ".x-harness", "federation", "imported-patterns.jsonl")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		t.Fatal(err)
	}
	existing := map[string]any{
		"schema_version":       "1",
		"pattern_id":           strings.Repeat("a", 64),
		"tenant_hash":          strings.Repeat("0", 64),
		"source_hash":          strings.Repeat("0", 64),
		"pattern_class":        "observation",
		"signal":               map[string]any{"predicate_hash": nil, "predicate_present": false, "evidence_layer": "raw"},
		"evidence_kind":        "other",
		"component_hashes":     []string{},
		"benchmark_metrics":    nil,
		"created_at":           "2024-01-01T00:00:00Z",
		"retention_expires_at": "2024-02-01T00:00:00Z",
		"redaction": map[string]any{
			"mode":                     "anonymized-pattern",
			"redacted_required":        true,
			"raw_content_included":     false,
			"secret_scan_replacements": 0,
		},
		"admission_authority": false,
	}
	b, _ := json.Marshal(existing)
	if err := os.WriteFile(targetFile, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"federation", "import-patterns", out, "--root", targetDir, "--merge"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "federation import wrote") {
		t.Fatalf("expected wrote message, got: %s", stdout.String())
	}
}

func TestFederationSecretScan(t *testing.T) {
	tmpDir := t.TempDir()
	pattern := map[string]any{
		"schema_version":       "1",
		"pattern_id":           "ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"tenant_hash":          strings.Repeat("0", 64),
		"source_hash":          strings.Repeat("0", 64),
		"pattern_class":        "failure",
		"signal":               map[string]any{"predicate_hash": nil, "predicate_present": false, "evidence_layer": "raw"},
		"evidence_kind":        "other",
		"component_hashes":     []string{},
		"benchmark_metrics":    nil,
		"created_at":           "2024-01-01T00:00:00Z",
		"retention_expires_at": "2024-02-01T00:00:00Z",
		"redaction": map[string]any{
			"mode":                     "anonymized-pattern",
			"redacted_required":        true,
			"raw_content_included":     false,
			"secret_scan_replacements": 0,
		},
		"admission_authority": false,
	}
	b, _ := json.Marshal(pattern)
	path := filepath.Join(tmpDir, "secret.jsonl")
	if err := os.WriteFile(path, []byte(string(b)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "validate", path}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stdout.String(), "secret-like value detected") && !strings.Contains(stderr.String(), "secret-like value detected") {
		t.Fatalf("expected secret scan error. stdout: %s, stderr: %s", stdout.String(), stderr.String())
	}
}

func TestFederationMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestFederationUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown federation subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestFederationUnknownFlag(t *testing.T) {
	root := setupFederationTestDir(t)
	out := filepath.Join(t.TempDir(), "patterns.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "export-patterns", "--out", out, "--tenant", "tenant-a", "--root", root, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestFederationValidateMissingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "validate"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "patterns path is required") {
		t.Fatalf("expected path required error, got: %s", stderr.String())
	}
}

func TestFederationImportMissingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"federation", "import-patterns"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "patterns path is required") {
		t.Fatalf("expected path required error, got: %s", stderr.String())
	}
}
