package authority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GovernanceWarning represents a single governance finding.
type GovernanceWarning struct {
	Path             string `json:"path"`
	Authority        string `json:"authority"`
	Rationale        string `json:"rationale"`
	Severity         string `json:"severity"`
	ApprovalRequired bool   `json:"approval_required"`
	ApprovalVerified bool   `json:"approval_verified"`
	ApprovalNote     string `json:"approval_note,omitempty"`
}

// GovernanceCheckResult holds the outcome of a governance check.
type GovernanceCheckResult struct {
	Violations      []GovernanceWarning `json:"violations"`
	Warnings        []GovernanceWarning `json:"warnings"`
	ReportOnly      bool                `json:"report_only"`
	Enforced        bool                `json:"enforced"`
	TotalViolations int                 `json:"total_violations"`
	TotalWarnings   int                 `json:"total_warnings"`
}

// GovernanceCheckOptions configures the governance check.
type GovernanceCheckOptions struct {
	Enforce    bool
	Governance map[string]any
}

// ExplainPathResult holds the result of explaining a path.
type ExplainPathResult struct {
	Path      string `json:"path"`
	Authority string `json:"authority"`
	Rationale string `json:"rationale"`
}

// ClassifyPathWithRationale returns authority and rationale for a file path.
func ClassifyPathWithRationale(policy *AuthorityPolicy, filePath string) (string, string) {
	normalizedPath := normalizePath(filePath)
	for _, protectedPath := range policy.ProtectedPaths {
		if matchPath(protectedPath.Path, normalizedPath) {
			return protectedPath.Authority, protectedPath.Rationale
		}
	}
	return "agent_editable", "Default: no protected path match"
}

// ExplainPath returns the authority classification for a path.
func ExplainPath(policy *AuthorityPolicy, filePath, root string) (*ExplainPathResult, error) {
	normalizedPath := normalizePath(filePath)

	if filepath.IsAbs(normalizedPath) {
		rel, err := filepath.Rel(root, normalizedPath)
		if err != nil {
			return nil, err
		}
		normalizedPath = normalizePath(rel)
	}

	auth, rationale := ClassifyPathWithRationale(policy, normalizedPath)
	return &ExplainPathResult{
		Path:      normalizedPath,
		Authority: auth,
		Rationale: rationale,
	}, nil
}

// GetProtectedPaths returns all protected paths from the policy.
func GetProtectedPaths(policy *AuthorityPolicy) []PathRule {
	return policy.ProtectedPaths
}

func isEnforced(policy *AuthorityPolicy, options *GovernanceCheckOptions) bool {
	if options != nil && options.Enforce {
		return true
	}
	return false
}

func normalizedHash(value string) string {
	s := strings.TrimSpace(value)
	s = strings.ToLower(s)
	s = strings.TrimPrefix(s, "sha256:")
	return s
}

func sha256File(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func scopePathsFromRecord(record map[string]any) []string {
	scope, _ := record["scope"].(map[string]any)
	if scope == nil {
		return nil
	}
	paths, _ := scope["paths"].([]any)
	var result []string
	for _, p := range paths {
		if s, ok := p.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func scopeCoversPath(scopePaths []string, protectedPath string) bool {
	for _, sp := range scopePaths {
		if matchPath(sp, protectedPath) {
			return true
		}
	}
	return false
}

func verifyApprovalForPath(governance map[string]any, protectedPath, root string) (bool, string) {
	if governance == nil {
		return false, "governance approval_status is not approved"
	}

	approvalStatus, _ := governance["approval_status"].(string)
	if approvalStatus != "approved" {
		return false, "governance approval_status is not approved"
	}

	artifact, _ := governance["approval_artifact"].(map[string]any)
	if artifact == nil {
		return false, "governance approval_artifact is missing"
	}

	artifactPath, _ := artifact["path"].(string)
	expectedHash, _ := artifact["sha256"].(string)

	if strings.TrimSpace(artifactPath) == "" {
		return false, "governance approval_artifact.path is missing"
	}
	if strings.TrimSpace(expectedHash) == "" {
		return false, "governance approval_artifact.sha256 is missing"
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false, "governance approval_artifact.path is outside repository root"
	}
	resolvedArtifactPath := filepath.Join(root, artifactPath)
	artifactAbs, err := filepath.Abs(resolvedArtifactPath)
	if err != nil {
		return false, "governance approval_artifact.path is outside repository root"
	}

	rootPrefix := rootAbs + string(filepath.Separator)
	if !strings.HasPrefix(artifactAbs, rootPrefix) && artifactAbs != rootAbs {
		return false, "governance approval_artifact.path is outside repository root"
	}

	if _, err := os.Stat(resolvedArtifactPath); os.IsNotExist(err) {
		return false, "governance approval_artifact.path not found"
	}

	artifactRel := normalizePath(strings.TrimPrefix(artifactAbs, rootPrefix))
	if !strings.HasPrefix(artifactRel, ".x-harness/approvals/") {
		return false, "governance approval_artifact.path must be under .x-harness/approvals"
	}

	actualHash := sha256File(resolvedArtifactPath)
	if normalizedHash(expectedHash) != actualHash {
		return false, "governance approval_artifact.sha256 mismatch"
	}

	// Check registry
	registryPath := filepath.Join(root, ".x-harness", "approvals", "registry.json")
	registryData, err := os.ReadFile(registryPath)
	if err != nil {
		return false, "approval artifact is not registered in .x-harness/approvals/registry.json"
	}

	var registry struct {
		Approvals []map[string]any `json:"approvals"`
	}
	if err := json.Unmarshal(registryData, &registry); err != nil {
		return false, "approval artifact is not registered in .x-harness/approvals/registry.json"
	}

	var registeredApproval map[string]any
	for _, entry := range registry.Approvals {
		entryPath, _ := entry["path"].(string)
		entrySHA, _ := entry["sha256"].(string)
		if entryPath == artifactRel && normalizedHash(entrySHA) == actualHash {
			registeredApproval = entry
			break
		}
	}

	if registeredApproval == nil {
		return false, "approval artifact is not registered in .x-harness/approvals/registry.json"
	}

	if status, _ := registeredApproval["status"].(string); status != "approved" {
		return false, "registered approval status is not approved"
	}

	// Load and verify approval artifact content
	approvalData, err := os.ReadFile(resolvedArtifactPath)
	if err != nil {
		return false, "approval artifact decision is not approved"
	}

	var approvalRecord map[string]any
	if err := json.Unmarshal(approvalData, &approvalRecord); err != nil {
		if err := yaml.Unmarshal(approvalData, &approvalRecord); err != nil {
			return false, "approval artifact decision is not approved"
		}
	}

	if decision, _ := approvalRecord["decision"].(string); decision != "approved" {
		return false, "approval artifact decision is not approved"
	}

	approvedBy, _ := approvalRecord["approved_by"].(string)
	if strings.TrimSpace(approvedBy) == "" {
		return false, "approval artifact approved_by is missing"
	}

	registeredApprovedBy, _ := registeredApproval["approved_by"].(string)
	if strings.TrimSpace(registeredApprovedBy) != "" {
		if registeredApprovedBy != approvedBy {
			return false, "registered approval approver does not match approval artifact"
		}
	}

	if approvedAt, _ := approvalRecord["approved_at"].(string); strings.TrimSpace(approvedAt) == "" {
		return false, "approval artifact approved_at is missing"
	}

	// Check scope
	scopePaths := scopePathsFromRecord(approvalRecord)
	if !scopeCoversPath(scopePaths, protectedPath) {
		return false, "approval artifact scope does not cover protected path"
	}

	registeredScopePaths := scopePathsFromRecord(registeredApproval)
	if len(registeredScopePaths) > 0 && !scopeCoversPath(registeredScopePaths, protectedPath) {
		return false, "registered approval scope does not cover protected path"
	}

	return true, "approval artifact verified"
}

// CheckGovernance checks files against the authority policy.
func CheckGovernance(files []string, root string, policy *AuthorityPolicy, options *GovernanceCheckOptions) (*GovernanceCheckResult, error) {
	enforced := isEnforced(policy, options)
	var warnings []GovernanceWarning
	var violations []GovernanceWarning

	for _, file := range files {
		normalizedPath := normalizePath(file)
		if filepath.IsAbs(normalizedPath) {
			rel, err := filepath.Rel(root, normalizedPath)
			if err != nil {
				continue
			}
			normalizedPath = normalizePath(rel)
		}

		authority, rationale := ClassifyPathWithRationale(policy, normalizedPath)

		if authority == "human_only" || authority == "agent_proposable_human_approved" {
			var approvalOK bool
			var approvalNote string

			if enforced {
				approvalOK, approvalNote = verifyApprovalForPath(options.Governance, normalizedPath, root)
			} else {
				approvalNote = "report-only mode"
			}

			severity := "warning"
			if enforced && !approvalOK {
				severity = "violation"
			}

			finding := GovernanceWarning{
				Path:             normalizedPath,
				Authority:        authority,
				Rationale:        rationale,
				Severity:         severity,
				ApprovalRequired: true,
				ApprovalVerified: approvalOK,
				ApprovalNote:     approvalNote,
			}

			if enforced && !approvalOK {
				violations = append(violations, finding)
			} else if !approvalOK {
				warnings = append(warnings, finding)
			}
		}
	}

	return &GovernanceCheckResult{
		Violations:      violations,
		Warnings:        warnings,
		ReportOnly:      !enforced,
		Enforced:        enforced,
		TotalViolations: len(violations),
		TotalWarnings:   len(warnings),
	}, nil
}

// LoadCardGovernanceData extracts files_changed and governance from a completion card.
func LoadCardGovernanceData(cardPath string) ([]string, map[string]any, error) {
	var doc map[string]any
	data, err := os.ReadFile(cardPath)
	if err != nil {
		return nil, nil, err
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, nil, fmt.Errorf("failed to parse card: %w", err)
		}
	}

	var files []string
	if evidence, ok := doc["evidence"].(map[string]any); ok {
		if fc, ok := evidence["files_changed"].([]any); ok {
			for _, item := range fc {
				if s, ok := item.(string); ok {
					files = append(files, s)
				}
			}
		}
	}

	var governance map[string]any
	if g, ok := doc["governance"].(map[string]any); ok {
		governance = g
	}

	return files, governance, nil
}
