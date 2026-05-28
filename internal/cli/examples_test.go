package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExamplesVerifyJSONStructure(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples", "verify", "--json"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result ExamplesVerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}

	if !result.OK {
		t.Fatal("expected ok=true")
	}
	if result.Total == 0 {
		t.Fatal("expected at least one example")
	}
	if result.Failed != 0 {
		t.Fatalf("expected failed=0, got %d", result.Failed)
	}
	if result.Passed != result.Total {
		t.Fatalf("expected all examples passed, got passed=%d total=%d", result.Passed, result.Total)
	}
	if len(result.Results) != result.Total {
		t.Fatalf("expected %d results, got %d", result.Total, len(result.Results))
	}
	if result.Passed+result.Failed != result.Total {
		t.Fatalf("passed+failed (%d+%d) should equal total (%d)", result.Passed, result.Failed, result.Total)
	}

	for _, r := range result.Results {
		if r.Name == "" {
			t.Fatal("expected result name to be non-empty")
		}
		if r.Outcome == "" {
			t.Fatal("expected outcome to be non-empty")
		}
		if r.AcceptanceStatus == "" {
			t.Fatal("expected acceptance_status to be non-empty")
		}
	}
}

func TestExamplesVerifyTextOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples", "verify"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Golden examples:") {
		t.Fatalf("expected text header, got:\n%s", out)
	}
	if !strings.Contains(out, "All golden examples passed.") {
		t.Fatalf("expected success summary, got:\n%s", out)
	}
}

func TestExamplesVerifyMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got %q", stderr.String())
	}
}

func TestExamplesVerifyUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown examples subcommand") {
		t.Fatalf("expected unknown subcommand message, got %q", stderr.String())
	}
}

func TestExamplesVerifySuiteRegression(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples", "verify", "--suite=regression", "--json"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result ExamplesVerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}

	if result.Total == 0 {
		t.Fatal("expected at least one regression example")
	}
	if result.Failed != 0 {
		t.Fatalf("expected failed=0, got %d", result.Failed)
	}
	for _, r := range result.Results {
		if !strings.HasPrefix(r.Name, "regression/") {
			t.Fatalf("expected regression suite prefix, got %q", r.Name)
		}
	}
}

func TestExamplesVerifySuiteCapability(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples", "verify", "--suite=capability", "--json"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result ExamplesVerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}

	if result.Total == 0 {
		t.Fatal("expected at least one capability example")
	}
	if result.Failed != 0 {
		t.Fatalf("expected failed=0, got %d", result.Failed)
	}
	for _, r := range result.Results {
		if !strings.HasPrefix(r.Name, "capability/") {
			t.Fatalf("expected capability suite prefix, got %q", r.Name)
		}
	}
}

func TestExamplesVerifySuiteAdversarial(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples", "verify", "--suite=adversarial", "--json"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result ExamplesVerifyResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}

	if result.Total == 0 {
		t.Fatal("expected at least one adversarial example")
	}
	if result.Failed != 0 {
		t.Fatalf("expected failed=0, got %d", result.Failed)
	}
	for _, r := range result.Results {
		if !strings.HasPrefix(r.Name, "adversarial/") {
			t.Fatalf("expected adversarial suite prefix, got %q", r.Name)
		}
	}
}

func TestExamplesVerifyInvalidSuite(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"examples", "verify", "--suite=invalid"}, &stdout, &stderr)

	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid suite") {
		t.Fatalf("expected invalid suite error, got %q", stderr.String())
	}
}
