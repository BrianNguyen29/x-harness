package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStandardCardWithDecisionRefs writes a standard-tier completion
// card to the working directory. The card passes admission by default;
// the decision_refs field is always present (the schema requires it)
// and can be tuned via `refs`:
//   - nil or []any{}: decision_refs is an empty array (triggers the
//     verify-layer block when enforcement is `block`)
//   - non-empty: decision_refs has at least one non-blank string
//
// The standard-tier context floor is auto-enabled by handleVerify, so
// the card carries a minimal context_alignment that satisfies the
// other ref checks (product_contract_refs points at a real file).
func writeStandardCardWithDecisionRefs(t *testing.T, refs []any) {
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
	refsBlock += "  unresolved_context_questions: []\n"
	refsBlock += "  context_evidence: []\n"
	if len(refs) == 0 {
		// Empty or nil slice: emit an empty YAML array so the schema
		// accepts the document and the runtime gate still triggers
		// (HasAnyDecisionRef treats an empty array the same as a
		// missing key).
		refsBlock += "  decision_refs: []\n"
	} else {
		refsBlock += "  decision_refs:\n"
		for _, r := range refs {
			refsBlock += "    - " + strings.TrimSpace(toYAMLString(r)) + "\n"
		}
	}
	cardYAML := `schema_version: "1"
task_id: TASK-DECISION-ENFORCE-001
tier: standard
owner: alice
accountable: bob
` + refsBlock + `state:
  read_set:
    - src/main.go
  write_set:
    - src/main.go
done_checklist:
  source_of_truth_read: true
  scope_explained: true
  read_write_sets_declared: true
  evidence_attached: true
  coverage_gap_declared: true
  risk_and_rollback_declared: true
  prediction_declared: true
prediction:
  claim: TASK-DECISION-ENFORCE-001 claim
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
      started_at: "2026-06-04T00:00:00Z"
      stdout_hash: "abc123"
claim:
  fix_status: fixed
  summary: TASK-DECISION-ENFORCE-001
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

// toYAMLString is a tiny stringifier for []any elements so we can
// embed them directly in the card. Non-string elements fall back to
// the empty string (kept narrow; this helper is only used for the
// decision_refs slice in these tests).
func toYAMLString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func setupDecisionEnforceTestDir(t *testing.T) string {
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
	// A no-op contract-oracle policy so ci-strict and governed-deep
	// (which auto-enable contract-oracles) do not fail on a missing
	// policy file.
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "contract-oracle.yaml"),
		[]byte("version: 1\ngrep_rules: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// A dummy decision file so refs like "decisions/intake-lite.md"
	// resolve under the auto-enabled context floor.
	if err := os.WriteFile(filepath.Join(tmpDir, "decisions", "intake-lite.md"),
		[]byte("# Intake Lite\n"), 0644); err != nil {
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

func TestVerifyDecisionEnforceOffDoesNotBlock(t *testing.T) {
	// off must never block; the advisory note from admission.Run is
	// preserved (suppressing it would require invasive admission
	// changes that the task scope defers).
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--decision-enforce", "off", "--json"}, &stdout, &stderr)
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

func TestVerifyDecisionEnforceAdvisoryDoesNotBlock(t *testing.T) {
	// advisory preserves the existing safe-V1 behavior: a note is
	// emitted, but admission is not withheld.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--decision-enforce", "advisory", "--json"}, &stdout, &stderr)
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
		if strings.Contains(n, "context_alignment.decision_refs is empty") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected decision_refs empty advisory note, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyDecisionEnforceBlockBlocksWhenMissing(t *testing.T) {
	// block on a standard card with no decision_refs must withhold
	// with the decision_refs_missing predicate.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--decision-enforce", "block", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok under block when decision_refs is missing")
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
	if result.WithheldReason.BlockingPredicate != "decision_refs_missing" {
		t.Fatalf("expected blocking_predicate=decision_refs_missing, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.FailureClass != "decision_refs_missing" {
		t.Fatalf("expected failure_class=decision_refs_missing, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.Class != "context_insufficient" {
		t.Fatalf("expected class=context_insufficient, got %s", result.WithheldReason.Class)
	}
	if result.WithheldReason.Owner != "implementation-worker" {
		t.Fatalf("expected owner=implementation-worker, got %s", result.WithheldReason.Owner)
	}
}

func TestVerifyDecisionEnforceBlockPassesWithRefs(t *testing.T) {
	// block on a standard card with at least one non-blank ref must
	// pass.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, []any{"decisions/intake-lite.md"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--decision-enforce", "block", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with non-empty decision_refs, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyDecisionEnforceBlockEmptyArrayBlocks(t *testing.T) {
	// block on a card with an empty decision_refs array must withhold
	// with the decision_refs_missing predicate. The runtime check
	// mirrors the advisory semantics in admission.evaluateDecisionRefs.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, []any{})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--decision-enforce", "block", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok for empty array under block")
	}
	if result.WithheldReason == nil || result.WithheldReason.BlockingPredicate != "decision_refs_missing" {
		t.Fatalf("expected decision_refs_missing predicate, got %+v", result.WithheldReason)
	}
}

func TestVerifyDecisionEnforceLightTierPasses(t *testing.T) {
	// The check is tier-scoped: light cards must pass under block
	// even without decision_refs. Use a light card that is otherwise
	// schema-valid.
	setupDecisionEnforceTestDir(t)
	cardYAML := `schema_version: "1"
task_id: TASK-DECISION-LIGHT-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.go
  manual_rationale: Simple change
claim:
  fix_status: fixed
  summary: TASK-DECISION-LIGHT-001
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--decision-enforce", "block", "--json"}, &stdout, &stderr)
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

func TestVerifyDecisionEnforceExplicitOffOverridesProfile(t *testing.T) {
	// ci-strict would default to block; an explicit --decision-enforce
	// off must win.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-strict", "--decision-enforce", "off", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
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

func TestVerifyDecisionEnforceExplicitAdvisoryOverridesProfile(t *testing.T) {
	// governed-deep would default to block; explicit advisory must
	// not block. The card is also missing intent_ref, so the
	// --intent-enforce advisory flag must be passed explicitly to
	// isolate the decision-enforce behavior under test; without it,
	// the governed-deep default of block would withhold the card on
	// the new intent_ref gate.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "governed-deep", "--decision-enforce", "advisory", "--intent-enforce", "advisory", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
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

func TestVerifyDecisionEnforceInvalidValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--decision-enforce", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid --decision-enforce") {
		t.Fatalf("expected invalid value error, got: %s", stderr.String())
	}
}

func TestVerifyDecisionEnforceMissingValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--decision-enforce"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires a value") {
		t.Fatalf("expected requires value error, got: %s", stderr.String())
	}
}

func TestVerifyHelpDocumentsDecisionEnforce(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--decision-enforce") {
		t.Fatalf("expected usage to mention --decision-enforce, got: %s", stderr.String())
	}
}

func TestVerifyProfileCIStrictBlocksMissingDecisionRefs(t *testing.T) {
	// ci-strict defaults to block; the missing-decision_refs case
	// must withhold with the decision_refs_missing predicate.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

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
		t.Fatal("expected not ok under ci-strict with missing decision_refs")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "decision_refs_missing" {
		t.Fatalf("expected blocking_predicate=decision_refs_missing, got %s", result.WithheldReason.BlockingPredicate)
	}
}

func TestVerifyProfileGovernedDeepBlocksMissingDecisionRefs(t *testing.T) {
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

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
	if result.WithheldReason == nil || result.WithheldReason.BlockingPredicate != "decision_refs_missing" {
		t.Fatalf("expected decision_refs_missing predicate, got %+v", result.WithheldReason)
	}
}

func TestVerifyProfileLightLocalAdvisoryForDecisionRefs(t *testing.T) {
	// light-local defaults to advisory; it must NOT block and the
	// advisory note must still appear in AdmissionNotes.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

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
		if strings.Contains(n, "context_alignment.decision_refs is empty") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected decision_refs advisory note under light-local, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyProfileCIStandardAdvisoryForDecisionRefs(t *testing.T) {
	// ci-standard defaults to advisory; it must NOT block even when
	// the card has no decision_refs.
	setupDecisionEnforceTestDir(t)
	writeStandardCardWithDecisionRefs(t, nil)

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

func TestIsValidDecisionEnforce(t *testing.T) {
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
		if got := isValidDecisionEnforce(v); got != want {
			t.Errorf("isValidDecisionEnforce(%q) = %v, want %v", v, got, want)
		}
	}
}
