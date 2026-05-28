package conformance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunMinimalHealthyRepo(t *testing.T) {
	report := RunMinimal("../..")
	if !report.OK {
		t.Fatalf("expected minimal conformance to pass, got: %+v", report)
	}
	if report.Profile != "minimal" {
		t.Fatalf("expected profile minimal, got %s", report.Profile)
	}

	expectedChecks := []string{
		"critical_files_exist",
		"schemas_compile",
		"policies_parse",
		"agents_managed_context",
		"golden_success_light",
		"golden_blocked_missing_evidence",
	}
	found := map[string]bool{}
	for _, c := range report.Checks {
		found[c.Name] = true
	}
	for _, name := range expectedChecks {
		if !found[name] {
			t.Fatalf("expected check %s to be present", name)
		}
	}
}

func TestRunMinimalMissingCriticalFiles(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# AGENTS\n"), 0644)
	os.MkdirAll(filepath.Join(tmp, "policies"), 0755)
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	// Missing X_HARNESS.md and policies/admission.yaml and schemas/completion-card.schema.json

	report := RunMinimal(tmp)
	if report.OK {
		t.Fatal("expected conformance to fail for incomplete repo")
	}

	found := false
	for _, c := range report.Checks {
		if c.Name == "critical_files_exist" && c.Status == "failed" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected critical_files_exist to fail")
	}
}
