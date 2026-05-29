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
		t.Fatalf("expected no dead links in real repo, got: %v", dead)
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

func TestCheckDeadLinksRepoWide(t *testing.T) {
	tmpDir := t.TempDir()
	areas := []string{"docs", "examples", "tests", "adapters", "templates", "packages/cli"}
	for _, d := range areas {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	content := "# Doc\n\nSee [missing](MISSING_FILE.md) for details.\n"
	files := []string{
		"docs/test.md",
		"examples/test.md",
		"tests/test.md",
		"adapters/test.md",
		"templates/test.md",
		"packages/cli/test.md",
		"root.md",
	}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	dead := CheckDeadLinks(tmpDir)
	if len(dead) == 0 {
		t.Fatal("expected dead link detection to find broken links in repo-wide markdown")
	}
	for _, area := range []string{"docs", "examples", "tests", "adapters", "templates", "packages/cli", "root.md"} {
		found := false
		for _, d := range dead {
			if strings.Contains(d, "MISSING_FILE.md") && strings.Contains(d, area) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected dead link in %s, got: %v", area, dead)
		}
	}
}

func TestValidateManagedBlockGenericFresh(t *testing.T) {
	body := "## Generated Contract\n\n- Rule one.\n- Rule two."
	hash := ContextHash(body)
	begin := "<!-- BEGIN MANAGED BLOCK: test -->"
	end := "<!-- END MANAGED BLOCK: test -->"
	block := begin + "\n<!-- hash: " + hash + " -->\n\n" + body + "\n\n" + end
	valid, note := ValidateManagedBlockGeneric(block, begin, end, "<!-- hash: ")
	if !valid {
		t.Fatalf("expected fresh block to be valid, got: %s", note)
	}
	if !strings.Contains(note, "fresh") {
		t.Fatalf("expected note to mention fresh, got: %s", note)
	}
}

func TestValidateManagedBlockGenericCRLF(t *testing.T) {
	body := "## Generated Contract\n\n- Rule one.\n- Rule two."
	hash := ContextHash(body)
	begin := "<!-- BEGIN MANAGED BLOCK: test -->"
	end := "<!-- END MANAGED BLOCK: test -->"
	// Block uses CRLF line endings, but hash was computed from LF body
	block := begin + "\r\n<!-- hash: " + hash + " -->\r\n\r\n" + body + "\r\n\r\n" + end
	valid, note := ValidateManagedBlockGeneric(block, begin, end, "<!-- hash: ")
	if !valid {
		t.Fatalf("expected CRLF block to validate against LF hash, got: %s", note)
	}
	if !strings.Contains(note, "fresh") {
		t.Fatalf("expected note to mention fresh, got: %s", note)
	}
}

func TestValidateManagedBlockGenericStaleHash(t *testing.T) {
	body := "## Generated Contract\n\n- Rule one.\n- Rule two."
	begin := "<!-- BEGIN MANAGED BLOCK: test -->"
	end := "<!-- END MANAGED BLOCK: test -->"
	block := begin + "\n<!-- hash: deadbeef -->\n\n" + body + "\n\n" + end
	valid, note := ValidateManagedBlockGeneric(block, begin, end, "<!-- hash: ")
	if valid {
		t.Fatal("expected stale hash to fail")
	}
	if !strings.Contains(note, "stale") {
		t.Fatalf("expected note about stale hash, got: %s", note)
	}
}

func TestValidateManagedBlockGenericMissingBlock(t *testing.T) {
	valid, note := ValidateManagedBlockGeneric("no block", "<!-- BEGIN -->", "<!-- END -->", "<!-- hash: ")
	if valid {
		t.Fatal("expected missing block to fail")
	}
	if !strings.Contains(note, "missing managed block") {
		t.Fatalf("expected note about missing block, got: %s", note)
	}
}

func TestValidateRegistryRealRepo(t *testing.T) {
	root := filepath.Join("..", "..")
	failures, err := ValidateRegistry(root)
	if err != nil {
		t.Fatalf("expected registry to be readable: %v", err)
	}
	if len(failures) > 0 {
		t.Fatalf("expected all registered blocks to be valid, got failures: %v", failures)
	}
}

func TestValidateRegistryMissingRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ValidateRegistry(tmpDir)
	if err == nil {
		t.Fatal("expected error when registry is missing")
	}
}

func TestValidateRegistryStaleBlock(t *testing.T) {
	tmpDir := t.TempDir()
	registryDir := filepath.Join(tmpDir, ".x-harness")
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		t.Fatalf("mkdir .x-harness: %v", err)
	}

	begin := "<!-- BEGIN MANAGED BLOCK: test -->"
	end := "<!-- END MANAGED BLOCK: test -->"
	content := "# File\n\n" + begin + "\n<!-- hash: deadbeef -->\n\nBody\n\n" + end + "\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write test.md: %v", err)
	}

	registry := `version: "1"
blocks:
  - path: test.md
    type: contract
    begin_marker: "<!-- BEGIN MANAGED BLOCK: test -->"
    end_marker: "<!-- END MANAGED BLOCK: test -->"
    hash_prefix: "<!-- hash: "
`
	if err := os.WriteFile(filepath.Join(registryDir, "managed-blocks.yaml"), []byte(registry), 0644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	failures, err := ValidateRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(failures) == 0 {
		t.Fatal("expected stale block to fail validation")
	}
	if !strings.Contains(failures[0], "stale") {
		t.Fatalf("expected failure to mention stale, got: %s", failures[0])
	}
}

func TestCheckDeadLinksExcludesGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	areas := []string{"node_modules", "dist", "coverage", "build", "vendor", ".x-harness", "docs"}
	for _, d := range areas {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	content := "# Doc\n\nSee [missing](MISSING_FILE.md) for details.\n"
	excludedFiles := []string{
		"node_modules/test.md",
		"dist/test.md",
		"coverage/test.md",
		"build/test.md",
		"vendor/test.md",
		".x-harness/test.md",
	}
	for _, f := range excludedFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "docs", "existing.md"), []byte("# Existing\n"), 0644); err != nil {
		t.Fatalf("write existing.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "docs", "broken.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write broken.md: %v", err)
	}

	dead := CheckDeadLinks(tmpDir)
	for _, d := range dead {
		if strings.Contains(d, "MISSING_FILE.md") {
			for _, ex := range excludedFiles {
				if strings.Contains(d, ex) {
					t.Fatalf("unexpected dead link from excluded path: %s", d)
				}
			}
		}
	}

	foundDocs := false
	for _, d := range dead {
		if strings.Contains(d, "docs/broken.md") && strings.Contains(d, "MISSING_FILE.md") {
			foundDocs = true
			break
		}
	}
	if !foundDocs {
		t.Fatalf("expected docs broken link to be detected, got: %v", dead)
	}
}
