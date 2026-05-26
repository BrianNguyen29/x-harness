package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootFindsCurrentRepo(t *testing.T) {
	root, err := FindRoot("")
	if err != nil {
		t.Fatalf("expected to find root from cwd, got error: %v", err)
	}

	// The repository should contain go.mod.
	goMod := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		t.Fatalf("expected go.mod at %s: %v", goMod, err)
	}
}

func TestFindRootFromNestedDirectory(t *testing.T) {
	// Start from a known nested directory.
	start := filepath.Join("..", "..", "internal", "cli")
	root, err := FindRoot(start)
	if err != nil {
		t.Fatalf("expected to find root from nested path, got error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("expected go.mod in found root: %v", err)
	}
}

func TestFindRootReturnsErrorWhenNotFound(t *testing.T) {
	// Use a temporary directory with no markers.
	tmp := t.TempDir()
	_, err := FindRoot(tmp)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}
