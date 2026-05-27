package agentprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"gopkg.in/yaml.v3"
)

// AgentProfile represents the advisory agent capability profile.
type AgentProfile struct {
	SchemaVersion        string         `json:"schema_version" yaml:"schema_version"`
	AgentID              string         `json:"agent_id" yaml:"agent_id"`
	MeasuredOn           string         `json:"measured_on" yaml:"measured_on"`
	ObservedFailureModes []string       `json:"observed_failure_modes" yaml:"observed_failure_modes"`
	RequiredExtraChecks  []string       `json:"required_extra_checks" yaml:"required_extra_checks"`
	BenchmarkMetrics     map[string]any `json:"benchmark_metrics" yaml:"benchmark_metrics"`
	AdvisoryOnly         bool           `json:"advisory_only" yaml:"advisory_only"`
	AdmissionAuthority   bool           `json:"admission_authority" yaml:"admission_authority"`
}

// SafeAgentID sanitizes an agent identifier for use in file paths.
func SafeAgentID(agentID string) string {
	return regexp.MustCompile(`[^A-Za-z0-9._-]`).ReplaceAllString(agentID, "_")
}

// DefaultAgentProfilePath returns the canonical path for an agent profile.
func DefaultAgentProfilePath(root, agentID string) string {
	return filepath.Join(root, ".x-harness", "agent-profiles", fmt.Sprintf("%s.json", SafeAgentID(agentID)))
}

// BuildAgentProfile creates a profile from an optional benchmark report.
func BuildAgentProfile(agentID string, benchmarkPath string) (*AgentProfile, error) {
	var benchmark map[string]any
	if benchmarkPath != "" {
		data, err := os.ReadFile(benchmarkPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read benchmark report: %w", err)
		}
		if err := yaml.Unmarshal(data, &benchmark); err != nil {
			return nil, fmt.Errorf("cannot parse benchmark report: %w", err)
		}
	}

	modes := collectFailureModes(benchmark)
	profile := &AgentProfile{
		SchemaVersion:        "1",
		AgentID:              agentID,
		MeasuredOn:           time.Now().UTC().Format(time.RFC3339),
		ObservedFailureModes: modes,
		RequiredExtraChecks:  extraChecks(modes),
		BenchmarkMetrics:     metricsFromBenchmark(benchmark),
		AdvisoryOnly:         true,
		AdmissionAuthority:   false,
	}
	return profile, nil
}

// ReadAgentProfile reads and unmarshals a profile from disk.
func ReadAgentProfile(path string) (*AgentProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read agent profile: %w", err)
	}
	var profile AgentProfile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("cannot parse agent profile: %w", err)
	}
	return &profile, nil
}

// WriteAgentProfile writes a profile to disk as indented JSON.
func WriteAgentProfile(profile *AgentProfile, outPath string) error {
	resolved, err := filepath.Abs(outPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(resolved, append(data, '\n'), 0644)
}

// ValidateAgentProfile validates a profile against the JSON Schema.
func ValidateAgentProfile(profile *AgentProfile, root string) error {
	schemaPath := assets.NewLocator(root).Schema("agent-profile.schema.json")
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("cannot compile schema: %w", err)
	}
	// jsonschema/v6 validates generic JSON values; convert struct to map.
	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("cannot marshal profile: %w", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("cannot unmarshal profile: %w", err)
	}
	if err := validator.Validate(doc); err != nil {
		return fmt.Errorf("agent profile validation failed: %w", err)
	}
	return nil
}

func metricsFromBenchmark(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	metrics, ok := input["metrics"].(map[string]any)
	if !ok || metrics == nil {
		return map[string]any{}
	}
	return metrics
}

func collectFailureModes(report map[string]any) []string {
	modes := make(map[string]struct{})
	metrics := metricsFromBenchmark(report)
	if intLikeValue(metrics["false_accept_count"]) > 0 {
		modes["false_accept_regression"] = struct{}{}
	}
	if intLikeValue(metrics["adversarial_false_accept_count"]) > 0 {
		modes["adversarial_false_accept"] = struct{}{}
	}
	if intLikeValue(metrics["false_reject_count"]) > 0 {
		modes["false_reject_regression"] = struct{}{}
	}

	integration, _ := report["integration"].(map[string]any)
	if integration == nil {
		integration = map[string]any{}
	}
	textBytes, _ := json.Marshal(integration)
	text := strings.ToLower(string(textBytes))
	if strings.Contains(text, "stale") {
		modes["stale_context_reference"] = struct{}{}
	}
	if strings.Contains(text, "evidence") {
		modes["evidence_scope_mismatch"] = struct{}{}
	}

	result := make([]string, 0, len(modes))
	for m := range modes {
		result = append(result, m)
	}
	sort.Strings(result)
	return result
}

func extraChecks(modes []string) []string {
	checks := map[string]struct{}{
		"standard_verify_gate": {},
	}
	for _, mode := range modes {
		switch mode {
		case "false_accept_regression":
			checks["adversarial_replay_required"] = struct{}{}
		case "adversarial_false_accept":
			checks["permission_and_mutation_replay_required"] = struct{}{}
		case "false_reject_regression":
			checks["fixture_review_required"] = struct{}{}
		case "stale_context_reference":
			checks["context_check_required"] = struct{}{}
		case "evidence_scope_mismatch":
			checks["evidence_digest_required"] = struct{}{}
		}
	}
	result := make([]string, 0, len(checks))
	for c := range checks {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

func intLikeValue(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
