package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Matrix rule status values. These match the policy-matrix.schema.json enum.
const (
	matrixStatusRuntimeBlocking = "runtime_blocking"
	matrixStatusAdvisory        = "advisory"
	matrixStatusOffByDefault    = "off_by_default"
	matrixStatusExperimental    = "experimental"
)

// Matrix rule source values. These match the policy-matrix.schema.json enum.
const (
	matrixSourceCurated = "curated"
	matrixSourcePolicy  = "policy"
)

// MatrixRule describes a single x-harness policy rule in the matrix output.
type MatrixRule struct {
	ID                 string   `json:"id"`
	Description        string   `json:"description,omitempty"`
	PolicyFile         string   `json:"policy_file,omitempty"`
	RuntimeModule      string   `json:"runtime_module,omitempty"`
	Status             string   `json:"status"`
	EnabledByDefault   bool     `json:"enabled_by_default"`
	AdmissionAuthority *bool    `json:"admission_authority,omitempty"`
	Profiles           []string `json:"profiles,omitempty"`
	EnabledByFlags     []string `json:"enabled_by_flags,omitempty"`
	Fixtures           []string `json:"fixtures,omitempty"`
	Source             string   `json:"source"`
}

// Matrix is the top-level output of `xh policy matrix --json`.
type Matrix struct {
	Version int          `json:"version"`
	Rules   []MatrixRule `json:"rules"`
}

// curatedMatrix is the v1 curated list of policy rules. The mapping is
// intentionally explicit and human-readable so reviewers can trace each
// entry to a real policy/runtime module in the repo. Later slices may
// derive entries dynamically from policy files.
func curatedMatrix() []MatrixRule {
	return []MatrixRule{
		{
			ID:                 "admission.evidence_floor",
			Description:        "Tier-based evidence floor (light/standard/deep) for completion card admission.",
			PolicyFile:         "policies/admission.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusRuntimeBlocking,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(true),
			Profiles:           []string{"light-local", "ci-standard", "ci-strict", "governed-deep"},
			Fixtures:           []string{"examples/golden/regression/success-light", "examples/golden/regression/blocked-missing-evidence"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "admission.schema_validation",
			Description:        "JSON Schema validation of completion card, evidence, and related artifacts.",
			PolicyFile:         "policies/admission.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusRuntimeBlocking,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(true),
			Profiles:           []string{"light-local", "ci-standard", "ci-strict", "governed-deep"},
			Fixtures:           []string{"examples/golden/regression/failed-invalid-status"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "admission.failure_taxonomy",
			Description:        "Classifies withheld outcomes with failure_class, failure_stage, recoverability, and next_action.",
			PolicyFile:         "policies/admission.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusRuntimeBlocking,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(true),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "admission.intent_contract",
			Description:        "Advisory checks for intent_contract fields (user_visible_change, product_goal, non_goals, acceptance_criteria). Never blocks admission in the first slice.",
			PolicyFile:         "policies/admission.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "admission.test_adequacy",
			Description:        "Advisory checks for test_adequacy fields (impacted_behaviors, tests_selected, why_sufficient, known_gaps). Never blocks admission in the first slice.",
			PolicyFile:         "policies/admission.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "admission.evidence_adequacy",
			Description:        "Advisory checks for evidence_adequacy.summary. Never blocks admission in the first slice.",
			PolicyFile:         "policies/admission.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "admission.escalation",
			Description:        "Verify-stage auto-escalation: light/standard cards touching high-risk governance surfaces must be declared as deep.",
			PolicyFile:         "policies/escalation.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusRuntimeBlocking,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(true),
			Profiles:           []string{"ci-standard", "ci-strict", "governed-deep"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "mutation_guard.verifier_read_only",
			Description:        "Read-only verifier: detects unintended file mutations during verification.",
			PolicyFile:         "policies/mutation-guard.yaml",
			RuntimeModule:      "internal/mutationguard",
			Status:             matrixStatusRuntimeBlocking,
			EnabledByDefault:   false,
			AdmissionAuthority: boolPtr(true),
			EnabledByFlags:     []string{"--mutation-guard", "--strict"},
			Profiles:           []string{"ci-strict", "governed-deep"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "context_floor.stale_ground",
			Description:        "Stale ground detection: withholds when source-of-truth is known stale.",
			PolicyFile:         "policies/stale-ground.yaml",
			RuntimeModule:      "internal/conformance",
			Status:             matrixStatusRuntimeBlocking,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(true),
			Profiles:           []string{"ci-standard", "ci-strict", "governed-deep"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "context_floor.tier_blocks",
			Description:        "Tier-specific context requirements (product_contract_refs, architecture_refs, decision_refs, test_matrix_refs).",
			PolicyFile:         "policies/context-floor.yaml",
			RuntimeModule:      "internal/conformance",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "contract_oracle.grep_rules",
			Description:        "Deterministic grep/import rules for forbidden dependencies and patterns. Off by default until a policy adds active rules.",
			PolicyFile:         "policies/contract-oracle.yaml",
			RuntimeModule:      "internal/contract",
			Status:             matrixStatusOffByDefault,
			EnabledByDefault:   false,
			AdmissionAuthority: boolPtr(false),
			EnabledByFlags:     []string{"--contract-oracles"},
			Profiles:           []string{"ci-strict", "governed-deep"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "pgv.suggestion",
			Description:        "Pre-gate validation advice. Advisory only; never overrides verify and never grants admission.",
			PolicyFile:         "policies/pgv.yaml",
			RuntimeModule:      "internal/pgv",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "federation.redacted_export",
			Description:        "Opt-in redacted export of anonymized failure predicates and metrics. Federation is off by default.",
			PolicyFile:         "policies/federation.yaml",
			RuntimeModule:      "internal/federation",
			Status:             matrixStatusOffByDefault,
			EnabledByDefault:   false,
			AdmissionAuthority: boolPtr(false),
			EnabledByFlags:     []string{"--opt-in"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "cost_budget.check",
			Description:        "Advisory cost budget check. Disabled by default and never affects admission.",
			PolicyFile:         "policies/cost-budget.yaml",
			RuntimeModule:      "internal/cost",
			Status:             matrixStatusOffByDefault,
			EnabledByDefault:   false,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "intake.classify",
			Description:        "Deterministic task intake tier classification (tiny/normal/high_risk).",
			PolicyFile:         "policies/intake.yaml",
			RuntimeModule:      "internal/intake",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "permissions.classify",
			Description:        "Permission intent classification and command-set rules.",
			PolicyFile:         "policies/permissions.yaml",
			RuntimeModule:      "internal/permissions",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "classifier.intent",
			Description:        "Deterministic command intent classifier (read_files, write_files, run_tests, etc.).",
			PolicyFile:         "policies/classifier.yaml",
			RuntimeModule:      "internal/classify",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "approval_risk.score",
			Description:        "Advisory approval risk scoring. Disabled by default.",
			PolicyFile:         "policies/approval-risk.yaml",
			RuntimeModule:      "internal/approvalrisk",
			Status:             matrixStatusOffByDefault,
			EnabledByDefault:   false,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "rollback.deep_required",
			Description:        "Deep tier requires rollback_policy declaration.",
			PolicyFile:         "policies/rollback.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusRuntimeBlocking,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(true),
			Profiles:           []string{"governed-deep"},
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "attribution.explain",
			Description:        "Advisory attribution metadata (provider, model, adapter, session).",
			RuntimeModule:      "internal/attribution",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "evidence.evidence_floor",
			Description:        "Authoritative description of the evidence floor per tier. Runtime enforcement is in internal/admission.",
			PolicyFile:         "policies/evidence.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "denominator.metric_fields",
			Description:        "Requires required_metric_fields (numerator/denominator/inclusion/exclusion/claim_boundary) for benchmark metrics.",
			PolicyFile:         "policies/denominator.yaml",
			RuntimeModule:      "internal/benchmark",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "ownership.fields",
			Description:        "Requires owner/accountable fields; withheld/ blocked requires next_owner and next_action.",
			PolicyFile:         "policies/ownership.yaml",
			RuntimeModule:      "internal/admission",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "cleanup.guidance",
			Description:        "Advisory cleanup guidance for generated harness state. Never auto-deletes completion card/templates/schemas/policies.",
			PolicyFile:         "policies/cleanup.yaml",
			RuntimeModule:      "internal/cli/clean.go",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "recovery.routing",
			Description:        "Advisory recovery routing table for withheld reasons.",
			PolicyFile:         "policies/recovery.yaml",
			RuntimeModule:      "internal/cli/recover.go",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "scanner.heuristics",
			Description:        "Deterministic static heuristics (curl|sh, rm -rf /, etc.). Report-only by default.",
			PolicyFile:         "policies/scanner.yaml",
			RuntimeModule:      "internal/scanner",
			Status:             matrixStatusOffByDefault,
			EnabledByDefault:   false,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "authority.classes",
			Description:        "Authority classes (agent_editable, agent_proposable_human_approved, human_only) for protected paths.",
			PolicyFile:         "policies/authority.yaml",
			RuntimeModule:      "internal/authority",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Source:             matrixSourceCurated,
		},
		{
			ID:                 "boundary.violation",
			Description:        "Path-glob + import-regex boundary enforcement (V1). Surfaces violations in `xh boundary check`; admission integration per profile is V1-safe advisory only.",
			PolicyFile:         "policies/boundaries.yaml",
			RuntimeModule:      "internal/boundary",
			Status:             matrixStatusAdvisory,
			EnabledByDefault:   true,
			AdmissionAuthority: boolPtr(false),
			Profiles:           []string{"light-local", "ci-standard", "ci-strict", "governed-deep"},
			Source:             matrixSourceCurated,
		},
	}
}

func boolPtr(v bool) *bool { return &v }

// buildMatrix returns a deterministic Matrix: rules sorted by ID.
func buildMatrix() Matrix {
	rules := curatedMatrix()
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})
	return Matrix{
		Version: 1,
		Rules:   rules,
	}
}

func handlePolicy(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: xh policy <matrix> [--json]")
		return ExitUsage
	}

	switch args[0] {
	case "matrix":
		return handlePolicyMatrix(args[1:], stdout, stderr)
	case "explain":
		return handlePolicyExplain(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: xh policy <matrix|explain> [--json]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown policy subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: xh policy <matrix|explain> [--json]")
		return ExitUsage
	}
}

func handlePolicyMatrix(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh policy matrix [--json]")
			return ExitUsage
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	matrix := buildMatrix()
	if jsonMode {
		data, err := json.MarshalIndent(matrix, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
		fmt.Fprintln(stdout, string(data))
		return ExitOK
	}

	renderPolicyMatrixText(&matrix, stdout)
	return ExitOK
}

func renderPolicyMatrixText(matrix *Matrix, w io.Writer) {
	WriteLine(w, "# x-harness Policy Matrix")
	WriteLine(w, "")
	WriteLine(w, "version: %d", matrix.Version)
	WriteLine(w, "rules: %d", len(matrix.Rules))
	WriteLine(w, "")

	// Group by status for the text rendering; deterministic.
	groups := map[string][]MatrixRule{}
	for _, rule := range matrix.Rules {
		groups[rule.Status] = append(groups[rule.Status], rule)
	}

	for _, status := range []string{
		matrixStatusRuntimeBlocking,
		matrixStatusAdvisory,
		matrixStatusExperimental,
		matrixStatusOffByDefault,
	} {
		rules := groups[status]
		if len(rules) == 0 {
			continue
		}
		WriteLine(w, "%s (%d):", status, len(rules))
		for _, rule := range rules {
			WriteLine(w, "  - %s", rule.ID)
			if rule.Description != "" {
				WriteLine(w, "    description: %s", rule.Description)
			}
			if rule.PolicyFile != "" {
				WriteLine(w, "    policy_file: %s", rule.PolicyFile)
			}
			if rule.RuntimeModule != "" {
				WriteLine(w, "    runtime_module: %s", rule.RuntimeModule)
			}
			WriteLine(w, "    enabled_by_default: %t", rule.EnabledByDefault)
			if rule.AdmissionAuthority != nil {
				WriteLine(w, "    admission_authority: %t", *rule.AdmissionAuthority)
			}
			if len(rule.Profiles) > 0 {
				WriteLine(w, "    profiles: %s", strings.Join(rule.Profiles, ", "))
			}
			if len(rule.EnabledByFlags) > 0 {
				WriteLine(w, "    enabled_by_flags: %s", strings.Join(rule.EnabledByFlags, ", "))
			}
			if len(rule.Fixtures) > 0 {
				WriteLine(w, "    fixtures: %s", strings.Join(rule.Fixtures, ", "))
			}
		}
		WriteLine(w, "")
	}
}

// handlePolicyExplain renders a single rule from the curated matrix in
// either text or JSON form. Slice 4 (P1-S4) replaces the earlier stub.
func handlePolicyExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode := false
	ruleID := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh policy explain <rule-id> [--json]")
			return ExitUsage
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			if ruleID != "" {
				fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
				return ExitUsage
			}
			ruleID = args[i]
		}
	}

	if ruleID == "" {
		fmt.Fprintln(stderr, "Error: rule-id is required")
		fmt.Fprintln(stderr, "usage: xh policy explain <rule-id> [--json]")
		fmt.Fprintln(stderr, "Tip: run `xh policy matrix` to list available rule IDs.")
		return ExitUsage
	}

	matrix := buildMatrix()
	var match *MatrixRule
	for i := range matrix.Rules {
		if matrix.Rules[i].ID == ruleID {
			match = &matrix.Rules[i]
			break
		}
	}
	if match == nil {
		fmt.Fprintf(stderr, "Error: rule %q not found in policy matrix.\n", ruleID)
		fmt.Fprintln(stderr, "Tip: run `xh policy matrix` to list available rule IDs.")
		return ExitUsage
	}

	if jsonMode {
		_ = WriteJSON(stdout, match)
		return ExitOK
	}

	renderPolicyExplainText(match, stdout)
	return ExitOK
}

func renderPolicyExplainText(rule *MatrixRule, w io.Writer) {
	WriteLine(w, "# x-harness Policy Rule")
	WriteLine(w, "")
	WriteLine(w, "id: %s", rule.ID)
	if rule.Description != "" {
		WriteLine(w, "description: %s", rule.Description)
	}
	if rule.PolicyFile != "" {
		WriteLine(w, "policy_file: %s", rule.PolicyFile)
	}
	if rule.RuntimeModule != "" {
		WriteLine(w, "runtime_module: %s", rule.RuntimeModule)
	}
	WriteLine(w, "status: %s", rule.Status)
	WriteLine(w, "enabled_by_default: %t", rule.EnabledByDefault)
	if rule.AdmissionAuthority != nil {
		WriteLine(w, "admission_authority: %t", *rule.AdmissionAuthority)
	}
	if len(rule.Profiles) > 0 {
		WriteLine(w, "profiles: %s", strings.Join(rule.Profiles, ", "))
	}
	if len(rule.EnabledByFlags) > 0 {
		WriteLine(w, "enabled_by_flags: %s", strings.Join(rule.EnabledByFlags, ", "))
	}
	if len(rule.Fixtures) > 0 {
		WriteLine(w, "fixtures: %s", strings.Join(rule.Fixtures, ", "))
	}
	WriteLine(w, "source: %s", rule.Source)
}
