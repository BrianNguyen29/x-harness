package intake

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/loader"
)

// IntakeLabel represents the intake classification label.
type IntakeLabel string

const (
	IntakeLabelTiny     IntakeLabel = "tiny"
	IntakeLabelNormal   IntakeLabel = "normal"
	IntakeLabelHighRisk IntakeLabel = "high_risk"
)

// RuntimeTier represents the canonical runtime tier.
type RuntimeTier string

const (
	RuntimeTierLight    RuntimeTier = "light"
	RuntimeTierStandard RuntimeTier = "standard"
	RuntimeTierDeep     RuntimeTier = "deep"
)

// IntakePolicy represents the intake.yaml structure.
type IntakePolicy struct {
	Version                 int                         `yaml:"version"`
	IntakeLabels            map[IntakeLabel]LabelConfig `yaml:"intake_labels"`
	HighRiskSignals         map[string]SignalConfig     `yaml:"high_risk_signals"`
	RuntimeTierConfirmation struct {
		Tiers []RuntimeTier `yaml:"tiers"`
		Note  string        `yaml:"note"`
	} `yaml:"runtime_tier_confirmation"`
}

// LabelConfig represents the config for an intake label.
type LabelConfig struct {
	RuntimeTier RuntimeTier `yaml:"runtime_tier"`
	Signals     []string    `yaml:"signals"`
}

// SignalConfig represents the config for a high-risk signal.
type SignalConfig struct {
	Description string   `yaml:"description"`
	Examples    []string `yaml:"examples"`
}

// IntakeClassification is the result of classifying a task.
type IntakeClassification struct {
	IntakeLabel               IntakeLabel `json:"intake_label"`
	RuntimeTier               RuntimeTier `json:"runtime_tier"`
	Reasoning                 []string    `json:"reasoning"`
	Signals                   []string    `json:"signals"`
	NegativeSignalsConsidered []string    `json:"negative_signals_considered"`
	AutoEscalated             bool        `json:"auto_escalated"`
}

// IntakeExplanation is the result of explaining a card's intake.
type IntakeExplanation struct {
	OK                        bool         `json:"ok"`
	Source                    string       `json:"source"`
	DeclaredTier              *RuntimeTier `json:"declared_tier"`
	IntakeLabel               IntakeLabel  `json:"intake_label"`
	MappedTier                RuntimeTier  `json:"mapped_tier"`
	TierDowngrade             bool         `json:"tier_downgrade"`
	InterventionRequired      bool         `json:"intervention_required"`
	InterventionApproved      bool         `json:"intervention_approved"`
	Reasoning                 []string     `json:"reasoning"`
	Signals                   []string     `json:"signals"`
	NegativeSignalsConsidered []string     `json:"negative_signals_considered"`
	Errors                    []string     `json:"errors"`
	Warnings                  []string     `json:"warnings"`
}

var highRiskKeywords = []string{
	"auth", "token", "session", "admission", "schema",
	"permissions", "permission", "ci", "release", "destroy", "delete",
}

var highRiskFilePatterns = []struct {
	Signal  string
	Pattern *regexp.Regexp
}{
	{Signal: "auth", Pattern: regexp.MustCompile(`auth`)},
	{Signal: "token", Pattern: regexp.MustCompile(`token`)},
	{Signal: "session", Pattern: regexp.MustCompile(`session`)},
	{Signal: "admission", Pattern: regexp.MustCompile(`admission`)},
	{Signal: "permissions", Pattern: regexp.MustCompile(`permission`)},
	{Signal: "schema", Pattern: regexp.MustCompile(`schema`)},
	{Signal: "release", Pattern: regexp.MustCompile(`release`)},
}

var destructivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`-rf`),
	regexp.MustCompile(`rm\s+-r`),
	regexp.MustCompile(`unlink`),
	regexp.MustCompile(`del.*tree`),
}

var tierRank = map[RuntimeTier]int{
	RuntimeTierLight:    0,
	RuntimeTierStandard: 1,
	RuntimeTierDeep:     2,
}

// IsRuntimeTier checks if a value is a valid runtime tier.
func IsRuntimeTier(value string) bool {
	return value == string(RuntimeTierLight) || value == string(RuntimeTierStandard) || value == string(RuntimeTierDeep)
}

// IsIntakeLabel checks if a value is a valid intake label.
func IsIntakeLabel(value string) bool {
	return value == string(IntakeLabelTiny) || value == string(IntakeLabelNormal) || value == string(IntakeLabelHighRisk)
}

// IsTierDowngrade checks if declared tier is lower than mapped tier.
func IsTierDowngrade(declaredTier, mappedTier RuntimeTier) bool {
	return tierRank[declaredTier] < tierRank[mappedTier]
}

// HasApprovedTierDowngradeIntervention checks if governance approves tier downgrade.
func HasApprovedTierDowngradeIntervention(governance map[string]any) bool {
	if governance == nil {
		return false
	}
	approvalStatus, _ := governance["approval_status"].(string)
	if approvalStatus != "approved" {
		return false
	}
	approvalRequiredFor, _ := governance["approval_required_for"].([]any)
	for _, item := range approvalRequiredFor {
		s, ok := item.(string)
		if !ok {
			continue
		}
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "-", "_"), " ", "_"))
		if normalized == "tier_downgrade" || normalized == "intake_tier_downgrade" {
			return true
		}
	}
	return false
}

// LoadIntakePolicy loads the intake policy from the given repository root.
func LoadIntakePolicy(root string) (*IntakePolicy, error) {
	path := filepath.Join(root, "policies", "intake.yaml")
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	var policy IntakePolicy
	if err := loader.LoadYAML(path, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// ClassifyTask classifies a task based on signals and file paths.
func ClassifyTask(task string, files []string, change string, policy *IntakePolicy) IntakeClassification {
	reasoning := []string{}
	signals := []string{}
	negativeSignalsConsidered := []string{}
	taskLower := strings.ToLower(task)

	if change == "comment-only" || change == "comments" || change == "comment" {
		signals = append(signals, "comment_only")
		reasoning = append(reasoning, "Change signal indicates comment-only modification")
		reasoning = append(reasoning, "Mapping to tiny/light")
		return IntakeClassification{
			IntakeLabel:               IntakeLabelTiny,
			RuntimeTier:               RuntimeTierLight,
			Reasoning:                 reasoning,
			Signals:                   signals,
			NegativeSignalsConsidered: []string{"behavior_change"},
			AutoEscalated:             false,
		}
	}
	negativeSignalsConsidered = append(negativeSignalsConsidered, "comment_only")

	for _, keyword := range highRiskKeywords {
		if strings.Contains(taskLower, keyword) {
			signals = append(signals, keyword)
			reasoning = append(reasoning, fmt.Sprintf("Task description contains high-risk keyword: %s", keyword))
			reasoning = append(reasoning, "Mapping to high_risk/deep")
			return IntakeClassification{
				IntakeLabel:               IntakeLabelHighRisk,
				RuntimeTier:               RuntimeTierDeep,
				Reasoning:                 reasoning,
				Signals:                   signals,
				NegativeSignalsConsidered: negativeSignalsConsidered,
				AutoEscalated:             true,
			}
		}
	}

	for _, file := range files {
		fileLower := strings.ToLower(file)
		for _, hrp := range highRiskFilePatterns {
			if hrp.Pattern.MatchString(fileLower) {
				signals = append(signals, hrp.Signal)
				reasoning = append(reasoning, fmt.Sprintf("File path matches high-risk pattern: %s", hrp.Pattern.String()))
				reasoning = append(reasoning, "Mapping to high_risk/deep")
				return IntakeClassification{
					IntakeLabel:               IntakeLabelHighRisk,
					RuntimeTier:               RuntimeTierDeep,
					Reasoning:                 reasoning,
					Signals:                   signals,
					NegativeSignalsConsidered: negativeSignalsConsidered,
					AutoEscalated:             true,
				}
			}
		}
	}

	for _, file := range files {
		if strings.Contains(file, ".github/workflows") {
			signals = append(signals, "ci")
			reasoning = append(reasoning, "Files include CI/CD workflows")
			reasoning = append(reasoning, "Mapping to high_risk/deep")
			return IntakeClassification{
				IntakeLabel:               IntakeLabelHighRisk,
				RuntimeTier:               RuntimeTierDeep,
				Reasoning:                 reasoning,
				Signals:                   signals,
				NegativeSignalsConsidered: negativeSignalsConsidered,
				AutoEscalated:             true,
			}
		}
	}

	for _, file := range files {
		for _, pattern := range destructivePatterns {
			if pattern.MatchString(file) {
				signals = append(signals, "destructive_filesystem")
				reasoning = append(reasoning, fmt.Sprintf("File path suggests destructive operation: %s", pattern.String()))
				reasoning = append(reasoning, "Mapping to high_risk/deep")
				return IntakeClassification{
					IntakeLabel:               IntakeLabelHighRisk,
					RuntimeTier:               RuntimeTierDeep,
					Reasoning:                 reasoning,
					Signals:                   signals,
					NegativeSignalsConsidered: negativeSignalsConsidered,
					AutoEscalated:             true,
				}
			}
		}
	}

	signals = append(signals, "routine_implementation")
	negativeSignalsConsidered = append(negativeSignalsConsidered, "auth", "token", "session", "ci", "release")
	reasoning = append(reasoning, "No high-risk signals detected")
	reasoning = append(reasoning, "Mapping to normal/standard")
	return IntakeClassification{
		IntakeLabel:               IntakeLabelNormal,
		RuntimeTier:               RuntimeTierStandard,
		Reasoning:                 reasoning,
		Signals:                   signals,
		NegativeSignalsConsidered: negativeSignalsConsidered,
		AutoEscalated:             false,
	}
}

func getStringArray(value any) []string {
	slice, ok := value.([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, item := range slice {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func getCardFiles(card map[string]any) []string {
	evidence, _ := card["evidence"].(map[string]any)
	state, _ := card["state"].(map[string]any)
	var files []string
	files = append(files, getStringArray(evidence["files_changed"])...)
	files = append(files, getStringArray(state["write_set"])...)
	return files
}

func getCardTask(card map[string]any) string {
	claim, _ := card["claim"].(map[string]any)
	if claim != nil {
		if summary, ok := claim["summary"].(string); ok && strings.TrimSpace(summary) != "" {
			return summary
		}
	}
	if taskID, ok := card["task_id"].(string); ok && strings.TrimSpace(taskID) != "" {
		return taskID
	}
	return "unknown"
}

// ExplainCardIntake explains the intake classification for a completion card.
func ExplainCardIntake(card map[string]any, policy *IntakePolicy) IntakeExplanation {
	errors := []string{}
	warnings := []string{}

	var declaredTier *RuntimeTier
	if tier, ok := card["tier"].(string); ok && IsRuntimeTier(tier) {
		t := RuntimeTier(tier)
		declaredTier = &t
	}

	governance, _ := card["governance"].(map[string]any)
	intake, _ := card["intake"].(map[string]any)

	source := "inferred"
	var classification IntakeClassification

	if intake != nil {
		source = "declared"
		intakeLabel, _ := intake["classification"].(string)
		mappedTier, _ := intake["mapped_tier"].(string)

		if !IsIntakeLabel(intakeLabel) {
			errors = append(errors, "intake.classification must be tiny, normal, or high_risk")
		}
		if !IsRuntimeTier(mappedTier) {
			errors = append(errors, "intake.mapped_tier must be light, standard, or deep")
		}

		normalizedLabel := IntakeLabelNormal
		if IsIntakeLabel(intakeLabel) {
			normalizedLabel = IntakeLabel(intakeLabel)
		}
		normalizedTier := policy.IntakeLabels[normalizedLabel].RuntimeTier
		if IsRuntimeTier(mappedTier) {
			normalizedTier = RuntimeTier(mappedTier)
		}
		policyTier := policy.IntakeLabels[normalizedLabel].RuntimeTier
		if normalizedTier != policyTier {
			errors = append(errors, fmt.Sprintf(`intake.mapped_tier "%s" does not match policy tier "%s" for %s`, normalizedTier, policyTier, normalizedLabel))
		}

		rationale := ""
		if r, ok := intake["rationale"].(string); ok {
			rationale = r
		}
		reasoning := []string{"Declared intake block has no rationale."}
		if strings.TrimSpace(rationale) != "" {
			reasoning = []string{rationale}
		}

		classification = IntakeClassification{
			IntakeLabel:               normalizedLabel,
			RuntimeTier:               normalizedTier,
			Reasoning:                 reasoning,
			Signals:                   getStringArray(intake["signals"]),
			NegativeSignalsConsidered: getStringArray(intake["negative_signals_considered"]),
			AutoEscalated:             intake["auto_escalated"] == true,
		}
	} else {
		warnings = append(warnings, "completion card has no intake block; explanation is inferred from claim/evidence")
		classification = ClassifyTask(getCardTask(card), getCardFiles(card), "", policy)
	}

	downgrade := declaredTier != nil && IsTierDowngrade(*declaredTier, classification.RuntimeTier)
	interventionApproved := downgrade && HasApprovedTierDowngradeIntervention(governance)
	if downgrade && !interventionApproved {
		errors = append(errors, fmt.Sprintf("intake tier downgrade requires governance intervention approval: declared %s, mapped %s", *declaredTier, classification.RuntimeTier))
	}

	return IntakeExplanation{
		OK:                        len(errors) == 0,
		Source:                    source,
		DeclaredTier:              declaredTier,
		IntakeLabel:               classification.IntakeLabel,
		MappedTier:                classification.RuntimeTier,
		TierDowngrade:             downgrade,
		InterventionRequired:      downgrade,
		InterventionApproved:      interventionApproved,
		Reasoning:                 classification.Reasoning,
		Signals:                   classification.Signals,
		NegativeSignalsConsidered: classification.NegativeSignalsConsidered,
		Errors:                    errors,
		Warnings:                  warnings,
	}
}
