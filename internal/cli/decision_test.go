package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

// writeDecisionLinkFixture writes a YAML completion-card fixture
// at path. body is the raw YAML text; the helper does not validate
// the document, it only ensures the file exists for the link
// command to read.
func writeDecisionLinkFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

// readDecisionLinkFixture loads the file at path and unmarshals
// it into a generic map. It is used to assert that the link
// command preserved the structure of an existing card.
func readDecisionLinkFixture(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	return doc
}

// TestDecisionLinkMissingCardUsageError covers the safe V1 rule
// that --card is required for the link command.
func TestDecisionLinkMissingCardUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "link", "--decision", "ADR-001"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--card is required") {
		t.Fatalf("expected --card required error, got: %s", stderr.String())
	}
}

// TestDecisionLinkMissingDecisionUsageError covers the safe V1
// rule that --decision is required for the link command.
func TestDecisionLinkMissingDecisionUsageError(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	writeDecisionLinkFixture(t, cardPath, "schema_version: \"1\"\ntask_id: t\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "link", "--card", cardPath}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--decision is required") {
		t.Fatalf("expected --decision required error, got: %s", stderr.String())
	}
}

// TestDecisionLinkCardNotFound covers the safe V1 rule that a
// missing card file is an error (not a usage error) so users can
// tell a typo from a missing flag.
func TestDecisionLinkCardNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	missing := filepath.Join(tmpDir, "does-not-exist.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "link", "--card", missing, "--decision", "ADR-001"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "card not found") {
		t.Fatalf("expected card-not-found error, got: %s", stderr.String())
	}
}

// TestDecisionLinkAppendsNewRef covers the safe V1 rule that
// linking a fresh decision to a card with no existing
// context_alignment creates the context_alignment map and the
// decision_refs array and writes the new ref.
func TestDecisionLinkAppendsNewRef(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	writeDecisionLinkFixture(t, cardPath, "schema_version: \"1\"\ntask_id: t\ntier: standard\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "link", "--card", cardPath, "--decision", "ADR-001"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Card: " + cardPath,
		"Output: " + cardPath,
		"Added: ADR-001",
		"Skipped: (none)",
		"Total decision refs: 1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected link output to contain %q, got:\n%s", want, out)
		}
	}

	doc := readDecisionLinkFixture(t, cardPath)
	ca, ok := doc["context_alignment"].(map[string]any)
	if !ok {
		t.Fatalf("expected context_alignment to be a map, got %T", doc["context_alignment"])
	}
	raw, ok := ca["decision_refs"].([]any)
	if !ok {
		t.Fatalf("expected decision_refs to be an array, got %T", ca["decision_refs"])
	}
	if len(raw) != 1 || raw[0] != "ADR-001" {
		t.Fatalf("expected decision_refs=[ADR-001], got %v", raw)
	}
	// Top-level fields must be preserved.
	if doc["schema_version"] != "1" || doc["task_id"] != "t" || doc["tier"] != "standard" {
		t.Fatalf("expected top-level fields to be preserved, got: %v", doc)
	}
}

// TestDecisionLinkIdempotentDuplicate covers the safe V1 rule
// that re-running the same link command does not duplicate the
// decision_refs array. The handler must report the ref as
// skipped and leave the file unchanged (semantically).
func TestDecisionLinkIdempotentDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	body := "schema_version: \"1\"\ntask_id: t\ncontext_alignment:\n  decision_refs:\n    - ADR-001\n"
	writeDecisionLinkFixture(t, cardPath, body)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "link", "--card", cardPath, "--decision", "ADR-001"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Added: (none)") {
		t.Fatalf("expected Added: (none), got: %s", out)
	}
	if !strings.Contains(out, "Skipped: ADR-001") {
		t.Fatalf("expected Skipped: ADR-001, got: %s", out)
	}
	if !strings.Contains(out, "Total decision refs: 1") {
		t.Fatalf("expected Total decision refs: 1, got: %s", out)
	}

	doc := readDecisionLinkFixture(t, cardPath)
	ca, ok := doc["context_alignment"].(map[string]any)
	if !ok {
		t.Fatalf("expected context_alignment to be a map, got %T", doc["context_alignment"])
	}
	raw := ca["decision_refs"].([]any)
	if len(raw) != 1 || raw[0] != "ADR-001" {
		t.Fatalf("expected decision_refs=[ADR-001] (no duplicate), got %v", raw)
	}
}

// TestDecisionLinkAppendsToExistingRefs covers the safe V1 rule
// that linking a new decision to a card that already has a
// decision_refs array appends the new ref after the existing
// ones and keeps the order stable.
func TestDecisionLinkAppendsToExistingRefs(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	body := "schema_version: \"1\"\ntask_id: t\ncontext_alignment:\n  decision_refs:\n    - ADR-001\n    - ADR-002\n"
	writeDecisionLinkFixture(t, cardPath, body)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "link", "--card", cardPath, "--decision", "ADR-003"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Added: ADR-003") {
		t.Fatalf("expected Added: ADR-003, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Total decision refs: 3") {
		t.Fatalf("expected Total decision refs: 3, got: %s", stdout.String())
	}

	doc := readDecisionLinkFixture(t, cardPath)
	raw := doc["context_alignment"].(map[string]any)["decision_refs"].([]any)
	want := []string{"ADR-001", "ADR-002", "ADR-003"}
	if len(raw) != len(want) {
		t.Fatalf("expected %d refs, got %d (%v)", len(want), len(raw), raw)
	}
	for i, w := range want {
		if raw[i] != w {
			t.Fatalf("expected ref[%d]=%s, got %v", i, w, raw[i])
		}
	}
}

// TestDecisionLinkCommaAndRepeatable covers the safe V1 rule that
// --decision accepts both repeated flags and comma-separated
// values, and that duplicate candidates within a single command
// are deduped (so `--decision ADR-001,ADR-001` adds ADR-001
// exactly once).
func TestDecisionLinkCommaAndRepeatable(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	writeDecisionLinkFixture(t, cardPath, "schema_version: \"1\"\ntask_id: t\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "link",
		"--card", cardPath,
		"--decision", "ADR-001,ADR-002",
		"--decision", "ADR-001",
		"--decision", "ADR-003",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Total decision refs: 3") {
		t.Fatalf("expected 3 unique refs, got: %s", stdout.String())
	}
	doc := readDecisionLinkFixture(t, cardPath)
	raw := doc["context_alignment"].(map[string]any)["decision_refs"].([]any)
	if len(raw) != 3 {
		t.Fatalf("expected 3 refs (intra-command dedup), got %v", raw)
	}
}

// TestDecisionLinkPreservesOtherContextAlignmentFields covers the
// safe V1 rule that the link command only mutates
// context_alignment.decision_refs; every other field on the card
// (top level or nested inside context_alignment) must be
// preserved unchanged.
func TestDecisionLinkPreservesOtherContextAlignmentFields(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	body := "" +
		"schema_version: \"1\"\n" +
		"task_id: p3-s3-decision-link\n" +
		"tier: standard\n" +
		"owner: fixer\n" +
		"accountable: orchestrator\n" +
		"context_alignment:\n" +
		"  stale_ground_checked: true\n" +
		"  product_contract_refs:\n" +
		"    - contracts/decision-memory.md\n" +
		"  architecture_refs:\n" +
		"    - docs/architecture.md\n" +
		"  decision_refs: []\n" +
		"  test_matrix_refs:\n" +
		"    - tests/smoke/decision.md\n" +
		"  unresolved_context_questions:\n" +
		"    - is the link slice safe V1?\n" +
		"  context_evidence:\n" +
		"    - ref: schemas/context-alignment.schema.json\n" +
		"      kind: schema\n" +
		"  context_delta: drafted the link slice\n" +
		"  context_pack_id: pack-001\n"
	writeDecisionLinkFixture(t, cardPath, body)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"decision", "link", "--card", cardPath, "--decision", "ADR-001"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	doc := readDecisionLinkFixture(t, cardPath)
	// Top-level fields must be preserved.
	for k, want := range map[string]string{
		"schema_version": "1",
		"task_id":        "p3-s3-decision-link",
		"tier":           "standard",
		"owner":          "fixer",
		"accountable":    "orchestrator",
	} {
		if doc[k] != want {
			t.Fatalf("expected %s=%q, got %v", k, want, doc[k])
		}
	}
	ca, ok := doc["context_alignment"].(map[string]any)
	if !ok {
		t.Fatalf("expected context_alignment to be a map, got %T", doc["context_alignment"])
	}
	if ca["stale_ground_checked"] != true {
		t.Fatalf("expected stale_ground_checked preserved, got %v", ca["stale_ground_checked"])
	}
	if ca["context_delta"] != "drafted the link slice" {
		t.Fatalf("expected context_delta preserved, got %v", ca["context_delta"])
	}
	if ca["context_pack_id"] != "pack-001" {
		t.Fatalf("expected context_pack_id preserved, got %v", ca["context_pack_id"])
	}
	// decision_refs must contain the new ref.
	raw := ca["decision_refs"].([]any)
	if len(raw) != 1 || raw[0] != "ADR-001" {
		t.Fatalf("expected decision_refs=[ADR-001], got %v", raw)
	}
	// Arrays of strings must survive the round trip.
	for k, want := range map[string][]string{
		"product_contract_refs":        {"contracts/decision-memory.md"},
		"architecture_refs":            {"docs/architecture.md"},
		"test_matrix_refs":             {"tests/smoke/decision.md"},
		"unresolved_context_questions": {"is the link slice safe V1?"},
	} {
		arr, ok := ca[k].([]any)
		if !ok {
			t.Fatalf("expected %s to be an array, got %T", k, ca[k])
		}
		if len(arr) != len(want) {
			t.Fatalf("expected %s len=%d, got %d", k, len(want), len(arr))
		}
		for i, w := range want {
			if arr[i] != w {
				t.Fatalf("expected %s[%d]=%q, got %v", k, i, w, arr[i])
			}
		}
	}
	// context_evidence is an array of objects; verify it is
	// preserved (not converted to a generic map of strings).
	ev, ok := ca["context_evidence"].([]any)
	if !ok || len(ev) != 1 {
		t.Fatalf("expected context_evidence to be an array of 1, got %T (%v)", ca["context_evidence"], ca["context_evidence"])
	}
}

// TestDecisionLinkOutDoesNotMutateOriginal covers the safe V1
// rule that --out writes a patched copy and leaves the original
// card file unchanged. The on-disk bytes of the original must be
// identical before and after the command runs.
func TestDecisionLinkOutDoesNotMutateOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	body := "schema_version: \"1\"\ntask_id: t\ncontext_alignment:\n  decision_refs: []\n"
	writeDecisionLinkFixture(t, cardPath, body)
	originalBytes, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	outPath := filepath.Join(tmpDir, "patched.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "link",
		"--card", cardPath,
		"--decision", "ADR-001",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Output: "+outPath) {
		t.Fatalf("expected Output: %s, got: %s", outPath, stdout.String())
	}

	// Original file must be byte-identical.
	afterBytes, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read original after: %v", err)
	}
	if !bytes.Equal(originalBytes, afterBytes) {
		t.Fatalf("original card was mutated. before:\n%s\nafter:\n%s", string(originalBytes), string(afterBytes))
	}

	// Patched file must contain the new ref.
	patched := readDecisionLinkFixture(t, outPath)
	raw := patched["context_alignment"].(map[string]any)["decision_refs"].([]any)
	if len(raw) != 1 || raw[0] != "ADR-001" {
		t.Fatalf("expected patched decision_refs=[ADR-001], got %v", raw)
	}
}

// TestDecisionLinkJSON covers the safe V1 --json output path:
// the JSON must include the schema_version, the card path, the
// output path, the added refs, the skipped refs, and the final
// decision_refs array.
func TestDecisionLinkJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	body := "schema_version: \"1\"\ntask_id: t\ncontext_alignment:\n  decision_refs:\n    - ADR-001\n"
	writeDecisionLinkFixture(t, cardPath, body)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "link",
		"--card", cardPath,
		"--decision", "ADR-002",
		"--json",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var doc struct {
		SchemaVersion string   `json:"schema_version"`
		Card          string   `json:"card"`
		Out           string   `json:"out"`
		Added         []string `json:"added"`
		Skipped       []string `json:"skipped"`
		DecisionRefs  []string `json:"decision_refs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("link --json output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
	if doc.SchemaVersion != "x-harness.decision.link.v1" {
		t.Fatalf("expected schema_version=x-harness.decision.link.v1, got %q", doc.SchemaVersion)
	}
	if doc.Card != cardPath {
		t.Fatalf("expected card=%s, got %q", cardPath, doc.Card)
	}
	if doc.Out != cardPath {
		t.Fatalf("expected out=%s (in-place), got %q", cardPath, doc.Out)
	}
	if len(doc.Added) != 1 || doc.Added[0] != "ADR-002" {
		t.Fatalf("expected added=[ADR-002], got %v", doc.Added)
	}
	if len(doc.Skipped) != 0 {
		t.Fatalf("expected skipped=[], got %v", doc.Skipped)
	}
	if len(doc.DecisionRefs) != 2 || doc.DecisionRefs[0] != "ADR-001" || doc.DecisionRefs[1] != "ADR-002" {
		t.Fatalf("expected decision_refs=[ADR-001 ADR-002], got %v", doc.DecisionRefs)
	}

	// The card file must also be valid YAML with the new ref.
	card := readDecisionLinkFixture(t, cardPath)
	raw := card["context_alignment"].(map[string]any)["decision_refs"].([]any)
	if len(raw) != 2 || raw[1] != "ADR-002" {
		t.Fatalf("expected on-disk card to have ADR-002 appended, got %v", raw)
	}
}

// TestDecisionLinkUnknownFlag covers the safe V1 rule that
// unknown flags produce a usage error and do not mutate the
// card.
func TestDecisionLinkUnknownFlag(t *testing.T) {
	tmpDir := t.TempDir()
	cardPath := filepath.Join(tmpDir, "card.yaml")
	writeDecisionLinkFixture(t, cardPath, "schema_version: \"1\"\ntask_id: t\n")
	originalBytes, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"decision", "link",
		"--card", cardPath,
		"--decision", "ADR-001",
		"--bogus", "value",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}

	// Card must be unchanged.
	afterBytes, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatalf("read original after: %v", err)
	}
	if !bytes.Equal(originalBytes, afterBytes) {
		t.Fatalf("card was mutated by unknown flag. before:\n%s\nafter:\n%s", string(originalBytes), string(afterBytes))
	}
}
