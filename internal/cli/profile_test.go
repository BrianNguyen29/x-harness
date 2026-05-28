package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProfileRecommendStandardPR(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--goal", "AI PR verification"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Recommended profile: standard") {
		t.Fatalf("expected standard recommendation, got: %s", out)
	}
	if !strings.Contains(out, "Reason:") {
		t.Fatalf("expected reason, got: %s", out)
	}
	if !strings.Contains(out, "Required commands:") {
		t.Fatalf("expected required commands, got: %s", out)
	}
	if !strings.Contains(out, "Recommended checks:") {
		t.Fatalf("expected recommended checks, got: %s", out)
	}
	if !strings.Contains(out, "Not needed:") {
		t.Fatalf("expected not needed list, got: %s", out)
	}
}

func TestProfileRecommendDeepRelease(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--goal", "release readiness"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Recommended profile: deep") {
		t.Fatalf("expected deep recommendation, got: %s", out)
	}
	if !strings.Contains(out, "release readiness") {
		t.Fatalf("expected goal in output, got: %s", out)
	}
}

func TestProfileRecommendJSON(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--goal", "AI PR verification", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var rec profileRecommendation
	if err := json.Unmarshal([]byte(stdout.String()), &rec); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if rec.RecommendedProfile != "standard" {
		t.Fatalf("expected standard, got %s", rec.RecommendedProfile)
	}
	if rec.Reason == "" {
		t.Fatal("expected non-empty reason")
	}
	if len(rec.RequiredCommands) == 0 {
		t.Fatal("expected required commands")
	}
	if len(rec.RecommendedChecks) == 0 {
		t.Fatal("expected recommended checks")
	}
}

func TestProfileRecommendDeepJSON(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--goal", "release readiness", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var rec profileRecommendation
	if err := json.Unmarshal([]byte(stdout.String()), &rec); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if rec.RecommendedProfile != "deep" {
		t.Fatalf("expected deep, got %s", rec.RecommendedProfile)
	}
	if len(rec.NotNeeded) != 0 {
		t.Fatalf("expected empty not_needed for deep, got %v", rec.NotNeeded)
	}
}

func TestProfileRecommendMissingGoal(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage error, got: %q", stderr.String())
	}
}

func TestProfileRecommendMinimalLocal(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--goal", "local quick task"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Recommended profile: minimal") {
		t.Fatalf("expected minimal recommendation, got: %s", out)
	}
}

func TestProfileRecommendUnknownGoalDefaultsStandard(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--goal", "something completely unrelated"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Recommended profile: standard") {
		t.Fatalf("expected standard default, got: %s", out)
	}
}

func TestProfileRecommendSecurityDeep(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--goal", "deep security-sensitive change"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Recommended profile: deep") {
		t.Fatalf("expected deep recommendation, got: %s", out)
	}
}

func TestProfileRecommendUnknownSubcommand(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown profile subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %q", stderr.String())
	}
}

func TestProfileRecommendUnknownFlag(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile", "recommend", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestProfileRecommendNoSubcommand(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	code := Run([]string{"profile"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %q", stderr.String())
	}
}
