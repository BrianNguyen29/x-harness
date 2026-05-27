package evolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

// EvolutionBudget wraps the evolution_budget block from evolution-budget.yaml.
type EvolutionBudget struct {
	EvolutionBudget struct {
		Enabled                 bool `yaml:"enabled"`
		MaxCandidatesPerDay     int  `yaml:"max_candidates_per_day"`
		MaxRuntimeMinutesPerRun int  `yaml:"max_runtime_minutes_per_run"`
		MaxCostUSDPerRun        int  `yaml:"max_cost_usd_per_run"`
		MinFailurePatternCount  int  `yaml:"min_failure_pattern_count"`
		RequireH2Maturity       bool `yaml:"require_h2_maturity"`
		RequireAdversarialSuite bool `yaml:"require_adversarial_suite"`
	} `yaml:"evolution_budget"`
}

// EvolutionInvariant is a single rule inside the constitution.
type EvolutionInvariant struct {
	ID                string   `yaml:"id"`
	Statement         string   `yaml:"statement"`
	ProtectedPaths    []string `yaml:"protected_paths"`
	ForbiddenChanges  []string `yaml:"forbidden_changes"`
	BenchmarkRequired bool     `yaml:"benchmark_required"`
}

// Constitution is the parsed evolution constitution.
type Constitution struct {
	Version    int                  `yaml:"version"`
	Invariants []EvolutionInvariant `yaml:"invariants"`
}

// Candidate is the parsed evolution candidate manifest.
type Candidate struct {
	SchemaVersion    int                    `yaml:"schema_version"`
	CandidateID      string                 `yaml:"candidate_id"`
	BaseCommit       string                 `yaml:"base_commit"`
	ComponentIDs     []string               `yaml:"component_ids"`
	ChangeSummary    string                 `yaml:"change_summary"`
	Prediction       map[string]interface{} `yaml:"prediction"`
	MetricsBefore    map[string]interface{} `yaml:"metrics_before"`
	MetricsAfter     map[string]interface{} `yaml:"metrics_after"`
	RegressionBudget map[string]interface{} `yaml:"regression_budget"`
	ForbiddenChanges []string               `yaml:"forbidden_changes"`
	TouchedPaths     []string               `yaml:"touched_paths"`
	Constitution     map[string]interface{} `yaml:"constitution"`
	PromotionStatus  string                 `yaml:"promotion_status"`
	Rollback         map[string]interface{} `yaml:"rollback"`
}

// ConstitutionCheckResult is returned by CheckConstitution.
type ConstitutionCheckResult struct {
	OK                 bool     `json:"ok"`
	Status             string   `json:"status"`
	CandidateID        string   `json:"candidate_id"`
	ConstitutionPath   string   `json:"constitution_path"`
	CandidatePath      string   `json:"candidate_path"`
	Violations         []string `json:"violations"`
	CheckedInvariants  []string `json:"checked_invariants"`
	AdmissionAuthority bool     `json:"admission_authority"`
}

// EvolutionRequestResult is the generic result used by evaluate and analyze.
type EvolutionRequestResult struct {
	OK                 bool   `json:"ok"`
	Status             string `json:"status"`
	Path               string `json:"path,omitempty"`
	Message            string `json:"message,omitempty"`
	AdmissionAuthority bool   `json:"admission_authority"`
}

// EvolutionResult is a generic envelope used by some subcommands.
type EvolutionResult struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}

func evolutionRoot(root string) string {
	return filepath.Join(root, "tools", "experimental", "evolve")
}

func defaultConstitutionPath(root string) string {
	return filepath.Join(evolutionRoot(root), "constitution.yaml")
}

func defaultBudgetPath(root string) string {
	return filepath.Join(evolutionRoot(root), "evolution-budget.yaml")
}

// LoadBudget loads the evolution budget from the default location under root.
func LoadBudget(root string) (*EvolutionBudget, error) {
	path := defaultBudgetPath(root)
	var budget EvolutionBudget
	if err := loader.LoadDocument(path, &budget); err != nil {
		return nil, err
	}
	return &budget, nil
}

// LoadConstitution loads and validates the constitution.
func LoadConstitution(root, explicitPath string) (*Constitution, string, error) {
	path := explicitPath
	if path == "" {
		path = defaultConstitutionPath(root)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}

	var constitution Constitution
	if err := loader.LoadDocument(path, &constitution); err != nil {
		return nil, "", err
	}

	// Validate against JSON schema if available.
	schemaPath := filepath.Join(root, "schemas", "evolution-constitution.schema.json")
	if _, err := os.Stat(schemaPath); err == nil {
		v, err := schema.Compile(schemaPath)
		if err != nil {
			return nil, "", fmt.Errorf("constitution schema compilation failed: %w", err)
		}
		var raw any
		if err := loader.LoadDocument(path, &raw); err != nil {
			return nil, "", err
		}
		if err := v.Validate(raw); err != nil {
			return nil, "", fmt.Errorf("constitution schema validation failed: %w", err)
		}
	}

	return &constitution, path, nil
}

// ResolveCandidatePath resolves a candidate identifier to an absolute file path.
func ResolveCandidatePath(root, candidate string) (string, error) {
	direct := candidate
	if !filepath.IsAbs(candidate) {
		direct = filepath.Join(root, candidate)
	}
	if _, err := os.Stat(direct); err == nil {
		return direct, nil
	}

	candidatesDir := filepath.Join(evolutionRoot(root), "candidates")
	for _, suffix := range []string{"candidate.yaml", "candidate.yml", candidate + ".yaml"} {
		p := filepath.Join(candidatesDir, candidate, suffix)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	flat := filepath.Join(candidatesDir, candidate+".yaml")
	if _, err := os.Stat(flat); err == nil {
		return flat, nil
	}

	return "", fmt.Errorf("candidate not found: %s", candidate)
}

// LoadCandidate loads a candidate manifest.
func LoadCandidate(root, candidate string) (*Candidate, string, error) {
	candidatePath, err := ResolveCandidatePath(root, candidate)
	if err != nil {
		return nil, "", err
	}
	var c Candidate
	if err := loader.LoadDocument(candidatePath, &c); err != nil {
		return nil, "", err
	}
	if c.CandidateID == "" {
		return nil, "", fmt.Errorf("candidate manifest missing candidate_id")
	}
	return &c, candidatePath, nil
}

// EvaluateBudget checks whether the evolution budget is enabled.
func EvaluateBudget(budget *EvolutionBudget) *EvolutionRequestResult {
	if !budget.EvolutionBudget.Enabled {
		return &EvolutionRequestResult{
			OK:                 true,
			Status:             "disabled",
			Path:               "",
			Message:            "evolution budget is disabled; no candidate loop will run",
			AdmissionAuthority: false,
		}
	}
	return &EvolutionRequestResult{
		OK:                 true,
		Status:             "proposed",
		Path:               "",
		Message:            "evolution budget enabled; external model loop is not implemented in local MVP",
		AdmissionAuthority: false,
	}
}

func listIncludes(items []string, item string) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}
	return false
}

func pathMatches(pattern, filePath string) bool {
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-2]
		return filePath == pattern[:len(pattern)-3] || strings.HasPrefix(filePath, prefix)
	}
	return pattern == filePath
}

func numberMetric(record map[string]interface{}, key string) float64 {
	if record == nil {
		return 0
	}
	v, ok := record[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

// CheckConstitution validates a candidate against the constitution.
func CheckConstitution(constitution *Constitution, constitutionPath string, candidate *Candidate, candidatePath string) *ConstitutionCheckResult {
	violations := []string{}
	forbiddenChanges := candidate.ForbiddenChanges
	touchedPaths := candidate.TouchedPaths

	for _, invariant := range constitution.Invariants {
		for _, forbidden := range invariant.ForbiddenChanges {
			if listIncludes(forbiddenChanges, forbidden) {
				violations = append(violations, fmt.Sprintf("%s: candidate declares forbidden change %s", invariant.ID, forbidden))
			}
		}
		for _, protectedPath := range invariant.ProtectedPaths {
			for _, item := range touchedPaths {
				if pathMatches(protectedPath, item) {
					violations = append(violations, fmt.Sprintf("%s: candidate touches protected path %s", invariant.ID, protectedPath))
				}
			}
		}
		if invariant.BenchmarkRequired {
			afterFalseAccept := numberMetric(candidate.MetricsAfter, "false_accept_count")
			afterAdversarialFalseAccept := numberMetric(candidate.MetricsAfter, "adversarial_false_accept_count")
			if afterFalseAccept > 0 || afterAdversarialFalseAccept > 0 {
				violations = append(violations, fmt.Sprintf("%s: benchmark false accepts must remain zero", invariant.ID))
			}
		}
	}

	beforeFalseAccept := numberMetric(candidate.MetricsBefore, "false_accept_count")
	afterFalseAccept := numberMetric(candidate.MetricsAfter, "false_accept_count")
	if afterFalseAccept > beforeFalseAccept {
		violations = append(violations, "false_accept_count increased from baseline")
	}

	checked := make([]string, len(constitution.Invariants))
	for i, inv := range constitution.Invariants {
		checked[i] = inv.ID
	}

	ok := len(violations) == 0
	status := "passed"
	if !ok {
		status = "failed"
	}

	return &ConstitutionCheckResult{
		OK:                 ok,
		Status:             status,
		CandidateID:        candidate.CandidateID,
		ConstitutionPath:   constitutionPath,
		CandidatePath:      candidatePath,
		Violations:         violations,
		CheckedInvariants:  checked,
		AdmissionAuthority: false,
	}
}

// RenderChangeRequest renders a change request markdown document.
func RenderChangeRequest(kind, summary, component, candidateID string, constitution *ConstitutionCheckResult) string {
	lines := []string{
		fmt.Sprintf("# x-harness Evolution %s", kind),
		"",
		fmt.Sprintf("summary: %s", summary),
		"admission_authority: false",
	}
	if component != "" {
		lines = append(lines, fmt.Sprintf("component: %s", component))
	}
	if candidateID != "" {
		lines = append(lines, fmt.Sprintf("candidate_id: %s", candidateID))
	}
	if constitution != nil {
		lines = append(lines, fmt.Sprintf("constitution_status: %s", constitution.Status))
		if len(constitution.Violations) > 0 {
			lines = append(lines, "", "## Violations")
			for _, v := range constitution.Violations {
				lines = append(lines, fmt.Sprintf("- %s", v))
			}
		}
	}
	lines = append(lines, "", "## Boundary", "", "This file is a change request only. It does not promote, merge, or mutate harness policy.")
	return strings.Join(lines, "\n") + "\n"
}

// WriteChangeRequest writes a change request to the evolution change-requests directory.
func WriteChangeRequest(root, content, outPath string) (string, error) {
	baseDir := filepath.Join(root, ".x-harness", "evolution", "change-requests")
	var target string
	if outPath != "" {
		target = filepath.Join(root, outPath)
	} else {
		target = filepath.Join(baseDir, fmt.Sprintf("request-%d.md", time.Now().UnixMilli()))
	}

	// Ensure target is under baseDir.
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) && absTarget != absBase {
		return "", fmt.Errorf("evolution change requests must be written under .x-harness/evolution/change-requests")
	}

	if _, err := os.Stat(absTarget); err == nil {
		return "", fmt.Errorf("evolution change request already exists; refusing to overwrite: %s", absTarget)
	}

	if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(absTarget, []byte(content), 0644); err != nil {
		return "", err
	}
	return absTarget, nil
}
