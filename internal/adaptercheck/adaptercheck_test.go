package adaptercheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindManagedBlocks(t *testing.T) {
	content := `<!-- BEGIN X-HARNESS MANAGED CONTRACT: test -->
content line
<!-- END X-HARNESS MANAGED CONTRACT: test -->
`
	blocks := FindManagedBlocks(content)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].ID != "test" {
		t.Fatalf("expected id test, got %s", blocks[0].ID)
	}
	if len(blocks[0].BodyLines) != 1 {
		t.Fatalf("expected 1 body line, got %d", len(blocks[0].BodyLines))
	}
}

func TestFindManagedBlocksNoMatch(t *testing.T) {
	blocks := FindManagedBlocks("plain text")
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestExtractMarkerID(t *testing.T) {
	id := ExtractMarkerID("<!-- BEGIN X-HARNESS MANAGED CONTRACT: my-id -->", "<!-- BEGIN X-HARNESS MANAGED CONTRACT:", "-->")
	if id != "my-id" {
		t.Fatalf("expected my-id, got %s", id)
	}
}

func TestValidateManagedBlockPass(t *testing.T) {
	body := "## Contract\n\n- Rule 1"
	block := ManagedBlock{
		ID:        "test",
		BodyLines: []string{"<!-- contract-hash: " + ComputeContractHash(body) + " -->", "## Contract", "", "- Rule 1"},
	}
	check := ValidateManagedBlock(block)
	if check.Status != "passed" {
		t.Fatalf("expected passed, got %s: %s", check.Status, check.Note)
	}
}

func TestValidateManagedBlockMissingHash(t *testing.T) {
	block := ManagedBlock{
		ID:        "test",
		BodyLines: []string{"no hash here"},
	}
	check := ValidateManagedBlock(block)
	if check.Status != "failed" {
		t.Fatalf("expected failed, got %s", check.Status)
	}
	if !strings.Contains(check.Note, "missing contract-hash") {
		t.Fatalf("expected missing hash note, got %s", check.Note)
	}
}

func TestValidateManagedBlockHashMismatch(t *testing.T) {
	body := "## Contract\n\n- Rule 1"
	block := ManagedBlock{
		ID:        "test",
		BodyLines: []string{"<!-- contract-hash: 0000000000000000 -->", body},
	}
	check := ValidateManagedBlock(block)
	if check.Status != "failed" {
		t.Fatalf("expected failed, got %s", check.Status)
	}
	if !strings.Contains(check.Note, "hash mismatch") {
		t.Fatalf("expected hash mismatch note, got %s", check.Note)
	}
}

func TestComputeContractHashDeterministic(t *testing.T) {
	h1 := ComputeContractHash("test")
	h2 := ComputeContractHash("test")
	if h1 != h2 {
		t.Fatalf("expected deterministic hash: %s vs %s", h1, h2)
	}
}

func TestExtractBodyForHash(t *testing.T) {
	block := ManagedBlock{
		ID:        "test",
		BodyLines: []string{"<!-- comment -->", "  actual body  "},
	}
	body := ExtractBodyForHash(block)
	if body != "actual body" {
		t.Fatalf("expected 'actual body', got %q", body)
	}
}

func TestRunDoctorOnRealRepo(t *testing.T) {
	root := filepath.Join("..", "..")
	results, ok := RunDoctor(root)
	if !ok {
		var notes []string
		for _, r := range results {
			if !r.OK {
				for _, c := range r.Checks {
					if c.Status != "passed" {
						notes = append(notes, r.Path+": "+c.Name+" "+c.Note)
					}
				}
			}
		}
		t.Fatalf("expected adapter doctor to pass on real repo: %v", notes)
	}
}

func TestRunDoctorDetectsDrift(t *testing.T) {
	tmpDir := t.TempDir()
	adaptersDir := filepath.Join(tmpDir, "adapters", "test")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatalf("failed to create temp adapters dir: %v", err)
	}

	content := `<!-- BEGIN X-HARNESS MANAGED CONTRACT: test-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: 0000000000000000 -->

## Generated Adapter Contract

- Completion is admitted, not claimed.

<!-- END X-HARNESS MANAGED CONTRACT: test-contract -->
`
	readmePath := filepath.Join(adaptersDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp readme: %v", err)
	}

	results, ok := RunDoctor(tmpDir)
	if ok {
		t.Fatal("expected drift to be detected")
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	found := false
	for _, r := range results {
		if r.Path == filepath.Join("adapters", "test", "README.md") {
			found = true
			if r.OK {
				t.Fatal("expected result to be not ok")
			}
			if len(r.Checks) == 0 {
				t.Fatal("expected at least one check")
			}
			if !strings.Contains(r.Checks[0].Note, "hash mismatch") {
				t.Fatalf("expected hash mismatch note, got: %s", r.Checks[0].Note)
			}
		}
	}
	if !found {
		t.Fatal("expected result for test README.md")
	}
}

func TestRunDoctorMissingHash(t *testing.T) {
	tmpDir := t.TempDir()
	adaptersDir := filepath.Join(tmpDir, "adapters", "test")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatalf("failed to create temp adapters dir: %v", err)
	}

	content := `<!-- BEGIN X-HARNESS MANAGED CONTRACT: test-contract -->
<!-- generated-by: x-harness -->

## Generated Adapter Contract

- Completion is admitted, not claimed.

<!-- END X-HARNESS MANAGED CONTRACT: test-contract -->
`
	readmePath := filepath.Join(adaptersDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp readme: %v", err)
	}

	results, ok := RunDoctor(tmpDir)
	if ok {
		t.Fatal("expected missing hash to fail")
	}
	found := false
	for _, r := range results {
		if r.Path == filepath.Join("adapters", "test", "README.md") {
			found = true
			if !strings.Contains(r.Checks[0].Note, "missing contract-hash") {
				t.Fatalf("expected missing contract-hash note, got: %s", r.Checks[0].Note)
			}
		}
	}
	if !found {
		t.Fatal("expected result for test README.md")
	}
}

func TestRunDoctorMissingReadme(t *testing.T) {
	tmpDir := t.TempDir()
	adaptersDir := filepath.Join(tmpDir, "adapters", "test")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatalf("failed to create temp adapters dir: %v", err)
	}

	// Write a valid managed block file but no README.md
	content := `<!-- BEGIN X-HARNESS MANAGED CONTRACT: test-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: ` + ComputeContractHash("## Contract\n\n- Rule 1") + ` -->

## Contract

- Rule 1

<!-- END X-HARNESS MANAGED CONTRACT: test-contract -->
`
	contractPath := filepath.Join(adaptersDir, "contract.md")
	if err := os.WriteFile(contractPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp contract: %v", err)
	}

	results, ok := RunDoctor(tmpDir)
	if ok {
		t.Fatal("expected missing README to fail")
	}
	found := false
	for _, r := range results {
		if strings.Contains(r.Path, filepath.Join("adapters", "test", "README.md")) {
			found = true
			if r.OK {
				t.Fatal("expected result to be not ok")
			}
			if len(r.Checks) == 0 {
				t.Fatal("expected at least one check")
			}
			if !strings.Contains(r.Checks[0].Note, "missing README.md") {
				t.Fatalf("expected missing README note, got: %s", r.Checks[0].Note)
			}
		}
	}
	if !found {
		t.Fatal("expected result for missing test README.md")
	}
}
