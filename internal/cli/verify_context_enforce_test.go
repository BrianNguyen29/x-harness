package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStandardCardWithManifest writes a standard-tier completion card
// to the working directory with a context_manifest reference. The card
// passes admission by default. A README.md is created so
// product_contract_refs resolves under the auto-enabled context floor.
func writeStandardCardWithManifest(t *testing.T, manifestPath string) {
	t.Helper()
	if _, err := os.Stat("README.md"); err != nil {
		if err := os.WriteFile("README.md", []byte("# Product\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	refsBlock := "context_alignment:\n"
	refsBlock += "  stale_ground_checked: true\n"
	refsBlock += "  product_contract_refs:\n    - README.md\n"
	refsBlock += "  architecture_refs: []\n"
	refsBlock += "  test_matrix_refs: []\n"
	refsBlock += "  decision_refs:\n    - decisions/ADR-1.md\n"
	refsBlock += "  unresolved_context_questions: []\n"
	refsBlock += "  context_evidence: []\n"
	manifestLine := ""
	if strings.TrimSpace(manifestPath) != "" {
		manifestLine = "context_manifest: " + strings.TrimSpace(manifestPath) + "\n"
	}
	cardYAML := `schema_version: "1"
task_id: TASK-CONTEXT-ENFORCE-001
tier: standard
owner: alice
accountable: bob
` + refsBlock + manifestLine + `done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: TASK-CONTEXT-ENFORCE-001 claim
  expected_effect: works
  measurable_signal: tests pass
  falsification_method: skip fix
  horizon: same_verify
evidence:
  files_changed:
    - src/main.go
  command_evidence:
    - command: go test ./...
      exit_code: 0
      runner: go-test
      started_at: "2026-06-06T00:00:00Z"
claim:
  fix_status: fixed
  summary: TASK-CONTEXT-ENFORCE-001
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
	if err := os.WriteFile("completion-card.yaml", []byte(cardYAML), 0644); err != nil {
		t.Fatal(err)
	}
}

// setupContextEnforceTestDir creates a minimal temp dir with schemas,
// policies, and a dummy decision file so the auto-enabled context floor
// for standard tier resolves.
func setupContextEnforceTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "decisions"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"completion-card.schema.json", "context-alignment.schema.json"} {
		src := filepath.Join("..", "..", "schemas", name)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "schemas", name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "contract-oracle.yaml"),
		[]byte("version: 1\ngrep_rules: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "decisions", "ADR-1.md"),
		[]byte("# ADR-1\n"), 0644); err != nil {
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
	return tmpDir
}

func TestVerifyContextEnforceOffDoesNotBlock(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-enforce", "off", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok under off, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyContextEnforceAdvisoryDoesNotBlock(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-enforce", "advisory", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok under advisory, got outcome=%s status=%s errors=%v", result.AdmissionOutcome, result.AcceptanceStatus, result.AdmissionErrors)
	}
	foundNote := false
	for _, n := range result.AdmissionNotes {
		if strings.Contains(n, "context_stale advisory") || strings.Contains(n, "manifest stale") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected stale manifest advisory note, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyContextEnforceBlockBlocksWhenStale(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-enforce", "block", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok under block when manifest is stale")
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
	if result.WithheldReason.BlockingPredicate != "context_stale" {
		t.Fatalf("expected blocking_predicate=context_stale, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.FailureClass != "context_stale" {
		t.Fatalf("expected failure_class=context_stale, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.Class != "context_stale" {
		t.Fatalf("expected class=context_stale, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
}

func TestVerifyContextEnforceBlockPassesWhenFresh(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("fresh.txt", []byte("fresh content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "manifest", "write", "--files", "fresh.txt", "--out", ".x-harness/context-manifest.yaml"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d writing manifest, got %d; stderr: %s", ExitOK, code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"verify", "--card", "completion-card.yaml", "--context-enforce", "block", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with fresh manifest, got outcome=%s status=%s errors=%v", result.AdmissionOutcome, result.AcceptanceStatus, result.AdmissionErrors)
	}
}

func TestVerifyContextEnforceMissingManifestSkips(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	// Do NOT create the manifest file

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-enforce", "block", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok when manifest missing, got outcome=%s status=%s errors=%v", result.AdmissionOutcome, result.AcceptanceStatus, result.AdmissionErrors)
	}
}

func TestVerifyContextEnforceInvalidManifestAdvisory(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("not_valid_yaml:::\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-enforce", "block", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with invalid manifest under block (advisory-only), got outcome=%s status=%s errors=%v", result.AdmissionOutcome, result.AcceptanceStatus, result.AdmissionErrors)
	}
	foundNote := false
	for _, n := range result.AdmissionNotes {
		if strings.Contains(n, "context_manifest advisory") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected invalid manifest advisory note, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyContextEnforceLightTierPasses(t *testing.T) {
	setupContextEnforceTestDir(t)
	if err := os.WriteFile("README.md", []byte("# Product\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cardYAML := `schema_version: "1"
task_id: TASK-CONTEXT-LIGHT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.go
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: TASK-CONTEXT-LIGHT-001
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
	if err := os.WriteFile("completion-card.yaml", []byte(cardYAML), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--context-enforce", "block", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d for light, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok for light tier under block, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyContextEnforceExplicitOffOverridesProfile(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "governed-deep", "--context-enforce", "off", "--intent-enforce", "off", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d with explicit off, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with explicit off, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyContextEnforceExplicitAdvisoryOverridesProfile(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-strict", "--context-enforce", "advisory", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d with explicit advisory, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with explicit advisory, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyContextEnforceInvalidValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--context-enforce", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid --context-enforce") {
		t.Fatalf("expected invalid value error, got: %s", stderr.String())
	}
}

func TestVerifyContextEnforceMissingValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--context-enforce"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires a value") {
		t.Fatalf("expected requires value error, got: %s", stderr.String())
	}
}

func TestVerifyHelpDocumentsContextEnforce(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--context-enforce") {
		t.Fatalf("expected usage to mention --context-enforce, got: %s", stderr.String())
	}
}

func TestVerifyProfileCIStrictBlocksStaleManifest(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-strict", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok under ci-strict with stale manifest")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "context_stale" {
		t.Fatalf("expected blocking_predicate=context_stale, got %s", result.WithheldReason.BlockingPredicate)
	}
}

func TestVerifyProfileGovernedDeepBlocksStaleManifest(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "governed-deep", "--intent-enforce", "off", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.WithheldReason == nil || result.WithheldReason.BlockingPredicate != "context_stale" {
		t.Fatalf("expected context_stale predicate, got %+v", result.WithheldReason)
	}
}

func TestVerifyProfileLightLocalAdvisoryForContextManifest(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "light-local", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d under light-local advisory, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok under light-local advisory, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
	foundNote := false
	for _, n := range result.AdmissionNotes {
		if strings.Contains(n, "context_stale advisory") || strings.Contains(n, "manifest stale") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected stale manifest advisory note under light-local, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyProfileCIStandardAdvisoryForContextManifest(t *testing.T) {
	setupContextEnforceTestDir(t)
	writeStandardCardWithManifest(t, ".x-harness/context-manifest.yaml")
	if err := os.MkdirAll(".x-harness", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".x-harness/context-manifest.yaml", []byte("version: \"1\"\nentries:\n  - path: stale.txt\n    sha256: deadbeef\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-standard", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d under ci-standard advisory, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok under ci-standard advisory, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestIsValidContextEnforce(t *testing.T) {
	cases := map[string]bool{
		"off":      true,
		"advisory": true,
		"block":    true,
		"":         false,
		"high":     false,
		"all":      false,
		"bogus":    false,
	}
	for v, want := range cases {
		if got := isValidContextEnforce(v); got != want {
			t.Errorf("isValidContextEnforce(%q) = %v, want %v", v, got, want)
		}
	}
}
