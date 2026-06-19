package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectInfoGitRepo(t *testing.T) {
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

	info := CollectInfo(tmpDir)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Root == "" {
		t.Fatal("expected root")
	}
	if info.GitCommonDir == "" {
		t.Fatal("expected git_common_dir")
	}
	if info.Commit == "" {
		t.Fatal("expected commit")
	}
	if !strings.HasPrefix(info.DirtyBaselineHash, "sha256:") {
		t.Fatalf("expected dirty_baseline_hash to start with sha256:, got %s", info.DirtyBaselineHash)
	}
}

func TestCollectInfoNotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	info := CollectInfo(tmpDir)
	if info != nil {
		t.Fatalf("expected nil for non-git dir, got %+v", info)
	}
}

func TestCollectInfoDirtyRepo(t *testing.T) {
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

	// Make it dirty
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := CollectInfo(tmpDir)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	cleanInfo := CollectInfo(tmpDir)
	if cleanInfo == nil {
		t.Fatal("expected non-nil clean info")
	}
	// Dirty repo should have a different hash than a hypothetical clean state
	if info.DirtyBaselineHash == "" {
		t.Fatal("expected dirty_baseline_hash to be non-empty")
	}
	if !strings.HasPrefix(info.DirtyBaselineHash, "sha256:") {
		t.Fatalf("expected dirty_baseline_hash to start with sha256:, got %s", info.DirtyBaselineHash)
	}
}

func TestChangedFiles(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", "a.txt").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	files, err := ChangedFiles(tmpDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no changed files for clean repo, got %v", files)
	}

	// Modify tracked file
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Add untracked file
	if err := os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b\n"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err = ChangedFiles(tmpDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 changed files, got %v", files)
	}
	set := make(map[string]bool)
	for _, f := range files {
		set[f] = true
	}
	if !set["a.txt"] {
		t.Fatal("expected a.txt in changed files")
	}
	if !set["b.txt"] {
		t.Fatal("expected b.txt in changed files")
	}
}

func TestCollectInfoBranch(t *testing.T) {
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
	if err := exec.Command("git", "-C", tmpDir, "checkout", "-b", "feature/test").Run(); err != nil {
		t.Fatalf("git checkout failed: %v", err)
	}

	info := CollectInfo(tmpDir)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Branch != "feature/test" {
		t.Fatalf("expected branch feature/test, got %s", info.Branch)
	}
}
