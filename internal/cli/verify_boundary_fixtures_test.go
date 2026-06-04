package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// copyGoldenFixture copies a directory tree from src to dst (which is
// created). Returns the absolute path to the new root. Symlinks are
// skipped; permissions are preserved best-effort.
func copyGoldenFixture(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		// Skip dot-files (e.g. .skip-ts-benchmark) but otherwise copy
		// the file content.
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// copySchemaTo copies the completion-card and context-alignment
// schemas from the repo into the temp dir under schemas/. This mirrors
// the helper used by other verify tests so the verify pipeline can
// find them.
func copySchemaTo(t *testing.T, dst string) {
	t.Helper()
	schemaDir := filepath.Join(dst, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"completion-card.schema.json", "context-alignment.schema.json"} {
		src := filepath.Join("..", "..", "schemas", name)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(schemaDir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// chdirTemp changes the working directory to dir and registers a
// cleanup to restore it.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })
}

func TestVerifyGoldenBoundaryViolationBlocks(t *testing.T) {
	// Copy the boundary-violation fixture to a temp root and verify
	// under --boundary-enforce block_all. Expect blocking with the
	// boundary_violation predicate.
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "fixture")
	copyGoldenFixture(t, filepath.Join("..", "..", "examples", "golden", "regression", "boundary-violation"), root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	copySchemaTo(t, root)
	chdirTemp(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "block_all", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok with boundary violation under block_all")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "boundary_violation" {
		t.Fatalf("expected blocking_predicate=boundary_violation, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.FailureClass != "boundary_violation" {
		t.Fatalf("expected failure_class=boundary_violation, got %s", result.WithheldReason.FailureClass)
	}
}

func TestVerifyGoldenBoundaryAllowPasses(t *testing.T) {
	// Copy the boundary-allow fixture and verify under the strictest
	// mode. Expect a clean pass: the import is permitted by the rule's
	// allow list, so no violation is raised.
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "fixture")
	copyGoldenFixture(t, filepath.Join("..", "..", "examples", "golden", "regression", "boundary-allow"), root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	copySchemaTo(t, root)
	chdirTemp(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "block_all", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with allow list, got outcome=%s status=%s errors=%v", result.AdmissionOutcome, result.AcceptanceStatus, result.AdmissionErrors)
	}
}

func TestVerifyGoldenBoundaryViolationAcceptsUnderLightLocal(t *testing.T) {
	// Copy the boundary-violation fixture and verify under
	// --profile light-local. The light-local profile is advisory
	// only, so even with a critical-severity violation the result
	// must remain accepted.
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "fixture")
	copyGoldenFixture(t, filepath.Join("..", "..", "examples", "golden", "regression", "boundary-violation"), root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	copySchemaTo(t, root)
	chdirTemp(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "light-local", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok under light-local with boundary violation, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

// TestExamplesVerifyPicksUpBoundaryFixtures ensures the new fixtures
// are discoverable by `xh examples verify` (which only validates
// schema + admission, not boundary enforcement). The test walks the
// fixture directory tree directly because `xh examples verify` is
// anchored to the repo root and cannot be redirected via chdir.
func TestExamplesVerifyPicksUpBoundaryFixtures(t *testing.T) {
	for _, name := range []string{"boundary-violation", "boundary-allow"} {
		dir := filepath.Join("..", "..", "examples", "golden", "regression", name)
		card := filepath.Join(dir, "completion-card.yaml")
		if _, err := os.Stat(card); err != nil {
			t.Fatalf("expected fixture card %s to exist: %v", card, err)
		}
		policy := filepath.Join(dir, "policies", "boundaries.yaml")
		if _, err := os.Stat(policy); err != nil {
			t.Fatalf("expected fixture policy %s to exist: %v", policy, err)
		}
	}
}
