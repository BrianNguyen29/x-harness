package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupFrozenTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	files := map[string]string{
		"README.md":                           "# test readme\n",
		"AGENTS.md":                           "agents\n",
		"X_HARNESS.md":                        "contract\n",
		"CHANGELOG.md":                        "changes\n",
		"LICENSE":                             "MIT\n",
		"docs/guide.md":                       "guide\n",
		"schemas/frozen-manifest.schema.json": "{}\n",
		"policies/admission.yaml":             "policy\n",
		"templates/SUBAGENT_TASK_light.md":    "light\n",
		"adapters/opencode/README.md":         "adapter\n",
		"components/registry.yaml":            "version: 1\ncomponents:\n  - id: test_component\n",
		"examples/golden/basic.json":          "{}\n",
		"examples/adversarial/tamper.json":    "{}\n",
		"tools/experimental/evolve/README.md": "evolve\n",
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

func TestFrozenExportTextOutput(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "frozen bundle written:") {
		t.Fatalf("expected bundle written message, got: %s", outStr)
	}
	if !strings.Contains(outStr, "files:") {
		t.Fatalf("expected file count, got: %s", outStr)
	}
}

func TestFrozenExportJSONOutput(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["out"] != out {
		t.Fatalf("expected out=%s, got: %v", out, result["out"])
	}
}

func TestFrozenExportMissingFrozenFlag(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires --frozen") {
		t.Fatalf("expected --frozen error, got: %s", stderr.String())
	}
}

func TestFrozenExportMissingOutFlag(t *testing.T) {
	root := setupFrozenTestDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--root", root}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--out") {
		t.Fatalf("expected --out error, got: %s", stderr.String())
	}
}

func TestFrozenVerifyTextOutput(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "verify", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "frozen bundle valid:") {
		t.Fatalf("expected valid message, got: %s", outStr)
	}
	if !strings.Contains(outStr, "files:") {
		t.Fatalf("expected file count, got: %s", outStr)
	}
}

func TestFrozenVerifyJSONOutput(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "verify", out, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
}

func TestFrozenVerifyInvalidBundle(t *testing.T) {
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := os.WriteFile(out, []byte("not a bundle"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "verify", out}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestFrozenImportDryRunTextOutput(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	target := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "import", out, "--frozen", "--target", target}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "frozen import dry-run:") {
		t.Fatalf("expected dry-run message, got: %s", outStr)
	}
}

func TestFrozenImportDryRunJSONOutput(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	target := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "import", out, "--frozen", "--target", target, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
	if result["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got: %v", result["dry_run"])
	}
}

func TestFrozenImportWrite(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	target := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "import", out, "--frozen", "--target", target, "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "frozen import wrote") {
		t.Fatalf("expected wrote message, got: %s", outStr)
	}
	// Verify a file was written
	if _, err := os.Stat(filepath.Join(target, "README.md")); err != nil {
		t.Fatalf("expected README.md to be written: %v", err)
	}
}

func TestFrozenImportConflict(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("existing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "import", out, "--frozen", "--target", target, "--no-dry-run"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "conflict:") {
		t.Fatalf("expected conflict message, got: %s", outStr)
	}
}

func TestFrozenImportMerge(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("existing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "import", out, "--frozen", "--target", target, "--merge"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, "frozen import wrote") {
		t.Fatalf("expected wrote message, got: %s", outStr)
	}
}

func TestFrozenImportMissingFrozenFlag(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	target := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "import", out, "--target", target}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires --frozen") {
		t.Fatalf("expected --frozen error, got: %s", stderr.String())
	}
}

func TestFrozenImportMissingTargetFlag(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"frozen", "import", out, "--frozen"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--target") {
		t.Fatalf("expected --target error, got: %s", stderr.String())
	}
}

func TestFrozenVerifyMissingBundle(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "verify", "/nonexistent/bundle.tar.gz"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
}

func TestFrozenMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestFrozenUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown frozen subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestFrozenUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"frozen", "export", "--frozen", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
