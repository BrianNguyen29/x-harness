package assets

import (
	"path/filepath"
	"testing"
)

func TestLocatorPaths(t *testing.T) {
	l := NewLocator("/fake/root")

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"policy", l.Policy("admission.yaml"), filepath.Join("/fake", "root", "policies", "admission.yaml")},
		{"schema", l.Schema("completion-card.schema.json"), filepath.Join("/fake", "root", "schemas", "completion-card.schema.json")},
		{"template", l.Template("SUBAGENT_TASK_standard.md"), filepath.Join("/fake", "root", "templates", "SUBAGENT_TASK_standard.md")},
		{"example", l.Example("golden"), filepath.Join("/fake", "root", "examples", "golden")},
		{"adapter", l.Adapter("opencode"), filepath.Join("/fake", "root", "adapters", "opencode")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}
}
