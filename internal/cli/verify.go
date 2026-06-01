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
	"github.com/BrianNguyen29/x-harness/internal/contract"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/mutationguard"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"github.com/BrianNguyen29/x-harness/internal/worktree"
)

// withheldReason is a compatibility superset that includes both schema-like fields
// (class, stage, owner, schema_recoverability) and legacy fields (failure_class, failure_stage, recoverability)
// for backward compatibility. The class/stage/owner fields are derived from failure_class/failure_stage
// and blocking_predicate respectively. schema_recoverability is the schema enum value derived from
// the legacy recoverability field.
type withheldReason struct {
	// Schema-like fields (as per withheld-reason.schema.json)
	Class                string `json:"class"`
	Stage                string `json:"stage"`
	Owner                string `json:"owner"`
	Recoverability       string `json:"recoverability"`
	SchemaRecoverability string `json:"schema_recoverability"`
	NextAction           string `json:"next_action"`
	BlockingPredicate    string `json:"blocking_predicate"`
	// Legacy fields for backward compatibility (omitted in strict mode)
	FailureClass string `json:"failure_class,omitempty"`
	FailureStage string `json:"failure_stage,omitempty"`
}

// schemaRecoverabilityFromLegacy maps legacy recoverability values to schema enum values.
// Legacy values: retry_after_refresh, retry_with_fixes, human_intervention, manual_review, or empty/unknown.
// Schema enum: automatic, manual, blocked, unknown.
func schemaRecoverabilityFromLegacy(legacy string) string {
	switch legacy {
	case "retry_after_refresh":
		return "automatic"
	case "retry_with_fixes", "human_intervention", "manual_review":
		return "manual"
	case "blocked":
		return "blocked"
	default:
		return "unknown"
	}
}

// ownerFromBlockingPredicate derives owner based on blocking_predicate value.
func ownerFromBlockingPredicate(predicate string) string {
	switch predicate {
	case "context_floor_blocked", "admission_failed":
		return "implementation-worker"
	case "verifier_not_read_only":
		return "admission-verifier"
	case "approval_missing", "permission_denied", "Fpermission":
		return "user"
	default:
		// Also map schema_invalid and schema_or_policy_invalid style predicates
		if strings.Contains(predicate, "schema_invalid") || strings.Contains(predicate, "schema_or_policy_invalid") {
			return "implementation-worker"
		}
		return "implementation-worker"
	}
}

// classFromFailureClass maps legacy failure_class to schema class enum value.
// This is a best-effort mapping for the compatibility superset.
func classFromFailureClass(failureClass string) string {
	switch failureClass {
	case "context_missing":
		return "context_floor_blocked"
	case "schema_invalid":
		return "schema_or_policy_invalid"
	case "mutation_detected":
		return "verifier_not_read_only"
	case "evidence_missing", "evidence_floor_missing":
		return "evidence_floor_missing"
	case "prediction_missing":
		return "prediction_missing"
	case "done_checklist_missing":
		return "done_checklist_missing"
	case "admission_mapping_invalid":
		return "admission_mapping_invalid"
	default:
		return failureClass
	}
}

// stageFromFailureStage maps legacy failure_stage to schema stage enum value.
// This is a best-effort mapping for the compatibility superset.
func stageFromFailureStage(failureStage string) string {
	switch failureStage {
	case "context_floor":
		return "context"
	case "verify_pipeline":
		return "verification"
	default:
		return failureStage
	}
}

// VerifyResult is the minimal output of the verify command.
type VerifyResult struct {
	OK               bool                  `json:"ok"`
	TaskID           string                `json:"task_id"`
	Tier             string                `json:"tier"`
	AdmissionOutcome string                `json:"admission_outcome"`
	AcceptanceStatus string                `json:"acceptance_status"`
	SchemaError      string                `json:"schema_error,omitempty"`
	AdmissionErrors  []string              `json:"admission_errors,omitempty"`
	MutationGuard    *mutationguard.Result `json:"mutation_guard,omitempty"`
	WithheldReason   *withheldReason       `json:"withheld_reason,omitempty"`
}

func handleVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := ""
	subagentPath := ""
	tier := ""
	jsonMode := false
	verbose := false
	useMutationGuard := false
	strict := false
	strictWithheldReason := false
	trace := false
	traceDir := ".x-harness/traces"
	worktreeAware := false
	contextFloor := false
	contractOracles := false
	contractOraclesPolicy := ""

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
		case "--strict-withheld-reason":
			strictWithheldReason = true
		case "--trace":
			trace = true
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--worktree-aware":
			worktreeAware = true
		case "--context-floor":
			contextFloor = true
		case "--contract-oracles":
			contractOracles = true
		case "--contract-oracles-policy":
			if i+1 < len(args) {
				contractOraclesPolicy = args[i+1]
				i++
			}
		}
	}

	if (cardPath == "" && subagentPath == "") || (cardPath != "" && subagentPath != "") {
		fmt.Fprintln(stderr, "usage: x-harness verify --card <path> | --subagent-return <path> [--tier <tier>] [--json] [--verbose] [--mutation-guard] [--strict] [--strict-withheld-reason] [--trace] [--trace-dir <dir>] [--worktree-aware] [--context-floor] [--contract-oracles] [--contract-oracles-policy <path>]")
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
			result := buildVerifyResult(nil, schemaErr, nil, strict, false, "")
			renderVerifyResult(result, jsonMode, verbose, strictWithheldReason, stdout, sourcePath, schemaPath)
			return ExitError
		}
		mappedDoc := map[string]any{
			"subagent_return": subagentDoc,
		}
		if tier == "" {
			tier = "standard"
		}
		mappedDoc["tier"] = tier
		for _, key := range []string{"verification", "evidence", "handoff", "done_checklist", "prediction", "pgv_advice", "state", "context_alignment"} {
			if v, ok := subagentDoc[key]; ok {
				mappedDoc[key] = v
			}
		}
		doc = mappedDoc
	}

	// Auto-enable mutation guard for standard and deep tiers
	effectiveTier := stringValue(doc, "tier")
	if effectiveTier == "" {
		effectiveTier = tier
	}
	if effectiveTier == "" {
		effectiveTier = "standard"
	}
	if !useMutationGuard && (effectiveTier == "standard" || effectiveTier == "deep") {
		useMutationGuard = true
	}

	// Auto-enable context floor for standard and deep tiers
	if !contextFloor && (effectiveTier == "standard" || effectiveTier == "deep") {
		contextFloor = true
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
		result = runWithMutationGuard(root, strict, doc, validateFn, stderr, contextFloor, cardPath)
	} else {
		if cardPath != "" {
			v, err := schema.Compile(schemaPath)
			if err != nil {
				fmt.Fprintf(stderr, "error: cannot compile schema: %v\n", err)
				return ExitError
			}
			schemaErr = v.Validate(doc)
		}
		result = buildVerifyResult(doc, schemaErr, nil, strict, contextFloor, cardPath)
	}

	// Contract Oracle check (opt-in via --contract-oracles flag)
	if contractOracles && result.OK {
		policyPath := contractOraclesPolicy
		if policyPath == "" {
			policyPath = filepath.Join(root, "policies", "contract-oracle.yaml")
		}
		contractResult, contractErr := contract.Check(policyPath, []string{root})
		if contractErr != nil {
			fmt.Fprintf(stderr, "error: contract oracle check failed: %v\n", contractErr)
			return ExitError
		}
		if !contractResult.OK {
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
			// Build compact violation summary (first few violations)
			var violationSummaries []string
			maxViolations := 3
			if len(contractResult.Violations) < maxViolations {
				maxViolations = len(contractResult.Violations)
			}
			for i := 0; i < maxViolations; i++ {
				v := contractResult.Violations[i]
				relPath, _ := filepath.Rel(root, v.File)
				violationSummaries = append(violationSummaries, fmt.Sprintf("%s:%d: %s (%s)", relPath, v.Line, v.Message, v.RuleID))
			}
			if len(contractResult.Violations) > maxViolations {
				violationSummaries = append(violationSummaries, fmt.Sprintf("... and %d more violations", len(contractResult.Violations)-maxViolations))
			}
			predicate := "contract_oracle_blocked"
			recoverability := "retry_with_fixes"
			result.WithheldReason = &withheldReason{
				Class:                "contract_mismatch",
				Stage:                "verification",
				Owner:                ownerFromBlockingPredicate(predicate),
				FailureClass:         "contract_mismatch",
				FailureStage:         "verify_pipeline",
				Recoverability:       recoverability,
				SchemaRecoverability: schemaRecoverabilityFromLegacy(recoverability),
				NextAction:           "review_and_resubmit",
				BlockingPredicate:    predicate,
			}
			result.AdmissionErrors = violationSummaries
		}
	}

	renderVerifyResult(result, jsonMode, verbose, strictWithheldReason, stdout, sourcePath, schemaPath)

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
		if worktreeAware {
			wt := worktree.CollectInfo(root)
			if wt != nil {
				event["worktree"] = wt
			}
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

func runWithMutationGuard(root string, strict bool, doc map[string]any, validateFn func() error, stderr io.Writer, contextFloor bool, cardPath string) VerifyResult {
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
		result := buildVerifyResult(doc, schemaErr, mg, strict, contextFloor, cardPath)
		if strict {
			fmt.Fprintf(stderr, "mutation_guard_error: guard failed in strict mode: %v\n", guardErr)
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
		}
		return result
	}

	result := buildVerifyResult(doc, schemaErr, mgResult, strict, contextFloor, cardPath)
	if mgResult != nil && mgResult.Violated {
		result.OK = false
		result.AdmissionOutcome = "blocked"
		result.AcceptanceStatus = "withheld"
	}
	return result
}

func buildVerifyResult(doc map[string]any, schemaErr error, mgResult *mutationguard.Result, strict bool, contextFloor bool, cardPath string) VerifyResult {
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
		predicate := "schema_invalid"
		recoverability := "retry_with_fixes"
		result.WithheldReason = &withheldReason{
			Class:                "schema_or_policy_invalid",
			Stage:                "verification",
			Owner:                ownerFromBlockingPredicate(predicate),
			FailureClass:         "schema_invalid",
			FailureStage:         "verify_pipeline",
			Recoverability:       recoverability,
			SchemaRecoverability: schemaRecoverabilityFromLegacy(recoverability),
			NextAction:           "review_and_resubmit",
			BlockingPredicate:    predicate,
		}
		return result
	}

	// Inject cardPath for context floor file resolution
	if cardPath != "" {
		doc["_cardPath"] = cardPath
	}

	admResult := admission.Run(doc, strict, contextFloor)
	result.AdmissionOutcome = admResult.Outcome
	result.AcceptanceStatus = admResult.AcceptanceStatus
	result.AdmissionErrors = admResult.Errors
	result.OK = admResult.Outcome == "success" && admResult.AcceptanceStatus == "accepted" && len(admResult.Errors) == 0
	result.MutationGuard = mgResult

	if admResult.WithheldReason != nil {
		result.WithheldReason = &withheldReason{
			Class:                classFromFailureClass(admResult.WithheldReason.FailureClass),
			Stage:                stageFromFailureStage(admResult.WithheldReason.FailureStage),
			Owner:                ownerFromBlockingPredicate(admResult.BlockingPredicate),
			FailureClass:         admResult.WithheldReason.FailureClass,
			FailureStage:         admResult.WithheldReason.FailureStage,
			Recoverability:       admResult.WithheldReason.Recoverability,
			SchemaRecoverability: schemaRecoverabilityFromLegacy(admResult.WithheldReason.Recoverability),
			NextAction:           admResult.WithheldReason.NextAction,
			BlockingPredicate:    admResult.BlockingPredicate,
		}
	}

	if mgResult != nil && mgResult.Violated {
		result.OK = false
		result.AdmissionOutcome = "blocked"
		result.AcceptanceStatus = "withheld"
		predicate := "verifier_not_read_only"
		recoverability := "manual_review"
		result.WithheldReason = &withheldReason{
			Class:                "verifier_not_read_only",
			Stage:                "verification",
			Owner:                ownerFromBlockingPredicate(predicate),
			FailureClass:         "mutation_detected",
			FailureStage:         "verify_pipeline",
			Recoverability:       recoverability,
			SchemaRecoverability: schemaRecoverabilityFromLegacy(recoverability),
			NextAction:           "review_and_resubmit",
			BlockingPredicate:    predicate,
		}
	}

	return result
}

func renderVerifyResult(result VerifyResult, jsonMode, verbose, strictWithheldReason bool, stdout io.Writer, sourcePath, schemaPath string) {
	if jsonMode {
		if strictWithheldReason && result.WithheldReason != nil {
			strictResult := result
			strictWR := *result.WithheldReason
			strictWR.FailureClass = ""
			strictWR.FailureStage = ""
			// In strict mode, recoverability shows schema enum value
			strictWR.Recoverability = strictWR.SchemaRecoverability
			strictResult.WithheldReason = &strictWR
			WriteJSON(stdout, strictResult)
		} else {
			WriteJSON(stdout, result)
		}
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
		WriteLine(stdout, "  class: %s", result.WithheldReason.Class)
		WriteLine(stdout, "  stage: %s", result.WithheldReason.Stage)
		WriteLine(stdout, "  owner: %s", result.WithheldReason.Owner)
		if strictWithheldReason {
			WriteLine(stdout, "  recoverability: %s", result.WithheldReason.SchemaRecoverability)
			WriteLine(stdout, "  schema_recoverability: %s", result.WithheldReason.SchemaRecoverability)
		} else {
			WriteLine(stdout, "  failure_class: %s", result.WithheldReason.FailureClass)
			WriteLine(stdout, "  failure_stage: %s", result.WithheldReason.FailureStage)
			WriteLine(stdout, "  recoverability: %s", result.WithheldReason.Recoverability)
			WriteLine(stdout, "  schema_recoverability: %s", result.WithheldReason.SchemaRecoverability)
		}
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
