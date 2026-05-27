package authority

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestAuthorityPolicy(t *testing.T, dir string) {
	t.Helper()
	policiesDir := filepath.Join(dir, "policies")
	if err := os.MkdirAll(policiesDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `version: 1
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
	if err := os.WriteFile(filepath.Join(policiesDir, "authority.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestExplainPath(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		path            string
		wantAuthority   string
		wantRationale   string
	}{
		{"schemas/completion-card.schema.json", "human_only", "Schema definitions are authoritative contracts"},
		{"policies/admission.yaml", "human_only", "Admission policy defines success criteria"},
		{"policies/recovery.yaml", "agent_proposable_human_approved", "Recovery routing may be updated by agents with human approval"},
		{"package.json", "human_only", "Package manifest controls build/test commands"},
		{"packages/cli/src/commands/verify.ts", "agent_editable", "Default: no protected path match"},
	}

	for _, c := range cases {
		result, err := ExplainPath(policy, c.path, tmpDir)
		if err != nil {
			t.Fatalf("ExplainPath(%q) error: %v", c.path, err)
		}
		if result.Authority != c.wantAuthority {
			t.Errorf("ExplainPath(%q) authority = %q, want %q", c.path, result.Authority, c.wantAuthority)
		}
		if result.Rationale != c.wantRationale {
			t.Errorf("ExplainPath(%q) rationale = %q, want %q", c.path, result.Rationale, c.wantRationale)
		}
	}
}

func TestExplainPathAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	absPath := filepath.Join(tmpDir, "schemas", "foo.json")
	result, err := ExplainPath(policy, absPath, tmpDir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Path != "schemas/foo.json" {
		t.Fatalf("expected relative path, got %q", result.Path)
	}
	if result.Authority != "human_only" {
		t.Fatalf("expected human_only, got %q", result.Authority)
	}
}

func TestGetProtectedPaths(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	paths := GetProtectedPaths(policy)
	if len(paths) != 4 {
		t.Fatalf("expected 4 protected paths, got %d", len(paths))
	}
}

func TestCheckGovernanceNoProtected(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files := []string{"src/main.go", "README.md"}
	result, err := CheckGovernance(files, tmpDir, policy, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalViolations != 0 {
		t.Fatalf("expected 0 violations, got %d", result.TotalViolations)
	}
	if result.TotalWarnings != 0 {
		t.Fatalf("expected 0 warnings, got %d", result.TotalWarnings)
	}
	if !result.ReportOnly {
		t.Fatal("expected report-only mode")
	}
}

func TestCheckGovernanceWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files := []string{"schemas/foo.json", "package.json", "src/main.go"}
	result, err := CheckGovernance(files, tmpDir, policy, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalViolations != 0 {
		t.Fatalf("expected 0 violations in report-only, got %d", result.TotalViolations)
	}
	if result.TotalWarnings != 2 {
		t.Fatalf("expected 2 warnings, got %d", result.TotalWarnings)
	}
}

func TestCheckGovernanceEnforcedViolations(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files := []string{"schemas/foo.json", "package.json"}
	result, err := CheckGovernance(files, tmpDir, policy, &GovernanceCheckOptions{Enforce: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalViolations != 2 {
		t.Fatalf("expected 2 violations in enforced mode, got %d", result.TotalViolations)
	}
	if result.TotalWarnings != 0 {
		t.Fatalf("expected 0 warnings when all become violations, got %d", result.TotalWarnings)
	}
	if result.ReportOnly {
		t.Fatal("expected enforced mode")
	}
}

func TestCheckGovernanceEnforcedWithApproval(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestAuthorityPolicy(t, tmpDir)

	// Create approval artifact structure
	approvalsDir := filepath.Join(tmpDir, ".x-harness", "approvals")
	if err := os.MkdirAll(approvalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	approvalContent := `{
  "decision": "approved",
  "approved_by": "human",
  "approved_at": "2024-01-01T00:00:00Z",
  "scope": {
    "paths": ["schemas/**"]
  }
}`
	approvalPath := filepath.Join(approvalsDir, "approval.json")
	if err := os.WriteFile(approvalPath, []byte(approvalContent), 0644); err != nil {
		t.Fatal(err)
	}

	hash := sha256File(approvalPath)

	registryContent := `{
  "approvals": [
    {
      "path": ".x-harness/approvals/approval.json",
      "sha256": "` + hash + `",
      "status": "approved",
      "approved_by": "human",
      "scope": {
        "paths": ["schemas/**"]
      }
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(approvalsDir, "registry.json"), []byte(registryContent), 0644); err != nil {
		t.Fatal(err)
	}

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	files := []string{"schemas/foo.json"}
	governance := map[string]any{
		"approval_status": "approved",
		"approval_artifact": map[string]any{
			"path":   ".x-harness/approvals/approval.json",
			"sha256": hash,
		},
	}

	result, err := CheckGovernance(files, tmpDir, policy, &GovernanceCheckOptions{
		Enforce:    true,
		Governance: governance,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalViolations != 0 {
		t.Fatalf("expected 0 violations with valid approval, got %d", result.TotalViolations)
	}
}

func TestLoadCardGovernanceData(t *testing.T) {
	tmpDir := t.TempDir()
	cardContent := `schema_version: "1.0"
evidence:
  files_changed:
    - schemas/foo.json
    - src/main.go
governance:
  approval_status: approved
`
	cardPath := filepath.Join(tmpDir, "card.yaml")
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	files, governance, err := LoadCardGovernanceData(cardPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if governance == nil {
		t.Fatal("expected governance data")
	}
	status, _ := governance["approval_status"].(string)
	if status != "approved" {
		t.Fatalf("expected approved status, got %q", status)
	}
}

func TestLoadCardGovernanceDataJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cardContent := `{
  "schema_version": "1.0",
  "evidence": {
    "files_changed": ["package.json"]
  }
}`
	cardPath := filepath.Join(tmpDir, "card.json")
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	files, governance, err := LoadCardGovernanceData(cardPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "package.json" {
		t.Fatalf("expected [package.json], got %v", files)
	}
	if governance != nil {
		t.Fatal("expected nil governance")
	}
}

func TestNormalizedHash(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"abc123", "abc123"},
		{"sha256:ABC", "abc"},
		{"  SHA256:DEF  ", "def"},
	}
	for _, c := range cases {
		got := normalizedHash(c.input)
		if got != c.want {
			t.Errorf("normalizedHash(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
