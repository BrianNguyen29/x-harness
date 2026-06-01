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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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

	// Create referenced context file so context floor passes
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Product\n"), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	// Schema validation failure path: blocked-missing-evidence card has schema validation errors
	// that trigger schema_invalid blocking_predicate with direct FailureClass/FailureStage values
	if result.WithheldReason.BlockingPredicate != "schema_invalid" {
		t.Fatalf("expected blocking_predicate=schema_invalid, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.FailureClass != "schema_invalid" {
		t.Fatalf("expected failure_class=schema_invalid, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.FailureStage != "verify_pipeline" {
		t.Fatalf("expected failure_stage=verify_pipeline, got %s", result.WithheldReason.FailureStage)
	}
	if result.WithheldReason.Recoverability != "retry_with_fixes" {
		t.Fatalf("expected recoverability=retry_with_fixes, got %s", result.WithheldReason.Recoverability)
	}
	if result.WithheldReason.SchemaRecoverability != "manual" {
		t.Fatalf("expected schema_recoverability=manual, got %s", result.WithheldReason.SchemaRecoverability)
	}
	if result.WithheldReason.NextAction != "review_and_resubmit" {
		t.Fatalf("expected next_action=review_and_resubmit, got %s", result.WithheldReason.NextAction)
	}
	// Check schema-like fields exact values (derived from schema_invalid -> schema_or_policy_invalid)
	if result.WithheldReason.Class != "schema_or_policy_invalid" {
		t.Fatalf("expected class=schema_or_policy_invalid, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Stage != "verification" {
		t.Fatalf("expected stage=verification, got %s", result.WithheldReason.Stage)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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

func TestVerifyWorktreeAwareTrace(t *testing.T) {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--trace", "--trace-dir", traceDir, "--worktree-aware", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	events, err := ReadTrace(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	wt, ok := event["worktree"]
	if !ok {
		t.Fatal("expected worktree in trace event")
	}
	wtMap, ok := wt.(map[string]interface{})
	if !ok {
		t.Fatalf("expected worktree to be a map, got %T", wt)
	}
	if wtMap["root"] == "" {
		t.Fatal("expected worktree root")
	}
	if wtMap["commit"] == "" {
		t.Fatal("expected worktree commit")
	}
}

func TestVerifyContextFloorMissingContextAlignment(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Standard-tier card without context_alignment
	cardYAML := `schema_version: "1"
task_id: TASK-CF-MISSING-001
tier: standard
owner: alice
accountable: bob
state:
  read_set:
    - src/utils/format.ts
  write_set:
    - src/utils/format.ts
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: Added format utility
  expected_effect: Format utility is available
  measurable_signal: npm test -- format
  falsification_method: Run without fix; utility should not exist
  horizon: same_verify
evidence:
  files_changed:
    - src/utils/format.ts
  command_evidence:
    - command: npm test -- format
      exit_code: 0
claim:
  fix_status: fixed
  summary: Added format utility
  evidence:
    - description: Source file
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-floor", "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.AdmissionOutcome != "failed" {
		t.Fatalf("expected admission_outcome=failed, got %s", result.AdmissionOutcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected acceptance_status=withheld, got %s", result.AcceptanceStatus)
	}

	// Check that error or taxonomy mentions context floor / missing context
	hasContextError := false
	for _, e := range result.AdmissionErrors {
		if strings.Contains(e, "context_alignment") || strings.Contains(e, "context") {
			hasContextError = true
		}
	}
	if result.WithheldReason != nil {
		if strings.Contains(result.WithheldReason.FailureClass, "context") ||
			strings.Contains(result.WithheldReason.BlockingPredicate, "context") {
			hasContextError = true
		}
	}
	if !hasContextError {
		t.Fatalf("expected error mentioning context_alignment, got errors=%v withheld_reason=%+v", result.AdmissionErrors, result.WithheldReason)
	}
}

func TestVerifyContextFloorMissingFileRef(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy the blocked-missing-context-ref fixture
	cardSrc := filepath.Join("..", "..", "examples", "golden", "regression", "blocked-missing-context-ref", "completion-card.yaml")
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-floor", "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.AdmissionOutcome != "failed" {
		t.Fatalf("expected admission_outcome=failed, got %s", result.AdmissionOutcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected acceptance_status=withheld, got %s", result.AcceptanceStatus)
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "context_floor_blocked" {
		t.Fatalf("expected blocking_predicate=context_floor_blocked, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.FailureClass != "context_missing" {
		t.Fatalf("expected failure_class=context_missing, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.FailureStage != "context_floor" {
		t.Fatalf("expected failure_stage=context_floor, got %s", result.WithheldReason.FailureStage)
	}
	// Check new schema-like fields
	if result.WithheldReason.Class != "context_floor_blocked" {
		t.Fatalf("expected class=context_floor_blocked, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Stage != "context" {
		t.Fatalf("expected stage=context, got %s", result.WithheldReason.Stage)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
}

func TestVerifyHelpDocumentsContextFloor(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--help"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("expected exit code %d for usage, got %d. stdout: %s\nstderr: %s", ExitUsage, code, stdout.String(), stderr.String())
	}

	if !strings.Contains(stderr.String(), "--context-floor") {
		t.Fatalf("expected usage to contain --context-floor, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--strict-withheld-reason") {
		t.Fatalf("expected usage to contain --strict-withheld-reason, got: %s", stderr.String())
	}
}

func TestVerifyStrictWithheldReasonContextFloor(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy the blocked-missing-context-ref fixture
	cardSrc := filepath.Join("..", "..", "examples", "golden", "regression", "blocked-missing-context-ref", "completion-card.yaml")
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-floor", "--strict-withheld-reason", "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var rawResult map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rawResult); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	// Strict mode: withheld_reason must not contain legacy keys
	wr, ok := rawResult["withheld_reason"].(map[string]any)
	if !ok {
		t.Fatal("expected withheld_reason to be a map")
	}
	if _, has := wr["failure_class"]; has {
		t.Fatal("expected failure_class key to be absent in strict mode")
	}
	if _, has := wr["failure_stage"]; has {
		t.Fatal("expected failure_stage key to be absent in strict mode")
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	// Check schema-like fields
	if result.WithheldReason.Class != "context_floor_blocked" {
		t.Fatalf("expected class=context_floor_blocked, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Stage != "context" {
		t.Fatalf("expected stage=context, got %s", result.WithheldReason.Stage)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
	if result.WithheldReason.BlockingPredicate != "context_floor_blocked" {
		t.Fatalf("expected blocking_predicate=context_floor_blocked, got %s", result.WithheldReason.BlockingPredicate)
	}
	// recoverability must be schema enum value
	if result.WithheldReason.Recoverability == "" {
		t.Fatal("expected recoverability (schema enum) to be set")
	}
	// schema_recoverability must be present
	if result.WithheldReason.SchemaRecoverability == "" {
		t.Fatal("expected schema_recoverability to be set")
	}
	// In strict mode, recoverability shows schema_recoverability value
	if result.WithheldReason.Recoverability != result.WithheldReason.SchemaRecoverability {
		t.Fatalf("expected recoverability == schema_recoverability in strict mode, got recoverability=%s schema_recoverability=%s",
			result.WithheldReason.Recoverability, result.WithheldReason.SchemaRecoverability)
	}
}

func TestVerifyStrictWithheldReasonSchemaInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Use blocked-missing-evidence fixture which triggers schema_invalid
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--strict-withheld-reason", "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var rawResult map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rawResult); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	// Strict mode: withheld_reason must not contain legacy keys
	wr, ok := rawResult["withheld_reason"].(map[string]any)
	if !ok {
		t.Fatal("expected withheld_reason to be a map")
	}
	if _, has := wr["failure_class"]; has {
		t.Fatal("expected failure_class key to be absent in strict mode")
	}
	if _, has := wr["failure_stage"]; has {
		t.Fatal("expected failure_stage key to be absent in strict mode")
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	// Check schema-like fields
	if result.WithheldReason.Class != "schema_or_policy_invalid" {
		t.Fatalf("expected class=schema_or_policy_invalid, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Stage != "verification" {
		t.Fatalf("expected stage=verification, got %s", result.WithheldReason.Stage)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
	if result.WithheldReason.BlockingPredicate != "schema_invalid" {
		t.Fatalf("expected blocking_predicate=schema_invalid, got %s", result.WithheldReason.BlockingPredicate)
	}
	// recoverability must be schema enum value
	if result.WithheldReason.Recoverability == "" {
		t.Fatal("expected recoverability (schema enum) to be set")
	}
	// schema_recoverability must be present
	if result.WithheldReason.SchemaRecoverability == "" {
		t.Fatal("expected schema_recoverability to be set")
	}
	// In strict mode, recoverability shows schema_recoverability value
	if result.WithheldReason.Recoverability != result.WithheldReason.SchemaRecoverability {
		t.Fatalf("expected recoverability == schema_recoverability in strict mode, got recoverability=%s schema_recoverability=%s",
			result.WithheldReason.Recoverability, result.WithheldReason.SchemaRecoverability)
	}
	// schema_recoverability should be "manual" for retry_with_fixes
	if result.WithheldReason.SchemaRecoverability != "manual" {
		t.Fatalf("expected schema_recoverability=manual, got %s", result.WithheldReason.SchemaRecoverability)
	}
}

func TestVerifyDefaultOutputIncludesLegacyFields(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Use blocked-missing-evidence fixture
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	// Default mode (no --strict-withheld-reason)
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	// Default mode must include legacy fields
	if result.WithheldReason.FailureClass == "" {
		t.Fatal("expected failure_class to be present in default mode")
	}
	if result.WithheldReason.FailureStage == "" {
		t.Fatal("expected failure_stage to be present in default mode")
	}
	// And must include schema_recoverability
	if result.WithheldReason.SchemaRecoverability == "" {
		t.Fatal("expected schema_recoverability to be present")
	}
}

func TestVerifyContractOraclesViolation(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a source file that will trigger the contract oracle rule
	srcFile := filepath.Join(tmpDir, "src", "utils", "format.ts")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	// Write console.log which should trigger the default policy rule (if uncommented)
	// Since default policy is empty, we need a custom policy with an active rule
	if err := os.WriteFile(srcFile, []byte("console.log('debug');\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a custom contract oracle policy that matches console.log in .ts files
	policyYAML := `version: 1
grep_rules:
  - id: console-log-typescript
    description: "console.log statement in TypeScript"
    file_pattern: "*.ts"
    pattern: 'console\.log\('
    message: "Remove console.log statements before committing"
`
	policyPath := filepath.Join(tmpDir, "contract-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid completion card
	cardYAML := `schema_version: "1"
task_id: TASK-CO-VIOLATION-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/utils/format.ts
  manual_rationale: Simple utility function
claim:
  fix_status: fixed
  summary: Added utility
  evidence:
    - description: Source file
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--contract-oracles", "--contract-oracles-policy", policyPath, "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.AdmissionOutcome != "blocked" {
		t.Fatalf("expected admission_outcome=blocked, got %s", result.AdmissionOutcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected acceptance_status=withheld, got %s", result.AcceptanceStatus)
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "contract_oracle_blocked" {
		t.Fatalf("expected blocking_predicate=contract_oracle_blocked, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.Class != "contract_mismatch" {
		t.Fatalf("expected class=contract_mismatch, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
	if result.WithheldReason.Stage != "verification" {
		t.Fatalf("expected stage=verification, got %s", result.WithheldReason.Stage)
	}
}

func TestVerifyContractOraclesCleanPass(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an empty contract oracle policy
	policyYAML := `version: 1
grep_rules: []
`
	policyPath := filepath.Join(tmpDir, "empty-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy success-light fixture
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--contract-oracles", "--contract-oracles-policy", policyPath, "--json"}, &stdout, &stderr)

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
	if result.WithheldReason != nil {
		t.Fatalf("expected no withheld_reason for clean pass, got %+v", result.WithheldReason)
	}
}

func TestVerifyContractOraclesDefaultUnchanged(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a custom contract oracle policy that would fail if checked
	policyYAML := `version: 1
grep_rules:
  - id: console-log-typescript
    description: "console.log statement in TypeScript"
    file_pattern: "*.ts"
    pattern: 'console\.log\('
    message: "Remove console.log statements before committing"
`
	policyPath := filepath.Join(tmpDir, "contract-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a source file that would violate the policy
	srcFile := filepath.Join(tmpDir, "src", "utils", "format.ts")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte("console.log('debug');\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy success-light fixture - should pass even though policy file exists
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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

	// Verify WITHOUT --contract-oracles flag - should pass even though policy file exists and would violate
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected exit code %d (default behavior unchanged), got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok without --contract-oracles flag, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyHelpDocumentsContractOracles(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--help"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("expected exit code %d for usage, got %d. stdout: %s\nstderr: %s", ExitUsage, code, stdout.String(), stderr.String())
	}

	if !strings.Contains(stderr.String(), "--contract-oracles") {
		t.Fatalf("expected usage to contain --contract-oracles, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--contract-oracles-policy") {
		t.Fatalf("expected usage to contain --contract-oracles-policy, got: %s", stderr.String())
	}
}

func TestVerifyContractOraclesStrictWithheldReason(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a source file that will trigger the contract oracle rule
	srcFile := filepath.Join(tmpDir, "src", "utils", "format.ts")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte("console.log('debug');\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a custom contract oracle policy
	policyYAML := `version: 1
grep_rules:
  - id: console-log-typescript
    description: "console.log statement in TypeScript"
    file_pattern: "*.ts"
    pattern: 'console\.log\('
    message: "Remove console.log statements before committing"
`
	policyPath := filepath.Join(tmpDir, "contract-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid completion card
	cardYAML := `schema_version: "1"
task_id: TASK-CO-STRICT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/utils/format.ts
  manual_rationale: Simple utility function
claim:
  fix_status: fixed
  summary: Added utility
  evidence:
    - description: Source file
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--contract-oracles", "--contract-oracles-policy", policyPath, "--strict-withheld-reason", "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var rawResult map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rawResult); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	// Strict mode: withheld_reason must not contain legacy keys
	wr, ok := rawResult["withheld_reason"].(map[string]any)
	if !ok {
		t.Fatal("expected withheld_reason to be a map")
	}
	if _, has := wr["failure_class"]; has {
		t.Fatal("expected failure_class key to be absent in strict mode")
	}
	if _, has := wr["failure_stage"]; has {
		t.Fatal("expected failure_stage key to be absent in strict mode")
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	// Violations should set result.OK to false
	if result.OK {
		t.Fatal("expected violations to set OK=false")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.Class != "contract_mismatch" {
		t.Fatalf("expected class=contract_mismatch, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Stage != "verification" {
		t.Fatalf("expected stage=verification, got %s", result.WithheldReason.Stage)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
	if result.WithheldReason.BlockingPredicate != "contract_oracle_blocked" {
		t.Fatalf("expected blocking_predicate=contract_oracle_blocked, got %s", result.WithheldReason.BlockingPredicate)
	}
	// In strict mode, recoverability shows schema_recoverability value
	if result.WithheldReason.Recoverability != result.WithheldReason.SchemaRecoverability {
		t.Fatalf("expected recoverability == schema_recoverability in strict mode")
	}
}

func TestVerifyContractOraclesGoldenFixture(t *testing.T) {
	// Test that uses the blocked-contract-oracle golden fixture
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy the golden fixture files
	fixtureBase := filepath.Join("..", "..", "examples", "golden", "regression", "blocked-contract-oracle")

	// Copy completion card
	cardSrc := filepath.Join(fixtureBase, "completion-card.yaml")
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	srcData, err := os.ReadFile(cardSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cardDst, srcData, 0644); err != nil {
		t.Fatal(err)
	}

	// Copy contract policy
	policySrc := filepath.Join(fixtureBase, "contract-policy.yaml")
	policyDst := filepath.Join(tmpDir, "contract-policy.yaml")
	policyData, err := os.ReadFile(policySrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyDst, policyData, 0644); err != nil {
		t.Fatal(err)
	}

	// Copy violation marker file
	markerSrc := filepath.Join(fixtureBase, "contract-oracle-violation-marker.txt")
	markerDst := filepath.Join(tmpDir, "contract-oracle-violation-marker.txt")
	markerData, err := os.ReadFile(markerSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerDst, markerData, 0644); err != nil {
		t.Fatal(err)
	}

	// Copy schemas
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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

	// First verify WITHOUT --contract-oracles - should pass
	var stdoutNoCO bytes.Buffer
	var stderrNoCO bytes.Buffer
	codeNoCO := Run([]string{"verify", "--card", "completion-card.yaml", "--json"}, &stdoutNoCO, &stderrNoCO)
	if codeNoCO != ExitOK {
		t.Fatalf("expected exit code %d without contract-oracles, got %d. stdout: %s\nstderr: %s", ExitOK, codeNoCO, stdoutNoCO.String(), stderrNoCO.String())
	}
	var resultNoCO VerifyResult
	if err := json.Unmarshal(stdoutNoCO.Bytes(), &resultNoCO); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdoutNoCO.String())
	}
	if !resultNoCO.OK {
		t.Fatalf("expected ok without contract-oracles flag, got outcome=%s status=%s", resultNoCO.AdmissionOutcome, resultNoCO.AcceptanceStatus)
	}

	// Now verify WITH --contract-oracles - should fail
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--contract-oracles", "--contract-oracles-policy", policyDst, "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit with contract-oracles, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok with contract-oracles")
	}
	if result.AdmissionOutcome != "blocked" {
		t.Fatalf("expected admission_outcome=blocked, got %s", result.AdmissionOutcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected acceptance_status=withheld, got %s", result.AcceptanceStatus)
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "contract_oracle_blocked" {
		t.Fatalf("expected blocking_predicate=contract_oracle_blocked, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.Class != "contract_mismatch" {
		t.Fatalf("expected class=contract_mismatch, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
	if result.WithheldReason.Stage != "verification" {
		t.Fatalf("expected stage=verification, got %s", result.WithheldReason.Stage)
	}
}

func TestVerifyContractOraclesDependencyRuleViolation(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a source file with a forbidden import
	srcFile := filepath.Join(tmpDir, "src", "utils", "format.go")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte(`package utils
import "github.com/forbidden/package"
func Format() {}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a contract oracle policy with dependency_rule
	policyYAML := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
    message: "Do not use forbidden packages"
`
	policyPath := filepath.Join(tmpDir, "contract-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid completion card
	cardYAML := `schema_version: "1"
task_id: TASK-CO-DEP-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/utils/format.go
  manual_rationale: Simple utility function
claim:
  fix_status: fixed
  summary: Added utility
  evidence:
    - description: Source file
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--contract-oracles", "--contract-oracles-policy", policyPath, "--json"}, &stdout, &stderr)

	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.AdmissionOutcome != "blocked" {
		t.Fatalf("expected admission_outcome=blocked, got %s", result.AdmissionOutcome)
	}
	if result.AcceptanceStatus != "withheld" {
		t.Fatalf("expected acceptance_status=withheld, got %s", result.AcceptanceStatus)
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "contract_oracle_blocked" {
		t.Fatalf("expected blocking_predicate=contract_oracle_blocked, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.Class != "contract_mismatch" {
		t.Fatalf("expected class=contract_mismatch, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
	if result.WithheldReason.Stage != "verification" {
		t.Fatalf("expected stage=verification, got %s", result.WithheldReason.Stage)
	}
}

func TestVerifyContractOraclesDependencyRuleNoViolation(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a source file with a clean import
	srcFile := filepath.Join(tmpDir, "src", "utils", "format.go")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte(`package utils
import "fmt"
func Format() { fmt.Println("hello") }`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a contract oracle policy with dependency_rule that doesn't match
	policyYAML := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	policyPath := filepath.Join(tmpDir, "contract-policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid completion card
	cardYAML := `schema_version: "1"
task_id: TASK-CO-DEP-002
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/utils/format.go
  manual_rationale: Simple utility function
claim:
  fix_status: fixed
  summary: Added utility
  evidence:
    - description: Source file
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--contract-oracles", "--contract-oracles-policy", policyPath, "--json"}, &stdout, &stderr)

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
	if result.WithheldReason != nil {
		t.Fatalf("expected no withheld_reason for clean pass, got %+v", result.WithheldReason)
	}
}

func TestVerifyStandardTierAutoEnablesMutationGuard(t *testing.T) {
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

	cardYAML := `schema_version: "1"
task_id: TASK-STD-MG-001
tier: standard
owner: alice
accountable: bob
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - "docs/product.md"
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
  unresolved_context_questions: []
  context_evidence: []
prediction:
  claim: Standard tier test
  expected_effect: Works
  measurable_signal: go test ./...
  falsification_method: Skip fix
  horizon: same_verify
evidence:
  files_changed:
    - src/main.go
  command_evidence:
    - command: go test ./...
      exit_code: 0
claim:
  fix_status: fixed
  summary: Standard tier test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create referenced context file so context floor passes
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "product.md"), []byte("# Product\n"), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
		t.Fatal(err)
	}

	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
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
	// No --mutation-guard or --strict flag
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.MutationGuard == nil {
		t.Fatal("expected mutation guard to be enabled for standard tier")
	}
	if !result.MutationGuard.Enabled {
		t.Fatal("expected mutation guard enabled")
	}
	if result.MutationGuard.Violated {
		t.Fatal("expected mutation guard not violated")
	}
}

func TestVerifyDeepTierAutoEnablesMutationGuard(t *testing.T) {
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

	cardYAML := `schema_version: "1"
task_id: TASK-DEEP-MG-001
tier: deep
owner: alice
accountable: bob
state:
  read_set:
    - src/main.go
  write_set:
    - src/main.go
evidence:
  files_changed:
    - src/main.go
  command_evidence:
    - command: go test ./...
      exit_code: 0
  verification_artifacts:
    - kind: unit_test
      command: go test ./...
      status: passed
      verifies:
        - "basic functionality"
      does_not_verify:
        - "edge cases"
      confidence: medium
  untested_regions:
    - "No integration tests."
  remaining_risks:
    - "May fail in production."
  rollback_policy:
    - "Revert commit."
  execution_controls:
    - "Deploy behind feature flag."
context_alignment:
  stale_ground_checked: true
  context_pack_id: "ctx-deep-001"
  product_contract_refs:
    - "docs/product.md"
  architecture_refs: []
  decision_refs: []
  test_matrix_refs: []
  unresolved_context_questions: []
  context_evidence: []
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: Deep tier test
  expected_effect: Works
  measurable_signal: go test
  falsification_method: Skip fix
  horizon: same_verify
claim:
  fix_status: fixed
  summary: Deep tier test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create referenced context file so context floor passes
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "product.md"), []byte("# Product\n"), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
		t.Fatal(err)
	}

	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
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
	// No --mutation-guard or --strict flag
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.MutationGuard == nil {
		t.Fatal("expected mutation guard to be enabled for deep tier")
	}
	if !result.MutationGuard.Enabled {
		t.Fatal("expected mutation guard enabled")
	}
	if result.MutationGuard.Violated {
		t.Fatal("expected mutation guard not violated")
	}
}

func TestVerifyLightTierDoesNotAutoEnableMutationGuard(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardYAML := `schema_version: "1"
task_id: TASK-LIGHT-MG-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.go
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: Light tier test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	// No --mutation-guard or --strict flag
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.MutationGuard != nil {
		t.Fatalf("expected mutation guard to be disabled for light tier, got %+v", result.MutationGuard)
	}
}

func TestVerifyExplicitMutationGuardEnablesForLight(t *testing.T) {
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

	cardYAML := `schema_version: "1"
task_id: TASK-LIGHT-EXPLICIT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.go
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: Light tier test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
		t.Fatal(err)
	}

	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--mutation-guard", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.MutationGuard == nil {
		t.Fatal("expected mutation guard to be enabled with explicit flag")
	}
	if !result.MutationGuard.Enabled {
		t.Fatal("expected mutation guard enabled")
	}
	if result.MutationGuard.Violated {
		t.Fatal("expected mutation guard not violated")
	}
}

func TestVerifyStandardTierAutoEnablesContextFloor(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardYAML := `schema_version: "1"
task_id: TASK-CF-AUTO-001
tier: standard
owner: alice
accountable: bob
done_checklist:
  source_of_truth_read: true
prediction:
  claim: Auto context floor test
  expected_effect: Tests pass
  measurable_signal: npm test
  falsification_method: Skip fix
  horizon: same_verify
evidence:
  files_changed:
    - src/main.ts
  command_evidence:
    - command: npm test
      exit_code: 0
claim:
  fix_status: fixed
  summary: Auto context floor test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
	}
	if result.AdmissionOutcome != "failed" {
		t.Fatalf("expected admission_outcome=failed, got %s", result.AdmissionOutcome)
	}
	hasContextError := false
	for _, e := range result.AdmissionErrors {
		if strings.Contains(e, "context_alignment") {
			hasContextError = true
			break
		}
	}
	if !hasContextError {
		t.Fatalf("expected error mentioning context_alignment, got errors=%v", result.AdmissionErrors)
	}
}

func TestVerifyLightTierDoesNotAutoEnableContextFloor(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardYAML := `schema_version: "1"
task_id: TASK-CF-LIGHT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.ts
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: Light tier context floor test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
}

func TestVerifyExplicitContextFloorEnablesForLight(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardYAML := `schema_version: "1"
task_id: TASK-CF-EXPLICIT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.ts
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: Light tier explicit context floor test
  evidence:
    - description: Test pass
verification:
  status: passed
  checks:
    - name: schema-valid
      result: passed
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: none
  owner: alice
`
	cardDst := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardDst, []byte(cardYAML), 0644); err != nil {
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
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextDst := filepath.Join(tmpDir, "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contextDst, contextData, 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-floor", "--json"}, &stdout, &stderr)

	// Light tier with --context-floor is advisory-only; should still pass
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok for light tier advisory context floor, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}
