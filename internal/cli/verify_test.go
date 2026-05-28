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
	cardSrc := filepath.Join("..", "..", "examples", "golden", "regression", "success-light", "completion-card.yaml")
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
	cardSrc := filepath.Join("..", "..", "examples", "golden", "regression", "success-light", "completion-card.yaml")
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

func TestVerifySubagentReturnPass(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	schemaSrc := filepath.Join("..", "..", "schemas", "subagent-return.schema.json")
	schemaDst := filepath.Join(tmpDir, "schemas", "subagent-return.schema.json")
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

	fixtureSrc := filepath.Join("..", "..", "packages", "cli", "tests", "fixtures", "subagent-pass.yaml")
	fixtureDst := filepath.Join(tmpDir, "subagent-pass.yaml")
	fixtureData, err := os.ReadFile(fixtureSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixtureDst, fixtureData, 0644); err != nil {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--subagent-return", "subagent-pass.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok, got outcome=%s status=%s errors=%v", result.AdmissionOutcome, result.AcceptanceStatus, result.AdmissionErrors)
	}
	if result.Tier != "standard" {
		t.Fatalf("expected tier standard, got %s", result.Tier)
	}
}

func TestVerifySubagentReturnInvalidSchema(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	schemaSrc := filepath.Join("..", "..", "schemas", "subagent-return.schema.json")
	schemaDst := filepath.Join(tmpDir, "schemas", "subagent-return.schema.json")
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

	if err := os.WriteFile(filepath.Join(tmpDir, "bad.yaml"), []byte("not_an_object: true\n"), 0644); err != nil {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--subagent-return", "bad.yaml", "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.SchemaError == "" {
		t.Fatal("expected schema error")
	}
}

func TestVerifyTraceWritesEvent(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardSrc := filepath.Join("..", "..", "examples", "golden", "regression", "success-light", "completion-card.yaml")
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

	traceDir := filepath.Join(tmpDir, "traces")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--trace", "--trace-dir", traceDir, "--json"}, &stdout, &stderr)
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

	eventsFile := filepath.Join(traceDir, "events.jsonl")
	if _, err := os.Stat(eventsFile); os.IsNotExist(err) {
		t.Fatalf("expected events.jsonl to exist in %s", traceDir)
	}

	events, err := ReadTrace(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.getString("event_type") != "verify_completed" {
		t.Fatalf("expected event_type verify_completed, got %s", event.getString("event_type"))
	}
	if event.getString("outcome") != "success" {
		t.Fatalf("expected outcome success, got %s", event.getString("outcome"))
	}
	if event.getString("acceptance_status") != "accepted" {
		t.Fatalf("expected acceptance_status accepted, got %s", event.getString("acceptance_status"))
	}

	chainResult := VerifyTraceChain(events)
	if !chainResult.Valid {
		t.Fatalf("expected valid chain, got broken at index %v", chainResult.FirstBrokenIndex)
	}
}

func TestVerifyJSONWithheldIncludesTaxonomy(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardSrc := filepath.Join("..", "..", "examples", "golden", "regression", "blocked-missing-evidence", "completion-card.yaml")
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d", code)
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason in JSON output")
	}
	if result.WithheldReason.FailureClass == "" {
		t.Fatal("expected failure_class in withheld_reason")
	}
	if result.WithheldReason.FailureStage == "" {
		t.Fatal("expected failure_stage in withheld_reason")
	}
	if result.WithheldReason.Recoverability == "" {
		t.Fatal("expected recoverability in withheld_reason")
	}
	if result.WithheldReason.NextAction == "" {
		t.Fatal("expected next_action in withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate == "" {
		t.Fatal("expected blocking_predicate in withheld_reason")
	}
}

func TestVerifyTraceEventIncludesTaxonomy(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardSrc := filepath.Join("..", "..", "examples", "golden", "regression", "blocked-missing-evidence", "completion-card.yaml")
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

	traceDir := filepath.Join(tmpDir, "traces")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--trace", "--trace-dir", traceDir, "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d", code)
	}

	events, err := ReadTrace(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.getString("blocking_predicate") == "" {
		t.Fatalf("expected blocking_predicate in trace event, got empty")
	}
	if event.getString("blocked_reason_class") == "" {
		t.Fatalf("expected blocked_reason_class in trace event, got empty")
	}
}

func TestVerifySubagentReturnContradiction(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	schemaSrc := filepath.Join("..", "..", "schemas", "subagent-return.schema.json")
	schemaDst := filepath.Join(tmpDir, "schemas", "subagent-return.schema.json")
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

	fixtureSrc := filepath.Join("..", "..", "packages", "cli", "tests", "fixtures", "subagent-contradiction.yaml")
	fixtureDst := filepath.Join(tmpDir, "subagent-contradiction.yaml")
	fixtureData, err := os.ReadFile(fixtureSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixtureDst, fixtureData, 0644); err != nil {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--subagent-return", "subagent-contradiction.yaml", "--tier", "light", "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitError, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}

	hasContradiction := false
	for _, e := range result.AdmissionErrors {
		if strings.Contains(e, "canonical contradiction") && strings.Contains(e, "partial") {
			hasContradiction = true
		}
	}
	if !hasContradiction {
		t.Fatalf("expected contradiction error, got: %v", result.AdmissionErrors)
	}
}
