package approvalrisk

import (
	"os"
	"path/filepath"
	"testing"
)

func setupApprovalRiskTestDir(t *testing.T) string {
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
  - path: "policies/recovery.yaml"
    authority: agent_proposable_human_approved
    rationale: "Recovery routing may be updated by agents with human approval"
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

func TestEvaluateApprovalRiskLow(t *testing.T) {
	tmpDir := setupApprovalRiskTestDir(t)
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `schema_version: "1"
task_id: task_001
tier: light
owner: test
accountable: test
evidence:
  files_changed:
    - "src/main.go"
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
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := EvaluateApprovalRisk(cardPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TaskID != "task_001" {
		t.Fatalf("expected task_id task_001, got %s", report.TaskID)
	}
	if report.RiskClass != "low" {
		t.Fatalf("expected risk_class low, got %s", report.RiskClass)
	}
	if report.Score != 0 {
		t.Fatalf("expected score 0, got %d", report.Score)
	}
	if report.RequiredApprovals != 0 {
		t.Fatalf("expected required_approvals 0, got %d", report.RequiredApprovals)
	}
	if len(report.Signals) != 0 {
		t.Fatalf("expected no signals, got %v", report.Signals)
	}
}

func TestEvaluateApprovalRiskDeepTier(t *testing.T) {
	tmpDir := setupApprovalRiskTestDir(t)
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `schema_version: "1"
task_id: task_002
tier: deep
owner: test
accountable: test
evidence:
  files_changed:
    - "src/main.go"
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
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := EvaluateApprovalRisk(cardPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.RiskClass != "moderate" {
		t.Fatalf("expected risk_class moderate, got %s", report.RiskClass)
	}
	if report.Score != 25 {
		t.Fatalf("expected score 25, got %d", report.Score)
	}
	if report.RequiredApprovals != 1 {
		t.Fatalf("expected required_approvals 1, got %d", report.RequiredApprovals)
	}
	if len(report.Signals) != 1 || report.Signals[0] != "deep_tier" {
		t.Fatalf("expected [deep_tier] signal, got %v", report.Signals)
	}
}

func TestEvaluateApprovalRiskHumanOnlyPath(t *testing.T) {
	tmpDir := setupApprovalRiskTestDir(t)
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `schema_version: "1"
task_id: task_003
tier: standard
owner: test
accountable: test
evidence:
  files_changed:
    - "schemas/completion-card.schema.json"
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
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := EvaluateApprovalRisk(cardPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.RiskClass != "elevated" {
		t.Fatalf("expected risk_class elevated, got %s", report.RiskClass)
	}
	if report.Score != 50 {
		t.Fatalf("expected score 50, got %d", report.Score)
	}
	if report.RequiredApprovals != 1 {
		t.Fatalf("expected required_approvals 1, got %d", report.RequiredApprovals)
	}
	expectedSignals := []string{"human_only_path", "missing_governance_approval"}
	if len(report.Signals) != len(expectedSignals) {
		t.Fatalf("expected signals %v, got %v", expectedSignals, report.Signals)
	}
	for i, s := range expectedSignals {
		if report.Signals[i] != s {
			t.Fatalf("expected signal %q at index %d, got %q", s, i, report.Signals[i])
		}
	}
}

func TestEvaluateApprovalRiskHumanOnlyWithGovernanceApproval(t *testing.T) {
	tmpDir := setupApprovalRiskTestDir(t)
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `schema_version: "1"
task_id: task_004
tier: standard
owner: test
accountable: test
evidence:
  files_changed:
    - "schemas/completion-card.schema.json"
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
governance:
  approval_status: approved
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := EvaluateApprovalRisk(cardPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.RiskClass != "moderate" {
		t.Fatalf("expected risk_class moderate, got %s", report.RiskClass)
	}
	if report.Score != 35 {
		t.Fatalf("expected score 35, got %d", report.Score)
	}
	if report.RequiredApprovals != 1 {
		t.Fatalf("expected required_approvals 1, got %d", report.RequiredApprovals)
	}
	expectedSignals := []string{"human_only_path"}
	if len(report.Signals) != len(expectedSignals) {
		t.Fatalf("expected signals %v, got %v", expectedSignals, report.Signals)
	}
}

func TestEvaluateApprovalRiskSecuritySensitivePath(t *testing.T) {
	tmpDir := setupApprovalRiskTestDir(t)
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `schema_version: "1"
task_id: task_005
tier: standard
owner: test
accountable: test
evidence:
  files_changed:
    - "src/auth.go"
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
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := EvaluateApprovalRisk(cardPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.RiskClass != "moderate" {
		t.Fatalf("expected risk_class moderate, got %s", report.RiskClass)
	}
	if report.Score != 20 {
		t.Fatalf("expected score 20, got %d", report.Score)
	}
	if report.RequiredApprovals != 1 {
		t.Fatalf("expected required_approvals 1, got %d", report.RequiredApprovals)
	}
	if len(report.Signals) != 1 || report.Signals[0] != "security_sensitive_path" {
		t.Fatalf("expected [security_sensitive_path] signal, got %v", report.Signals)
	}
}

func TestEvaluateApprovalRiskHumanApprovedPath(t *testing.T) {
	tmpDir := setupApprovalRiskTestDir(t)
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `schema_version: "1"
task_id: task_006
tier: standard
owner: test
accountable: test
evidence:
  files_changed:
    - "policies/recovery.yaml"
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
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := EvaluateApprovalRisk(cardPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.RiskClass != "moderate" {
		t.Fatalf("expected risk_class moderate, got %s", report.RiskClass)
	}
	if report.Score != 20 {
		t.Fatalf("expected score 20, got %d", report.Score)
	}
	if report.RequiredApprovals != 1 {
		t.Fatalf("expected required_approvals 1, got %d", report.RequiredApprovals)
	}
	if len(report.Signals) != 1 || report.Signals[0] != "human_approved_path" {
		t.Fatalf("expected [human_approved_path] signal, got %v", report.Signals)
	}
}

func TestEvaluateApprovalRiskMissingCard(t *testing.T) {
	tmpDir := setupApprovalRiskTestDir(t)
	_, err := EvaluateApprovalRisk(filepath.Join(tmpDir, "missing.yaml"), tmpDir)
	if err == nil {
		t.Fatal("expected error for missing card")
	}
}
