package prediction

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidatePredictionValid(t *testing.T) {
	pred := &Prediction{
		Claim:               "This fix will resolve the issue",
		ExpectedEffect:      "Tests pass",
		FalsificationMethod: "Run test suite",
		Horizon:             "same_verify",
		MeasurableSignal:    "exit code 0",
		Confidence:          "high",
	}
	result := ValidatePrediction(pred)
	if !result.OK {
		t.Fatalf("expected valid prediction, got errors: %v", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got: %v", result.Warnings)
	}
}

func TestValidatePredictionMissingRequired(t *testing.T) {
	pred := &Prediction{
		Claim:               "",
		ExpectedEffect:      "",
		FalsificationMethod: "",
		Horizon:             "",
	}
	result := ValidatePrediction(pred)
	if result.OK {
		t.Fatal("expected invalid prediction")
	}
	expectedErrors := 4
	if len(result.Errors) != expectedErrors {
		t.Fatalf("expected %d errors, got %d: %v", expectedErrors, len(result.Errors), result.Errors)
	}
}

func TestValidatePredictionInvalidHorizon(t *testing.T) {
	pred := &Prediction{
		Claim:               "Fix",
		ExpectedEffect:      "Effect",
		FalsificationMethod: "Method",
		Horizon:             "invalid_horizon",
	}
	result := ValidatePrediction(pred)
	if result.OK {
		t.Fatal("expected invalid prediction")
	}
	found := false
	for _, e := range result.Errors {
		if e == "prediction.horizon must be one of: same_verify, next_ci_run, next_release, manual_review, production_7d, production_30d" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected horizon enum error, got: %v", result.Errors)
	}
}

func TestValidatePredictionMissingMeasurableSignal(t *testing.T) {
	pred := &Prediction{
		Claim:               "Fix",
		ExpectedEffect:      "Effect",
		FalsificationMethod: "Method",
		Horizon:             "same_verify",
	}
	result := ValidatePrediction(pred)
	if !result.OK {
		t.Fatalf("expected valid prediction, got errors: %v", result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if w == "prediction.measurable_signal is recommended for falsifiable predictions" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected measurable_signal warning, got: %v", result.Warnings)
	}
}

func TestValidatePredictionInvalidConfidence(t *testing.T) {
	pred := &Prediction{
		Claim:               "Fix",
		ExpectedEffect:      "Effect",
		FalsificationMethod: "Method",
		Horizon:             "same_verify",
		Confidence:          "very_high",
	}
	result := ValidatePrediction(pred)
	if !result.OK {
		t.Fatalf("expected valid prediction (confidence is warning only), got errors: %v", result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if w == "prediction.confidence should be one of: low, medium, high" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected confidence warning, got: %v", result.Warnings)
	}
}

func TestVerifyPredictionFromEpisodeMissingPrediction(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"verdict": {
			"admission_outcome": "success",
			"acceptance_status": "accepted"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyPredictionFromEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "inconclusive" {
		t.Fatalf("expected inconclusive, got: %s", result.Status)
	}
	if result.Reason != "missing_prediction" {
		t.Fatalf("expected missing_prediction reason, got: %s", result.Reason)
	}
}

func TestVerifyPredictionFromEpisodeConfirmed(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"verdict": {
			"admission_outcome": "success",
			"acceptance_status": "accepted"
		}
	}`
	card := `
prediction:
  claim: "Fix will work"
  expected_effect: "Tests pass"
  falsification_method: "Run tests"
  horizon: "same_verify"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(card), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyPredictionFromEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "confirmed" {
		t.Fatalf("expected confirmed, got: %s", result.Status)
	}
	if result.Reason != "same_verify_episode_accepted" {
		t.Fatalf("expected same_verify_episode_accepted reason, got: %s", result.Reason)
	}
	if !result.OK {
		t.Fatal("expected OK=true")
	}
}

func TestVerifyPredictionFromEpisodeFalsified(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"verdict": {
			"admission_outcome": "failed",
			"acceptance_status": "withheld"
		}
	}`
	card := `
prediction:
  claim: "Fix will work"
  expected_effect: "Tests pass"
  falsification_method: "Run tests"
  horizon: "same_verify"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(card), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyPredictionFromEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "falsified" {
		t.Fatalf("expected falsified, got: %s", result.Status)
	}
	if result.OK {
		t.Fatal("expected OK=false")
	}
}

func TestVerifyPredictionFromEpisodeUnsupportedHorizon(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"verdict": {
			"admission_outcome": "success",
			"acceptance_status": "accepted"
		}
	}`
	card := `
prediction:
  claim: "Fix will work"
  expected_effect: "Tests pass"
  falsification_method: "Run tests"
  horizon: "next_release"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(card), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyPredictionFromEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "inconclusive" {
		t.Fatalf("expected inconclusive, got: %s", result.Status)
	}
	if result.Reason != "unsupported_horizon:next_release" {
		t.Fatalf("expected unsupported_horizon reason, got: %s", result.Reason)
	}
}

func TestVerifyPredictionFromEpisodeInvalidPrediction(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `{
		"episode_id": "ep_001",
		"task_id": "task_001",
		"verdict": {
			"admission_outcome": "success",
			"acceptance_status": "accepted"
		}
	}`
	card := `
prediction:
  claim: ""
  expected_effect: ""
  falsification_method: ""
  horizon: ""
`
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "completion-card.yaml"), []byte(card), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyPredictionFromEpisode(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "inconclusive" {
		t.Fatalf("expected inconclusive, got: %s", result.Status)
	}
	if result.Reason != "invalid_prediction" {
		t.Fatalf("expected invalid_prediction reason, got: %s", result.Reason)
	}
}

func TestGenerateReportEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	report, err := GenerateReport(tmpDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.EpisodesAnalyzed != 0 {
		t.Fatalf("expected 0 episodes, got: %d", report.EpisodesAnalyzed)
	}
}

func TestGenerateReportWithEpisodes(t *testing.T) {
	tmpDir := t.TempDir()
	episodesDir := filepath.Join(tmpDir, ".x-harness", "episodes")
	if err := os.MkdirAll(episodesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create episode 1: confirmed
	ep1 := filepath.Join(episodesDir, "ep_2024-01-01T00-00-00Z_task1")
	if err := os.MkdirAll(ep1, 0755); err != nil {
		t.Fatal(err)
	}
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

	// Create episode 2: falsified
	ep2 := filepath.Join(episodesDir, "ep_2024-01-01T00-00-00Z_task2")
	if err := os.MkdirAll(ep2, 0755); err != nil {
		t.Fatal(err)
	}
	manifest2 := `{
		"episode_id": "ep_002",
		"task_id": "task_002",
		"created_at": "` + time.Now().UTC().Format(time.RFC3339) + `",
		"verdict": {
			"admission_outcome": "failed",
			"acceptance_status": "withheld"
		}
	}`
	card2 := `
prediction:
  claim: "Fix"
  expected_effect: "Effect"
  falsification_method: "Method"
  horizon: "same_verify"
`
	os.WriteFile(filepath.Join(ep2, "manifest.json"), []byte(manifest2), 0644)
	os.WriteFile(filepath.Join(ep2, "completion-card.yaml"), []byte(card2), 0644)

	report, err := GenerateReport(episodesDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.EpisodesAnalyzed != 2 {
		t.Fatalf("expected 2 episodes, got: %d", report.EpisodesAnalyzed)
	}
	if report.Confirmed != 1 {
		t.Fatalf("expected 1 confirmed, got: %d", report.Confirmed)
	}
	if report.Falsified != 1 {
		t.Fatalf("expected 1 falsified, got: %d", report.Falsified)
	}
}

func TestGenerateReportSinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	episodesDir := filepath.Join(tmpDir, ".x-harness", "episodes")
	if err := os.MkdirAll(episodesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Old episode
	ep1 := filepath.Join(episodesDir, "ep_2024-01-01T00-00-00Z_task1")
	if err := os.MkdirAll(ep1, 0755); err != nil {
		t.Fatal(err)
	}
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

	// Recent episode
	ep2 := filepath.Join(episodesDir, "ep_2024-01-01T00-00-00Z_task2")
	if err := os.MkdirAll(ep2, 0755); err != nil {
		t.Fatal(err)
	}
	manifest2 := `{
		"episode_id": "ep_002",
		"task_id": "task_002",
		"created_at": "` + time.Now().UTC().Format(time.RFC3339) + `",
		"verdict": {
			"admission_outcome": "failed",
			"acceptance_status": "withheld"
		}
	}`
	card2 := `
prediction:
  claim: "Fix"
  expected_effect: "Effect"
  falsification_method: "Method"
  horizon: "same_verify"
`
	os.WriteFile(filepath.Join(ep2, "manifest.json"), []byte(manifest2), 0644)
	os.WriteFile(filepath.Join(ep2, "completion-card.yaml"), []byte(card2), 0644)

	report, err := GenerateReport(episodesDir, "7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.EpisodesAnalyzed != 1 {
		t.Fatalf("expected 1 episode, got: %d", report.EpisodesAnalyzed)
	}
	if report.Falsified != 1 {
		t.Fatalf("expected 1 falsified, got: %d", report.Falsified)
	}
}
