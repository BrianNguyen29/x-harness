package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestContextContractPlain(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "--contract"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	out := stdout.String()
	for _, phrase := range []string{
		"Completion is admitted, not claimed",
		"verifier is read-only",
		"Success is the only accepted outcome",
		"Canonical tiers",
		"PGV is advisory-only",
	} {
		if !strings.Contains(out, phrase) {
			t.Fatalf("expected output to contain %q, got:\n%s", phrase, out)
		}
	}
}

func TestContextContractJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "--contract", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var contract struct {
		Facts []struct {
			Rule        string `json:"rule"`
			Description string `json:"description"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &contract); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if len(contract.Facts) == 0 {
		t.Fatal("expected at least one fact in JSON output")
	}
}

func TestContextUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "--unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown context subcommand") {
		t.Fatalf("expected unknown subcommand error, got %q", stderr.String())
	}
}

func TestContextContractCoreFactsCount(t *testing.T) {
	contract := CoreContract()
	if len(contract.Facts) != 5 {
		t.Fatalf("expected 5 core contract facts, got %d", len(contract.Facts))
	}
	expectedRules := []string{
		"completion_admitted_not_claimed",
		"verifier_read_only",
		"success_only_accepted",
		"canonical_tiers",
		"pgv_advisory_only",
	}
	for i, rule := range expectedRules {
		if contract.Facts[i].Rule != rule {
			t.Fatalf("expected fact[%d].Rule == %q, got %q", i, rule, contract.Facts[i].Rule)
		}
	}
}
