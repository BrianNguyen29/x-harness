package attribution

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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

// ReportGroup represents a single group in the attribution report.
type ReportGroup struct {
	Key        string   `json:"key"`
	Count      int      `json:"count"`
	EpisodeIDs []string `json:"episode_ids"`
	Predicates []string `json:"predicates"`
	Taxonomies []string `json:"taxonomies"`
	Components []string `json:"components"`
}

// AttributionReport is the aggregate report across episodes.
type AttributionReport struct {
	OK               bool          `json:"ok"`
	GroupBy          string        `json:"group_by"`
	TotalEpisodes    int           `json:"total_episodes"`
	WithheldEpisodes int           `json:"withheld_episodes"`
	UnknownCount     int           `json:"unknown_count"`
	UnknownRate      float64       `json:"unknown_rate"`
	Groups           []ReportGroup `json:"groups"`
	EntropyWarning   *string       `json:"entropy_warning"`
}

// ListAttributions scans an episodes directory and loads/creates attributions.
func ListAttributions(episodesDir string) ([]FailureAttribution, error) {
	if _, err := os.Stat(episodesDir); os.IsNotExist(err) {
		return []FailureAttribution{}, nil
	}

	entries, err := os.ReadDir(episodesDir)
	if err != nil {
		return nil, err
	}

	var attributions []FailureAttribution
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "ep_") {
			continue
		}
		dir := filepath.Join(episodesDir, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, "manifest.json")); os.IsNotExist(err) {
			continue
		}
		attr, err := LoadOrCreateAttribution(dir)
		if err != nil {
			continue
		}
		attributions = append(attributions, *attr)
	}

	sort.Slice(attributions, func(i, j int) bool {
		a, errA := time.Parse(time.RFC3339, attributions[i].CreatedAt)
		b, errB := time.Parse(time.RFC3339, attributions[j].CreatedAt)
		if errA != nil || errB != nil {
			return attributions[i].CreatedAt < attributions[j].CreatedAt
		}
		return a.Before(b)
	})

	return attributions, nil
}

// BuildAttributionReport aggregates attributions into a report grouped by the specified field.
func BuildAttributionReport(attributions []FailureAttribution, groupBy string) AttributionReport {
	var withheld []FailureAttribution
	for _, item := range attributions {
		if item.Verdict.AcceptanceStatus == "withheld" {
			withheld = append(withheld, item)
		}
	}

	unknownCount := 0
	for _, item := range withheld {
		if item.Primary != nil && item.Primary.Taxonomy == Funknown {
			unknownCount++
		}
	}

	type groupAggregate struct {
		count      int
		episodeIDs map[string]struct{}
		predicates map[string]struct{}
		taxonomies map[string]struct{}
		components map[string]struct{}
	}

	groups := make(map[string]*groupAggregate)

	for _, item := range withheld {
		primary := item.Primary
		var key string
		switch groupBy {
		case "predicate":
			if primary != nil {
				key = primary.Predicate
			} else {
				key = "none"
			}
		case "taxonomy":
			if primary != nil {
				key = string(primary.Taxonomy)
			} else {
				key = "none"
			}
		case "component":
			if primary != nil {
				key = primary.ComponentID
			} else {
				key = "none"
			}
		default:
			key = "none"
		}

		g, ok := groups[key]
		if !ok {
			g = &groupAggregate{
				episodeIDs: make(map[string]struct{}),
				predicates: make(map[string]struct{}),
				taxonomies: make(map[string]struct{}),
				components: make(map[string]struct{}),
			}
			groups[key] = g
		}
		g.count++
		g.episodeIDs[item.EpisodeID] = struct{}{}
		if primary != nil {
			g.predicates[primary.Predicate] = struct{}{}
			g.taxonomies[string(primary.Taxonomy)] = struct{}{}
			g.components[primary.ComponentID] = struct{}{}
		}
	}

	var reportGroups []ReportGroup
	for key, g := range groups {
		rg := ReportGroup{
			Key:   key,
			Count: g.count,
		}
		for id := range g.episodeIDs {
			rg.EpisodeIDs = append(rg.EpisodeIDs, id)
		}
		for p := range g.predicates {
			rg.Predicates = append(rg.Predicates, p)
		}
		for t := range g.taxonomies {
			rg.Taxonomies = append(rg.Taxonomies, t)
		}
		for c := range g.components {
			rg.Components = append(rg.Components, c)
		}
		sort.Strings(rg.EpisodeIDs)
		sort.Strings(rg.Predicates)
		sort.Strings(rg.Taxonomies)
		sort.Strings(rg.Components)
		reportGroups = append(reportGroups, rg)
	}

	sort.Slice(reportGroups, func(i, j int) bool {
		if reportGroups[i].Count != reportGroups[j].Count {
			return reportGroups[i].Count > reportGroups[j].Count
		}
		return reportGroups[i].Key < reportGroups[j].Key
	})

	var unknownRate float64
	if len(withheld) > 0 {
		unknownRate = float64(unknownCount) / float64(len(withheld))
	}
	unknownRate = float64(int(unknownRate*10000+0.5)) / 10000

	var entropyWarning *string
	if len(withheld) > 0 && unknownRate >= 0.5 {
		msg := "high Funknown attribution rate; inspect failure taxonomy and episode observability"
		entropyWarning = &msg
	}

	return AttributionReport{
		OK:               true,
		GroupBy:          groupBy,
		TotalEpisodes:    len(attributions),
		WithheldEpisodes: len(withheld),
		UnknownCount:     unknownCount,
		UnknownRate:      unknownRate,
		Groups:           reportGroups,
		EntropyWarning:   entropyWarning,
	}
}

var sinceRegex = regexp.MustCompile(`^(\d+)([dh])$`)

// ParseSinceDuration parses a since string like "7d" or "12h" into a time.Duration.
func ParseSinceDuration(since string) time.Duration {
	if since == "" {
		return 0
	}
	match := sinceRegex.FindStringSubmatch(since)
	if match == nil {
		return 0
	}
	value, _ := strconv.Atoi(match[1])
	if match[2] == "d" {
		return time.Duration(value) * 24 * time.Hour
	}
	return time.Duration(value) * time.Hour
}

// FilterSince filters attributions by created_at age.
func FilterSince(attributions []FailureAttribution, since string) []FailureAttribution {
	d := ParseSinceDuration(since)
	if d == 0 {
		return attributions
	}
	cutoff := time.Now().Add(-d)
	var filtered []FailureAttribution
	for _, item := range attributions {
		ts, err := time.Parse(time.RFC3339, item.CreatedAt)
		if err != nil {
			continue
		}
		if ts.After(cutoff) || ts.Equal(cutoff) {
			filtered = append(filtered, item)
		}
	}
	return filtered
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
