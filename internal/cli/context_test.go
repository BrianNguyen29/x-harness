package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/contextcheck"
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
		"Verifier is read-only",
		"Success is the only accepted outcome",
		"Canonical tiers",
		"PGV is advisory-only",
		"## Fix Status Fields",
		"## Completion Candidate",
		"## Accepted Completion",
		"## Evidence Floor",
		"## Strict Evidence Provenance",
		"contract-hash:",
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

	var output struct {
		Facts []struct {
			Rule        string `json:"rule"`
			Description string `json:"description"`
		} `json:"facts"`
		Rules               []string `json:"rules"`
		FixStatus           struct {
			CompletionCard string `json:"completionCard"`
			SubagentReturn string `json:"subagentReturn"`
		} `json:"fixStatus"`
		CompletionCandidate struct {
			Claim        map[string]string `json:"claim"`
			Verification map[string]string `json:"verification"`
		} `json:"completionCandidate"`
		AcceptedCompletion struct {
			Admission        map[string]string `json:"admission"`
			AcceptanceStatus string            `json:"acceptanceStatus"`
		} `json:"acceptedCompletion"`
		EvidenceFloor struct {
			Light    struct{ Required []string `json:"required"` } `json:"light"`
			Standard struct{ Required []string `json:"required"` } `json:"standard"`
			Deep     struct{ Required []string `json:"required"` } `json:"deep"`
		} `json:"evidenceFloor"`
		StrictProvenance []string `json:"strictProvenance"`
		Hash             string   `json:"hash"`
		Markdown         string   `json:"markdown"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(output.Facts) == 0 {
		t.Fatal("expected at least one fact in JSON output")
	}
	if len(output.Rules) == 0 {
		t.Fatal("expected at least one rule in JSON output")
	}
	if output.Hash == "" {
		t.Fatal("expected hash in JSON output")
	}
	if output.Markdown == "" {
		t.Fatal("expected markdown in JSON output")
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
	valid, note := contextcheck.ValidateManagedBlock(string(updatedContent))
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

func TestContextGCCheckFresh(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skipf("could not find repo root: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--check", "--root", repoRoot}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "passed") {
		t.Fatalf("expected passed message, got stdout: %q", stdout.String())
	}
}

func TestContextGCCheckStaleHash(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	block := generateManagedBlock()
	block = strings.Replace(block, "<!-- context-hash: ", "<!-- context-hash: deadbeef", 1)
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\n"+block+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "stale") {
		t.Fatalf("expected stale hash message, got stderr: %q", stderr.String())
	}
}

func TestContextGCCheckMissingBlock(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\nSome content.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--check", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "missing managed context block") {
		t.Fatalf("expected missing block message, got stderr: %q", stderr.String())
	}
}

func TestContextGCCheckJSON(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--check", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false, got %v", result["ok"])
	}
	findings, ok := result["findings"].([]any)
	if !ok || len(findings) == 0 {
		t.Fatalf("expected non-empty findings, got %v", result["findings"])
	}
}

func TestContextGCWriteUpdatesStaleBlock(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	block := generateManagedBlock()
	block = strings.Replace(block, "<!-- context-hash: ", "<!-- context-hash: deadbeef", 1)
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\n"+block+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--write", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "refreshed") {
		t.Fatalf("expected refreshed message, got stdout: %q", stdout.String())
	}

	updatedContent, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	valid, note := contextcheck.ValidateManagedBlock(string(updatedContent))
	if !valid {
		t.Fatalf("expected updated block to be valid: %s", note)
	}
}

func TestContextGCWriteIdempotentFresh(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	block := generateManagedBlock()
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\n"+block+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--write", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "up-to-date") {
		t.Fatalf("expected up-to-date message, got stdout: %q", stdout.String())
	}

	// Verify file content is unchanged
	updatedContent, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(updatedContent) != "# AGENTS\n\n"+block+"\n" {
		t.Fatal("expected AGENTS.md content to be unchanged for fresh block")
	}
}

func TestContextGCWritePreservesSurroundingContent(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	before := "# My Project\n\nCustom instructions here.\n\n"
	after := "\n\n## Notes\n\nKeep this section.\n"
	existingBlock := generateManagedBlock()
	existingBlock = strings.Replace(existingBlock, "<!-- context-hash: ", "<!-- context-hash: deadbeef", 1)
	content := before + existingBlock + after
	if err := os.WriteFile(agentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--write", "--root", tmpDir}, &stdout, &stderr)
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
	valid, note := contextcheck.ValidateManagedBlock(updatedStr)
	if !valid {
		t.Fatalf("expected updated block to be valid: %s", note)
	}
}

func TestContextGCWriteJSON(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--write", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %v", result["ok"])
	}
	if result["changed"] != true {
		t.Fatalf("expected changed=true, got %v", result["changed"])
	}
	if result["context_hash"] == "" {
		t.Fatalf("expected context_hash to be present")
	}
}

func TestContextGCWriteJSONIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	block := generateManagedBlock()
	if err := os.WriteFile(agentsPath, []byte("# AGENTS\n\n"+block+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "gc", "--write", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %v", result["ok"])
	}
	if result["changed"] != false {
		t.Fatalf("expected changed=false, got %v", result["changed"])
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
	existing := before + contextcheck.ManagedBegin + "\nold content\n" + contextcheck.ManagedEnd + after

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
	valid, note := contextcheck.ValidateManagedBlock(updatedStr)
	if !valid {
		t.Fatalf("expected updated block to be valid: %s", note)
	}
}

func TestContextManifestWriteAndCheckFresh(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmpDir, "manifest.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "write", "--files", f1 + "," + f2, "--out", manifestPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "wrote manifest") {
		t.Fatalf("expected write confirmation, got stdout: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"context", "manifest", "check", "--manifest", manifestPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "all entries fresh") {
		t.Fatalf("expected fresh message, got stdout: %q", stdout.String())
	}
}

func TestContextManifestWriteAndCheckStale(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmpDir, "manifest.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "write", "--files", f1, "--out", manifestPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	// Modify file to make it stale
	if err := os.WriteFile(f1, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"context", "manifest", "check", "--manifest", manifestPath}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "stale") {
		t.Fatalf("expected stale message, got combined: %q", combined)
	}
}

func TestContextManifestCheckDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmpDir, "manifest.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "write", "--files", f1, "--out", manifestPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	if err := os.Remove(f1); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"context", "manifest", "check", "--manifest", manifestPath}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "stale") {
		t.Fatalf("expected stale message for deleted file, got combined: %q", combined)
	}
}

func TestContextManifestWriteJSON(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmpDir, "manifest.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "write", "--files", f1, "--out", manifestPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %v", result["ok"])
	}
}

func TestContextManifestCheckJSON(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmpDir, "manifest.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "write", "--files", f1, "--out", manifestPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"context", "manifest", "check", "--manifest", manifestPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %v", result["ok"])
	}
	stale, ok := result["stale"].([]any)
	if !ok || len(stale) != 0 {
		t.Fatalf("expected empty stale list, got %v", result["stale"])
	}
}

func TestContextManifestCheckMissingManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "missing.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "check", "--manifest", manifestPath}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "cannot read manifest") && !strings.Contains(combined, "Error") {
		t.Fatalf("expected error message, got combined: %q", combined)
	}
}

func TestContextManifestWriteMissingFilesFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "write"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
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
