package authority

import (
	"os"
	"path/filepath"
	"testing"
)

func setupAuthorityPolicy(t *testing.T, dir string) {
	t.Helper()
	policiesDir := filepath.Join(dir, "policies")
	if err := os.MkdirAll(policiesDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `version: 1
authority_classes:
  agent_editable:
    description: "Files agents can freely modify"
    examples:
      - "packages/cli/src/**/*.ts"
  agent_proposable_human_approved:
    description: "Files agents may propose changes to, but require human approval"
    examples:
      - "policies/recovery.yaml"
  human_only:
    description: "Files only humans may directly modify"
    examples:
      - "schemas/**"
      - "policies/admission.yaml"
protected_paths:
  - path: "schemas/**"
    authority: human_only
    rationale: "Schema definitions are authoritative contracts"
  - path: "policies/admission.yaml"
    authority: human_only
    rationale: "Admission policy defines success criteria"
  - path: "policies/recovery.yaml"
    authority: agent_proposable_human_approved
    rationale: "Recovery routing may be updated by agents with human approval"
  - path: "package.json"
    authority: human_only
    rationale: "Package manifest controls build/test commands"
report_only: true
governance_check:
  behavior: warn
  exit_on_warnings: false
  block_on_violations: false
`
	if err := os.WriteFile(filepath.Join(policiesDir, "authority.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAuthorityPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	setupAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Version != 1 {
		t.Fatalf("expected version 1, got %d", policy.Version)
	}
	if len(policy.ProtectedPaths) != 4 {
		t.Fatalf("expected 4 protected paths, got %d", len(policy.ProtectedPaths))
	}
}

func TestLoadAuthorityPolicyMissing(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadAuthorityPolicy(tmpDir)
	if err == nil {
		t.Fatal("expected error for missing policy")
	}
}

func TestClassifyPath(t *testing.T) {
	tmpDir := t.TempDir()
	setupAuthorityPolicy(t, tmpDir)

	policy, err := LoadAuthorityPolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		path string
		want string
	}{
		{"schemas/completion-card.schema.json", "human_only"},
		{"policies/admission.yaml", "human_only"},
		{"policies/recovery.yaml", "agent_proposable_human_approved"},
		{"package.json", "human_only"},
		{"packages/cli/src/commands/verify.ts", "agent_editable"},
		{"src/main.go", "agent_editable"},
		{".github/workflows/ci.yml", "agent_editable"},
	}

	for _, c := range cases {
		got := ClassifyPath(policy, c.path)
		if got != c.want {
			t.Errorf("ClassifyPath(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestMatchPath(t *testing.T) {
	cases := []struct {
		pattern string
		file    string
		want    bool
	}{
		{"schemas/**", "schemas/foo.schema.json", true},
		{"schemas/**", "schemas/sub/bar.json", true},
		{"policies/admission.yaml", "policies/admission.yaml", true},
		{"policies/admission.yaml", "policies/recovery.yaml", false},
		{"package.json", "package.json", true},
		{"package.json", "sub/package.json", false},
		{".github/workflows/*.yml", ".github/workflows/ci.yml", true},
		{".github/workflows/*.yml", ".github/workflows/sub/ci.yml", false},
		{"templates/**", "templates/SUBAGENT_TASK_light.md", true},
		{"packages/cli/src/validators/*.ts", "packages/cli/src/validators/base.ts", true},
	}

	for _, c := range cases {
		got := matchPath(c.pattern, c.file)
		if got != c.want {
			t.Errorf("matchPath(%q, %q) = %v, want %v", c.pattern, c.file, got, c.want)
		}
	}
}
