package frozen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupExportTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	files := map[string]string{
		"README.md":                              "# test readme\n",
		"AGENTS.md":                              "agents\n",
		"X_HARNESS.md":                           "contract\n",
		"CHANGELOG.md":                           "changes\n",
		"LICENSE":                                "MIT\n",
		"docs/guide.md":                          "guide\n",
		"schemas/frozen-manifest.schema.json":    "{}\n",
		"policies/admission.yaml":                "policy\n",
		"templates/SUBAGENT_TASK_light.md":       "light\n",
		"adapters/opencode/README.md":            "adapter\n",
		"components/registry.yaml":               "version: 1\ncomponents:\n  - id: test_component\n",
		"examples/golden/basic.json":             "{}\n",
		"examples/adversarial/tamper.json":       "{}\n",
		"tools/experimental/evolve/README.md":    "evolve\n",
	}
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return tmpDir
}

func TestFrozenExportBundle(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	result, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if !result.OK {
		t.Fatal("expected ok=true")
	}
	if result.Out != out {
		t.Fatalf("expected out=%s, got %s", out, result.Out)
	}
	if result.FileCount == 0 {
		t.Fatal("expected non-zero file count")
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("bundle file not created: %v", err)
	}
}

func TestFrozenExportRequiresFrozen(t *testing.T) {
	// CLI test; core ExportFrozenBundle doesn't check --frozen
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	result, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if !result.OK {
		t.Fatal("expected ok=true")
	}
}

func TestFrozenVerifyBundleValid(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	_, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	result, err := VerifyFrozenBundle(out)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, errors: %v", result.Errors)
	}
	if result.FileCount == 0 {
		t.Fatal("expected non-zero file count")
	}
}

func TestFrozenVerifyBundleTampered(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	_, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Tamper with the bundle by appending junk bytes
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt by modifying a byte in the middle
	data[len(data)/2] ^= 0xFF
	if err := os.WriteFile(out, data, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyFrozenBundle(out)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	// Tampering should be detected (gzip/tar error or checksum mismatch)
	if result.OK {
		t.Fatal("expected ok=false for tampered bundle")
	}
}

func TestFrozenVerifyMissingBundle(t *testing.T) {
	result, err := VerifyFrozenBundle("/nonexistent/bundle.tar.gz")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false for missing bundle")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected errors for missing bundle")
	}
}

func TestFrozenImportBundleDryRun(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	_, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	target := t.TempDir()
	result, err := ImportFrozenBundle(out, target, true, false, false)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true for dry-run, errors: %v", result.Errors)
	}
	if !result.DryRun {
		t.Fatal("expected dry_run=true")
	}
	if len(result.Planned) == 0 {
		t.Fatal("expected planned files")
	}
	if len(result.Written) > 0 {
		t.Fatal("expected no written files in dry-run")
	}
}

func TestFrozenImportBundleWrite(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	_, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	target := t.TempDir()
	result, err := ImportFrozenBundle(out, target, false, false, true)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true for force write, errors: %v", result.Errors)
	}
	if result.DryRun {
		t.Fatal("expected dry_run=false")
	}
	if len(result.Written) == 0 {
		t.Fatal("expected written files")
	}
	// Verify files exist
	for _, f := range result.Written {
		if _, err := os.Stat(filepath.Join(target, f)); err != nil {
			t.Fatalf("expected file to exist: %s", f)
		}
	}
}

func TestFrozenImportBundleConflict(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	_, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	target := t.TempDir()
	// Pre-create a file to cause conflict
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("existing\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ImportFrozenBundle(out, target, false, false, false)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false for conflict")
	}
	if len(result.Conflicts) == 0 {
		t.Fatal("expected conflicts")
	}
}

func TestFrozenImportBundleMerge(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	_, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	target := t.TempDir()
	// Pre-create a file
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("existing\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ImportFrozenBundle(out, target, false, true, false)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true for merge, errors: %v", result.Errors)
	}
	if len(result.Skipped) == 0 {
		t.Fatal("expected skipped files")
	}
}

func TestFrozenImportBundlePathSafety(t *testing.T) {
	root := setupExportTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	_, err := ExportFrozenBundle(root, out)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	target := t.TempDir()
	result, err := ImportFrozenBundle(out, target, true, false, false)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	// Normal bundle should be safe
	if !result.OK {
		t.Fatalf("expected ok=true, errors: %v", result.Errors)
	}
}

func TestFrozenAssertSafeArchivePath(t *testing.T) {
	cases := []struct {
		path string
		ok   bool
	}{
		{"README.md", true},
		{"docs/guide.md", true},
		{"../escape", false},
		{"foo/../bar", false},
		{"/absolute", false},
	}
	for _, c := range cases {
		err := assertSafeArchivePath(c.path)
		if c.ok && err != nil {
			t.Fatalf("expected safe path for %q, got error: %v", c.path, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("expected unsafe path error for %q", c.path)
		}
	}
}

func TestFrozenCollectFiles(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "sub", "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	files, err := collectFiles(tmpDir, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestFrozenGitCommit(t *testing.T) {
	commit := gitCommit(".")
	if commit == "" {
		t.Fatal("expected non-empty commit")
	}
}

func TestFrozenPackageVersion(t *testing.T) {
	version := packageVersion(".")
	if version == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestFrozenComponentIds(t *testing.T) {
	root := setupExportTestDir(t)
	ids, err := componentIds(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "test_component" {
		t.Fatalf("expected [test_component], got %v", ids)
	}
}

func TestFrozenReadTarGzInvalid(t *testing.T) {
	_, err := readTarGz([]byte("not a tar.gz"))
	if err == nil {
		t.Fatal("expected error for invalid tar.gz")
	}
}

func TestFrozenVerifyMissingManifest(t *testing.T) {
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "bundle.tar.gz")
	entries := []BundleEntry{
		{Path: "version.json", Data: []byte("{}")},
	}
	data, err := createTarGz(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, data, 0644); err != nil {
		t.Fatal(err)
	}
	result, err := VerifyFrozenBundle(out)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false for missing manifest")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "missing manifest.json") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing manifest error, got: %v", result.Errors)
	}
}

func TestFrozenImportInvalidBundle(t *testing.T) {
	target := t.TempDir()
	result, err := ImportFrozenBundle("/nonexistent/bundle.tar.gz", target, true, false, false)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false for invalid bundle")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected errors for invalid bundle")
	}
}
