package evolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupEvolveTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create evolution directory structure.
	evolveDir := filepath.Join(tmpDir, "tools", "experimental", "evolve")
	if err := os.MkdirAll(evolveDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(evolveDir, "candidates", "test-candidate"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	budgetContent := `evolution_budget:
  enabled: true
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
  - id: false_accept_zero
    statement: "No candidate can be promoted if false_accept_count increases above zero."
    forbidden_changes:
      - allow_false_accept_regression
    benchmark_required: true
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

	flatCandidateContent := `schema_version: 1
candidate_id: flat-candidate
base_commit: def456
`
	if err := os.WriteFile(filepath.Join(evolveDir, "candidates", "flat.yaml"), []byte(flatCandidateContent), 0644); err != nil {
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

func TestLoadBudget(t *testing.T) {
	root := setupEvolveTestDir(t)
	budget, err := LoadBudget(root)
	if err != nil {
		t.Fatalf("expected budget to load, got error: %v", err)
	}
	if !budget.EvolutionBudget.Enabled {
		t.Fatal("expected budget to be enabled")
	}
	if budget.EvolutionBudget.MaxCandidatesPerDay != 5 {
		t.Fatalf("expected max_candidates_per_day=5, got %d", budget.EvolutionBudget.MaxCandidatesPerDay)
	}
}

func TestLoadBudgetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadBudget(tmpDir)
	if err == nil {
		t.Fatal("expected error for missing budget")
	}
}

func TestLoadConstitution(t *testing.T) {
	root := setupEvolveTestDir(t)
	constitution, path, err := LoadConstitution(root, "")
	if err != nil {
		t.Fatalf("expected constitution to load, got error: %v", err)
	}
	if constitution.Version != 1 {
		t.Fatalf("expected version=1, got %d", constitution.Version)
	}
	if len(constitution.Invariants) != 2 {
		t.Fatalf("expected 2 invariants, got %d", len(constitution.Invariants))
	}
	if !strings.Contains(path, "constitution.yaml") {
		t.Fatalf("expected path to contain constitution.yaml, got %s", path)
	}
}

func TestLoadConstitutionExplicitPath(t *testing.T) {
	root := setupEvolveTestDir(t)
	explicit := filepath.Join(root, "tools", "experimental", "evolve", "constitution.yaml")
	constitution, path, err := LoadConstitution(root, explicit)
	if err != nil {
		t.Fatalf("expected constitution to load, got error: %v", err)
	}
	if path != explicit {
		t.Fatalf("expected path=%s, got %s", explicit, path)
	}
	if constitution.Version != 1 {
		t.Fatalf("expected version=1, got %d", constitution.Version)
	}
}

func TestResolveCandidatePath(t *testing.T) {
	root := setupEvolveTestDir(t)

	// Direct path
	direct := filepath.Join(root, "tools", "experimental", "evolve", "candidates", "test-candidate", "candidate.yaml")
	p, err := ResolveCandidatePath(root, direct)
	if err != nil {
		t.Fatalf("expected direct path to resolve, got error: %v", err)
	}
	if p != direct {
		t.Fatalf("expected %s, got %s", direct, p)
	}

	// By ID with candidate.yaml
	p, err = ResolveCandidatePath(root, "test-candidate")
	if err != nil {
		t.Fatalf("expected id to resolve, got error: %v", err)
	}
	if p != direct {
		t.Fatalf("expected %s, got %s", direct, p)
	}

	// Flat file
	flat := filepath.Join(root, "tools", "experimental", "evolve", "candidates", "flat.yaml")
	p, err = ResolveCandidatePath(root, "flat")
	if err != nil {
		t.Fatalf("expected flat to resolve, got error: %v", err)
	}
	if p != flat {
		t.Fatalf("expected %s, got %s", flat, p)
	}
}

func TestResolveCandidatePathNotFound(t *testing.T) {
	root := setupEvolveTestDir(t)
	_, err := ResolveCandidatePath(root, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing candidate")
	}
}

func TestEvaluateBudgetDisabled(t *testing.T) {
	budget := &EvolutionBudget{}
	budget.EvolutionBudget.Enabled = false
	result := EvaluateBudget(budget)
	if result.Status != "disabled" {
		t.Fatalf("expected status=disabled, got %s", result.Status)
	}
	if !result.OK {
		t.Fatal("expected ok=true")
	}
}

func TestEvaluateBudgetEnabled(t *testing.T) {
	budget := &EvolutionBudget{}
	budget.EvolutionBudget.Enabled = true
	result := EvaluateBudget(budget)
	if result.Status != "proposed" {
		t.Fatalf("expected status=proposed, got %s", result.Status)
	}
	if !result.OK {
		t.Fatal("expected ok=true")
	}
}

func TestCheckConstitutionPass(t *testing.T) {
	root := setupEvolveTestDir(t)
	constitution, cpath, err := LoadConstitution(root, "")
	if err != nil {
		t.Fatal(err)
	}
	candidate := &Candidate{
		CandidateID:      "test-candidate",
		MetricsBefore:    map[string]interface{}{"false_accept_count": 0},
		MetricsAfter:     map[string]interface{}{"false_accept_count": 0},
		TouchedPaths:     []string{"docs/README.md"},
		ForbiddenChanges: []string{},
	}
	result := CheckConstitution(constitution, cpath, candidate, "")
	if !result.OK {
		t.Fatalf("expected constitution check to pass, got violations: %v", result.Violations)
	}
	if result.Status != "passed" {
		t.Fatalf("expected status=passed, got %s", result.Status)
	}
	if result.CandidateID != "test-candidate" {
		t.Fatalf("expected candidate_id=test-candidate, got %s", result.CandidateID)
	}
}

func TestCheckConstitutionForbiddenChange(t *testing.T) {
	root := setupEvolveTestDir(t)
	constitution, cpath, err := LoadConstitution(root, "")
	if err != nil {
		t.Fatal(err)
	}
	candidate := &Candidate{
		CandidateID:      "test-candidate",
		ForbiddenChanges: []string{"disable_mutation_guard"},
	}
	result := CheckConstitution(constitution, cpath, candidate, "")
	if result.OK {
		t.Fatal("expected constitution check to fail")
	}
	if len(result.Violations) == 0 {
		t.Fatal("expected violations")
	}
	found := false
	for _, v := range result.Violations {
		if strings.Contains(v, "disable_mutation_guard") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected forbidden change violation, got: %v", result.Violations)
	}
}

func TestCheckConstitutionProtectedPath(t *testing.T) {
	root := setupEvolveTestDir(t)
	constitution, cpath, err := LoadConstitution(root, "")
	if err != nil {
		t.Fatal(err)
	}
	candidate := &Candidate{
		CandidateID:  "test-candidate",
		TouchedPaths: []string{"packages/cli/src/core/mutation-guard.ts"},
	}
	result := CheckConstitution(constitution, cpath, candidate, "")
	if result.OK {
		t.Fatal("expected constitution check to fail")
	}
	found := false
	for _, v := range result.Violations {
		if strings.Contains(v, "protected path") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected protected path violation, got: %v", result.Violations)
	}
}

func TestCheckConstitutionBenchmarkFalseAccept(t *testing.T) {
	root := setupEvolveTestDir(t)
	constitution, cpath, err := LoadConstitution(root, "")
	if err != nil {
		t.Fatal(err)
	}
	candidate := &Candidate{
		CandidateID:   "test-candidate",
		MetricsBefore: map[string]interface{}{"false_accept_count": 0},
		MetricsAfter:  map[string]interface{}{"false_accept_count": 1},
	}
	result := CheckConstitution(constitution, cpath, candidate, "")
	if result.OK {
		t.Fatal("expected constitution check to fail")
	}
	found := false
	for _, v := range result.Violations {
		if strings.Contains(v, "false_accept_count increased from baseline") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected false accept violation, got: %v", result.Violations)
	}
}

func TestRenderChangeRequest(t *testing.T) {
	content := RenderChangeRequest("proposal", "Test summary", "test-component", "cand-1", nil)
	if !strings.Contains(content, "# x-harness Evolution proposal") {
		t.Fatal("expected proposal header")
	}
	if !strings.Contains(content, "component: test-component") {
		t.Fatal("expected component line")
	}
	if !strings.Contains(content, "candidate_id: cand-1") {
		t.Fatal("expected candidate_id line")
	}
	if !strings.Contains(content, "This file is a change request only") {
		t.Fatal("expected boundary line")
	}
}

func TestRenderChangeRequestWithConstitution(t *testing.T) {
	checkResult := &ConstitutionCheckResult{
		Status:     "failed",
		Violations: []string{"v1", "v2"},
	}
	content := RenderChangeRequest("promotion", "Promote", "", "cand-1", checkResult)
	if !strings.Contains(content, "constitution_status: failed") {
		t.Fatal("expected constitution_status")
	}
	if !strings.Contains(content, "## Violations") {
		t.Fatal("expected violations section")
	}
	if !strings.Contains(content, "- v1") {
		t.Fatal("expected violation v1")
	}
}

func TestWriteChangeRequest(t *testing.T) {
	tmpDir := t.TempDir()
	content := "test content"
	path, err := WriteChangeRequest(tmpDir, content, "")
	if err != nil {
		t.Fatalf("expected write to succeed, got error: %v", err)
	}
	if !strings.Contains(path, ".x-harness/evolution/change-requests") {
		t.Fatalf("expected path under change-requests, got %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("expected %q, got %q", content, string(b))
	}
}

func TestWriteChangeRequestExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	content := "explicit content"
	path, err := WriteChangeRequest(tmpDir, content, ".x-harness/evolution/change-requests/custom.md")
	if err != nil {
		t.Fatalf("expected write to succeed, got error: %v", err)
	}
	if !strings.HasSuffix(path, "custom.md") {
		t.Fatalf("expected path to end with custom.md, got %s", path)
	}
}

func TestWriteChangeRequestOutsideDir(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := WriteChangeRequest(tmpDir, "bad", "../../escape.md")
	if err == nil {
		t.Fatal("expected error for path outside change-requests dir")
	}
}

func TestWriteChangeRequestAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".x-harness", "evolution", "change-requests", "existing.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := WriteChangeRequest(tmpDir, "new", "existing.md")
	if err == nil {
		t.Fatal("expected error for existing file")
	}
}
