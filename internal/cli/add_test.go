package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func readYAML(path string) (map[string]interface{}, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func TestAddClaimDefaultName(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "claim-test.yaml")

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add", "claim", "fix_status=fixed,summary=did work", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Added claim ->") {
		t.Fatalf("expected Added claim ->, got: %s", stdout.String())
	}

	data, err := readYAML(outPath)
	if err != nil {
		t.Fatalf("failed to read yaml: %v", err)
	}
	if data["fix_status"] != "fixed" {
		t.Fatalf("expected fix_status=fixed, got %v", data["fix_status"])
	}
	if data["summary"] != "did work" {
		t.Fatalf("expected summary=did work, got %v", data["summary"])
	}
	id, ok := data["id"].(string)
	if !ok || !strings.Contains(id, "CLAIM-") {
		t.Fatalf("expected id to contain CLAIM-, got %v", data["id"])
	}
	if data["created_at"] == nil {
		t.Fatalf("expected created_at to be defined")
	}
}

func TestAddEvidence(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "evidence-test.yaml")

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add", "evidence", "owner=alice", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Added evidence ->") {
		t.Fatalf("expected Added evidence ->, got: %s", stdout.String())
	}

	data, err := readYAML(outPath)
	if err != nil {
		t.Fatalf("failed to read yaml: %v", err)
	}
	if data["owner"] != "alice" {
		t.Fatalf("expected owner=alice, got %v", data["owner"])
	}
	id, ok := data["id"].(string)
	if !ok || !strings.Contains(id, "EVIDENCE-") {
		t.Fatalf("expected id to contain EVIDENCE-, got %v", data["id"])
	}
}

func TestAddCompletionCard(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "completion-card-test.yaml")

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add", "completion-card", "task_id=T1,tier=light", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Added completion-card ->") {
		t.Fatalf("expected Added completion-card ->, got: %s", stdout.String())
	}

	data, err := readYAML(outPath)
	if err != nil {
		t.Fatalf("failed to read yaml: %v", err)
	}
	if data["task_id"] != "T1" {
		t.Fatalf("expected task_id=T1, got %v", data["task_id"])
	}
	if data["tier"] != "light" {
		t.Fatalf("expected tier=light, got %v", data["tier"])
	}
}

func TestAddEdgeValueWithColonAndNewline(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "edge-test.yaml")

	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add", "claim", "description=foo: bar\nbaz: qux", "--out", outPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	data, err := readYAML(outPath)
	if err != nil {
		t.Fatalf("failed to read yaml: %v", err)
	}
	if data["description"] != "foo: bar\nbaz: qux" {
		t.Fatalf("expected description with colon/newline, got %v", data["description"])
	}
}

func TestAddMissingModule(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestAddInvalidModule(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestAddUnknownFlag(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add", "claim", "--force"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestAddMissingOutValue(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"add", "claim", "--out"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}
