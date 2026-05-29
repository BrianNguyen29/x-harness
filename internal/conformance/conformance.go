package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/mutationguard"
	"github.com/BrianNguyen29/x-harness/internal/scanner"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"github.com/BrianNguyen29/x-harness/internal/worktree"
)

// Check is a single conformance check result.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

// Report is the conformance run report.
type Report struct {
	Profile string  `json:"profile"`
	OK      bool    `json:"ok"`
	Checks  []Check `json:"checks"`
}

// RunMinimal performs the minimal conformance profile checks.
func RunMinimal(root string) *Report {
	report := &Report{
		Profile: "minimal",
		OK:      true,
		Checks:  []Check{},
	}

	checkCriticalFiles(report, root)
	checkSchemasCompile(report, root)
	checkPoliciesParse(report, root)
	checkAgentsManagedContext(report, root)
	checkGoldenSuccessLight(report, root)
	checkGoldenBlockedMissingEvidence(report, root)
	checkDenominatorContract(report, root)

	for _, c := range report.Checks {
		if c.Status != "passed" {
			report.OK = false
			break
		}
	}

	return report
}

func checkCriticalFiles(report *Report, root string) {
	files := []string{
		"AGENTS.md",
		"X_HARNESS.md",
		filepath.Join("policies", "admission.yaml"),
		filepath.Join("schemas", "completion-card.schema.json"),
	}

	missing := []string{}
	for _, f := range files {
		path := filepath.Join(root, f)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, f)
		}
	}

	if len(missing) == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "critical_files_exist",
			Status: "passed",
			Note:   "all critical files present",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "critical_files_exist",
			Status: "failed",
			Note:   "missing: " + strings.Join(missing, ", "),
		})
		report.OK = false
	}
}

func checkSchemasCompile(report *Report, root string) {
	schemaDir := filepath.Join(root, "schemas")
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "schemas_compile",
			Status: "failed",
			Note:   err.Error(),
		})
		report.OK = false
		return
	}

	compiled := 0
	failed := 0
	var failNotes []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(schemaDir, entry.Name())
		_, err := schema.Compile(path)
		if err != nil {
			failed++
			failNotes = append(failNotes, fmt.Sprintf("%s: %v", entry.Name(), err))
			report.OK = false
		} else {
			compiled++
		}
	}

	if failed == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "schemas_compile",
			Status: "passed",
			Note:   fmt.Sprintf("%d schema(s) compiled", compiled),
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "schemas_compile",
			Status: "failed",
			Note:   strings.Join(failNotes, "; "),
		})
	}
}

func checkPoliciesParse(report *Report, root string) {
	policyDir := filepath.Join(root, "policies")
	entries, err := os.ReadDir(policyDir)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "policies_parse",
			Status: "failed",
			Note:   err.Error(),
		})
		report.OK = false
		return
	}

	parsed := 0
	failed := 0
	var failNotes []string
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml") {
			continue
		}
		path := filepath.Join(policyDir, entry.Name())
		var v any
		if err := loader.LoadDocument(path, &v); err != nil {
			failed++
			failNotes = append(failNotes, fmt.Sprintf("%s: %v", entry.Name(), err))
			report.OK = false
		} else {
			parsed++
		}
	}

	if failed == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "policies_parse",
			Status: "passed",
			Note:   fmt.Sprintf("%d policy file(s) parsed", parsed),
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "policies_parse",
			Status: "failed",
			Note:   strings.Join(failNotes, "; "),
		})
	}
}

func checkAgentsManagedContext(report *Report, root string) {
	path := filepath.Join(root, "AGENTS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "agents_managed_context",
			Status: "failed",
			Note:   "AGENTS.md not readable",
		})
		report.OK = false
		return
	}

	if strings.Contains(string(b), "BEGIN X-HARNESS MANAGED CONTEXT") {
		report.Checks = append(report.Checks, Check{
			Name:   "agents_managed_context",
			Status: "passed",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "agents_managed_context",
			Status: "failed",
			Note:   "managed context block not found",
		})
		report.OK = false
	}
}

func checkGoldenSuccessLight(report *Report, root string) {
	cardPath := filepath.Join(root, "examples", "golden", "regression", "success-light", "completion-card.yaml")
	outcome, acceptance, note := checkGoldenCard(root, cardPath)

	if acceptance == "accepted" {
		report.Checks = append(report.Checks, Check{
			Name:   "golden_success_light",
			Status: "passed",
			Note:   fmt.Sprintf("outcome=%s acceptance=%s", outcome, acceptance),
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "golden_success_light",
			Status: "failed",
			Note:   note,
		})
		report.OK = false
	}
}

func checkGoldenBlockedMissingEvidence(report *Report, root string) {
	cardPath := filepath.Join(root, "examples", "golden", "regression", "blocked-missing-evidence", "completion-card.yaml")
	outcome, acceptance, note := checkGoldenCard(root, cardPath)

	if acceptance == "withheld" {
		report.Checks = append(report.Checks, Check{
			Name:   "golden_blocked_missing_evidence",
			Status: "passed",
			Note:   fmt.Sprintf("outcome=%s acceptance=%s", outcome, acceptance),
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "golden_blocked_missing_evidence",
			Status: "failed",
			Note:   note,
		})
		report.OK = false
	}
}

func checkDenominatorContract(report *Report, root string) {
	schemaPath := filepath.Join(root, "schemas", "report.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "denominator_contract",
			Status: "failed",
			Note:   "report schema compile error: " + err.Error(),
		})
		report.OK = false
		return
	}

	sample := map[string]any{
		"card_id": "test",
		"task_id": "test",
		"tier":    "standard",
		"metrics": map[string]any{
			"verification_strength": map[string]any{
				"command_evidence_count": 0,
				"oracle_kinds":           []any{},
				"untested_regions_count": 0,
				"remaining_risks_count":  0,
			},
			"state_consistency": map[string]any{
				"owner_present":           true,
				"accountable_present":     true,
				"files_changed_present":   true,
				"admission_mapping_valid": true,
			},
			"recovery_ability": map[string]any{
				"blocked_has_next_action": true,
				"blocked_has_owner":       true,
				"recovery_route_present":  true,
			},
			"replayability": map[string]any{
				"completion_card_present": true,
				"input_card_hash_present": true,
				"policy_hash_present":     true,
			},
			"cost": map[string]any{
				"default_context_class": "medium",
				"verify_runtime_ms":     0,
			},
			"verify_event_success_rate": map[string]any{
				"numerator":      1,
				"denominator":    1,
				"unit":           "verify_event",
				"not_task_level": true,
			},
			"task_completion_coverage": map[string]any{
				"status": "not_computable",
				"reason": "missing_aligned_task_denominator",
			},
			"withheld_rate": map[string]any{
				"numerator":      0,
				"denominator":    1,
				"unit":           "verify_event",
				"not_task_level": true,
			},
		},
		"admission": map[string]any{
			"outcome":           "success",
			"acceptance_status": "accepted",
			"errors":            []any{},
			"notes":             []any{},
		},
		"verify_event_accounting": map[string]any{
			"cards_analyzed": 1,
			"note":           "test",
		},
		"task_lifecycle_accounting": map[string]any{
			"admitted": 1,
			"withheld": 0,
			"note":     "test",
		},
		"admission_accounting": map[string]any{
			"accepted":       1,
			"total_analyzed": 1,
			"note":           "test",
		},
		"withheld_accounting": map[string]any{
			"failed":  0,
			"blocked": 0,
			"skipped": 0,
			"timeout": 0,
			"error":   0,
			"note":    "test",
		},
		"unknown_or_unlinked_events": map[string]any{
			"count": 0,
			"note":  "test",
		},
		"denominator_warning": "test",
	}

	if err := v.Validate(sample); err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "denominator_contract",
			Status: "failed",
			Note:   "sample report with denominator contract failed schema validation: " + err.Error(),
		})
		report.OK = false
		return
	}

	report.Checks = append(report.Checks, Check{
		Name:   "denominator_contract",
		Status: "passed",
		Note:   "report schema validates denominator-safe rate metrics",
	})
}

// RunStrict performs the strict conformance profile checks.
func RunStrict(root string) *Report {
	// Run minimal first
	minimalReport := RunMinimal(root)

	report := &Report{
		Profile: "strict",
		OK:      minimalReport.OK,
		Checks:  append([]Check{}, minimalReport.Checks...),
	}

	if !report.OK {
		return report
	}

	// Take before snapshot for mutation guard
	before, beforeErr := mutationguard.TakeSnapshot(root)

	// Run strict-specific checks
	strictChecks := []Check{}
	strictChecks = append(strictChecks, checkScannerHighSeverity(root))
	strictChecks = append(strictChecks, checkWorktreeMetadata(root))

	// Deferred checks as not_implemented (advisory, non-blocking)
	strictChecks = append(strictChecks, Check{
		Name:   "adapter_doctor_no_drift",
		Status: "not_implemented",
		Note:   "deferred to future slice",
	})
	strictChecks = append(strictChecks, Check{
		Name:   "context_gc_no_stale_drift",
		Status: "not_implemented",
		Note:   "deferred to future slice",
	})
	strictChecks = append(strictChecks, Check{
		Name:   "approval_receipt_for_high_risk",
		Status: "not_implemented",
		Note:   "deferred to future slice",
	})
	strictChecks = append(strictChecks, Check{
		Name:   "regression_suite_passed",
		Status: "not_implemented",
		Note:   "deferred to future slice",
	})
	strictChecks = append(strictChecks, Check{
		Name:   "adversarial_suite_passed",
		Status: "not_implemented",
		Note:   "deferred to future slice",
	})

	// Mutation guard evaluation
	if beforeErr != nil {
		strictChecks = append([]Check{{
			Name:   "mutation_guard_verified",
			Status: "failed",
			Note:   "before snapshot failed: " + beforeErr.Error(),
		}}, strictChecks...)
	} else {
		after, afterErr := mutationguard.TakeSnapshot(root)
		if afterErr != nil {
			strictChecks = append([]Check{{
				Name:   "mutation_guard_verified",
				Status: "failed",
				Note:   "after snapshot failed: " + afterErr.Error(),
			}}, strictChecks...)
		} else {
			deltas := mutationguard.Compare(before, after)
			unexpected := mutationguard.FilterUnexpected(deltas)
			if len(unexpected) > 0 {
				var paths []string
				for _, d := range unexpected {
					paths = append(paths, d.Path)
				}
				strictChecks = append([]Check{{
					Name:   "mutation_guard_verified",
					Status: "failed",
					Note:   "unexpected delta: " + strings.Join(paths, ", "),
				}}, strictChecks...)
			} else {
				strictChecks = append([]Check{{
					Name:   "mutation_guard_verified",
					Status: "passed",
					Note:   "no unexpected changes detected",
				}}, strictChecks...)
			}
		}
	}

	report.Checks = append(report.Checks, strictChecks...)

	// Recalculate OK
	report.OK = true
	for _, c := range report.Checks {
		if c.Status == "failed" {
			report.OK = false
			break
		}
	}

	return report
}

func checkScannerHighSeverity(root string) Check {
	rules := scanner.DefaultRules()

	// Slice 1 scope: scan adapters, skills, and templates directories
	var paths []string
	for _, dir := range []string{"adapters", "skills", "templates"} {
		p := filepath.Join(root, dir)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			paths = append(paths, p)
		}
	}

	if len(paths) == 0 {
		return Check{
			Name:   "scanner_high_severity_clear",
			Status: "passed",
			Note:   "no scan directories present",
		}
	}

	result, err := scanner.Scan(rules, paths)
	if err != nil {
		return Check{
			Name:   "scanner_high_severity_clear",
			Status: "failed",
			Note:   "scan error: " + err.Error(),
		}
	}

	var highFindings, mediumFindings []scanner.Finding
	for _, f := range result.Findings {
		switch f.Severity {
		case string(scanner.SeverityHigh):
			highFindings = append(highFindings, f)
		case string(scanner.SeverityMedium):
			mediumFindings = append(mediumFindings, f)
		}
	}

	if len(highFindings) > 0 {
		var notes []string
		for _, f := range highFindings {
			notes = append(notes, fmt.Sprintf("%s in %s:%d (%s)", f.RuleID, f.File, f.Line, f.Snippet))
		}
		return Check{
			Name:   "scanner_high_severity_clear",
			Status: "failed",
			Note:   strings.Join(notes, "; "),
		}
	}

	if len(mediumFindings) > 0 {
		var notes []string
		for _, f := range mediumFindings {
			notes = append(notes, fmt.Sprintf("%s in %s:%d", f.RuleID, f.File, f.Line))
		}
		return Check{
			Name:   "scanner_high_severity_clear",
			Status: "advisory",
			Note:   strings.Join(notes, "; "),
		}
	}

	return Check{
		Name:   "scanner_high_severity_clear",
		Status: "passed",
		Note:   fmt.Sprintf("no high or medium severity findings (%d files scanned)", result.FilesScanned),
	}
}

func checkWorktreeMetadata(root string) Check {
	info := worktree.CollectInfo(root)
	if info == nil {
		return Check{
			Name:   "worktree_metadata_valid",
			Status: "failed",
			Note:   "not a git workspace or git unavailable",
		}
	}
	return Check{
		Name:   "worktree_metadata_valid",
		Status: "passed",
		Note:   fmt.Sprintf("root=%s branch=%s commit=%s", info.Root, info.Branch, info.Commit),
	}
}

func checkGoldenCard(root, cardPath string) (outcome, acceptance, note string) {
	schemaPath := filepath.Join(root, "schemas", "completion-card.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		return "", "", fmt.Sprintf("schema compile error: %v", err)
	}

	var doc map[string]any
	if err := loader.LoadDocument(cardPath, &doc); err != nil {
		return "", "", fmt.Sprintf("load error: %v", err)
	}

	schemaErr := v.Validate(doc)
	if schemaErr != nil {
		return "failed", "withheld", fmt.Sprintf("schema invalid: outcome=failed acceptance=withheld")
	}

	result := admission.Run(doc, false)
	return result.Outcome, result.AcceptanceStatus, fmt.Sprintf("outcome=%s acceptance=%s errors=%v", result.Outcome, result.AcceptanceStatus, result.Errors)
}
