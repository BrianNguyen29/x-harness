package approvalrisk

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/BrianNguyen29/x-harness/internal/authority"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

// ApprovalRiskPolicy represents the approval-risk.yaml structure.
type ApprovalRiskPolicy struct {
	Version      int `yaml:"version"`
	ApprovalRisk struct {
		Enabled           bool           `yaml:"enabled"`
		PersonalScoring   bool           `yaml:"personal_scoring"`
		Thresholds        map[string]int `yaml:"thresholds"`
		RequiredApprovals map[string]int `yaml:"required_approvals"`
		Signals           map[string]int `yaml:"signals"`
	} `yaml:"approval_risk"`
}

// ApprovalRiskReport represents the evaluation result.
type ApprovalRiskReport struct {
	SchemaVersion      string   `json:"schema_version"`
	TaskID             string   `json:"task_id"`
	Tier               string   `json:"tier,omitempty"`
	RiskClass          string   `json:"risk_class"`
	Score              int      `json:"score"`
	Signals            []string `json:"signals"`
	RequiredApprovals  int      `json:"required_approvals"`
	PersonalScoring    bool     `json:"personal_scoring"`
	PolicyEnabled      bool     `json:"policy_enabled"`
	AdmissionAuthority bool     `json:"admission_authority"`
}

// LoadApprovalRiskPolicy loads the approval risk policy from the given repository root.
func LoadApprovalRiskPolicy(root string) (*ApprovalRiskPolicy, error) {
	path := filepath.Join(root, "policies", "approval-risk.yaml")
	var policy ApprovalRiskPolicy
	if err := loader.LoadYAML(path, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func evidenceFiles(card map[string]interface{}) []string {
	evidence, ok := card["evidence"].(map[string]interface{})
	if !ok {
		return nil
	}
	files, ok := evidence["files_changed"].([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, f := range files {
		if s, ok := f.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func riskClass(score int, thresholds map[string]int) string {
	if score >= thresholds["critical"] {
		return "critical"
	}
	if score >= thresholds["elevated"] {
		return "elevated"
	}
	if score >= thresholds["moderate"] {
		return "moderate"
	}
	return "low"
}

func validateReport(report *ApprovalRiskReport, root string) error {
	schemaPath := filepath.Join(root, "schemas", "approval-risk.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		return err
	}
	// Convert struct to generic map for schema validation
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	return v.Validate(doc)
}

var securityPattern = regexp.MustCompile(`(?i)(auth|token|secret|session|permission|policy)`)

// EvaluateApprovalRisk evaluates the approval risk for a completion card.
func EvaluateApprovalRisk(cardPath, root string) (*ApprovalRiskReport, error) {
	var card map[string]interface{}
	if err := loader.LoadDocument(cardPath, &card); err != nil {
		return nil, fmt.Errorf("failed to load completion card: %w", err)
	}

	policy, err := LoadApprovalRiskPolicy(root)
	if err != nil {
		return nil, fmt.Errorf("failed to load approval risk policy: %w", err)
	}

	authPolicy, err := authority.LoadAuthorityPolicy(root)
	if err != nil {
		return nil, fmt.Errorf("failed to load authority policy: %w", err)
	}

	signals := make(map[string]bool)
	score := 0

	tier := ""
	if t, ok := card["tier"].(string); ok {
		tier = t
	}

	if tier == "deep" {
		signals["deep_tier"] = true
		score += policy.ApprovalRisk.Signals["deep_tier"]
	}

	for _, file := range evidenceFiles(card) {
		classified := authority.ClassifyPath(authPolicy, file)
		if classified == "human_only" {
			signals["human_only_path"] = true
			score += policy.ApprovalRisk.Signals["human_only_path"]
		}
		if classified == "agent_proposable_human_approved" {
			signals["human_approved_path"] = true
			score += policy.ApprovalRisk.Signals["human_approved_path"]
		}
		if securityPattern.MatchString(file) {
			signals["security_sensitive_path"] = true
			score += policy.ApprovalRisk.Signals["security_sensitive_path"]
		}
	}

	governance, _ := card["governance"].(map[string]interface{})

	if signals["human_only_path"] {
		approvalStatus := ""
		if governance != nil {
			if s, ok := governance["approval_status"].(string); ok {
				approvalStatus = s
			}
		}
		if approvalStatus != "approved" {
			signals["missing_governance_approval"] = true
			score += policy.ApprovalRisk.Signals["missing_governance_approval"]
		}
	}

	classified := riskClass(score, policy.ApprovalRisk.Thresholds)
	requiredApprovals := policy.ApprovalRisk.RequiredApprovals[classified]

	signalList := []string{}
	for s := range signals {
		signalList = append(signalList, s)
	}
	sort.Strings(signalList)

	taskID := ""
	if t, ok := card["task_id"].(string); ok {
		taskID = t
	}
	if taskID == "" {
		taskID = "unknown"
	}

	report := &ApprovalRiskReport{
		SchemaVersion:      "1",
		TaskID:             taskID,
		Tier:               tier,
		RiskClass:          classified,
		Score:              score,
		Signals:            signalList,
		RequiredApprovals:  requiredApprovals,
		PersonalScoring:    false,
		PolicyEnabled:      policy.ApprovalRisk.Enabled,
		AdmissionAuthority: false,
	}

	if err := validateReport(report, root); err != nil {
		return nil, fmt.Errorf("approval risk report validation failed: %w", err)
	}

	return report, nil
}
