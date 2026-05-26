package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResetWithoutConfirmReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"reset"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "requires --confirm") {
		t.Fatalf("expected --confirm safety message, got:\n%s", out)
	}
	if !strings.Contains(out, ".x-harness/tmp/") {
		t.Fatalf("expected tmp path in safety message, got:\n%s", out)
	}
}

func TestResetUnknownFlagReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"reset", "--force"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestResetWithConfirmCleansDirs(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	os.MkdirAll(filepath.Join(tmpDir, ".x-harness", "tmp"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".x-harness", "cache"), 0755)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"reset", "--confirm"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "deleted: .x-harness/tmp/") {
		t.Fatalf("expected deleted tmp, got:\n%s", out)
	}
	if !strings.Contains(out, "deleted: .x-harness/cache/") {
		t.Fatalf("expected deleted cache, got:\n%s", out)
	}
	if !strings.Contains(out, "reset complete.") {
		t.Fatalf("expected reset complete, got:\n%s", out)
	}
}

func TestResetWithConfirmSkipsMissingDirs(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"reset", "--confirm"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "not found (skipping): .x-harness/tmp/") {
		t.Fatalf("expected skip message for tmp, got:\n%s", out)
	}
	if !strings.Contains(out, "not found (skipping): .x-harness/cache/") {
		t.Fatalf("expected skip message for cache, got:\n%s", out)
	}
}
