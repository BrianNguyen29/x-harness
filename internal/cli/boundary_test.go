package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBoundaryHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected usage exit, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage in stderr, got %q", stderr.String())
	}
}

func TestBoundaryUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected usage exit, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown boundary subcommand") {
		t.Errorf("expected unknown subcommand error, got %q", stderr.String())
	}
}

func TestBoundaryLintBundledPolicy(t *testing.T) {
	// The repo ships with policies/boundaries.yaml. Running lint
	// against the working tree should always succeed with three
	// example rules.
	repoRoot := findAdaptersRepoRoot()
	if repoRoot == "" {
		t.Skip("cannot locate repo root")
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "lint", "--root", repoRoot, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	var report struct {
		OK           bool `json:"ok"`
		PolicyLoaded bool `json:"policy_loaded"`
		RulesChecked int  `json:"rules_checked"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !report.OK {
		t.Errorf("expected ok=true, got false")
	}
	if !report.PolicyLoaded {
		t.Errorf("expected policy_loaded=true")
	}
	if report.RulesChecked != 3 {
		t.Errorf("expected 3 rules, got %d", report.RulesChecked)
	}
}

func TestBoundaryLintMissingPolicyIsNoop(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "lint", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0 (no policy is no-op), got %d. stderr: %s", code, stderr.String())
	}
	var report struct {
		OK           bool     `json:"ok"`
		PolicyLoaded bool     `json:"policy_loaded"`
		Warnings     []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !report.OK {
		t.Errorf("expected ok=true")
	}
	if report.PolicyLoaded {
		t.Errorf("expected policy_loaded=false")
	}
	if len(report.Warnings) == 0 {
		t.Errorf("expected warning when no policy loaded")
	}
}

func TestBoundaryLintInvalidPolicy(t *testing.T) {
	dir := t.TempDir()
	policyPath := writeTempFile(t, dir, "policies/boundaries.yaml", "version: 2\nboundaries: []\n")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "lint", "--policy", policyPath, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit error, got %d", code)
	}
	var report struct {
		OK    bool     `json:"ok"`
		Error []string `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if report.OK {
		t.Errorf("expected ok=false")
	}
	if len(report.Error) == 0 {
		t.Errorf("expected at least one error")
	}
}

func TestBoundaryCheckAllCleanRepo(t *testing.T) {
	repoRoot := findAdaptersRepoRoot()
	if repoRoot == "" {
		t.Skip("cannot locate repo root")
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--root", repoRoot, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0 on clean repo, got %d. stderr: %s", code, stderr.String())
	}
	var result struct {
		OK           bool `json:"ok"`
		FilesScanned int  `json:"files_scanned"`
		RulesChecked int  `json:"rules_checked"`
		Violations   []struct {
			RuleID string `json:"rule_id"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Errorf("expected ok=true on clean repo, got false")
	}
	if result.RulesChecked == 0 {
		t.Errorf("expected rules_checked > 0")
	}
	if len(result.Violations) > 0 {
		t.Errorf("expected no violations, got %d: %+v", len(result.Violations), result.Violations)
	}
}

func TestBoundaryCheckViolatingFixture(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "policies/boundaries.yaml", `version: 1
boundaries:
  - id: ui-cannot-access-db
    description: "UI must not import internal DB"
    from: "src/ui/**"
    to_import: "internal/db/**"
    action: deny
    severity: high
    applies_to_languages: [typescript]
`)
	writeTempFile(t, dir, "src/ui/login.ts", `import "internal/db/users";
`)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit error on violation, got %d. stderr: %s", code, stderr.String())
	}
	var result struct {
		OK         bool `json:"ok"`
		Violations []struct {
			RuleID string `json:"rule_id"`
			File   string `json:"file"`
			Import string `json:"import"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.OK {
		t.Errorf("expected ok=false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	v := result.Violations[0]
	if v.RuleID != "ui-cannot-access-db" {
		t.Errorf("rule_id = %q", v.RuleID)
	}
	if v.File != "src/ui/login.ts" {
		t.Errorf("file = %q", v.File)
	}
	if v.Import != "internal/db/users" {
		t.Errorf("import = %q", v.Import)
	}
}

func TestBoundaryCheckAllowSuppresses(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "policies/boundaries.yaml", `version: 1
boundaries:
  - id: r
    from: "src/**"
    to_import: "internal/db/**"
    action: deny
    severity: high
    applies_to_languages: [typescript]
    allow:
      - "internal/db/public/**"
`)
	writeTempFile(t, dir, "src/login.ts", `import "internal/db/public/users";
`)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0 (allow suppresses), got %d. stderr: %s", code, stderr.String())
	}
}

func TestBoundaryCheckMissingPolicyIsNoop(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0 (no policy is no-op), got %d. stderr: %s", code, stderr.String())
	}
	var result struct {
		OK           bool     `json:"ok"`
		PolicyLoaded bool     `json:"policy_loaded"`
		Warnings     []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !result.OK {
		t.Errorf("expected ok=true")
	}
	if result.PolicyLoaded {
		t.Errorf("expected policy_loaded=false")
	}
	if len(result.Warnings) == 0 {
		t.Errorf("expected warning when no policy loaded")
	}
}

func TestBoundaryCheckMissingPolicyExplicit(t *testing.T) {
	// When --policy points to a missing file, we should error out
	// rather than silently no-op.
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--root", dir, "--policy", "/nonexistent/policy.yaml", "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit error, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "policy not found") {
		t.Errorf("expected policy not found error, got %q", stderr.String())
	}
}

func TestBoundaryCheckChangedAcceptsPathArgWithWarning(t *testing.T) {
	// We can't easily mock git in this test, but we can verify the
	// warning is emitted when extra positional args are passed.
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--changed", "--root", dir, dir, "--format", "json"}, &stdout, &stderr)
	// The command may exit OK or error depending on git state; we
	// just want to confirm the warning text was emitted.
	if !strings.Contains(stderr.String(), "--changed ignores positional") {
		t.Errorf("expected warning about positional args, got %q", stderr.String())
	}
	_ = code
}

// TestBoundaryCheckChangedDetectsViolationFromGitDiff is the
// end-to-end regression for the P3-S1 blocker: `xh boundary check
// --changed` must surface the same violations as `--all` when the
// rule's `from` glob contains path separators.
//
// The test stands up a real git repo in a temp dir, commits a
// baseline, then modifies a TypeScript file that violates a `from:
// "src/ui/**"` rule. Running `--changed` against the working tree
// must return exit error and report the violation; before the fix,
// it silently returned exit 0 with `files_checked: 0`.
func TestBoundaryCheckChangedDetectsViolationFromGitDiff(t *testing.T) {
	dir := initTempGitRepo(t)
	writeTempFile(t, dir, "policies/boundaries.yaml", `version: 1
boundaries:
  - id: ui-cannot-access-db
    from: "src/ui/**"
    to_import: "internal/db/**"
    action: deny
    severity: high
    applies_to_languages: [typescript]
`)
	writeTempFile(t, dir, "src/ui/login.ts", `import "internal/db/users";
`)
	gitCommit(t, dir, "baseline")
	// Modify the file so it shows up in `git diff --name-only HEAD`.
	if err := os.WriteFile(
		filepath.Join(dir, "src/ui/login.ts"),
		[]byte(`import "internal/db/users";
// changed
`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--changed", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit error on --changed violation, got %d. stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if strings.Contains(stderr.String(), "warning: cannot read git diff") {
		t.Fatalf("git diff failed: %s", stderr.String())
	}
	var result struct {
		OK           bool `json:"ok"`
		FilesScanned int  `json:"files_scanned"`
		FilesChecked int  `json:"files_checked"`
		Violations   []struct {
			RuleID string `json:"rule_id"`
			File   string `json:"file"`
			Import string `json:"import"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Errorf("expected ok=false, got true (stdout=%s)", stdout.String())
	}
	if result.FilesChecked != 1 {
		t.Errorf("expected files_checked=1, got %d (--changed silently skipped the rule)", result.FilesChecked)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %s", len(result.Violations), stdout.String())
	}
	v := result.Violations[0]
	if v.RuleID != "ui-cannot-access-db" {
		t.Errorf("rule_id = %q", v.RuleID)
	}
	if v.File != "src/ui/login.ts" {
		t.Errorf("file = %q", v.File)
	}
	if v.Import != "internal/db/users" {
		t.Errorf("import = %q", v.Import)
	}
}

// TestBoundaryCheckChangedAllCleanIsClean is the inverse guard: when
// a clean file is the only change, `--changed` must report OK=true.
func TestBoundaryCheckChangedAllCleanIsClean(t *testing.T) {
	dir := initTempGitRepo(t)
	writeTempFile(t, dir, "policies/boundaries.yaml", `version: 1
boundaries:
  - id: r
    from: "src/ui/**"
    to_import: "internal/db/**"
    action: deny
    severity: high
    applies_to_languages: [typescript]
`)
	writeTempFile(t, dir, "src/ui/login.ts", `import "./local";
`)
	gitCommit(t, dir, "baseline")
	if err := os.WriteFile(
		filepath.Join(dir, "src/ui/login.ts"),
		[]byte(`import "./local";
// benign comment
`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--changed", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0 (clean change), got %d. stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if strings.Contains(stderr.String(), "warning: cannot read git diff") {
		t.Fatalf("git diff failed: %s", stderr.String())
	}
}

// initTempGitRepo creates a fresh temp directory initialised as a
// git repository with a local user identity. The test is skipped if
// the `git` binary is not available on PATH.
func initTempGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// gitCommit stages every file under dir and creates an initial
// commit. Used by the --changed tests to establish a HEAD so that
// subsequent edits show up in `git diff --name-only HEAD`.
func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	for _, args := range [][]string{
		{"add", "-A"},
		{"commit", "-q", "-m", msg},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestBoundaryExplainBundled(t *testing.T) {
	repoRoot := findAdaptersRepoRoot()
	if repoRoot == "" {
		t.Skip("cannot locate repo root")
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "explain", "internal/cli/root.go", "--root", repoRoot, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	var report struct {
		OK           bool   `json:"ok"`
		PolicyLoaded bool   `json:"policy_loaded"`
		File         string `json:"file"`
		Rules        []struct {
			RuleID         string `json:"rule_id"`
			AppliesToFile  bool   `json:"applies_to_file"`
			ViolationCount int    `json:"violation_count"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !report.PolicyLoaded {
		t.Errorf("expected policy_loaded=true")
	}
	if !strings.HasSuffix(report.File, "internal/cli/root.go") {
		t.Errorf("file = %q", report.File)
	}
	if len(report.Rules) == 0 {
		t.Errorf("expected at least one rule")
	}
}

func TestBoundaryExplainMissingFile(t *testing.T) {
	repoRoot := findAdaptersRepoRoot()
	if repoRoot == "" {
		t.Skip("cannot locate repo root")
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "explain", "this/does/not/exist.go", "--root", repoRoot, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit error, got %d. stderr: %s", code, stderr.String())
	}
}

func TestBoundaryExplainRequiresFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "explain", "--format", "json"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected usage, got %d", code)
	}
}

func TestBoundaryCheckUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected usage, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Errorf("expected unknown flag error, got %q", stderr.String())
	}
}

func TestBoundaryCheckUnknownFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--format", "yaml"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected usage, got %d", code)
	}
}

func TestBoundaryCheckDeterministic(t *testing.T) {
	// Run twice and compare output bytes.
	var first, second bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"boundary", "check", "--all", "--format", "json"}, &first, &stderr); code != ExitOK {
		t.Fatalf("first run failed: %d stderr=%s", code, stderr.String())
	}
	if code := Run([]string{"boundary", "check", "--all", "--format", "json"}, &second, &stderr); code != ExitOK {
		t.Fatalf("second run failed: %d stderr=%s", code, stderr.String())
	}
	if first.String() != second.String() {
		t.Errorf("boundary check --all is not deterministic")
	}
}

func TestBoundaryTextOutputHasHeader(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "# x-harness Boundary Check") {
		t.Errorf("expected text header, got %q", stdout.String())
	}
}

func TestBoundaryExplainTextOutput(t *testing.T) {
	repoRoot := findAdaptersRepoRoot()
	if repoRoot == "" {
		t.Skip("cannot locate repo root")
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "explain", "internal/cli/root.go", "--root", repoRoot}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "# x-harness Boundary Explain") {
		t.Errorf("expected explain header, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "rules:") {
		t.Errorf("expected rules section, got %q", stdout.String())
	}
}

func TestBoundaryLintTextOutput(t *testing.T) {
	repoRoot := findAdaptersRepoRoot()
	if repoRoot == "" {
		t.Skip("cannot locate repo root")
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "lint", "--root", repoRoot}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "# x-harness Boundary Lint") {
		t.Errorf("expected lint header, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "rules_checked:") {
		t.Errorf("expected rules_checked, got %q", stdout.String())
	}
}

func TestBoundaryCheckGoImportBlock(t *testing.T) {
	// Verifies the multi-line import ( ... ) block is parsed.
	dir := t.TempDir()
	writeTempFile(t, dir, "policies/boundaries.yaml", `version: 1
boundaries:
  - id: r
    from: "**/*.go"
    to_import: "internal/data/**"
    action: deny
    severity: high
    applies_to_languages: [go]
`)
	writeTempFile(t, dir, "main.go", `package main

import (
	"fmt"
	"internal/data/users"
)
`)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "check", "--all", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit error, got %d. stderr: %s", code, stderr.String())
	}
	var result struct {
		Violations []struct {
			Line   int    `json:"line"`
			Import string `json:"import"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Import != "internal/data/users" {
		t.Errorf("import = %q", result.Violations[0].Import)
	}
	// The violation is attributed to the `import (` line (line 3).
	if result.Violations[0].Line != 3 {
		t.Errorf("line = %d, want 3", result.Violations[0].Line)
	}
}

func TestBoundaryExplainViolationHits(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "policies/boundaries.yaml", `version: 1
boundaries:
  - id: r1
    from: "src/**"
    to_import: "internal/db/**"
    action: deny
    severity: high
    applies_to_languages: [typescript]
  - id: r2
    from: "src/**"
    to_import: "internal/safe/**"
    action: deny
    severity: high
    applies_to_languages: [typescript]
`)
	writeTempFile(t, dir, "src/login.ts", `import "internal/db/users";
import "internal/safe/util";
`)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"boundary", "explain", "src/login.ts", "--root", dir, "--format", "json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit error, got %d. stderr: %s", code, stderr.String())
	}
	var report struct {
		OK    bool `json:"ok"`
		Rules []struct {
			RuleID           string   `json:"rule_id"`
			ViolationCount   int      `json:"violation_count"`
			ViolationImports []string `json:"violation_imports"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if report.OK {
		t.Errorf("expected ok=false")
	}
	if len(report.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(report.Rules))
	}
	hits := map[string]int{}
	for _, r := range report.Rules {
		hits[r.RuleID] = r.ViolationCount
	}
	if hits["r1"] != 1 {
		t.Errorf("r1 violation_count = %d, want 1", hits["r1"])
	}
	if hits["r2"] != 1 {
		t.Errorf("r2 violation_count = %d, want 1", hits["r2"])
	}
}
