package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDecisionRecordMissingIDUsageError covers the safe V1 rule that
// --id is required. The CLI must return a usage error.
func TestDecisionRecordMissingIDUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "record", "--decision", "x", "--rationale", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--id is required") {
		t.Fatalf("expected --id required error, got: %s", stderr.String())
	}
}

// TestDecisionRecordMissingDecisionUsageError covers the safe V1 rule
// that --decision is required.
func TestDecisionRecordMissingDecisionUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "record", "--id", "x", "--rationale", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--decision is required") {
		t.Fatalf("expected --decision required error, got: %s", stderr.String())
	}
}

// TestDecisionRecordMissingRationaleUsageError covers the safe V1 rule
// that --rationale is required.
func TestDecisionRecordMissingRationaleUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "record", "--id", "x", "--decision", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--rationale is required") {
		t.Fatalf("expected --rationale required error, got: %s", stderr.String())
	}
}

// TestDecisionRecordInvalidStatus covers the safe V1 rule that
// --status must be one of proposed/accepted/superseded/deprecated.
func TestDecisionRecordInvalidStatus(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "record", "--id", "x", "--decision", "y", "--rationale", "z", "--status", "maybe"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--status") {
		t.Fatalf("expected --status error, got: %s", stderr.String())
	}
}

// TestDecisionRecordWritesDefaultYAMLFile covers the deterministic
// default-output behavior: when the caller omits --output, the record
// is written to decisions/<id>.yaml with the YAML encoding. The
// decisions/ directory is created on demand.
func TestDecisionRecordWritesDefaultYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "record",
		"--id", "intake-lite",
		"--decision", "ship the safe V1 slice",
		"--rationale", "keep scope minimal; defer query/link/affected",
		"--title", "P3-S3 Decision Memory Safe V1",
		"--status", "accepted",
		"--context", "first vertical slice",
		"--consequence", "advisory note only",
		"--tag", "p3-s3,decision-memory",
		"--affected-path", "schemas/decision-record.schema.json",
		"--note", "follow-up slices may add query/link/affected",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "decisions/intake-lite.yaml") {
		t.Fatalf("expected default output path, got: %s", out)
	}

	data, err := os.ReadFile(filepath.Join("decisions", "intake-lite.yaml"))
	if err != nil {
		t.Fatalf("default output file not written: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"schema_version: \"1\"",
		"id: intake-lite",
		"title: P3-S3 Decision Memory Safe V1",
		"status: accepted",
		"decision: ship the safe V1 slice",
		"rationale: keep scope minimal; defer query/link/affected",
		"context: first vertical slice",
		"consequences: advisory note only",
		"- p3-s3",
		"- decision-memory",
		"- schemas/decision-record.schema.json",
		"notes: follow-up slices may add query/link/affected",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected file to contain %q, got:\n%s", want, text)
		}
	}
}

// TestDecisionRecordWritesExplicitOutputJSON covers the --output +
// --json path. The parent directory must already exist; the file
// content must round-trip through encoding/json.
func TestDecisionRecordWritesExplicitOutputJSON(t *testing.T) {
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "decision.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "record",
		"--id", "intake-lite",
		"--decision", "ship the safe V1 slice",
		"--rationale", "keep scope minimal; defer query/link/affected",
		"--output", out,
		"--json",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), out) {
		t.Fatalf("expected stdout to announce the output path, got: %s", stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("output file is not valid JSON: %v\noutput:\n%s", err, string(data))
	}
	if doc["schema_version"] != "1" {
		t.Fatalf("expected schema_version=1, got %v", doc["schema_version"])
	}
	if doc["id"] != "intake-lite" {
		t.Fatalf("expected id=intake-lite, got %v", doc["id"])
	}
	if doc["decision"] != "ship the safe V1 slice" {
		t.Fatalf("expected decision text, got %v", doc["decision"])
	}
}

// TestDecisionRecordOutputMissingParent covers the safe V1 rule that
// an explicit --output path's parent directory must already exist.
func TestDecisionRecordOutputMissingParent(t *testing.T) {
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "does", "not", "exist", "decision.yaml")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "record",
		"--id", "x",
		"--decision", "y",
		"--rationale", "z",
		"--output", out,
	}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "parent directory does not exist") {
		t.Fatalf("expected missing parent error, got: %s", stderr.String())
	}
}

// TestDecisionRecordUnknownFlag covers the safe V1 rule that unknown
// flags produce a usage error.
func TestDecisionRecordUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "record",
		"--id", "x",
		"--decision", "y",
		"--rationale", "z",
		"--from", "issue.md",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

// TestDecisionListEmptyDir covers the safe V1 rule that an absent or
// empty decisions/ directory is reported as count=0 with no error.
func TestDecisionListEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "list", "--dir", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Count: 0") {
		t.Fatalf("expected Count: 0, got: %s", stdout.String())
	}
}

// TestDecisionListJSON covers the JSON output path: count and records
// array must reflect the on-disk decision files in deterministic
// (id-sorted) order.
func TestDecisionListJSON(t *testing.T) {
	tmpDir := t.TempDir()
	// Create two records out of id-sorted order to exercise the sort.
	records := []struct {
		id   string
		body string
	}{
		{"zeta", "schema_version: \"1\"\nid: zeta\ndecision: z\nrationale: zr\nstatus: accepted\n"},
		{"alpha", "schema_version: \"1\"\nid: alpha\ndecision: a\nrationale: ar\nstatus: proposed\n"},
	}
	for _, r := range records {
		if err := os.WriteFile(filepath.Join(tmpDir, r.id+".yaml"), []byte(r.body), 0644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "list", "--dir", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var doc struct {
		Directory string                   `json:"directory"`
		Count     int                      `json:"count"`
		Records   []map[string]interface{} `json:"records"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("list --json output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
	if doc.Count != 2 {
		t.Fatalf("expected count=2, got %d", doc.Count)
	}
	if len(doc.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(doc.Records))
	}
	if doc.Records[0]["id"] != "alpha" || doc.Records[1]["id"] != "zeta" {
		t.Fatalf("expected alpha then zeta (id-sorted), got %v / %v", doc.Records[0]["id"], doc.Records[1]["id"])
	}
	if doc.Records[0]["status"] != "proposed" {
		t.Fatalf("expected status=proposed for alpha, got %v", doc.Records[0]["status"])
	}
}

// TestDecisionListText covers the human-readable output path. The
// list must surface id, status, title, decision, and path for each
// record, sorted by id.
func TestDecisionListText(t *testing.T) {
	tmpDir := t.TempDir()
	body := "schema_version: \"1\"\nid: intake-lite\ntitle: P3-S3\ndecision: ship\nrationale: minimal\nstatus: accepted\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "intake-lite.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "list", "--dir", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Directory: " + tmpDir,
		"Count: 1",
		"id=intake-lite",
		"status=accepted",
		"title=\"P3-S3\"",
		"decision: ship",
		filepath.Join(tmpDir, "intake-lite.yaml"),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected list output to contain %q, got:\n%s", want, out)
		}
	}
}

// TestDecisionRecordThenListRoundtrip is the smoke test for the safe
// V1 vertical slice: write a record, then list the directory and
// confirm the new entry shows up with the expected fields.
func TestDecisionRecordThenListRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{
		"decision", "record",
		"--id", "roundtrip",
		"--decision", "ship the slice",
		"--rationale", "minimize scope",
		"--status", "accepted",
	}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("record: exit=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"decision", "list"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("list: exit=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Count: 1",
		"id=roundtrip",
		"status=accepted",
		"decision: ship the slice",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected list output to contain %q, got:\n%s", want, out)
		}
	}
}
