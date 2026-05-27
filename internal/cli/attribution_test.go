package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/attribution"
)

func makeEpisodeDir(t *testing.T, manifest map[string]any, trace string) string {
	t.Helper()
	tmp := t.TempDir()
	mdata, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "manifest.json"), mdata, 0644); err != nil {
		t.Fatal(err)
	}
	if trace != "" {
		if err := os.WriteFile(filepath.Join(tmp, "trace.jsonl"), []byte(trace), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return tmp
}

func TestAttributionExplainTextOutput(t *testing.T) {
	manifest := map[string]any{
		"episode_id": "ep_text",
		"task_id":    "task_text",
		"created_at": "2024-01-01T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome":  "failed",
			"acceptance_status":  "withheld",
			"blocking_predicate": "verification_failed",
		},
	}
	dir := makeEpisodeDir(t, manifest, `{"errors":["typecheck failed"]}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode", dir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "# x-harness Failure Attribution") {
		t.Fatalf("expected header, got:\n%s", out)
	}
	if !strings.Contains(out, "episode_id: ep_text") {
		t.Fatalf("expected episode_id, got:\n%s", out)
	}
	if !strings.Contains(out, "taxonomy: Fverification") {
		t.Fatalf("expected Fverification taxonomy, got:\n%s", out)
	}
	if !strings.Contains(out, "predicate: verification_failed") {
		t.Fatalf("expected predicate, got:\n%s", out)
	}
}

func TestAttributionExplainJSONOutput(t *testing.T) {
	manifest := map[string]any{
		"episode_id": "ep_json",
		"task_id":    "task_json",
		"created_at": "2024-01-01T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome":  "failed",
			"acceptance_status":  "withheld",
			"blocking_predicate": "verification_failed",
		},
	}
	dir := makeEpisodeDir(t, manifest, `{"errors":["typecheck failed"]}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode", dir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result attribution.FailureAttribution
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.EpisodeID != "ep_json" {
		t.Fatalf("expected ep_json, got %s", result.EpisodeID)
	}
	if result.Primary == nil || result.Primary.Taxonomy != attribution.Fverification {
		t.Fatalf("expected Fverification, got %+v", result.Primary)
	}
}

func TestAttributionExplainPreExistingShortCircuit(t *testing.T) {
	manifest := map[string]any{
		"episode_id": "ep_pre",
		"task_id":    "task_pre",
		"created_at": "2024-01-01T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome": "failed",
			"acceptance_status": "withheld",
		},
	}
	dir := makeEpisodeDir(t, manifest, "")
	existing := attribution.FailureAttribution{
		SchemaVersion:     "1",
		EpisodeID:         "ep_pre_override",
		TaskID:            "task_pre",
		CreatedAt:         "2024-01-01T00:00:00Z",
		Verdict:           attribution.Verdict{AdmissionOutcome: "failed", AcceptanceStatus: "withheld"},
		Primary:           &attribution.AttributionCandidate{Taxonomy: attribution.Funknown, Predicate: "p", ComponentID: "c", Confidence: "low", Rationale: "r"},
		Candidates:        []attribution.AttributionCandidate{{Taxonomy: attribution.Funknown, Predicate: "p", ComponentID: "c", Confidence: "low", Rationale: "r"}},
		UnknownRateSignal: attribution.UnknownRateSignal{IsUnknown: true, Reason: "r"},
	}
	edata, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "failure-attribution.json"), edata, 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode", dir, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result attribution.FailureAttribution
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.EpisodeID != "ep_pre_override" {
		t.Fatalf("expected pre-existing attribution to short-circuit, got %s", result.EpisodeID)
	}
}

func TestAttributionExplainMissingEpisode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %q", stderr.String())
	}
}

func TestAttributionExplainNonexistentEpisode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode", "/tmp/nonexistent-episode-12345"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "does not exist") {
		t.Fatalf("expected does not exist error, got: %q", stderr.String())
	}
}

func TestAttributionExplainUnsupportedSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "report"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown attribution subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %q", stderr.String())
	}
}

func TestAttributionExplainUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode", ".", "--verbose"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got: %q", stderr.String())
	}
}

func TestAttributionExplainMissingEpisodeValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--episode requires a value") {
		t.Fatalf("expected missing value error, got: %q", stderr.String())
	}
}

func TestAttributionExplainUnexpectedArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "extra"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unexpected argument") {
		t.Fatalf("expected unexpected argument error, got: %q", stderr.String())
	}
}

func TestAttributionExplainEpisodeMustBeDirectory(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "episode.json")
	if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode", filePath}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "not a directory") {
		t.Fatalf("expected not a directory error, got: %q", stderr.String())
	}
}

func TestAttributionExplainAcceptedEpisode(t *testing.T) {
	manifest := map[string]any{
		"episode_id": "ep_accept",
		"task_id":    "task_accept",
		"created_at": "2024-01-01T00:00:00Z",
		"verdict": map[string]any{
			"admission_outcome": "success",
			"acceptance_status": "accepted",
		},
	}
	dir := makeEpisodeDir(t, manifest, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"attribution", "explain", "--episode", dir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "taxonomy: none") {
		t.Fatalf("expected taxonomy none for accepted episode, got:\n%s", out)
	}
	if !strings.Contains(out, "predicate: none") {
		t.Fatalf("expected predicate none for accepted episode, got:\n%s", out)
	}
}
