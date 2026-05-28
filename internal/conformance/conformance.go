package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
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
	cardPath := filepath.Join(root, "examples", "golden", "success-light", "completion-card.yaml")
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
	cardPath := filepath.Join(root, "examples", "golden", "blocked-missing-evidence", "completion-card.yaml")
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
