package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupCostCLITestDir(t *testing.T) string {
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

func TestCostCheckWithinBudget(t *testing.T) {
	tmpDir := setupCostCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "check", "--actual-usd", "3.0", "--input-tokens", "1000", "--output-tokens", "500", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "status: within_budget") {
		t.Fatalf("expected within_budget status, got: %s", out)
	}
	if !strings.Contains(out, "over_budget: false") {
		t.Fatalf("expected over_budget=false, got: %s", out)
	}
}

func TestCostCheckOverBudget(t *testing.T) {
	tmpDir := setupCostCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "check", "--actual-usd", "10.0", "--input-tokens", "1000", "--output-tokens", "500", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "status: over_budget") {
		t.Fatalf("expected over_budget status, got: %s", out)
	}
}

func TestCostCheckOverBudgetEnforce(t *testing.T) {
	tmpDir := setupCostCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "check", "--actual-usd", "10.0", "--input-tokens", "1000", "--output-tokens", "500", "--root", tmpDir, "--enforce"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
}

func TestCostCheckMissingFlags(t *testing.T) {
	tmpDir := setupCostCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "check", "--actual-usd", "3.0", "--input-tokens", "1000", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--output-tokens is required") {
		t.Fatalf("expected missing flag error, got: %s", stderr.String())
	}
}

func TestCostCheckInvalidNumber(t *testing.T) {
	tmpDir := setupCostCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "check", "--actual-usd", "abc", "--input-tokens", "1000", "--output-tokens", "500", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
}

func TestCostCheckJSONOutput(t *testing.T) {
	tmpDir := setupCostCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "check", "--actual-usd", "3.0", "--input-tokens", "1000", "--output-tokens", "500", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["status"] != "within_budget" {
		t.Fatalf("expected status=within_budget, got: %v", result)
	}
}

func TestCostReport(t *testing.T) {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "report", "--from", reportPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "cost budget: within_budget") {
		t.Fatalf("expected cost budget status, got: %s", out)
	}
}

func TestCostReportJSON(t *testing.T) {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "report", "--from", reportPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["status"] != "within_budget" {
		t.Fatalf("expected status=within_budget, got: %v", result)
	}
}

func TestCostReportSchemaValidation(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "schemas", "cost-budget.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "cost-budget.schema.json"), schemaBytes, 0644); err != nil {
		t.Fatal(err)
	}

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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "report", "--from", reportPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "cost budget: within_budget") {
		t.Fatalf("expected cost budget status, got: %s", out)
	}
}

func TestCostReportSchemaValidationFail(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "schemas", "cost-budget.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "cost-budget.schema.json"), schemaBytes, 0644); err != nil {
		t.Fatal(err)
	}

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
  "enforcement_enabled": false
}
`
	// Missing admission_authority - should fail validation
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "report", "--from", reportPath}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "validation failed") {
		t.Fatalf("expected validation failed error, got: %s", stderr.String())
	}
}

func TestCostReportMissingFrom(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "report"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--from is required") {
		t.Fatalf("expected missing --from error, got: %s", stderr.String())
	}
}

func TestCostMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestCostUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown cost subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestCostUnknownFlag(t *testing.T) {
	tmpDir := setupCostCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"cost", "check", "--actual-usd", "3.0", "--input-tokens", "1000", "--output-tokens", "500", "--root", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
