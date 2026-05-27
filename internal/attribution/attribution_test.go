package attribution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestListAttributionsSorted(t *testing.T) {
	root := t.TempDir()
	ep1 := filepath.Join(root, "ep_001")
	ep2 := filepath.Join(root, "ep_002")
	os.MkdirAll(ep1, 0755)
	os.MkdirAll(ep2, 0755)

	m1 := map[string]any{
		"episode_id": "ep_001",
		"task_id":    "task_1",
		"created_at": "2024-01-02T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome": "success",
			"acceptance_status": "accepted",
		},
	}
	m2 := map[string]any{
		"episode_id": "ep_002",
		"task_id":    "task_2",
		"created_at": "2024-01-01T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome": "failed",
			"acceptance_status": "withheld",
		},
	}
	m1data, _ := json.Marshal(m1)
	m2data, _ := json.Marshal(m2)
	os.WriteFile(filepath.Join(ep1, "manifest.json"), m1data, 0644)
	os.WriteFile(filepath.Join(ep2, "manifest.json"), m2data, 0644)

	attrs, err := ListAttributions(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributions, got %d", len(attrs))
	}
	if attrs[0].EpisodeID != "ep_002" {
		t.Fatalf("expected first ep_002 (earlier created_at), got %s", attrs[0].EpisodeID)
	}
	if attrs[1].EpisodeID != "ep_001" {
		t.Fatalf("expected second ep_001, got %s", attrs[1].EpisodeID)
	}
}

func TestListAttributionsMissingDir(t *testing.T) {
	attrs, err := ListAttributions("/tmp/nonexistent-episodes-12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attrs) != 0 {
		t.Fatalf("expected empty, got %d", len(attrs))
	}
}

func TestBuildAttributionReportGroupsByPredicate(t *testing.T) {
	attrs := []FailureAttribution{
		{
			EpisodeID: "ep_1",
			CreatedAt: "2024-01-01T00:00:00Z",
			Verdict:   Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
			Primary:   &AttributionCandidate{Taxonomy: Fverification, Predicate: "p1", ComponentID: "c1", Confidence: "high", Rationale: "r"},
		},
		{
			EpisodeID: "ep_2",
			CreatedAt: "2024-01-02T00:00:00Z",
			Verdict:   Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
			Primary:   &AttributionCandidate{Taxonomy: Fverification, Predicate: "p1", ComponentID: "c1", Confidence: "high", Rationale: "r"},
		},
		{
			EpisodeID: "ep_3",
			CreatedAt: "2024-01-03T00:00:00Z",
			Verdict:   Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
			Primary:   &AttributionCandidate{Taxonomy: Funknown, Predicate: "unknown_failure", ComponentID: "unknown", Confidence: "low", Rationale: "r"},
		},
	}

	report := BuildAttributionReport(attrs, "predicate")
	if report.TotalEpisodes != 3 {
		t.Fatalf("expected total 3, got %d", report.TotalEpisodes)
	}
	if report.WithheldEpisodes != 3 {
		t.Fatalf("expected withheld 3, got %d", report.WithheldEpisodes)
	}
	if report.UnknownCount != 1 {
		t.Fatalf("expected unknown 1, got %d", report.UnknownCount)
	}
	if report.UnknownRate != 0.3333 {
		t.Fatalf("expected unknown_rate 0.3333, got %g", report.UnknownRate)
	}
	if len(report.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(report.Groups))
	}
	if report.Groups[0].Key != "p1" || report.Groups[0].Count != 2 {
		t.Fatalf("expected first group p1 count 2, got %s %d", report.Groups[0].Key, report.Groups[0].Count)
	}
	if report.Groups[1].Key != "unknown_failure" || report.Groups[1].Count != 1 {
		t.Fatalf("expected second group unknown_failure count 1, got %s %d", report.Groups[1].Key, report.Groups[1].Count)
	}
	if report.EntropyWarning != nil {
		t.Fatal("expected no entropy warning when unknown rate < 0.5")
	}
}

func TestBuildAttributionReportGroupsByTaxonomy(t *testing.T) {
	attrs := []FailureAttribution{
		{
			EpisodeID: "ep_1",
			CreatedAt: "2024-01-01T00:00:00Z",
			Verdict:   Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
			Primary:   &AttributionCandidate{Taxonomy: Fverification, Predicate: "p1", ComponentID: "c1", Confidence: "high", Rationale: "r"},
		},
		{
			EpisodeID: "ep_2",
			CreatedAt: "2024-01-02T00:00:00Z",
			Verdict:   Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
			Primary:   &AttributionCandidate{Taxonomy: Fpermission, Predicate: "p2", ComponentID: "c2", Confidence: "high", Rationale: "r"},
		},
	}

	report := BuildAttributionReport(attrs, "taxonomy")
	if len(report.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(report.Groups))
	}
	if report.Groups[0].Key != string(Fverification) && report.Groups[0].Key != string(Fpermission) {
		t.Fatalf("unexpected group key: %s", report.Groups[0].Key)
	}
}

func TestBuildAttributionReportGroupsByComponent(t *testing.T) {
	attrs := []FailureAttribution{
		{
			EpisodeID: "ep_1",
			CreatedAt: "2024-01-01T00:00:00Z",
			Verdict:   Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
			Primary:   &AttributionCandidate{Taxonomy: Fverification, Predicate: "p1", ComponentID: "c1", Confidence: "high", Rationale: "r"},
		},
	}

	report := BuildAttributionReport(attrs, "component")
	if len(report.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(report.Groups))
	}
	if report.Groups[0].Key != "c1" {
		t.Fatalf("expected component c1, got %s", report.Groups[0].Key)
	}
}

func TestBuildAttributionReportEmpty(t *testing.T) {
	report := BuildAttributionReport([]FailureAttribution{}, "predicate")
	if report.TotalEpisodes != 0 {
		t.Fatalf("expected total 0, got %d", report.TotalEpisodes)
	}
	if report.UnknownRate != 0 {
		t.Fatalf("expected unknown_rate 0, got %g", report.UnknownRate)
	}
	if report.EntropyWarning != nil {
		t.Fatal("expected no entropy warning")
	}
}

func TestParseSinceDuration(t *testing.T) {
	if ParseSinceDuration("") != 0 {
		t.Fatal("expected 0 for empty")
	}
	if ParseSinceDuration("invalid") != 0 {
		t.Fatal("expected 0 for invalid")
	}
	if ParseSinceDuration("7d") != 7*24*time.Hour {
		t.Fatal("expected 7*24 hours for 7d")
	}
	if ParseSinceDuration("12h") != 12*time.Hour {
		t.Fatal("expected 12 hours for 12h")
	}
}

func TestFilterSince(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour).Format(time.RFC3339)
	recent := now.Add(-1 * time.Hour).Format(time.RFC3339)

	attrs := []FailureAttribution{
		{EpisodeID: "ep_old", CreatedAt: old},
		{EpisodeID: "ep_recent", CreatedAt: recent},
	}

	filtered := FilterSince(attrs, "24h")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered, got %d", len(filtered))
	}
	if filtered[0].EpisodeID != "ep_recent" {
		t.Fatalf("expected ep_recent, got %s", filtered[0].EpisodeID)
	}

	all := FilterSince(attrs, "")
	if len(all) != 2 {
		t.Fatalf("expected 2 for empty since, got %d", len(all))
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
