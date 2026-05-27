package permissions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	policyContent := `version: 1
command_sets:
  safe:
    allow:
      - "echo ok"
roles:
  worker:
    standard:
      allow_command_sets:
        - safe
`
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "permissions.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["version", "command_sets", "roles"],
  "properties": {
    "version": { "type": "integer", "minimum": 1 },
    "command_sets": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "properties": {
          "allow": { "type": "array", "items": { "type": "string" } }
        },
        "additionalProperties": false
      }
    },
    "roles": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "minProperties": 1,
        "additionalProperties": {
          "type": "object",
          "properties": {
            "allow_command_sets": { "type": "array", "items": { "type": "string" } }
          },
          "additionalProperties": false
        }
      }
    }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "permissions.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	policy, err := LoadPolicy(tmpDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if policy.Version != 1 {
		t.Fatalf("expected version 1, got: %d", policy.Version)
	}
	if len(policy.CommandSets) != 1 {
		t.Fatalf("expected 1 command set, got: %d", len(policy.CommandSets))
	}
	if len(policy.Roles) != 1 {
		t.Fatalf("expected 1 role, got: %d", len(policy.Roles))
	}
}

func TestLoadPolicyMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadPolicy(tmpDir)
	if err == nil {
		t.Fatal("expected error for missing policy file")
	}
}

func TestValidatePolicyUnknownCommandSet(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	policyContent := `version: 1
command_sets:
  safe:
    allow:
      - "echo ok"
roles:
  worker:
    standard:
      allow_command_sets:
        - unknown
`
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "permissions.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["version", "command_sets", "roles"],
  "properties": {
    "version": { "type": "integer", "minimum": 1 },
    "command_sets": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "properties": {
          "allow": { "type": "array", "items": { "type": "string" } }
        },
        "additionalProperties": false
      }
    },
    "roles": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "minProperties": 1,
        "additionalProperties": {
          "type": "object",
          "properties": {
            "allow_command_sets": { "type": "array", "items": { "type": "string" } }
          },
          "additionalProperties": false
        }
      }
    }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "permissions.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPolicy(tmpDir)
	if err == nil {
		t.Fatal("expected error for unknown command set")
	}
	if !strings.Contains(err.Error(), "unknown command set") {
		t.Fatalf("expected unknown command set error, got: %v", err)
	}
}
