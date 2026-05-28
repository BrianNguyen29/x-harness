package doctor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/components"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

// Check is a single health check result.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

// Report is the doctor health report.
type Report struct {
	Healthy      bool     `json:"healthy"`
	PresentCount int      `json:"present_count"`
	MissingCount int      `json:"missing_count"`
	Present      []string `json:"present"`
	Missing      []string `json:"missing"`
	Checks       []Check  `json:"checks"`
	Notes        []string `json:"notes"`
}

// Run performs health checks against the given root directory.
func Run(root string) *Report {
	report := &Report{
		Healthy: true,
		Present: []string{},
		Missing: []string{},
		Checks:  []Check{},
		Notes:   []string{},
	}

	if root == "" {
		report.Checks = append(report.Checks, Check{Name: "root_exists", Status: "failed", Note: "root path is empty"})
		report.Healthy = false
		report.Missing = append(report.Missing, "root")
		report.MissingCount = 1
		return report
	}

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		report.Checks = append(report.Checks, Check{Name: "root_exists", Status: "failed", Note: "root path does not exist or is not a directory"})
		report.Healthy = false
		report.Missing = append(report.Missing, root)
		report.MissingCount = 1
		return report
	}

	checkCriticalAssets(report, root)
	checkSchemas(report, root)
	checkPolicies(report, root)
	checkAgentsContext(report, root)
	checkCIWorkflow(report, root)
	checkTierLabels(report, root)
	checkComponentRegistry(report, root)

	report.PresentCount = len(report.Present)
	report.MissingCount = len(report.Missing)

	return report
}

func checkCriticalAssets(report *Report, root string) {
	assets := []struct {
		path string
		name string
	}{
		{filepath.Join(root, "AGENTS.md"), "AGENTS.md"},
		{filepath.Join(root, "X_HARNESS.md"), "X_HARNESS.md"},
		{filepath.Join(root, "policies"), "policies/"},
		{filepath.Join(root, "schemas"), "schemas/"},
		{filepath.Join(root, "templates"), "templates/"},
		{filepath.Join(root, "examples", "golden"), "examples/golden/"},
		{filepath.Join(root, "policies", "mutation-guard.yaml"), "policies/mutation-guard.yaml"},
		{filepath.Join(root, ".github", "workflows", "x-harness-verify.yml"), ".github/workflows/x-harness-verify.yml"},
	}

	for _, asset := range assets {
		if _, err := os.Stat(asset.path); err == nil {
			report.Present = append(report.Present, asset.name)
		} else {
			report.Missing = append(report.Missing, asset.name)
			report.Healthy = false
		}
	}

	if len(report.Missing) > 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "critical_assets",
			Status: "failed",
			Note:   "missing: " + strings.Join(report.Missing, ", "),
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "critical_assets",
			Status: "passed",
		})
	}
}

func checkSchemas(report *Report, root string) {
	schemaDir := filepath.Join(root, "schemas")
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "schemas_compile",
			Status: "failed",
			Note:   err.Error(),
		})
		report.Healthy = false
		return
	}

	compiled := 0
	failed := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(schemaDir, entry.Name())
		_, err := schema.Compile(path)
		if err != nil {
			failed++
			report.Checks = append(report.Checks, Check{
				Name:   "schema_compile_" + entry.Name(),
				Status: "failed",
				Note:   err.Error(),
			})
			report.Healthy = false
		} else {
			compiled++
		}
	}

	if failed == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "schemas_compile",
			Status: "passed",
			Note:   "all schemas compiled",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "schemas_compile",
			Status: "failed",
			Note:   "some schemas failed to compile",
		})
	}
}

func checkPolicies(report *Report, root string) {
	policyDir := filepath.Join(root, "policies")
	entries, err := os.ReadDir(policyDir)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "policies_parse",
			Status: "failed",
			Note:   err.Error(),
		})
		report.Healthy = false
		return
	}

	parsed := 0
	failed := 0
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml") {
			continue
		}
		path := filepath.Join(policyDir, entry.Name())
		var v any
		if err := loader.LoadDocument(path, &v); err != nil {
			failed++
			report.Checks = append(report.Checks, Check{
				Name:   "policy_parse_" + entry.Name(),
				Status: "failed",
				Note:   err.Error(),
			})
			report.Healthy = false
		} else {
			parsed++
		}
	}

	if failed == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "policies_parse",
			Status: "passed",
			Note:   "all policies parsed",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "policies_parse",
			Status: "failed",
			Note:   "some policies failed to parse",
		})
	}
}

func checkAgentsContext(report *Report, root string) {
	path := filepath.Join(root, "AGENTS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "agents_managed_context",
			Status: "failed",
			Note:   "AGENTS.md not readable",
		})
		report.Healthy = false
		return
	}

	content := string(b)
	if strings.Contains(content, "BEGIN X-HARNESS MANAGED CONTEXT") {
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
		report.Healthy = false
	}
}

func checkCIWorkflow(report *Report, root string) {
	path := filepath.Join(root, ".github", "workflows", "x-harness-verify.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "ci_workflow",
			Status: "failed",
			Note:   "CI workflow not readable",
		})
		report.Healthy = false
		return
	}

	content := string(b)
	missing := []string{}
	if !strings.Contains(content, "doctor") {
		missing = append(missing, "doctor")
	}
	if !strings.Contains(content, "verify") {
		missing = append(missing, "verify")
	}
	if !strings.Contains(content, "examples") {
		missing = append(missing, "examples")
	}

	if len(missing) == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "ci_workflow",
			Status: "passed",
			Note:   "verify, doctor, and examples gates present",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "ci_workflow",
			Status: "failed",
			Note:   "missing gates: " + strings.Join(missing, ", "),
		})
		report.Healthy = false
	}
}

var invalidTierLabels = []string{"small", "medium", "large"}

var allowedTierReferencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)do not use\b.*\b(small|medium|large)\b`),
	regexp.MustCompile(`(?i)forbidden active aliases`),
	regexp.MustCompile(`(?i)invalid tier labels`),
	regexp.MustCompile(`(?i)risk.*\b(medium|large|small)\b`),
	regexp.MustCompile(`(?i)confidence.*\b(medium|large|small)\b`),
	regexp.MustCompile(`(?i)priority.*\b(medium|large|small)\b`),
	regexp.MustCompile(`(?i)context[_\ -]?class.*\b(medium|large|small)\b`),
	regexp.MustCompile(`(?i)default_token_impact.*\b(medium|large|small)\b`),
	regexp.MustCompile(`(?i)runtime_impact.*\b(medium|large|small)\b`),
	regexp.MustCompile(`(?i)severity.*\b(medium|large|small)\b`),
}

var tierScanDirs = []string{"docs", "templates", "adapters", "packages/cli/src", "internal", "cmd"}
var tierScanExts = map[string]bool{".md": true, ".mdc": true, ".ts": true, ".json": true, ".go": true}

func isAllowedTierReference(line string) bool {
	for _, re := range allowedTierReferencePatterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func checkTierLabels(report *Report, root string) {
	excludedFiles := map[string]bool{
		filepath.Join(root, "docs", "RUNTIME_CONTRACT.md"):              true,
		filepath.Join(root, "packages", "cli", "src", "commands", "doctor.ts"): true,
		filepath.Join(root, "packages", "cli", "src", "core", "metrics.ts"):    true,
		filepath.Join(root, "packages", "cli", "src", "core", "context.ts"):    true,
		filepath.Join(root, "packages", "cli", "src", "core", "contract.ts"):   true,
		filepath.Join(root, "packages", "cli", "src", "core", "recovery.ts"):   true,
		filepath.Join(root, "packages", "cli", "src", "core", "attribution.ts"): true,
		filepath.Join(root, "internal", "attribution", "attribution.go"):       true,
		filepath.Join(root, "internal", "prediction", "prediction.go"):         true,
		filepath.Join(root, "internal", "doctor", "doctor.go"):                 true,
		filepath.Join(root, "internal", "scanner", "scanner.go"):               true,
		filepath.Join(root, "internal", "cli", "scan.go"):                     true,
	}

	labelRe := regexp.MustCompile(`(?i)\b(small|medium|large)\b`)
	ok := true
	notes := []string{}

	for _, relDir := range tierScanDirs {
		dir := filepath.Join(root, relDir)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !tierScanExts[filepath.Ext(path)] {
				return nil
			}
			if strings.HasSuffix(filepath.Base(path), "_test.go") {
				return nil
			}
			if excludedFiles[path] {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if isAllowedTierReference(line) {
					continue
				}
				matches := labelRe.FindAllStringSubmatchIndex(line, -1)
				for _, m := range matches {
					if len(m) >= 4 {
						label := strings.ToLower(line[m[2]:m[3]])
						rel, _ := filepath.Rel(root, path)
						notes = append(notes, "invalid tier label \""+label+"\" in "+rel)
						ok = false
					}
				}
			}
			return nil
		})
	}

	if ok {
		report.Checks = append(report.Checks, Check{
			Name:   "tier_labels",
			Status: "passed",
			Note:   "no invalid tier labels found (small/medium/large)",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "tier_labels",
			Status: "failed",
			Note:   strings.Join(notes, "; "),
		})
		report.Healthy = false
	}
}

func checkComponentRegistry(report *Report, root string) {
	result, err := components.ValidateRegistry(root)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "component_registry",
			Status: "failed",
			Note:   err.Error(),
		})
		report.Healthy = false
		return
	}

	if result.OK {
		report.Checks = append(report.Checks, Check{
			Name:   "component_registry",
			Status: "passed",
			Note:   fmt.Sprintf("%d component(s); protected paths %d/%d covered", result.ComponentCount, result.ProtectedPathsCovered, result.ProtectedPathsChecked),
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "component_registry",
			Status: "failed",
			Note:   strings.Join(result.Errors, "; "),
		})
		report.Healthy = false
	}
}
