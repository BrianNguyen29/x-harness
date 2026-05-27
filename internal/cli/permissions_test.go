package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupPermissionsTestDir(t *testing.T) string {
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
      - "git diff --name-only"
      - "git diff --check"
  safe_tests:
    allow_patterns:
      - "^npm test( .*)?$"
      - "^pnpm test( .*)?$"
      - "^npm run test( .*)?$"
      - "^npm run typecheck$"
      - "^tsc --noEmit$"
      - "^pytest( .*)?$"
      - "^go test ./\\.\\.\\.$"
      - "^cargo test( .*)?$"
  dangerous:
    deny_patterns:
      - "rm -rf"
      - "curl .*\\|.*bash"
      - "wget .*\\|.*sh"
      - "npm publish"
      - "pnpm publish"
      - "kubectl apply"
      - "terraform apply"
      - "aws .*"
      - "gcloud .*"
      - "az .*"
roles:
  worker:
    light:
      allow_capabilities:
        - read_files
        - write_declared_files
      allow_command_sets:
        - safe_readonly
        - safe_tests
      deny_command_sets:
        - dangerous
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
  maintainer:
    all:
      allow_capabilities:
        - approve_intervention
        - change_policy
        - publish_release
`
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "permissions.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "permissions-policy",
  "type": "object",
  "required": ["version", "command_sets", "roles"],
  "properties": {
    "version": {
      "type": "integer",
      "minimum": 1
    },
    "command_sets": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "$ref": "#/$defs/command_set"
      }
    },
    "roles": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "minProperties": 1,
        "additionalProperties": {
          "$ref": "#/$defs/profile"
        }
      }
    }
  },
  "$defs": {
    "command_set": {
      "type": "object",
      "properties": {
        "allow": {
          "type": "array",
          "items": {
            "type": "string",
            "minLength": 1
          }
        },
        "allow_patterns": {
          "type": "array",
          "items": {
            "type": "string",
            "minLength": 1
          }
        },
        "deny": {
          "type": "array",
          "items": {
            "type": "string",
            "minLength": 1
          }
        },
        "deny_patterns": {
          "type": "array",
          "items": {
            "type": "string",
            "minLength": 1
          }
        }
      },
      "additionalProperties": false
    },
    "profile": {
      "type": "object",
      "properties": {
        "allow_capabilities": {
          "$ref": "#/$defs/string_list"
        },
        "deny_capabilities": {
          "$ref": "#/$defs/string_list"
        },
        "require_approval": {
          "$ref": "#/$defs/string_list"
        },
        "allow_command_sets": {
          "$ref": "#/$defs/string_list"
        },
        "deny_command_sets": {
          "$ref": "#/$defs/string_list"
        }
      },
      "additionalProperties": false
    },
    "string_list": {
      "type": "array",
      "items": {
        "type": "string",
        "minLength": 1
      }
    }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "permissions.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	interventionSchema := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "intervention",
  "type": "object",
  "required": ["actor", "task", "scope", "decision", "reason", "expiration"],
  "properties": {
    "actor": {
      "type": "string"
    },
    "task": {
      "type": "string"
    },
    "scope": {
      "type": "string",
      "enum": ["file", "directory", "path", "global"]
    },
    "paths": {
      "type": "array",
      "items": { "type": "string" }
    },
    "decision": {
      "type": "string",
      "enum": ["allow", "deny", "flag", "override"]
    },
    "reason": {
      "type": "string"
    },
    "expiration": {
      "type": "string",
      "format": "date-time"
    },
    "authorizer": {
      "type": "string"
    },
    "created_at": {
      "type": "string",
      "format": "date-time"
    }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "intervention.schema.json"), []byte(interventionSchema), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func TestPermissionsCheckAllowed(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "check", "--role", "worker", "--command", "npm test", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "allowed") {
		t.Fatalf("expected allowed status, got: %s", out)
	}
}

func TestPermissionsCheckDenied(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "check", "--role", "worker", "--command", "rm -rf /", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "denied") {
		t.Fatalf("expected denied status, got: %s", out)
	}
}

func TestPermissionsExplainDenied(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "explain", "--role", "worker", "--command", "rm -rf /", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "denied") {
		t.Fatalf("expected denied status, got: %s", out)
	}
}

func TestPermissionsCheckMissingRole(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "check", "--command", "npm test", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--role is required") {
		t.Fatalf("expected missing role error, got: %s", stderr.String())
	}
}

func TestPermissionsCheckBothCommandAndCapability(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "check", "--role", "worker", "--command", "npm test", "--capability", "read_files", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "provide only one of --command or --capability") {
		t.Fatalf("expected mutual exclusion error, got: %s", stderr.String())
	}
}

func TestPermissionsCheckJSONOutput(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "check", "--role", "worker", "--command", "npm test", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["status"] != "allowed" {
		t.Fatalf("expected status=allowed, got: %v", result)
	}
}

func TestPermissionsTestFixtures(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "test-fixtures", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "x-harness Permission Fixtures") {
		t.Fatalf("expected fixtures header, got: %s", out)
	}
}

func TestPermissionsTestFixturesJSON(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "test-fixtures", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if fixtures, ok := result["fixtures"].([]interface{}); !ok || len(fixtures) != 5 {
		t.Fatalf("expected 5 fixtures, got: %v", result)
	}
}

func TestPermissionsMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestPermissionsUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown permissions subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestPermissionsUnknownFlag(t *testing.T) {
	tmpDir := setupPermissionsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"permissions", "check", "--role", "worker", "--command", "npm test", "--root", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
