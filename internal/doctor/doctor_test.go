package doctor

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/contextcheck"
)

func TestRunHealthyRepo(t *testing.T) {
	report := Run("../..")
	if !report.Healthy {
		t.Fatalf("expected healthy repo, got unhealthy. Missing: %v", report.Missing)
	}
	if report.MissingCount != 0 {
		t.Fatalf("expected 0 missing, got %d: %v", report.MissingCount, report.Missing)
	}
	if report.PresentCount == 0 {
		t.Fatal("expected some present assets")
	}

	foundChecks := map[string]bool{}
	for _, c := range report.Checks {
		foundChecks[c.Name] = true
		if c.Status != "passed" {
			t.Fatalf("expected check %s to pass, got %s: %s", c.Name, c.Status, c.Note)
		}
	}
	for _, name := range []string{"critical_assets", "schemas_compile", "policies_parse", "agents_managed_context", "ci_workflow", "tier_labels", "component_registry", "installed_profile"} {
		if !foundChecks[name] {
			t.Fatalf("expected check %s to be present", name)
		}
	}
}

func TestRunMissingRoot(t *testing.T) {
	report := Run("")
	if report.Healthy {
		t.Fatal("expected unhealthy for empty root")
	}
	if report.MissingCount == 0 {
		t.Fatal("expected missing items for empty root")
	}
}

func TestRunNonExistentRoot(t *testing.T) {
	report := Run("/tmp/nonexistent-x-harness-12345")
	if report.Healthy {
		t.Fatal("expected unhealthy for nonexistent root")
	}
	if report.MissingCount == 0 {
		t.Fatal("expected missing items for nonexistent root")
	}
}

func TestRunBrokenRepo(t *testing.T) {
	tmp := t.TempDir()
	// Create an incomplete repo
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n"), 0644)

	report := Run(tmp)
	if report.Healthy {
		t.Fatal("expected unhealthy for broken repo")
	}
	if report.MissingCount == 0 {
		t.Fatal("expected missing items for broken repo")
	}
	if report.PresentCount == 0 {
		t.Fatal("expected some present items")
	}
}

func TestRunMissingManagedContext(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\nno managed context here\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	report := Run(tmp)
	found := false
	for _, c := range report.Checks {
		if c.Name == "agents_managed_context" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected agents_managed_context to fail, got %s", c.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected agents_managed_context check")
	}
}

func TestRunBadTierLabel(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "docs"), 0755)
	os.WriteFile(filepath.Join(tmp, "docs", "bad.md"), []byte("This task is small.\n"), 0644)

	report := Run(tmp)
	found := false
	for _, c := range report.Checks {
		if c.Name == "tier_labels" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected tier_labels to fail, got %s", c.Status)
			}
			if !strings.Contains(c.Note, "small") {
				t.Fatalf("expected tier_labels note to mention small, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected tier_labels check")
	}
}

func TestRunBrokenComponentRegistry(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "components"), 0755)
	os.WriteFile(filepath.Join(tmp, "components", "registry.yaml"), []byte("invalid: yaml: [\n"), 0644)

	report := Run(tmp)
	found := false
	for _, c := range report.Checks {
		if c.Name == "component_registry" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected component_registry to fail, got %s", c.Status)
			}
			if !strings.Contains(c.Note, "registry") {
				t.Fatalf("expected component_registry note to mention registry, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected component_registry check")
	}
}

func TestRunManifestClean(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".x-harness"), 0755)
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "X_HARNESS.md"), []byte("# X-HARNESS\n"), 0644)
	manifest := `version: "1"
profile: minimal
generated_at: "2026-05-28T00:00:00Z"
entries:
  - path: AGENTS.md
    hash: sha256:` + hashOf("# AGENTS\n") + `
  - path: X_HARNESS.md
    hash: sha256:` + hashOf("# X-HARNESS\n") + `
`
	os.WriteFile(filepath.Join(tmp, ".x-harness", "manifest.yaml"), []byte(manifest), 0644)

	report := Run(tmp)
	found := false
	for _, c := range report.Checks {
		if c.Name == "installed_profile" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected installed_profile to pass, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "minimal") {
				t.Fatalf("expected installed_profile note to mention profile, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected installed_profile check")
	}
}

func TestRunManifestMissingFile(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".x-harness"), 0755)
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n"), 0644)
	manifest := `version: "1"
profile: minimal
generated_at: "2026-05-28T00:00:00Z"
entries:
  - path: AGENTS.md
    hash: sha256:` + hashOf("# AGENTS\n") + `
  - path: X_HARNESS.md
    hash: sha256:0000000000000000000000000000000000000000000000000000000000000000
`
	os.WriteFile(filepath.Join(tmp, ".x-harness", "manifest.yaml"), []byte(manifest), 0644)

	report := Run(tmp)
	found := false
	for _, c := range report.Checks {
		if c.Name == "installed_profile" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected installed_profile to fail, got %s", c.Status)
			}
			if !strings.Contains(c.Note, "missing") {
				t.Fatalf("expected installed_profile note to mention missing, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected installed_profile check")
	}
	if report.Healthy {
		t.Fatal("expected unhealthy when manifest file is missing")
	}
}

func TestRunManifestModifiedFile(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".x-harness"), 0755)
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS modified\n"), 0644)
	manifest := `version: "1"
profile: minimal
generated_at: "2026-05-28T00:00:00Z"
entries:
  - path: AGENTS.md
    hash: sha256:0000000000000000000000000000000000000000000000000000000000000000
`
	os.WriteFile(filepath.Join(tmp, ".x-harness", "manifest.yaml"), []byte(manifest), 0644)

	report := Run(tmp)
	found := false
	for _, c := range report.Checks {
		if c.Name == "installed_profile" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected installed_profile to fail, got %s", c.Status)
			}
			if !strings.Contains(c.Note, "modified") {
				t.Fatalf("expected installed_profile note to mention modified, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected installed_profile check")
	}
	if report.Healthy {
		t.Fatal("expected unhealthy when manifest file is modified")
	}
}

func hashOf(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

func TestRunStalenessFresh(t *testing.T) {
	report := RunWithOptions("../..", Options{Staleness: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "agents_context_staleness" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected agents_context_staleness to pass, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected agents_context_staleness check")
	}
}

func TestRunStalenessStaleHash(t *testing.T) {
	tmp := t.TempDir()
	ctx := contextcheck.CanonicalContext()
	block := contextcheck.ManagedBegin + "\n<!-- generated-by: x-harness -->\n<!-- context-hash: deadbeefdeadbeef -->\n\n" + ctx + "\n\n" + contextcheck.ManagedEnd
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n\n"+block+"\n"), 0644)

	report := RunWithOptions(tmp, Options{Staleness: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "agents_context_staleness" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected agents_context_staleness to fail, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "stale") {
				t.Fatalf("expected note to contain 'stale', got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected agents_context_staleness check")
	}
}

func TestRunStalenessStaleBody(t *testing.T) {
	tmp := t.TempDir()
	ctx := contextcheck.CanonicalContext()
	hash := contextcheck.ContextHash(ctx)
	modifiedCtx := strings.Replace(ctx, "admitted, not claimed", "admitted, not claimed. (modified)", 1)
	block := contextcheck.ManagedBegin + "\n<!-- generated-by: x-harness -->\n<!-- context-hash: " + hash + " -->\n\n" + modifiedCtx + "\n\n" + contextcheck.ManagedEnd
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n\n"+block+"\n"), 0644)

	report := RunWithOptions(tmp, Options{Staleness: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "agents_context_staleness" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected agents_context_staleness to fail, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "differs") {
				t.Fatalf("expected note to contain 'differs', got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected agents_context_staleness check")
	}
}

func TestRunStalenessMissingBlock(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n\nSome content.\n"), 0644)

	report := RunWithOptions(tmp, Options{Staleness: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "agents_context_staleness" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected agents_context_staleness to fail, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "missing managed context block") {
				t.Fatalf("expected note about missing block, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected agents_context_staleness check")
	}
}
