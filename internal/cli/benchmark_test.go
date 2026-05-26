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
