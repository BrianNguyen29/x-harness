package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupApprovalRiskCLITestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	policiesDir := filepath.Join(tmpDir, "policies")
	schemasDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(policiesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}

	approvalRiskContent := `version: 1
approval_risk:
  enabled: false
  personal_scoring: false
  thresholds:
    moderate: 20
    elevated: 40
    critical: 70
  required_approvals:
    low: 0
    moderate: 1
    elevated: 1
    critical: 2
  signals:
    deep_tier: 25
    human_only_path: 35
    human_approved_path: 20
    security_sensitive_path: 20
    missing_governance_approval: 15
`
	if err := os.WriteFile(filepath.Join(policiesDir, "approval-risk.yaml"), []byte(approvalRiskContent), 0644); err != nil {
		t.Fatal(err)
	}

	authorityContent := `version: 1
authority_classes:
  agent_editable:
    description: "Files agents can freely modify"
    examples: []
  agent_proposable_human_approved:
    description: "Files agents may propose changes to, but require human approval"
    examples: []
  human_only:
    description: "Files only humans may directly modify"
    examples: []
protected_paths:
  - path: "schemas/**"
    authority: human_only
    rationale: "Schema definitions are authoritative contracts"
  - path: "policies/admission.yaml"
    authority: human_only
    rationale: "Admission policy defines success criteria"
report_only: true
governance_check:
  behavior: warn
  exit_on_warnings: false
  block_on_violations: false
`
	if err := os.WriteFile(filepath.Join(policiesDir, "authority.yaml"), []byte(authorityContent), 0644); err != nil {
		t.Fatal(err)
	}

	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "approval-risk",
  "type": "object",
  "required": [
    "schema_version",
    "task_id",
    "risk_class",
    "score",
    "signals",
    "required_approvals",
    "personal_scoring",
    "policy_enabled",
    "admission_authority"
  ],
  "properties": {
    "schema_version": { "const": "1" },
    "task_id": { "type": "string", "minLength": 1 },
    "tier": { "enum": ["light", "standard", "deep"] },
    "risk_class": { "enum": ["low", "moderate", "elevated", "critical"] },
    "score": { "type": "integer", "minimum": 0 },
    "signals": {
      "type": "array",
      "items": { "type": "string" }
    },
    "required_approvals": { "type": "integer", "minimum": 0 },
    "personal_scoring": { "const": false },
    "policy_enabled": { "type": "boolean" },
    "admission_authority": { "const": false }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(schemasDir, "approval-risk.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func writeTestCard(t *testing.T, tmpDir, tier string, files []string) string {
	t.Helper()
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	filesYAML := ""
	for _, f := range files {
		filesYAML += "    - \"" + f + "\"\n"
	}
	cardContent := `schema_version: "1"
task_id: task_001
tier: ` + tier + `
owner: test
accountable: test
evidence:
  files_changed:
` + filesYAML + `claim:
  fix_status: fixed
  summary: "Fix"
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: done
  owner: test
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}
	return cardPath
}

func TestApprovalRiskMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestApprovalRiskUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown approval-risk subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestApprovalRiskEvaluateMissingCard(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "evaluate"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--card is required") {
		t.Fatalf("expected missing card error, got: %s", stderr.String())
	}
}

func TestApprovalRiskCheckMissingCard(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "check"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--card is required") {
		t.Fatalf("expected missing card error, got: %s", stderr.String())
	}
}

func TestApprovalRiskEvaluateTextOutput(t *testing.T) {
	tmpDir := setupApprovalRiskCLITestDir(t)
	cardPath := writeTestCard(t, tmpDir, "light", []string{"src/main.go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "evaluate", "--card", cardPath, "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# x-harness Approval Risk") {
		t.Fatalf("expected header, got: %s", out)
	}
	if !strings.Contains(out, "task_id: task_001") {
		t.Fatalf("expected task_id, got: %s", out)
	}
	if !strings.Contains(out, "risk_class: low") {
		t.Fatalf("expected risk_class low, got: %s", out)
	}
	if !strings.Contains(out, "score: 0") {
		t.Fatalf("expected score 0, got: %s", out)
	}
	if !strings.Contains(out, "required_approvals: 0") {
		t.Fatalf("expected required_approvals 0, got: %s", out)
	}
	if !strings.Contains(out, "admission_authority: false") {
		t.Fatalf("expected admission_authority false, got: %s", out)
	}
}

func TestApprovalRiskEvaluateJSONOutput(t *testing.T) {
	tmpDir := setupApprovalRiskCLITestDir(t)
	cardPath := writeTestCard(t, tmpDir, "light", []string{"src/main.go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "evaluate", "--card", cardPath, "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["task_id"] != "task_001" {
		t.Fatalf("expected task_id=task_001, got: %v", result)
	}
	if result["risk_class"] != "low" {
		t.Fatalf("expected risk_class=low, got: %v", result)
	}
	if result["score"] != float64(0) {
		t.Fatalf("expected score=0, got: %v", result)
	}
}

func TestApprovalRiskCheckTextOutput(t *testing.T) {
	tmpDir := setupApprovalRiskCLITestDir(t)
	cardPath := writeTestCard(t, tmpDir, "light", []string{"src/main.go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "check", "--card", cardPath, "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "approval risk: low") {
		t.Fatalf("expected 'approval risk: low', got: %s", out)
	}
}

func TestApprovalRiskCheckJSONOutput(t *testing.T) {
	tmpDir := setupApprovalRiskCLITestDir(t)
	cardPath := writeTestCard(t, tmpDir, "deep", []string{"src/main.go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "check", "--card", cardPath, "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["task_id"] != "task_001" {
		t.Fatalf("expected task_id=task_001, got: %v", result)
	}
	if result["risk_class"] != "moderate" {
		t.Fatalf("expected risk_class=moderate, got: %v", result)
	}
	if result["score"] != float64(25) {
		t.Fatalf("expected score=25, got: %v", result)
	}
}

func TestApprovalRiskUnknownFlag(t *testing.T) {
	tmpDir := setupApprovalRiskCLITestDir(t)
	cardPath := writeTestCard(t, tmpDir, "light", []string{"src/main.go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approval-risk", "evaluate", "--card", cardPath, "--root", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestApprovalRiskEvaluateRelativeCardPath(t *testing.T) {
	tmpDir := setupApprovalRiskCLITestDir(t)
	writeTestCard(t, tmpDir, "light", []string{"src/main.go"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	// Run from tmpDir with relative card path
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	code := Run([]string{"approval-risk", "evaluate", "--card", "completion-card.yaml", "--root", "."}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "risk_class: low") {
		t.Fatalf("expected risk_class low, got: %s", stdout.String())
	}
}

func TestApprovalRiskAdvisoryOnlyReturnsOKOnError(t *testing.T) {
	// Missing policy files should still return ExitOK because advisory-only
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `schema_version: "1"
task_id: task_001
tier: light
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: "Fix"
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: done
  owner: test
`
	os.WriteFile(cardPath, []byte(cardContent), 0644)

	code := Run([]string{"approval-risk", "evaluate", "--card", cardPath, "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d (advisory-only), got %d. stderr: %s", ExitOK, code, stderr.String())
	}
}
