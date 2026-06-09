package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuickDefaultOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Next-action recommender") {
		t.Fatalf("expected quick header, got: %s", out)
	}
	if !strings.Contains(out, "root:") {
		t.Fatalf("expected root line, got: %s", out)
	}
	if !strings.Contains(out, "recommendation:") {
		t.Fatalf("expected recommendation line, got: %s", out)
	}
	if !strings.Contains(out, "reason:") {
		t.Fatalf("expected reason line, got: %s", out)
	}
	if !strings.Contains(out, "Next steps:") {
		t.Fatalf("expected next steps, got: %s", out)
	}
	// Should always include these read-only commands
	if !strings.Contains(out, "xh run builtin:ci --dry-run") {
		t.Fatalf("expected dry-run ci suggestion, got: %s", out)
	}
	if !strings.Contains(out, "xh learn") {
		t.Fatalf("expected learn suggestion, got: %s", out)
	}
}

func TestQuickJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result QuickResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Root == "" {
		t.Fatalf("expected non-empty root, got: %+v", result)
	}
	if result.Recommendation == "" {
		t.Fatalf("expected non-empty recommendation, got: %+v", result)
	}
	if result.Reason == "" {
		t.Fatalf("expected non-empty reason, got: %+v", result)
	}
	if len(result.NextSteps) == 0 {
		t.Fatalf("expected next_steps, got: %+v", result)
	}
	foundDryRun := false
	foundLearn := false
	for _, s := range result.NextSteps {
		if strings.Contains(s, "builtin:ci --dry-run") {
			foundDryRun = true
		}
		if s == "xh learn" {
			foundLearn = true
		}
	}
	if !foundDryRun {
		t.Fatalf("expected dry-run ci in next_steps, got: %+v", result)
	}
	if !foundLearn {
		t.Fatalf("expected learn in next_steps, got: %+v", result)
	}
}

func TestQuickHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %q", stderr.String())
	}
}

func TestQuickInHelpListing(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "quick") {
		t.Fatalf("expected help to include quick, got: %s", stdout.String())
	}
}

func TestQuickMaturityBeta(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help-maturity"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "quick") {
		t.Fatalf("expected --help-maturity to include quick, got: %s", out)
	}
	betaIdx := strings.Index(out, "beta:")
	expIdx := strings.Index(out, "experimental:")
	quickIdx := strings.Index(out, "quick")
	if betaIdx == -1 || expIdx == -1 || quickIdx == -1 {
		t.Fatalf("missing expected sections")
	}
	if quickIdx < betaIdx || quickIdx > expIdx {
		t.Fatalf("expected quick to appear under beta section")
	}
}

func TestQuickNoHarnessRecommendsStart(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result QuickResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !strings.Contains(result.Recommendation, "xh start") {
		t.Fatalf("expected start recommendation for empty dir, got: %s", result.Recommendation)
	}
	foundStart := false
	for _, s := range result.NextSteps {
		if s == "xh start" || s == "xh init" {
			foundStart = true
			break
		}
	}
	if !foundStart {
		t.Fatalf("expected start/init in next_steps for empty dir, got: %+v", result)
	}
}

func TestQuickWithCardRecommendsCheck(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a harness marker and a completion card
	_ = os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# AGENTS\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte("task_id: test\n"), 0644)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result QuickResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !strings.Contains(result.Recommendation, "xh check --card") {
		t.Fatalf("expected check recommendation when card exists, got: %s", result.Recommendation)
	}
	foundCheck := false
	for _, s := range result.NextSteps {
		if strings.Contains(s, "xh check --card") {
			foundCheck = true
			break
		}
	}
	if !foundCheck {
		t.Fatalf("expected check in next_steps when card exists, got: %+v", result)
	}
}

func TestQuickWithHarnessNoCardRecommendsDoctor(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# AGENTS\n"), 0644)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick", "--root", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result QuickResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !strings.Contains(result.Recommendation, "xh doctor") {
		t.Fatalf("expected doctor recommendation when harness exists without card, got: %s", result.Recommendation)
	}
}

func TestQuickVietnamese(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick", "--lang", "vi"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Gợi ý hành động tiếp theo") {
		t.Fatalf("expected Vietnamese 'Gợi ý hành động tiếp theo', got: %s", out)
	}
	if !strings.Contains(out, "thư mục gốc:") {
		t.Fatalf("expected Vietnamese 'thư mục gốc:', got: %s", out)
	}
	if !strings.Contains(out, "gợi ý:") {
		t.Fatalf("expected Vietnamese 'gợi ý:', got: %s", out)
	}
	if !strings.Contains(out, "lý do:") {
		t.Fatalf("expected Vietnamese 'lý do:', got: %s", out)
	}
	if !strings.Contains(out, "Tín hiệu phát hiện:") {
		t.Fatalf("expected Vietnamese 'Tín hiệu phát hiện:', got: %s", out)
	}
	if !strings.Contains(out, "Bước tiếp theo:") {
		t.Fatalf("expected Vietnamese 'Bước tiếp theo:', got: %s", out)
	}
}

func TestQuickVietnameseOrderIndependent(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--lang", "vi", "quick"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Gợi ý hành động tiếp theo") {
		t.Fatalf("expected Vietnamese title, got: %s", out)
	}
}

func TestQuickJSONRemainsEnglish(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"quick", "--lang", "vi", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result QuickResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.Recommendation == "" {
		t.Fatalf("expected non-empty recommendation, got: %+v", result)
	}
	// JSON values should remain English
	if strings.Contains(result.Recommendation, "thư mục gốc") {
		t.Fatalf("expected English JSON recommendation, got: %s", result.Recommendation)
	}
}
