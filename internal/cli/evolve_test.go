package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupEvolveCLITestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	evolveDir := filepath.Join(tmpDir, "tools", "experimental", "evolve")
	if err := os.MkdirAll(filepath.Join(evolveDir, "candidates", "test-candidate"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	budgetContent := `evolution_budget:
  enabled: false
  max_candidates_per_day: 5
  max_runtime_minutes_per_run: 30
  max_cost_usd_per_run: 10
  min_failure_pattern_count: 3
  require_h2_maturity: true
  require_adversarial_suite: true
`
	if err := os.WriteFile(filepath.Join(evolveDir, "evolution-budget.yaml"), []byte(budgetContent), 0644); err != nil {
		t.Fatal(err)
	}

	constitutionContent := `version: 1
invariants:
  - id: verify_gate_read_only
    statement: "Verify gate must remain read-only."
    protected_paths:
      - packages/cli/src/core/mutation-guard.ts
    forbidden_changes:
      - disable_mutation_guard
`
	if err := os.WriteFile(filepath.Join(evolveDir, "constitution.yaml"), []byte(constitutionContent), 0644); err != nil {
		t.Fatal(err)
	}

	candidateContent := `schema_version: 1
candidate_id: test-candidate
base_commit: abc123
component_ids:
  - test-component
change_summary: "Test change"
metrics_before:
  false_accept_count: 0
metrics_after:
  false_accept_count: 0
touched_paths:
  - docs/README.md
forbidden_changes: []
`
	if err := os.WriteFile(filepath.Join(evolveDir, "candidates", "test-candidate", "candidate.yaml"), []byte(candidateContent), 0644); err != nil {
		t.Fatal(err)
	}

	badCandidateContent := `schema_version: 1
candidate_id: bad-candidate
base_commit: abc123
metrics_before:
  false_accept_count: 0
metrics_after:
  false_accept_count: 1
touched_paths:
  - packages/cli/src/core/mutation-guard.ts
forbidden_changes:
  - disable_mutation_guard
`
	if err := os.WriteFile(filepath.Join(evolveDir, "candidates", "bad-candidate.yaml"), []byte(badCandidateContent), 0644); err != nil {
		t.Fatal(err)
	}

	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "evolution-constitution",
  "type": "object",
  "required": ["version", "invariants"],
  "properties": {
    "version": { "type": "integer", "minimum": 1 },
    "invariants": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["id", "statement"],
        "properties": {
          "id": { "type": "string", "minLength": 1 },
          "statement": { "type": "string", "minLength": 1 },
          "protected_paths": { "type": "array", "items": { "type": "string" } },
          "forbidden_changes": { "type": "array", "items": { "type": "string" } },
          "benchmark_required": { "type": "boolean" }
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "evolution-constitution.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func TestEvolveMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestEvolveUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown evolve subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestEvolveEvaluateTextOutput(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "evaluate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "disabled") {
		t.Fatalf("expected disabled message, got: %s", out)
	}
}

func TestEvolveEvaluateJSONOutput(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "evaluate", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["status"] != "disabled" {
		t.Fatalf("expected status=disabled, got: %v", result)
	}
}

func TestEvolveAnalyzeMissingRun(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "analyze", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--run") {
		t.Fatalf("expected missing --run error, got: %s", stderr.String())
	}
}

func TestEvolveAnalyzeWritesRequest(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "analyze", "--run", "run-123", "--root", tmpDir, "--out", ".x-harness/evolution/change-requests/test-analysis.md"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "analysis request written:") {
		t.Fatalf("expected written message, got: %s", out)
	}
	path := filepath.Join(tmpDir, ".x-harness", "evolution", "change-requests", "test-analysis.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist at %s", path)
	}
}

func TestEvolveAnalyzeJSONOutput(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "analyze", "--run", "run-123", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["run_id"] != "run-123" {
		t.Fatalf("expected run_id=run-123, got: %v", result)
	}
	if result["status"] != "proposed" {
		t.Fatalf("expected status=proposed, got: %v", result)
	}
}

func TestEvolveProposeMissingComponent(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "propose", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--component") {
		t.Fatalf("expected missing --component error, got: %s", stderr.String())
	}
}

func TestEvolveProposeWritesRequest(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "propose", "--component", "test-comp", "--root", tmpDir, "--write"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "change request written:") {
		t.Fatalf("expected written message, got: %s", out)
	}
}

func TestEvolveProposeJSONOutput(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "propose", "--component", "test-comp", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["component"] != "test-comp" {
		t.Fatalf("expected component=test-comp, got: %v", result)
	}
	if result["status"] != "proposed" {
		t.Fatalf("expected status=proposed, got: %v", result)
	}
}

func TestEvolveConstitutionCheckPass(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "constitution-check", "--candidate", "test-candidate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "constitution passed: test-candidate") {
		t.Fatalf("expected passed message, got: %s", out)
	}
}

func TestEvolveConstitutionCheckFail(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "constitution-check", "--candidate", "bad-candidate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "constitution failed: bad-candidate") {
		t.Fatalf("expected failed message, got: %s", out)
	}
}

func TestEvolveConstitutionCheckJSON(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "constitution-check", "--candidate", "test-candidate", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["status"] != "passed" {
		t.Fatalf("expected status=passed, got: %v", result)
	}
}

func TestEvolveConstitutionCheckMissingCandidate(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "constitution-check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--candidate") {
		t.Fatalf("expected missing --candidate error, got: %s", stderr.String())
	}
}

func TestEvolveComparePass(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "compare", "--candidate", "test-candidate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
}

func TestEvolveCompareFail(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "compare", "--candidate", "bad-candidate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
}

func TestEvolveCompareJSON(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "compare", "--candidate", "test-candidate", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["false_accept_regression"] != false {
		t.Fatalf("expected false_accept_regression=false, got: %v", result)
	}
}

func TestEvolvePromotePass(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "promote", "--candidate", "test-candidate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "promotion request written:") {
		t.Fatalf("expected written message, got: %s", out)
	}
}

func TestEvolvePromoteBlocked(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "promote", "--candidate", "bad-candidate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "promotion blocked by constitution") {
		t.Fatalf("expected blocked message, got: %s", stderr.String())
	}
}

func TestEvolvePromoteJSON(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "promote", "--candidate", "test-candidate", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["status"] != "written" {
		t.Fatalf("expected status=written, got: %v", result)
	}
}

func TestEvolvePromoteMissingCandidate(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "promote", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--candidate") {
		t.Fatalf("expected missing --candidate error, got: %s", stderr.String())
	}
}

func TestEvolveRollback(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "rollback", "--candidate", "test-candidate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "rollback request written:") {
		t.Fatalf("expected written message, got: %s", out)
	}
}

func TestEvolveRollbackJSON(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "rollback", "--candidate", "test-candidate", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["status"] != "written" {
		t.Fatalf("expected status=written, got: %v", result)
	}
}

func TestEvolveRollbackMissingCandidate(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "rollback", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--candidate") {
		t.Fatalf("expected missing --candidate error, got: %s", stderr.String())
	}
}

func TestEvolveUnknownFlag(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "evaluate", "--root", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestEvolveEvaluateEnabledBudget(t *testing.T) {
	tmpDir := setupEvolveCLITestDir(t)
	budgetContent := `evolution_budget:
  enabled: true
  max_candidates_per_day: 5
  max_runtime_minutes_per_run: 30
  max_cost_usd_per_run: 10
  min_failure_pattern_count: 3
  require_h2_maturity: true
  require_adversarial_suite: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "tools", "experimental", "evolve", "evolution-budget.yaml"), []byte(budgetContent), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evolve", "evaluate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "evolution budget enabled") {
		t.Fatalf("expected enabled message, got: %s", out)
	}
}
