package mutationguard

import (
	"crypto/sha256"
	"fmt"
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
		{".x-harness-mutation-guard-probe-123-456.probe", false},
		{".x-harness-mutation-guard-probe-foo", false},
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

func TestFallbackSnapshotBasic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("world"), 0644)

	snap, err := TakeFallbackSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snap.StatusMap) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(snap.StatusMap))
	}
	if snap.StatusMap["hello.txt"] != "F" {
		t.Fatalf("expected status F, got %s", snap.StatusMap["hello.txt"])
	}
	if snap.HashMap["hello.txt"] == "" {
		t.Fatal("expected hash")
	}
}

func TestFallbackSnapshotDetectsMutation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("before"), 0644)

	before, err := TakeFallbackSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("after"), 0644)

	after, err := TakeFallbackSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deltas := Compare(before, after)
	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d", len(deltas))
	}
	if deltas[0].Path != "tracked.txt" {
		t.Fatalf("expected tracked.txt, got %s", deltas[0].Path)
	}
}

func TestFallbackSnapshotDetectsCreateDelete(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)

	before, err := TakeFallbackSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	os.Remove(filepath.Join(dir, "a.txt"))

	after, err := TakeFallbackSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deltas := Compare(before, after)
	if len(deltas) != 2 {
		t.Fatalf("expected 2 deltas, got %d", len(deltas))
	}

	paths := make(map[string]struct{})
	for _, d := range deltas {
		paths[d.Path] = struct{}{}
	}
	if _, ok := paths["a.txt"]; !ok {
		t.Fatal("expected a.txt delta")
	}
	if _, ok := paths["b.txt"]; !ok {
		t.Fatal("expected b.txt delta")
	}
}

func TestFallbackSnapshotIgnores(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("git"), 0644)
	os.MkdirAll(filepath.Join(dir, "node_modules", "foo"), 0755)
	os.WriteFile(filepath.Join(dir, "node_modules", "foo", "index.js"), []byte("js"), 0644)
	os.MkdirAll(filepath.Join(dir, ".x-harness"), 0755)
	os.WriteFile(filepath.Join(dir, ".x-harness", "trace.json"), []byte("{}"), 0644)

	snap, err := TakeFallbackSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snap.StatusMap) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(snap.StatusMap), snap.StatusMap)
	}
	if _, ok := snap.StatusMap["keep.txt"]; !ok {
		t.Fatalf("expected keep.txt, got %v", snap.StatusMap)
	}
}

func TestFallbackSnapshotAllowlistsXHarness(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("before"), 0644)

	result, err := GuardFallback(dir, func() error {
		os.MkdirAll(filepath.Join(dir, ".x-harness"), 0755)
		os.WriteFile(filepath.Join(dir, ".x-harness", "trace.json"), []byte("{}"), 0644)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Violated {
		t.Fatalf("expected no violation for .x-harness writes, got deltas: %v", result.UnexpectedDeltas)
	}
}

func TestFallbackSnapshotSymlink(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "target.txt"), []byte("target"), 0644)
	if err := os.Symlink("target.txt", filepath.Join(dir, "link.txt")); err != nil {
		t.Skip("cannot create symlink:", err)
	}

	snap, err := TakeFallbackSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snap.StatusMap) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(snap.StatusMap), snap.StatusMap)
	}

	expectedHash := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("symlink:target.txt")))
	if snap.HashMap["link.txt"] != expectedHash {
		t.Fatalf("expected symlink hash %s, got %s", expectedHash, snap.HashMap["link.txt"])
	}
}

func TestFallbackSnapshotConcurrencyEnv(t *testing.T) {
	tests := []struct {
		env  string
		want int
	}{
		{"", 16},
		{"8", 8},
		{"0", 16},
		{"-1", 16},
		{"invalid", 16},
		{"64", 64},
		{"100", 64},
		{"1", 1},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY", tt.env)
			} else {
				os.Unsetenv("X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY")
			}
			if got := hashConcurrency(); got != tt.want {
				t.Fatalf("hashConcurrency() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGuardFallbackDetectsMutation(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("before"), 0644)

	result, err := GuardFallback(dir, func() error {
		os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("after"), 0644)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Violated {
		t.Fatal("expected violation")
	}
	if len(result.UnexpectedDeltas) != 1 || result.UnexpectedDeltas[0].Path != "tracked.txt" {
		t.Fatalf("expected tracked.txt delta, got %v", result.UnexpectedDeltas)
	}
}
