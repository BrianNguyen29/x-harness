package doctor

import (
	"os"
	"path/filepath"
	"testing"
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
	for _, name := range []string{"critical_assets", "schemas_compile", "policies_parse", "agents_managed_context", "ci_workflow"} {
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
