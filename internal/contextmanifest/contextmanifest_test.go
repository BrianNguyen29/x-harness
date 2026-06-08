package contextmanifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAndCheckFresh(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Generate([]string{f1, f2}, tmpDir, "test")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if m.Version != "1" {
		t.Fatalf("expected version 1, got %s", m.Version)
	}
	if len(m.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m.Entries))
	}

	stale, err := Check(m, tmpDir)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("expected all fresh, got stale: %v", stale)
	}
}

func TestCheckModifiedStale(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Generate([]string{f1}, tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Modify after generation
	if err := os.WriteFile(f1, []byte("goodbye"), 0644); err != nil {
		t.Fatal(err)
	}

	stale, err := Check(m, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0] != "a.txt" {
		t.Fatalf("expected a.txt stale, got %v", stale)
	}
}

func TestCheckDeletedStale(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Generate([]string{f1}, tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Delete after generation
	if err := os.Remove(f1); err != nil {
		t.Fatal(err)
	}

	stale, err := Check(m, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0] != "a.txt" {
		t.Fatalf("expected a.txt stale, got %v", stale)
	}
}

func TestCheckExtraUntrackedIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Generate([]string{f1}, tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Add extra untracked file
	f2 := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(f2, []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	stale, err := Check(m, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("expected no stale entries, got %v", stale)
	}
}

func TestRoundTripYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "manifest.yaml")

	m := &Manifest{
		Version: "1",
		Entries: []Entry{
			{Path: "x.md", SHA256: "abc123", ReadAt: "2026-01-01T00:00:00Z", Reason: "r"},
		},
	}
	if err := Write(m, path); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	readBack, err := Read(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if readBack.Version != m.Version {
		t.Fatalf("expected version %s, got %s", m.Version, readBack.Version)
	}
	if len(readBack.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(readBack.Entries))
	}
	if readBack.Entries[0].Path != "x.md" {
		t.Fatalf("expected path x.md, got %s", readBack.Entries[0].Path)
	}
	if readBack.Entries[0].SHA256 != "abc123" {
		t.Fatalf("expected sha256 abc123, got %s", readBack.Entries[0].SHA256)
	}
}

func TestValidateMissingVersion(t *testing.T) {
	m := &Manifest{Entries: []Entry{{Path: "x", SHA256: "y"}}}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected version required error, got %v", err)
	}
}

func TestValidateUnsupportedVersion(t *testing.T) {
	m := &Manifest{Version: "99", Entries: []Entry{{Path: "x", SHA256: "y"}}}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported version error, got %v", err)
	}
}

func TestValidateMissingPath(t *testing.T) {
	m := &Manifest{Version: "1", Entries: []Entry{{Path: "", SHA256: "y"}}}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("expected path required error, got %v", err)
	}
}

func TestValidateMissingSHA256(t *testing.T) {
	m := &Manifest{Version: "1", Entries: []Entry{{Path: "x", SHA256: ""}}}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "sha256 is required") {
		t.Fatalf("expected sha256 required error, got %v", err)
	}
}

func TestValidateDuplicatePath(t *testing.T) {
	m := &Manifest{Version: "1", Entries: []Entry{
		{Path: "x", SHA256: "a"},
		{Path: "x", SHA256: "b"},
	}}
	if err := Validate(m); err == nil || !strings.Contains(err.Error(), "duplicate path") {
		t.Fatalf("expected duplicate path error, got %v", err)
	}
}

func TestEmptyManifestNonStale(t *testing.T) {
	m := &Manifest{Version: "1", Entries: []Entry{}}
	stale, err := Check(m, t.TempDir())
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("expected empty manifest non-stale, got %v", stale)
	}
}

func TestGenerateDeterministicOrder(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(f1, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	m1, err := Generate([]string{f1, f2}, tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Generate([]string{f1, f2}, tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	if len(m1.Entries) != len(m2.Entries) {
		t.Fatal("entry count mismatch")
	}
	for i := range m1.Entries {
		if m1.Entries[i].Path != m2.Entries[i].Path {
			t.Fatalf("path mismatch at %d: %s vs %s", i, m1.Entries[i].Path, m2.Entries[i].Path)
		}
		if m1.Entries[i].SHA256 != m2.Entries[i].SHA256 {
			t.Fatalf("sha256 mismatch at %d", i)
		}
	}
}

func TestReadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(path, []byte("not: [ valid yaml {{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

func TestGenerateSkipsEmptyPaths(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Generate([]string{f1, "", "  "}, tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry after skipping empties, got %d", len(m.Entries))
	}
}
