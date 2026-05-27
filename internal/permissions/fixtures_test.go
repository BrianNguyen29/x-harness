package permissions

import (
	"os"
	"path/filepath"
	"testing"
)

func setupFixturesTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}

	policyContent := `version: 1
command_sets:
  safe_readonly:
    allow:
      - "git status --porcelain"
  safe_tests:
    allow_patterns:
      - "^npm test( .*)?$"
  dangerous:
    deny_patterns:
      - "rm -rf"
roles:
  worker:
    standard:
      allow_capabilities:
        - read_files
        - write_declared_files
        - run_tests
      allow_command_sets:
        - safe_readonly
        - safe_tests
      deny_command_sets:
        - dangerous
    deep:
      allow_capabilities:
        - read_files
        - write_declared_files
        - run_tests
      allow_command_sets:
        - safe_readonly
        - safe_tests
      require_approval:
        - network
        - dependency_install
        - migration
        - destructive_filesystem
      deny_command_sets:
        - dangerous
  verifier:
    all:
      allow_capabilities:
        - read_files
        - read_evidence
        - run_readonly_commands
      allow_command_sets:
        - safe_readonly
        - safe_tests
      deny_capabilities:
        - write_source
        - repair_code
        - release_publish
        - destructive_filesystem
      deny_command_sets:
        - dangerous
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
          "allow": { "type": "array", "items": { "type": "string" } },
          "allow_patterns": { "type": "array", "items": { "type": "string" } },
          "deny": { "type": "array", "items": { "type": "string" } },
          "deny_patterns": { "type": "array", "items": { "type": "string" } }
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
            "allow_capabilities": { "type": "array", "items": { "type": "string" } },
            "deny_capabilities": { "type": "array", "items": { "type": "string" } },
            "require_approval": { "type": "array", "items": { "type": "string" } },
            "allow_command_sets": { "type": "array", "items": { "type": "string" } },
            "deny_command_sets": { "type": "array", "items": { "type": "string" } }
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

	return tmpDir
}

func TestRunFixtures(t *testing.T) {
	tmpDir := setupFixturesTestDir(t)
	policy, err := LoadPolicy(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	result, err := RunFixtures(policy, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("expected all fixtures to pass, got failures: %v", result.Fixtures)
	}
	if len(result.Fixtures) != 5 {
		t.Fatalf("expected 5 fixtures, got: %d", len(result.Fixtures))
	}
}
