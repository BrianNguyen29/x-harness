package conformance

import (
	"io"
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
		"denominator_contract",
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

func TestRunStrictHealthyRepo(t *testing.T) {
	report := RunStrict("../..")
	if !report.OK {
		t.Fatalf("expected strict conformance to pass, got: %+v", report)
	}
	if report.Profile != "strict" {
		t.Fatalf("expected profile strict, got %s", report.Profile)
	}

	expectedChecks := []string{
		"critical_files_exist",
		"schemas_compile",
		"policies_parse",
		"agents_managed_context",
		"golden_success_light",
		"golden_blocked_missing_evidence",
		"denominator_contract",
		"scanner_high_severity_clear",
		"worktree_metadata_valid",
		"mutation_guard_verified",
		"adapter_doctor_no_drift",
		"context_gc_no_stale_drift",
		"approval_receipt_for_high_risk",
		"regression_suite_passed",
		"adversarial_suite_passed",
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

func TestRunStrictNonGit(t *testing.T) {
	tmp := t.TempDir()

	// Copy minimal required files from the real repo so minimal passes
	copyDir(t, filepath.Join("..", "..", "schemas"), filepath.Join(tmp, "schemas"))
	copyDir(t, filepath.Join("..", "..", "policies"), filepath.Join(tmp, "policies"))
	copyDir(t, filepath.Join("..", "..", "examples"), filepath.Join(tmp, "examples"))
	copyFile(t, filepath.Join("..", "..", "AGENTS.md"), filepath.Join(tmp, "AGENTS.md"))
	copyFile(t, filepath.Join("..", "..", "X_HARNESS.md"), filepath.Join(tmp, "X_HARNESS.md"))

	report := RunStrict(tmp)
	if report.OK {
		t.Fatal("expected strict conformance to fail for non-git repo")
	}

	foundWorktree := false
	foundMutation := false
	for _, c := range report.Checks {
		if c.Name == "worktree_metadata_valid" && c.Status == "failed" {
			foundWorktree = true
		}
		if c.Name == "mutation_guard_verified" && c.Status == "failed" {
			foundMutation = true
		}
	}
	if !foundWorktree {
		t.Fatal("expected worktree_metadata_valid to fail")
	}
	if !foundMutation {
		t.Fatal("expected mutation_guard_verified to fail")
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read dir %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dst, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
		} else {
			copyFile(t, srcPath, dstPath)
		}
	}
}
