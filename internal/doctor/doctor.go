package doctor

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/components"
	"github.com/BrianNguyen29/x-harness/internal/contextcheck"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"gopkg.in/yaml.v3"
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

// Options configures doctor.Run behavior.
type Options struct {
	Staleness bool
	Overclaim bool
	Context   bool
}

// Run performs health checks against the given root directory.
func Run(root string) *Report {
	return RunWithOptions(root, Options{})
}

func RunWithOptions(root string, opts Options) *Report {
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

	profile := detectInstalledProfile(root)

	checkCriticalAssets(report, root, profile)
	checkSchemas(report, root, profile)
	checkPolicies(report, root)
	checkAgentsContext(report, root)
	if opts.Staleness {
		checkAgentsContextStaleness(report, root)
	}
	if opts.Overclaim {
		checkOverclaimPhrases(report, root)
	}
	if opts.Context {
		checkContextRefs(report, root)
	}
	checkManagedBlocksRegistry(report, root, profile)
	checkCIWorkflow(report, root, profile)
	checkTierLabels(report, root)
	checkComponentRegistry(report, root, profile)
	checkManifest(report, root)

	report.PresentCount = len(report.Present)
	report.MissingCount = len(report.Missing)

	return report
}

// detectInstalledProfile reads .x-harness/manifest.yaml and returns the
// installed profile name (e.g. "minimal") when the manifest is present and
// parses successfully. Returns "" when no manifest is installed, the manifest
// is invalid, or the profile field is empty.
func detectInstalledProfile(root string) string {
	manifestFile := filepath.Join(root, ".x-harness", "manifest.yaml")
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		return ""
	}
	var m struct {
		Profile string `yaml:"profile"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return ""
	}
	return m.Profile
}

func checkCriticalAssets(report *Report, root string, profile string) {
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

	// Minimal profile omits full-only assets (schemas, examples/golden,
	// mutation-guard, CI workflow). Require only the minimal core set.
	if profile == "minimal" {
		assets = []struct {
			path string
			name string
		}{
			{filepath.Join(root, "AGENTS.md"), "AGENTS.md"},
			{filepath.Join(root, "X_HARNESS.md"), "X_HARNESS.md"},
			{filepath.Join(root, "policies"), "policies/"},
			{filepath.Join(root, "templates"), "templates/"},
			{filepath.Join(root, "docs"), "docs/"},
		}
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

func checkSchemas(report *Report, root string, profile string) {
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

func checkCIWorkflow(report *Report, root string, profile string) {
	if profile == "minimal" {
		report.Checks = append(report.Checks, Check{
			Name:   "ci_workflow",
			Status: "skipped",
			Note:   "minimal profile: CI workflow not required",
		})
		return
	}
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
		filepath.Join(root, "docs", "RUNTIME_CONTRACT.md"):                      true,
		filepath.Join(root, "docs", "CONFORMANCE_STRICT_PROFILE.md"):            true,
		filepath.Join(root, "packages", "cli", "src", "commands", "doctor.ts"):  true,
		filepath.Join(root, "packages", "cli", "src", "core", "metrics.ts"):     true,
		filepath.Join(root, "packages", "cli", "src", "core", "context.ts"):     true,
		filepath.Join(root, "packages", "cli", "src", "core", "contract.ts"):    true,
		filepath.Join(root, "packages", "cli", "src", "core", "recovery.ts"):    true,
		filepath.Join(root, "packages", "cli", "src", "core", "attribution.ts"): true,
		filepath.Join(root, "internal", "attribution", "attribution.go"):        true,
		filepath.Join(root, "internal", "prediction", "prediction.go"):          true,
		filepath.Join(root, "internal", "doctor", "doctor.go"):                  true,
		filepath.Join(root, "internal", "scanner", "scanner.go"):                true,
		filepath.Join(root, "internal", "cli", "scan.go"):                       true,
		filepath.Join(root, "internal", "conformance", "conformance.go"):        true,
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

func checkComponentRegistry(report *Report, root string, profile string) {
	if profile == "minimal" {
		report.Checks = append(report.Checks, Check{
			Name:   "component_registry",
			Status: "skipped",
			Note:   "minimal profile: components registry not required",
		})
		return
	}
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

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data)), nil
}

func checkManifest(report *Report, root string) {
	manifestFile := filepath.Join(root, ".x-harness", "manifest.yaml")
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "installed_profile",
			Status: "passed",
			Note:   "no manifest installed",
		})
		return
	}

	var m struct {
		Profile string `yaml:"profile"`
		Entries []struct {
			Path string `yaml:"path"`
			Hash string `yaml:"hash"`
		} `yaml:"entries"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "installed_profile",
			Status: "failed",
			Note:   "invalid manifest: " + err.Error(),
		})
		report.Healthy = false
		return
	}

	missing := []string{}
	modified := []string{}
	for _, entry := range m.Entries {
		entryPath := filepath.Join(root, filepath.FromSlash(entry.Path))
		if _, err := os.Stat(entryPath); err != nil {
			missing = append(missing, entry.Path)
			continue
		}
		hash, err := fileHash(entryPath)
		if err != nil || hash != entry.Hash {
			modified = append(modified, entry.Path)
		}
	}

	if len(missing) > 0 || len(modified) > 0 {
		notes := []string{}
		if len(missing) > 0 {
			notes = append(notes, "missing: "+strings.Join(missing, ", "))
		}
		if len(modified) > 0 {
			notes = append(notes, "modified: "+strings.Join(modified, ", "))
		}
		report.Checks = append(report.Checks, Check{
			Name:   "installed_profile",
			Status: "failed",
			Note:   strings.Join(notes, "; "),
		})
		report.Healthy = false
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "installed_profile",
			Status: "passed",
			Note:   "profile: " + m.Profile,
		})
	}
}

func checkAgentsContextStaleness(report *Report, root string) {
	path := filepath.Join(root, "AGENTS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "agents_context_staleness",
			Status: "failed",
			Note:   "AGENTS.md not readable",
		})
		report.Healthy = false
		return
	}

	valid, note := contextcheck.ValidateManagedBlock(string(b))
	if valid {
		report.Checks = append(report.Checks, Check{
			Name:   "agents_context_staleness",
			Status: "passed",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "agents_context_staleness",
			Status: "failed",
			Note:   note,
		})
		report.Healthy = false
	}
}

func checkManagedBlocksRegistry(report *Report, root string, profile string) {
	if profile == "minimal" {
		report.Checks = append(report.Checks, Check{
			Name:   "managed_blocks_registry",
			Status: "skipped",
			Note:   "minimal profile: managed-blocks registry not required",
		})
		return
	}
	failures, err := contextcheck.ValidateRegistry(root)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "managed_blocks_registry",
			Status: "failed",
			Note:   err.Error(),
		})
		report.Healthy = false
		return
	}

	if len(failures) == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "managed_blocks_registry",
			Status: "passed",
			Note:   "all registered managed blocks present and valid",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "managed_blocks_registry",
			Status: "failed",
			Note:   strings.Join(failures, "; "),
		})
		report.Healthy = false
	}
}

// Overclaim phrase patterns (case-insensitive)
var overclaimPhrases = []string{
	"guarantees correctness",
	"prevents all bugs",
	"solves hallucination",
	"proves production reliability",
	"replaces CI",
	"fully autonomous",
	"production validated",
	"benchmark success rate",
	"TypeScript-first",
}

// Exclusion patterns for allowed contexts
var overclaimExclusions = []*regexp.Regexp{
	// Negated disclaimers: "does not replace CI", "does not guarantee"
	regexp.MustCompile(`(?i)\bdoes not\b.*\b(replace|guarantee|correctness|production|benchmark|autonomous)\b`),
	// Historical/compatibility context for TypeScript-first
	regexp.MustCompile(`(?i)\b(historical|compatibility|compatibility note)\b.*\bTypeScript-first\b`),
	regexp.MustCompile(`(?i)\bTypeScript-first\b.*\b(historical|compatibility|compatibility note)\b`),
	// References to the overclaim as something to avoid/detect
	regexp.MustCompile(`(?i)\boverclaim\b`),
	regexp.MustCompile(`(?i)\bdetect.*overclaim\b`),
	regexp.MustCompile(`(?i)\bavoid.*overclaim\b`),
}

var overclaimScanDirs = []string{"."}
var overclaimScanExts = map[string]bool{".md": true, ".mdc": true}
var overclaimExcludedDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
	"coverage":     true,
	".x-harness":   true,
	"vendor":       true,
}

// overclaimExcludedFilenames matches files that are reference/changelog documents
// listing overclaim phrases to detect, rather than docs using those phrases.
var overclaimExcludedFilenames = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ROADMAP`),
	regexp.MustCompile(`(?i)IMPROVEMENT_PLAN`),
	regexp.MustCompile(`(?i)CHANGELOG`),
}

func isOverclaimExcludedFile(path string) bool {
	basename := filepath.Base(path)
	for _, re := range overclaimExcludedFilenames {
		if re.MatchString(basename) {
			return true
		}
	}
	return false
}

func isAllowedOverclaimContext(line string) bool {
	for _, re := range overclaimExclusions {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func checkOverclaimPhrases(report *Report, root string) {
	overclaimRe := regexp.MustCompile("(?i)" + strings.Join(overclaimPhrases, "|"))
	ok := true
	notes := []string{}

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip excluded directories
		if info.IsDir() {
			basename := filepath.Base(path)
			if overclaimExcludedDirs[basename] {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan relevant file types
		if !overclaimScanExts[filepath.Ext(path)] {
			return nil
		}

		// Skip roadmap/changelog reference docs that list phrases to detect
		if isOverclaimExcludedFile(path) {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Skip allowed context lines (disclaimers, historical references)
			if isAllowedOverclaimContext(line) {
				continue
			}

			matches := overclaimRe.FindAllStringSubmatchIndex(line, -1)
			for _, m := range matches {
				if len(m) >= 2 {
					phrase := line
					rel, _ := filepath.Rel(root, path)
					notes = append(notes, fmt.Sprintf("%s:%d: %s", rel, lineNum, phrase))
					ok = false
				}
			}
		}
		return nil
	})

	if ok {
		report.Checks = append(report.Checks, Check{
			Name:   "overclaim_phrases",
			Status: "passed",
			Note:   "no overclaim phrases found",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "overclaim_phrases",
			Status: "failed",
			Note:   strings.Join(notes, "; "),
		})
		report.Healthy = false
	}
}

func checkContextRefs(report *Report, root string) {
	cardsScanned := 0
	cardsWithAlignment := 0
	missingFiles := []string{}
	cardsSkipped := 0
	cardsUnreadable := 0

	// Define scan roots: examples and skills (if they exist)
	scanRoots := []string{}
	examplesDir := filepath.Join(root, "examples")
	if _, err := os.Stat(examplesDir); err == nil {
		scanRoots = append(scanRoots, examplesDir)
	}
	skillsDir := filepath.Join(root, "skills")
	if _, err := os.Stat(skillsDir); err == nil {
		scanRoots = append(scanRoots, skillsDir)
	}

	matches := []string{}

	for _, scanRoot := range scanRoots {
		err := filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if d.IsDir() {
				basename := d.Name()
				if overclaimExcludedDirs[basename] {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() == "completion-card.yaml" {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			report.Checks = append(report.Checks, Check{
				Name:   "context_refs",
				Status: "failed",
				Note:   "failed to scan completion cards: " + err.Error(),
			})
			report.Healthy = false
			return
		}
	}

	for _, cardPath := range matches {
		cardsScanned++
		data, err := os.ReadFile(cardPath)
		if err != nil {
			cardsUnreadable++
			continue
		}

		var doc map[string]any
		if err := yaml.Unmarshal(data, &doc); err != nil {
			cardsUnreadable++
			continue
		}

		ctxAlign, ok := doc["context_alignment"].(map[string]any)
		if !ok {
			cardsSkipped++
			continue
		}

		cardsWithAlignment++
		cardDir := filepath.Dir(cardPath)

		// Check ref arrays
		refArrays := []string{"product_contract_refs", "architecture_refs", "decision_refs", "test_matrix_refs"}
		for _, refKey := range refArrays {
			refs, ok := ctxAlign[refKey].([]any)
			if !ok {
				continue
			}
			for _, r := range refs {
				refStr, ok := r.(string)
				if !ok {
					continue
				}
				refPath := admission.StripAnchor(refStr)
				if !admission.FileExists(refPath, cardDir) {
					relCard, _ := filepath.Rel(root, cardPath)
					missingFiles = append(missingFiles, fmt.Sprintf("%s: %s (%s)", relCard, refPath, refKey))
				} else {
					// File exists; check anchor if present
					anchor := admission.ExtractAnchor(refStr)
					if anchor != "" {
						resolvedPath := refPath
						if !filepath.IsAbs(refPath) {
							resolvedPath = filepath.Join(cardDir, refPath)
						}
						if !admission.AnchorExists(resolvedPath, anchor) {
							relCard, _ := filepath.Rel(root, cardPath)
							report.Notes = append(report.Notes, fmt.Sprintf("anchor warning: anchor '#%s' not found in %s (card: %s)", anchor, refPath, relCard))
						}
					}
				}
			}
		}

		// Check context_evidence refs
		evidence, ok := ctxAlign["context_evidence"].([]any)
		if !ok {
			continue
		}
		for _, ev := range evidence {
			evMap, ok := ev.(map[string]any)
			if !ok {
				continue
			}
			refStr, ok := evMap["ref"].(string)
			if !ok || refStr == "" {
				continue
			}
			refPath := admission.StripAnchor(refStr)
			if !admission.FileExists(refPath, cardDir) {
				relCard, _ := filepath.Rel(root, cardPath)
				missingFiles = append(missingFiles, fmt.Sprintf("%s: %s (context_evidence)", relCard, refPath))
			} else {
				// File exists; check anchor if present
				anchor := admission.ExtractAnchor(refStr)
				if anchor != "" {
					resolvedPath := refPath
					if !filepath.IsAbs(refPath) {
						resolvedPath = filepath.Join(cardDir, refPath)
					}
					if !admission.AnchorExists(resolvedPath, anchor) {
						relCard, _ := filepath.Rel(root, cardPath)
						report.Notes = append(report.Notes, fmt.Sprintf("anchor warning: anchor '#%s' not found in %s (card: %s)", anchor, refPath, relCard))
					}
				}
			}
		}
	}

	if cardsScanned == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "context_refs",
			Status: "passed",
			Note:   "no completion cards found",
		})
		return
	}

	if len(missingFiles) > 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "context_refs",
			Status: "failed",
			Note:   fmt.Sprintf("%d missing file(s): %s", len(missingFiles), strings.Join(missingFiles, "; ")),
		})
		report.Healthy = false
	} else {
		note := fmt.Sprintf("%d card(s) scanned, %d with context_alignment, %d without, %d unreadable/unparseable", cardsScanned, cardsWithAlignment, cardsSkipped, cardsUnreadable)
		report.Checks = append(report.Checks, Check{
			Name:   "context_refs",
			Status: "passed",
			Note:   note,
		})
	}
}
