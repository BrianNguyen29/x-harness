package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLearnDefaultOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"learn"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Concept tour") {
		t.Fatalf("expected concept tour header, got: %s", out)
	}
	if !strings.Contains(out, "Overview") {
		t.Fatalf("expected Overview section, got: %s", out)
	}
	if !strings.Contains(out, "Core concepts") {
		t.Fatalf("expected Core concepts section, got: %s", out)
	}
	if !strings.Contains(out, "Tiers and evidence") {
		t.Fatalf("expected Tiers and evidence section, got: %s", out)
	}
	if !strings.Contains(out, "Next steps") {
		t.Fatalf("expected Next steps, got: %s", out)
	}
}

func TestLearnJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"learn", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result LearnResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Sections) == 0 {
		t.Fatalf("expected sections, got: %+v", result)
	}
	if len(result.NextSteps) == 0 {
		t.Fatalf("expected next_steps, got: %+v", result)
	}
}

func TestLearnHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"learn", "--help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %q", stderr.String())
	}
}

func TestLearnInHelpListing(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "learn") {
		t.Fatalf("expected help to include learn, got: %s", stdout.String())
	}
}

func TestLearnMaturityBeta(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--help-maturity"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	out := stdout.String()
	if !strings.Contains(out, "learn") {
		t.Fatalf("expected --help-maturity to include learn, got: %s", out)
	}
	betaIdx := strings.Index(out, "beta:")
	expIdx := strings.Index(out, "experimental:")
	learnIdx := strings.Index(out, "learn")
	if betaIdx == -1 || expIdx == -1 || learnIdx == -1 {
		t.Fatalf("missing expected sections")
	}
	if learnIdx < betaIdx || learnIdx > expIdx {
		t.Fatalf("expected learn to appear under beta section")
	}
}

func TestLearnVietnamese(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"learn", "--lang", "vi"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Khái niệm cơ bản") {
		t.Fatalf("expected Vietnamese 'Khái niệm cơ bản', got: %s", out)
	}
	if !strings.Contains(out, "Tổng quan") {
		t.Fatalf("expected Vietnamese 'Tổng quan', got: %s", out)
	}
	if !strings.Contains(out, "Khái niệm cốt lõi") {
		t.Fatalf("expected Vietnamese 'Khái niệm cốt lõi', got: %s", out)
	}
	if !strings.Contains(out, "Cấp độ và bằng chứng") {
		t.Fatalf("expected Vietnamese 'Cấp độ và bằng chứng', got: %s", out)
	}
	if !strings.Contains(out, "Bước tiếp theo:") {
		t.Fatalf("expected Vietnamese 'Bước tiếp theo:', got: %s", out)
	}
}

func TestLearnVietnameseOrderIndependent(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"--lang", "vi", "learn"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Tổng quan") {
		t.Fatalf("expected Vietnamese 'Tổng quan', got: %s", out)
	}
}

func TestLearnJSONRemainsEnglish(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"learn", "--lang", "vi", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result LearnResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if len(result.Sections) == 0 {
		t.Fatalf("expected sections, got: %+v", result)
	}
	if result.Sections[0].Title != "Overview" {
		t.Fatalf("expected English JSON title, got: %s", result.Sections[0].Title)
	}
}
