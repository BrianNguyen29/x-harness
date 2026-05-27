package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPredictionMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires a subcommand") {
		t.Fatalf("expected subcommand required error, got: %s", stderr.String())
	}
}

func TestPredictionUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown prediction subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func setupPredictionCheckDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	cardContent := `schema_version: "1"
task_id: task_001
tier: standard
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: "Fix"
  evidence: []
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: done
  owner: test
prediction:
  claim: "Fix will work"
  expected_effect: "Tests pass"
  falsification_method: "Run tests"
  horizon: "same_verify"
  measurable_signal: "exit code 0"
  confidence: "high"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(cardContent), 0644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestPredictionCheckValid(t *testing.T) {
	tmpDir := setupPredictionCheckDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check", "--card", filepath.Join(tmpDir, "completion-card.yaml")}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Prediction is valid.") {
		t.Fatalf("expected valid prediction message, got: %s", stdout.String())
	}
}

func TestPredictionCheckValidVerbose(t *testing.T) {
	tmpDir := setupPredictionCheckDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check", "--card", filepath.Join(tmpDir, "completion-card.yaml"), "--verbose"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "✓ Prediction is valid") {
		t.Fatalf("expected verbose valid message, got: %s", stdout.String())
	}
}

func TestPredictionCheckValidJSON(t *testing.T) {
	tmpDir := setupPredictionCheckDir(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check", "--card", filepath.Join(tmpDir, "completion-card.yaml"), "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
}

func TestPredictionCheckMissingCard(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "No completion card found") {
		t.Fatalf("expected missing card error, got: %s", stderr.String())
	}
}

func TestPredictionCheckMissingPrediction(t *testing.T) {
	tmpDir := t.TempDir()
	cardContent := `schema_version: "1"
task_id: task_001
tier: standard
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: "Fix"
  evidence: []
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: done
  owner: test
`
	os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(cardContent), 0644)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check", "--card", filepath.Join(tmpDir, "completion-card.yaml")}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "No prediction found") {
		t.Fatalf("expected missing prediction error, got: %s", stderr.String())
	}
}

func TestPredictionCheckInvalidPrediction(t *testing.T) {
	tmpDir := t.TempDir()
	cardContent := `schema_version: "1"
task_id: task_001
tier: standard
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: "Fix"
  evidence: []
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: done
  owner: test
prediction:
  claim: ""
  expected_effect: ""
  falsification_method: ""
  horizon: ""
`
	os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(cardContent), 0644)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check", "--card", filepath.Join(tmpDir, "completion-card.yaml")}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "Prediction validation failed") {
		t.Fatalf("expected validation failed message, got: %s", stderr.String())
	}
}

func setupPredictionVerifyDir(t *testing.T, admissionOutcome, acceptanceStatus string) string {
	t.Helper()
	tmpDir := t.TempDir()
	manifest := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"created_at": "` + time.Now().UTC().Format(time.RFC3339) + `",
		"verdict": {
			"admission_outcome": "` + admissionOutcome + `",
			"acceptance_status": "` + acceptanceStatus + `"
		}
	}`
	card := `
prediction:
  claim: "Fix will work"
  expected_effect: "Tests pass"
  falsification_method: "Run tests"
  horizon: "same_verify"
`
	os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(card), 0644)
	return tmpDir
}

func TestPredictionVerifyConfirmed(t *testing.T) {
	tmpDir := setupPredictionVerifyDir(t, "success", "accepted")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "verify", "--episode", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "confirmed") {
		t.Fatalf("expected confirmed status, got: %s", stdout.String())
	}
}

func TestPredictionVerifyFalsified(t *testing.T) {
	tmpDir := setupPredictionVerifyDir(t, "failed", "withheld")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "verify", "--episode", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "falsified") {
		t.Fatalf("expected falsified status, got: %s", stdout.String())
	}
}

func TestPredictionVerifyJSON(t *testing.T) {
	tmpDir := setupPredictionVerifyDir(t, "success", "accepted")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "verify", "--episode", tmpDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["status"] != "confirmed" {
		t.Fatalf("expected status=confirmed, got: %v", result)
	}
}

func TestPredictionVerifyMissingEpisode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "verify"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "requires --episode") {
		t.Fatalf("expected missing episode error, got: %s", stderr.String())
	}
}

func TestPredictionReport(t *testing.T) {
	tmpDir := t.TempDir()
	episodesDir := filepath.Join(tmpDir, ".x-harness", "episodes")
	os.MkdirAll(episodesDir, 0755)

	ep1 := filepath.Join(episodesDir, "ep_2024-01-01T00-00-00Z_task1")
	os.MkdirAll(ep1, 0755)
	manifest1 := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"created_at": "` + time.Now().UTC().Format(time.RFC3339) + `",
		"verdict": {
			"admission_outcome": "success",
			"acceptance_status": "accepted"
		}
	}`
	card1 := `
prediction:
  claim: "Fix"
  expected_effect: "Effect"
  falsification_method: "Method"
  horizon: "same_verify"
`
	os.WriteFile(filepath.Join(ep1, "manifest.json"), []byte(manifest1), 0644)
	os.WriteFile(filepath.Join(ep1, "completion-card.yaml"), []byte(card1), 0644)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "report", "--episodes-dir", episodesDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "episodes_analyzed: 1") {
		t.Fatalf("expected episodes_analyzed: 1, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "confirmed: 1") {
		t.Fatalf("expected confirmed: 1, got: %s", stdout.String())
	}
}

func TestPredictionReportJSON(t *testing.T) {
	tmpDir := t.TempDir()
	episodesDir := filepath.Join(tmpDir, ".x-harness", "episodes")
	os.MkdirAll(episodesDir, 0755)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "report", "--episodes-dir", episodesDir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", result)
	}
}

func TestPredictionReportSinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	episodesDir := filepath.Join(tmpDir, ".x-harness", "episodes")
	os.MkdirAll(episodesDir, 0755)

	// Old episode
	ep1 := filepath.Join(episodesDir, "ep_2020-01-01T00-00-00Z_task1")
	os.MkdirAll(ep1, 0755)
	manifest1 := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"created_at": "2020-01-01T00:00:00Z",
		"verdict": {
			"admission_outcome": "success",
			"acceptance_status": "accepted"
		}
	}`
	card1 := `
prediction:
  claim: "Fix"
  expected_effect: "Effect"
  falsification_method: "Method"
  horizon: "same_verify"
`
	os.WriteFile(filepath.Join(ep1, "manifest.json"), []byte(manifest1), 0644)
	os.WriteFile(filepath.Join(ep1, "completion-card.yaml"), []byte(card1), 0644)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "report", "--episodes-dir", episodesDir, "--since", "7d"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "episodes_analyzed: 0") {
		t.Fatalf("expected episodes_analyzed: 0, got: %s", stdout.String())
	}
}

func TestPredictionCheckJSONOutputMissingPrediction(t *testing.T) {
	tmpDir := t.TempDir()
	cardContent := `schema_version: "1"
task_id: task_001
tier: standard
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: "Fix"
  evidence: []
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: done
  owner: test
`
	os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(cardContent), 0644)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check", "--card", filepath.Join(tmpDir, "completion-card.yaml"), "--json"}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, stdout.String())
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false, got: %v", result)
	}
}

func TestPredictionCheckAutoDetect(t *testing.T) {
	tmpDir := t.TempDir()
	cardContent := `schema_version: "1"
task_id: task_001
tier: light
owner: test
accountable: test
claim:
  fix_status: fixed
  summary: "Fix"
  evidence: []
verification:
  status: passed
  checks: []
admission:
  outcome: success
acceptance_status: accepted
handoff:
  next_action: done
  owner: test
prediction:
  claim: "Fix"
  expected_effect: "Effect"
  falsification_method: "Method"
  horizon: "same_verify"
`
	os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(cardContent), 0644)

	// Change to tmpDir for auto-detection
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Prediction is valid.") {
		t.Fatalf("expected valid prediction message, got: %s", stdout.String())
	}
}

func TestPredictionUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"prediction", "check", "--bogus"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", stderr.String())
	}
}
