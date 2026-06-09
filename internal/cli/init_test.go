package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitDefaultMinimal(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (minimal) complete") {
		t.Fatalf("expected minimal complete, got: %s", stdout.String())
	}

	for _, file := range []string{
		"AGENTS.md",
		"X_HARNESS.md",
		"docs/VERIFY_GATE.md",
		"docs/RUNTIME_CONTRACT.md",
		"templates/SUBAGENT_TASK_light.md",
		"templates/SUBAGENT_TASK_standard.md",
		"templates/SUBAGENT_TASK_deep.md",
		"templates/COMPLETION_CARD.md",
		"policies/admission.yaml",
		"schemas/completion-card.schema.json",
	} {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", file, err)
		}
	}
}

func TestInitStandardMode(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--standard"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (standard) complete") {
		t.Fatalf("expected standard complete, got: %s", stdout.String())
	}

	for _, item := range []string{"schemas", "policies", "01-solo-agent", "02-assisted-agent", "docs/ADAPTERS.md"} {
		path := filepath.Join(tmpDir, item)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", item, err)
		}
	}
}

func TestInitFullMode(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--full"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (full) complete") {
		t.Fatalf("expected full complete, got: %s", stdout.String())
	}

	for _, dir := range []string{"examples", "schemas", "policies", "templates", "adapters"} {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", dir, err)
		}
	}
}

func TestInitDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--dry-run"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dry run") {
		t.Fatalf("expected dry run header, got: %s", stdout.String())
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err == nil {
		t.Fatal("expected AGENTS.md to not exist in dry-run")
	}
}

func TestInitConflictWithoutForce(t *testing.T) {
	tmpDir := t.TempDir()

	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})
	if err := os.Remove(filepath.Join(tmpDir, "X_HARNESS.md")); err != nil {
		t.Fatalf("failed to remove file for mixed conflict test: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	output := stdout.String() + stderr.String()
	if !strings.Contains(output, "blocked") {
		t.Fatalf("expected blocked message, got: %s", output)
	}
	if !strings.Contains(output, "conflict") {
		t.Fatalf("expected conflict message, got: %s", output)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "X_HARNESS.md")); err == nil {
		t.Fatal("expected no partial write when conflicts are detected")
	}
}

func TestInitForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", tmpDir, "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (minimal) complete") {
		t.Fatalf("expected complete message, got: %s", stdout.String())
	}
}

func TestInitMergePreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("custom content"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", tmpDir, "--merge"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (minimal) complete") {
		t.Fatalf("expected complete message, got: %s", stdout.String())
	}

	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "custom content" {
		t.Fatalf("expected custom content to be preserved, got: %s", string(content))
	}
}

func TestInitAdapters(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--adapters", "opencode"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "adapters", "opencode")); err != nil {
		t.Fatalf("expected adapters/opencode to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "docs", "ADAPTERS.md")); err != nil {
		t.Fatalf("expected docs/ADAPTERS.md to exist: %v", err)
	}
}

func TestInitUnknownFlag(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestInitMissingAssetRoot(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", "--asset-root", "/nonexistent/path"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "invalid --asset-root") {
		t.Fatalf("expected invalid --asset-root error, got: %q", stderr.String())
	}
}

func TestInitDryRunAdapters(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--dry-run", "--adapters", "opencode,cursor"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "would copy:") {
		t.Fatalf("expected would copy lines, got: %s", out)
	}
	if !strings.Contains(out, "adapters/opencode") {
		t.Fatalf("expected adapters/opencode in plan, got: %s", out)
	}
	if !strings.Contains(out, "adapters/cursor") {
		t.Fatalf("expected adapters/cursor in plan, got: %s", out)
	}
	if !strings.Contains(out, "docs/ADAPTERS.md") {
		t.Fatalf("expected docs/ADAPTERS.md in plan, got: %s", out)
	}
}

func TestInitProfileMinimal(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--profile", "minimal"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (minimal) complete") {
		t.Fatalf("expected minimal complete, got: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "schemas", "completion-card.schema.json")); err != nil {
		t.Fatalf("expected schemas/completion-card.schema.json to exist: %v", err)
	}
}

func TestInitProfileStandard(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--profile", "standard"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (standard) complete") {
		t.Fatalf("expected standard complete, got: %s", stdout.String())
	}
	for _, item := range []string{"schemas", "policies", "01-solo-agent", "02-assisted-agent", "docs/ADAPTERS.md"} {
		path := filepath.Join(tmpDir, item)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", item, err)
		}
	}
}

func TestInitProfileDeep(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--profile", "deep"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (deep) complete") {
		t.Fatalf("expected deep complete, got: %s", stdout.String())
	}
	for _, dir := range []string{"examples", "schemas", "policies", "templates", "adapters"} {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", dir, err)
		}
	}
}

func TestInitPreview(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--preview"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dry run") {
		t.Fatalf("expected dry run header, got: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err == nil {
		t.Fatal("expected AGENTS.md to not exist in preview")
	}
}

func TestInitApply(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--apply"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "init (minimal) complete") {
		t.Fatalf("expected minimal complete, got: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to exist: %v", err)
	}
}

func TestInitProfileConflictLegacy(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", "--profile", "minimal", "--full"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "cannot use --profile") {
		t.Fatalf("expected conflict error, got: %q", stderr.String())
	}
}

func TestInitUnknownProfile(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", "--profile", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid profile") {
		t.Fatalf("expected invalid profile error, got: %q", stderr.String())
	}
}

func TestInitIdempotentSameProfile(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--profile", "minimal"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("first init failed: code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"init", tmpDir, "--profile", "minimal"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "already up-to-date") && !strings.Contains(out, "no changes needed") {
		t.Fatalf("expected idempotent no-op message, got: %s", out)
	}
}

func TestInitIdempotentPreservesUnmanagedExtraFiles(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir, "--profile", "minimal"}, &strings.Builder{}, &strings.Builder{})

	extraFile := filepath.Join(tmpDir, "EXTRA.md")
	if err := os.WriteFile(extraFile, []byte("extra content"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", tmpDir, "--profile", "minimal"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "already up-to-date") && !strings.Contains(out, "no changes needed") {
		t.Fatalf("expected idempotent no-op message, got: %s", out)
	}

	content, err := os.ReadFile(extraFile)
	if err != nil {
		t.Fatalf("expected extra file to survive: %v", err)
	}
	if string(content) != "extra content" {
		t.Fatalf("expected extra content to be preserved, got: %s", string(content))
	}
}

func TestInitIdempotentModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir, "--profile", "minimal"}, &strings.Builder{}, &strings.Builder{})

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("modified content"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", tmpDir, "--profile", "minimal"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	output := stdout.String() + stderr.String()
	if !strings.Contains(output, "blocked") && !strings.Contains(output, "conflict") {
		t.Fatalf("expected blocked/conflict message, got: %s", output)
	}
}

func TestInitIdempotentMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir, "--profile", "minimal"}, &strings.Builder{}, &strings.Builder{})

	if err := os.Remove(filepath.Join(tmpDir, "AGENTS.md")); err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"init", tmpDir, "--profile", "minimal"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	output := stdout.String() + stderr.String()
	if !strings.Contains(output, "blocked") && !strings.Contains(output, "conflict") {
		t.Fatalf("expected blocked/conflict message, got: %s", output)
	}
}

// TestInitMinimalEndToEndCheck exercises the user journey:
//  1. xh init --minimal sets up a workspace
//  2. xh add completion-card scaffolds a card
//  3. xh check --card <card.yaml> validates against schemas/completion-card.schema.json
//
// The third step used to fail with "cannot compile schema at <workspace>/schemas/..."
// because minimal init omitted the schemas/ directory. After the fix, the schema is
// present and the verify command must be able to compile it (any further outcome is
// driven by the card's content, which is out of scope here).
func TestInitMinimalEndToEndCheck(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: init --minimal must place the schema required by verify.
	{
		var stdout strings.Builder
		var stderr strings.Builder
		code := Run([]string{"init", tmpDir, "--minimal"}, &stdout, &stderr)
		if code != ExitOK {
			t.Fatalf("init --minimal failed: code=%d stderr=%s", code, stderr.String())
		}
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "schemas", "completion-card.schema.json")); err != nil {
		t.Fatalf("expected schemas/completion-card.schema.json after minimal init: %v", err)
	}

	// Step 2: add completion-card scaffolds a card into the workspace.
	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	{
		var stdout strings.Builder
		var stderr strings.Builder
		code := Run([]string{"add", "completion-card", "task_id=T-MIN-1,tier=light", "--out", cardPath}, &stdout, &stderr)
		if code != ExitOK {
			t.Fatalf("add completion-card failed: code=%d stderr=%s", code, stderr.String())
		}
	}
	if _, err := os.Stat(cardPath); err != nil {
		t.Fatalf("expected completion card to exist: %v", err)
	}

	// Step 3: run check from the workspace root. The original failure
	// (cannot compile schema) must not occur. Card content errors are
	// acceptable here and are out of scope for this regression test.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"check", "--card", cardPath}, &stdout, &stderr)
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "cannot compile schema") {
		t.Fatalf("unexpected schema-compile failure after minimal init:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	if strings.Contains(combined, "no such file or directory") && strings.Contains(combined, "schemas/") {
		t.Fatalf("unexpected missing-schemas failure after minimal init:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	_ = code // non-zero is acceptable when the scaffolded card lacks required fields.
}

// ---------------------------------------------------------------------------
// Wizard tests (P2-S1)
//
// The wizard is a thin, deterministic wrapper around the existing init
// plan/copy logic. It must:
//   - never block on stdin (TTY-free, safe in CI),
//   - never change behavior of plain `xh init` / `xh init --minimal` /
//     `xh init --profile <name>` (regression tested separately),
//   - print a 3-step plan on stdout for both preview and apply paths,
//   - optionally scaffold a first completion card via the existing
//     `xh add completion-card` helper when --wizard-with-card is set
//     (and skip that step on dry-run).
// ---------------------------------------------------------------------------

// TestInitWizardDefaultApply exercises the default wizard path:
// no --wizard-profile, no --wizard-dry-run, no --wizard-with-card.
// It must apply the minimal profile and print the wizard summary.
func TestInitWizardDefaultApply(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--wizard"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()

	// Wizard summary lines must appear on stdout.
	for _, marker := range []string{
		"# xh init --wizard complete",
		"profile: minimal",
		"target:",
	} {
		if !strings.Contains(out, marker) {
			t.Fatalf("expected wizard summary to contain %q, got:\n%s", marker, out)
		}
	}

	// The underlying init must still complete normally.
	if !strings.Contains(out, "init (minimal) complete") {
		t.Fatalf("expected init (minimal) complete in output, got:\n%s", out)
	}

	// And the minimal asset set must be present on disk.
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to exist after wizard apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "schemas", "completion-card.schema.json")); err != nil {
		t.Fatalf("expected schemas/completion-card.schema.json to exist after wizard apply: %v", err)
	}
}

// TestInitWizardDryRunNoMutation ensures --wizard-dry-run (and the
// plain --dry-run alias) print the 3-step plan and do not touch the
// filesystem, even when --wizard-with-card is requested (card
// scaffold is intentionally apply-only).
func TestInitWizardDryRunNoMutation(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{
		"init", tmpDir,
		"--wizard", "--wizard-dry-run", "--wizard-with-card", "T-WIZ-1",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()

	for _, marker := range []string{
		"# xh init --wizard",
		"step 1/3: profile",
		"step 2/3: planned actions",
		"step 3/3: apply decision",
		"-> minimal",
		"preview only",
		"task_id=T-WIZ-1",
		"dry run",
	} {
		if !strings.Contains(out, marker) {
			t.Fatalf("expected wizard dry-run output to contain %q, got:\n%s", marker, out)
		}
	}

	// No files may be written.
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err == nil {
		t.Fatal("expected AGENTS.md to NOT exist after wizard dry-run")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "completion-card.yaml")); err == nil {
		t.Fatal("expected completion-card.yaml to NOT exist after wizard dry-run")
	}

	// And the wizard apply summary must not appear in the dry-run path.
	if strings.Contains(out, "# xh init --wizard complete") {
		t.Fatalf("wizard apply summary should not appear in dry-run output:\n%s", out)
	}
}

// TestInitWizardDryRunAlias ensures the existing --dry-run flag
// works as a wizard dry-run alias: --wizard + --dry-run must be
// equivalent to --wizard + --wizard-dry-run.
func TestInitWizardDryRunAlias(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--wizard", "--dry-run"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# xh init --wizard") {
		t.Fatalf("expected wizard banner via --dry-run alias, got:\n%s", out)
	}
	if !strings.Contains(out, "preview only") {
		t.Fatalf("expected preview-only marker via --dry-run alias, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err == nil {
		t.Fatal("expected AGENTS.md to NOT exist after wizard + --dry-run")
	}
}

// TestInitWizardProfileStandardApply ensures --wizard-profile routes
// through the existing profile->mode conversion and that the
// "standard" profile is honored by the underlying init logic.
func TestInitWizardProfileStandardApply(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{
		"init", tmpDir, "--wizard", "--wizard-profile", "standard",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "profile: standard") {
		t.Fatalf("expected 'profile: standard' in wizard output, got:\n%s", out)
	}
	if !strings.Contains(out, "init (standard) complete") {
		t.Fatalf("expected init (standard) complete in output, got:\n%s", out)
	}
	for _, dir := range []string{"schemas", "policies", "01-solo-agent", "02-assisted-agent"} {
		if _, err := os.Stat(filepath.Join(tmpDir, dir)); err != nil {
			t.Fatalf("expected %s to exist after wizard standard apply: %v", dir, err)
		}
	}
}

// TestInitWizardProfileDeepApply ensures --wizard-profile deep maps
// to the "full" mode internally while displaying as "deep" in both
// the wizard summary and the init completion line.
func TestInitWizardProfileDeepApply(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{
		"init", tmpDir, "--wizard", "--wizard-profile", "deep",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "profile: deep") {
		t.Fatalf("expected 'profile: deep' in wizard output, got:\n%s", out)
	}
	if !strings.Contains(out, "init (deep) complete") {
		t.Fatalf("expected init (deep) complete in output, got:\n%s", out)
	}
}

// TestInitWizardInvalidProfile guards the wizard-specific profile
// validation. An unknown --wizard-profile must produce an ExitUsage
// (matching the existing --profile validation contract).
func TestInitWizardInvalidProfile(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{
		"init", t.TempDir(), "--wizard", "--wizard-profile", "bogus",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid profile") {
		t.Fatalf("expected 'invalid profile' error, got: %q", stderr.String())
	}
}

// TestInitWizardConflictWithProfile ensures --wizard and --profile
// cannot be combined: they describe the same concern via two
// different flag families and mixing them is a usage error.
func TestInitWizardConflictWithProfile(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{
		"init", t.TempDir(), "--wizard", "--profile", "minimal",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "cannot use --wizard with --profile") {
		t.Fatalf("expected wizard/profile conflict error, got: %q", stderr.String())
	}
}

// TestInitWizardConflictWithLegacyFlags ensures --wizard and the
// legacy --minimal/--standard/--full flags cannot be combined.
func TestInitWizardConflictWithLegacyFlags(t *testing.T) {
	for _, legacy := range []string{"--minimal", "--standard", "--full"} {
		t.Run(legacy, func(t *testing.T) {
			var stdout strings.Builder
			var stderr strings.Builder
			code := Run([]string{
				"init", t.TempDir(), "--wizard", legacy,
			}, &stdout, &stderr)
			if code != ExitUsage {
				t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
			}
			if !strings.Contains(stderr.String(), "cannot use --wizard with --minimal, --standard, or --full") {
				t.Fatalf("expected wizard/legacy conflict error, got: %q", stderr.String())
			}
		})
	}
}

// TestInitWizardMissingProfileValue ensures --wizard-profile without
// a value produces a usage error.
func TestInitWizardMissingProfileValue(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{
		"init", t.TempDir(), "--wizard", "--wizard-profile",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "missing value for --wizard-profile") {
		t.Fatalf("expected missing-value error, got: %q", stderr.String())
	}
}

// TestInitWizardWithCardScaffoldsCard ensures --wizard-with-card
// causes a completion card to be scaffolded at the expected path
// inside the init target after a successful wizard apply.
func TestInitWizardWithCardScaffoldsCard(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{
		"init", tmpDir,
		"--wizard", "--wizard-with-card", "T-WIZ-CARD-1",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	if _, err := os.Stat(cardPath); err != nil {
		t.Fatalf("expected completion card at %s, got: %v", cardPath, err)
	}

	// Card file must contain the task_id passed to the wizard and
	// the auto-generated scaffold fields produced by handleAdd.
	data, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("failed to read card: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "task_id: T-WIZ-CARD-1") {
		t.Fatalf("expected card to contain task_id T-WIZ-CARD-1, got:\n%s", body)
	}
	if !strings.Contains(body, "tier: light") {
		t.Fatalf("expected card to contain tier: light, got:\n%s", body)
	}
	if !strings.Contains(body, "id:") || !strings.Contains(body, "created_at:") {
		t.Fatalf("expected card to contain id and created_at, got:\n%s", body)
	}

	// Wizard summary must mention the scaffold.
	if !strings.Contains(stdout.String(), "scaffold: completion-card") {
		t.Fatalf("expected wizard summary to mention scaffold, got:\n%s", stdout.String())
	}
}

// TestInitNoWizardRegression is a defensive guard: the wizard flag
// family must not appear in plain `xh init` output and must not
// change the default init (no --wizard) behavior. If this test ever
// fires, someone has coupled wizard logic into the default path.
func TestInitNoWizardRegression(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()

	// No wizard markers in non-wizard init output.
	for _, marker := range []string{
		"# xh init --wizard",
		"# xh init --wizard complete",
		"step 1/3: profile",
		"step 2/3: planned actions",
		"step 3/3: apply decision",
	} {
		if strings.Contains(out, marker) {
			t.Fatalf("plain `xh init` must not emit %q, got:\n%s", marker, out)
		}
	}

	// The pre-existing minimal complete message must still be present.
	if !strings.Contains(out, "init (minimal) complete") {
		t.Fatalf("expected 'init (minimal) complete' in plain init output, got:\n%s", out)
	}
}

// TestInitWizardUnknownWizardFlag ensures unknown --wizard-* flags
// are rejected by the existing "unknown flag" branch (rather than
// silently ignored), so wizard flag typos surface in tests.
func TestInitWizardUnknownWizardFlag(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{
		"init", t.TempDir(), "--wizard", "--wizard-bogus",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

// TestInitWizardIdempotent ensures re-running the wizard against
// an already-initialized target is a no-op (delegated to the
// existing idempotency check) and does not scaffold a second card.
func TestInitWizardIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	// First run: apply with card scaffold.
	code := Run([]string{
		"init", tmpDir, "--wizard", "--wizard-with-card", "T-WIZ-IDEMP",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("first wizard run failed: code=%d stderr=%s", code, stderr.String())
	}

	// Second run: must report "already up-to-date" and not write a
	// second card. The wizard summary must NOT print "complete"
	// because the underlying init short-circuited.
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"init", tmpDir, "--wizard", "--wizard-with-card", "T-WIZ-IDEMP-2",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("second wizard run failed: code=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "already up-to-date") {
		t.Fatalf("expected idempotent message on second wizard run, got:\n%s", out)
	}
	if strings.Contains(out, "scaffold: completion-card") {
		t.Fatalf("wizard must not scaffold a second card on idempotent run, got:\n%s", out)
	}
}
