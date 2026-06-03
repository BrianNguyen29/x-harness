package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoverMarkdownOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--errors", "verification.status failed; missing evidence"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# Recovery Playbook (Review Required)") {
		t.Fatalf("expected playbook header, got:\n%s", out)
	}
	if !strings.Contains(out, "evidence_missing") {
		t.Fatalf("expected evidence_missing predicate, got:\n%s", out)
	}
	if !strings.Contains(out, "admission_failed") {
		t.Fatalf("expected admission_failed predicate, got:\n%s", out)
	}
}

func TestRecoverJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--errors", "verification.status failed; missing evidence", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Suggestions []struct {
			Predicate string `json:"predicate"`
			Route     struct {
				NextAction string `json:"next_action"`
				Owner      string `json:"owner"`
			} `json:"route"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d: %+v", len(result.Suggestions), result)
	}
	found := make(map[string]bool)
	for _, s := range result.Suggestions {
		found[s.Predicate] = true
	}
	if !found["evidence_missing"] {
		t.Fatalf("expected evidence_missing in suggestions: %+v", result)
	}
	if !found["admission_failed"] {
		t.Fatalf("expected admission_failed in suggestions: %+v", result)
	}
}

func TestRecoverUnknownFlagReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--force"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestRecoverSuccessOutcomeReturnsEmpty(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--errors", "something", "--outcome", "success"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "No recovery actions suggested.") {
		t.Fatalf("expected no suggestions for success outcome, got:\n%s", stdout.String())
	}
}

func TestRecoverySuggestJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recovery", "suggest", "--errors", "missing evidence", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Suggestions []struct {
			Predicate string `json:"predicate"`
			Route     struct {
				NextAction string `json:"next_action"`
				Owner      string `json:"owner"`
			} `json:"route"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d: %+v", len(result.Suggestions), result)
	}
	if result.Suggestions[0].Predicate != "evidence_missing" {
		t.Fatalf("expected evidence_missing, got %s", result.Suggestions[0].Predicate)
	}
}

func TestRecoveryUnsupportedSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recovery", "plan"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestRecoveryMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recovery"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestRecoverAutoNoTrace(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	traceDir := t.TempDir()
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "No trace events found") {
		t.Fatalf("expected no-trace message, got:\n%s", out)
	}
	if strings.Contains(out, "## ") {
		t.Fatalf("expected no suggestion sections, got:\n%s", out)
	}
}

func TestRecoverAutoAllSuccess(t *testing.T) {
	traceDir := t.TempDir()
	event := map[string]interface{}{
		"event_id":   "VE-1",
		"event_type": "verify_completed",
		"task_id":    "T1",
		"outcome":    "success",
		"created_at": "2026-05-29T00:00:00Z",
	}
	if _, err := AppendTrace(event, traceDir); err != nil {
		t.Fatalf("failed to append trace: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "No failures detected in trace") {
		t.Fatalf("expected no-failure message, got:\n%s", out)
	}
}

func TestRecoverAutoWithTrace(t *testing.T) {
	traceDir := t.TempDir()
	event := map[string]interface{}{
		"event_id":             "VE-1",
		"event_type":           "verify_completed",
		"task_id":              "T1",
		"outcome":              "failed",
		"blocking_predicate":   "admission_failed",
		"blocked_reason_class": "schema_or_policy_invalid",
		"notes":                []interface{}{"missing evidence", "typecheck failed"},
		"created_at":           "2026-05-29T00:00:00Z",
	}
	if _, err := AppendTrace(event, traceDir); err != nil {
		t.Fatalf("failed to append trace: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Suggestions []struct {
			Predicate string `json:"predicate"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Suggestions) == 0 {
		t.Fatalf("expected suggestions, got none")
	}
	foundEvidence := false
	foundTypecheck := false
	for _, s := range result.Suggestions {
		if s.Predicate == "evidence_missing" {
			foundEvidence = true
		}
		if s.Predicate == "typecheck_failed" {
			foundTypecheck = true
		}
	}
	if !foundEvidence {
		t.Fatalf("expected evidence_missing suggestion, got: %+v", result.Suggestions)
	}
	if !foundTypecheck {
		t.Fatalf("expected typecheck_failed suggestion, got: %+v", result.Suggestions)
	}
}

func TestRecoverAutoReadOnlyDoesNotMutate(t *testing.T) {
	traceDir := t.TempDir()
	event := map[string]interface{}{
		"event_id":   "VE-1",
		"event_type": "verify_completed",
		"task_id":    "T1",
		"outcome":    "failed",
		"notes":      []interface{}{"missing evidence"},
		"created_at": "2026-05-29T00:00:00Z",
	}
	if _, err := AppendTrace(event, traceDir); err != nil {
		t.Fatalf("failed to append trace: %v", err)
	}

	// Capture pre-run file listing
	entriesBefore, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace dir: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"recover", "--auto", "--trace-dir", traceDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	entriesAfter, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace dir after: %v", err)
	}
	if len(entriesAfter) != len(entriesBefore) {
		t.Fatalf("recover --auto mutated trace directory: before=%d after=%d", len(entriesBefore), len(entriesAfter))
	}
	out := stdout.String()
	if !strings.Contains(out, "Read-Only") {
		t.Fatalf("expected read-only label in output, got:\n%s", out)
	}
}

func TestRecoverDeterministicHeuristic(t *testing.T) {
	var stdout1 bytes.Buffer
	var stdout2 bytes.Buffer
	var stderr bytes.Buffer
	code1 := Run([]string{"recover", "--errors", "typecheck failed", "--json"}, &stdout1, &stderr)
	code2 := Run([]string{"recover", "--errors", "typecheck failed", "--json"}, &stdout2, &stderr)
	if code1 != ExitOK || code2 != ExitOK {
		t.Fatalf("expected both to succeed")
	}
	if stdout1.String() != stdout2.String() {
		t.Fatalf("expected deterministic output, got diff:\n%s\nvs\n%s", stdout1.String(), stdout2.String())
	}
}

// ---------- P2-S3: `xh recover --patch-card` ----------

// patchCardFixture returns a small completion card with missing
// handoff fields. It is the canonical "withheld" fixture used by
// the patch tests below.
func patchCardFixture(t *testing.T, dir string) string {
	t.Helper()
	card := `schema_version: "1"
task_id: TASK-PATCH-001
tier: standard
owner: alice
accountable: bob
claim:
  fix_status: partial
  summary: Recover flow patch
  evidence: []
verification:
  status: failed
  checks: []
admission:
  outcome: failed
acceptance_status: withheld
`
	if err := os.WriteFile(filepath.Join(dir, "card.yaml"), []byte(card), 0o644); err != nil {
		t.Fatalf("write card: %v", err)
	}
	return filepath.Join(dir, "card.yaml")
}

func TestRecoverPatchCardPreviewDoesNotMutate(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)

	before, err := os.ReadFile(card)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", card}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	after, err := os.ReadFile(card)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("preview must not mutate card.\nbefore:\n%s\nafter:\n%s", string(before), string(after))
	}

	out := stdout.String()
	if !strings.Contains(out, "dry_run: true") {
		t.Fatalf("expected dry_run=true in preview, got:\n%s", out)
	}
	if !strings.Contains(out, "confirmed: false") {
		t.Fatalf("expected confirmed=false in preview, got:\n%s", out)
	}
	if !strings.Contains(out, "would_set") {
		t.Fatalf("expected at least one would_set op, got:\n%s", out)
	}
}

func TestRecoverPatchCardConfirmMutates(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)

	// Snapshot the rest of the directory to make sure the patcher
	// does not create spurious files outside the card.
	beforeListing, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("list before: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", card, "--confirm"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	afterListing, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}

	// The patcher creates exactly one extra file: the .bak. We do
	// not allow any other collateral (e.g. sidecar metadata).
	if len(afterListing) != len(beforeListing)+1 {
		t.Fatalf("expected exactly one new file (the .bak), got before=%d after=%d", len(beforeListing), len(afterListing))
	}
	var backup string
	for _, e := range afterListing {
		name := e.Name()
		if name == "card.yaml" {
			continue
		}
		if strings.HasPrefix(name, "card.yaml.bak.") {
			backup = filepath.Join(tmp, name)
		} else {
			t.Fatalf("unexpected file created: %s", name)
		}
	}
	if backup == "" {
		t.Fatalf("expected a card.yaml.bak.<ts> file, got: %+v", afterListing)
	}

	data, err := os.ReadFile(card)
	if err != nil {
		t.Fatalf("read patched card: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "next_action:") {
		t.Fatalf("expected handoff.next_action to be set, got:\n%s", text)
	}
	if !strings.Contains(text, "owner: implementation-worker") {
		t.Fatalf("expected handoff.owner to be set, got:\n%s", text)
	}

	out := stdout.String()
	if !strings.Contains(out, "dry_run: false") {
		t.Fatalf("expected dry_run=false after confirm, got:\n%s", out)
	}
	if !strings.Contains(out, "confirmed: true") {
		t.Fatalf("expected confirmed=true after confirm, got:\n%s", out)
	}
	if !strings.Contains(out, "backup:") {
		t.Fatalf("expected backup path in output, got:\n%s", out)
	}
}

func TestRecoverPatchCardNoOverwrite(t *testing.T) {
	tmp := t.TempDir()
	card := `schema_version: "1"
task_id: TASK-PATCH-002
tier: light
owner: alice
accountable: bob
claim:
  fix_status: partial
  summary: Preserve user values
  evidence: []
verification:
  status: failed
  checks: []
admission:
  outcome: failed
acceptance_status: withheld
handoff:
  next_action: keep this value
  owner: keep-owner
`
	path := filepath.Join(tmp, "card.yaml")
	if err := os.WriteFile(path, []byte(card), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", path, "--confirm"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "next_action: keep this value") {
		t.Fatalf("user-provided handoff.next_action was overwritten:\n%s", text)
	}
	if !strings.Contains(text, "owner: keep-owner") {
		t.Fatalf("user-provided handoff.owner was overwritten:\n%s", text)
	}

	// The ops list should report both as skipped with reason.
	out := stdout.String()
	if !strings.Contains(out, "user-provided value preserved") {
		t.Fatalf("expected 'user-provided value preserved' reason, got:\n%s", out)
	}
}

func TestRecoverPatchCardUnclearRoute(t *testing.T) {
	tmp := t.TempDir()
	// Admitted card: the patcher must not second-guess an accepted
	// handoff, but it may still fill missing handoff.owner (schema
	// requires it). V1: skip both when card is admitted.
	card := `schema_version: "1"
task_id: TASK-PATCH-003
tier: light
owner: alice
accountable: bob
claim:
  fix_status: fixed
  summary: Already admitted
  evidence:
    - description: nothing to patch
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: shipped
  owner: alice
`
	path := filepath.Join(tmp, "card.yaml")
	if err := os.WriteFile(path, []byte(card), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", path, "--confirm"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "no deterministic patches applicable") {
		t.Fatalf("expected 'no deterministic patches applicable' note, got:\n%s", out)
	}
	if !strings.Contains(out, "admitted: true") {
		t.Fatalf("expected admitted=true in output, got:\n%s", out)
	}
}

func TestRecoverPatchCardMissingFile(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "no-such-card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", missing}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit %d for missing card, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "cannot read card") {
		t.Fatalf("expected 'cannot read card' error, got: %s", stderr.String())
	}
}

func TestRecoverPatchCardInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "card.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", path}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit %d for invalid yaml, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "cannot parse card") {
		t.Fatalf("expected 'cannot parse card' error, got: %s", stderr.String())
	}
}

func TestRecoverPatchCardMissingPathFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit %d for missing --patch-card value, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage in stderr, got: %s", stderr.String())
	}
}

func TestRecoverPatchCardUnknownFlag(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", card, "--force"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit %d for unknown flag, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected 'unknown flag' error, got: %s", stderr.String())
	}
}

func TestRecoverPatchCardEvidenceFlag(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", card, "--confirm", "--evidence", "src/main.go", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var out recoverPatchOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v\noutput: %s", err, stdout.String())
	}
	if !out.Confirmed {
		t.Fatalf("expected confirmed=true, got %+v", out)
	}

	// claim.evidence was empty in the fixture, so --evidence should
	// have been applied. evidence was absent (no `evidence:` key), so
	// evidence.files_changed should also have been applied.
	wantFields := map[string]bool{
		"handoff.next_action":    false,
		"handoff.owner":          false,
		"claim.evidence":         false,
		"evidence.files_changed": false,
	}
	for _, op := range out.Ops {
		if _, ok := wantFields[op.Field]; ok {
			if op.Action == "set" {
				wantFields[op.Field] = true
			}
		}
	}
	for f, seen := range wantFields {
		if !seen {
			t.Fatalf("expected %s to be set in ops, got: %+v", f, out.Ops)
		}
	}

	// Sanity: the file should now contain the evidence reference.
	data, err := os.ReadFile(card)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "src/main.go") {
		t.Fatalf("expected card to contain src/main.go, got:\n%s", string(data))
	}
}

func TestRecoverPatchCardEvidenceDoesNotOverwrite(t *testing.T) {
	tmp := t.TempDir()
	// Pre-populated claim.evidence must not be appended to.
	card := `schema_version: "1"
task_id: TASK-PATCH-004
tier: light
owner: alice
accountable: bob
claim:
  fix_status: partial
  summary: Existing claim evidence
  evidence:
    - description: keep me
verification:
  status: failed
  checks: []
admission:
  outcome: failed
acceptance_status: withheld
handoff:
  next_action: keep this
  owner: keep-owner
evidence:
  files_changed:
    - existing/path.go
`
	path := filepath.Join(tmp, "card.yaml")
	if err := os.WriteFile(path, []byte(card), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", path, "--confirm", "--evidence", "new/path.go", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "new/path.go") {
		t.Fatalf("patcher overwrote populated evidence.files_changed:\n%s", text)
	}
	if !strings.Contains(text, "existing/path.go") {
		t.Fatalf("original files_changed lost:\n%s", text)
	}
}

func TestRecoverPatchCardJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", card, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var out recoverPatchOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v\noutput: %s", err, stdout.String())
	}
	if out.Card != card {
		t.Fatalf("expected card=%s, got %s", card, out.Card)
	}
	if !out.DryRun {
		t.Fatalf("expected dry_run=true in preview, got %+v", out)
	}
	if out.Confirmed {
		t.Fatalf("expected confirmed=false in preview, got %+v", out)
	}
	if out.SchemaVersion == "" {
		t.Fatalf("expected schema_version to be set, got %+v", out)
	}
	if out.RoutePredicate == "" {
		t.Fatalf("expected route_predicate to be set, got %+v", out)
	}
	if len(out.Ops) == 0 {
		t.Fatalf("expected at least one op, got %+v", out)
	}
}

func TestRecoverPatchCardOutFilePreview(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)
	out := filepath.Join(tmp, "out", "patched-card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", card, "--out", out}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	// Original card must remain unchanged (we're in preview).
	before, err := os.ReadFile(card)
	if err != nil {
		t.Fatalf("read card: %v", err)
	}
	if !strings.Contains(string(before), "admission_failed") && !strings.Contains(string(before), "admission:") {
		t.Fatalf("original card looks corrupted:\n%s", string(before))
	}
	if strings.Contains(string(before), "next_action:") {
		t.Fatalf("preview must not mutate original card, but it contains handoff.next_action:\n%s", string(before))
	}

	// The --out file should exist and contain the proposed patch.
	patched, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if !strings.Contains(string(patched), "next_action:") {
		t.Fatalf("expected --out file to contain handoff.next_action, got:\n%s", string(patched))
	}
}

func TestRecoverPatchCardDoesNotTouchOtherFiles(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)
	// Drop a sibling "source" file that the patcher MUST NOT touch.
	srcFile := filepath.Join(tmp, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := []byte("package main\n")
	if err := os.WriteFile(srcFile, original, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Use --evidence pointing to a fake path so the patcher would
	// have reason to write elsewhere if it were so inclined.
	var stdout, stderr bytes.Buffer
	code := Run([]string{"recover", "--patch-card", card, "--confirm", "--evidence", "src/main.go"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	got, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("read src: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("patcher touched source file src/main.go: before=%q after=%q", string(original), string(got))
	}
}

func TestRecoverPatchCardBackupSurvivesSecondRun(t *testing.T) {
	tmp := t.TempDir()
	card := patchCardFixture(t, tmp)

	// First confirm run: creates .bak, patches card.
	var stdout1, stderr1 bytes.Buffer
	code1 := Run([]string{"recover", "--patch-card", card, "--confirm"}, &stdout1, &stderr1)
	if code1 != ExitOK {
		t.Fatalf("first run failed: %d stderr=%s", code1, stderr1.String())
	}
	first, err := os.ReadFile(card)
	if err != nil {
		t.Fatalf("read after first: %v", err)
	}

	// Second confirm run: nothing to patch (harness path is a no-op
	// now), but the existing .bak must remain untouched.
	var stdout2, stderr2 bytes.Buffer
	code2 := Run([]string{"recover", "--patch-card", card, "--confirm"}, &stdout2, &stderr2)
	if code2 != ExitOK {
		t.Fatalf("second run failed: %d stderr=%s", code2, stderr2.String())
	}
	second, err := os.ReadFile(card)
	if err != nil {
		t.Fatalf("read after second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("idempotent re-run should not change the card.\nfirst:\n%s\nsecond:\n%s", string(first), string(second))
	}
}
