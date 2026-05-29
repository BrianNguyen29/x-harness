package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
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

// --- card init tests ---

func TestCardInitLightMissingRequiredFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"card", "init", "--tier", "light"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "missing required fields") {
		t.Fatalf("expected missing required fields error, got: %s", stderrStr)
	}
	if !strings.Contains(stderrStr, "--task-id") {
		t.Fatalf("expected --task-id in error, got: %s", stderrStr)
	}
}

func TestCardInitLightMinimal(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"card", "init",
		"--tier", "light",
		"--task-id", "test-task",
		"--owner", "fixer",
		"--accountable", "fixer",
		"--summary", "test summary",
		"--file", "foo.go",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected card file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "task_id: test-task") {
		t.Fatalf("expected task_id in card, got:\n%s", content)
	}
	if !strings.Contains(content, "tier: light") {
		t.Fatalf("expected tier light, got:\n%s", content)
	}
	if strings.Contains(content, "done_checklist") {
		t.Fatalf("light tier should not include done_checklist, got:\n%s", content)
	}
	if strings.Contains(content, "prediction") {
		t.Fatalf("light tier should not include prediction, got:\n%s", content)
	}
}

func TestCardInitStandardIncludesDoneChecklistAndPrediction(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"card", "init",
		"--tier", "standard",
		"--task-id", "std-task",
		"--owner", "fixer",
		"--accountable", "fixer",
		"--summary", "standard summary",
		"--file", "foo.go",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected card file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "done_checklist:") {
		t.Fatalf("standard tier should include done_checklist, got:\n%s", content)
	}
	if !strings.Contains(content, "prediction:") {
		t.Fatalf("standard tier should include prediction, got:\n%s", content)
	}
	if !strings.Contains(content, "claim: change produces intended effect") {
		t.Fatalf("expected prediction claim, got:\n%s", content)
	}
}

func TestCardInitInvalidTier(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"card", "init",
		"--tier", "invalid",
		"--task-id", "t",
		"--owner", "o",
		"--accountable", "a",
		"--summary", "s",
		"--file", "f.go",
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid tier") {
		t.Fatalf("expected invalid tier error, got: %s", stderr.String())
	}
}

func TestCardInitLightSchemaValid(t *testing.T) {
	root, err := repo.FindRoot("")
	if err != nil {
		t.Skipf("cannot find repo root: %v", err)
	}
	schemaPath := filepath.Join(root, "schemas", "completion-card.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		t.Fatalf("cannot compile schema: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"card", "init",
		"--tier", "light",
		"--task-id", "light-schema-task",
		"--owner", "fixer",
		"--accountable", "fixer",
		"--summary", "light test",
		"--file", "foo.go",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var doc any
	if err := loader.LoadDocument(outPath, &doc); err != nil {
		t.Fatalf("cannot load generated card: %v", err)
	}
	if err := v.Validate(doc); err != nil {
		t.Fatalf("generated light card failed schema validation: %v", err)
	}
}

func TestCardInitStandardSchemaValid(t *testing.T) {
	root, err := repo.FindRoot("")
	if err != nil {
		t.Skipf("cannot find repo root: %v", err)
	}
	schemaPath := filepath.Join(root, "schemas", "completion-card.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		t.Fatalf("cannot compile schema: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"card", "init",
		"--tier", "standard",
		"--task-id", "std-schema-task",
		"--owner", "fixer",
		"--accountable", "fixer",
		"--summary", "standard test",
		"--file", "foo.go",
		"--command", "go test ./...",
		"--manual-rationale", "tested locally",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var doc any
	if err := loader.LoadDocument(outPath, &doc); err != nil {
		t.Fatalf("cannot load generated card: %v", err)
	}
	if err := v.Validate(doc); err != nil {
		t.Fatalf("generated standard card failed schema validation: %v", err)
	}
}

func TestCardInitDeepSchemaValid(t *testing.T) {
	root, err := repo.FindRoot("")
	if err != nil {
		t.Skipf("cannot find repo root: %v", err)
	}
	schemaPath := filepath.Join(root, "schemas", "completion-card.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		t.Fatalf("cannot compile schema: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "card.yaml")

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"card", "init",
		"--tier", "deep",
		"--task-id", "deep-schema-task",
		"--owner", "fixer",
		"--accountable", "fixer",
		"--summary", "deep test",
		"--file", "foo.go",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var doc any
	if err := loader.LoadDocument(outPath, &doc); err != nil {
		t.Fatalf("cannot load generated card: %v", err)
	}
	if err := v.Validate(doc); err != nil {
		t.Fatalf("generated deep card failed schema validation: %v", err)
	}
}

func TestCardInitStdoutWhenNoOut(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"card", "init",
		"--tier", "light",
		"--task-id", "stdout-task",
		"--owner", "fixer",
		"--accountable", "fixer",
		"--summary", "stdout test",
		"--file", "foo.go",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "task_id: stdout-task") {
		t.Fatalf("expected card on stdout, got:\n%s", out)
	}
}
