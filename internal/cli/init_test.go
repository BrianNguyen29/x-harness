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
	} {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", file, err)
		}
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "schemas")); err == nil {
		t.Fatal("expected schemas to not exist in minimal mode")
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

	for _, dir := range []string{"schemas", "policies", "01-solo-agent", "02-assisted-agent"} {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", dir, err)
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
	for _, dir := range []string{"schemas", "policies", "01-solo-agent", "02-assisted-agent"} {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", dir, err)
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
