package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAdaptersMatrixJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "matrix", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Adapters []struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Capabilities []string `json:"capabilities"`
			Formats      []string `json:"formats"`
		} `json:"adapters"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}

	expected := []string{"opencode", "claude-code", "cursor", "generic", "antigravity"}
	found := map[string]bool{}
	for _, a := range result.Adapters {
		found[a.Name] = true
	}
	for _, name := range expected {
		if !found[name] {
			t.Fatalf("expected adapter %s to be present in matrix", name)
		}
	}
}

func TestAdaptersMatrixText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "matrix"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	expected := []string{"opencode", "claude-code", "cursor", "generic", "antigravity"}
	for _, name := range expected {
		if !strings.Contains(out, name) {
			t.Fatalf("expected adapter %s in text output, got: %s", name, out)
		}
	}
	if !strings.Contains(out, "adapters: 5") {
		t.Fatalf("expected adapter count in output, got: %s", out)
	}
}

func TestAdaptersMatrixUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"adapters", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown adapters subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestAdaptersMissingSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"adapters"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}

func TestAdaptersEvalText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "eval"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	expected := []string{"opencode", "claude-code", "cursor", "generic", "antigravity"}
	for _, name := range expected {
		if !strings.Contains(out, name) {
			t.Fatalf("expected adapter %s in eval output, got: %s", name, out)
		}
	}
	if !strings.Contains(out, "pass: 5/5") {
		t.Fatalf("expected pass count in output, got: %s", out)
	}
}

func TestAdaptersEvalJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "eval", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Adapters  []adapterEvalResult `json:"adapters"`
		PassCount int                 `json:"pass_count"`
		Total     int                 `json:"total"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.PassCount != 5 {
		t.Fatalf("expected pass_count 5, got %d", result.PassCount)
	}
	if result.Total != 5 {
		t.Fatalf("expected total 5, got %d", result.Total)
	}
	expected := map[string]bool{"opencode": false, "claude-code": false, "cursor": false, "generic": false, "antigravity": false}
	for _, a := range result.Adapters {
		if _, ok := expected[a.Name]; !ok {
			t.Fatalf("unexpected adapter name: %s", a.Name)
		}
		expected[a.Name] = true
		if !a.OK {
			t.Fatalf("expected adapter %s to be ok", a.Name)
		}
		if !a.HasReadme {
			t.Fatalf("expected adapter %s to have readme", a.Name)
		}
		if !a.HasCaps {
			t.Fatalf("expected adapter %s to have capabilities", a.Name)
		}
		if !a.HasFormats {
			t.Fatalf("expected adapter %s to have formats", a.Name)
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("expected adapter %s to be present", name)
		}
	}
}
