package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextContractPlain(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "--contract"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	out := stdout.String()
	for _, phrase := range []string{
		"Completion is admitted, not claimed",
		"verifier is read-only",
		"Success is the only accepted outcome",
		"Canonical tiers",
		"PGV is advisory-only",
	} {
		if !strings.Contains(out, phrase) {
			t.Fatalf("expected output to contain %q, got:\n%s", phrase, out)
		}
	}
}

func TestContextContractJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "--contract", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var contract struct {
		Facts []struct {
			Rule        string `json:"rule"`
			Description string `json:"description"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &contract); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(contract.Facts) == 0 {
		t.Fatal("expected at least one fact in JSON output")
	}
}

func TestContextUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "--unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown context subcommand") {
		t.Fatalf("expected unknown subcommand error, got %q", stderr.String())
	}
}

func TestContextContractCoreFactsCount(t *testing.T) {
	contract := CoreContract()
	if len(contract.Facts) != 5 {
		t.Fatalf("expected 5 core contract facts, got %d", len(contract.Facts))
	}
	expectedRules := []string{
		"completion_admitted_not_claimed",
		"verifier_read_only",
		"success_only_accepted",
		"canonical_tiers",
		"pgv_advisory_only",
	}
	for i, rule := range expectedRules {
		if contract.Facts[i].Rule != rule {
			t.Fatalf("expected fact[%d].Rule == %q, got %q", i, rule, contract.Facts[i].Rule)
		}
	}
}

func TestContextSyncCheckFresh(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skipf("could not find repo root: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--check", "--root", repoRoot}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid") {
		t.Fatalf("expected fresh message, got stdout: %q", stdout.String())
	}
}

func TestContextSyncCheckStaleHash(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	block := generateManagedBlock()
	block = strings.Replace(block, "<!-- context-hash: ", "<!-- context-hash: deadbeef", 1)
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\n"+block+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "stale") {
		t.Fatalf("expected stale hash message, got stderr: %q", stderr.String())
	}
}

func TestContextSyncCheckStaleBody(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	block := generateManagedBlock()
	// Modify body but keep old hash by not regenerating
	block = strings.Replace(block, "- Completion is admitted, not claimed.", "- Completion is admitted, not claimed. (modified)", 1)
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\n"+block+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "stale") && !strings.Contains(combined, "differs") {
		t.Fatalf("expected stale/differs message, got stdout: %q stderr: %q", stdout.String(), stderr.String())
	}
}

func TestContextSyncCheckMissingBlock(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\nSome content.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "missing managed context block") {
		t.Fatalf("expected missing block message, got stderr: %q", stderr.String())
	}
}

func TestContextSyncWriteUpdatesBlock(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\nSome content.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--write", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "refreshed") {
		t.Fatalf("expected refreshed message, got stdout: %q", stdout.String())
	}

	// Verify the block is now fresh
	updatedContent, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	valid, note := validateManagedBlock(string(updatedContent))
	if !valid {
		t.Fatalf("expected updated block to be valid: %s", note)
	}
}

func TestContextSyncCheckJSON(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--check", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["valid"] != false {
		t.Fatalf("expected valid=false, got %v", result["valid"])
	}
}

func TestContextSyncWriteJSON(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--write", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["updated"] != true {
		t.Fatalf("expected updated=true, got %v", result["updated"])
	}
	if result["context_hash"] == "" {
		t.Fatalf("expected context_hash to be present")
	}
}

func TestContextSyncMissingAgentsMd(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("expected not found message, got stderr: %q", stderr.String())
	}
}

func TestGenerateManagedBlockDeterministic(t *testing.T) {
	a := generateManagedBlock()
	b := generateManagedBlock()
	if a != b {
		t.Fatalf("expected deterministic output, got different blocks")
	}
}

func TestInjectManagedBlockPreservesSurroundingContent(t *testing.T) {
	before := "# Header\n\nSome intro text.\n\n"
	after := "\n\n# Footer\n\nSome outro text.\n"
	existing := before + managedBegin + "\nold content\n" + managedEnd + after

	block := generateManagedBlock()
	updated := injectManagedBlock(existing, block)

	if !strings.HasPrefix(updated, before) {
		t.Fatalf("expected prefix to be preserved, got:\n%s", updated)
	}
	if !strings.HasSuffix(updated, after) {
		t.Fatalf("expected suffix to be preserved, got:\n%s", updated)
	}
	if !strings.Contains(updated, block) {
		t.Fatalf("expected new block to be injected, got:\n%s", updated)
	}
}

func TestContextSyncWritePreservesSurroundingContent(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	before := "# My Project\n\nCustom instructions here.\n\n"
	after := "\n\n## Notes\n\nKeep this section.\n"
	existingBlock := generateManagedBlock()
	content := before + existingBlock + after
	if err := os.WriteFile(agentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "sync", "--write", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	updated, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedStr := string(updated)
	if !strings.Contains(updatedStr, before) {
		t.Fatalf("expected content before block to be preserved")
	}
	if !strings.Contains(updatedStr, after) {
		t.Fatalf("expected content after block to be preserved")
	}
	valid, note := validateManagedBlock(updatedStr)
	if !valid {
		t.Fatalf("expected updated block to be valid: %s", note)
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		for _, marker := range []string{".git", "go.mod", "X_HARNESS.md", "AGENTS.md"} {
			path := filepath.Join(wd, marker)
			if _, err := os.Stat(path); err == nil {
				return wd, nil
			}
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", os.ErrNotExist
}
