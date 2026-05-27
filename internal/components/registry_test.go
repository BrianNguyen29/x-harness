package components

import (
	"os"
	"path/filepath"
	"testing"
)

func createValidRegistry(t *testing.T, dir string) {
	t.Helper()
	registryContent := `version: 1
components:
  - id: test_component
    kind: runtime
    paths:
      - internal/**
      - cmd/x-harness
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - go test ./...
  - id: docs
    kind: docs
    paths:
      - docs/**
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - echo ok
`
	componentsDir := filepath.Join(dir, "components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(componentsDir, "registry.yaml"), []byte(registryContent), 0644); err != nil {
		t.Fatal(err)
	}
}

func createSchema(t *testing.T, dir string) {
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
	schemasDir := filepath.Join(dir, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemasDir, "components-registry.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}
}

func createAuthorityPolicy(t *testing.T, dir string, protectedPaths []string) {
	t.Helper()
	content := "version: 1\nauthority_classes:\n  human_only:\n    description: protected\n    examples: []\nprotected_paths:\n"
	for _, p := range protectedPaths {
		content += "  - path: \"" + p + "\"\n    authority: human_only\n    rationale: test\n"
	}
	content += "report_only: true\ngovernance_check:\n  behavior: warn\n  exit_on_warnings: false\n  block_on_violations: false\n"
	policiesDir := filepath.Join(dir, "policies")
	if err := os.MkdirAll(policiesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policiesDir, "authority.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRegistryValid(t *testing.T) {
	tmpDir := t.TempDir()
	createValidRegistry(t, tmpDir)

	reg, err := LoadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.Version != 1 {
		t.Fatalf("expected version 1, got %d", reg.Version)
	}
	if len(reg.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(reg.Components))
	}
}

func TestLoadRegistryMissing(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadRegistry(tmpDir)
	if err == nil {
		t.Fatal("expected error for missing registry")
	}
}

func TestValidateRegistryValid(t *testing.T) {
	tmpDir := t.TempDir()
	createValidRegistry(t, tmpDir)
	createSchema(t, tmpDir)
	createAuthorityPolicy(t, tmpDir, []string{"internal/**"})

	result, err := ValidateRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected validation ok, got errors: %v", result.Errors)
	}
	if result.ComponentCount != 2 {
		t.Fatalf("expected 2 components, got %d", result.ComponentCount)
	}
	if result.ProtectedPathsChecked != 1 {
		t.Fatalf("expected 1 protected path checked, got %d", result.ProtectedPathsChecked)
	}
	if result.ProtectedPathsCovered != 1 {
		t.Fatalf("expected 1 protected path covered, got %d", result.ProtectedPathsCovered)
	}
}

func TestValidateRegistryMissingProtectedPath(t *testing.T) {
	tmpDir := t.TempDir()
	createValidRegistry(t, tmpDir)
	createSchema(t, tmpDir)
	createAuthorityPolicy(t, tmpDir, []string{"internal/**", "missing/**"})

	result, err := ValidateRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected validation to fail")
	}
	found := false
	for _, e := range result.Errors {
		if e == "protected path is not registered to any component: missing/**" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing protected path error, got: %v", result.Errors)
	}
}

func TestValidateRegistryDuplicateID(t *testing.T) {
	tmpDir := t.TempDir()
	registryContent := `version: 1
components:
  - id: dup
    kind: runtime
    paths:
      - a/**
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - echo ok
  - id: dup
    kind: docs
    paths:
      - b/**
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - echo ok
`
	componentsDir := filepath.Join(tmpDir, "components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(componentsDir, "registry.yaml"), []byte(registryContent), 0644); err != nil {
		t.Fatal(err)
	}
	createSchema(t, tmpDir)
	createAuthorityPolicy(t, tmpDir, []string{})

	result, err := ValidateRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatal("expected validation to fail")
	}
	found := false
	for _, e := range result.Errors {
		if e == "duplicate component id: dup" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected duplicate id error, got: %v", result.Errors)
	}
}

func TestFindComponent(t *testing.T) {
	reg := &ComponentsRegistry{
		Version: 1,
		Components: []ComponentEntry{
			{ID: "a", Kind: "runtime"},
			{ID: "b", Kind: "docs"},
		},
	}
	if c := FindComponent(reg, "a"); c == nil || c.ID != "a" {
		t.Fatal("expected to find component a")
	}
	if c := FindComponent(reg, "c"); c != nil {
		t.Fatal("expected nil for missing component")
	}
}

func TestClassifyFiles(t *testing.T) {
	reg := &ComponentsRegistry{
		Version: 1,
		Components: []ComponentEntry{
			{
				ID:    "core",
				Kind:  "runtime",
				Paths: []string{"internal/**", "cmd/x-harness"},
			},
			{
				ID:    "docs",
				Kind:  "docs",
				Paths: []string{"docs/**"},
			},
		},
	}
	matches, unregistered := ClassifyFiles(reg, []string{"internal/foo.go", "docs/README.md", "unknown.txt"})
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if len(unregistered) != 1 || unregistered[0] != "unknown.txt" {
		t.Fatalf("expected unregistered [unknown.txt], got %v", unregistered)
	}
}

func TestComponentPathMatches(t *testing.T) {
	cases := []struct {
		pattern string
		file    string
		want    bool
	}{
		{"schemas/**", "schemas/foo.schema.json", true},
		{"packages/cli/src/validators/*.ts", "packages/cli/src/validators/base.ts", true},
		{"templates/**", "docs/README.md", false},
		{"AGENTS.md", "AGENTS.md", true},
		{"AGENTS.md", "X_HARNESS.md", false},
	}
	for _, c := range cases {
		got := componentPathMatches(c.pattern, c.file)
		if got != c.want {
			t.Errorf("componentPathMatches(%q, %q) = %v, want %v", c.pattern, c.file, got, c.want)
		}
	}
}

func TestComponentPathCoversPattern(t *testing.T) {
	cases := []struct {
		component string
		protected string
		want      bool
	}{
		{"schemas/**", "schemas/**", true},
		{"schemas/**", "schemas/completion-card.schema.json", true},
		{"docs/**", "policies/admission.yaml", false},
		{"policies/admission.yaml", "policies/admission.yaml", true},
		{"templates/**", "templates/SUBAGENT_TASK_light.md", true},
	}
	for _, c := range cases {
		got := componentPathCoversPattern(c.component, c.protected)
		if got != c.want {
			t.Errorf("componentPathCoversPattern(%q, %q) = %v, want %v", c.component, c.protected, got, c.want)
		}
	}
}
