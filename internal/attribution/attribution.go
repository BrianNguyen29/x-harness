package attribution

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FailureTaxonomy is the deterministic taxonomy for failure attribution.
type FailureTaxonomy string

const (
	Ftask_spec     FailureTaxonomy = "Ftask_spec"
	Fcontext       FailureTaxonomy = "Fcontext"
	Ftool          FailureTaxonomy = "Ftool"
	Fmemory        FailureTaxonomy = "Fmemory"
	Fstate         FailureTaxonomy = "Fstate"
	Fobservability FailureTaxonomy = "Fobservability"
	Fattribution   FailureTaxonomy = "Fattribution"
	Fverification  FailureTaxonomy = "Fverification"
	Fpermission    FailureTaxonomy = "Fpermission"
	Fentropy       FailureTaxonomy = "Fentropy"
	Fintervention  FailureTaxonomy = "Fintervention"
	Fmodel         FailureTaxonomy = "Fmodel"
	Funknown       FailureTaxonomy = "Funknown"
)

// AttributionCandidate represents a single attribution candidate.
type AttributionCandidate struct {
	Taxonomy    FailureTaxonomy `json:"taxonomy"`
	Predicate   string          `json:"predicate"`
	ComponentID string          `json:"component_id"`
	Confidence  string          `json:"confidence"`
	Rationale   string          `json:"rationale"`
}

// FailureAttribution is the deterministic attribution for withheld or failed episode outcomes.
type FailureAttribution struct {
	SchemaVersion      string                 `json:"schema_version"`
	EpisodeID          string                 `json:"episode_id"`
	TaskID             string                 `json:"task_id"`
	CreatedAt          string                 `json:"created_at"`
	Verdict            Verdict                `json:"verdict"`
	Primary            *AttributionCandidate  `json:"primary"`
	Candidates         []AttributionCandidate `json:"candidates"`
	UnknownRateSignal  UnknownRateSignal      `json:"unknown_rate_signal"`
	AdmissionAuthority bool                   `json:"admission_authority"`
}

// Verdict captures the episode verdict.
type Verdict struct {
	AdmissionOutcome  string  `json:"admission_outcome"`
	AcceptanceStatus  string  `json:"acceptance_status"`
	BlockingPredicate *string `json:"blocking_predicate"`
}

// UnknownRateSignal signals whether the attribution fell into unknown.
type UnknownRateSignal struct {
	IsUnknown bool   `json:"is_unknown"`
	Reason    string `json:"reason"`
}

// AttributionInput is the input for creating a failure attribution.
type AttributionInput struct {
	EpisodeID         string
	TaskID            string
	CreatedAt         string
	AdmissionOutcome  string
	AcceptanceStatus  string
	BlockingPredicate *string
	Errors            []string
	Notes             []string
}

func newCandidate(taxonomy FailureTaxonomy, predicate, componentID, confidence, rationale string) AttributionCandidate {
	return AttributionCandidate{
		Taxonomy:    taxonomy,
		Predicate:   predicate,
		ComponentID: componentID,
		Confidence:  confidence,
		Rationale:   rationale,
	}
}

// CreateFailureAttribution deterministically creates a FailureAttribution from input.
func CreateFailureAttribution(input AttributionInput) FailureAttribution {
	verdict := Verdict{
		AdmissionOutcome:  input.AdmissionOutcome,
		AcceptanceStatus:  input.AcceptanceStatus,
		BlockingPredicate: input.BlockingPredicate,
	}

	if verdict.AdmissionOutcome == "success" && verdict.AcceptanceStatus == "accepted" {
		return FailureAttribution{
			SchemaVersion:      "1",
			EpisodeID:          input.EpisodeID,
			TaskID:             input.TaskID,
			CreatedAt:          input.CreatedAt,
			Verdict:            verdict,
			Primary:            nil,
			Candidates:         []AttributionCandidate{},
			UnknownRateSignal:  UnknownRateSignal{IsUnknown: false, Reason: "accepted episode has no failure attribution"},
			AdmissionAuthority: false,
		}
	}

	textParts := []string{}
	if input.BlockingPredicate != nil {
		textParts = append(textParts, *input.BlockingPredicate)
	}
	textParts = append(textParts, input.Errors...)
	textParts = append(textParts, input.Notes...)
	text := strings.ToLower(strings.Join(textParts, "\n"))

	var primary AttributionCandidate
	switch {
	case strings.Contains(text, "evidence") || strings.Contains(text, "prediction") || strings.Contains(text, "verification") || strings.Contains(text, "typecheck"):
		pred := "verification_failed"
		if input.BlockingPredicate != nil {
			pred = *input.BlockingPredicate
		}
		primary = newCandidate(Fverification, pred, "admission_policy", "high", "Verify/admission evidence or prediction requirements were not satisfied.")
	case strings.Contains(text, "mutation guard") || strings.Contains(text, "verifier_not_read_only") || strings.Contains(text, "read-only"):
		pred := "verifier_not_read_only"
		if input.BlockingPredicate != nil {
			pred = *input.BlockingPredicate
		}
		primary = newCandidate(Fpermission, pred, "verify_runtime", "high", "Verifier read-only or mutation guard boundary was violated.")
	case strings.Contains(text, "approval") || strings.Contains(text, "intervention") || strings.Contains(text, "downgrade"):
		pred := "approval_missing"
		if input.BlockingPredicate != nil {
			pred = *input.BlockingPredicate
		}
		primary = newCandidate(Fintervention, pred, "governance_boundary", "high", "Human approval, intervention, or tier downgrade authorization was missing or invalid.")
	case strings.Contains(text, "context") || strings.Contains(text, "stale") || strings.Contains(text, "managed block"):
		pred := "context_stale"
		if input.BlockingPredicate != nil {
			pred = *input.BlockingPredicate
		}
		primary = newCandidate(Fcontext, pred, "agent_contract", "medium", "Context was stale, missing, or not acknowledged.")
	case strings.Contains(text, "schema") || strings.Contains(text, "manifest") || strings.Contains(text, "trace") || strings.Contains(text, "episode"):
		pred := "observability_invalid"
		if input.BlockingPredicate != nil {
			pred = *input.BlockingPredicate
		}
		primary = newCandidate(Fobservability, pred, "episode_packager", "medium", "Trace, schema, or episode observability artifact was missing or malformed.")
	case strings.Contains(text, "component") || strings.Contains(text, "policy drift") || strings.Contains(text, "tier label"):
		pred := "harness_drift"
		if input.BlockingPredicate != nil {
			pred = *input.BlockingPredicate
		}
		primary = newCandidate(Fentropy, pred, "component_registry", "medium", "Harness metadata or policy drift signal was detected.")
	default:
		pred := "unknown_failure"
		if input.BlockingPredicate != nil {
			pred = *input.BlockingPredicate
		}
		primary = newCandidate(Funknown, pred, "unknown", "low", "No deterministic attribution rule matched the episode data.")
	}

	isUnknown := primary.Taxonomy == Funknown
	reason := "deterministic attribution rule matched"
	if isUnknown {
		reason = "no attribution rule matched"
	}

	return FailureAttribution{
		SchemaVersion:      "1",
		EpisodeID:          input.EpisodeID,
		TaskID:             input.TaskID,
		CreatedAt:          input.CreatedAt,
		Verdict:            verdict,
		Primary:            &primary,
		Candidates:         []AttributionCandidate{primary},
		UnknownRateSignal:  UnknownRateSignal{IsUnknown: isUnknown, Reason: reason},
		AdmissionAuthority: false,
	}
}

// LoadOrCreateAttribution loads an existing failure-attribution.json or creates one from manifest.json and trace.jsonl.
func LoadOrCreateAttribution(episodeDir string) (*FailureAttribution, error) {
	attributionPath := filepath.Join(episodeDir, "failure-attribution.json")
	if data, err := os.ReadFile(attributionPath); err == nil {
		var attr FailureAttribution
		if err := json.Unmarshal(data, &attr); err != nil {
			return nil, fmt.Errorf("failed to parse existing failure-attribution.json: %w", err)
		}
		return &attr, nil
	}

	manifestPath := filepath.Join(episodeDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest.json: %w", err)
	}

	var manifest struct {
		EpisodeID string `json:"episode_id"`
		TaskID    string `json:"task_id"`
		CreatedAt string `json:"created_at"`
		Verdict   struct {
			AdmissionOutcome  string  `json:"admission_outcome"`
			AcceptanceStatus  string  `json:"acceptance_status"`
			BlockingPredicate *string `json:"blocking_predicate"`
		} `json:"verdict"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest.json: %w", err)
	}

	var errors, notes []string
	tracePath := filepath.Join(episodeDir, "trace.jsonl")
	if f, err := os.Open(tracePath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		var lastEvent map[string]any
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var event map[string]any
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				lastEvent = event
			}
		}
		if lastEvent != nil {
			if e, ok := lastEvent["errors"]; ok {
				errors = parseStringSlice(e)
			}
			if n, ok := lastEvent["notes"]; ok {
				notes = parseStringSlice(n)
			}
		}
	}

	attr := CreateFailureAttribution(AttributionInput{
		EpisodeID:         manifest.EpisodeID,
		TaskID:            manifest.TaskID,
		CreatedAt:         manifest.CreatedAt,
		AdmissionOutcome:  manifest.Verdict.AdmissionOutcome,
		AcceptanceStatus:  manifest.Verdict.AcceptanceStatus,
		BlockingPredicate: manifest.Verdict.BlockingPredicate,
		Errors:            errors,
		Notes:             notes,
	})

	out, err := json.MarshalIndent(attr, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attribution: %w", err)
	}
	if err := os.WriteFile(attributionPath, out, 0644); err != nil {
		return nil, fmt.Errorf("failed to write failure-attribution.json: %w", err)
	}

	return &attr, nil
}

func parseStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
