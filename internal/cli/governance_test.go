package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupGovernanceTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	policyContent := `version: 1
authority_classes:
  agent_editable:
    description: "Files agents can freely modify"
    examples:
      - "packages/cli/src/**/*.ts"
  agent_proposable_human_approved:
    description: "Files agents may propose changes to, but require human approval"
    examples:
      - "policies/recovery.yaml"
  human_only:
    description: "Files only humans may directly modify"
    examples:
      - "schemas/**"
      - "policies/admission.yaml"
protected_paths:
  - path: "schemas/**"
    authority: human_only
    rationale: "Schema definitions are authoritative contracts"
  - path: "policies/admission.yaml"
    authority: human_only
    rationale: "Admission policy defines success criteria"
  - path: "policies/recovery.yaml"
    authority: agent_proposable_human_approved
    rationale: "Recovery routing may be updated by agents with human approval"
  - path: "package.json"
    authority: human_only
    rationale: "Package manifest controls build/test commands"
report_only: true
governance_check:
  behavior: warn
  exit_on_warnings: false
  block_on_violations: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "authority.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func writeCard(t *testing.T, dir string, files []string) string {
	t.Helper()
	cardPath := filepath.Join(dir, "completion-card.yaml")
	content := "schema_version: \"1.0\"\nevidence:\n  files_changed:\n"
	for _, f := range files {
		content += "    - " + f + "\n"
	}
	if err := os.WriteFile(cardPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return cardPath
}

func TestGovernanceExplain(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "explain", "--path", "schemas/foo.json", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "human_only") {
		t.Fatalf("expected human_only authority, got: %s", out)
	}
	if !strings.Contains(out, "Schema definitions") {
		t.Fatalf("expected rationale, got: %s", out)
	}
}

func TestGovernanceExplainJSON(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "explain", "--path", "schemas/foo.json", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["authority"] != "human_only" {
		t.Fatalf("expected human_only, got: %v", result)
	}
}

func TestGovernanceExplainMissingPath(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "explain", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--path is required") {
		t.Fatalf("expected missing path error, got: %s", stderr.String())
	}
}

func TestGovernanceListProtected(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "list-protected", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "schemas/**") {
		t.Fatalf("expected schemas/** in output, got: %s", out)
	}
	if !strings.Contains(out, "human_only") {
		t.Fatalf("expected human_only in output, got: %s", out)
	}
}

func TestGovernanceListProtectedJSON(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "list-protected", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	protected, ok := result["protected_paths"].([]interface{})
	if !ok || len(protected) != 4 {
		t.Fatalf("expected 4 protected paths, got: %v", result)
	}
}

func TestGovernanceCheckPass(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	writeCard(t, tmpDir, []string{"src/main.go", "README.md"})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--card", "completion-card.yaml", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "No governance violations found") {
		t.Fatalf("expected no violations message, got: %s", out)
	}
}

func TestGovernanceCheckViolation(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	writeCard(t, tmpDir, []string{"schemas/foo.json", "package.json"})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--card", "completion-card.yaml", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d in report-only mode, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "warning(s)") {
		t.Fatalf("expected warnings message, got: %s", out)
	}
	if !strings.Contains(out, "schemas/foo.json") {
		t.Fatalf("expected schemas/foo.json in output, got: %s", out)
	}
}

func TestGovernanceCheckEnforcedViolation(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	writeCard(t, tmpDir, []string{"schemas/foo.json"})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--card", "completion-card.yaml", "--root", tmpDir, "--enforce"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d in enforced mode, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "violation(s)") {
		t.Fatalf("expected violations message, got: %s", out)
	}
}

func TestGovernanceCheckMissingCardFlag(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--card is required") {
		t.Fatalf("expected missing card error, got: %s", stderr.String())
	}
}

func TestGovernanceCheckCardNotFound(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--card", "missing.yaml", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Card not found") {
		t.Fatalf("expected card not found error, got: %s", stderr.String())
	}
}

func TestGovernanceCheckJSONOutput(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	writeCard(t, tmpDir, []string{"schemas/foo.json"})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--card", "completion-card.yaml", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false when warnings present, got: %v", result)
	}
	if warnings, ok := result["total_warnings"].(float64); !ok || warnings != 1 {
		t.Fatalf("expected 1 warning, got: %v", result)
	}
}

func TestGovernanceCheckEmptyFiles(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	writeCard(t, tmpDir, []string{})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--card", "completion-card.yaml", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "No files to check") {
		t.Fatalf("expected no files message, got: %s", out)
	}
}

func TestGovernanceMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestGovernanceUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown governance subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestGovernanceUnknownFlag(t *testing.T) {
	tmpDir := setupGovernanceTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "check", "--card", "card.yaml", "--root", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestGovernanceHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"governance", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}
