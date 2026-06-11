package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/approvalrisk"
	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/boundary"
	"github.com/BrianNguyen29/x-harness/internal/contextmanifest"
	"github.com/BrianNguyen29/x-harness/internal/contract"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/mutationguard"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"github.com/BrianNguyen29/x-harness/internal/worktree"
)

// evaluateApprovalRiskAdvisory runs the advisory approval-risk engine for a
// completion card. It is strictly read-only and never affects ok,
// admission.outcome, acceptance_status, errors, blocking_predicate, or
// admission_authority. Returns "" when the policy is disabled, the card cannot
// be evaluated, or the engine is not available. Mirrors the TypeScript
// implementation in packages/cli/src/core/verify-pipeline.ts.
func evaluateApprovalRiskAdvisory(cardPath, root string) string {
	if cardPath == "" {
		return ""
	}
	report, err := approvalrisk.EvaluateApprovalRisk(cardPath, root)
	if err != nil {
		// Advisory-only: skip silently on evaluation errors so the verify
		// pipeline never fails because approval-risk is unavailable.
		return ""
	}
	if !report.PolicyEnabled {
		return ""
	}
	return fmt.Sprintf(
		"approval-risk advisory: score=%d risk_class=%s signals=[%s] required_approvals=%d",
		report.Score,
		report.RiskClass,
		strings.Join(report.Signals, ","),
		report.RequiredApprovals,
	)
}

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
	case "boundary_violation":
		return "implementation-worker"
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
	case "stale_context":
		return "context_stale"
	case "context_missing":
		return "context_floor_blocked"
	case "schema_invalid":
		return "schema_or_policy_invalid"
	case "mutation_detected":
		return "verifier_not_read_only"
	case "governance_missing":
		return "approval_receipt_missing"
	case "command_risky":
		return "approval_receipt_missing"
	case "evidence_provenance_invalid":
		return "external_dependency_unverified"
	case "tier_under_escalated":
		return "approval_scope_invalid"
	case "evidence_missing", "evidence_floor_missing":
		return "evidence_floor_missing"
	case "prediction_missing":
		return "prediction_missing"
	case "done_checklist_missing":
		return "done_checklist_missing"
	case "admission_mapping_invalid":
		return "admission_mapping_invalid"
	case "boundary_violation":
		return "approval_scope_invalid"
	case "decision_refs_missing", "intent_ref_missing":
		return "context_insufficient"
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
	case "admission_gate":
		return "admission"
	default:
		return failureStage
	}
}

// VerifyResult is the minimal output of the verify command.
type VerifyResult struct {
	SchemaVersion       string                `json:"schema_version"`
	OK                  bool                  `json:"ok"`
	TaskID              string                `json:"task_id"`
	Tier                string                `json:"tier"`
	Profile             string                `json:"profile,omitempty"`
	AdmissionOutcome    string                `json:"admission_outcome"`
	AcceptanceStatus    string                `json:"acceptance_status"`
	SchemaError         string                `json:"schema_error,omitempty"`
	AdmissionErrors     []string              `json:"admission_errors,omitempty"`
	AdmissionNotes      []string              `json:"admission_notes,omitempty"`
	ProductIntentStatus string                `json:"product_intent_status,omitempty"`
	MutationGuard       *mutationguard.Result `json:"mutation_guard,omitempty"`
	WithheldReason      *withheldReason       `json:"withheld_reason,omitempty"`
}

// VerifyProfile is a named bundle of verify flags. The map is intentionally
// hardcoded in V1 (no dynamic policy file) so reviewers can trace each entry
// to a real flag in handleVerify. The list of names must match
// schemas/policy-matrix.schema.json profile enums.
type VerifyProfile struct {
	Name            string
	Description     string
	MutationGuard   bool
	Strict          bool
	ContextFloor    bool
	ContractOracles bool
	WorktreeAware   bool
	StrictWithheld  bool
	BoundaryEnforce string // off|advisory|block_high|block_all (default "" = use flag)
	DecisionEnforce string // off|advisory|block for context_alignment.decision_refs (default "" = use flag)
	IntentEnforce   string // off|advisory|block for top-level intent_ref (default "" = use flag)
	ContextEnforce  string // off|advisory|block for context manifest staleness (default "" = use flag)
}

// verifyProfileNames returns the sorted list of profile names for usage
// messages and policy matrix reference.
func verifyProfileNames() []string {
	names := make([]string, 0, len(verifyProfiles))
	for name := range verifyProfiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var verifyProfiles = map[string]VerifyProfile{
	"light-local": {
		Name:            "light-local",
		Description:     "Local development: minimal checks, no mutation guard. Boundary violations are advisory/warning only and never block. intent_ref is advisory-only and never blocks.",
		MutationGuard:   false,
		Strict:          false,
		ContextFloor:    false,
		ContractOracles: false,
		WorktreeAware:   false,
		BoundaryEnforce: "advisory",
		DecisionEnforce: "advisory",
		IntentEnforce:   "advisory",
		ContextEnforce:  "advisory",
	},
	"ci-standard": {
		Name:            "ci-standard",
		Description:     "CI standard: mutation guard + context floor, advisory contract oracles, advisory-only boundary enforcement. intent_ref is advisory-only and never blocks by default. context manifest staleness is advisory-only.",
		MutationGuard:   true,
		Strict:          false,
		ContextFloor:    true,
		ContractOracles: false,
		WorktreeAware:   true,
		BoundaryEnforce: "advisory",
		DecisionEnforce: "advisory",
		IntentEnforce:   "advisory",
		ContextEnforce:  "advisory",
	},
	"ci-strict": {
		Name:            "ci-strict",
		Description:     "CI strict: mutation guard, context floor, contract oracles, strict withheld reason schema. Blocks high/critical boundary violations and missing context_alignment.decision_refs on standard/deep cards. intent_ref is advisory-only by default; explicit --intent-enforce block can block. context manifest staleness is blocked by default.",
		MutationGuard:   true,
		Strict:          true,
		ContextFloor:    true,
		ContractOracles: true,
		WorktreeAware:   true,
		StrictWithheld:  true,
		BoundaryEnforce: "block_high",
		DecisionEnforce: "block",
		IntentEnforce:   "advisory",
		ContextEnforce:  "block",
	},
	"governed-deep": {
		Name:            "governed-deep",
		Description:     "Governed deep: all ci-strict checks plus strict withheld reason schema. Blocks all boundary violations unless approved via boundary_approvals, blocks missing context_alignment.decision_refs on standard/deep cards, blocks missing/blank intent_ref on standard/deep cards, and blocks stale context manifests on standard/deep cards.",
		MutationGuard:   true,
		Strict:          true,
		ContextFloor:    true,
		ContractOracles: true,
		WorktreeAware:   true,
		StrictWithheld:  true,
		BoundaryEnforce: "block_all",
		DecisionEnforce: "block",
		IntentEnforce:   "block",
		ContextEnforce:  "block",
	},
}

// resolveVerifyProfile is retained as a documentation reference for the
// flag-override semantics; the production code path inlines the same logic
// to keep the explicit-flag override decision obvious to reviewers.
//
//nolint:unused
func resolveVerifyProfile(args []string, useMutationGuard *bool, strict *bool, contextFloor *bool, contractOracles *bool, worktreeAware *bool, strictWithheld *bool, boundaryEnforce *string, decisionEnforce *string, intentEnforce *string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != "--profile" {
			continue
		}
		if i+1 >= len(args) {
			return ""
		}
		name := args[i+1]
		profile, ok := verifyProfiles[name]
		if !ok {
			return ""
		}
		// Profile is only applied when the corresponding flag was not
		// explicitly set by the caller. Explicit flags win over profile.
		if !*useMutationGuard {
			*useMutationGuard = profile.MutationGuard
		}
		if !*strict {
			*strict = profile.Strict
		}
		if !*contextFloor {
			*contextFloor = profile.ContextFloor
		}
		if !*contractOracles {
			*contractOracles = profile.ContractOracles
		}
		if !*worktreeAware {
			*worktreeAware = profile.WorktreeAware
		}
		if !*strictWithheld {
			*strictWithheld = profile.StrictWithheld
		}
		if *boundaryEnforce == "" {
			*boundaryEnforce = profile.BoundaryEnforce
		}
		if *decisionEnforce == "" {
			*decisionEnforce = profile.DecisionEnforce
		}
		if *intentEnforce == "" {
			*intentEnforce = profile.IntentEnforce
		}
		return name
	}
	return ""
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
	boundaryEnforce := ""
	boundaryPolicy := ""
	decisionEnforce := ""
	intentEnforce := ""
	contextEnforce := ""
	profileName := ""

	// Track which flags the caller set explicitly so the profile layer
	// does not silently override them.
	explicitMutationGuard := false
	explicitStrict := false
	explicitContextFloor := false
	explicitContractOracles := false
	explicitWorktreeAware := false
	explicitStrictWithheld := false
	explicitBoundaryEnforce := false
	explicitDecisionEnforce := false
	explicitIntentEnforce := false
	explicitContextEnforce := false

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
		case "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --profile requires a value")
				return ExitUsage
			}
			profileName = args[i+1]
			i++
		case "--json":
			jsonMode = true
		case "--verbose":
			verbose = true
		case "--mutation-guard":
			useMutationGuard = true
			explicitMutationGuard = true
		case "--strict":
			strict = true
			useMutationGuard = true
			explicitStrict = true
			explicitMutationGuard = true
		case "--strict-withheld-reason":
			strictWithheldReason = true
			explicitStrictWithheld = true
		case "--trace":
			trace = true
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--worktree-aware":
			worktreeAware = true
			explicitWorktreeAware = true
		case "--context-floor":
			contextFloor = true
			explicitContextFloor = true
		case "--contract-oracles":
			contractOracles = true
			explicitContractOracles = true
		case "--contract-oracles-policy":
			if i+1 < len(args) {
				contractOraclesPolicy = args[i+1]
				i++
			}
		case "--boundary-enforce":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --boundary-enforce requires a value (off|advisory|block_high|block_all)")
				return ExitUsage
			}
			v := args[i+1]
			if !isValidBoundaryEnforce(v) {
				fmt.Fprintf(stderr, "error: invalid --boundary-enforce %q (allowed: off, advisory, block_high, block_all)\n", v)
				return ExitUsage
			}
			boundaryEnforce = v
			explicitBoundaryEnforce = true
			i++
		case "--boundary-policy":
			if i+1 < len(args) {
				boundaryPolicy = args[i+1]
				i++
			}
		case "--decision-enforce":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --decision-enforce requires a value (off|advisory|block)")
				return ExitUsage
			}
			v := args[i+1]
			if !isValidDecisionEnforce(v) {
				fmt.Fprintf(stderr, "error: invalid --decision-enforce %q (allowed: off, advisory, block)\n", v)
				return ExitUsage
			}
			decisionEnforce = v
			explicitDecisionEnforce = true
			i++
		case "--intent-enforce":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --intent-enforce requires a value (off|advisory|block)")
				return ExitUsage
			}
			v := args[i+1]
			if !isValidIntentEnforce(v) {
				fmt.Fprintf(stderr, "error: invalid --intent-enforce %q (allowed: off, advisory, block)\n", v)
				return ExitUsage
			}
			intentEnforce = v
			explicitIntentEnforce = true
			i++
		case "--context-enforce":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --context-enforce requires a value (off|advisory|block)")
				return ExitUsage
			}
			v := args[i+1]
			if !isValidContextEnforce(v) {
				fmt.Fprintf(stderr, "error: invalid --context-enforce %q (allowed: off, advisory, block)\n", v)
				return ExitUsage
			}
			contextEnforce = v
			explicitContextEnforce = true
			i++
		}
	}

	if profileName != "" {
		profile, ok := verifyProfiles[profileName]
		if !ok {
			fmt.Fprintf(stderr, "error: unknown --profile %q. Available: %s\n", profileName, strings.Join(verifyProfileNames(), ", "))
			return ExitUsage
		}
		if !explicitMutationGuard {
			useMutationGuard = profile.MutationGuard
		}
		if !explicitStrict {
			strict = profile.Strict
		}
		if !explicitContextFloor {
			contextFloor = profile.ContextFloor
		}
		if !explicitContractOracles {
			contractOracles = profile.ContractOracles
		}
		if !explicitWorktreeAware {
			worktreeAware = profile.WorktreeAware
		}
		if !explicitStrictWithheld {
			strictWithheldReason = profile.StrictWithheld
		}
		if !explicitBoundaryEnforce {
			boundaryEnforce = profile.BoundaryEnforce
		}
		if !explicitDecisionEnforce {
			decisionEnforce = profile.DecisionEnforce
		}
		if !explicitIntentEnforce {
			intentEnforce = profile.IntentEnforce
		}
		if !explicitContextEnforce {
			contextEnforce = profile.ContextEnforce
		}
	}

	if (cardPath == "" && subagentPath == "") || (cardPath != "" && subagentPath != "") {
		fmt.Fprintln(stderr, "usage: x-harness verify --card <path> | --subagent-return <path> [--profile <light-local|ci-standard|ci-strict|governed-deep>] [--tier <tier>] [--json] [--verbose] [--mutation-guard] [--strict] [--strict-withheld-reason] [--trace] [--trace-dir <dir>] [--worktree-aware] [--context-floor] [--contract-oracles] [--contract-oracles-policy <path>] [--boundary-enforce off|advisory|block_high|block_all] [--boundary-policy <path>] [--decision-enforce off|advisory|block] [--intent-enforce off|advisory|block] [--context-enforce off|advisory|block]")
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
			result := buildVerifyResult(nil, schemaErr, nil, strict, false, "", root)
			renderVerifyResult(result, jsonMode, verbose, strictWithheldReason, stdout, sourcePath, schemaPath)
			return ExitError
		}
		handoff := mapValue(subagentDoc, "handoff")
		owner := strings.TrimSpace(stringValue(handoff, "owner"))
		if owner == "" {
			owner = "compatibility-subagent"
		}
		resultDoc := mapValue(subagentDoc, "result")
		fixStatus := strings.TrimSpace(stringValue(resultDoc, "fix_status"))
		verifyStatus := strings.TrimSpace(stringInMap(mapValue(subagentDoc, "verification"), "status"))
		admissionOutcome := "blocked"
		acceptanceStatus := "withheld"
		if fixStatus == "fixed" && verifyStatus == "passed" {
			admissionOutcome = "success"
			acceptanceStatus = "accepted"
		}
		taskID := strings.TrimSpace(stringValue(subagentDoc, "task_id"))
		if taskID == "" {
			taskID = "compat-" + strings.TrimSuffix(filepath.Base(subagentPath), filepath.Ext(subagentPath))
		}
		mappedDoc := map[string]any{
			"schema_version":    "x-harness.completion-card.v1",
			"task_id":           taskID,
			"owner":             owner,
			"accountable":       owner,
			"subagent_return":   subagentDoc,
			"admission":         map[string]any{"outcome": admissionOutcome},
			"acceptance_status": acceptanceStatus,
			"claim": map[string]any{
				"fix_status": fixStatus,
				"summary":    stringValue(resultDoc, "summary"),
				"evidence":   mapValue(subagentDoc, "evidence"),
			},
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

	var outerGuard *verifyMutationSnapshot
	if useMutationGuard {
		outerGuard, err = beginVerifyMutationSnapshot(root)
		if err != nil {
			mg := &mutationguard.Result{Enabled: true, SkippedReason: err.Error(), Violated: true}
			result := buildVerifyResult(doc, nil, mg, strict, contextFloor, cardPath, root)
			blockForMutationGuardError(&result, err)
			if renderErr := renderVerifyResult(result, jsonMode, verbose, strictWithheldReason, stdout, sourcePath, schemaPath); renderErr != nil {
				fmt.Fprintf(stderr, "error: failed to render verify result: %v\n", renderErr)
			}
			return ExitError
		}
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
		result = buildVerifyResult(doc, schemaErr, nil, strict, contextFloor, cardPath, root)
	}

	// Stamp the active profile (if any) on the result. Field is omitted
	// when no --profile flag was supplied.
	if profileName != "" {
		result.Profile = profileName
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

	// Boundary check (opt-in via profile or --boundary-enforce). Skipped
	// silently when no policy is loaded (boundary.Check treats nil
	// policy as a no-op).
	if boundaryEnforce != "" && boundaryEnforce != "off" && result.OK {
		// Default policy path mirrors the boundary CLI command.
		policyPath := boundaryPolicy
		if policyPath == "" {
			policyPath = filepath.Join(root, "policies", "boundaries.yaml")
		}
		// Load policy (nil is allowed; Check treats it as no-op).
		var pol *boundary.Policy
		if _, statErr := os.Stat(policyPath); statErr == nil {
			loaded, loadErr := boundary.Load(policyPath)
			if loadErr != nil {
				fmt.Fprintf(stderr, "error: cannot load boundary policy %s: %v\n", policyPath, loadErr)
				return ExitError
			}
			pol = loaded
		} else if boundaryPolicy != "" {
			// Explicit policy path that doesn't exist: surface the error.
			fmt.Fprintf(stderr, "error: boundary policy not found: %s\n", policyPath)
			return ExitError
		}
		boundaryResult, boundaryErr := boundary.Check(pol, policyPath, []string{root})
		if boundaryErr != nil {
			fmt.Fprintf(stderr, "error: boundary check failed: %v\n", boundaryErr)
			return ExitError
		}
		// Apply enforcement to surviving violations. An approval list
		// from the card suppresses blocking for matching rule_ids.
		approved := extractBoundaryApprovals(doc)
		blockingViolations := filterBoundaryViolationsByEnforce(boundaryResult.Violations, boundaryEnforce, approved)
		if len(blockingViolations) > 0 {
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
			// Compact summary.
			var violationSummaries []string
			maxViolations := 3
			if len(blockingViolations) < maxViolations {
				maxViolations = len(blockingViolations)
			}
			for i := 0; i < maxViolations; i++ {
				v := blockingViolations[i]
				relPath, _ := filepath.Rel(root, v.File)
				violationSummaries = append(violationSummaries, fmt.Sprintf("%s:%d: %s [%s/%s]", relPath, v.Line, v.Message, v.Severity, v.RuleID))
			}
			if len(blockingViolations) > maxViolations {
				violationSummaries = append(violationSummaries, fmt.Sprintf("... and %d more violations", len(blockingViolations)-maxViolations))
			}
			predicate := "boundary_violation"
			recoverability := "retry_with_fixes"
			result.WithheldReason = &withheldReason{
				Class:                classFromFailureClass("boundary_violation"),
				Stage:                "verification",
				Owner:                ownerFromBlockingPredicate(predicate),
				FailureClass:         "boundary_violation",
				FailureStage:         "verify_pipeline",
				Recoverability:       recoverability,
				SchemaRecoverability: schemaRecoverabilityFromLegacy(recoverability),
				NextAction:           "review_and_resubmit",
				BlockingPredicate:    predicate,
			}
			result.AdmissionErrors = violationSummaries
		} else if boundaryResult.OK == false && len(boundaryResult.Violations) > 0 {
			// Advisory mode: surface violations as a note without
			// altering admission outcome. The note reports the total
			// count so reviewers can see the suppressions and
			// remaining items.
			note := fmt.Sprintf("boundary advisory: %d total violation(s), 0 blocking under enforce=%s", len(boundaryResult.Violations), boundaryEnforce)
			result.AdmissionNotes = append(result.AdmissionNotes, note)
		}
	}

	// Decision-refs gate (profile-controlled). Mirrors the
	// BoundaryEnforce style: off|advisory|block with the closed enum
	// validated in isValidDecisionEnforce. The off and advisory modes
	// never block at the verify layer; the advisory note from
	// admission.Run is preserved as-is. The block mode withholds
	// standard/deep cards whose context_alignment.decision_refs array
	// is missing or contains no non-blank string entries, with the
	// explicit `decision_refs_missing` blocking predicate so the
	// failure remains traceable.
	if decisionEnforce == "block" && result.OK {
		if (effectiveTier == "standard" || effectiveTier == "deep") && !admission.HasAnyDecisionRef(doc) {
			predicate := "decision_refs_missing"
			recoverability := "retry_with_fixes"
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
			result.WithheldReason = &withheldReason{
				Class:                classFromFailureClass("decision_refs_missing"),
				Stage:                "verification",
				Owner:                ownerFromBlockingPredicate(predicate),
				FailureClass:         "decision_refs_missing",
				FailureStage:         "verify_pipeline",
				Recoverability:       recoverability,
				SchemaRecoverability: schemaRecoverabilityFromLegacy(recoverability),
				NextAction:           "add_decision_refs_and_resubmit",
				BlockingPredicate:    predicate,
			}
			result.AdmissionErrors = []string{"context_alignment.decision_refs is empty (verify-stage block; admission acceptance is not decision correctness)"}
		}
	}

	// Intent-ref gate (profile-controlled). Mirrors the decision-refs
	// gate: off|advisory|block with the closed enum validated in
	// isValidIntentEnforce. The off and advisory modes never block at
	// the verify layer; the advisory note from admission.Run is
	// preserved as-is. The block mode withholds standard/deep cards
	// whose top-level intent_ref is missing or blank, with the
	// explicit `intent_ref_missing` blocking predicate so the failure
	// remains traceable. The predicate is deliberately
	// process/governance-only; it does not imply product correctness.
	// Profile defaults are conservative per oracle review: light-local,
	// ci-standard, and ci-strict are advisory by default; only
	// governed-deep blocks by default. An explicit --intent-enforce
	// block can still block under any profile.
	if intentEnforce == "block" && result.OK {
		if (effectiveTier == "standard" || effectiveTier == "deep") && !admission.HasAnyIntentRef(doc) {
			predicate := "intent_ref_missing"
			recoverability := "retry_with_fixes"
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
			result.WithheldReason = &withheldReason{
				Class:                classFromFailureClass("intent_ref_missing"),
				Stage:                "verification",
				Owner:                ownerFromBlockingPredicate(predicate),
				FailureClass:         "intent_ref_missing",
				FailureStage:         "verify_pipeline",
				Recoverability:       recoverability,
				SchemaRecoverability: schemaRecoverabilityFromLegacy(recoverability),
				NextAction:           "add_intent_ref_and_resubmit",
				BlockingPredicate:    predicate,
			}
			result.AdmissionErrors = []string{"intent_ref not declared (verify-stage block; admission acceptance is not intent correctness)"}
		}
	}

	// Context-enforce gate (profile-controlled). Mirrors the
	// decision-refs and intent-ref gates: off|advisory|block with the
	// closed enum validated in isValidContextEnforce. The off and
	// advisory modes never block at the verify layer. The block mode
	// withholds standard/deep cards whose context_manifest is present,
	// readable, valid, and stale. Missing manifest skips silently;
	// invalid/unreadable manifest is advisory-only and never blocks.
	// Light tier is always non-blocking regardless of mode.
	if contextEnforce != "" && contextEnforce != "off" && result.OK {
		if effectiveTier == "standard" || effectiveTier == "deep" {
			manifestPath := resolveContextManifestPathFromDoc(doc)
			if manifestPath != "" {
				resolved := resolveManifestPath(manifestPath, filepath.Dir(cardPath))
				if resolved != "" {
					manifest, err := contextmanifest.Read(resolved)
					if err == nil {
						if err := contextmanifest.Validate(manifest); err == nil {
							stale, _ := contextmanifest.Check(manifest, root)
							if len(stale) > 0 {
								if contextEnforce == "block" {
									predicate := "context_stale"
									recoverability := "retry_with_fixes"
									result.OK = false
									result.AdmissionOutcome = "blocked"
									result.AcceptanceStatus = "withheld"
									result.WithheldReason = &withheldReason{
										Class:                "context_stale",
										Stage:                "verification",
										Owner:                ownerFromBlockingPredicate(predicate),
										FailureClass:         "context_stale",
										FailureStage:         "verify_pipeline",
										Recoverability:       recoverability,
										SchemaRecoverability: schemaRecoverabilityFromLegacy(recoverability),
										NextAction:           "refresh_context_manifest_and_resubmit",
										BlockingPredicate:    predicate,
									}
									result.AdmissionErrors = append(result.AdmissionErrors, fmt.Sprintf("context manifest stale: %s (verify-stage block; admission acceptance is not context correctness)", strings.Join(stale, ", ")))
								} else {
									result.AdmissionNotes = append(result.AdmissionNotes, fmt.Sprintf("context_stale advisory: manifest stale for %s", strings.Join(stale, ", ")))
								}
							}
						} else {
							result.AdmissionNotes = append(result.AdmissionNotes, fmt.Sprintf("context_manifest advisory: invalid manifest at %s", resolved))
						}
					} else {
						result.AdmissionNotes = append(result.AdmissionNotes, fmt.Sprintf("context_manifest advisory: cannot read manifest at %s", resolved))
					}
				}
			}
		}
	}

	if outerGuard != nil {
		mgResult, err := outerGuard.finish()
		if err != nil {
			if result.MutationGuard == nil {
				result.MutationGuard = &mutationguard.Result{Enabled: true, SkippedReason: err.Error(), Violated: true}
			}
			blockForMutationGuardError(&result, err)
		} else if mgResult != nil && mgResult.Violated {
			result.MutationGuard = mgResult
			blockForMutationViolation(&result)
		} else if result.MutationGuard == nil {
			result.MutationGuard = mgResult
		}
	}

	if err := renderVerifyResult(result, jsonMode, verbose, strictWithheldReason, stdout, sourcePath, schemaPath); err != nil {
		fmt.Fprintf(stderr, "error: failed to render verify result: %v\n", err)
		return ExitError
	}

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
			"notes":                result.AdmissionNotes,
			"errors":               append([]string{}, result.AdmissionErrors...),
		}
		if result.WithheldReason != nil {
			event["blocking_predicate"] = result.WithheldReason.BlockingPredicate
			event["blocked_reason_class"] = result.WithheldReason.FailureClass
			event["next_owner"] = result.WithheldReason.Owner
			event["next_action"] = result.WithheldReason.NextAction
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

type verifyMutationSnapshot struct {
	root    string
	useGit  bool
	before  *mutationguard.Snapshot
	gitRoot string
}

func beginVerifyMutationSnapshot(root string) (*verifyMutationSnapshot, error) {
	if mutationguard.IsGitAvailable() {
		if gitRoot, err := mutationguard.FindGitRoot(root); err == nil {
			before, err := mutationguard.TakeSnapshot(gitRoot)
			if err != nil {
				return nil, fmt.Errorf("before snapshot failed: %w", err)
			}
			return &verifyMutationSnapshot{root: root, useGit: true, before: before, gitRoot: gitRoot}, nil
		}
	}
	before, err := mutationguard.TakeFallbackSnapshot(root)
	if err != nil {
		return nil, fmt.Errorf("fallback before snapshot failed: %w", err)
	}
	return &verifyMutationSnapshot{root: root, before: before}, nil
}

func (s *verifyMutationSnapshot) finish() (*mutationguard.Result, error) {
	var after *mutationguard.Snapshot
	var err error
	if s.useGit {
		after, err = mutationguard.TakeSnapshot(s.gitRoot)
	} else {
		after, err = mutationguard.TakeFallbackSnapshot(s.root)
	}
	if err != nil {
		return nil, fmt.Errorf("after snapshot failed: %w", err)
	}
	deltas := mutationguard.Compare(s.before, after)
	unexpected := mutationguard.FilterUnexpected(deltas)
	return &mutationguard.Result{
		Enabled:          true,
		Deltas:           deltas,
		UnexpectedDeltas: unexpected,
		Violated:         len(unexpected) > 0,
	}, nil
}

func blockForMutationGuardError(result *VerifyResult, err error) {
	if result == nil {
		return
	}
	if result.MutationGuard == nil {
		result.MutationGuard = &mutationguard.Result{Enabled: true, SkippedReason: err.Error(), Violated: true}
	}
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
	result.AdmissionErrors = append(result.AdmissionErrors, fmt.Sprintf("mutation_guard_error: %v", err))
}

func blockForMutationViolation(result *VerifyResult) {
	if result == nil {
		return
	}
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
		mg := &mutationguard.Result{Enabled: true, SkippedReason: guardErr.Error(), Violated: true}
		result := buildVerifyResult(doc, schemaErr, mg, strict, contextFloor, cardPath, root)
		fmt.Fprintf(stderr, "mutation_guard_error: guard failed: %v\n", guardErr)
		blockForMutationGuardError(&result, guardErr)
		return result
	}

	result := buildVerifyResult(doc, schemaErr, mgResult, strict, contextFloor, cardPath, root)
	if mgResult != nil && mgResult.Violated {
		blockForMutationViolation(&result)
	}
	return result
}

func buildVerifyResult(doc map[string]any, schemaErr error, mgResult *mutationguard.Result, strict bool, contextFloor bool, cardPath string, root string) VerifyResult {
	result := VerifyResult{
		SchemaVersion: "x-harness.verify-result.v1",
		TaskID:        stringValue(doc, "task_id"),
		Tier:          stringValue(doc, "tier"),
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
	result.AdmissionNotes = admResult.Notes
	result.ProductIntentStatus = productIntentStatusFromDoc(doc)
	result.OK = admResult.Outcome == "success" && admResult.AcceptanceStatus == "accepted" && len(admResult.Errors) == 0
	result.MutationGuard = mgResult

	// Advisory approval-risk note. Emitted only when the policy is enabled
	// and the engine evaluates successfully. Never alters ok,
	// admission.outcome, acceptance_status, errors, blocking_predicate, or
	// admission_authority. Skipped silently on evaluation errors.
	if cardPath != "" {
		if note := evaluateApprovalRiskAdvisory(cardPath, root); note != "" {
			result.AdmissionNotes = append(result.AdmissionNotes, note)
		}
	}

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

func renderVerifyResult(result VerifyResult, jsonMode, verbose, strictWithheldReason bool, stdout io.Writer, sourcePath, schemaPath string) error {
	if jsonMode {
		if strictWithheldReason && result.WithheldReason != nil {
			strictResult := result
			strictWR := *result.WithheldReason
			strictWR.FailureClass = ""
			strictWR.FailureStage = ""
			// In strict mode, recoverability shows schema enum value
			strictWR.Recoverability = strictWR.SchemaRecoverability
			strictResult.WithheldReason = &strictWR
			return WriteJSON(stdout, strictResult)
		} else {
			return WriteJSON(stdout, result)
		}
		return nil
	}
	if verbose {
		WriteLine(stdout, "source: %s", sourcePath)
		WriteLine(stdout, "schema: %s", schemaPath)
	}
	WriteLine(stdout, "task_id: %s", result.TaskID)
	WriteLine(stdout, "tier: %s", result.Tier)
	if result.Profile != "" {
		WriteLine(stdout, "profile: %s", result.Profile)
	}
	WriteLine(stdout, "outcome: %s", result.AdmissionOutcome)
	WriteLine(stdout, "acceptance_status: %s", result.AcceptanceStatus)
	if result.ProductIntentStatus != "" {
		WriteLine(stdout, "product_intent: %s", result.ProductIntentStatus)
	}
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
	return nil
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

// productIntentStatusFromDoc extracts the optional product_intent.status from
// a card document. Returns the empty string when absent so the JSON field
// stays omitted and the rendered output stays parity-safe with TS.
func productIntentStatusFromDoc(doc map[string]any) string {
	productIntent := mapValue(doc, "product_intent")
	if productIntent == nil {
		return ""
	}
	return strings.TrimSpace(stringValue(productIntent, "status"))
}

// isValidBoundaryEnforce reports whether value is one of the supported
// enforcement modes for the verify-stage boundary check. The enum is
// intentionally closed: future values must be added here and the
// matrix/schema updated alongside.
func isValidBoundaryEnforce(v string) bool {
	switch v {
	case "off", "advisory", "block_high", "block_all":
		return true
	}
	return false
}

// isValidDecisionEnforce reports whether value is one of the supported
// enforcement modes for the verify-stage decision_refs gate. Mirrors
// the closed-enum style of isValidBoundaryEnforce.
func isValidDecisionEnforce(v string) bool {
	switch v {
	case "off", "advisory", "block":
		return true
	}
	return false
}

// isValidIntentEnforce reports whether value is one of the supported
// enforcement modes for the verify-stage intent_ref gate. Mirrors the
// closed-enum style of isValidDecisionEnforce.
func isValidIntentEnforce(v string) bool {
	switch v {
	case "off", "advisory", "block":
		return true
	}
	return false
}

// extractBoundaryApprovals reads externally authorized `boundary_approvals`
// entries and returns the set of rule_ids that can suppress boundary blocking.
// Rule-only entries are deliberately ignored so an agent cannot self-approve a
// protected boundary in the same completion card.
func extractBoundaryApprovals(doc map[string]any) map[string]bool {
	approved := map[string]bool{}
	if doc == nil {
		return approved
	}
	raw, ok := doc["boundary_approvals"]
	if !ok || raw == nil {
		return approved
	}
	arr, ok := raw.([]any)
	if !ok {
		return approved
	}
	for _, item := range arr {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ruleID, _ := entry["rule_id"].(string)
		ruleID = strings.TrimSpace(ruleID)
		if ruleID == "" {
			continue
		}
		approver, _ := entry["approver"].(string)
		reason, _ := entry["reason"].(string)
		approvedAt, _ := entry["approved_at"].(string)
		if strings.TrimSpace(approver) == "" || strings.TrimSpace(reason) == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(approvedAt)); err != nil {
			continue
		}
		approved[ruleID] = true
	}
	return approved
}

// boundaryBlocksSeverity returns true when the given severity should
// block under the named enforcement mode. The mapping mirrors the V1
// contract: light-local and ci-standard are advisory-only; ci-strict
// blocks high and critical; governed-deep blocks everything.
func boundaryBlocksSeverity(mode string, sev boundary.Severity) bool {
	switch mode {
	case "block_all":
		return true
	case "block_high":
		return sev == boundary.SeverityHigh || sev == boundary.SeverityCritical
	default:
		return false
	}
}

// filterBoundaryViolationsByEnforce returns the subset of violations
// that should block admission under mode, after subtracting any
// approved rule_ids. An empty result means the violations are
// non-blocking (either advisory or approved).
func filterBoundaryViolationsByEnforce(violations []boundary.Violation, mode string, approved map[string]bool) []boundary.Violation {
	if mode == "off" || mode == "advisory" {
		return nil
	}
	var blocking []boundary.Violation
	for _, v := range violations {
		if approved[v.RuleID] {
			continue
		}
		if !boundaryBlocksSeverity(mode, v.Severity) {
			continue
		}
		blocking = append(blocking, v)
	}
	return blocking
}

// isValidContextEnforce reports whether value is one of the supported
// enforcement modes for the verify-stage context manifest staleness
// gate. Mirrors the closed-enum style of isValidDecisionEnforce.
func isValidContextEnforce(v string) bool {
	switch v {
	case "off", "advisory", "block":
		return true
	}
	return false
}

// resolveContextManifestPathFromDoc extracts the manifest path from a
// completion card document. It supports a top-level string
// "context_manifest" or an object with a "path" field.
func resolveContextManifestPathFromDoc(doc map[string]any) string {
	if doc == nil {
		return ""
	}
	raw, ok := doc["context_manifest"]
	if !ok || raw == nil {
		return ""
	}
	if s, ok := raw.(string); ok {
		return strings.TrimSpace(s)
	}
	if m, ok := raw.(map[string]any); ok {
		if s, ok := m["path"].(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// resolveManifestPath resolves a manifest path relative to cardDir or
// cwd. Returns empty string when the file does not exist.
func resolveManifestPath(manifestPath, cardDir string) string {
	if filepath.IsAbs(manifestPath) {
		if _, err := os.Stat(manifestPath); err == nil {
			return manifestPath
		}
		return ""
	}
	for _, base := range []string{"", cardDir} {
		candidate := manifestPath
		if base != "" {
			candidate = filepath.Join(base, manifestPath)
		} else {
			absPath, err := filepath.Abs(manifestPath)
			if err != nil {
				continue
			}
			candidate = absPath
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
