package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntakeContractMissingIDUsageError covers the safe V1 rule that
// --id is required. The CLI must return a usage error (not a generic
// panic or success with empty fields).
func TestIntakeContractMissingIDUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "contract", "--goal", "x", "--acceptance", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--id is required") {
		t.Fatalf("expected --id required error, got: %s", stderr.String())
	}
}

// TestIntakeContractMissingGoalUsageError covers the safe V1 rule that
// --goal is required.
func TestIntakeContractMissingGoalUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "contract", "--id", "x", "--acceptance", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--goal is required") {
		t.Fatalf("expected --goal required error, got: %s", stderr.String())
	}
}

// TestIntakeContractMissingAcceptanceUsageError covers the safe V1 rule
// that at least one --acceptance is required.
func TestIntakeContractMissingAcceptanceUsageError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "contract", "--id", "x", "--goal", "y"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--acceptance is required") {
		t.Fatalf("expected --acceptance required error, got: %s", stderr.String())
	}
}

// TestIntakeContractYAMLToStdout covers the deterministic YAML output
// path: structured flags -> normalized record printed to stdout. The
// record must include all required fields and the optional ones when
// provided.
func TestIntakeContractYAMLToStdout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "intake-lite",
		"--goal", "ship the safe V1 slice",
		"--visible", "true",
		"--non-goal", "block admission",
		"--non-goal", "add new admission predicate",
		"--acceptance", "advisory note emitted on standard",
		"--acceptance", "no --from flag is added",
		"--protected-behavior", "intent_ref is never required",
		"--ambiguity", "none",
		"--note", "first vertical slice",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"schema_version: \"1\"",
		"id: intake-lite",
		"product_goal: ship the safe V1 slice",
		"user_visible_change: true",
		"- block admission",
		"- add new admission predicate",
		"- id: ac-1",
		"statement: advisory note emitted on standard",
		"- id: ac-2",
		"statement: no --from flag is added",
		"- intent_ref is never required",
		"status: none",
		"notes: first vertical slice",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

// TestIntakeContractJSONToStdout covers the deterministic JSON output
// path. The record must include the required fields and round-trip
// cleanly through encoding/json.
func TestIntakeContractJSONToStdout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "intake-lite",
		"--goal", "ship the safe V1 slice",
		"--visible", "false",
		"--acceptance", "advisory note emitted on standard",
		"--ambiguity", "partial",
		"--ambiguity-question", "Should intent_ref be deep-only?",
		"--json",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
	if doc["schema_version"] != "1" {
		t.Fatalf("expected schema_version=1, got %v", doc["schema_version"])
	}
	if doc["id"] != "intake-lite" {
		t.Fatalf("expected id=intake-lite, got %v", doc["id"])
	}
	if doc["product_goal"] != "ship the safe V1 slice" {
		t.Fatalf("unexpected product_goal: %v", doc["product_goal"])
	}
	// user_visible_change false is explicit and must round-trip as
	// the boolean false (not null) so the schema accepts it as a
	// non-user-visible declaration.
	uv, ok := doc["user_visible_change"].(bool)
	if !ok || uv != false {
		t.Fatalf("expected user_visible_change=false, got %v (%T)", doc["user_visible_change"], doc["user_visible_change"])
	}
	ambiguity, _ := doc["ambiguity"].(map[string]any)
	if ambiguity == nil {
		t.Fatalf("expected ambiguity object, got: %v", doc["ambiguity"])
	}
	if ambiguity["status"] != "partial" {
		t.Fatalf("expected ambiguity.status=partial, got %v", ambiguity["status"])
	}
}

// TestIntakeContractWritesFile covers --output: the parent directory
// must already exist (safe V1 rule) and the file content must equal the
// YAML/JSON rendered to stdout.
func TestIntakeContractWritesFile(t *testing.T) {
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "intent.yaml")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "intake-lite",
		"--goal", "ship the safe V1 slice",
		"--acceptance", "advisory note emitted on standard",
		"--output", out,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout when --output is set, got: %s", stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	for _, want := range []string{
		"id: intake-lite",
		"product_goal: ship the safe V1 slice",
		"statement: advisory note emitted on standard",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("expected file to contain %q, got:\n%s", want, string(data))
		}
	}
}

// TestIntakeContractOutputMissingParent covers the safe V1 rule that the
// output parent directory must already exist. The CLI must surface a
// clear error rather than auto-creating directories.
func TestIntakeContractOutputMissingParent(t *testing.T) {
	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "does", "not", "exist", "intent.yaml")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "intake-lite",
		"--goal", "ship the safe V1 slice",
		"--acceptance", "advisory note emitted on standard",
		"--output", out,
	}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "parent directory does not exist") {
		t.Fatalf("expected missing parent error, got: %s", stderr.String())
	}
}

// TestIntakeContractUnknownFlag covers the safe V1 rule that unknown
// flags produce a usage error. The flag is rejected even when --id and
// --goal are provided.
func TestIntakeContractUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "x",
		"--goal", "y",
		"--acceptance", "z",
		"--from", "issue.md",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}

// TestIntakeContractInvalidVisible covers --visible strict parsing. The
// CLI must reject anything other than true/false variants.
func TestIntakeContractInvalidVisible(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "x",
		"--goal", "y",
		"--acceptance", "z",
		"--visible", "maybe",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--visible") {
		t.Fatalf("expected --visible error, got: %s", stderr.String())
	}
}

// TestIntakeContractInvalidAmbiguity covers --ambiguity strict parsing.
// The CLI must reject anything other than none/unresolved/partial.
func TestIntakeContractInvalidAmbiguity(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "x",
		"--goal", "y",
		"--acceptance", "z",
		"--ambiguity", "maybe",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--ambiguity") {
		t.Fatalf("expected --ambiguity error, got: %s", stderr.String())
	}
}

// TestIntakeContractCommaDelimited covers the documented behavior that
// --non-goal/--acceptance/--protected-behavior accept comma-delimited
// values as well as repeatable flags.
func TestIntakeContractCommaDelimited(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "contract",
		"--id", "intake-lite",
		"--goal", "ship",
		"--non-goal", "a, b, c",
		"--acceptance", "x, y",
		"--json",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	ng, _ := doc["non_goals"].([]any)
	if len(ng) != 3 {
		t.Fatalf("expected 3 non_goals, got %d (%v)", len(ng), ng)
	}
	acc, _ := doc["acceptance_criteria"].([]any)
	if len(acc) != 2 {
		t.Fatalf("expected 2 acceptance_criteria, got %d (%v)", len(acc), acc)
	}
}
