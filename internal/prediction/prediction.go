package prediction

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Prediction represents a prediction block from completion card.
type Prediction struct {
	Claim               string `yaml:"claim" json:"claim"`
	ExpectedEffect      string `yaml:"expected_effect" json:"expected_effect"`
	FalsificationMethod string `yaml:"falsification_method" json:"falsification_method"`
	Horizon             string `yaml:"horizon" json:"horizon"`
	MeasurableSignal    string `yaml:"measurable_signal,omitempty" json:"measurable_signal,omitempty"`
	Confidence          string `yaml:"confidence,omitempty" json:"confidence,omitempty"`
}

// ValidationResult represents prediction validation outcome.
type ValidationResult struct {
	OK         bool        `json:"ok"`
	Errors     []string    `json:"errors"`
	Warnings   []string    `json:"warnings"`
	Prediction *Prediction `json:"prediction,omitempty"`
	Tier       string      `json:"tier,omitempty"`
}

var allowedHorizons = []string{
	"same_verify",
	"next_ci_run",
	"next_release",
	"manual_review",
	"production_7d",
	"production_30d",
}

func isAllowedHorizon(h string) bool {
	for _, v := range allowedHorizons {
		if v == h {
			return true
		}
	}
	return false
}

// ValidatePrediction validates prediction structure.
func ValidatePrediction(pred *Prediction) *ValidationResult {
	result := &ValidationResult{
		OK:       true,
		Errors:   []string{},
		Warnings: []string{},
		Prediction: pred,
	}

	if pred == nil {
		result.OK = false
		result.Errors = append(result.Errors, "prediction is nil")
		return result
	}

	if strings.TrimSpace(pred.Claim) == "" {
		result.OK = false
		result.Errors = append(result.Errors, "prediction.claim is required and must be a non-empty string")
	}

	if strings.TrimSpace(pred.ExpectedEffect) == "" {
		result.OK = false
		result.Errors = append(result.Errors, "prediction.expected_effect is required and must be a non-empty string")
	}

	if strings.TrimSpace(pred.FalsificationMethod) == "" {
		result.OK = false
		result.Errors = append(result.Errors, "prediction.falsification_method is required and must be a non-empty string")
	}

	if pred.Horizon == "" {
		result.OK = false
		result.Errors = append(result.Errors, "prediction.horizon is required")
	} else if !isAllowedHorizon(pred.Horizon) {
		result.OK = false
		result.Errors = append(result.Errors, fmt.Sprintf("prediction.horizon must be one of: %s", strings.Join(allowedHorizons, ", ")))
	}

	if strings.TrimSpace(pred.MeasurableSignal) == "" {
		result.Warnings = append(result.Warnings, "prediction.measurable_signal is recommended for falsifiable predictions")
	}

	if pred.Confidence != "" && !isAllowedConfidence(pred.Confidence) {
		result.Warnings = append(result.Warnings, "prediction.confidence should be one of: low, medium, high")
	}

	return result
}

func isAllowedConfidence(c string) bool {
	switch c {
	case "low", "medium", "high":
		return true
	}
	return false
}

// Verdict represents the episode verdict.
type Verdict struct {
	AdmissionOutcome string `json:"admission_outcome"`
	AcceptanceStatus string `json:"acceptance_status"`
}

// PredictionVerificationResult represents verification outcome.
type PredictionVerificationResult struct {
	OK         bool              `json:"ok"`
	Status     string            `json:"status"` // confirmed, falsified, inconclusive
	Reason     string            `json:"reason"`
	EpisodeID  string            `json:"episode_id"`
	TaskID     string            `json:"task_id"`
	Horizon    string            `json:"horizon"`
	Prediction *Prediction       `json:"prediction,omitempty"`
	Validation *ValidationResult `json:"validation,omitempty"`
	Verdict    *Verdict          `json:"verdict,omitempty"`
}

// VerifyPredictionFromEpisode verifies prediction from episode outcome.
func VerifyPredictionFromEpisode(episodePath string) (*PredictionVerificationResult, error) {
	resolved, err := filepath.Abs(episodePath)
	if err != nil {
		resolved = episodePath
	}

	// Read manifest.json
	manifestPath := filepath.Join(resolved, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest.json not found: %w", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("manifest.json parse error: %w", err)
	}

	episodeID := stringValue(manifest["episode_id"])
	taskID := stringValue(manifest["task_id"])

	var verdict Verdict
	if v, ok := manifest["verdict"].(map[string]interface{}); ok {
		verdict.AdmissionOutcome = stringValue(v["admission_outcome"])
		verdict.AcceptanceStatus = stringValue(v["acceptance_status"])
	}

	// Read completion card
	card, err := loadEpisodeCard(resolved)
	if err != nil {
		return nil, err
	}

	pred, _ := extractPrediction(card)

	if pred == nil {
		return &PredictionVerificationResult{
			OK:        false,
			Status:    "inconclusive",
			Reason:    "missing_prediction",
			EpisodeID: episodeID,
			TaskID:    taskID,
			Horizon:   "",
			Prediction: nil,
			Validation: nil,
			Verdict:    &verdict,
		}, nil
	}

	validation := ValidatePrediction(pred)
	if !validation.OK {
		return &PredictionVerificationResult{
			OK:        false,
			Status:    "inconclusive",
			Reason:    "invalid_prediction",
			EpisodeID: episodeID,
			TaskID:    taskID,
			Horizon:   pred.Horizon,
			Prediction: pred,
			Validation: validation,
			Verdict:    &verdict,
		}, nil
	}

	if pred.Horizon != "same_verify" {
		return &PredictionVerificationResult{
			OK:        true,
			Status:    "inconclusive",
			Reason:    fmt.Sprintf("unsupported_horizon:%s", pred.Horizon),
			EpisodeID: episodeID,
			TaskID:    taskID,
			Horizon:   pred.Horizon,
			Prediction: pred,
			Validation: validation,
			Verdict:    &verdict,
		}, nil
	}

	confirmed := verdict.AdmissionOutcome == "success" && verdict.AcceptanceStatus == "accepted"
	reason := "same_verify_episode_withheld"
	if confirmed {
		reason = "same_verify_episode_accepted"
	}

	return &PredictionVerificationResult{
		OK:        confirmed,
		Status:    map[bool]string{true: "confirmed", false: "falsified"}[confirmed],
		Reason:    reason,
		EpisodeID: episodeID,
		TaskID:    taskID,
		Horizon:   pred.Horizon,
		Prediction: pred,
		Validation: validation,
		Verdict:    &verdict,
	}, nil
}

func loadEpisodeCard(episodeDir string) (map[string]interface{}, error) {
	yamlPath := filepath.Join(episodeDir, "completion-card.yaml")
	if data, err := os.ReadFile(yamlPath); err == nil {
		var card map[string]interface{}
		if err := yaml.Unmarshal(data, &card); err != nil {
			return nil, fmt.Errorf("completion-card.yaml parse error: %w", err)
		}
		return card, nil
	}

	jsonPath := filepath.Join(episodeDir, "completion-card.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var card map[string]interface{}
		if err := json.Unmarshal(data, &card); err != nil {
			return nil, fmt.Errorf("completion-card.json parse error: %w", err)
		}
		return card, nil
	}

	return nil, nil
}

func extractPrediction(card map[string]interface{}) (*Prediction, bool) {
	if card == nil {
		return nil, false
	}
	raw, ok := card["prediction"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	pred := &Prediction{
		Claim:               stringValue(raw["claim"]),
		ExpectedEffect:      stringValue(raw["expected_effect"]),
		FalsificationMethod: stringValue(raw["falsification_method"]),
		Horizon:             stringValue(raw["horizon"]),
		MeasurableSignal:    stringValue(raw["measurable_signal"]),
		Confidence:          stringValue(raw["confidence"]),
	}
	return pred, true
}

func stringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// PredictionReport represents aggregated report.
type PredictionReport struct {
	OK               bool                           `json:"ok"`
	Since            string                         `json:"since,omitempty"`
	EpisodesAnalyzed int                            `json:"episodes_analyzed"`
	Confirmed        int                            `json:"confirmed"`
	Falsified        int                            `json:"falsified"`
	Inconclusive     int                            `json:"inconclusive"`
	Results          []PredictionVerificationResult `json:"results"`
}

// GenerateReport aggregates prediction history.
func GenerateReport(episodesDir, since string) (*PredictionReport, error) {
	report := &PredictionReport{
		OK:               true,
		Since:            since,
		EpisodesAnalyzed: 0,
		Confirmed:        0,
		Falsified:        0,
		Inconclusive:     0,
		Results:          []PredictionVerificationResult{},
	}

	entries, err := os.ReadDir(episodesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return nil, fmt.Errorf("failed to read episodes directory: %w", err)
	}

	cutoff, err := parseSince(since)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "ep_") {
			continue
		}
		dir := filepath.Join(episodesDir, entry.Name())
		manifestPath := filepath.Join(dir, "manifest.json")
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest map[string]interface{}
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			continue
		}

		if cutoff != nil {
			createdAt := stringValue(manifest["created_at"])
			t, err := time.Parse(time.RFC3339, createdAt)
			if err != nil {
				continue
			}
			if t.Before(*cutoff) {
				continue
			}
		}

		dirs = append(dirs, dir)
	}

	sort.Strings(dirs)

	for _, dir := range dirs {
		result, err := VerifyPredictionFromEpisode(dir)
		if err != nil {
			continue
		}
		report.Results = append(report.Results, *result)
		switch result.Status {
		case "confirmed":
			report.Confirmed++
		case "falsified":
			report.Falsified++
		case "inconclusive":
			report.Inconclusive++
		}
	}

	report.EpisodesAnalyzed = len(report.Results)
	return report, nil
}

func parseSince(since string) (*time.Time, error) {
	if since == "" {
		return nil, nil
	}
	re := regexp.MustCompile(`^(\d+)([dh])$`)
	match := re.FindStringSubmatch(since)
	if match == nil {
		return nil, fmt.Errorf("invalid since format: %s", since)
	}
	value, _ := strconv.Atoi(match[1])
	var duration time.Duration
	if match[2] == "d" {
		duration = time.Duration(value) * 24 * time.Hour
	} else {
		duration = time.Duration(value) * time.Hour
	}
	cutoff := time.Now().UTC().Add(-duration)
	return &cutoff, nil
}
