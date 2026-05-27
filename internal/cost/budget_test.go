package cost

import (
	"os"
	"path/filepath"
	"testing"
)

func setupCostTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	policyContent := `version: 1
cost_budget:
  enabled: true
  max_usd: 5.0
  max_input_tokens: 150000
  max_output_tokens: 45000
  over_budget_recovery: escalate_to_human
  affects_admission: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "cost-budget.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestLoadPolicy(t *testing.T) {
	tmpDir := setupCostTestDir(t)
	policy, err := LoadPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !policy.Enabled {
		t.Fatal("expected policy to be enabled")
	}
	if policy.MaxActualUSD != 5.0 {
		t.Fatalf("expected max_usd=5.0, got %f", policy.MaxActualUSD)
	}
	if policy.MaxInputTokens != 150000 {
		t.Fatalf("expected max_input_tokens=150000, got %d", policy.MaxInputTokens)
	}
	if policy.MaxOutputTokens != 45000 {
		t.Fatalf("expected max_output_tokens=45000, got %d", policy.MaxOutputTokens)
	}
}

func TestEvaluateCostBudgetWithinBudget(t *testing.T) {
	policy := &CostBudgetPolicy{
		MaxInputTokens:   100,
		MaxOutputTokens:  100,
		MaxActualUSD:     10.0,
		Enabled:          true,
		AffectsAdmission: false,
	}
	report := EvaluateCostBudget(policy, 5.0, 50, 50, false)
	if report.OverBudget {
		t.Fatal("expected within budget")
	}
	if report.Status != "within_budget" {
		t.Fatalf("expected status within_budget, got %s", report.Status)
	}
	if report.EnforcementEnabled {
		t.Fatal("expected enforcement disabled")
	}
	if report.ActualUSD != 5.0 {
		t.Fatalf("expected actual_usd=5.0, got %f", report.ActualUSD)
	}
}

func TestEvaluateCostBudgetOverBudget(t *testing.T) {
	policy := &CostBudgetPolicy{
		MaxInputTokens:   100,
		MaxOutputTokens:  100,
		MaxActualUSD:     10.0,
		Enabled:          true,
		AffectsAdmission: false,
	}
	report := EvaluateCostBudget(policy, 15.0, 50, 50, true)
	if !report.OverBudget {
		t.Fatal("expected over budget")
	}
	if report.Status != "over_budget" {
		t.Fatalf("expected status over_budget, got %s", report.Status)
	}
	if !report.EnforcementEnabled {
		t.Fatal("expected enforcement enabled")
	}
}

func TestEvaluateCostBudgetOverTokens(t *testing.T) {
	policy := &CostBudgetPolicy{
		MaxInputTokens:   100,
		MaxOutputTokens:  100,
		MaxActualUSD:     10.0,
		Enabled:          false,
		AffectsAdmission: false,
	}
	report := EvaluateCostBudget(policy, 5.0, 200, 50, true)
	if !report.OverBudget {
		t.Fatal("expected over budget")
	}
	if report.EnforcementEnabled {
		t.Fatal("expected enforcement disabled because policy is disabled")
	}
}

func TestReadCostBudgetReportSpecFormat(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")
	reportContent := `{
  "ok": true,
  "status": "within_budget",
  "over_budget": false,
  "enforcement_enabled": false,
  "actual_usd": 3.0,
  "max_usd": 5.0,
  "input_tokens": 1000,
  "max_input_tokens": 150000,
  "output_tokens": 500,
  "max_output_tokens": 45000,
  "affects_admission": false
}
`
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		t.Fatal(err)
	}
	report, err := ReadCostBudgetReport(reportPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Status != "within_budget" {
		t.Fatalf("expected within_budget, got %s", report.Status)
	}
	if report.ActualUSD != 3.0 {
		t.Fatalf("expected actual_usd=3.0, got %f", report.ActualUSD)
	}
}

func TestReadCostBudgetReportSchemaFormat(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")
	reportContent := `{
  "schema_version": "1",
  "max_usd": 5.0,
  "actual_usd": 3.0,
  "token_usage": {"input": 1000, "output": 500},
  "over_budget": false,
  "status": "within_budget",
  "recovery": "none",
  "policy_enabled": true,
  "enforcement_enabled": false,
  "admission_authority": false
}
`
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		t.Fatal(err)
	}
	report, err := ReadCostBudgetReport(reportPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Status != "within_budget" {
		t.Fatalf("expected within_budget, got %s", report.Status)
	}
	if report.InputTokens != 1000 {
		t.Fatalf("expected input_tokens=1000, got %d", report.InputTokens)
	}
	if report.OutputTokens != 500 {
		t.Fatalf("expected output_tokens=500, got %d", report.OutputTokens)
	}
}

func TestReadCostBudgetReportMissingFile(t *testing.T) {
	_, err := ReadCostBudgetReport("/nonexistent/report.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
