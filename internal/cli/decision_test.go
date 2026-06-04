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

// TestDecisionQueryMatchSubstring covers the case-insensitive
// substring match across multiple records. The fixture contains
// two records that match the keyword and one that does not, and
// the output must surface the matches in id-sorted order with the
// keyword, directory, and count surfaced on stdout.
func TestDecisionQueryMatchSubstring(t *testing.T) {
	tmpDir := t.TempDir()
	records := []struct {
		id   string
		body string
	}{
		{"zeta", "schema_version: \"1\"\nid: zeta\ntitle: AuthZ\ndecision: ship RBAC\nrationale: zr\nstatus: accepted\n"},
		{"alpha", "schema_version: \"1\"\nid: alpha\ntitle: AuthN\ndecision: ship OIDC\nrationale: ar\nstatus: proposed\n"},
		{"beta", "schema_version: \"1\"\nid: beta\ntitle: Logging\ndecision: structured logs\nrationale: br\nstatus: accepted\n"},
	}
	for _, r := range records {
		if err := os.WriteFile(filepath.Join(tmpDir, r.id+".yaml"), []byte(r.body), 0644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "query", "--dir", tmpDir, "--keyword", "auth"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Directory: " + tmpDir,
		`Keyword: "auth"`,
		"Count: 2",
		"id=alpha",
		"id=zeta",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected query output to contain %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "id=beta") {
		t.Fatalf("did not expect beta in query output, got:\n%s", out)
	}
	// id-sorted order: alpha must appear before zeta in the output.
	if strings.Index(out, "id=alpha") > strings.Index(out, "id=zeta") {
		t.Fatalf("expected id-sorted output (alpha before zeta), got:\n%s", out)
	}
}

// TestDecisionQueryNoMatch covers the safe V1 rule that a search
// that returns zero matches must exit OK with Count: 0 and no
// Records: section, not an error.
func TestDecisionQueryNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	body := "schema_version: \"1\"\nid: alpha\ntitle: Logging\ndecision: structured logs\nrationale: ar\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "alpha.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "query", "--dir", tmpDir, "--keyword", "nonexistent-token"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Count: 0") {
		t.Fatalf("expected Count: 0, got: %s", out)
	}
	if strings.Contains(out, "Records:") {
		t.Fatalf("did not expect Records: section on no-match, got: %s", out)
	}
}

// TestDecisionQueryMissingKeywordUsageError covers the safe V1 rule
// that --keyword is required and missing it produces a usage error.
func TestDecisionQueryMissingKeywordUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "query", "--dir", "x"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--keyword is required") {
		t.Fatalf("expected --keyword required error, got: %s", stderr.String())
	}
}

// TestDecisionQueryUnknownFlag covers the safe V1 rule that unknown
// flags produce a usage error.
func TestDecisionQueryUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "query", "--keyword", "x", "--bogus", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

// TestDecisionQueryJSON covers the JSON output mode: directory,
// keyword, count, and records must round-trip and reflect the
// on-disk decisions in deterministic id-sorted order.
func TestDecisionQueryJSON(t *testing.T) {
	tmpDir := t.TempDir()
	records := []struct {
		id   string
		body string
	}{
		{"zeta", "schema_version: \"1\"\nid: zeta\ntitle: AuthZ\ndecision: ship\nrationale: r\nstatus: accepted\n"},
		{"alpha", "schema_version: \"1\"\nid: alpha\ntitle: Logging\ndecision: logs\nrationale: r\nstatus: proposed\n"},
	}
	for _, r := range records {
		if err := os.WriteFile(filepath.Join(tmpDir, r.id+".yaml"), []byte(r.body), 0644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "query", "--dir", tmpDir, "--keyword", "auth", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var doc struct {
		Directory string                   `json:"directory"`
		Keyword   string                   `json:"keyword"`
		Count     int                      `json:"count"`
		Records   []map[string]interface{} `json:"records"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("query --json output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
	if doc.Directory != tmpDir {
		t.Fatalf("expected directory=%s, got %q", tmpDir, doc.Directory)
	}
	if doc.Keyword != "auth" {
		t.Fatalf("expected keyword=auth, got %q", doc.Keyword)
	}
	if doc.Count != 1 || len(doc.Records) != 1 {
		t.Fatalf("expected count=1 with 1 record, got count=%d records=%d", doc.Count, len(doc.Records))
	}
	if doc.Records[0]["id"] != "zeta" {
		t.Fatalf("expected id=zeta, got %v", doc.Records[0]["id"])
	}
}

// TestDecisionQuerySearchesTagsAndAffectedPaths covers the safe V1
// rule that the keyword search must inspect both tags and
// affected_paths, not just the scalar string fields.
func TestDecisionQuerySearchesTagsAndAffectedPaths(t *testing.T) {
	tmpDir := t.TempDir()
	body := "schema_version: \"1\"\nid: pathless\ntitle: Generic\ndecision: ship\nrationale: r\nstatus: accepted\ntags:\n  - auth-rotation\naffected_paths:\n  - src/auth/login.ts\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "pathless.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"decision", "query", "--dir", tmpDir, "--keyword", "auth-rotation"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("tag match: expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Count: 1") {
		t.Fatalf("tag match: expected Count: 1, got: %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"decision", "query", "--dir", tmpDir, "--keyword", "login.ts"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("affected_paths match: expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Count: 1") {
		t.Fatalf("affected_paths match: expected Count: 1, got: %s", stdout.String())
	}
}

// TestDecisionAffectedExactMatch covers the safe V1 rule that
// --path matches a record whose affected_paths contains the exact
// same path after filepath.Clean normalization.
func TestDecisionAffectedExactMatch(t *testing.T) {
	tmpDir := t.TempDir()
	body := "schema_version: \"1\"\nid: auth-login\ntitle: Auth Login\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/login.ts\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "auth-login.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "affected", "--dir", tmpDir, "--path", "src/auth/login.ts"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Directory: " + tmpDir,
		"Path: src/auth/login.ts",
		"Count: 1",
		"id=auth-login",
		"title=\"Auth Login\"",
		"decision: ship",
		filepath.Join(tmpDir, "auth-login.yaml"),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected affected output to contain %q, got:\n%s", want, out)
		}
	}
}

// TestDecisionAffectedGlobMatch covers the safe V1 rule that
// --path matches a record whose affected_paths contains a glob
// pattern that matches the input path. The glob "src/auth/*.ts"
// must match "src/auth/login.ts".
func TestDecisionAffectedGlobMatch(t *testing.T) {
	tmpDir := t.TempDir()
	body := "schema_version: \"1\"\nid: auth-bulk\ntitle: Auth Bulk\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/*.ts\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "auth-bulk.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "affected", "--dir", tmpDir, "--path", "src/auth/login.ts"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Count: 1") {
		t.Fatalf("expected Count: 1 for glob match, got: %s", out)
	}
	if !strings.Contains(out, "id=auth-bulk") {
		t.Fatalf("expected id=auth-bulk, got: %s", out)
	}
}

// TestDecisionAffectedNoMatch covers the safe V1 rule that a
// non-matching --path returns count=0 (and exit OK) without
// surfacing any records.
func TestDecisionAffectedNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	body := "schema_version: \"1\"\nid: auth-login\ntitle: Auth Login\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/login.ts\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "auth-login.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "affected", "--dir", tmpDir, "--path", "src/other/file.ts"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Count: 0") {
		t.Fatalf("expected Count: 0, got: %s", out)
	}
	if strings.Contains(out, "Records:") {
		t.Fatalf("did not expect Records: section on no-match, got: %s", out)
	}
}

// TestDecisionAffectedMissingPathUsageError covers the safe V1
// rule that --path is required and missing it produces a usage
// error.
func TestDecisionAffectedMissingPathUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "affected", "--dir", "x"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--path is required") {
		t.Fatalf("expected --path required error, got: %s", stderr.String())
	}
}

// TestDecisionAffectedUnknownFlag covers the safe V1 rule that
// unknown flags produce a usage error.
func TestDecisionAffectedUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "affected", "--path", "x", "--bogus", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

// TestDecisionAffectedJSON covers the JSON output mode: directory,
// path, count, and records must round-trip and reflect the
// on-disk decisions in deterministic id-sorted order.
func TestDecisionAffectedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	records := []struct {
		id   string
		body string
	}{
		{"zeta", "schema_version: \"1\"\nid: zeta\ntitle: AuthZ\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/*.ts\n"},
		{"alpha", "schema_version: \"1\"\nid: alpha\ntitle: Other\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/other/*.ts\n"},
	}
	for _, r := range records {
		if err := os.WriteFile(filepath.Join(tmpDir, r.id+".yaml"), []byte(r.body), 0644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "affected", "--dir", tmpDir, "--path", "src/auth/login.ts", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var doc struct {
		Directory string                   `json:"directory"`
		Path      string                   `json:"path"`
		Count     int                      `json:"count"`
		Records   []map[string]interface{} `json:"records"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("affected --json output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
	if doc.Directory != tmpDir {
		t.Fatalf("expected directory=%s, got %q", tmpDir, doc.Directory)
	}
	if doc.Path != "src/auth/login.ts" {
		t.Fatalf("expected path=src/auth/login.ts, got %q", doc.Path)
	}
	if doc.Count != 1 || len(doc.Records) != 1 {
		t.Fatalf("expected count=1 with 1 record, got count=%d records=%d", doc.Count, len(doc.Records))
	}
	if doc.Records[0]["id"] != "zeta" {
		t.Fatalf("expected id=zeta, got %v", doc.Records[0]["id"])
	}
}

// TestDecisionAffectedDeterministicOrder covers the safe V1 rule
// that affected results are returned in id-sorted order even when
// the underlying files are written out of order.
func TestDecisionAffectedDeterministicOrder(t *testing.T) {
	tmpDir := t.TempDir()
	// Write records out of id-sorted order. Both must match the glob.
	records := []struct {
		id   string
		body string
	}{
		{"zeta", "schema_version: \"1\"\nid: zeta\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/*.ts\n"},
		{"alpha", "schema_version: \"1\"\nid: alpha\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/auth/*.ts\n"},
		{"beta", "schema_version: \"1\"\nid: beta\ndecision: ship\nrationale: r\nstatus: accepted\naffected_paths:\n  - src/other/*.ts\n"},
	}
	for _, r := range records {
		if err := os.WriteFile(filepath.Join(tmpDir, r.id+".yaml"), []byte(r.body), 0644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "affected", "--dir", tmpDir, "--path", "src/auth/login.ts"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	alphaIdx := strings.Index(out, "id=alpha")
	zetaIdx := strings.Index(out, "id=zeta")
	if alphaIdx < 0 || zetaIdx < 0 {
		t.Fatalf("expected both alpha and zeta in output, got:\n%s", out)
	}
	if alphaIdx > zetaIdx {
		t.Fatalf("expected id-sorted output (alpha before zeta), got:\n%s", out)
	}
	if strings.Contains(out, "id=beta") {
		t.Fatalf("did not expect beta (no match) in output, got:\n%s", out)
	}
}
