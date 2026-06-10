package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

	expected := []string{"opencode", "claude-code", "cursor", "generic", "antigravity", "codex"}
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
	expected := []string{"opencode", "claude-code", "cursor", "generic", "antigravity", "codex"}
	for _, name := range expected {
		if !strings.Contains(out, name) {
			t.Fatalf("expected adapter %s in text output, got: %s", name, out)
		}
	}
	if !strings.Contains(out, "adapters: 6") {
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
	expected := []string{"opencode", "claude-code", "cursor", "generic", "antigravity", "codex"}
	for _, name := range expected {
		if !strings.Contains(out, name) {
			t.Fatalf("expected adapter %s in eval output, got: %s", name, out)
		}
	}
	if !strings.Contains(out, "pass: 6/6") {
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
	if result.PassCount != 6 {
		t.Fatalf("expected pass_count 6, got %d", result.PassCount)
	}
	if result.Total != 6 {
		t.Fatalf("expected total 6, got %d", result.Total)
	}
	expected := map[string]bool{"opencode": false, "claude-code": false, "cursor": false, "generic": false, "antigravity": false, "codex": false}
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

func TestAdaptersDoctorText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "doctor"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# Adapter Doctor") {
		t.Fatalf("expected doctor header, got: %s", out)
	}
	if !strings.Contains(out, "pass:") {
		t.Fatalf("expected pass count, got: %s", out)
	}
}

func TestAdaptersDoctorJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "doctor", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Adapters   []adapterDoctorResult `json:"adapters"`
		PassCount  int                   `json:"pass_count"`
		TotalFiles int                   `json:"total_files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.PassCount != result.TotalFiles {
		t.Fatalf("expected all files to pass, got pass_count=%d total_files=%d", result.PassCount, result.TotalFiles)
	}
	if result.TotalFiles == 0 {
		t.Fatalf("expected at least one adapter file checked, got 0")
	}
	for _, r := range result.Adapters {
		if !r.OK {
			t.Fatalf("expected file %s to pass", r.Path)
		}
		for _, c := range r.Checks {
			if c.Status != "passed" {
				t.Fatalf("expected check %s for %s to pass, got %s", c.Name, r.Path, c.Status)
			}
		}
	}
}

func TestAdaptersDoctorDetectsDrift(t *testing.T) {
	tmpDir := t.TempDir()
	adaptersDir := filepath.Join(tmpDir, "adapters", "test")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatalf("failed to create temp adapters dir: %v", err)
	}

	content := `<!-- BEGIN X-HARNESS MANAGED CONTRACT: test-contract -->
<!-- generated-by: x-harness -->
<!-- contract-hash: 0000000000000000 -->

## Generated Adapter Contract

- Completion is admitted, not claimed.

<!-- END X-HARNESS MANAGED CONTRACT: test-contract -->
`
	readmePath := filepath.Join(adaptersDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp readme: %v", err)
	}

	results, ok := runAdaptersDoctor(tmpDir)
	if ok {
		t.Fatalf("expected drift to be detected")
	}
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}
	found := false
	for _, r := range results {
		if r.Path == filepath.Join("adapters", "test", "README.md") {
			found = true
			if r.OK {
				t.Fatalf("expected result to be not ok")
			}
			if len(r.Checks) == 0 {
				t.Fatalf("expected at least one check")
			}
			if !strings.Contains(r.Checks[0].Note, "hash mismatch") {
				t.Fatalf("expected hash mismatch note, got: %s", r.Checks[0].Note)
			}
		}
	}
	if !found {
		t.Fatalf("expected result for test README.md")
	}
}

func TestAdaptersDoctorMissingHash(t *testing.T) {
	tmpDir := t.TempDir()
	adaptersDir := filepath.Join(tmpDir, "adapters", "test")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatalf("failed to create temp adapters dir: %v", err)
	}

	content := `<!-- BEGIN X-HARNESS MANAGED CONTRACT: test-contract -->
<!-- generated-by: x-harness -->

## Generated Adapter Contract

- Completion is admitted, not claimed.

<!-- END X-HARNESS MANAGED CONTRACT: test-contract -->
`
	readmePath := filepath.Join(adaptersDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp readme: %v", err)
	}

	results, ok := runAdaptersDoctor(tmpDir)
	if ok {
		t.Fatalf("expected missing hash to fail")
	}
	found := false
	for _, r := range results {
		if r.Path == filepath.Join("adapters", "test", "README.md") {
			found = true
			if !strings.Contains(r.Checks[0].Note, "missing contract-hash") {
				t.Fatalf("expected missing contract-hash note, got: %s", r.Checks[0].Note)
			}
		}
	}
	if !found {
		t.Fatalf("expected result for test README.md")
	}
}

func TestAdaptersEvalWithRoot(t *testing.T) {
	tmpDir := t.TempDir()
	adaptersDir := filepath.Join(tmpDir, "adapters", "generic")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatalf("failed to create temp adapters dir: %v", err)
	}
	readmePath := filepath.Join(adaptersDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Generic Adapter\n"), 0644); err != nil {
		t.Fatalf("failed to write temp readme: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "eval", "--root", tmpDir, "generic", "--json"}, &stdout, &stderr)
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
	if result.PassCount != 1 {
		t.Fatalf("expected pass_count 1, got %d", result.PassCount)
	}
	if len(result.Adapters) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(result.Adapters))
	}
	if result.Adapters[0].Name != "generic" {
		t.Fatalf("expected generic adapter, got %s", result.Adapters[0].Name)
	}
}

func TestAdaptersDoctorWithRoot(t *testing.T) {
	tmpDir := t.TempDir()
	adaptersDir := filepath.Join(tmpDir, "adapters", "generic")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatalf("failed to create temp adapters dir: %v", err)
	}
	readmePath := filepath.Join(adaptersDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Generic Adapter\n"), 0644); err != nil {
		t.Fatalf("failed to write temp readme: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"adapters", "doctor", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		Adapters   []adapterDoctorResult `json:"adapters"`
		PassCount  int                   `json:"pass_count"`
		TotalFiles int                   `json:"total_files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	// No managed blocks in temp readme, so 0 files is valid; command should succeed
	if result.PassCount != result.TotalFiles {
		t.Fatalf("expected pass_count == total_files, got pass_count=%d total_files=%d", result.PassCount, result.TotalFiles)
	}
}
