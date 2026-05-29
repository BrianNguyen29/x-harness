package contextcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalContextNotEmpty(t *testing.T) {
	ctx := CanonicalContext()
	if strings.TrimSpace(ctx) == "" {
		t.Fatal("canonical context should not be empty")
	}
	if !strings.Contains(ctx, "light") {
		t.Fatal("canonical context should mention light tier")
	}
}

func TestContextHashDeterministic(t *testing.T) {
	h1 := ContextHash("test")
	h2 := ContextHash("test")
	if h1 != h2 {
		t.Fatalf("hash should be deterministic: %s vs %s", h1, h2)
	}
	if h1 == ContextHash("different") {
		t.Fatal("different content should produce different hash")
	}
}

func TestExtractManagedBlock(t *testing.T) {
	content := "# Header\n" + ManagedBegin + "\ncontent\n" + ManagedEnd + "\nfooter"
	block, ok := ExtractManagedBlock(content)
	if !ok {
		t.Fatal("expected to extract managed block")
	}
	if !strings.Contains(block, ManagedBegin) {
		t.Fatal("extracted block should contain begin marker")
	}
	if !strings.Contains(block, ManagedEnd) {
		t.Fatal("extracted block should contain end marker")
	}
}

func TestExtractManagedBlockMissing(t *testing.T) {
	_, ok := ExtractManagedBlock("no managed block here")
	if ok {
		t.Fatal("should not extract block from plain content")
	}
}

func TestValidateManagedBlockFresh(t *testing.T) {
	ctx := CanonicalContext()
	hash := ContextHash(ctx)
	block := ManagedBegin + "\n<!-- generated-by: x-harness -->\n<!-- context-hash: " + hash + " -->\n\n" + ctx + "\n\n" + ManagedEnd
	valid, note := ValidateManagedBlock(block)
	if !valid {
		t.Fatalf("expected fresh block to be valid, got: %s", note)
	}
	if !strings.Contains(note, "fresh") {
		t.Fatalf("expected note to mention fresh, got: %s", note)
	}
}

func TestValidateManagedBlockStaleHash(t *testing.T) {
	ctx := CanonicalContext()
	block := ManagedBegin + "\n<!-- generated-by: x-harness -->\n<!-- context-hash: deadbeefdeadbeef -->\n\n" + ctx + "\n\n" + ManagedEnd
	valid, note := ValidateManagedBlock(block)
	if valid {
		t.Fatal("expected stale hash to fail")
	}
	if !strings.Contains(note, "stale") {
		t.Fatalf("expected note about stale hash, got: %s", note)
	}
}

func TestValidateManagedBlockStaleBody(t *testing.T) {
	ctx := CanonicalContext()
	hash := ContextHash(ctx)
	modifiedCtx := strings.Replace(ctx, "admitted, not claimed", "admitted, not claimed. (modified)", 1)
	block := ManagedBegin + "\n<!-- generated-by: x-harness -->\n<!-- context-hash: " + hash + " -->\n\n" + modifiedCtx + "\n\n" + ManagedEnd
	valid, note := ValidateManagedBlock(block)
	if valid {
		t.Fatal("expected stale body to fail")
	}
	if !strings.Contains(note, "differs") {
		t.Fatalf("expected note about body differs, got: %s", note)
	}
}

func TestValidateManagedBlockMissingHash(t *testing.T) {
	ctx := CanonicalContext()
	block := ManagedBegin + "\n<!-- generated-by: x-harness -->\n\n" + ctx + "\n\n" + ManagedEnd
	valid, note := ValidateManagedBlock(block)
	if valid {
		t.Fatal("expected missing hash to fail")
	}
	if !strings.Contains(note, "context-hash") {
		t.Fatalf("expected note about missing hash, got: %s", note)
	}
}

func TestValidateManagedBlockMissingBlock(t *testing.T) {
	valid, note := ValidateManagedBlock("no managed block")
	if valid {
		t.Fatal("expected missing block to fail")
	}
	if !strings.Contains(note, "missing managed context block") {
		t.Fatalf("expected note about missing block, got: %s", note)
	}
}

func TestValidateManagedBlockFromFile(t *testing.T) {
	// Use the real repo AGENTS.md
	path := filepath.Join("..", "..", "AGENTS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("AGENTS.md not readable: %v", err)
	}
	valid, note := ValidateManagedBlock(string(b))
	if !valid {
		t.Fatalf("expected real AGENTS.md to pass validation: %s", note)
	}
}

func TestCheckDeadLinksNone(t *testing.T) {
	root := filepath.Join("..", "..")
	dead := CheckDeadLinks(root)
	if len(dead) > 0 {
		t.Fatalf("expected no dead links in real repo docs, got: %v", dead)
	}
}

func TestCheckDeadLinksDetectsBroken(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "# Doc\n\nSee [missing](MISSING_FILE.md) for details.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write test.md: %v", err)
	}

	dead := CheckDeadLinks(tmpDir)
	if len(dead) == 0 {
		t.Fatal("expected dead link detection to find MISSING_FILE.md")
	}
	found := false
	for _, d := range dead {
		if strings.Contains(d, "MISSING_FILE.md") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dead link note to contain MISSING_FILE.md, got: %v", dead)
	}
}
