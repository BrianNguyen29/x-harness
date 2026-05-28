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
