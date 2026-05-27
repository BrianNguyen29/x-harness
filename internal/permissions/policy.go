package permissions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// PermissionsPolicy represents the policy structure
type PermissionsPolicy struct {
	Version     int                               `yaml:"version" json:"version"`
	CommandSets map[string]CommandSet             `yaml:"command_sets" json:"command_sets"`
	Roles       map[string]map[string]TierProfile `yaml:"roles" json:"roles"`
}

// CommandSet represents a set of command rules
type CommandSet struct {
	Allow         []string `yaml:"allow" json:"allow"`
	AllowPatterns []string `yaml:"allow_patterns" json:"allow_patterns"`
	Deny          []string `yaml:"deny" json:"deny"`
	DenyPatterns  []string `yaml:"deny_patterns" json:"deny_patterns"`
}

// TierProfile represents permissions for a specific role/tier
type TierProfile struct {
	AllowCommandSets  []string `yaml:"allow_command_sets" json:"allow_command_sets"`
	DenyCommandSets   []string `yaml:"deny_command_sets" json:"deny_command_sets"`
	AllowCapabilities []string `yaml:"allow_capabilities" json:"allow_capabilities"`
	DenyCapabilities  []string `yaml:"deny_capabilities" json:"deny_capabilities"`
	RequireApproval   []string `yaml:"require_approval" json:"require_approval"`
}

// LoadPolicy loads and parses policies/permissions.yaml
func LoadPolicy(root string) (*PermissionsPolicy, error) {
	policyPath := filepath.Join(root, "policies", "permissions.yaml")
	var policy PermissionsPolicy
	if err := loader.LoadYAML(policyPath, &policy); err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}
	if err := ValidatePolicy(root, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// ValidatePolicy validates policy against schema and cross-references
func ValidatePolicy(root string, policy *PermissionsPolicy) error {
	schemaPath := filepath.Join(root, "schemas", "permissions.schema.json")
	if _, err := os.Stat(schemaPath); err != nil {
		return fmt.Errorf("permissions schema not found: %w", err)
	}
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to compile permissions schema: %w", err)
	}

	// Load as map for schema validation to avoid tag issues
	var doc map[string]interface{}
	if err := loader.LoadYAML(filepath.Join(root, "policies", "permissions.yaml"), &doc); err != nil {
		return fmt.Errorf("failed to load policy for validation: %w", err)
	}
	if err := validator.Validate(doc); err != nil {
		if verr, ok := err.(*jsonschema.ValidationError); ok {
			return fmt.Errorf("permissions policy schema validation failed: %s", verr.Error())
		}
		return fmt.Errorf("permissions policy schema validation failed: %w", err)
	}

	// Cross-reference check
	for role, tiers := range policy.Roles {
		for tier, profile := range tiers {
			for _, setName := range append(profile.AllowCommandSets, profile.DenyCommandSets...) {
				if _, ok := policy.CommandSets[setName]; !ok {
					return fmt.Errorf("%s.%s references unknown command set %s", role, tier, setName)
				}
			}
		}
	}
	return nil
}
