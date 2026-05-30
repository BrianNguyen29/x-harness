package conformance

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/contextcheck"
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
		"managed_blocks_registry",
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

func TestCheckSuiteMissingExpectedOutput(t *testing.T) {
	tmp := t.TempDir()

	// Set up minimal schema and a fixture with card but missing expected output
	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	copyFile(t, filepath.Join("..", "..", "schemas", "completion-card.schema.json"), filepath.Join(tmp, "schemas", "completion-card.schema.json"))
	copyFile(t, filepath.Join("..", "..", "schemas", "context-alignment.schema.json"), filepath.Join(tmp, "schemas", "context-alignment.schema.json"))

	fixtureDir := filepath.Join(tmp, "examples", "golden", "regression", "test-fixture")
	os.MkdirAll(fixtureDir, 0755)
	copyFile(t, filepath.Join("..", "..", "examples", "golden", "regression", "success-light", "completion-card.yaml"), filepath.Join(fixtureDir, "completion-card.yaml"))
	// Deliberately omit expected-verify-output.txt

	check := checkSuite(tmp, "regression", "regression_suite_passed")
	if check.Status != "failed" {
		t.Fatalf("expected suite to fail when expected output missing, got status=%s note=%s", check.Status, check.Note)
	}
	if check.Note != "no fixtures found in "+filepath.Join(tmp, "examples", "golden", "regression") {
		t.Fatalf("unexpected note: %s", check.Note)
	}
}

func TestCheckSuiteMismatch(t *testing.T) {
	tmp := t.TempDir()

	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	copyFile(t, filepath.Join("..", "..", "schemas", "completion-card.schema.json"), filepath.Join(tmp, "schemas", "completion-card.schema.json"))
	copyFile(t, filepath.Join("..", "..", "schemas", "context-alignment.schema.json"), filepath.Join(tmp, "schemas", "context-alignment.schema.json"))

	fixtureDir := filepath.Join(tmp, "examples", "golden", "regression", "mismatch-fixture")
	os.MkdirAll(fixtureDir, 0755)
	copyFile(t, filepath.Join("..", "..", "examples", "golden", "regression", "success-light", "completion-card.yaml"), filepath.Join(fixtureDir, "completion-card.yaml"))
	// Write wrong expected output
	os.WriteFile(filepath.Join(fixtureDir, "expected-verify-output.txt"), []byte("outcome: failed\nacceptance_status: withheld\n"), 0644)

	check := checkSuite(tmp, "regression", "regression_suite_passed")
	if check.Status != "failed" {
		t.Fatalf("expected suite to fail on mismatch, got status=%s note=%s", check.Status, check.Note)
	}
	if !strings.Contains(check.Note, "expected outcome=failed got=success") {
		t.Fatalf("expected mismatch note to contain outcome mismatch, got: %s", check.Note)
	}
}

func TestCheckSuitePass(t *testing.T) {
	tmp := t.TempDir()

	os.MkdirAll(filepath.Join(tmp, "schemas"), 0755)
	copyFile(t, filepath.Join("..", "..", "schemas", "completion-card.schema.json"), filepath.Join(tmp, "schemas", "completion-card.schema.json"))
	copyFile(t, filepath.Join("..", "..", "schemas", "context-alignment.schema.json"), filepath.Join(tmp, "schemas", "context-alignment.schema.json"))

	fixtureDir := filepath.Join(tmp, "examples", "golden", "regression", "success-fixture")
	os.MkdirAll(fixtureDir, 0755)
	copyFile(t, filepath.Join("..", "..", "examples", "golden", "regression", "success-light", "completion-card.yaml"), filepath.Join(fixtureDir, "completion-card.yaml"))
	os.WriteFile(filepath.Join(fixtureDir, "expected-verify-output.txt"), []byte("outcome: success\nacceptance_status: accepted\n"), 0644)

	check := checkSuite(tmp, "regression", "regression_suite_passed")
	if check.Status != "passed" {
		t.Fatalf("expected suite to pass, got status=%s note=%s", check.Status, check.Note)
	}
	if check.Note != "1 fixture(s) matched" {
		t.Fatalf("unexpected note: %s", check.Note)
	}
}

func TestWorktreeMetadataValidBlocksPathEscape(t *testing.T) {
	tmp := t.TempDir()

	// Set up a git repo
	if err := exec.Command("git", "-C", tmp, "init").Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmp, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmp, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmp, "add", "file.txt").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmp, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create a golden fixture with an escaping path
	fixtureDir := filepath.Join(tmp, "examples", "golden", "regression", "escape-fixture")
	os.MkdirAll(fixtureDir, 0755)
	card := `schema_version: "1"
task_id: TASK-ESC-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - ../etc/passwd
claim:
  fix_status: fixed
  summary: Escaping path test
verification:
  status: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	os.WriteFile(filepath.Join(fixtureDir, "completion-card.yaml"), []byte(card), 0644)
	os.WriteFile(filepath.Join(fixtureDir, "expected-verify-output.txt"), []byte("outcome: failed\nacceptance_status: withheld\n"), 0644)

	check := checkWorktreeMetadata(tmp)
	if check.Status != "failed" {
		t.Fatalf("expected worktree_metadata_valid to fail on path escape, got status=%s note=%s", check.Status, check.Note)
	}
	if !strings.Contains(check.Note, "../etc/passwd") {
		t.Fatalf("expected note to mention escaping path, got: %s", check.Note)
	}
}

func TestManagedBlocksRegistryStrict(t *testing.T) {
	report := RunStrict("../..")
	found := false
	for _, c := range report.Checks {
		if c.Name == "managed_blocks_registry" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected managed_blocks_registry to pass, got %s: %s", c.Status, c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected managed_blocks_registry check in strict conformance")
	}
}

func TestManagedBlocksRegistryFailsOnStale(t *testing.T) {
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

	check := checkManagedBlocksRegistry(tmp)
	if check.Status != "failed" {
		t.Fatalf("expected managed_blocks_registry to fail on stale block, got status=%s note=%s", check.Status, check.Note)
	}
	if !strings.Contains(check.Note, "stale") {
		t.Fatalf("expected note to mention stale, got: %s", check.Note)
	}
}

func TestContextGCNoStaleDriftBlocksDeadLinks(t *testing.T) {
	tmp := t.TempDir()

	// Write AGENTS.md with a fresh managed block
	ctx := contextcheck.CanonicalContext()
	hash := contextcheck.ContextHash(ctx)
	agentsContent := "# AGENTS\n\n" + contextcheck.ManagedBegin + "\n<!-- context-hash: " + hash + " -->\n\n" + ctx + "\n\n" + contextcheck.ManagedEnd + "\n"
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte(agentsContent), 0644)

	// Write a docs file with a dead link
	docsDir := filepath.Join(tmp, "docs")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "broken.md"), []byte("# Broken\n\nSee [missing](NONEXISTENT.md).\n"), 0644)

	check := checkContextGCNoStaleDrift(tmp)
	if check.Status != "failed" {
		t.Fatalf("expected context_gc_no_stale_drift to fail on dead links, got status=%s note=%s", check.Status, check.Note)
	}
	if !strings.Contains(check.Note, "NONEXISTENT.md") {
		t.Fatalf("expected note to mention dead link, got: %s", check.Note)
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
