package contract

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Policy represents a contract oracle policy with grep_rules and dependency_rules.
type Policy struct {
	Version         int              `yaml:"version"`
	GrepRules       []GrepRule       `yaml:"grep_rules"`
	DependencyRules []DependencyRule `yaml:"dependency_rules"`
}

// GrepRule defines a single grep-based rule.
type GrepRule struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	FilePattern string   `yaml:"file_pattern"`
	Pattern     string   `yaml:"pattern"`
	Exclude     []string `yaml:"exclude,omitempty"`
	Message     string   `yaml:"message,omitempty"`
}

// DependencyRule defines a line-level import scanning rule.
type DependencyRule struct {
	ID               string   `yaml:"id"`
	Description      string   `yaml:"description"`
	FilePattern      string   `yaml:"file_pattern"`
	ForbiddenImports []string `yaml:"forbidden_imports"`
	AllowedImports   []string `yaml:"allowed_imports,omitempty"`
	Exclude          []string `yaml:"exclude,omitempty"`
	Message          string   `yaml:"message,omitempty"`
}

// Violation represents a single rule violation.
type Violation struct {
	RuleID  string `json:"rule_id"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
	Message string `json:"message"`
}

// Result represents the full contract check result.
type Result struct {
	OK           bool        `json:"ok"`
	Policy       string      `json:"policy"`
	FilesScanned int         `json:"files_scanned"`
	Violations   []Violation `json:"violations"`
}

// compiledRule pairs a rule with its compiled regex.
type compiledRule struct {
	Rule    GrepRule
	Pattern *regexp.Regexp
}

// Check runs the contract oracle against the given policy and paths.
// It returns a Result with OK=true if no violations were found.
func Check(policyPath string, paths []string) (*Result, error) {
	// Load policy
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy: %w", err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy: %w", err)
	}

	if policy.Version != 1 {
		return nil, fmt.Errorf("unsupported policy version: %d (expected 1)", policy.Version)
	}

	// Compile regex patterns
	var compiledRules []compiledRule
	for _, rule := range policy.GrepRules {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern for rule %s: %w", rule.ID, err)
		}
		compiledRules = append(compiledRules, compiledRule{Rule: rule, Pattern: re})
	}

	result := &Result{
		OK:         true,
		Policy:     policyPath,
		Violations: []Violation{},
	}

	// Collect all files to scan
	var filesToScan []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("cannot stat %s: %w", path, err)
		}
		if info.IsDir() {
			dirFiles, err := collectFiles(path)
			if err != nil {
				return nil, fmt.Errorf("failed to collect files from %s: %w", path, err)
			}
			filesToScan = append(filesToScan, dirFiles...)
		} else {
			filesToScan = append(filesToScan, path)
		}
	}

	// Scan each file
	for _, file := range filesToScan {
		result.FilesScanned++
		violations, err := scanFile(file, compiledRules)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", file, err)
		}
		result.Violations = append(result.Violations, violations...)

		// Scan for dependency rules
		depViolations, err := scanFileForDependencyRules(file, policy.DependencyRules)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s for dependency rules: %w", file, err)
		}
		result.Violations = append(result.Violations, depViolations...)
	}

	if len(result.Violations) > 0 {
		result.OK = false
	}

	return result, nil
}

func collectFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip broken symlinks or inaccessible paths
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// Use Lstat to check if it's a symlink; if so, use Stat to follow and check target
		if info.Mode()&os.ModeSymlink != 0 {
			targetInfo, err := os.Stat(path)
			if err != nil {
				// Broken symlink, skip
				return nil
			}
			if targetInfo.IsDir() {
				// Symlink to directory, skip
				return nil
			}
		}
		// Skip binary files by extension
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".pdf", ".zip", ".tar", ".gz":
			return nil
		}
		// Skip files larger than 1MB
		if info.Size() > 1024*1024 {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func scanFile(path string, rules []compiledRule) ([]Violation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var violations []Violation
	baseName := filepath.Base(path)

	scanner := bufio.NewScanner(f)
	const maxCapacity = 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, cr := range rules {
			// Check file pattern match
			matched, err := filepath.Match(cr.Rule.FilePattern, baseName)
			if err != nil {
				// Invalid pattern, skip rule
				continue
			}
			if !matched {
				continue
			}

			// Check pattern match
			if !cr.Pattern.MatchString(line) {
				continue
			}

			// Check exclude patterns against file path
			excluded := false
			for _, ex := range cr.Rule.Exclude {
				if strings.Contains(path, ex) {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}

			// Build message
			msg := cr.Rule.Message
			if msg == "" {
				msg = cr.Rule.Description
			}

			snippet := strings.TrimSpace(line)
			if len(snippet) > 120 {
				snippet = snippet[:120] + "..."
			}

			violations = append(violations, Violation{
				RuleID:  cr.Rule.ID,
				File:    path,
				Line:    lineNum,
				Snippet: snippet,
				Message: msg,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return violations, nil
}

// importLinePattern matches import-like lines in various languages.
// Matches: import foo from "bar", require("bar"), import "bar", etc.
// Uses multiline mode (m) so ^ matches start of each line for indented imports.
var importLinePattern = regexp.MustCompile(`(?im)(import\s+.*?\s+from\s+["']|require\s*\(\s*["']|^\s*import\s+["'])`)

// importPathDoubleQuotePattern extracts double-quoted import paths.
var importPathDoubleQuotePattern = regexp.MustCompile(`"([^""]+)"`)

// importPathSingleQuotePattern extracts single-quoted import paths.
var importPathSingleQuotePattern = regexp.MustCompile(`'([^'']+)'`)

func scanFileForDependencyRules(path string, rules []DependencyRule) ([]Violation, error) {
	if len(rules) == 0 {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var violations []Violation
	baseName := filepath.Base(path)

	scanner := bufio.NewScanner(f)
	const maxCapacity = 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Only process lines that look like import statements
		if !importLinePattern.MatchString(line) {
			continue
		}

		// Extract import path from the line
		importPath := extractImportPath(line)
		if importPath == "" {
			continue
		}

		for _, rule := range rules {
			// Check file pattern match
			matched, err := filepath.Match(rule.FilePattern, baseName)
			if err != nil {
				// Invalid pattern, skip rule
				continue
			}
			if !matched {
				continue
			}

			// Check exclude patterns against file path
			excluded := false
			for _, ex := range rule.Exclude {
				if strings.Contains(path, ex) {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}

			// Check if import path matches any forbidden import
			isForbidden := false
			for _, forbidden := range rule.ForbiddenImports {
				if strings.Contains(importPath, forbidden) {
					isForbidden = true
					break
				}
			}
			if !isForbidden {
				continue
			}

			// Check if allowed_imports suppresses this violation
			isAllowed := false
			for _, allowed := range rule.AllowedImports {
				if strings.Contains(importPath, allowed) {
					isAllowed = true
					break
				}
			}
			if isAllowed {
				continue
			}

			// Build message
			msg := rule.Message
			if msg == "" {
				msg = rule.Description
			}

			snippet := strings.TrimSpace(line)
			if len(snippet) > 120 {
				snippet = snippet[:120] + "..."
			}

			violations = append(violations, Violation{
				RuleID:  rule.ID,
				File:    path,
				Line:    lineNum,
				Snippet: snippet,
				Message: msg,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return violations, nil
}

// extractImportPath extracts the quoted import path from an import line.
// Returns empty string if no import path is found.
func extractImportPath(line string) string {
	// Try double quotes first (more common)
	matches := importPathDoubleQuotePattern.FindAllStringSubmatch(line, -1)
	if len(matches) > 0 {
		// Return the last match (usually the actual import path in import foo from "bar")
		return matches[len(matches)-1][1]
	}
	// Try single quotes
	matches = importPathSingleQuotePattern.FindAllStringSubmatch(line, -1)
	if len(matches) > 0 {
		return matches[len(matches)-1][1]
	}
	return ""
}

// ToJSON returns the result as formatted JSON.
func (r *Result) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
