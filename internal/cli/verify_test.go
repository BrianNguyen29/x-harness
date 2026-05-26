package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyStrictBlocksMutationInjectionInsideRoot(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cardSrc := filepath.Join("..", "..", "examples", "golden", "success-light", "completion-card.yaml")
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	srcData, err := os.ReadFile(cardSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cardDst, srcData, 0644); err != nil {
		t.Fatal(err)
	}
	schemaSrc := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	schemaDst := filepath.Join(tmpDir, "schemas", "completion-card.schema.json")
	if err := os.MkdirAll(filepath.Dir(schemaDst), 0755); err != nil {
		t.Fatal(err)
	}
	schemaData, err := os.ReadFile(schemaSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaDst, schemaData, 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	t.Setenv("X_HARNESS_ENABLE_TEST_HOOKS", "1")
	t.Setenv("X_HARNESS_TEST_INJECT_MUTATION", "unexpected.txt")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--strict", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)

	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.AdmissionOutcome != "blocked" {
		t.Fatalf("expected blocked, got %s", result.AdmissionOutcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected withheld, got %s", result.AcceptanceStatus)
	}
	if result.MutationGuard == nil || !result.MutationGuard.Violated {
		t.Fatal("expected mutation guard violated")
	}
}

func TestVerifyRejectsMutationInjectionOutsideRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	if err := exec.Command("git", "-C", tmpDir, "init").Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cardSrc := filepath.Join("..", "..", "examples", "golden", "success-light", "completion-card.yaml")
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	srcData, err := os.ReadFile(cardSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cardDst, srcData, 0644); err != nil {
		t.Fatal(err)
	}
	schemaSrc := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	schemaDst := filepath.Join(tmpDir, "schemas", "completion-card.schema.json")
	if err := os.MkdirAll(filepath.Dir(schemaDst), 0755); err != nil {
		t.Fatal(err)
	}
	schemaData, err := os.ReadFile(schemaSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaDst, schemaData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", "completion-card.yaml", "go.mod").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	outsidePath := filepath.Join(tmpDir, "..", "should-not-be-created.txt")
	os.Remove(outsidePath)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	t.Setenv("X_HARNESS_ENABLE_TEST_HOOKS", "1")
	t.Setenv("X_HARNESS_TEST_INJECT_MUTATION", outsidePath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--mutation-guard", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}

	if _, err := os.Stat(outsidePath); !os.IsNotExist(err) {
		t.Fatalf("outside path should not have been created: %s", outsidePath)
	}
	if !strings.Contains(stderr.String(), "test hook: rejected injection path") {
		t.Fatalf("expected rejection message in stderr, got: %s", stderr.String())
	}
}
