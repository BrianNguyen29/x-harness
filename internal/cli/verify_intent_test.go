package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStandardCardWithIntentRef writes a standard-tier completion
// card to the working directory. The card passes admission by default;
// the optional top-level intent_ref field can be tuned via `ref`:
//   - nil or empty string: intent_ref is missing (triggers the
//     verify-layer block when enforcement is `block`)
//   - non-blank: intent_ref carries a value that suppresses the gate
//
// The card always carries a non-empty decision_refs entry so the
// decision_refs gate does not fire when the test profile (e.g.
// governed-deep) defaults to block. This isolates the intent_ref
// gate under test.
//
// The standard-tier context floor is auto-enabled by handleVerify, so
// the card carries a minimal context_alignment that satisfies the
// other ref checks (product_contract_refs points at a real file).
func writeStandardCardWithIntentRef(t *testing.T, ref string) {
	t.Helper()
	// A README.md file is required so product_contract_refs resolves
	// under the auto-enabled context floor for standard tier.
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
	intentRefLine := ""
	if strings.TrimSpace(ref) != "" {
		intentRefLine = "intent_ref: " + strings.TrimSpace(ref) + "\n"
	}
	cardYAML := `schema_version: "1"
task_id: TASK-INTENT-ENFORCE-001
tier: standard
owner: alice
accountable: bob
` + refsBlock + intentRefLine + `done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: TASK-INTENT-ENFORCE-001 claim
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
  summary: TASK-INTENT-ENFORCE-001
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

// setupIntentEnforceTestDir mirrors the helpers used by
// verify_decision_test.go: a minimal temp dir with completion card
// and the two schemas the verify pipeline needs (completion-card +
// context-alignment), plus a no-op contract-oracle policy so
// ci-strict and governed-deep (which auto-enable contract-oracles)
// do not fail on a missing policy file. A dummy decisions/ADR-1.md
// file is created so decision_refs references resolve under the
// auto-enabled context floor for standard tier.
func setupIntentEnforceTestDir(t *testing.T) string {
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

func TestVerifyIntentEnforceOffDoesNotBlock(t *testing.T) {
	// off must never block; the advisory note from admission.Run is
	// preserved (suppressing it would require invasive admission
	// changes that the task scope defers).
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--intent-enforce", "off", "--json"}, &stdout, &stderr)
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

func TestVerifyIntentEnforceAdvisoryDoesNotBlock(t *testing.T) {
	// advisory preserves the existing safe-V1 behavior: a note is
	// emitted, but admission is not withheld.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--intent-enforce", "advisory", "--json"}, &stdout, &stderr)
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
		if strings.Contains(n, "intent_ref not declared") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected intent_ref missing advisory note, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyIntentEnforceBlockBlocksWhenMissing(t *testing.T) {
	// block on a standard card with no intent_ref must withhold
	// with the intent_ref_missing predicate.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--intent-enforce", "block", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok under block when intent_ref is missing")
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
	if result.WithheldReason.BlockingPredicate != "intent_ref_missing" {
		t.Fatalf("expected blocking_predicate=intent_ref_missing, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.FailureClass != "intent_ref_missing" {
		t.Fatalf("expected failure_class=intent_ref_missing, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.Class != "intent_ref_missing" {
		t.Fatalf("expected class=intent_ref_missing, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
}

func TestVerifyIntentEnforceBlockPassesWithRef(t *testing.T) {
	// block on a standard card with a non-blank intent_ref must pass.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "doc/intake-lite.md")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--intent-enforce", "block", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with non-blank intent_ref, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyIntentEnforceBlockBlankStringBlocks(t *testing.T) {
	// block on a card with a blank intent_ref string must withhold
	// with the intent_ref_missing predicate. The runtime check
	// mirrors the advisory semantics in admission.evaluateIntentRef.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "   ")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--intent-enforce", "block", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok for blank string under block")
	}
	if result.WithheldReason == nil || result.WithheldReason.BlockingPredicate != "intent_ref_missing" {
		t.Fatalf("expected intent_ref_missing predicate, got %+v", result.WithheldReason)
	}
}

func TestVerifyIntentEnforceLightTierPasses(t *testing.T) {
	// The check is tier-scoped: light cards must pass under block
	// even without intent_ref. Use a light card that is otherwise
	// schema-valid.
	setupIntentEnforceTestDir(t)
	cardYAML := `schema_version: "1"
task_id: TASK-INTENT-LIGHT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.go
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: TASK-INTENT-LIGHT-001
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--intent-enforce", "block", "--json"}, &stdout, &stderr)
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

func TestVerifyIntentEnforceExplicitOffOverridesProfile(t *testing.T) {
	// governed-deep would default to block; an explicit
	// --intent-enforce off must win.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "governed-deep", "--intent-enforce", "off", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
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

func TestVerifyIntentEnforceExplicitAdvisoryOverridesProfile(t *testing.T) {
	// governed-deep would default to block; explicit advisory must
	// not block.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "governed-deep", "--intent-enforce", "advisory", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
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

func TestVerifyIntentEnforceInvalidValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--intent-enforce", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid --intent-enforce") {
		t.Fatalf("expected invalid value error, got: %s", stderr.String())
	}
}

func TestVerifyIntentEnforceMissingValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--intent-enforce"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires a value") {
		t.Fatalf("expected requires value error, got: %s", stderr.String())
	}
}

func TestVerifyHelpDocumentsIntentEnforce(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--intent-enforce") {
		t.Fatalf("expected usage to mention --intent-enforce, got: %s", stderr.String())
	}
}

func TestVerifyProfileGovernedDeepBlocksMissingIntentRef(t *testing.T) {
	// governed-deep defaults to block; the missing-intent_ref case
	// must withhold with the intent_ref_missing predicate.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "governed-deep", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok under governed-deep with missing intent_ref")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "intent_ref_missing" {
		t.Fatalf("expected blocking_predicate=intent_ref_missing, got %s", result.WithheldReason.BlockingPredicate)
	}
}

func TestVerifyProfileCIStrictAdvisoryForIntentRefByDefault(t *testing.T) {
	// Conservative per oracle review: ci-strict stays advisory for
	// intent_ref by default. The card is missing intent_ref but the
	// verify layer must not block; the advisory note from
	// admission.Run is preserved.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-strict", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d under ci-strict advisory, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}
	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok under ci-strict advisory, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
	foundNote := false
	for _, n := range result.AdmissionNotes {
		if strings.Contains(n, "intent_ref not declared") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected intent_ref advisory note under ci-strict, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyProfileCIStrictExplicitBlockBlocksMissingIntentRef(t *testing.T) {
	// ci-strict stays advisory by default; explicit --intent-enforce
	// block can still block under any profile.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-strict", "--intent-enforce", "block", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok under ci-strict with explicit block and missing intent_ref")
	}
	if result.WithheldReason == nil || result.WithheldReason.BlockingPredicate != "intent_ref_missing" {
		t.Fatalf("expected intent_ref_missing predicate, got %+v", result.WithheldReason)
	}
}

func TestVerifyProfileLightLocalAdvisoryForIntentRef(t *testing.T) {
	// light-local defaults to advisory; it must NOT block and the
	// advisory note must still appear in AdmissionNotes (light tier
	// stays quiet at the admission layer; the verify layer is also
	// quiet because the intent_ref gate never applies to light).
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

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
}

func TestVerifyProfileCIStandardAdvisoryForIntentRef(t *testing.T) {
	// ci-standard defaults to advisory; it must NOT block even when
	// the card has no intent_ref.
	setupIntentEnforceTestDir(t)
	writeStandardCardWithIntentRef(t, "")

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

func TestIsValidIntentEnforce(t *testing.T) {
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
		if got := isValidIntentEnforce(v); got != want {
			t.Errorf("isValidIntentEnforce(%q) = %v, want %v", v, got, want)
		}
	}
}
