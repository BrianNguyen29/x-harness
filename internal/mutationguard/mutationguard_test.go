package mutationguard

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := exec.Command("git", "-C", dir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	return dir
}

func TestIsGitAvailable(t *testing.T) {
	// In CI and most dev environments git should be available.
	if !IsGitAvailable() {
		t.Skip("git not available in environment")
	}
}

func TestFindGitRoot(t *testing.T) {
	if !IsGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)
	root, err := FindGitRoot(dir)
	if err != nil {
		t.Fatalf("expected to find git root, got error: %v", err)
	}
	if root == "" {
		t.Fatal("expected non-empty git root")
	}
}

func TestTakeSnapshotCleanRepo(t *testing.T) {
	if !IsGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)
	snap, err := TakeSnapshot(dir)
	if err != nil {
		t.Fatalf("expected snapshot, got error: %v", err)
	}
	if len(snap.StatusMap) != 0 {
		t.Fatalf("expected empty status map for clean repo, got %v", snap.StatusMap)
	}
}

func TestTakeSnapshotWithUntracked(t *testing.T) {
	if !IsGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("world"), 0644)

	snap, err := TakeSnapshot(dir)
	if err != nil {
		t.Fatalf("expected snapshot, got error: %v", err)
	}
	if len(snap.StatusMap) != 1 {
		t.Fatalf("expected 1 entry, got %v", snap.StatusMap)
	}
	if snap.StatusMap["hello.txt"] != "??" {
		t.Fatalf("expected ?? status, got %v", snap.StatusMap["hello.txt"])
	}
}

func TestGuardDetectsMutation(t *testing.T) {
	if !IsGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("before"), 0644)
	exec.Command("git", "-C", dir, "add", "tracked.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	mutated := false
	result, err := Guard(dir, func() error {
		os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("after"), 0644)
		mutated = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error from guard, got: %v", err)
	}
	if !mutated {
		t.Fatal("expected mutation to have occurred")
	}
	if !result.Violated {
		t.Fatalf("expected violation, got result: %+v", result)
	}
	if len(result.UnexpectedDeltas) == 0 {
		t.Fatal("expected unexpected deltas")
	}
	if result.UnexpectedDeltas[0].Path != "tracked.txt" {
		t.Fatalf("expected tracked.txt delta, got %v", result.UnexpectedDeltas[0].Path)
	}
}

func TestGuardAllowlistsXHarness(t *testing.T) {
	if !IsGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("before"), 0644)
	exec.Command("git", "-C", dir, "add", "tracked.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	result, err := Guard(dir, func() error {
		os.MkdirAll(filepath.Join(dir, ".x-harness"), 0755)
		os.WriteFile(filepath.Join(dir, ".x-harness", "trace.json"), []byte("{}"), 0644)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Violated {
		t.Fatalf("expected no violation for .x-harness writes, got deltas: %v", result.UnexpectedDeltas)
	}
}

func TestGuardNoMutation(t *testing.T) {
	if !IsGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("before"), 0644)
	exec.Command("git", "-C", dir, "add", "tracked.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	result, err := Guard(dir, func() error {
		// no mutation
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Violated {
		t.Fatalf("expected no violation for no mutation, got deltas: %v", result.UnexpectedDeltas)
	}
}

func TestIsAllowlisted(t *testing.T) {
	tests := []struct {
		path  string
		allow bool
	}{
		{".x-harness", true},
		{".x-harness/trace.json", true},
		{"foo/.x-harness/bar", true},
		{"foo.x-harness", true},
		{"src/main.go", false},
		{"README.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsAllowlisted(tt.path); got != tt.allow {
				t.Fatalf("IsAllowlisted(%q) = %v, want %v", tt.path, got, tt.allow)
			}
		})
	}
}
