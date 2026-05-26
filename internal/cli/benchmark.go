package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/mutationguard"
)

type benchmarkResult struct {
	OK                     bool                    `json:"ok"`
	Filter                 string                  `json:"filter"`
	GeneratedAt            string                  `json:"generated_at"`
	Integration            interface{}             `json:"integration"`
	Iterations             int                     `json:"iterations"`
	Metrics                benchmarkMetrics        `json:"metrics"`
	MutationGuardBenchmark *mutationGuardBenchmark `json:"mutation_guard_benchmark"`
	Results                []interface{}           `json:"results"`
	TimeoutMs              int                     `json:"timeout_ms"`
}

type benchmarkMetrics struct {
	AdversarialBlockRate             interface{} `json:"adversarial_block_rate"`
	AdversarialFalseAcceptCount      int         `json:"adversarial_false_accept_count"`
	AuthorityViolationDetectionRate  interface{} `json:"authority_violation_detection_rate"`
	EpisodePackagingSuccessRate      interface{} `json:"episode_packaging_success_rate"`
	ExpectedBlockCount               int         `json:"expected_block_count"`
	ExpectedPassCount                int         `json:"expected_pass_count"`
	FalseAcceptCount                 int         `json:"false_accept_count"`
	FalseRejectCount                 int         `json:"false_reject_count"`
	MutationGuardDetectionRate       interface{} `json:"mutation_guard_detection_rate"`
	PermissionViolationDetectionRate interface{} `json:"permission_violation_detection_rate"`
	PolicyValidationPassRate         interface{} `json:"policy_validation_pass_rate"`
	RuntimeMs                        int         `json:"runtime_ms"`
	SchemaValidationPassRate         interface{} `json:"schema_validation_pass_rate"`
}

type mutationGuardBenchmark struct {
	OK          bool                `json:"ok"`
	RuntimeMs   int                 `json:"runtime_ms"`
	FileCounts  []int               `json:"file_counts"`
	Concurrency []int               `json:"concurrency"`
	Cases       []mutationGuardCase `json:"cases"`
}

type mutationGuardCase struct {
	Mode        string `json:"mode"`
	FileCount   int    `json:"file_count"`
	Concurrency int    `json:"concurrency"`
	DurationMs  int    `json:"duration_ms"`
	HashedPaths int    `json:"hashed_paths"`
	OK          bool   `json:"ok"`
}

func handleBenchmark(args []string, stdout io.Writer, stderr io.Writer) int {
	filter := "all"
	mutationFiles := "100,1000,5000"
	mutationConcurrency := "1,4,16,64"
	jsonMode := false
	updateSnapshots := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--filter":
			if i+1 < len(args) {
				filter = args[i+1]
				i++
			}
		case "--mutation-files":
			if i+1 < len(args) {
				mutationFiles = args[i+1]
				i++
			}
		case "--mutation-concurrency":
			if i+1 < len(args) {
				mutationConcurrency = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		case "--update-snapshots":
			updateSnapshots = true
		}
	}

	if updateSnapshots {
		fmt.Fprintln(stderr, "--update-snapshots requires a human-approved boundary change; this command will not update snapshots automatically")
		return ExitUsage
	}

	if filter != "mutation-guard" {
		fmt.Fprintf(stderr, "benchmark filter %q is not implemented in the Go CLI; use --filter mutation-guard\n", filter)
		return ExitUsage
	}

	fileCounts, err := parsePositiveIntList(mutationFiles)
	if err != nil {
		fmt.Fprintf(stderr, "error parsing --mutation-files: %v\n", err)
		return ExitUsage
	}
	if len(fileCounts) == 0 {
		fileCounts = []int{100, 1000, 5000}
	}

	concurrencyLevels, err := parsePositiveIntList(mutationConcurrency)
	if err != nil {
		fmt.Fprintf(stderr, "error parsing --mutation-concurrency: %v\n", err)
		return ExitUsage
	}
	if len(concurrencyLevels) == 0 {
		concurrencyLevels = []int{1, 4, 16, 64}
	}

	mgReport, err := runMutationGuardBenchmark(fileCounts, concurrencyLevels)
	if err != nil {
		fmt.Fprintf(stderr, "mutation guard benchmark failed: %v\n", err)
		return ExitError
	}

	metrics := defaultBenchmarkMetrics()
	metrics.RuntimeMs = mgReport.RuntimeMs

	result := benchmarkResult{
		OK:                     mgReport.OK,
		Filter:                 filter,
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		Integration:            nil,
		Iterations:             1,
		Metrics:                metrics,
		MutationGuardBenchmark: mgReport,
		Results:                []interface{}{},
		TimeoutMs:              120000,
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		renderBenchmarkText(stdout, result)
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func defaultBenchmarkMetrics() benchmarkMetrics {
	return benchmarkMetrics{
		AdversarialBlockRate:             nil,
		AdversarialFalseAcceptCount:      0,
		AuthorityViolationDetectionRate:  nil,
		EpisodePackagingSuccessRate:      nil,
		ExpectedBlockCount:               0,
		ExpectedPassCount:                0,
		FalseAcceptCount:                 0,
		FalseRejectCount:                 0,
		MutationGuardDetectionRate:       nil,
		PermissionViolationDetectionRate: nil,
		PolicyValidationPassRate:         nil,
		RuntimeMs:                        0,
		SchemaValidationPassRate:         nil,
	}
}

func parsePositiveIntList(s string) ([]int, error) {
	var result []int
	seen := make(map[int]struct{})
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("invalid positive integer: %s", part)
		}
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			result = append(result, n)
		}
	}
	return result, nil
}

func runMutationGuardBenchmark(fileCounts []int, concurrencyLevels []int) (*mutationGuardBenchmark, error) {
	if !mutationguard.IsGitAvailable() {
		return nil, fmt.Errorf("git is not available")
	}

	started := time.Now()
	tmpRoot, err := os.MkdirTemp("", "x-harness-mutation-bench-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpRoot)

	var cases []mutationGuardCase

	for _, fileCount := range fileCounts {
		gitFixture := filepath.Join(tmpRoot, fmt.Sprintf("git-%d", fileCount))
		nonGitFixture := filepath.Join(tmpRoot, fmt.Sprintf("nongit-%d", fileCount))
		if err := os.MkdirAll(gitFixture, 0755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(nonGitFixture, 0755); err != nil {
			return nil, err
		}
		if out, err := exec.Command("git", "init", gitFixture).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git init failed: %v\n%s", err, out)
		}
		if err := writeMutationGuardFixture(gitFixture, fileCount); err != nil {
			return nil, err
		}
		if err := writeMutationGuardFixture(nonGitFixture, fileCount); err != nil {
			return nil, err
		}

		for _, concurrency := range concurrencyLevels {
			gitCase, err := measureMutationGuardSnapshot("git", gitFixture, fileCount, concurrency)
			if err != nil {
				return nil, err
			}
			cases = append(cases, gitCase)

			nonGitCase, err := measureMutationGuardSnapshot("non-git", nonGitFixture, fileCount, concurrency)
			if err != nil {
				return nil, err
			}
			cases = append(cases, nonGitCase)
		}
	}

	ok := true
	for _, c := range cases {
		if !c.OK {
			ok = false
			break
		}
	}

	return &mutationGuardBenchmark{
		OK:          ok,
		RuntimeMs:   int(time.Since(started).Milliseconds()),
		FileCounts:  fileCounts,
		Concurrency: concurrencyLevels,
		Cases:       cases,
	}, nil
}

func writeMutationGuardFixture(root string, fileCount int) error {
	for i := 0; i < fileCount; i++ {
		dir := filepath.Join(root, fmt.Sprintf("group-%d", i/100))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file-%d.txt", i)), []byte(fmt.Sprintf("mutation guard benchmark file %d\n", i)), 0644); err != nil {
			return err
		}
	}
	return nil
}

func measureMutationGuardSnapshot(mode string, fixtureRoot string, fileCount int, concurrency int) (mutationGuardCase, error) {
	prev := os.Getenv("X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY")
	os.Setenv("X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY", strconv.Itoa(concurrency))
	if prev == "" {
		defer os.Unsetenv("X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY")
	} else {
		defer os.Setenv("X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY", prev)
	}

	started := time.Now()
	var snapshot *mutationguard.Snapshot
	var err error
	if mode == "git" {
		snapshot, err = mutationguard.TakeSnapshot(fixtureRoot)
	} else {
		snapshot, err = mutationguard.TakeFallbackSnapshot(fixtureRoot)
	}
	durationMs := int(time.Since(started).Milliseconds())
	if err != nil {
		return mutationGuardCase{
			Mode:        mode,
			FileCount:   fileCount,
			Concurrency: concurrency,
			DurationMs:  durationMs,
			HashedPaths: 0,
			OK:          false,
		}, nil
	}
	hashedPaths := len(snapshot.HashMap)
	return mutationGuardCase{
		Mode:        mode,
		FileCount:   fileCount,
		Concurrency: concurrency,
		DurationMs:  durationMs,
		HashedPaths: hashedPaths,
		OK:          hashedPaths == fileCount,
	}, nil
}

func renderBenchmarkText(w io.Writer, result benchmarkResult) {
	WriteLine(w, "# x-harness Mutation Guard Benchmark")
	WriteLine(w, "")
	WriteLine(w, "- runtime_ms: %d", result.MutationGuardBenchmark.RuntimeMs)
	WriteLine(w, "| mode | files | concurrency | duration_ms | hashed_paths | ok |")
	WriteLine(w, "| :-- | --: | --: | --: | --: | :-- |")
	for _, item := range result.MutationGuardBenchmark.Cases {
		WriteLine(w, "| %s | %d | %d | %d | %d | %v |", item.Mode, item.FileCount, item.Concurrency, item.DurationMs, item.HashedPaths, item.OK)
	}
}
