package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	root := "/tmp/repo"
	got := Resolve(root, "cost-budget.yaml")
	want := filepath.Join(root, "policies", "cost-budget.yaml")
	if got != want {
		t.Fatalf("Resolve(%q, %q) = %q, want %q", root, "cost-budget.yaml", got, want)
	}
}

func TestLoadYAML(t *testing.T) {
	tmpDir := t.TempDir()
	policiesDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policiesDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `version: 1
test:
  value: hello
`
	if err := os.WriteFile(filepath.Join(policiesDir, "test.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Version int `yaml:"version"`
		Test    struct {
			Value string `yaml:"value"`
		} `yaml:"test"`
	}
	if err := LoadYAML(tmpDir, "test.yaml", &doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Version != 1 {
		t.Fatalf("expected version 1, got %d", doc.Version)
	}
	if doc.Test.Value != "hello" {
		t.Fatalf("expected value hello, got %s", doc.Test.Value)
	}
}

func TestLoadYAMLMissing(t *testing.T) {
	tmpDir := t.TempDir()
	var doc map[string]any
	err := LoadYAML(tmpDir, "missing.yaml", &doc)
	if err == nil {
		t.Fatal("expected error for missing policy")
	}
	// Error should contain path context
	if !strings.Contains(err.Error(), filepath.Join(tmpDir, "policies", "missing.yaml")) {
		t.Fatalf("expected error to contain path, got: %v", err)
	}
}

func TestLoadDocumentJSON(t *testing.T) {
	tmpDir := t.TempDir()
	policiesDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policiesDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `{"version": 2}`
	if err := os.WriteFile(filepath.Join(policiesDir, "test.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Version int `json:"version"`
	}
	if err := LoadDocument(tmpDir, "test.json", &doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Version != 2 {
		t.Fatalf("expected version 2, got %d", doc.Version)
	}
}
