package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/doctor"
)

func TestDoctorWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	if err := exec.Command("git", "-C", tmpDir, "init").Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", "file.txt").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--worktree"}, &stdout, &stderr)
	// doctor may fail because tmpDir lacks x-harness assets; we only care that worktree_info is present and non-blocking

	var report doctor.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}

	found := false
	for _, c := range report.Checks {
		if c.Name == "worktree_info" {
			found = true
			if c.Status != "passed" {
				t.Fatalf("expected worktree_info passed, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "branch=") {
				t.Fatalf("expected worktree note to contain branch=, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected worktree_info check")
	}
	// worktree_info must not change exit code on its own
	_ = code
}

func TestDoctorWorktreeNotGit(t *testing.T) {
	tmpDir := t.TempDir()
	// No git init

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--worktree"}, &stdout, &stderr)

	var report doctor.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}

	found := false
	for _, c := range report.Checks {
		if c.Name == "worktree_info" {
			found = true
			if c.Status != "skipped" {
				t.Fatalf("expected worktree_info skipped, got %s: %s", c.Status, c.Note)
			}
			if !strings.Contains(c.Note, "not a git repository") {
				t.Fatalf("expected skipped note about git, got %s", c.Note)
			}
		}
	}
	if !found {
		t.Fatal("expected worktree_info check")
	}
	_ = code
}

func TestDoctorHelpDocumentsFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--help"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("expected exit code %d for usage, got %d. stdout: %s\nstderr: %s", ExitUsage, code, stdout.String(), stderr.String())
	}

	usage := stderr.String()
	for _, flag := range []string{"--context", "--staleness", "--overclaim", "--fix", "--confirm"} {
		if !strings.Contains(usage, flag) {
			t.Fatalf("expected usage to contain %s, got: %s", flag, usage)
		}
	}
}

// writeHealthyDocsFixture builds a minimal fixture where the
// docs-drift checks pass: a CI workflow invoking x-harness verify
// and a package.json that mentions a verify script.
func writeHealthyDocsFixture(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\njobs:\n  q:\n    steps:\n      - run: x-harness verify --card x.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"scripts":{"verify":"tsc && vitest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

// writeUnhealthyDocsFixture builds a fixture that triggers the
// workflow_missing_verify drift tag.
func writeUnhealthyDocsFixture(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\njobs:\n  q:\n    steps:\n      - run: echo no-verify\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"scripts":{"verify":"tsc"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestDoctorDocsDriftHealthyJSON(t *testing.T) {
	tmpDir := writeHealthyDocsFixture(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--docs-drift", "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var report doctor.DocsDriftReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !report.Healthy {
		t.Fatalf("expected healthy report, got %+v", report)
	}
	if report.Root != tmpDir {
		t.Fatalf("expected root=%s, got %s", tmpDir, report.Root)
	}
}

func TestDoctorDocsDriftUnhealthyJSON(t *testing.T) {
	tmpDir := writeUnhealthyDocsFixture(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--docs-drift", "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}

	var report doctor.DocsDriftReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if report.Healthy {
		t.Fatalf("expected unhealthy report, got %+v", report)
	}
	found := false
	for _, tag := range report.DriftTags {
		if tag == "workflow_missing_verify" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected drift_tag=workflow_missing_verify, got %+v", report.DriftTags)
	}
}

func TestDoctorDocsDriftTextFormat(t *testing.T) {
	tmpDir := writeHealthyDocsFixture(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--docs-drift", "--format", "text"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("expected text output, got JSON: %s", out)
	}
	if !strings.Contains(out, "healthy: true") {
		t.Fatalf("expected text output to contain 'healthy: true', got: %s", out)
	}
	if !strings.Contains(out, "x-harness Docs Drift") {
		t.Fatalf("expected text output header, got: %s", out)
	}
}

func TestDoctorDocsDriftMissingRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--docs-drift"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitUsage, code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--docs-drift") {
		t.Fatalf("expected usage to mention --docs-drift, got: %s", stderr.String())
	}
}

// doctorFixFixture runs `xh init <dir>` to produce a workspace with a
// complete manifest. It returns the tmpDir and exits the test on
// failure. Tests that need a missing deterministic asset delete the
// file from the returned directory before invoking doctor --fix.
func doctorFixFixture(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"init", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("init failed: code=%d stderr=%s", code, stderr.String())
	}
	return tmpDir
}

func TestDoctorFixDryRunDoesNotMutate(t *testing.T) {
	tmpDir := doctorFixFixture(t)
	target := filepath.Join(tmpDir, "docs", "VERIFY_GATE.md")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--fix"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d for dry-run, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected file to remain missing after dry-run, stat err=%v", err)
	}

	var out struct {
		*doctor.Report
		Fix *DoctorFix `json:"fix"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v\noutput: %s", err, stdout.String())
	}
	if out.Fix == nil {
		t.Fatalf("expected fix block in JSON output, got: %s", stdout.String())
	}
	if !out.Fix.DryRun {
		t.Fatalf("expected dry_run=true without --confirm, got fix=%+v", out.Fix)
	}
	if out.Fix.Confirmed {
		t.Fatalf("expected confirmed=false without --confirm, got fix=%+v", out.Fix)
	}
	if !out.Fix.ManifestFound {
		t.Fatalf("expected manifest_found=true, got fix=%+v", out.Fix)
	}
	if !containsString(out.Fix.Applied, "would restore: docs/VERIFY_GATE.md") {
		t.Fatalf("expected 'would restore: docs/VERIFY_GATE.md' in fix.applied, got: %v", out.Fix.Applied)
	}
}

func TestDoctorFixConfirmRestoresMissingAsset(t *testing.T) {
	tmpDir := doctorFixFixture(t)
	target := filepath.Join(tmpDir, "docs", "VERIFY_GATE.md")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--fix", "--confirm"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected %s to be restored, stat err=%v", target, err)
	}

	var out struct {
		*doctor.Report
		Fix *DoctorFix `json:"fix"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v\noutput: %s", err, stdout.String())
	}
	if out.Fix == nil {
		t.Fatalf("expected fix block in JSON output, got: %s", stdout.String())
	}
	if out.Fix.DryRun {
		t.Fatalf("expected dry_run=false with --confirm, got fix=%+v", out.Fix)
	}
	if !out.Fix.Confirmed {
		t.Fatalf("expected confirmed=true with --confirm, got fix=%+v", out.Fix)
	}
	if !containsString(out.Fix.Applied, "fixed: docs/VERIFY_GATE.md") {
		t.Fatalf("expected 'fixed: docs/VERIFY_GATE.md' in fix.applied, got: %v", out.Fix.Applied)
	}
}

func TestDoctorFixNoOpHealthyWorkspace(t *testing.T) {
	tmpDir := doctorFixFixture(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--fix", "--confirm"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var out struct {
		*doctor.Report
		Fix *DoctorFix `json:"fix"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v\noutput: %s", err, stdout.String())
	}
	if out.Fix == nil {
		t.Fatalf("expected fix block in JSON output, got: %s", stdout.String())
	}
	if len(out.Fix.Applied) != 0 {
		t.Fatalf("expected no applied fixes on a healthy workspace, got: %v", out.Fix.Applied)
	}
	if !containsString(out.Fix.Notes, "no managed files require repair") {
		t.Fatalf("expected 'no managed files require repair' note, got: %v", out.Fix.Notes)
	}
}

func TestDoctorFixConfirmRequiredForMutation(t *testing.T) {
	// This is the same invariant as the dry-run test, but it is
	// phrased as an explicit safety check: running --fix without
	// --confirm on a missing deterministic asset must leave the
	// workspace untouched even when an asset root is available.
	tmpDir := doctorFixFixture(t)
	target := filepath.Join(tmpDir, "schemas", "completion-card.schema.json")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--fix"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d for dry-run, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected file to remain missing without --confirm, stat err=%v", err)
	}
}

func TestDoctorFixNonDeterministicNotFixed(t *testing.T) {
	// An overclaim phrase in docs/ is detected by `xh doctor --overclaim`
	// but is NOT a manifest-tracked file, so `xh doctor --fix --confirm`
	// must leave the file untouched.
	tmpDir := doctorFixFixture(t)
	overclaimPath := filepath.Join(tmpDir, "docs", "overclaim.md")
	overclaimContent := "# Overclaim\n\nThis tool guarantees correctness.\n"
	if err := os.WriteFile(overclaimPath, []byte(overclaimContent), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--fix", "--confirm", "--overclaim"}, &stdout, &stderr)
	// The doctor is still unhealthy because the overclaim is present;
	// the fix flow does not change that.
	if code != ExitError {
		t.Fatalf("expected exit code %d (overclaim remains), got %d. stderr: %s", ExitError, code, stderr.String())
	}

	got, err := os.ReadFile(overclaimPath)
	if err != nil {
		t.Fatalf("read overclaim: %v", err)
	}
	if string(got) != overclaimContent {
		t.Fatalf("expected overclaim file to be untouched; got %q want %q", string(got), overclaimContent)
	}

	var out struct {
		*doctor.Report
		Fix *DoctorFix `json:"fix"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v\noutput: %s", err, stdout.String())
	}
	if out.Fix == nil {
		t.Fatalf("expected fix block, got: %s", stdout.String())
	}
	if containsString(out.Fix.Applied, "fixed: docs/overclaim.md") {
		t.Fatalf("doctor --fix must not auto-fix non-deterministic issues; applied=%v", out.Fix.Applied)
	}
}

func TestDoctorFixNoManifestReportsError(t *testing.T) {
	tmpDir := t.TempDir() // no init; no manifest

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--fix", "--confirm"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}

	var out struct {
		*doctor.Report
		Fix *DoctorFix `json:"fix"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v\noutput: %s", err, stdout.String())
	}
	if out.Fix == nil {
		t.Fatalf("expected fix block, got: %s", stdout.String())
	}
	if out.Fix.ManifestFound {
		t.Fatalf("expected manifest_found=false on empty workspace, got: %+v", out.Fix)
	}
	found := false
	for _, n := range out.Fix.Notes {
		if strings.Contains(n, "no manifest at .x-harness/manifest.yaml") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected note about missing manifest, got: %v", out.Fix.Notes)
	}
}

func TestDoctorFixTextOutputDryRun(t *testing.T) {
	tmpDir := doctorFixFixture(t)
	target := filepath.Join(tmpDir, "docs", "RUNTIME_CONTRACT.md")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--root", tmpDir, "--fix", "--format", "text"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d for dry-run, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected file to remain missing in dry-run text mode, stat err=%v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "# xh doctor --fix") {
		t.Fatalf("expected text header in dry-run, got: %s", out)
	}
	if !strings.Contains(out, "would restore: docs/RUNTIME_CONTRACT.md") {
		t.Fatalf("expected dry-run plan line, got: %s", out)
	}
}
