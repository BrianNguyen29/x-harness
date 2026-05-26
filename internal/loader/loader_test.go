package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		path     string
		expected Format
	}{
		{"schema.json", FormatJSON},
		{"policy.yaml", FormatYAML},
		{"config.yml", FormatYAML},
		{"README.md", FormatUnknown},
		{"Makefile", FormatUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectFormat(tt.path)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestLoadJSONWithRealSchema(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	var schema map[string]any
	if err := LoadJSON(path, &schema); err != nil {
		t.Fatalf("expected to load real schema, got error: %v", err)
	}
	if schema["$schema"] == nil {
		t.Fatal("expected $schema field in loaded JSON")
	}
}

func TestLoadJSONMissingFile(t *testing.T) {
	var v any
	err := LoadJSON(filepath.Join(t.TempDir(), "missing.json"), &v)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadDocumentJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.json")
	if err := os.WriteFile(path, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var result map[string]any
	if err := LoadDocument(path, &result); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result["key"] != "value" {
		t.Fatalf("expected key=value, got %v", result["key"])
	}
}

func TestLoadDocumentYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("name: test\nvalue: 42\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var result map[string]any
	if err := LoadDocument(path, &result); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result["name"] != "test" {
		t.Fatalf("expected name=test, got %v", result["name"])
	}
}

func TestLoadDocumentUnknownFormat(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var v any
	err := LoadDocument(path, &v)
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}
