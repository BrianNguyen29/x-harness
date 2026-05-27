package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createComponentsRegistry(t *testing.T, dir string) {
	t.Helper()
	registryContent := `version: 1
components:
  - id: admission_policy
    kind: policy
    paths:
      - policies/admission.yaml
    owner: maintainers
    stability: stable
    agent_edit: human_only
    tests:
      - go test
  - id: examples_and_golden
    kind: fixture
    paths:
      - examples/**
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - echo ok
`
	if err := os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(registryContent), 0644); err != nil {
		t.Fatal(err)
	}
}

func createComponentsSchema(t *testing.T, dir string) {
	t.Helper()
	schemaContent := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "components-registry",
  "type": "object",
  "required": ["version", "components"],
  "properties": {
    "version": { "type": "integer", "minimum": 1 },
    "components": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["id", "kind", "paths", "owner", "stability", "agent_edit", "tests"],
        "properties": {
          "id": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" },
          "kind": { "enum": ["contract", "schema", "policy", "runtime", "command", "adapter", "template", "fixture", "ci", "release", "docs"] },
          "paths": { "type": "array", "minItems": 1, "items": { "type": "string", "minLength": 1 } },
          "owner": { "type": "string", "minLength": 1 },
          "stability": { "enum": ["experimental", "stable", "deprecated"] },
          "agent_edit": { "enum": ["agent_editable", "human_approved", "human_only"] },
          "tests": { "type": "array", "minItems": 1, "items": { "type": "string", "minLength": 1 } },
          "description": { "type": "string" }
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}
`
	if err := os.WriteFile(filepath.Join(dir, "components-registry.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}
}

func createComponentsAuthority(t *testing.T, dir string) {
	t.Helper()
	content := `version: 1
authority_classes:
  human_only:
    description: protected
    examples: []
protected_paths:
  - path: "policies/admission.yaml"
    authority: human_only
    rationale: test
report_only: true
governance_check:
  behavior: warn
  exit_on_warnings: false
  block_on_violations: false
`
	if err := os.WriteFile(filepath.Join(dir, "authority.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupComponentsTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "components"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	createComponentsRegistry(t, filepath.Join(tmpDir, "components"))
	createComponentsSchema(t, filepath.Join(tmpDir, "schemas"))
	createComponentsAuthority(t, filepath.Join(tmpDir, "policies"))
	return tmpDir
}

func TestComponentsValidateTextOutput(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "validate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "ok: true") {
		t.Fatalf("expected ok=true, got: %s", out)
	}
	if !strings.Contains(out, "components: 2") {
		t.Fatalf("expected component count, got: %s", out)
	}
	if !strings.Contains(out, "protected_paths: 1/1 covered") {
		t.Fatalf("expected protected path coverage, got: %s", out)
	}
}

func TestComponentsValidateJSONOutput(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "validate", "--root", tmpDir, "--json"}, &stdout, &stderr)
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
	if result["component_count"] != float64(2) {
		t.Fatalf("expected component_count=2, got: %v", result)
	}
}

func TestComponentsValidateFailure(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "components"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	createComponentsSchema(t, filepath.Join(tmpDir, "schemas"))
	createComponentsAuthority(t, filepath.Join(tmpDir, "policies"))

	registryContent := `version: 1
components:
  - id: docs_only
    kind: docs
    paths:
      - docs/**
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - echo ok
`
	if err := os.WriteFile(filepath.Join(tmpDir, "components", "registry.yaml"), []byte(registryContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "validate", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "ok: false") {
		t.Fatalf("expected ok=false, got: %s", out)
	}
	if !strings.Contains(out, "protected path is not registered to any component") {
		t.Fatalf("expected protected path error, got: %s", out)
	}
}

func TestComponentsListTextOutput(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "list", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# x-harness Components") {
		t.Fatalf("expected header, got: %s", out)
	}
	if !strings.Contains(out, "admission_policy") {
		t.Fatalf("expected admission_policy, got: %s", out)
	}
}

func TestComponentsListJSONOutput(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "list", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	comps, ok := result["components"].([]interface{})
	if !ok || len(comps) != 2 {
		t.Fatalf("expected 2 components, got: %v", result)
	}
}

func TestComponentsExplainValidID(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "explain", "--id", "admission_policy", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Component: admission_policy") {
		t.Fatalf("expected component name, got: %s", out)
	}
	if !strings.Contains(out, "Kind: policy") {
		t.Fatalf("expected kind, got: %s", out)
	}
}

func TestComponentsExplainJSONOutput(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "explain", "--id", "admission_policy", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["id"] != "admission_policy" {
		t.Fatalf("expected admission_policy, got: %v", result)
	}
}

func TestComponentsExplainMissingID(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "explain", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--id <component-id> is required") {
		t.Fatalf("expected missing id error, got: %s", stderr.String())
	}
}

func TestComponentsExplainUnknownID(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "explain", "--id", "unknown", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "component not found") {
		t.Fatalf("expected not found error, got: %s", stderr.String())
	}
}

func TestComponentsChangedWithFiles(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "changed", "--files", "policies/admission.yaml,examples/ci/test.yaml,unknown/file.txt", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Changed files: 3") {
		t.Fatalf("expected changed files count, got: %s", out)
	}
	if !strings.Contains(out, "admission_policy") {
		t.Fatalf("expected admission_policy, got: %s", out)
	}
	if !strings.Contains(out, "Unregistered files:") {
		t.Fatalf("expected unregistered files section, got: %s", out)
	}
}

func TestComponentsChangedJSONOutput(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "changed", "--files", "policies/admission.yaml", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["base"] != "main" {
		t.Fatalf("expected base=main, got: %v", result)
	}
	comps, ok := result["components"].([]interface{})
	if !ok || len(comps) != 1 {
		t.Fatalf("expected 1 component, got: %v", result)
	}
}

func TestComponentsChangedGitFailure(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "components"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	createComponentsRegistry(t, filepath.Join(tmpDir, "components"))
	createComponentsSchema(t, filepath.Join(tmpDir, "schemas"))
	createComponentsAuthority(t, filepath.Join(tmpDir, "policies"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "changed", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error reading changed files") {
		t.Fatalf("expected git error, got: %s", stderr.String())
	}
}

func TestComponentsMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestComponentsUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown components subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestComponentsUnknownFlag(t *testing.T) {
	tmpDir := setupComponentsTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"components", "validate", "--root", tmpDir, "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
