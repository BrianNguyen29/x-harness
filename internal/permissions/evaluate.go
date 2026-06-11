package permissions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// PermissionDecision represents the evaluation result
type PermissionDecision struct {
	OK                 bool             `json:"ok"`
	Status             string           `json:"status"`
	Role               string           `json:"role"`
	Tier               string           `json:"tier"`
	Command            *string          `json:"command"`
	Capability         *string          `json:"capability"`
	Reason             string           `json:"reason"`
	Matched            MatchedInfo      `json:"matched"`
	Intervention       InterventionInfo `json:"intervention"`
	AdmissionAuthority bool             `json:"admission_authority"`
}

// MatchedInfo represents a matched rule
type MatchedInfo struct {
	CommandSet *string `json:"command_set"`
	Rule       *string `json:"rule"`
}

// InterventionInfo represents intervention validation result
type InterventionInfo struct {
	Provided bool    `json:"provided"`
	Valid    bool    `json:"valid"`
	Reason   *string `json:"reason"`
	Path     *string `json:"path"`
}

func strPtr(s string) *string {
	return &s
}

func normalizeCommand(command string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(command, " "))
}

func profileFor(policy *PermissionsPolicy, role, tier string) *TierProfile {
	roleTiers, ok := policy.Roles[role]
	if !ok {
		return nil
	}
	if profile, ok := roleTiers[tier]; ok {
		return &profile
	}
	if profile, ok := roleTiers["all"]; ok {
		return &profile
	}
	return nil
}

func listIncludes(items []string, item string) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}
	return false
}

func commandMatches(command string, set CommandSet, mode string) string {
	if mode == "allow" {
		for _, c := range set.Allow {
			if c == command {
				return command
			}
		}
		for _, p := range set.AllowPatterns {
			re, err := regexp.Compile(p)
			if err != nil {
				continue
			}
			if re.MatchString(command) {
				return p
			}
		}
	} else {
		for _, c := range set.Deny {
			if c == command {
				return command
			}
		}
		for _, p := range set.DenyPatterns {
			re, err := regexp.Compile(p)
			if err != nil {
				continue
			}
			if re.MatchString(command) {
				return p
			}
		}
	}
	return ""
}

func shellMetacharacter(command string) string {
	checks := []struct {
		token   string
		pattern *regexp.Regexp
	}{
		{"&&", regexp.MustCompile(`&&`)},
		{"||", regexp.MustCompile(`\|\|`)},
		{";", regexp.MustCompile(`;`)},
		{"|", regexp.MustCompile(`\|`)},
		{"`", regexp.MustCompile("`")},
		{"$(", regexp.MustCompile(`\$\(`)},
		{">", regexp.MustCompile(`>`)},
		{"<", regexp.MustCompile(`<`)},
	}
	for _, check := range checks {
		if check.pattern.MatchString(command) {
			return check.token
		}
	}
	return ""
}

func findCommandMatch(policy *PermissionsPolicy, command string, setNames []string, mode string) *MatchedInfo {
	for _, setName := range setNames {
		set, ok := policy.CommandSets[setName]
		if !ok {
			return &MatchedInfo{
				CommandSet: strPtr(setName),
				Rule:       strPtr("unknown_command_set"),
			}
		}
		rule := commandMatches(command, set, mode)
		if rule != "" {
			return &MatchedInfo{
				CommandSet: strPtr(setName),
				Rule:       strPtr(rule),
			}
		}
	}
	return nil
}

func baseDecision(role, tier, command, capability, status, reason string, matched *MatchedInfo, intervention InterventionInfo) PermissionDecision {
	var cmdPtr, capPtr *string
	if command != "" {
		cmdPtr = strPtr(command)
	}
	if capability != "" {
		capPtr = strPtr(capability)
	}
	if matched == nil {
		matched = &MatchedInfo{}
	}
	return PermissionDecision{
		OK:                 status == "allowed",
		Status:             status,
		Role:               role,
		Tier:               tier,
		Command:            cmdPtr,
		Capability:         capPtr,
		Reason:             reason,
		Matched:            *matched,
		Intervention:       intervention,
		AdmissionAuthority: false,
	}
}

func defaultIntervention(interventionPath, root string) InterventionInfo {
	if interventionPath != "" {
		absPath := interventionPath
		if !filepath.IsAbs(interventionPath) {
			absPath = filepath.Join(root, interventionPath)
		}
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr("not evaluated"),
			Path:     strPtr(absPath),
		}
	}
	return InterventionInfo{
		Provided: false,
		Valid:    false,
		Reason:   nil,
		Path:     nil,
	}
}

func interventionTarget(capability, command string) string {
	if capability != "" {
		return "capability:" + capability
	}
	if command != "" {
		return "command:" + normalizeCommand(command)
	}
	return "permissions"
}

func interventionCovers(artifact map[string]interface{}, capability, command string) bool {
	scope, _ := artifact["scope"].(string)
	if scope == "global" {
		return true
	}
	target := interventionTarget(capability, command)
	pathsRaw, _ := artifact["paths"].([]interface{})
	for _, p := range pathsRaw {
		if entry, ok := p.(string); ok {
			if entry == target || entry == "permissions" || entry == "permissions/**" || entry == "policies/permissions.yaml" {
				return true
			}
		}
	}
	return false
}

// ValidateIntervention validates an intervention artifact
func ValidateIntervention(root, pathStr, capability, command string) InterventionInfo {
	if pathStr == "" {
		return InterventionInfo{
			Provided: false,
			Valid:    false,
			Reason:   strPtr("intervention required"),
			Path:     nil,
		}
	}

	interventionPath := pathStr
	if !filepath.IsAbs(pathStr) {
		interventionPath = filepath.Join(root, pathStr)
	}
	resolvedPath, err := filepath.Abs(interventionPath)
	if err != nil {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr(fmt.Sprintf("failed to resolve intervention path: %v", err)),
			Path:     strPtr(interventionPath),
		}
	}
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr(fmt.Sprintf("failed to resolve workspace root: %v", err)),
			Path:     strPtr(resolvedPath),
		}
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr("intervention path escapes workspace root"),
			Path:     strPtr(resolvedPath),
		}
	}
	interventionPath = resolvedPath

	if _, err := os.Stat(interventionPath); err != nil {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr("intervention file not found"),
			Path:     strPtr(interventionPath),
		}
	}

	var artifact map[string]interface{}
	if err := loader.LoadDocument(interventionPath, &artifact); err != nil {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr(fmt.Sprintf("failed to load intervention: %v", err)),
			Path:     strPtr(interventionPath),
		}
	}

	schemaPath := filepath.Join(root, "schemas", "intervention.schema.json")
	if _, err := os.Stat(schemaPath); err != nil {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr("intervention schema not found"),
			Path:     strPtr(interventionPath),
		}
	}
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr(fmt.Sprintf("failed to compile intervention schema: %v", err)),
			Path:     strPtr(interventionPath),
		}
	}
	if err := validator.Validate(artifact); err != nil {
		if verr, ok := err.(*jsonschema.ValidationError); ok {
			return InterventionInfo{
				Provided: true,
				Valid:    false,
				Reason:   strPtr(verr.Error()),
				Path:     strPtr(interventionPath),
			}
		}
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr(err.Error()),
			Path:     strPtr(interventionPath),
		}
	}

	authorizer, _ := artifact["authorizer"].(string)
	if strings.TrimSpace(authorizer) == "" {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr("intervention authorizer is required"),
			Path:     strPtr(interventionPath),
		}
	}

	decision, _ := artifact["decision"].(string)
	if decision != "allow" && decision != "override" {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr("intervention decision must be allow or override"),
			Path:     strPtr(interventionPath),
		}
	}

	expirationStr, _ := artifact["expiration"].(string)
	expiration, err := time.Parse(time.RFC3339, expirationStr)
	if err != nil {
		expiration, err = time.Parse("2006-01-02T15:04:05Z", expirationStr)
	}
	if err != nil || !expiration.After(time.Now()) {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr("intervention is expired"),
			Path:     strPtr(interventionPath),
		}
	}

	if !interventionCovers(artifact, capability, command) {
		return InterventionInfo{
			Provided: true,
			Valid:    false,
			Reason:   strPtr(fmt.Sprintf("intervention scope does not cover %s", interventionTarget(capability, command))),
			Path:     strPtr(interventionPath),
		}
	}

	return InterventionInfo{
		Provided: true,
		Valid:    true,
		Reason:   strPtr("valid intervention exception"),
		Path:     strPtr(interventionPath),
	}
}

// CheckPermission evaluates a command or capability for a role/tier
func CheckPermission(policy *PermissionsPolicy, root, role, tier, command, capability, interventionPath string) (PermissionDecision, error) {
	if command == "" && capability == "" {
		return baseDecision(role, tier, command, capability, "denied", "command or capability is required", nil, defaultIntervention(interventionPath, root)), nil
	}
	if command != "" && capability != "" {
		return baseDecision(role, tier, command, capability, "denied", "provide only one of command or capability", nil, defaultIntervention(interventionPath, root)), nil
	}

	if tier == "" {
		tier = "standard"
	}

	profile := profileFor(policy, role, tier)
	if profile == nil {
		return baseDecision(role, tier, command, capability, "denied", fmt.Sprintf("no permissions profile for role %s tier %s", role, tier), nil, defaultIntervention(interventionPath, root)), nil
	}

	if command != "" {
		cmd := normalizeCommand(command)

		denied := findCommandMatch(policy, cmd, profile.DenyCommandSets, "deny")
		if denied != nil {
			return baseDecision(role, tier, cmd, "", "denied", fmt.Sprintf("command denied by %s", *denied.CommandSet), denied, defaultIntervention(interventionPath, root)), nil
		}

		meta := shellMetacharacter(cmd)
		if meta != "" {
			return baseDecision(role, tier, cmd, "", "denied", fmt.Sprintf("command contains shell metacharacter %s", meta), &MatchedInfo{
				CommandSet: strPtr("shell_metacharacter"),
				Rule:       strPtr(meta),
			}, defaultIntervention(interventionPath, root)), nil
		}

		allowed := findCommandMatch(policy, cmd, profile.AllowCommandSets, "allow")
		if allowed != nil {
			return baseDecision(role, tier, cmd, "", "allowed", fmt.Sprintf("command allowed by %s", *allowed.CommandSet), allowed, defaultIntervention(interventionPath, root)), nil
		}

		return baseDecision(role, tier, cmd, "", "denied", fmt.Sprintf("command is not allowlisted for role %s tier %s", role, tier), nil, defaultIntervention(interventionPath, root)), nil
	}

	cap := capability
	if listIncludes(profile.DenyCapabilities, cap) {
		return baseDecision(role, tier, "", cap, "denied", fmt.Sprintf("capability %s is denied for role %s", cap, role), nil, defaultIntervention(interventionPath, root)), nil
	}

	if listIncludes(profile.RequireApproval, cap) {
		intervention := ValidateIntervention(root, interventionPath, cap, command)
		if intervention.Valid {
			return baseDecision(role, tier, "", cap, "allowed", fmt.Sprintf("capability %s allowed by valid intervention", cap), &MatchedInfo{
				CommandSet: nil,
				Rule:       strPtr("intervention"),
			}, intervention), nil
		}
		return baseDecision(role, tier, "", cap, "requires_intervention", fmt.Sprintf("capability %s requires valid intervention", cap), &MatchedInfo{
			CommandSet: nil,
			Rule:       strPtr("require_approval"),
		}, intervention), nil
	}

	if listIncludes(profile.AllowCapabilities, cap) {
		return baseDecision(role, tier, "", cap, "allowed", fmt.Sprintf("capability %s allowed for role %s", cap, role), nil, defaultIntervention(interventionPath, root)), nil
	}

	return baseDecision(role, tier, "", cap, "denied", fmt.Sprintf("capability %s is not allowed for role %s tier %s", cap, role, tier), nil, defaultIntervention(interventionPath, root)), nil
}
