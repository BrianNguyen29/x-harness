package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupVerifyProfileCard mirrors the helpers used by verify_test.go: a
// minimal temp dir with a completion card and the two schemas the verify
// pipeline needs (completion-card + context-alignment).
func setupVerifyProfileCard(t *testing.T, tier, taskID string) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cardYAML := `schema_version: "1"
task_id: ` + taskID + `
tier: ` + tier + `
owner: alice
accountable: bob
`
	switch tier {
	case "standard":
		cardYAML += `done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: ` + taskID + ` claim
  expected_effect: works
  measurable_signal: tests pass
  falsification_method: skip fix
  horizon: same_verify
`
	case "deep":
		cardYAML += `state:
  read_set:
    - src/main.go
  write_set:
    - src/main.go
context_alignment:
  stale_ground_checked: true
  product_contract_refs:
    - README.md
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
  claim: ` + taskID + ` claim
  expected_effect: works
  measurable_signal: tests pass
  falsification_method: skip fix
  horizon: same_verify
`
	}
	cardYAML += `evidence:
  files_changed:
    - src/main.go
`
	if tier == "light" {
		cardYAML += `  manual_rationale: Simple change
`
	} else {
		cardYAML += `  command_evidence:
    - command: go test ./...
      exit_code: 0
`
	}
	cardYAML += `claim:
  fix_status: fixed
  summary: ` + taskID + `
  evidence:
    - description: source change
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

	if tier == "deep" || tier == "standard" {
		// Create a referenced context file so context floor passes when
		// the profile or tier auto-enables it.
		if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Product\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })

	return tmpDir
}

func TestVerifyProfileLightLocalAcceptsLightCard(t *testing.T) {
	setupVerifyProfileCard(t, "light", "TASK-PROFILE-LIGHT-001")

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
	if result.Profile != "light-local" {
		t.Fatalf("expected profile=light-local, got %q", result.Profile)
	}
	if !result.OK {
		t.Fatalf("expected ok, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyProfileCIStandardAppliesContextFloor(t *testing.T) {
	// A light-tier card SHOULD pass under ci-standard because the
	// context floor is advisory for non-standard/deep tiers. The
	// important thing is the profile is recorded and the result
	// matches the input tier.
	setupVerifyProfileCard(t, "light", "TASK-PROFILE-CI-STD-001")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-standard", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Profile != "ci-standard" {
		t.Fatalf("expected profile=ci-standard, got %q", result.Profile)
	}
	if !result.OK {
		t.Fatalf("expected ok, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyProfileUnknownProfileUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "bogus-profile", "--card", "x.yaml"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown --profile") {
		t.Fatalf("expected unknown profile error, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "light-local") {
		t.Fatalf("expected list of available profiles, got: %s", stderr.String())
	}
}

func TestVerifyProfileMissingValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--profile requires a value") {
		t.Fatalf("expected profile value required, got: %s", stderr.String())
	}
}

func TestVerifyProfileHelpDocumentsFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d for usage, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--profile") {
		t.Fatalf("expected usage to mention --profile, got: %s", stderr.String())
	}
}

func TestVerifyProfileAllKnownNames(t *testing.T) {
	// Guard against accidental removal of a profile name. The schema
	// enum and the matrix rule profiles depend on these names.
	for _, name := range []string{"light-local", "ci-standard", "ci-strict", "governed-deep"} {
		if _, ok := verifyProfiles[name]; !ok {
			t.Fatalf("expected profile %q in verifyProfiles", name)
		}
	}
}

func TestVerifyProfileExplicitFlagOverridesProfile(t *testing.T) {
	// When --profile says mutation-guard is on but the caller passes
	// --no-mutation-guard semantics through omitting the flag, the
	// profile wins. When the caller explicitly passes a flag that
	// conflicts, explicit flag wins. This test uses --mutation-guard
	// as the explicit-enable path, paired with --profile light-local
	// (which would set mutation-guard to false). The explicit
	// --mutation-guard must remain on.
	setupVerifyProfileCard(t, "light", "TASK-PROFILE-OVERRIDE-001")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "light-local", "--mutation-guard", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Profile != "light-local" {
		t.Fatalf("expected profile=light-local, got %q", result.Profile)
	}
	// The mutation guard was forced on by --mutation-guard, so it
	// should appear in the result.
	if result.MutationGuard == nil {
		t.Fatal("expected mutation guard present (explicit --mutation-guard)")
	}
}
