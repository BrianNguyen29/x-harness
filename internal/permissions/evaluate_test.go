package permissions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func createTestPolicy(t *testing.T) *PermissionsPolicy {
	return &PermissionsPolicy{
		Version: 1,
		CommandSets: map[string]CommandSet{
			"safe": {
				Allow: []string{"echo ok"},
			},
			"dangerous": {
				DenyPatterns: []string{"rm -rf"},
			},
		},
		Roles: map[string]map[string]TierProfile{
			"worker": {
				"standard": {
					AllowCommandSets:  []string{"safe"},
					DenyCommandSets:   []string{"dangerous"},
					AllowCapabilities: []string{"read"},
				},
				"deep": {
					AllowCommandSets:  []string{"safe"},
					DenyCommandSets:   []string{"dangerous"},
					AllowCapabilities: []string{"read"},
					RequireApproval:   []string{"network"},
				},
			},
		},
	}
}

func TestCheckPermissionCommandAllowed(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "standard", "echo ok", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !decision.OK {
		t.Fatalf("expected allowed, got: %v", decision)
	}
	if decision.Status != "allowed" {
		t.Fatalf("expected status allowed, got: %s", decision.Status)
	}
}

func TestCheckPermissionCommandDenied(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "standard", "rm -rf /", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.OK {
		t.Fatalf("expected denied, got: %v", decision)
	}
	if decision.Status != "denied" {
		t.Fatalf("expected status denied, got: %s", decision.Status)
	}
}

func TestCheckPermissionShellMetacharacter(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "standard", "echo ok && echo hi", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.OK {
		t.Fatalf("expected denied, got: %v", decision)
	}
	if !strings.Contains(decision.Reason, "shell metacharacter") {
		t.Fatalf("expected shell metacharacter reason, got: %s", decision.Reason)
	}
}

func TestCheckPermissionCapabilityAllowed(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "standard", "", "read", "")
	if err != nil {
		t.Fatal(err)
	}
	if !decision.OK {
		t.Fatalf("expected allowed, got: %v", decision)
	}
}

func TestCheckPermissionCapabilityDenied(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "standard", "", "write", "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.OK {
		t.Fatalf("expected denied, got: %v", decision)
	}
}

func TestCheckPermissionRequiresIntervention(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "deep", "", "network", "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.OK {
		t.Fatalf("expected requires_intervention, got: %v", decision)
	}
	if decision.Status != "requires_intervention" {
		t.Fatalf("expected status requires_intervention, got: %s", decision.Status)
	}
}

func TestCheckPermissionMissingRole(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "unknown", "standard", "echo ok", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.OK {
		t.Fatalf("expected denied for unknown role, got: %v", decision)
	}
}

func TestCheckPermissionMissingInput(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "standard", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.OK {
		t.Fatalf("expected denied for missing input, got: %v", decision)
	}
}

func TestCheckPermissionBothInputs(t *testing.T) {
	policy := createTestPolicy(t)
	decision, err := CheckPermission(policy, ".", "worker", "standard", "echo ok", "read", "")
	if err != nil {
		t.Fatal(err)
	}
	if decision.OK {
		t.Fatalf("expected denied for both inputs, got: %v", decision)
	}
}

func TestValidateInterventionMissingFile(t *testing.T) {
	info := ValidateIntervention(".", "nonexistent.yaml", "read", "")
	if info.Valid {
		t.Fatal("expected invalid for missing file")
	}
	if info.Reason == nil || *info.Reason != "intervention file not found" {
		t.Fatalf("expected file not found reason, got: %v", info.Reason)
	}
}

func TestValidateInterventionExpired(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["actor", "task", "scope", "decision", "reason", "expiration"],
  "properties": {
    "actor": { "type": "string" },
    "task": { "type": "string" },
    "scope": { "type": "string", "enum": ["file", "directory", "path", "global"] },
    "paths": { "type": "array", "items": { "type": "string" } },
    "decision": { "type": "string", "enum": ["allow", "deny", "flag", "override"] },
    "reason": { "type": "string" },
    "expiration": { "type": "string", "format": "date-time" },
    "authorizer": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "intervention.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	intervention := `actor: test
task: test
scope: global
decision: allow
reason: test
expiration: "2020-01-01T00:00:00Z"
`
	path := filepath.Join(tmpDir, "intervention.yaml")
	if err := os.WriteFile(path, []byte(intervention), 0644); err != nil {
		t.Fatal(err)
	}

	info := ValidateIntervention(tmpDir, path, "read", "")
	if info.Valid {
		t.Fatal("expected invalid for expired intervention")
	}
	if info.Reason == nil || *info.Reason != "intervention is expired" {
		t.Fatalf("expected expired reason, got: %v", info.Reason)
	}
}

func TestValidateInterventionValid(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["actor", "task", "scope", "decision", "reason", "expiration"],
  "properties": {
    "actor": { "type": "string" },
    "task": { "type": "string" },
    "scope": { "type": "string", "enum": ["file", "directory", "path", "global"] },
    "paths": { "type": "array", "items": { "type": "string" } },
    "decision": { "type": "string", "enum": ["allow", "deny", "flag", "override"] },
    "reason": { "type": "string" },
    "expiration": { "type": "string", "format": "date-time" },
    "authorizer": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "intervention.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	intervention := fmt.Sprintf(`actor: test
task: test
scope: global
decision: allow
reason: test
expiration: "%s"
`, future)
	path := filepath.Join(tmpDir, "intervention.yaml")
	if err := os.WriteFile(path, []byte(intervention), 0644); err != nil {
		t.Fatal(err)
	}

	info := ValidateIntervention(tmpDir, path, "read", "")
	if !info.Valid {
		t.Fatalf("expected valid intervention, got: %v", info.Reason)
	}
}
