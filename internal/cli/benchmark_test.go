package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBenchmarkUpdateSnapshotsRejects(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"benchmark", "--update-snapshots"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "human-approved boundary change") {
		t.Fatalf("expected rejection message, got %q", stderr.String())
	}
}

func TestBenchmarkMutationGuardJSONShape(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"benchmark", "--filter", "mutation-guard", "--mutation-files", "3", "--mutation-concurrency", "1,2", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result benchmarkResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got ok=%v", result.OK)
	}
	if result.Filter != "mutation-guard" {
		t.Fatalf("expected filter=mutation-guard, got %s", result.Filter)
	}
	if result.Integration != nil {
		t.Fatalf("expected integration=null, got %v", result.Integration)
	}
	if len(result.Results) != 0 {
		t.Fatalf("expected results=[], got %v", result.Results)
	}
	if result.MutationGuardBenchmark == nil {
		t.Fatal("expected mutation_guard_benchmark")
	}
	if !result.MutationGuardBenchmark.OK {
		t.Fatal("expected mutation_guard_benchmark.ok=true")
	}
	if !reflect.DeepEqual(result.MutationGuardBenchmark.FileCounts, []int{3}) {
		t.Fatalf("expected file_counts=[3], got %v", result.MutationGuardBenchmark.FileCounts)
	}
	if !reflect.DeepEqual(result.MutationGuardBenchmark.Concurrency, []int{1, 2}) {
		t.Fatalf("expected concurrency=[1,2], got %v", result.MutationGuardBenchmark.Concurrency)
	}
	if len(result.MutationGuardBenchmark.Cases) != 4 {
		t.Fatalf("expected 4 cases, got %d", len(result.MutationGuardBenchmark.Cases))
	}
	for _, c := range result.MutationGuardBenchmark.Cases {
		if c.HashedPaths != 3 {
			t.Fatalf("expected hashed_paths=3, got %d", c.HashedPaths)
		}
		if !c.OK {
			t.Fatalf("expected case ok=true for mode=%s", c.Mode)
		}
	}
	modes := make(map[string]int)
	for _, c := range result.MutationGuardBenchmark.Cases {
		modes[c.Mode]++
	}
	if modes["git"] != 2 || modes["non-git"] != 2 {
		t.Fatalf("expected 2 git and 2 non-git cases, got %v", modes)
	}
}

func TestBenchmarkAdversarialJSONShape(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"benchmark", "--filter", "adversarial", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		OK          bool   `json:"ok"`
		Filter      string `json:"filter"`
		Integration struct {
			Adversarial struct {
				CasesTotal int `json:"cases_total"`
				Cases      []struct {
					Name                        string   `json:"name"`
					PermissionViolationDetected bool     `json:"permission_violation_detected"`
					AuthorityViolationDetected  bool     `json:"authority_violation_detected"`
					MutationGuardDetected       bool     `json:"mutation_guard_detected"`
					BlockingPredicate           string   `json:"blocking_predicate"`
					Errors                      []string `json:"errors"`
				} `json:"cases"`
			} `json:"adversarial"`
		} `json:"integration"`
		Metrics struct {
			AdversarialFalseAcceptCount      int     `json:"adversarial_false_accept_count"`
			AdversarialBlockRate             float64 `json:"adversarial_block_rate"`
			MutationGuardDetectionRate       float64 `json:"mutation_guard_detection_rate"`
			PermissionViolationDetectionRate float64 `json:"permission_violation_detection_rate"`
			AuthorityViolationDetectionRate  float64 `json:"authority_violation_detection_rate"`
		} `json:"metrics"`
		MutationGuardBenchmark interface{}   `json:"mutation_guard_benchmark"`
		Results                []interface{} `json:"results"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got ok=%v", result.OK)
	}
	if result.Filter != "adversarial" {
		t.Fatalf("expected filter=adversarial, got %s", result.Filter)
	}
	if result.MutationGuardBenchmark != nil {
		t.Fatalf("expected mutation_guard_benchmark=null, got %v", result.MutationGuardBenchmark)
	}
	if len(result.Results) != 0 {
		t.Fatalf("expected results=[], got %v", result.Results)
	}
	if result.Integration.Adversarial.CasesTotal <= 0 {
		t.Fatalf("expected cases_total > 0, got %d", result.Integration.Adversarial.CasesTotal)
	}
	if result.Metrics.AdversarialFalseAcceptCount != 0 {
		t.Fatalf("expected adversarial_false_accept_count=0, got %d", result.Metrics.AdversarialFalseAcceptCount)
	}
	if result.Metrics.AdversarialBlockRate != 1 {
		t.Fatalf("expected adversarial_block_rate=1, got %v", result.Metrics.AdversarialBlockRate)
	}
	if result.Metrics.MutationGuardDetectionRate != 1 {
		t.Fatalf("expected mutation_guard_detection_rate=1, got %v", result.Metrics.MutationGuardDetectionRate)
	}
	if result.Metrics.PermissionViolationDetectionRate != 1 {
		t.Fatalf("expected permission_violation_detection_rate=1, got %v", result.Metrics.PermissionViolationDetectionRate)
	}
	if result.Metrics.AuthorityViolationDetectionRate != 1 {
		t.Fatalf("expected authority_violation_detection_rate=1, got %v", result.Metrics.AuthorityViolationDetectionRate)
	}

	var dangerousCase, authorityCase, mutationCase *struct {
		Name                        string   `json:"name"`
		PermissionViolationDetected bool     `json:"permission_violation_detected"`
		AuthorityViolationDetected  bool     `json:"authority_violation_detected"`
		MutationGuardDetected       bool     `json:"mutation_guard_detected"`
		BlockingPredicate           string   `json:"blocking_predicate"`
		Errors                      []string `json:"errors"`
	}
	for i := range result.Integration.Adversarial.Cases {
		c := &result.Integration.Adversarial.Cases[i]
		switch c.Name {
		case "hidden-dangerous-command":
			dangerousCase = c
		case "spoofed-protected-approval":
			authorityCase = c
		case "verifier-mutates-source":
			mutationCase = c
		}
	}

	if dangerousCase == nil {
		t.Fatal("missing hidden-dangerous-command case")
	}
	if !dangerousCase.PermissionViolationDetected {
		t.Fatalf("expected permission_violation_detected=true for hidden-dangerous-command")
	}
	if !strings.Contains(strings.Join(dangerousCase.Errors, "\n"), "permission benchmark blocked command") {
		t.Fatalf("expected permission benchmark blocked command error, got %v", dangerousCase.Errors)
	}

	if authorityCase == nil {
		t.Fatal("missing spoofed-protected-approval case")
	}
	if !authorityCase.AuthorityViolationDetected {
		t.Fatalf("expected authority_violation_detected=true for spoofed-protected-approval")
	}
	if !strings.Contains(strings.Join(authorityCase.Errors, "\n"), "governance permission violation") {
		t.Fatalf("expected governance permission violation error, got %v", authorityCase.Errors)
	}

	if mutationCase == nil {
		t.Fatal("missing verifier-mutates-source case")
	}
	if !mutationCase.MutationGuardDetected {
		t.Fatalf("expected mutation_guard_detected=true for verifier-mutates-source")
	}
	if mutationCase.BlockingPredicate != "verifier_not_read_only" {
		t.Fatalf("expected blocking_predicate=verifier_not_read_only, got %s", mutationCase.BlockingPredicate)
	}
}

func TestBenchmarkLatencyJSONShape(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"benchmark", "--filter", "latency", "--commands", "verify", "--iterations", "1", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}

	var result struct {
		OK         bool   `json:"ok"`
		Filter     string `json:"filter"`
		Iterations int    `json:"iterations"`
		Results    []struct {
			Command    string `json:"command"`
			Iterations int    `json:"iterations"`
			OK         bool   `json:"ok"`
			MinMs      int    `json:"min_ms"`
			AvgMs      int    `json:"avg_ms"`
			MaxMs      int    `json:"max_ms"`
			ExitCodes  []int  `json:"exit_codes"`
			Samples    []struct {
				DurationMs int  `json:"duration_ms"`
				ExitCode   int  `json:"exit_code"`
				TimedOut   bool `json:"timed_out"`
			} `json:"samples"`
		} `json:"results"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got output: %s", stdout.String())
	}
	if result.Filter != "latency" || result.Iterations != 1 {
		t.Fatalf("unexpected filter/iterations: %+v", result)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(result.Results))
	}
	entry := result.Results[0]
	if entry.Command != "verify" || !entry.OK || entry.Iterations != 1 {
		t.Fatalf("unexpected latency entry: %+v", entry)
	}
	if len(entry.ExitCodes) != 1 || entry.ExitCodes[0] != ExitOK {
		t.Fatalf("expected one successful exit code, got %v", entry.ExitCodes)
	}
	if len(entry.Samples) != 1 || entry.Samples[0].TimedOut {
		t.Fatalf("expected one non-timeout sample, got %+v", entry.Samples)
	}
}
