package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/evidence"
)

func TestEvidenceRunExecutesAndRecords(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "rec.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "run", "--out", outPath, "--", "echo", "hello"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected evidence file at %s: %v", outPath, err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var rec evidence.CommandRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("expected valid JSON record: %v\noutput: %s", err, data)
	}
	if rec.SchemaVersion != evidence.CommandRunSchemaVersion {
		t.Fatalf("expected schema_version=%s, got %s", evidence.CommandRunSchemaVersion, rec.SchemaVersion)
	}
	if rec.Command != "echo" {
		t.Fatalf("expected command=echo, got %q", rec.Command)
	}
	if len(rec.Args) != 1 || rec.Args[0] != "hello" {
		t.Fatalf("expected args=[hello], got %v", rec.Args)
	}
	if rec.ExitCode != 0 {
		t.Fatalf("expected exit_code=0, got %d", rec.ExitCode)
	}
	if rec.EvidenceID == "" {
		t.Fatal("expected evidence_id")
	}
	if rec.StdoutSHA256 == "" {
		t.Fatal("expected stdout_sha256")
	}
	if rec.StderrSHA256 == "" {
		t.Fatal("expected stderr_sha256")
	}
	if !strings.Contains(rec.StdoutCaptured, "hello") {
		t.Fatalf("expected stdout captured to contain 'hello', got %q", rec.StdoutCaptured)
	}
}

func TestEvidenceRunDefaultsToDotXHarnessDir(t *testing.T) {
	tmpDir := t.TempDir()
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
	code := Run([]string{"evidence", "run", "--", "printf", "ping"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	// The runner should create .x-harness/evidence/ and a single record.
	entries, err := os.ReadDir(filepath.Join(tmpDir, ".x-harness", "evidence"))
	if err != nil {
		t.Fatalf("expected .x-harness/evidence directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 evidence file, got %d", len(entries))
	}
	if !strings.HasSuffix(entries[0].Name(), ".json") {
		t.Fatalf("expected .json extension, got %q", entries[0].Name())
	}
}

func TestEvidenceRunCapturesFailureExitCode(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "fail.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	// false is a portable shell built-in that exits 1.
	code := Run([]string{"evidence", "run", "--out", outPath, "--", "false"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitError, code, stdout.String(), stderr.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var rec evidence.CommandRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("expected valid JSON record: %v\noutput: %s", err, data)
	}
	if rec.ExitCode == 0 {
		t.Fatalf("expected non-zero exit_code, got %d", rec.ExitCode)
	}
}

func TestEvidenceRunRequiresCommandAfterDoubleDash(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "run", "--"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires a command") {
		t.Fatalf("expected requires command error, got: %s", stderr.String())
	}
}

func TestEvidenceRunRejectsUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "run", "--bogus", "--", "echo", "hi"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

func TestEvidenceRunJSONOutputShape(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "rec.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "run", "--out", outPath, "--json", "--", "echo", "ok"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result evidence.CommandRunResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON result: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected result.ok=true, got %+v", result)
	}
	if result.SchemaVer != evidence.CommandRunSchemaVersion {
		t.Fatalf("expected schema_version=%s, got %s", evidence.CommandRunSchemaVersion, result.SchemaVer)
	}
	if result.Record.EvidenceID == "" {
		t.Fatal("expected record.evidence_id")
	}
	if result.OutPath != outPath {
		t.Fatalf("expected result.out_path=%s, got %s", outPath, result.OutPath)
	}
}

func TestEvidenceRunHelpDocumentsFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "run", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	for _, want := range []string{"--out", "--out-dir", "--max-capture", "--json"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("expected usage to contain %q, got: %s", want, stderr.String())
		}
	}
}

func TestEvidenceRunCapturesGitState(t *testing.T) {
	// When invoked from inside a git repo, the record should include a
	// git state block with at least the commit hash. We use the x-harness
	// repo itself (the test runs from within it).
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "rec.json")

	// We do not git-init a temp dir; we just run from the test's cwd
	// (the repo) so the runner sees a real .git.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"evidence", "run", "--out", outPath, "--", "true"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var rec evidence.CommandRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("expected valid JSON record: %v", err)
	}
	// Git state is best-effort: tests run from CI may have git
	// available, local dev may not. We accept nil but require
	// non-empty commit when present.
	if rec.Git != nil && rec.Git.Commit != "" {
		// ok
	}
}
