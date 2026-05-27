package attribution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func ptr(s string) *string {
	return &s
}

func TestCreateFailureAttributionAccepted(t *testing.T) {
	input := AttributionInput{
		EpisodeID:        "ep_accepted",
		TaskID:           "task_1",
		CreatedAt:        "2024-01-01T00:00:00Z",
		AdmissionOutcome: "success",
		AcceptanceStatus: "accepted",
	}
	attr := CreateFailureAttribution(input)
	if attr.SchemaVersion != "1" {
		t.Fatalf("expected schema_version 1, got %s", attr.SchemaVersion)
	}
	if attr.Primary != nil {
		t.Fatalf("expected no primary for accepted episode")
	}
	if len(attr.Candidates) != 0 {
		t.Fatalf("expected no candidates for accepted episode")
	}
	if attr.UnknownRateSignal.IsUnknown {
		t.Fatal("expected is_unknown false for accepted")
	}
	if attr.AdmissionAuthority {
		t.Fatal("expected admission_authority false")
	}
}

func TestCreateFailureAttributionFverification(t *testing.T) {
	input := AttributionInput{
		EpisodeID:         "ep_1",
		TaskID:            "task_1",
		CreatedAt:         "2024-01-01T00:00:00Z",
		AdmissionOutcome:  "failed",
		AcceptanceStatus:  "withheld",
		BlockingPredicate: ptr("evidence_missing"),
		Errors:            []string{"prediction mismatch"},
	}
	attr := CreateFailureAttribution(input)
	if attr.Primary == nil {
		t.Fatal("expected primary")
	}
	if attr.Primary.Taxonomy != Fverification {
		t.Fatalf("expected Fverification, got %s", attr.Primary.Taxonomy)
	}
	if attr.Primary.Predicate != "evidence_missing" {
		t.Fatalf("expected predicate evidence_missing, got %s", attr.Primary.Predicate)
	}
	if attr.Primary.ComponentID != "admission_policy" {
		t.Fatalf("expected component_id admission_policy, got %s", attr.Primary.ComponentID)
	}
	if attr.Primary.Confidence != "high" {
		t.Fatalf("expected confidence high, got %s", attr.Primary.Confidence)
	}
	if attr.UnknownRateSignal.IsUnknown {
		t.Fatal("expected is_unknown false")
	}
}

func TestCreateFailureAttributionFpermission(t *testing.T) {
	input := AttributionInput{
		EpisodeID:         "ep_1",
		TaskID:            "task_1",
		CreatedAt:         "2024-01-01T00:00:00Z",
		AdmissionOutcome:  "blocked",
		AcceptanceStatus:  "withheld",
		BlockingPredicate: ptr("verifier_not_read_only"),
		Errors:            []string{"mutation guard violated"},
	}
	attr := CreateFailureAttribution(input)
	if attr.Primary == nil || attr.Primary.Taxonomy != Fpermission {
		t.Fatalf("expected Fpermission, got %+v", attr.Primary)
	}
	if attr.Primary.Predicate != "verifier_not_read_only" {
		t.Fatalf("expected predicate verifier_not_read_only, got %s", attr.Primary.Predicate)
	}
}

func TestCreateFailureAttributionFintervention(t *testing.T) {
	input := AttributionInput{
		EpisodeID:        "ep_1",
		TaskID:           "task_1",
		CreatedAt:        "2024-01-01T00:00:00Z",
		AdmissionOutcome: "failed",
		AcceptanceStatus: "withheld",
		Notes:            []string{"approval missing for downgrade"},
	}
	attr := CreateFailureAttribution(input)
	if attr.Primary == nil || attr.Primary.Taxonomy != Fintervention {
		t.Fatalf("expected Fintervention, got %+v", attr.Primary)
	}
}

func TestCreateFailureAttributionFcontext(t *testing.T) {
	input := AttributionInput{
		EpisodeID:        "ep_1",
		TaskID:           "task_1",
		CreatedAt:        "2024-01-01T00:00:00Z",
		AdmissionOutcome: "failed",
		AcceptanceStatus: "withheld",
		Errors:           []string{"stale context detected"},
	}
	attr := CreateFailureAttribution(input)
	if attr.Primary == nil || attr.Primary.Taxonomy != Fcontext {
		t.Fatalf("expected Fcontext, got %+v", attr.Primary)
	}
}

func TestCreateFailureAttributionFobservability(t *testing.T) {
	input := AttributionInput{
		EpisodeID:        "ep_1",
		TaskID:           "task_1",
		CreatedAt:        "2024-01-01T00:00:00Z",
		AdmissionOutcome: "failed",
		AcceptanceStatus: "withheld",
		Errors:           []string{"manifest schema invalid"},
	}
	attr := CreateFailureAttribution(input)
	if attr.Primary == nil || attr.Primary.Taxonomy != Fobservability {
		t.Fatalf("expected Fobservability, got %+v", attr.Primary)
	}
}

func TestCreateFailureAttributionFentropy(t *testing.T) {
	input := AttributionInput{
		EpisodeID:        "ep_1",
		TaskID:           "task_1",
		CreatedAt:        "2024-01-01T00:00:00Z",
		AdmissionOutcome: "failed",
		AcceptanceStatus: "withheld",
		Notes:            []string{"policy drift in tier label"},
	}
	attr := CreateFailureAttribution(input)
	if attr.Primary == nil || attr.Primary.Taxonomy != Fentropy {
		t.Fatalf("expected Fentropy, got %+v", attr.Primary)
	}
}

func TestCreateFailureAttributionFunknown(t *testing.T) {
	input := AttributionInput{
		EpisodeID:        "ep_1",
		TaskID:           "task_1",
		CreatedAt:        "2024-01-01T00:00:00Z",
		AdmissionOutcome: "failed",
		AcceptanceStatus: "withheld",
		Errors:           []string{"something completely unrelated"},
	}
	attr := CreateFailureAttribution(input)
	if attr.Primary == nil || attr.Primary.Taxonomy != Funknown {
		t.Fatalf("expected Funknown, got %+v", attr.Primary)
	}
	if !attr.UnknownRateSignal.IsUnknown {
		t.Fatal("expected is_unknown true")
	}
}

func TestLoadOrCreateAttributionReadsExisting(t *testing.T) {
	tmp := t.TempDir()
	existing := FailureAttribution{
		SchemaVersion:      "1",
		EpisodeID:          "ep_existing",
		TaskID:             "task_existing",
		CreatedAt:          "2024-01-01T00:00:00Z",
		Verdict:            Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
		Primary:            &AttributionCandidate{Taxonomy: Fverification, Predicate: "p", ComponentID: "c", Confidence: "high", Rationale: "r"},
		Candidates:         []AttributionCandidate{{Taxonomy: Fverification, Predicate: "p", ComponentID: "c", Confidence: "high", Rationale: "r"}},
		UnknownRateSignal:  UnknownRateSignal{IsUnknown: false, Reason: "r"},
		AdmissionAuthority: false,
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, "failure-attribution.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	attr, err := LoadOrCreateAttribution(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attr.EpisodeID != "ep_existing" {
		t.Fatalf("expected existing episode, got %s", attr.EpisodeID)
	}
}

func TestLoadOrCreateAttributionCreatesFromManifestAndTrace(t *testing.T) {
	tmp := t.TempDir()
	manifest := map[string]any{
		"episode_id": "ep_new",
		"task_id":    "task_new",
		"created_at": "2024-01-01T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome":  "failed",
			"acceptance_status":  "withheld",
			"blocking_predicate": "typecheck_failed",
		},
	}
	mdata, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(tmp, "manifest.json"), mdata, 0644); err != nil {
		t.Fatal(err)
	}

	trace := `{"errors":["typecheck failed"],"notes":["lint issue"]}
`
	if err := os.WriteFile(filepath.Join(tmp, "trace.jsonl"), []byte(trace), 0644); err != nil {
		t.Fatal(err)
	}

	attr, err := LoadOrCreateAttribution(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attr.EpisodeID != "ep_new" {
		t.Fatalf("expected ep_new, got %s", attr.EpisodeID)
	}
	if attr.Primary == nil || attr.Primary.Taxonomy != Fverification {
		t.Fatalf("expected Fverification from trace errors, got %+v", attr.Primary)
	}

	// Check file was written
	if _, err := os.Stat(filepath.Join(tmp, "failure-attribution.json")); os.IsNotExist(err) {
		t.Fatal("expected failure-attribution.json to be written")
	}
}

func TestLoadOrCreateAttributionMissingTrace(t *testing.T) {
	tmp := t.TempDir()
	manifest := map[string]any{
		"episode_id": "ep_notrace",
		"task_id":    "task_notrace",
		"created_at": "2024-01-01T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome": "failed",
			"acceptance_status": "withheld",
		},
	}
	mdata, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(tmp, "manifest.json"), mdata, 0644); err != nil {
		t.Fatal(err)
	}

	attr, err := LoadOrCreateAttribution(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attr.EpisodeID != "ep_notrace" {
		t.Fatalf("expected ep_notrace, got %s", attr.EpisodeID)
	}
}
