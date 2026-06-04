package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupIntakeHandoffPolicy(t *testing.T, root string) {
	t.Helper()
	policyPath := filepath.Join(root, "policies")
	if err := os.MkdirAll(policyPath, 0755); err != nil {
		t.Fatal(err)
	}
	policyContent := `version: 1
intake_labels:
  tiny:
    runtime_tier: light
    signals:
      - comment_only
  normal:
    runtime_tier: standard
    signals:
      - routine_implementation
  high_risk:
    runtime_tier: deep
    signals:
      - auth
high_risk_signals:
  auth:
    description: Auth changes
    examples:
      - login
runtime_tier_confirmation:
  tiers: [light, standard, deep]
  note: Tiers remain light, standard, deep.
`
	if err := os.WriteFile(filepath.Join(policyPath, "intake.yaml"), []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestIntakeHandoffMissingTier covers the safe V1 rule that --tier is
// required. The CLI must return a usage error.
func TestIntakeHandoffMissingTier(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"intake", "handoff", "--task", "fix bug"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--tier is required") {
		t.Fatalf("expected --tier required error, got: %s", stderr.String())
	}
}

// TestIntakeHandoffExplicitTierRejected covers the safe V1 rule that
// safe V1 only supports --tier auto. Explicit light/standard/deep
// must route to `xh handoff <tier>` instead of `xh intake handoff`.
func TestIntakeHandoffExplicitTierRejected(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakeHandoffPolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "handoff",
		"--tier", "standard",
		"--root", tmpDir,
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "safe V1") {
		t.Fatalf("expected safe V1 message, got: %s", stderr.String())
	}
}

// TestIntakeHandoffAutoNormal covers the auto-classification path for a
// routine task. The selected tier should be standard and the output
// should include a command suggestion that uses the explicit tier.
func TestIntakeHandoffAutoNormal(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakeHandoffPolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "handoff",
		"--tier", "auto",
		"--task", "fix bug in formatter",
		"--file", "src/formatter.go",
		"--root", tmpDir,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Selected tier: standard") {
		t.Fatalf("expected standard tier, got: %s", out)
	}
	if !strings.Contains(out, "Intake label: normal") {
		t.Fatalf("expected normal label, got: %s", out)
	}
	if !strings.Contains(out, "Suggested next: xh handoff standard") {
		t.Fatalf("expected command suggestion, got: %s", out)
	}
}

// TestIntakeHandoffAutoHighRisk covers the auto-classification path for
// a high-risk task. The selected tier should be deep and the output
// should flag auto escalation.
func TestIntakeHandoffAutoHighRisk(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakeHandoffPolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "handoff",
		"--tier", "auto",
		"--task", "update auth logic",
		"--root", tmpDir,
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Selected tier: deep") {
		t.Fatalf("expected deep tier, got: %s", out)
	}
	if !strings.Contains(out, "Intake label: high_risk") {
		t.Fatalf("expected high_risk label, got: %s", out)
	}
	if !strings.Contains(out, "Auto escalated: yes") {
		t.Fatalf("expected auto-escalated message, got: %s", out)
	}
}

// TestIntakeHandoffAutoJSON covers the JSON output path: the result must
// round-trip through encoding/json and include the selected tier, the
// intake label, and the command suggestion.
func TestIntakeHandoffAutoJSON(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakeHandoffPolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "handoff",
		"--tier", "auto",
		"--task", "fix bug in formatter",
		"--root", tmpDir,
		"--json",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
	if doc["selected_tier"] != "standard" {
		t.Fatalf("expected selected_tier=standard, got %v", doc["selected_tier"])
	}
	if doc["intake_label"] != "normal" {
		t.Fatalf("expected intake_label=normal, got %v", doc["intake_label"])
	}
	cmd, _ := doc["command_suggestion"].(string)
	if !strings.Contains(cmd, "xh handoff standard") {
		t.Fatalf("expected command suggestion to include xh handoff standard, got %q", cmd)
	}
}

// TestIntakeHandoffAutoMissingPolicy covers the case where the
// repository has no intake policy. The CLI must surface a clear usage
// error rather than crashing.
func TestIntakeHandoffAutoMissingPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "handoff",
		"--tier", "auto",
		"--root", tmpDir,
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "intake.yaml not found") {
		t.Fatalf("expected policy not found error, got: %s", stderr.String())
	}
}

// TestIntakeHandoffUnknownFlag covers the safe V1 rule that unknown
// flags produce a usage error. The --from flag is intentionally
// rejected in safe V1.
func TestIntakeHandoffUnknownFlag(t *testing.T) {
	tmpDir := t.TempDir()
	setupIntakeHandoffPolicy(t, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"intake", "handoff",
		"--tier", "auto",
		"--from", "issue.md",
		"--root", tmpDir,
	}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitUsage, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
