package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplainRequiresCardOrReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"explain"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage, got: %s", stderr.String())
	}
}

func TestExplainRejectsUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"explain", "--bogus", "--card", "x.yaml"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestExplainCardAccepted(t *testing.T) {
	setupVerifyProfileCard(t, "light", "TASK-EXPLAIN-OK-001")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"explain", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}
	var expl ExplainExplanation
	if err := json.Unmarshal(stdout.Bytes(), &expl); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !expl.OK {
		t.Fatalf("expected ok=true, got %+v", expl)
	}
	if expl.SchemaVersion != "x-harness.explain.v1" {
		t.Fatalf("expected schema_version=x-harness.explain.v1, got %s", expl.SchemaVersion)
	}
	if expl.Summary == "" {
		t.Fatal("expected summary to be set")
	}
}

func TestExplainCardWithheld(t *testing.T) {
	// Use a card that withholds (the same blocked-missing-evidence
	// fixture used elsewhere in the test suite).
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
	t.Cleanup(func() { os.Chdir(origWd) })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"explain", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}
	var expl ExplainExplanation
	if err := json.Unmarshal(stdout.Bytes(), &expl); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if expl.OK {
		t.Fatalf("expected ok=false for withheld card, got %+v", expl)
	}
	if expl.WithheldReason == nil {
		t.Fatal("expected withheld_reason in explanation")
	}
	if expl.Summary == "" {
		t.Fatal("expected summary to be set")
	}
	// Summary should mention withheld/withheld
	if !strings.Contains(expl.Summary, "withheld") {
		t.Fatalf("expected summary to mention withheld, got %q", expl.Summary)
	}
}

func TestExplainFromReport(t *testing.T) {
	// Build a minimal VerifyResult JSON file and use explain to read
	// it offline (no verify rerun).
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(tmpDir, "report.json")
	report := VerifyResult{
		OK:               false,
		TaskID:           "TASK-EXPLAIN-REPORT-001",
		Tier:             "standard",
		Profile:          "ci-strict",
		AdmissionOutcome: "failed",
		AcceptanceStatus: "withheld",
		SchemaError:      "missing required field",
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"explain", "--from-report", reportPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}
	var expl ExplainExplanation
	if err := json.Unmarshal(stdout.Bytes(), &expl); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if expl.TaskID != "TASK-EXPLAIN-REPORT-001" {
		t.Fatalf("expected task_id from report, got %q", expl.TaskID)
	}
	if expl.Profile != "ci-strict" {
		t.Fatalf("expected profile=ci-strict, got %q", expl.Profile)
	}
	if expl.OK {
		t.Fatal("expected ok=false from withheld report")
	}
}

func TestExplainCardTextOutput(t *testing.T) {
	setupVerifyProfileCard(t, "light", "TASK-EXPLAIN-TEXT-001")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"explain", "--card", "completion-card.yaml"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "summary:") {
		t.Fatalf("expected summary line, got: %s", out)
	}
	if !strings.Contains(out, "admission_outcome:") {
		t.Fatalf("expected admission_outcome, got: %s", out)
	}
}

func TestExplainHelpDocumentsFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"explain", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	for _, want := range []string{"--card", "--from-report", "--profile", "--json"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("expected usage to contain %q, got: %s", want, stderr.String())
		}
	}
}

func TestExplainInRootHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help-all"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "explain") {
		t.Fatalf("expected --help-all to mention explain, got: %s", stdout.String())
	}
}
