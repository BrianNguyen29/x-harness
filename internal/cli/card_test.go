package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCardGenerateCreatesFile(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "admission-card.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"card", "generate", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected card file to exist: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "admission card written to") {
		t.Fatalf("expected success message, got: %s", out)
	}
}

func TestCardGenerateJSONFormat(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "admission-card.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"card", "generate", "--out", outPath, "--format", "json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected card file to exist: %v", err)
	}

	var card struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &card); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if card.SchemaVersion == "" {
		t.Fatal("expected schema_version in generated card")
	}
}

func TestCardVerifyValidCard(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "admission-card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"card", "generate", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("generate failed: %d", code)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"card", "verify", "--card", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "card: valid") {
		t.Fatalf("expected valid card, got: %s", out)
	}
}

func TestCardVerifyValidCardJSON(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "admission-card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"card", "generate", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("generate failed: %d", code)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"card", "verify", "--card", outPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got: %s", stdout.String())
	}
}

func TestCardVerifyMissingRefFails(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "admission-card.yaml")

	// Create a card with a non-existent ref
	cardContent := `schema_version: "1.0"
generated_at: "2024-01-01T00:00:00Z"
x_harness_card:
  source_refs:
    - path: DOES_NOT_EXIST_12345
      exists: true
  status:
    ok: true
`
	if err := os.WriteFile(outPath, []byte(cardContent), 0644); err != nil {
		t.Fatalf("failed to write test card: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"card", "verify", "--card", outPath, "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}

	var result struct {
		OK          bool     `json:"ok"`
		MissingRefs []string `json:"missing_refs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if result.OK {
		t.Fatal("expected ok=false for missing refs")
	}
	if len(result.MissingRefs) == 0 {
		t.Fatal("expected missing refs")
	}
}

func TestCardUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"card", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown card subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestCardGenerateBadFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"card", "generate", "--format", "xml"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestCardVerifyExtensionlessFile(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "admission-card")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"card", "generate", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("generate failed: %d. stderr: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"card", "verify", "--card", outPath, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("verify failed: %d. stderr: %s", code, stderr.String())
	}

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got: %s", stdout.String())
	}
}
