package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/loader"
)

func TestCleanDryRunDefault(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	os.MkdirAll(filepath.Join(tmpDir, ".x-harness", "tmp"), 0755)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"clean", "--tmp"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("expected dry-run message, got:\n%s", out)
	}
	if !strings.Contains(out, "would delete") {
		t.Fatalf("expected would delete message, got:\n%s", out)
	}
}

func TestCleanForceDeletes(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	os.MkdirAll(filepath.Join(tmpDir, ".x-harness", "tmp"), 0755)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"clean", "--tmp", "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "deleted:") {
		t.Fatalf("expected deleted message, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".x-harness", "tmp")); err == nil {
		t.Fatalf("expected tmp dir to be deleted")
	}
}

func TestCleanUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"clean", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestCleanResetCardDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"clean", "--reset-card"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "would rename") {
		t.Fatalf("expected would rename message, got:\n%s", out)
	}
}

func TestCleanArchiveSuccessNotAccepted(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardPath, []byte("acceptance_status: pending\noutcome: failed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"clean", "--archive-success", "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "not accepted; skipping archive") {
		t.Fatalf("expected skip archive message, got:\n%s", out)
	}
}

func TestCleanArchiveSuccessAccepted(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	if err := os.WriteFile(cardPath, []byte("acceptance_status: accepted\noutcome: success\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"clean", "--archive-success", "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "archived:") {
		t.Fatalf("expected archived message, got:\n%s", out)
	}
	if _, err := os.Stat(cardPath); err == nil {
		t.Fatalf("expected card to be archived")
	}
}

func TestInterventionValidateMissingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intervention", "validate"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--intervention") {
		t.Fatalf("expected --intervention error, got: %s", stderr.String())
	}
}

func TestInterventionValidateValid(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	schemaContent := `{
  "title": "intervention",
  "type": "object",
  "required": ["actor", "task", "scope", "decision", "reason", "expiration", "authorizer"],
  "properties": {
    "actor": { "type": "string" },
    "task": { "type": "string" },
    "scope": { "type": "string", "enum": ["file", "directory", "path", "global"] },
    "paths": { "type": "array", "items": { "type": "string" } },
    "decision": { "type": "string", "enum": ["allow", "deny", "flag", "override"] },
    "reason": { "type": "string" },
    "expiration": { "type": "string", "format": "date-time" },
    "authorizer": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" }
  },
  "additionalProperties": false
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "intervention.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	intervention := `actor: test
task: test-task
scope: global
decision: allow
reason: test
expiration: 2099-01-01T00:00:00Z
authorizer: maintainer
`
	interventionPath := filepath.Join(tmpDir, "intervention.yaml")
	if err := os.WriteFile(interventionPath, []byte(intervention), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intervention", "validate", "--intervention", interventionPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid") {
		t.Fatalf("expected valid message, got: %s", stdout.String())
	}
}

func TestInterventionValidateInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	schemaContent := `{
  "title": "intervention",
  "type": "object",
  "required": ["actor", "task", "scope", "decision", "reason", "expiration", "authorizer"],
  "properties": {
    "actor": { "type": "string" },
    "task": { "type": "string" },
    "scope": { "type": "string", "enum": ["file", "directory", "path", "global"] },
    "paths": { "type": "array", "items": { "type": "string" } },
    "decision": { "type": "string", "enum": ["allow", "deny", "flag", "override"] },
    "reason": { "type": "string" },
    "expiration": { "type": "string", "format": "date-time" },
    "authorizer": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" }
  },
  "additionalProperties": false
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "intervention.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	intervention := `actor: test
task: test-task
scope: global
decision: deny
reason: test
expiration: 2099-01-01T00:00:00Z
authorizer: maintainer
`
	interventionPath := filepath.Join(tmpDir, "intervention.yaml")
	if err := os.WriteFile(interventionPath, []byte(intervention), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intervention", "validate", "--intervention", interventionPath}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stdout.String(), "decision must be allow or override") {
		t.Fatalf("expected decision error, got: %s", stdout.String())
	}
}

func TestInterventionValidateJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	schemaContent := `{
  "title": "intervention",
  "type": "object",
  "required": ["actor", "task", "scope", "decision", "reason", "expiration", "authorizer"],
  "properties": {
    "actor": { "type": "string" },
    "task": { "type": "string" },
    "scope": { "type": "string", "enum": ["file", "directory", "path", "global"] },
    "paths": { "type": "array", "items": { "type": "string" } },
    "decision": { "type": "string", "enum": ["allow", "deny", "flag", "override"] },
    "reason": { "type": "string" },
    "expiration": { "type": "string", "format": "date-time" },
    "authorizer": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" }
  },
  "additionalProperties": false
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "intervention.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	intervention := `actor: test
task: test-task
scope: global
decision: allow
reason: test
expiration: 2099-01-01T00:00:00Z
authorizer: maintainer
`
	interventionPath := filepath.Join(tmpDir, "intervention.yaml")
	if err := os.WriteFile(interventionPath, []byte(intervention), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intervention", "validate", "--intervention", interventionPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"valid": true`) {
		t.Fatalf("expected valid=true in JSON, got: %s", stdout.String())
	}
}

func TestIntakeClassifyMissingPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "classify", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "intake.yaml not found") {
		t.Fatalf("expected policy not found error, got: %s", stderr.String())
	}
}

func TestIntakeClassifyCommentOnly(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakePolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "classify", "--root", tmpDir, "--task", "fix docs", "--change", "comment-only"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "tiny") {
		t.Fatalf("expected tiny label, got: %s", stdout.String())
	}
}

func TestIntakeClassifyHighRisk(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakePolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "classify", "--root", tmpDir, "--task", "update auth logic"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "high_risk") {
		t.Fatalf("expected high_risk label, got: %s", stdout.String())
	}
}

func TestIntakeClassifyJSON(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakePolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "classify", "--root", tmpDir, "--task", "routine fix", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"intake_label"`) {
		t.Fatalf("expected JSON output, got: %s", stdout.String())
	}
}

func TestIntakeExplainMissingCard(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakePolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "explain", "--root", tmpDir}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--card") {
		t.Fatalf("expected --card error, got: %s", stderr.String())
	}
}

func TestIntakeExplainValidCard(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakePolicy(t, tmpDir)

	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `tier: standard
claim:
  summary: fix typo
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "explain", "--root", tmpDir, "--card", cardPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "standard") {
		t.Fatalf("expected tier mention, got: %s", stdout.String())
	}
}

func TestIntakeExplainDowngrade(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakePolicy(t, tmpDir)

	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `tier: light
claim:
  summary: update auth logic
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "explain", "--root", tmpDir, "--card", cardPath}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stdout.String(), "downgrade") {
		t.Fatalf("expected downgrade mention, got: %s", stdout.String())
	}
}

func TestIntakeExplainJSON(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakePolicy(t, tmpDir)

	cardPath := filepath.Join(tmpDir, "completion-card.yaml")
	cardContent := `tier: standard
claim:
  summary: routine fix
`
	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "explain", "--root", tmpDir, "--card", cardPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok"`) {
		t.Fatalf("expected JSON output, got: %s", stdout.String())
	}
}

func TestIntakeUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown intake subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestExportRoutesToFrozenExport(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"export", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "frozen bundle written:") {
		t.Fatalf("expected bundle written message, got: %s", stdout.String())
	}
}

func TestImportRoutesToFrozenImport(t *testing.T) {
	root := setupFrozenTestDir(t)
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"export", "--out", out, "--root", root}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("export failed: %d. stderr: %s", code, stderr.String())
	}
	target := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"import", out, "--target", target, "--force"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "frozen import wrote") {
		t.Fatalf("expected wrote message, got: %s", stdout.String())
	}
}

func setupIntakePolicy(t *testing.T, root string) {
	t.Helper()
	policyPath := filepath.Join(root, "policies")
	if err := os.MkdirAll(policyPath, 0755); err != nil {
		t.Fatal(err)
	}
	policyContent := `version: 1
intake_labels:
  tiny:
    runtime_tier: light
    signals:
      - comment_only
  normal:
    runtime_tier: standard
    signals:
      - routine_implementation
  high_risk:
    runtime_tier: deep
    signals:
      - auth
high_risk_signals:
  auth:
    description: Auth changes
    examples:
      - login
runtime_tier_confirmation:
  tiers: [light, standard, deep]
  note: Tiers remain light, standard, deep.
`
	if err := os.WriteFile(filepath.Join(policyPath, "intake.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestInterventionValidateYAMLTimeType(t *testing.T) {
	// Ensure YAML-loaded time strings don't break schema validation
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	schemaContent := `{
  "title": "intervention",
  "type": "object",
  "required": ["actor", "task", "scope", "decision", "reason", "expiration", "authorizer"],
  "properties": {
    "actor": { "type": "string" },
    "task": { "type": "string" },
    "scope": { "type": "string", "enum": ["file", "directory", "path", "global"] },
    "paths": { "type": "array", "items": { "type": "string" } },
    "decision": { "type": "string", "enum": ["allow", "deny", "flag", "override"] },
    "reason": { "type": "string" },
    "expiration": { "type": "string", "format": "date-time" },
    "authorizer": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" }
  },
  "additionalProperties": false
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "intervention.schema.json"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	intervention := `actor: test
task: test-task
scope: global
decision: allow
reason: test
expiration: 2099-01-01T00:00:00Z
authorizer: maintainer
`
	interventionPath := filepath.Join(tmpDir, "intervention.yaml")
	if err := os.WriteFile(interventionPath, []byte(intervention), 0644); err != nil {
		t.Fatal(err)
	}

	var artifact map[string]any
	if err := loader.LoadDocument(interventionPath, &artifact); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// yaml.v3 may convert date-like strings to time.Time.
	// Verify our conversion works.
	converted := convertYAMLTimes(artifact)
	artifact, _ = converted.(map[string]any)
	if _, ok := artifact["expiration"].(string); !ok {
		t.Fatalf("expected expiration to be string after conversion, got %T", artifact["expiration"])
	}
}
