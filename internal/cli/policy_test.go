package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPolicyMatrixTextOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "matrix"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	out := stdout.String()

	// Header
	if !strings.Contains(out, "# x-harness Policy Matrix") {
		t.Fatalf("expected matrix header, got:\n%s", out)
	}
	if !strings.Contains(out, "version: 1") {
		t.Fatalf("expected version: 1, got:\n%s", out)
	}
	if !strings.Contains(out, "rules: 27") {
		t.Fatalf("expected rules count, got:\n%s", out)
	}

	// Status group headers (text mode groups by status, not by id order)
	for _, group := range []string{"runtime_blocking", "advisory", "off_by_default"} {
		if !strings.Contains(out, group+" (") {
			t.Fatalf("expected status group %q in output, got:\n%s", group, out)
		}
	}

	// Core rules must show up
	for _, id := range []string{
		"admission.evidence_floor",
		"admission.schema_validation",
		"pgv.suggestion",
		"mutation_guard.verifier_read_only",
		"context_floor.stale_ground",
	} {
		if !strings.Contains(out, id) {
			t.Fatalf("expected rule %q in output, got:\n%s", id, out)
		}
	}
}

func TestPolicyMatrixJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "matrix", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var matrix Matrix
	if err := json.Unmarshal(stdout.Bytes(), &matrix); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}

	if matrix.Version != 1 {
		t.Fatalf("expected version 1, got %d", matrix.Version)
	}
	if len(matrix.Rules) == 0 {
		t.Fatal("expected non-empty rules list")
	}

	// Rules must be sorted by id for determinism.
	for i := 1; i < len(matrix.Rules); i++ {
		if matrix.Rules[i-1].ID >= matrix.Rules[i].ID {
			t.Fatalf("rules not sorted by id at index %d: %q >= %q", i, matrix.Rules[i-1].ID, matrix.Rules[i].ID)
		}
	}

	// Validate required fields and enum membership.
	allowedStatus := map[string]bool{
		matrixStatusRuntimeBlocking: true,
		matrixStatusAdvisory:        true,
		matrixStatusOffByDefault:    true,
		matrixStatusExperimental:    true,
	}
	allowedSource := map[string]bool{
		matrixSourceCurated: true,
		matrixSourcePolicy:  true,
	}
	allowedProfile := map[string]bool{
		"light-local":   true,
		"ci-standard":   true,
		"ci-strict":     true,
		"governed-deep": true,
	}
	seen := map[string]bool{}
	for _, rule := range matrix.Rules {
		if rule.ID == "" {
			t.Fatalf("rule missing id: %+v", rule)
		}
		if seen[rule.ID] {
			t.Fatalf("duplicate rule id: %q", rule.ID)
		}
		seen[rule.ID] = true
		if !allowedStatus[rule.Status] {
			t.Fatalf("rule %q has unknown status %q", rule.ID, rule.Status)
		}
		if !allowedSource[rule.Source] {
			t.Fatalf("rule %q has unknown source %q", rule.ID, rule.Source)
		}
		for _, p := range rule.Profiles {
			if !allowedProfile[p] {
				t.Fatalf("rule %q has unknown profile %q", rule.ID, p)
			}
		}
	}

	// Core rules must be present
	for _, id := range []string{
		"admission.evidence_floor",
		"admission.schema_validation",
		"pgv.suggestion",
		"mutation_guard.verifier_read_only",
		"context_floor.stale_ground",
	} {
		if !seen[id] {
			t.Fatalf("expected rule %q in JSON output, missing", id)
		}
	}

	// runtime_blocking rules must have admission_authority == true
	for _, rule := range matrix.Rules {
		if rule.Status == matrixStatusRuntimeBlocking {
			if rule.AdmissionAuthority == nil || !*rule.AdmissionAuthority {
				t.Fatalf("runtime_blocking rule %q should have admission_authority=true, got %+v", rule.ID, rule.AdmissionAuthority)
			}
		}
	}
}

func TestPolicyMatrixJSONDeterministic(t *testing.T) {
	var first bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"policy", "matrix", "--json"}, &first, &stderr); code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}

	// Run again and compare; the output must be byte-identical (deterministic).
	var second bytes.Buffer
	if code := Run([]string{"policy", "matrix", "--json"}, &second, &stderr); code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if first.String() != second.String() {
		t.Fatalf("matrix JSON output is not deterministic across runs.\nfirst:\n%s\nsecond:\n%s", first.String(), second.String())
	}
}

func TestPolicyMatrixJSONMatchesSchemaShape(t *testing.T) {
	// This test asserts the on-the-wire JSON shape matches the
	// schemas/policy-matrix.schema.json contract at the field level.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"policy", "matrix", "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}

	var raw struct {
		Version int `json:"version"`
		Rules   []struct {
			ID                 string   `json:"id"`
			Description        string   `json:"description"`
			PolicyFile         string   `json:"policy_file"`
			RuntimeModule      string   `json:"runtime_module"`
			Status             string   `json:"status"`
			EnabledByDefault   bool     `json:"enabled_by_default"`
			AdmissionAuthority *bool    `json:"admission_authority"`
			Profiles           []string `json:"profiles"`
			EnabledByFlags     []string `json:"enabled_by_flags"`
			Fixtures           []string `json:"fixtures"`
			Source             string   `json:"source"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}
	if raw.Version < 1 {
		t.Fatalf("expected version >= 1, got %d", raw.Version)
	}
	if len(raw.Rules) == 0 {
		t.Fatal("expected non-empty rules list")
	}
	for _, r := range raw.Rules {
		if r.ID == "" || r.Status == "" || r.Source == "" {
			t.Fatalf("rule missing required field: %+v", r)
		}
	}
}

func TestPolicyMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got %q", stderr.String())
	}
}

func TestPolicyUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown policy subcommand") {
		t.Fatalf("expected unknown subcommand error, got %q", stderr.String())
	}
}

func TestPolicyMatrixUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "matrix", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got %q", stderr.String())
	}
}

func TestPolicyMatrixUnexpectedArg(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "matrix", "extra"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unexpected argument") {
		t.Fatalf("expected unexpected argument error, got %q", stderr.String())
	}
}

func TestPolicyExplainMissingID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "explain"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "rule-id is required") {
		t.Fatalf("expected missing id error, got %q", stderr.String())
	}
}

func TestPolicyExplainStubReturnsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "explain", "admission.evidence_floor"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "admission.evidence_floor") {
		t.Fatalf("expected rule id in output, got %q", out)
	}
	if !strings.Contains(out, "status:") {
		t.Fatalf("expected status field in output, got %q", out)
	}
}

func TestPolicyExplainJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "explain", "admission.evidence_floor", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var rule MatrixRule
	if err := json.Unmarshal(stdout.Bytes(), &rule); err != nil {
		t.Fatalf("expected valid JSON, got %v\noutput: %s", err, stdout.String())
	}
	if rule.ID != "admission.evidence_floor" {
		t.Fatalf("expected id=admission.evidence_floor, got %s", rule.ID)
	}
	if rule.Status != "runtime_blocking" {
		t.Fatalf("expected status=runtime_blocking, got %s", rule.Status)
	}
}

func TestPolicyExplainUnknownRule(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "explain", "definitely.not.a.real.rule"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "not found in policy matrix") {
		t.Fatalf("expected not found error, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "xh policy matrix") {
		t.Fatalf("expected hint to run matrix, got %q", stderr.String())
	}
}

func TestPolicyExplainUnexpectedArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"policy", "explain", "admission.evidence_floor", "extra"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unexpected argument") {
		t.Fatalf("expected unexpected argument error, got %q", stderr.String())
	}
}
