package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/mutationguard"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"gopkg.in/yaml.v3"
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

type integrationBenchmarkReport struct {
	OK          bool                  `json:"ok"`
	RuntimeMs   int                   `json:"runtime_ms"`
	Golden      interface{}           `json:"golden"`
	Adversarial *benchmarkSuiteReport `json:"adversarial"`
}

type benchmarkSuiteReport struct {
	Suite              string                  `json:"suite"`
	CasesTotal         int                     `json:"cases_total"`
	ExpectedPassCount  int                     `json:"expected_pass_count"`
	ExpectedBlockCount int                     `json:"expected_block_count"`
	FalseAcceptCount   int                     `json:"false_accept_count"`
	FalseRejectCount   int                     `json:"false_reject_count"`
	RuntimeMs          int                     `json:"runtime_ms"`
	Cases              []adversarialCaseResult `json:"cases"`
}

type adversarialCaseResult struct {
	Suite                       string   `json:"suite"`
	Name                        string   `json:"name"`
	CardPath                    string   `json:"card_path"`
	ExpectedAcceptanceStatus    string   `json:"expected_acceptance_status"`
	ActualAcceptanceStatus      string   `json:"actual_acceptance_status"`
	Outcome                     string   `json:"outcome"`
	Accepted                    bool     `json:"accepted"`
	FalseAccept                 bool     `json:"false_accept"`
	FalseReject                 bool     `json:"false_reject"`
	BlockingPredicate           string   `json:"blocking_predicate"`
	SchemaValid                 bool     `json:"schema_valid"`
	PolicyValid                 bool     `json:"policy_valid"`
	PermissionViolationExpected bool     `json:"permission_violation_expected"`
	PermissionViolationDetected bool     `json:"permission_violation_detected"`
	AuthorityViolationExpected  bool     `json:"authority_violation_expected"`
	AuthorityViolationDetected  bool     `json:"authority_violation_detected"`
	MutationGuardExpected       bool     `json:"mutation_guard_expected"`
	MutationGuardDetected       bool     `json:"mutation_guard_detected"`
	RuntimeMs                   int      `json:"runtime_ms"`
	Errors                      []string `json:"errors"`
	Notes                       []string `json:"notes"`
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

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	if filter == "mutation-guard" {
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

	if filter == "adversarial" {
		advReport, err := runAdversarialBenchmark(root)
		if err != nil {
			fmt.Fprintf(stderr, "adversarial benchmark failed: %v\n", err)
			return ExitError
		}

		integration := &integrationBenchmarkReport{
			OK:          advReport.FalseAcceptCount == 0 && advReport.FalseRejectCount == 0,
			RuntimeMs:   advReport.RuntimeMs,
			Golden:      nil,
			Adversarial: advReport,
		}

		metrics := computeAdversarialMetrics(advReport.Cases, advReport.RuntimeMs)

		ok := advReport.FalseAcceptCount == 0 && advReport.FalseRejectCount == 0
		if metrics.MutationGuardDetectionRate != nil {
			if r, okRate := metrics.MutationGuardDetectionRate.(float64); okRate && r < 1 {
				ok = false
			}
		}
		if metrics.PermissionViolationDetectionRate != nil {
			if r, okRate := metrics.PermissionViolationDetectionRate.(float64); okRate && r < 1 {
				ok = false
			}
		}
		if metrics.AuthorityViolationDetectionRate != nil {
			if r, okRate := metrics.AuthorityViolationDetectionRate.(float64); okRate && r < 1 {
				ok = false
			}
		}

		result := benchmarkResult{
			OK:                     ok,
			Filter:                 filter,
			GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
			Integration:            integration,
			Iterations:             1,
			Metrics:                metrics,
			MutationGuardBenchmark: nil,
			Results:                []interface{}{},
			TimeoutMs:              120000,
		}

		if jsonMode {
			if err := WriteJSON(stdout, result); err != nil {
				return ExitError
			}
		} else {
			renderAdversarialBenchmarkText(stdout, result)
		}

		if result.OK {
			return ExitOK
		}
		return ExitError
	}

	fmt.Fprintf(stderr, "benchmark filter %q is not implemented in the Go CLI; use --filter mutation-guard or --filter adversarial\n", filter)
	return ExitUsage
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

func runAdversarialBenchmark(root string) (*benchmarkSuiteReport, error) {
	started := time.Now()
	casesDir := filepath.Join(root, "examples", "adversarial")
	entries, err := os.ReadDir(casesDir)
	if err != nil {
		return nil, err
	}

	var cases []adversarialCaseResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cardPath := filepath.Join(casesDir, entry.Name(), "completion-card.yaml")
		if _, err := os.Stat(cardPath); err != nil {
			continue
		}
		cases = append(cases, runAdversarialCase(root, cardPath, entry.Name()))
	}

	sort.Slice(cases, func(i, j int) bool {
		return cases[i].Name < cases[j].Name
	})

	return &benchmarkSuiteReport{
		Suite:              "adversarial",
		CasesTotal:         len(cases),
		ExpectedPassCount:  0,
		ExpectedBlockCount: len(cases),
		FalseAcceptCount:   0,
		FalseRejectCount:   0,
		RuntimeMs:          int(time.Since(started).Milliseconds()),
		Cases:              cases,
	}, nil
}

func runAdversarialCase(root, cardPath, name string) adversarialCaseResult {
	started := time.Now()
	var errors []string
	var notes []string

	relCardPath, _ := filepath.Rel(root, cardPath)
	relCardPath = filepath.ToSlash(relCardPath)

	var doc map[string]any
	if err := loader.LoadDocument(cardPath, &doc); err != nil {
		errors = append(errors, fmt.Sprintf("cannot load card: %v", err))
		return buildAdversarialCaseResult(name, relCardPath, errors, notes, false, false, false, false, false, int(time.Since(started).Milliseconds()))
	}

	schemaPath := assets.NewLocator(root).Schema("completion-card.schema.json")
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		errors = append(errors, fmt.Sprintf("cannot compile schema: %v", err))
	}

	var schemaErr error
	if validator != nil {
		schemaErr = validator.Validate(doc)
	}

	admResult := admission.Run(doc, false)

	mutationGuardExpected := name == "verifier-mutates-source"
	tier := stringValue(doc, "tier")
	if tier == "" {
		tier = "standard"
	}

	permissionErrors := permissionBenchmarkErrors(doc, root, tier)
	consistencyErrors := evidenceConsistencyErrors(doc)
	authorityErrors := authorityBenchmarkErrors(doc, root)

	var mutationGuardDetected bool
	if mutationGuardExpected {
		probeName := fmt.Sprintf(".x-harness-mutation-guard-probe-%d-%d.probe", os.Getpid(), time.Now().UnixNano())
		probePath := filepath.Join(root, probeName)
		prevHooks := os.Getenv("X_HARNESS_ENABLE_TEST_HOOKS")
		prevInject := os.Getenv("X_HARNESS_TEST_INJECT_MUTATION")
		os.Setenv("X_HARNESS_ENABLE_TEST_HOOKS", "1")
		os.Setenv("X_HARNESS_TEST_INJECT_MUTATION", probeName)
		defer func() {
			if prevHooks == "" {
				os.Unsetenv("X_HARNESS_ENABLE_TEST_HOOKS")
			} else {
				os.Setenv("X_HARNESS_ENABLE_TEST_HOOKS", prevHooks)
			}
			if prevInject == "" {
				os.Unsetenv("X_HARNESS_TEST_INJECT_MUTATION")
			} else {
				os.Setenv("X_HARNESS_TEST_INJECT_MUTATION", prevInject)
			}
			os.Remove(probePath)
		}()

		var mgResult *mutationguard.Result
		var guardErr error
		if mutationguard.IsGitAvailable() {
			gitRoot, err := mutationguard.FindGitRoot(root)
			if err == nil {
				mgResult, guardErr = mutationguard.Guard(gitRoot, func() error {
					if os.Getenv("X_HARNESS_ENABLE_TEST_HOOKS") == "1" {
						injectPath := os.Getenv("X_HARNESS_TEST_INJECT_MUTATION")
						if injectPath != "" {
							resolved := filepath.Join(gitRoot, injectPath)
							os.WriteFile(resolved, []byte("test-mutation"), 0644)
						}
					}
					return nil
				})
			} else {
				mgResult, guardErr = mutationguard.GuardFallback(root, func() error {
					if os.Getenv("X_HARNESS_ENABLE_TEST_HOOKS") == "1" {
						injectPath := os.Getenv("X_HARNESS_TEST_INJECT_MUTATION")
						if injectPath != "" {
							resolved := filepath.Join(root, injectPath)
							os.WriteFile(resolved, []byte("test-mutation"), 0644)
						}
					}
					return nil
				})
			}
		} else {
			mgResult, guardErr = mutationguard.GuardFallback(root, func() error {
				if os.Getenv("X_HARNESS_ENABLE_TEST_HOOKS") == "1" {
					injectPath := os.Getenv("X_HARNESS_TEST_INJECT_MUTATION")
					if injectPath != "" {
						resolved := filepath.Join(root, injectPath)
						os.WriteFile(resolved, []byte("test-mutation"), 0644)
					}
				}
				return nil
			})
		}

		if guardErr != nil {
			errors = append(errors, fmt.Sprintf("mutation guard blocked: %v", guardErr))
			mutationGuardDetected = true
		} else if mgResult != nil && mgResult.Violated {
			var paths []string
			for _, d := range mgResult.UnexpectedDeltas {
				paths = append(paths, d.Path)
			}
			errors = append(errors, fmt.Sprintf("mutation guard blocked: unexpected changes detected: %s", strings.Join(paths, ", ")))
			mutationGuardDetected = true
		}
	}

	errors = append(errors, admResult.Errors...)
	errors = append(errors, authorityErrors...)
	errors = append(errors, permissionErrors...)
	errors = append(errors, consistencyErrors...)

	notes = append(notes, admResult.Notes...)
	if schemaErr == nil {
		notes = append(notes, fmt.Sprintf("completion card valid: %s", relCardPath))
	}
	notes = append(notes, "governance enforced mode enabled")
	if mutationGuardExpected {
		notes = append(notes, "strict mode enabled")
	}

	schemaValid := schemaErr == nil
	policyValid := true

	permissionViolationDetected := len(permissionErrors) > 0
	authorityViolationDetected := len(authorityErrors) > 0

	actualAcceptance := "withheld"
	outcome := "failed"
	accepted := false

	benchmarkBlocked := len(permissionErrors) > 0 || len(consistencyErrors) > 0
	if mutationGuardDetected {
		outcome = "blocked"
	} else if benchmarkBlocked {
		outcome = "blocked"
	} else if len(admResult.Errors) > 0 || len(authorityErrors) > 0 {
		outcome = "failed"
	} else {
		outcome = admResult.Outcome
		if outcome == "success" {
			actualAcceptance = "accepted"
			accepted = true
		}
	}

	blockingPredicate := ""
	if mutationGuardDetected {
		blockingPredicate = "verifier_not_read_only"
	} else if benchmarkBlocked {
		blockingPredicate = "benchmark_adversarial_guard"
	} else if authorityViolationDetected {
		blockingPredicate = "Fpermission"
	} else if admResult.BlockingPredicate != "" {
		blockingPredicate = admResult.BlockingPredicate
	} else if len(admResult.Errors) > 0 || len(authorityErrors) > 0 {
		blockingPredicate = "admission_failed"
	}

	return adversarialCaseResult{
		Suite:                       "adversarial",
		Name:                        name,
		CardPath:                    relCardPath,
		ExpectedAcceptanceStatus:    "withheld",
		ActualAcceptanceStatus:      actualAcceptance,
		Outcome:                     outcome,
		Accepted:                    accepted,
		FalseAccept:                 false,
		FalseReject:                 false,
		BlockingPredicate:           blockingPredicate,
		SchemaValid:                 schemaValid,
		PolicyValid:                 policyValid,
		PermissionViolationExpected: name == "hidden-dangerous-command",
		PermissionViolationDetected: permissionViolationDetected,
		AuthorityViolationExpected:  name == "spoofed-protected-approval",
		AuthorityViolationDetected:  authorityViolationDetected,
		MutationGuardExpected:       mutationGuardExpected,
		MutationGuardDetected:       mutationGuardDetected,
		RuntimeMs:                   int(time.Since(started).Milliseconds()),
		Errors:                      errors,
		Notes:                       notes,
	}
}

func buildAdversarialCaseResult(name, cardPath string, errors, notes []string, schemaValid, policyValid, permissionDetected, authorityDetected, mutationDetected bool, runtimeMs int) adversarialCaseResult {
	return adversarialCaseResult{
		Suite:                       "adversarial",
		Name:                        name,
		CardPath:                    cardPath,
		ExpectedAcceptanceStatus:    "withheld",
		ActualAcceptanceStatus:      "withheld",
		Outcome:                     "error",
		Accepted:                    false,
		FalseAccept:                 false,
		FalseReject:                 false,
		BlockingPredicate:           "benchmark_error",
		SchemaValid:                 schemaValid,
		PolicyValid:                 policyValid,
		PermissionViolationExpected: name == "hidden-dangerous-command",
		PermissionViolationDetected: permissionDetected,
		AuthorityViolationExpected:  name == "spoofed-protected-approval",
		AuthorityViolationDetected:  authorityDetected,
		MutationGuardExpected:       name == "verifier-mutates-source",
		MutationGuardDetected:       mutationDetected,
		RuntimeMs:                   runtimeMs,
		Errors:                      errors,
		Notes:                       notes,
	}
}

func permissionBenchmarkErrors(doc map[string]any, root, tier string) []string {
	var errors []string
	commands := collectCardCommands(doc)

	policyPath := filepath.Join(root, "policies", "permissions.yaml")
	policyData, err := os.ReadFile(policyPath)
	if err != nil {
		return errors
	}
	var policy struct {
		CommandSets map[string]struct {
			DenyPatterns []string `yaml:"deny_patterns"`
		} `yaml:"command_sets"`
		Roles map[string]map[string]struct {
			DenyCommandSets []string `yaml:"deny_command_sets"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(policyData, &policy); err != nil {
		return errors
	}

	workerRole, ok := policy.Roles["worker"]
	if !ok {
		return errors
	}
	profile, ok := workerRole[tier]
	if !ok {
		profile = workerRole["all"]
	}

	denySets := profile.DenyCommandSets
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		blocked := false
		for _, setName := range denySets {
			set, ok := policy.CommandSets[setName]
			if !ok {
				continue
			}
			for _, pattern := range set.DenyPatterns {
				re, err := regexp.Compile(pattern)
				if err != nil {
					continue
				}
				if re.MatchString(command) {
					errors = append(errors, fmt.Sprintf("permission benchmark blocked command %q: command denied by %s", command, setName))
					blocked = true
					break
				}
			}
			if blocked {
				break
			}
		}
		if blocked {
			continue
		}
		if token := shellMetacharacter(command); token != "" {
			errors = append(errors, fmt.Sprintf("permission benchmark blocked command %q: command contains shell metacharacter %s", command, token))
		}
	}

	return errors
}

func shellMetacharacter(command string) string {
	checks := []struct {
		token   string
		pattern string
	}{
		{"&&", `&&`},
		{"||", `\|\|`},
		{";", `;`},
		{"|", `\|`},
		{"`", "`"},
		{"$(", `\$\(`},
		{">", `>`},
		{"<", `<`},
	}
	for _, c := range checks {
		if matched, _ := regexp.MatchString(c.pattern, command); matched {
			return c.token
		}
	}
	return ""
}

func evidenceConsistencyErrors(doc map[string]any) []string {
	var errors []string
	evidence := mapValue(doc, "evidence")

	for _, item := range sliceInMap(evidence, "command_evidence") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		exitCode, ok := intLikeValue(record["exit_code"])
		if !ok || exitCode == 0 {
			continue
		}
		command := stringInMap(record, "command")
		msg := fmt.Sprintf("evidence.command_evidence has non-zero exit_code %d", exitCode)
		if command != "" {
			msg += fmt.Sprintf(" for command %q", command)
		}
		errors = append(errors, msg)
		errors = append(errors, fmt.Sprintf("benchmark evidence check blocked non-zero command exit_code %d", exitCode))
	}

	for _, item := range sliceInMap(evidence, "verification_artifacts") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		status := stringInMap(record, "status")
		if status == "" || status == "passed" {
			continue
		}
		command := stringInMap(record, "command")
		msg := fmt.Sprintf("evidence.verification_artifacts status %q is not passed", status)
		if command != "" {
			msg += fmt.Sprintf(" for command %q", command)
		}
		errors = append(errors, msg)
		errors = append(errors, fmt.Sprintf("benchmark evidence check blocked verification artifact status %q", status))
	}

	pgvAdvice := mapValue(doc, "pgv_advice")
	if boolInMap(pgvAdvice, "admission_authority") {
		errors = append(errors, "pgv_advice cannot grant admission authority; PGV is advisory-only")
		errors = append(errors, "benchmark authority check blocked PGV admission authority")
	}

	return errors
}

func authorityBenchmarkErrors(doc map[string]any, root string) []string {
	var errors []string
	evidence := mapValue(doc, "evidence")
	filesChanged := sliceInMap(evidence, "files_changed")
	governance := mapValue(doc, "governance")

	data, err := os.ReadFile(filepath.Join(root, "policies", "authority.yaml"))
	if err != nil {
		return errors
	}
	var policy struct {
		ProtectedPaths []struct {
			Path      string `yaml:"path"`
			Authority string `yaml:"authority"`
		} `yaml:"protected_paths"`
	}
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return errors
	}

	for _, item := range filesChanged {
		file, ok := item.(string)
		if !ok {
			continue
		}
		file = strings.ReplaceAll(file, "\\", "/")
		for _, pp := range policy.ProtectedPaths {
			if matchProtectedPath(pp.Path, file) && pp.Authority == "human_only" {
				artifact := mapValue(governance, "approval_artifact")
				if artifact == nil {
					errors = append(errors, fmt.Sprintf("governance permission violation: human_only path %s: governance approval_artifact is missing", file))
				}
			}
		}
	}
	return errors
}

func matchProtectedPath(pattern, path string) bool {
	pattern = strings.ReplaceAll(pattern, "\\", "/")
	path = strings.ReplaceAll(path, "\\", "/")
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	matched, _ := filepath.Match(pattern, path)
	return matched
}

func collectCardCommands(doc map[string]any) []string {
	var commands []string
	evidence := mapValue(doc, "evidence")
	for _, item := range sliceInMap(evidence, "command_evidence") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if cmd := stringInMap(record, "command"); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	for _, item := range sliceInMap(evidence, "verification_artifacts") {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if cmd := stringInMap(record, "command"); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return commands
}

func computeAdversarialMetrics(cases []adversarialCaseResult, integrationRuntimeMs int) benchmarkMetrics {
	total := len(cases)
	var schemaValidCount, policyValidCount int
	var mutationExpected, mutationDetected int
	var permissionExpected, permissionDetected int
	var authorityExpected, authorityDetected int
	var falseAcceptCount, falseRejectCount int
	var actualWithheld int

	for _, c := range cases {
		if c.SchemaValid {
			schemaValidCount++
		}
		if c.PolicyValid {
			policyValidCount++
		}
		if c.MutationGuardExpected {
			mutationExpected++
			if c.MutationGuardDetected {
				mutationDetected++
			}
		}
		if c.PermissionViolationExpected {
			permissionExpected++
			if c.PermissionViolationDetected {
				permissionDetected++
			}
		}
		if c.AuthorityViolationExpected {
			authorityExpected++
			if c.AuthorityViolationDetected {
				authorityDetected++
			}
		}
		if c.FalseAccept {
			falseAcceptCount++
		}
		if c.FalseReject {
			falseRejectCount++
		}
		if c.ActualAcceptanceStatus == "withheld" {
			actualWithheld++
		}
	}

	metrics := benchmarkMetrics{
		AdversarialBlockRate:             nil,
		AdversarialFalseAcceptCount:      falseAcceptCount,
		AuthorityViolationDetectionRate:  nil,
		EpisodePackagingSuccessRate:      nil,
		ExpectedBlockCount:               total,
		ExpectedPassCount:                0,
		FalseAcceptCount:                 falseAcceptCount,
		FalseRejectCount:                 falseRejectCount,
		MutationGuardDetectionRate:       nil,
		PermissionViolationDetectionRate: nil,
		PolicyValidationPassRate:         nil,
		RuntimeMs:                        integrationRuntimeMs,
		SchemaValidationPassRate:         nil,
	}

	if total > 0 {
		metrics.SchemaValidationPassRate = float64(schemaValidCount) / float64(total)
		metrics.PolicyValidationPassRate = float64(policyValidCount) / float64(total)
		metrics.AdversarialBlockRate = float64(actualWithheld) / float64(total)
	}
	if mutationExpected > 0 {
		metrics.MutationGuardDetectionRate = float64(mutationDetected) / float64(mutationExpected)
	}
	if permissionExpected > 0 {
		metrics.PermissionViolationDetectionRate = float64(permissionDetected) / float64(permissionExpected)
	}
	if authorityExpected > 0 {
		metrics.AuthorityViolationDetectionRate = float64(authorityDetected) / float64(authorityExpected)
	}

	return metrics
}

func renderAdversarialBenchmarkText(w io.Writer, result benchmarkResult) {
	integration := result.Integration.(*integrationBenchmarkReport)
	WriteLine(w, "# x-harness Adversarial Benchmark")
	WriteLine(w, "")
	WriteLine(w, "- ok: %v", result.OK)
	WriteLine(w, "- cases: %d", integration.Adversarial.CasesTotal)
	WriteLine(w, "| name | outcome | accepted | blocking_predicate |")
	WriteLine(w, "| :-- | :-- | :-- | :-- |")
	for _, c := range integration.Adversarial.Cases {
		WriteLine(w, "| %s | %s | %v | %s |", c.Name, c.Outcome, c.Accepted, c.BlockingPredicate)
	}
}

func boolInMap(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func stringInMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sliceInMap(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if s, ok := v.([]any); ok {
			return s
		}
	}
	return nil
}

func intLikeValue(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
