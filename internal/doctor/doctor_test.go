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
	for _, name := range []string{"critical_assets", "schemas_compile", "policies_parse", "agents_managed_context", "ci_workflow", "tier_labels", "component_registry", "installed_profile", "managed_blocks_registry"} {
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

func TestRunManagedBlocksRegistryMissing(t *testing.T) {
	tmp := t.TempDir()
	// Create minimal required assets but no registry
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
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
		if c.Name == "managed_blocks_registry" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected managed_blocks_registry to fail, got %s", c.Status)
			}
			if !strings.Contains(c.Note, "managed-blocks registry not found") {
				t.Fatalf("expected note about missing registry, got: %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected managed_blocks_registry check")
	}
}

func TestRunManagedBlocksRegistryStale(t *testing.T) {
	tmp := t.TempDir()
	registryDir := filepath.Join(tmp, ".x-harness")
	os.MkdirAll(registryDir, 0755)

	// Create a stale managed block
	begin := "<!-- BEGIN MANAGED BLOCK: test -->"
	end := "<!-- END MANAGED BLOCK: test -->"
	content := "# File\n\n" + begin + "\n<!-- hash: deadbeef -->\n\nBody\n\n" + end + "\n"
	os.WriteFile(filepath.Join(tmp, "test.md"), []byte(content), 0644)

	registry := `version: "1"
blocks:
  - path: test.md
    type: contract
    begin_marker: "<!-- BEGIN MANAGED BLOCK: test -->"
    end_marker: "<!-- END MANAGED BLOCK: test -->"
    hash_prefix: "<!-- hash: "
`
	os.WriteFile(filepath.Join(registryDir, "managed-blocks.yaml"), []byte(registry), 0644)

	// Create minimal required assets
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
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
		if c.Name == "managed_blocks_registry" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected managed_blocks_registry to fail, got %s", c.Status)
			}
			if !strings.Contains(c.Note, "stale") {
				t.Fatalf("expected note about stale hash, got: %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected managed_blocks_registry check")
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

func TestRunOverclaimNotPresentByDefault(t *testing.T) {
	report := Run("../..")
	found := false
	for _, c := range report.Checks {
		if c.Name == "overclaim_phrases" {
			found = true
		}
	}
	if found {
		t.Fatal("expected overclaim_phrases check to NOT be present without Overclaim option")
	}
}

func TestRunOverclaimClean(t *testing.T) {
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
	os.WriteFile(filepath.Join(tmp, "docs", "clean.md"), []byte("# Clean Docs\n\nThis is a clean document with no overclaims.\n"), 0644)

	report := RunWithOptions(tmp, Options{Overclaim: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "overclaim_phrases" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected overclaim_phrases to pass, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected overclaim_phrases check")
	}
}

func TestRunOverclaimDetected(t *testing.T) {
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
	os.WriteFile(filepath.Join(tmp, "docs", "overclaim.md"), []byte("# Docs\n\nThis tool guarantees correctness.\n"), 0644)

	report := RunWithOptions(tmp, Options{Overclaim: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "overclaim_phrases" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected overclaim_phrases to fail, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "guarantees correctness") {
				t.Fatalf("expected note to mention phrase, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected overclaim_phrases check")
	}
	if report.Healthy {
		t.Fatal("expected unhealthy when overclaim is found")
	}
}

func TestRunOverclaimNegatedDisclaimer(t *testing.T) {
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
	os.WriteFile(filepath.Join(tmp, "docs", "disclaimer.md"), []byte("# Docs\n\nThis does not guarantee correctness.\n"), 0644)

	report := RunWithOptions(tmp, Options{Overclaim: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "overclaim_phrases" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected overclaim_phrases to pass with negated disclaimer, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected overclaim_phrases check")
	}
}

func TestRunOverclaimTypeScriptHistorical(t *testing.T) {
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
	os.WriteFile(filepath.Join(tmp, "docs", "history.md"), []byte("# Docs\n\nHistorical note: TypeScript-first was the original approach.\n"), 0644)

	report := RunWithOptions(tmp, Options{Overclaim: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "overclaim_phrases" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected overclaim_phrases to pass with historical TypeScript-first, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected overclaim_phrases check")
	}
}

func TestRunOverclaimRoadmapExcluded(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)
	// Roadmap file listing overclaim phrases to detect - should be excluded
	os.WriteFile(filepath.Join(tmp, "X_HARNESS_ROADMAP.md"), []byte("# Roadmap\n\n- `TypeScript-first` is an overclaim phrase\n- `guarantees correctness` should be detected\n"), 0644)
	// Changelog file - should be excluded
	os.WriteFile(filepath.Join(tmp, "CHANGELOG.md"), []byte("# Changelog\n\n## v1.0\n- TypeScript-first was the original design\n"), 0644)
	// Improvement plan - should be excluded
	os.WriteFile(filepath.Join(tmp, "X_HARNESS_IMPROVEMENT_PLAN.md"), []byte("# Improvement Plan\n\nPhrases to detect:\n- `prevents all bugs`\n"), 0644)

	report := RunWithOptions(tmp, Options{Overclaim: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "overclaim_phrases" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected overclaim_phrases to pass with excluded roadmap/changelog files, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected overclaim_phrases check")
	}
}

func TestRunOverclaimNormalDocStillFails(t *testing.T) {
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
	// Normal doc with actual overclaim should still fail
	os.WriteFile(filepath.Join(tmp, "docs", "normal.md"), []byte("# Normal Doc\n\nThis tool guarantees correctness.\n"), 0644)

	report := RunWithOptions(tmp, Options{Overclaim: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "overclaim_phrases" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected overclaim_phrases to fail with actual overclaim in normal doc, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "guarantees correctness") {
				t.Fatalf("expected note to mention phrase, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected overclaim_phrases check")
	}
	if report.Healthy {
		t.Fatal("expected unhealthy when overclaim is found in normal doc")
	}
}

func TestRunContextRefsNotPresentByDefault(t *testing.T) {
	report := Run("../..")
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
		}
	}
	if found {
		t.Fatal("expected context_refs check to NOT be present without Context option")
	}
}

func TestRunContextRefsNoCards(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass with no cards, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "no completion cards") {
				t.Fatalf("expected note about no cards, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
}

func TestRunContextRefsValidRefs(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card with valid refs relative to card location
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "contract.md"), []byte("# Contract\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass with valid refs, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
}

func TestRunContextRefsMissingRefs(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card with missing refs
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - nonexistent-file.md
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "failed" {
				t.Fatalf("expected context_refs to fail with missing refs, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "nonexistent-file.md") {
				t.Fatalf("expected note to mention missing file, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
	if report.Healthy {
		t.Fatal("expected unhealthy when refs are missing")
	}
}

func TestRunContextRefsAnchorStripped(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card with anchor refs but file without anchor exists
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "contract.md"), []byte("# Contract\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md#some-anchor
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass when file exists (anchor stripped), got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
}

func TestRunContextRefsCardWithoutAlignment(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card without context_alignment - should not fail
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: light
owner: alice
accountable: bob
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass when card has no context_alignment, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
}

func TestRunContextRefsMissingAnchorWarns(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "X_HARNESS.md"), []byte("# X-Harness\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card with anchor ref but file exists without the anchor
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "contract.md"), []byte("# Contract\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md#some-anchor
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass (warning-only), got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
	// Check that a warning note was added
	foundNote := false
	for _, n := range report.Notes {
		if strings.Contains(n, "anchor warning") && strings.Contains(n, "some-anchor") {
			foundNote = true
			break
		}
	}
	if !foundNote {
		t.Fatalf("expected anchor warning note in report.Notes, got: %v", report.Notes)
	}
}

func TestRunContextRefsValidAnchorNoWarning(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card with anchor ref and file contains the anchor (literal)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "contract.md"), []byte("# Contract\n\n#some-anchor\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md#some-anchor
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass with valid anchor, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
	// Check that no anchor warning was added
	for _, n := range report.Notes {
		if strings.Contains(n, "anchor warning") {
			t.Fatalf("expected no anchor warning for valid anchor, got: %v", report.Notes)
		}
	}
}

func TestRunContextRefsValidHeadingSlugNoWarning(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card with heading slug anchor and file contains matching heading
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "contract.md"), []byte("# Contract\n\n## Some Anchor\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md#some-anchor
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass with valid heading slug, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
	// Check that no anchor warning was added
	for _, n := range report.Notes {
		if strings.Contains(n, "anchor warning") {
			t.Fatalf("expected no anchor warning for valid heading slug, got: %v", report.Notes)
		}
	}
}

func TestRunContextRefsScansNonGoldenExamples(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card in examples/adversarial (non-golden) with valid refs
	os.MkdirAll(filepath.Join(tmp, "examples", "adversarial", "spoof-test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "adversarial", "spoof-test", "contract.md"), []byte("# Contract\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "adversarial", "spoof-test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-ADV-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass with valid refs in non-golden example, got %s: %s", c.Status, c.Note)
			}
			// Verify the card was scanned (note should mention card count >= 1)
			if !strings.Contains(c.Note, "1 card") {
				t.Fatalf("expected note to mention card was scanned, got: %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
}

func TestRunContextRefsUnreadableCard(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a valid card
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "valid"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "valid", "contract.md"), []byte("# Contract\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "valid", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-VALID-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	// Create an unreadable card (malformed YAML - tab indent followed by invalid structure)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "bad"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "bad", "completion-card.yaml"), []byte("schema_version: \"1\"\ntask_id: TASK-BAD-001\ntier: standard\nowner: alice\naccountable: bob\ncontext_alignment:\n\tstale_ground_checked: true\n  product_contract_refs:\n    - contract.md\n  invalid: [unclosed\n"), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass, got %s: %s", c.Status, c.Note)
			}
			// Should report 2 scanned, 1 with alignment, 0 without, 1 unreadable/unparseable
			if !strings.Contains(c.Note, "2 card(s) scanned") {
				t.Fatalf("expected note to mention 2 cards scanned, got: %s", c.Note)
			}
			if !strings.Contains(c.Note, "1 with context_alignment") {
				t.Fatalf("expected note to mention 1 with context_alignment, got: %s", c.Note)
			}
			if !strings.Contains(c.Note, "1 unreadable/unparseable") {
				t.Fatalf("expected note to mention 1 unreadable/unparseable, got: %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
}

func TestRunContextRefsExcludedDirsSkipped(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n<!-- BEGIN X-HARNESS MANAGED CONTEXT -->\n<!-- END X-HARNESS MANAGED CONTEXT -->\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	os.MkdirAll(filepath.Join(tmp, "templates"), 0755)
	os.MkdirAll(filepath.Join(tmp, "examples", "golden"), 0755)
	os.WriteFile(filepath.Join(tmp, "policies", "mutation-guard.yaml"), []byte("{}\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "x-harness-verify.yml"), []byte("name: ci\njobs:\n  verify:\n    steps:\n      - run: echo ok\n"), 0644)

	// Create a card in examples/node_modules (excluded dir) - should NOT cause failure
	os.MkdirAll(filepath.Join(tmp, "examples", "node_modules", "fake-package"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "node_modules", "fake-package", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-FAKE-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - nonexistent.md
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	// Create a valid card in golden to verify scan still works
	os.MkdirAll(filepath.Join(tmp, "examples", "golden", "test"), 0755)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "contract.md"), []byte("# Contract\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "examples", "golden", "test", "completion-card.yaml"), []byte(`schema_version: "1"
task_id: TASK-TEST-001
tier: standard
owner: alice
accountable: bob
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - contract.md
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
`), 0644)

	report := RunWithOptions(tmp, Options{Context: true})
	found := false
	for _, c := range report.Checks {
		if c.Name == "context_refs" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected context_refs to pass (excluded dir card should be skipped), got %s: %s", c.Status, c.Note)
			}
			// Should only scan 1 card (golden one), not the node_modules one
			if !strings.Contains(c.Note, "1 card") {
				t.Fatalf("expected note to mention only 1 card scanned (excluding node_modules), got: %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected context_refs check")
	}
}
