package doctor

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDocsDriftEmptyRootFails(t *testing.T) {
	report := CheckDocsDrift("")
	if report.Healthy {
		t.Fatal("expected unhealthy report for empty root")
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected at least one check")
	}
}

func TestCheckDocsDriftMissingDirFails(t *testing.T) {
	report := CheckDocsDrift("/nonexistent/path/abc/xyz")
	if report.Healthy {
		t.Fatal("expected unhealthy report for missing root")
	}
}

func TestCheckDocsDriftHealthyWhenWorkflowPresent(t *testing.T) {
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\njobs:\n  q:\n    steps:\n      - run: x-harness verify --card x.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// package.json with a verify script
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"scripts":{"verify":"tsc && vitest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := CheckDocsDrift(tmpDir)
	if !report.Healthy {
		t.Fatalf("expected healthy, got %+v", report)
	}
}

func TestCheckDocsDriftFailsWhenWorkflowMissingVerify(t *testing.T) {
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\njobs:\n  q:\n    steps:\n      - run: echo no-verify\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"scripts":{"build":"tsc"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := CheckDocsDrift(tmpDir)
	if report.Healthy {
		t.Fatalf("expected unhealthy, got %+v", report)
	}
	found := false
	for _, tag := range report.DriftTags {
		if tag == "workflow_missing_verify" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected drift_tag=workflow_missing_verify, got %+v", report.DriftTags)
	}
}

func TestCheckDocsDriftFailsOnMixedPackageManager(t *testing.T) {
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\njobs:\n  q:\n    steps:\n      - run: x-harness verify --card x.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"scripts":{"verify":"npm test && pnpm build"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := CheckDocsDrift(tmpDir)
	if report.Healthy {
		t.Fatalf("expected unhealthy due to mixed package managers, got %+v", report)
	}
}

func TestCheckDocsDriftFailsOnStalePublicVersionReference(t *testing.T) {
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\njobs:\n  q:\n    steps:\n      - run: x-harness verify --card x.yaml\n      - run: x-harness policy matrix --json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "packages", "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"version":"1.0.0","scripts":{"verify":"tsc && vitest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "packages", "cli", "package.json"), []byte(`{"version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("Version: 0.99.0-rc7\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := CheckDocsDrift(tmpDir)
	if report.Healthy {
		t.Fatalf("expected unhealthy due to stale version, got %+v", report)
	}
	found := false
	for _, tag := range report.DriftTags {
		if tag == "version_drift:README.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected version_drift:README.md, got %+v", report.DriftTags)
	}
}

func TestCheckDocsDriftPassesOnCurrentPublicVersionReference(t *testing.T) {
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\njobs:\n  q:\n    steps:\n      - run: x-harness verify --card x.yaml\n      - run: x-harness policy matrix --json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "packages", "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"version":"1.0.0","scripts":{"verify":"tsc && vitest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "packages", "cli", "package.json"), []byte(`{"version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("Version: 1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := CheckDocsDrift(tmpDir)
	if !report.Healthy {
		t.Fatalf("expected healthy, got %+v", report)
	}
}

func TestFormatDocsDriftTextRenders(t *testing.T) {
	report := &DocsDriftReport{Healthy: true, Root: "/tmp/x"}
	report.Checks = append(report.Checks, Check{Name: "demo", Status: "passed", Note: "ok"})
	var buf bytes.Buffer
	FormatDocsDriftText(report, &buf)
	out := buf.String()
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}
}

func TestCheckDocsDriftJSONShape(t *testing.T) {
	tmpDir := t.TempDir()
	workflow := filepath.Join(tmpDir, ".github", "workflows", "x-harness-verify.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("name: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := CheckDocsDrift(tmpDir)

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var round DocsDriftReport
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("expected round-trip: %v", err)
	}
	if round.Root != tmpDir {
		t.Fatalf("expected root=%s, got %s", tmpDir, round.Root)
	}
}
