package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/boundary"
)

// setupBoundaryEnforceTestDir creates a temp working dir with a source
// file (src/ui/login.ts) that triggers a deny rule whose `from` glob
// matches the file and whose `to_import` matches an internal/db import.
// A boundary policy that fires the rule is written to
// policies/boundaries.yaml, and a minimal completion card is placed in
// the temp root.
func setupBoundaryEnforceTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "policies"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "src", "ui"), 0755); err != nil {
		t.Fatal(err)
	}

	// Boundary policy: deny rule that fires for any src/ui/*.ts file
	// importing from internal/db. The severity is `critical` so both
	// block_high and block_all modes will block.
	policyYAML := `version: 1
boundaries:
  - id: ui-cannot-access-db
    description: "UI must not import internal/db directly"
    from: "src/ui/**"
    to_import: "internal/db/**"
    action: deny
    severity: critical
    applies_to_languages: [typescript]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "policies", "boundaries.yaml"), []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Source file that violates the policy.
	if err := os.WriteFile(filepath.Join(tmpDir, "src", "ui", "login.ts"),
		[]byte(`import { getUser } from "internal/db/users";
`),
		0644); err != nil {
		t.Fatal(err)
	}

	// Copy the schema assets.
	schemaSrc := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	schemaData, err := os.ReadFile(schemaSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "completion-card.schema.json"), schemaData, 0644); err != nil {
		t.Fatal(err)
	}
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "context-alignment.schema.json"), contextData, 0644); err != nil {
		t.Fatal(err)
	}

	// Minimal light-tier completion card.
	cardYAML := `schema_version: "1"
task_id: TASK-BOUNDARY-ENFORCE-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/ui/login.ts
  manual_rationale: Touches a UI file
claim:
  fix_status: fixed
  summary: Wired boundary enforcement
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
	if err := os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(cardYAML), 0644); err != nil {
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

func TestVerifyBoundaryEnforceOffDoesNotBlock(t *testing.T) {
	setupBoundaryEnforceTestDir(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "off", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyBoundaryEnforceAdvisoryDoesNotBlock(t *testing.T) {
	setupBoundaryEnforceTestDir(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "advisory", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok under advisory mode, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}

	// Look for an advisory note recording the violations.
	foundNote := false
	for _, n := range result.AdmissionNotes {
		if strings.Contains(n, "boundary advisory:") && strings.Contains(n, "1 total violation") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected advisory note about boundary violations, got: %v", result.AdmissionNotes)
	}
}

func TestVerifyBoundaryEnforceBlockHighBlocksCritical(t *testing.T) {
	setupBoundaryEnforceTestDir(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "block_high", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok")
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
	if result.WithheldReason.BlockingPredicate != "boundary_violation" {
		t.Fatalf("expected blocking_predicate=boundary_violation, got %s", result.WithheldReason.BlockingPredicate)
	}
	if result.WithheldReason.FailureClass != "boundary_violation" {
		t.Fatalf("expected failure_class=boundary_violation, got %s", result.WithheldReason.FailureClass)
	}
	if result.WithheldReason.Class != "approval_scope_invalid" {
		t.Fatalf("expected class=approval_scope_invalid, got %s", result.WithheldReason.Class)
	}
}

func TestVerifyBoundaryEnforceBlockAllBlocksInfo(t *testing.T) {
	// A boundary violation at info severity should still block under
	// block_all (governed-deep).
	setupBoundaryEnforceTestDir(t)
	// Rewrite the policy to use info severity instead of critical.
	policyYAML := `version: 1
boundaries:
  - id: ui-cannot-access-db
    description: "UI must not import internal/db directly"
    from: "src/ui/**"
    to_import: "internal/db/**"
    action: deny
    severity: info
    applies_to_languages: [typescript]
`
	if err := os.WriteFile(filepath.Join("policies", "boundaries.yaml"), []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "block_all", "--json"}, &stdout, &stderr)
	if code == ExitOK {
		t.Fatalf("expected non-ok exit, got %d. stdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatal("expected not ok under block_all")
	}
	if result.WithheldReason == nil || result.WithheldReason.BlockingPredicate != "boundary_violation" {
		t.Fatalf("expected boundary_violation predicate, got %+v", result.WithheldReason)
	}
}

func TestVerifyBoundaryEnforceMissingPolicyIsNoop(t *testing.T) {
	// No policies/boundaries.yaml at all: enforce must not block.
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755); err != nil {
		t.Fatal(err)
	}
	schemaSrc := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	schemaData, err := os.ReadFile(schemaSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "completion-card.schema.json"), schemaData, 0644); err != nil {
		t.Fatal(err)
	}
	contextSrc := filepath.Join("..", "..", "schemas", "context-alignment.schema.json")
	contextData, err := os.ReadFile(contextSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "context-alignment.schema.json"), contextData, 0644); err != nil {
		t.Fatal(err)
	}
	cardYAML := `schema_version: "1"
task_id: TASK-BOUNDARY-NO-POLICY-001
tier: light
owner: alice
accountable: bob
evidence:
  files_changed:
    - src/main.ts
  manual_rationale: No policy
claim:
  fix_status: fixed
  summary: No policy present
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
	if err := os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(cardYAML), 0644); err != nil {
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "block_all", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d when policy missing, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok when policy missing, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyBoundaryEnforceApprovalsSuppressBlocking(t *testing.T) {
	setupBoundaryEnforceTestDir(t)

	// Add boundary_approvals to the card to suppress the rule.
	cardYAML := `schema_version: "1"
task_id: TASK-BOUNDARY-APPROVE-001
tier: light
owner: alice
accountable: bob
boundary_approvals:
  - rule_id: ui-cannot-access-db
    approver: alice
    approved_at: "2026-01-01T00:00:00Z"
    reason: approved for fixture
evidence:
  files_changed:
    - src/ui/login.ts
  manual_rationale: Touches a UI file
claim:
  fix_status: fixed
  summary: Boundary approval suppresses blocking
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
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "block_all", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d with approval, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok with approval, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyBoundaryEnforceBlockHighSkipsInfo(t *testing.T) {
	// Info-level violations must NOT block under block_high.
	setupBoundaryEnforceTestDir(t)
	policyYAML := `version: 1
boundaries:
  - id: ui-cannot-access-db
    description: "UI must not import internal/db directly"
    from: "src/ui/**"
    to_import: "internal/db/**"
    action: deny
    severity: info
    applies_to_languages: [typescript]
`
	if err := os.WriteFile(filepath.Join("policies", "boundaries.yaml"), []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "completion-card.yaml", "--boundary-enforce", "block_high", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d for info-severity under block_high, got %d. stdout: %s\nstderr: %s", ExitOK, code, stdout.String(), stderr.String())
	}

	var result VerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok for info-severity, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyProfileCIStrictBlocksHighBoundary(t *testing.T) {
	setupBoundaryEnforceTestDir(t)
	// ci-strict also enables contract-oracles, so provide a no-op
	// policy to isolate the boundary check.
	if err := os.WriteFile(filepath.Join("policies", "contract-oracle.yaml"),
		[]byte("version: 1\ngrep_rules: []\n"),
		0644); err != nil {
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
		t.Fatal("expected not ok under ci-strict with critical boundary violation")
	}
	if result.WithheldReason == nil {
		t.Fatal("expected withheld_reason")
	}
	if result.WithheldReason.BlockingPredicate != "boundary_violation" {
		t.Fatalf("expected blocking_predicate=boundary_violation, got %s", result.WithheldReason.BlockingPredicate)
	}
}

func TestVerifyProfileGovernedDeepBlocksAllBoundary(t *testing.T) {
	setupBoundaryEnforceTestDir(t)
	// Use a warning-level rule so block_all is the differentiator.
	policyYAML := `version: 1
boundaries:
  - id: ui-cannot-access-db
    description: "UI must not import internal/db directly"
    from: "src/ui/**"
    to_import: "internal/db/**"
    action: deny
    severity: warning
    applies_to_languages: [typescript]
`
	if err := os.WriteFile(filepath.Join("policies", "boundaries.yaml"), []byte(policyYAML), 0644); err != nil {
		t.Fatal(err)
	}
	// governed-deep also enables contract-oracles, so provide a no-op.
	if err := os.WriteFile(filepath.Join("policies", "contract-oracle.yaml"),
		[]byte("version: 1\ngrep_rules: []\n"),
		0644); err != nil {
		t.Fatal(err)
	}

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
		t.Fatal("expected not ok under governed-deep for warning boundary violation")
	}
	if result.WithheldReason == nil || result.WithheldReason.BlockingPredicate != "boundary_violation" {
		t.Fatalf("expected boundary_violation predicate, got %+v", result.WithheldReason)
	}
}

func TestVerifyProfileLightLocalDoesNotBlockBoundary(t *testing.T) {
	setupBoundaryEnforceTestDir(t)

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
	if !result.OK {
		t.Fatalf("expected ok under light-local even with boundary violation, got outcome=%s status=%s", result.AdmissionOutcome, result.AcceptanceStatus)
	}
}

func TestVerifyBoundaryEnforceInvalidValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--boundary-enforce", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid --boundary-enforce") {
		t.Fatalf("expected invalid value error, got: %s", stderr.String())
	}
}

func TestVerifyBoundaryEnforceMissingValueUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--card", "x.yaml", "--boundary-enforce"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "requires a value") {
		t.Fatalf("expected requires value error, got: %s", stderr.String())
	}
}

func TestVerifyHelpDocumentsBoundaryEnforce(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--boundary-enforce") {
		t.Fatalf("expected usage to mention --boundary-enforce, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--boundary-policy") {
		t.Fatalf("expected usage to mention --boundary-policy, got: %s", stderr.String())
	}
}

func TestVerifyBoundaryEnforceExplicitWinsOverProfile(t *testing.T) {
	// --profile ci-strict would block by default, but --boundary-enforce
	// off explicitly disables it.
	setupBoundaryEnforceTestDir(t)

	// ci-strict also enables contract-oracles, so we must provide a
	// policy that does not match the source file.
	if err := os.WriteFile(filepath.Join("policies", "contract-oracle.yaml"),
		[]byte("version: 1\ngrep_rules: []\n"),
		0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"verify", "--profile", "ci-strict", "--boundary-enforce", "off", "--card", "completion-card.yaml", "--json"}, &stdout, &stderr)
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

func TestExtractBoundaryApprovals(t *testing.T) {
	// Empty doc returns empty set.
	if got := extractBoundaryApprovals(nil); len(got) != 0 {
		t.Fatalf("expected empty set for nil doc, got %v", got)
	}
	doc := map[string]any{
		"boundary_approvals": []any{
			map[string]any{"rule_id": "r1", "approver": "alice", "approved_at": "2026-01-01T00:00:00Z", "reason": "ok"},
			map[string]any{"rule_id": "  r2  ", "approver": "bob", "approved_at": "2026-01-01T00:00:00Z", "reason": "ok"},
			// missing rule_id is skipped
			map[string]any{"approver": "alice"},
			// non-object entry is skipped
			"r3",
		},
	}
	got := extractBoundaryApprovals(doc)
	if !got["r1"] {
		t.Errorf("expected r1 approved")
	}
	if !got["r2"] {
		t.Errorf("expected r2 approved (whitespace trimmed)")
	}
	if got["r3"] {
		t.Errorf("did not expect r3 approved")
	}
}

func TestFilterBoundaryViolationsByEnforce(t *testing.T) {
	violations := []boundary.Violation{
		{RuleID: "r-info", Severity: boundary.SeverityInfo},
		{RuleID: "r-warn", Severity: boundary.SeverityWarning},
		{RuleID: "r-high", Severity: boundary.SeverityHigh},
		{RuleID: "r-crit", Severity: boundary.SeverityCritical},
	}
	approved := map[string]bool{"r-high": true}

	cases := []struct {
		name string
		mode string
		want []string
	}{
		{"off returns nil", "off", nil},
		{"advisory returns nil", "advisory", nil},
		{"block_high blocks high and critical, not info/warning, and respects approvals", "block_high", []string{"r-crit"}},
		{"block_all blocks everything except approved", "block_all", []string{"r-info", "r-warn", "r-crit"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterBoundaryViolationsByEnforce(violations, tc.mode, approved)
			gotIDs := make([]string, 0, len(got))
			for _, v := range got {
				gotIDs = append(gotIDs, v.RuleID)
			}
			if !equalStringSlices(gotIDs, tc.want) {
				t.Fatalf("mode=%s got %v, want %v", tc.mode, gotIDs, tc.want)
			}
		})
	}
}

// equalStringSlices is a tiny helper for test readability.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
