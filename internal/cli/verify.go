package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/mutationguard"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

type withheldReason struct {
	FailureClass      string `json:"failure_class"`
	FailureStage      string `json:"failure_stage"`
	Recoverability    string `json:"recoverability"`
	NextAction        string `json:"next_action"`
	BlockingPredicate string `json:"blocking_predicate"`
}

// VerifyResult is the minimal output of the verify command.
type VerifyResult struct {
	OK               bool                   `json:"ok"`
	TaskID           string                 `json:"task_id"`
	Tier             string                 `json:"tier"`
	AdmissionOutcome string                 `json:"admission_outcome"`
	AcceptanceStatus string                 `json:"acceptance_status"`
	SchemaError      string                 `json:"schema_error,omitempty"`
	AdmissionErrors  []string               `json:"admission_errors,omitempty"`
	MutationGuard    *mutationguard.Result  `json:"mutation_guard,omitempty"`
	WithheldReason   *withheldReason        `json:"withheld_reason,omitempty"`
}

func handleVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := ""
	subagentPath := ""
	tier := ""
	jsonMode := false
	verbose := false
	useMutationGuard := false
	strict := false
	trace := false
	traceDir := ".x-harness/traces"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 < len(args) {
				cardPath = args[i+1]
				i++
			}
		case "--subagent-return":
			if i+1 < len(args) {
				subagentPath = args[i+1]
				i++
			}
		case "--tier":
			if i+1 < len(args) {
				tier = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		case "--verbose":
			verbose = true
		case "--mutation-guard":
			useMutationGuard = true
		case "--strict":
			strict = true
			useMutationGuard = true
		case "--trace":
			trace = true
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		}
	}

	if (cardPath == "" && subagentPath == "") || (cardPath != "" && subagentPath != "") {
		fmt.Fprintln(stderr, "usage: x-harness verify --card <path> | --subagent-return <path> [--tier <tier>] [--json] [--verbose] [--mutation-guard] [--strict] [--trace] [--trace-dir <dir>]")
		return ExitUsage
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	var doc map[string]any
	var schemaPath string
	var sourcePath string
	var schemaErr error

	if cardPath != "" {
		schemaPath = assets.NewLocator(root).Schema("completion-card.schema.json")
		sourcePath = cardPath
		if err := loader.LoadDocument(cardPath, &doc); err != nil {
			fmt.Fprintf(stderr, "error: cannot load card: %v\n", err)
			return ExitError
		}
	} else {
		schemaPath = assets.NewLocator(root).Schema("subagent-return.schema.json")
		sourcePath = subagentPath
		var subagentDoc map[string]any
		if err := loader.LoadDocument(subagentPath, &subagentDoc); err != nil {
			fmt.Fprintf(stderr, "error: cannot load subagent return: %v\n", err)
			return ExitError
		}
		v, err := schema.Compile(schemaPath)
		if err != nil {
			fmt.Fprintf(stderr, "error: cannot compile schema: %v\n", err)
			return ExitError
		}
		schemaErr = v.Validate(subagentDoc)
		if schemaErr != nil {
			result := buildVerifyResult(nil, schemaErr, nil, strict)
			renderVerifyResult(result, jsonMode, verbose, stdout, sourcePath, schemaPath)
			return ExitError
		}
		mappedDoc := map[string]any{
			"subagent_return": subagentDoc,
		}
		if tier == "" {
			tier = "standard"
		}
		mappedDoc["tier"] = tier
		for _, key := range []string{"verification", "evidence", "handoff", "done_checklist", "prediction", "pgv_advice", "state"} {
			if v, ok := subagentDoc[key]; ok {
				mappedDoc[key] = v
			}
		}
		doc = mappedDoc
	}

	var result VerifyResult

	if useMutationGuard {
		var validateFn func() error
		if cardPath != "" {
			v, err := schema.Compile(schemaPath)
			if err != nil {
				fmt.Fprintf(stderr, "error: cannot compile schema: %v\n", err)
				return ExitError
			}
			validateFn = func() error { return v.Validate(doc) }
		}
		result = runWithMutationGuard(root, strict, doc, validateFn, stderr)
	} else {
		if cardPath != "" {
			v, err := schema.Compile(schemaPath)
			if err != nil {
				fmt.Fprintf(stderr, "error: cannot compile schema: %v\n", err)
				return ExitError
			}
			schemaErr = v.Validate(doc)
		}
		result = buildVerifyResult(doc, schemaErr, nil, strict)
	}

	renderVerifyResult(result, jsonMode, verbose, stdout, sourcePath, schemaPath)

	if trace {
		event := TraceEvent{
			"event_id":             fmt.Sprintf("VE-%d", time.Now().UnixMilli()),
			"event_type":           "verify_completed",
			"task_id":              result.TaskID,
			"story_id":             nil,
			"tier":                 result.Tier,
			"claim_id":             nil,
			"evidence_id":          nil,
			"verifier":             "x-harness",
			"verifier_mode":        "read_only",
			"outcome":              result.AdmissionOutcome,
			"acceptance_status":    result.AcceptanceStatus,
			"blocking_predicate":   nil,
			"blocked_reason_class": nil,
			"next_owner":           nil,
			"next_action":          nil,
			"created_at":           time.Now().UTC().Format(time.RFC3339Nano),
			"notes":                result.AdmissionErrors,
			"errors":               []string{},
		}
		if result.WithheldReason != nil {
			event["blocking_predicate"] = result.WithheldReason.BlockingPredicate
			event["blocked_reason_class"] = result.WithheldReason.FailureClass
		}
		if result.SchemaError != "" {
			event["errors"] = append(event["errors"].([]string), result.SchemaError)
		}
		_, err := AppendTrace(event, traceDir)
		if err != nil {
			fmt.Fprintf(stderr, "failed to append trace: %v\n", err)
			return ExitError
		}
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func injectTestMutation(root string, stderr io.Writer) {
	if os.Getenv("X_HARNESS_ENABLE_TEST_HOOKS") != "1" {
		return
	}
	injectPath := os.Getenv("X_HARNESS_TEST_INJECT_MUTATION")
	if injectPath == "" {
		return
	}

	var resolved string
	if filepath.IsAbs(injectPath) {
		resolved = injectPath
	} else {
		resolved = filepath.Join(root, injectPath)
	}

	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		fmt.Fprintf(stderr, "test hook: failed to resolve injection path: %v\n", err)
		return
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "test hook: failed to resolve root: %v\n", err)
		return
	}

	rootPrefix := absRoot + string(filepath.Separator)
	if absResolved != absRoot && !strings.HasPrefix(absResolved, rootPrefix) {
		fmt.Fprintf(stderr, "test hook: rejected injection path %s outside root (%s)\n", absResolved, absRoot)
		return
	}

	if err := os.WriteFile(absResolved, []byte("test-mutation"), 0644); err != nil {
		fmt.Fprintf(stderr, "test hook: failed to write injection file: %v\n", err)
	}
}

func runWithMutationGuard(root string, strict bool, doc map[string]any, validateFn func() error, stderr io.Writer) VerifyResult {
	var useGit bool
	var gitRoot string
	var gitErr error

	if mutationguard.IsGitAvailable() {
		gitRoot, gitErr = mutationguard.FindGitRoot(root)
		if gitErr == nil {
			useGit = true
		}
	}

	var schemaErr error
	var mgResult *mutationguard.Result
	var guardErr error

	if useGit {
		mgResult, guardErr = mutationguard.Guard(gitRoot, func() error {
			injectTestMutation(gitRoot, stderr)
			if validateFn != nil {
				schemaErr = validateFn()
			}
			return nil
		})
	} else {
		mgResult, guardErr = mutationguard.GuardFallback(root, func() error {
			injectTestMutation(root, stderr)
			if validateFn != nil {
				schemaErr = validateFn()
			}
			return nil
		})
		if guardErr != nil && gitErr != nil {
			guardErr = fmt.Errorf("git: %v; fallback: %v", gitErr, guardErr)
		}
	}

	if guardErr != nil {
		mg := &mutationguard.Result{Enabled: true, SkippedReason: guardErr.Error(), Violated: strict}
		result := buildVerifyResult(doc, schemaErr, mg, strict)
		if strict {
			fmt.Fprintf(stderr, "mutation_guard_error: guard failed in strict mode: %v\n", guardErr)
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
		}
		return result
	}

	result := buildVerifyResult(doc, schemaErr, mgResult, strict)
	if mgResult != nil && mgResult.Violated {
		result.OK = false
		result.AdmissionOutcome = "blocked"
		result.AcceptanceStatus = "withheld"
	}
	return result
}

func buildVerifyResult(doc map[string]any, schemaErr error, mgResult *mutationguard.Result, strict bool) VerifyResult {
	result := VerifyResult{
		TaskID: stringValue(doc, "task_id"),
		Tier:   stringValue(doc, "tier"),
	}

	if schemaErr != nil {
		result.SchemaError = schemaErr.Error()
		result.AdmissionOutcome = "failed"
		result.AcceptanceStatus = "withheld"
		result.OK = false
		result.MutationGuard = mgResult
		result.WithheldReason = &withheldReason{
			FailureClass:      "schema_invalid",
			FailureStage:      "verify_pipeline",
			Recoverability:    "retry_with_fixes",
			NextAction:        "review_and_resubmit",
			BlockingPredicate: "schema_invalid",
		}
		return result
	}

	admResult := admission.Run(doc, strict)
	result.AdmissionOutcome = admResult.Outcome
	result.AcceptanceStatus = admResult.AcceptanceStatus
	result.AdmissionErrors = admResult.Errors
	result.OK = admResult.Outcome == "success" && admResult.AcceptanceStatus == "accepted" && len(admResult.Errors) == 0
	result.MutationGuard = mgResult

	if admResult.WithheldReason != nil {
		result.WithheldReason = &withheldReason{
			FailureClass:      admResult.WithheldReason.FailureClass,
			FailureStage:      admResult.WithheldReason.FailureStage,
			Recoverability:    admResult.WithheldReason.Recoverability,
			NextAction:        admResult.WithheldReason.NextAction,
			BlockingPredicate: admResult.BlockingPredicate,
		}
	}

	if mgResult != nil && mgResult.Violated {
		result.OK = false
		result.AdmissionOutcome = "blocked"
		result.AcceptanceStatus = "withheld"
		result.WithheldReason = &withheldReason{
			FailureClass:      "mutation_detected",
			FailureStage:      "verify_pipeline",
			Recoverability:    "manual_review",
			NextAction:        "review_and_resubmit",
			BlockingPredicate: "verifier_not_read_only",
		}
	}

	return result
}

func renderVerifyResult(result VerifyResult, jsonMode, verbose bool, stdout io.Writer, sourcePath, schemaPath string) {
	if jsonMode {
		WriteJSON(stdout, result)
		return
	}
	if verbose {
		WriteLine(stdout, "source: %s", sourcePath)
		WriteLine(stdout, "schema: %s", schemaPath)
	}
	WriteLine(stdout, "task_id: %s", result.TaskID)
	WriteLine(stdout, "tier: %s", result.Tier)
	WriteLine(stdout, "outcome: %s", result.AdmissionOutcome)
	WriteLine(stdout, "acceptance_status: %s", result.AcceptanceStatus)
	if result.SchemaError != "" {
		WriteLine(stdout, "schema_error: %s", result.SchemaError)
	}
	for _, e := range result.AdmissionErrors {
		WriteLine(stdout, "admission_error: %s", e)
	}
	if result.WithheldReason != nil {
		WriteLine(stdout, "withheld_reason:")
		WriteLine(stdout, "  failure_class: %s", result.WithheldReason.FailureClass)
		WriteLine(stdout, "  failure_stage: %s", result.WithheldReason.FailureStage)
		WriteLine(stdout, "  recoverability: %s", result.WithheldReason.Recoverability)
		WriteLine(stdout, "  next_action: %s", result.WithheldReason.NextAction)
		WriteLine(stdout, "  blocking_predicate: %s", result.WithheldReason.BlockingPredicate)
	}
	if result.MutationGuard != nil {
		if result.MutationGuard.SkippedReason != "" {
			WriteLine(stdout, "mutation_guard: skipped (%s)", result.MutationGuard.SkippedReason)
		} else if result.MutationGuard.Violated {
			WriteLine(stdout, "mutation_guard: violated")
			for _, d := range result.MutationGuard.UnexpectedDeltas {
				WriteLine(stdout, "  - %s", d.Path)
			}
		} else {
			WriteLine(stdout, "mutation_guard: clean")
		}
	}
}

func stringValue(doc map[string]any, key string) string {
	if doc == nil {
		return ""
	}
	if v, ok := doc[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func mapValue(doc map[string]any, key string) map[string]any {
	if doc == nil {
		return nil
	}
	if v, ok := doc[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}
