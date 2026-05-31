package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWritesManifest(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	manifestPath := filepath.Join(tmpDir, ".x-harness", "manifest.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest to exist: %v", err)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if !strings.Contains(string(data), "version:") {
		t.Fatalf("expected version in manifest, got: %s", string(data))
	}
	if !strings.Contains(string(data), "profile: minimal") {
		t.Fatalf("expected profile minimal in manifest, got: %s", string(data))
	}
}

func TestInitDryRunDoesNotWriteManifest(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"init", tmpDir, "--dry-run"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	manifestPath := filepath.Join(tmpDir, ".x-harness", "manifest.yaml")
	if _, err := os.Stat(manifestPath); err == nil {
		t.Fatal("expected manifest to not exist in dry-run")
	}
}

func TestRepairPreviewNoManifest(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"repair", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "no manifest found") {
		t.Fatalf("expected no manifest error, got: %q", stderr.String())
	}
}

func TestRepairPreviewShowsDrift(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	// Delete a managed file
	os.Remove(filepath.Join(tmpDir, "X_HARNESS.md"))
	// Modify a managed file
	os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("modified"), 0644)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"repair", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "missing: X_HARNESS.md") {
		t.Fatalf("expected missing X_HARNESS.md in preview, got: %s", out)
	}
	if !strings.Contains(out, "modified: AGENTS.md") {
		t.Fatalf("expected modified AGENTS.md in preview, got: %s", out)
	}
}

func TestRepairApplyRestoresAndBackups(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	os.Remove(filepath.Join(tmpDir, "X_HARNESS.md"))
	os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("modified"), 0644)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"repair", tmpDir, "--apply"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "restored: X_HARNESS.md") {
		t.Fatalf("expected restored X_HARNESS.md, got: %s", out)
	}
	if !strings.Contains(out, "backup: AGENTS.md") {
		t.Fatalf("expected backup AGENTS.md, got: %s", out)
	}
	if !strings.Contains(out, "restored: AGENTS.md") {
		t.Fatalf("expected restored AGENTS.md, got: %s", out)
	}

	// Verify files are restored
	if _, err := os.Stat(filepath.Join(tmpDir, "X_HARNESS.md")); err != nil {
		t.Fatalf("expected X_HARNESS.md to exist after repair: %v", err)
	}
	content, _ := os.ReadFile(filepath.Join(tmpDir, "AGENTS.md"))
	if strings.Contains(string(content), "modified") {
		t.Fatal("expected AGENTS.md to be restored from source")
	}
}

func TestRepairDoesNotTouchUnmanaged(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	unmanaged := filepath.Join(tmpDir, "unmanaged.txt")
	os.WriteFile(unmanaged, []byte("keep me"), 0644)

	Run([]string{"repair", tmpDir, "--apply"}, &strings.Builder{}, &strings.Builder{})

	content, err := os.ReadFile(unmanaged)
	if err != nil {
		t.Fatalf("expected unmanaged file to survive: %v", err)
	}
	if string(content) != "keep me" {
		t.Fatalf("expected unmanaged content to be preserved, got: %s", string(content))
	}
}

func TestUninstallPreviewNoManifest(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder

	code := Run([]string{"uninstall", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "no manifest found") {
		t.Fatalf("expected no manifest error, got: %q", stderr.String())
	}
}

func TestUninstallPreviewListsEntries(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"uninstall", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "would remove: AGENTS.md") {
		t.Fatalf("expected would remove AGENTS.md, got: %s", out)
	}
	if !strings.Contains(out, "To apply, run: xh uninstall --apply --force") {
		t.Fatalf("expected apply hint, got: %s", out)
	}
}

func TestUninstallApplyRequiresForce(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"uninstall", tmpDir, "--apply"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "requires --force") {
		t.Fatalf("expected requires --force error, got: %q", stderr.String())
	}
}

func TestUninstallApplyRemovesManagedPreservesUnmanaged(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	unmanaged := filepath.Join(tmpDir, "unmanaged.txt")
	os.WriteFile(unmanaged, []byte("keep me"), 0644)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"uninstall", tmpDir, "--apply", "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "removed: AGENTS.md") {
		t.Fatalf("expected removed AGENTS.md, got: %s", out)
	}
	if !strings.Contains(out, "removed: .x-harness/manifest.yaml") {
		t.Fatalf("expected manifest removed last, got: %s", out)
	}

	// Verify unmanaged file survives
	content, err := os.ReadFile(unmanaged)
	if err != nil {
		t.Fatalf("expected unmanaged file to survive: %v", err)
	}
	if string(content) != "keep me" {
		t.Fatalf("expected unmanaged content to be preserved, got: %s", string(content))
	}

	// Verify managed files are gone
	if _, err := os.Stat(filepath.Join(tmpDir, "AGENTS.md")); err == nil {
		t.Fatal("expected AGENTS.md to be removed")
	}

	// Verify manifest is gone
	if _, err := os.Stat(filepath.Join(tmpDir, ".x-harness", "manifest.yaml")); err == nil {
		t.Fatal("expected manifest to be removed")
	}
}

func TestUninstallApplyBacksUpExisting(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"uninstall", tmpDir, "--apply", "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "backup:") {
		t.Fatalf("expected backup output, got: %s", out)
	}

	// Verify backup directory exists
	backupDir := filepath.Join(tmpDir, ".x-harness", "backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("expected backup dir to exist: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one backup subdirectory")
	}
}

func TestUninstallPreviewMarksModified(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("modified"), 0644)

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"uninstall", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "modified: AGENTS.md") {
		t.Fatalf("expected modified AGENTS.md in preview, got: %s", out)
	}
}

func TestRepairPreviewNoDrift(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"repair", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "no drift detected") {
		t.Fatalf("expected no drift message, got: %s", out)
	}
}

func TestRepairApplyNoDriftNoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	Run([]string{"init", tmpDir}, &strings.Builder{}, &strings.Builder{})

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"repair", tmpDir, "--apply"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "no drift detected") {
		t.Fatalf("expected no drift message, got: %s", out)
	}

	// Ensure no backup files were created anywhere in the tree
	var backupFound []string
	_ = filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if strings.Contains(d.Name(), ".bak.") {
			backupFound = append(backupFound, path)
		}
		return nil
	})
	if len(backupFound) > 0 {
		t.Fatalf("expected no backup files, found: %v", backupFound)
	}
}
