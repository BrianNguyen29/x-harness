// Package boundary implements V1 of the x-harness boundary enforcement
// feature. V1 is deliberately minimal: it loads a YAML policy file
// validated against schemas/boundary-policy.schema.json, then matches
// the file paths of candidate source files against the rule's `from`
// glob and scans the file's import lines against the rule's `to_import`
// pattern. There is no AST parser, no semgrep/codeql backend, and no
// LLM involvement; matching is deterministic and reproducible.
//
// The package is read-only. Check() never modifies source files or the
// policy. When the policy file is missing, Check returns a Result with
// OK=true and a non-nil warning so callers can opt in to boundary
// enforcement without hard-failing existing verify gates.
package boundary

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"gopkg.in/yaml.v3"
)

// Policy is the top-level structure of policies/boundaries.yaml (V1).
type Policy struct {
	Version     int    `yaml:"version" json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Boundaries  []Rule `yaml:"boundaries" json:"boundaries"`
}

// Action is the rule action. V1 supports deny, require_intermediate, warn.
type Action string

const (
	ActionDeny                Action = "deny"
	ActionRequireIntermediate Action = "require_intermediate"
	ActionWarn                Action = "warn"
)

// Severity is the rule severity. V1 supports info, warning, high, critical.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Rule is a single boundary rule. The From/ToImport/Intermediate/Allow
// fields are forward-slash globs (`**` and `*`) matched against repo
// relative paths. Patterns are compared case-sensitively using a
// purpose-built matcher (no shell-style expansion outside `**` and `*`).
type Rule struct {
	ID                 string   `yaml:"id" json:"id"`
	Description        string   `yaml:"description,omitempty" json:"description,omitempty"`
	From               string   `yaml:"from" json:"from"`
	ToImport           string   `yaml:"to_import" json:"to_import"`
	Action             Action   `yaml:"action" json:"action"`
	Severity           Severity `yaml:"severity" json:"severity"`
	Intermediate       string   `yaml:"intermediate,omitempty" json:"intermediate,omitempty"`
	Allow              []string `yaml:"allow,omitempty" json:"allow,omitempty"`
	AppliesToLanguages []string `yaml:"applies_to_languages,omitempty" json:"applies_to_languages,omitempty"`
}

// Violation is a single rule violation surfaced by Check.
type Violation struct {
	RuleID    string   `json:"rule_id"`
	Severity  Severity `json:"severity"`
	Action    Action   `json:"action"`
	File      string   `json:"file"`
	Line      int      `json:"line"`
	Import    string   `json:"import,omitempty"`
	Snippet   string   `json:"snippet,omitempty"`
	Message   string   `json:"message"`
	AppliesTo []string `json:"applies_to_languages,omitempty"`
}

// Result is the top-level output of Check. JSON-friendly and stable.
type Result struct {
	OK            bool        `json:"ok"`
	Policy        string      `json:"policy,omitempty"`
	PolicyLoaded  bool        `json:"policy_loaded"`
	FilesScanned  int         `json:"files_scanned"`
	FilesChecked  int         `json:"files_checked"`
	RulesChecked  int         `json:"rules_checked"`
	Violations    []Violation `json:"violations"`
	Warnings      []string    `json:"warnings,omitempty"`
	SchemaVersion string      `json:"schema_version"`
}

// SchemaVersion identifies the report shape. The schema is intentionally
// stable for V1.
const SchemaVersion = "x-harness.boundary.v1"

// Load reads and validates a boundary policy from path. The validation
// is intentionally hand-rolled to keep the V1 dependency surface
// minimal: a JSON Schema validator may be added in V2 once the field
// set stabilises.
func Load(path string) (*Policy, error) {
	if path == "" {
		return nil, fmt.Errorf("boundary policy path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot stat policy %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("policy path %s is a directory, expected a file", path)
	}

	// The boundary policy file is always YAML for V1. We still use the
	// shared loader so the same extension sniffing is applied.
	if loader.DetectFormat(path) != loader.FormatYAML {
		return nil, fmt.Errorf("boundary policy must be YAML in V1: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy: %w", err)
	}
	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse policy: %w", err)
	}
	if p.Version != 1 {
		return nil, fmt.Errorf("unsupported boundary policy version %d (V1 only)", p.Version)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate performs hand-rolled V1 validation. We intentionally avoid
// JSON Schema compilation here to keep the V1 dependency footprint
// minimal and the rule schema explicit in code.
func (p *Policy) Validate() error {
	if p.Version != 1 {
		return fmt.Errorf("version must be 1, got %d", p.Version)
	}
	seen := map[string]bool{}
	for i, rule := range p.Boundaries {
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("boundaries[%d]: id is required", i)
		}
		if seen[rule.ID] {
			return fmt.Errorf("boundaries[%d]: duplicate rule id %q", i, rule.ID)
		}
		seen[rule.ID] = true
		if strings.TrimSpace(rule.From) == "" {
			return fmt.Errorf("boundaries[%d] %q: from is required", i, rule.ID)
		}
		if strings.TrimSpace(rule.ToImport) == "" {
			return fmt.Errorf("boundaries[%d] %q: to_import is required", i, rule.ID)
		}
		switch rule.Action {
		case ActionDeny, ActionRequireIntermediate, ActionWarn:
			// ok
		case "":
			return fmt.Errorf("boundaries[%d] %q: action is required", i, rule.ID)
		default:
			return fmt.Errorf("boundaries[%d] %q: unknown action %q (allowed: deny, require_intermediate, warn)", i, rule.ID, rule.Action)
		}
		switch rule.Severity {
		case SeverityInfo, SeverityWarning, SeverityHigh, SeverityCritical:
			// ok
		case "":
			return fmt.Errorf("boundaries[%d] %q: severity is required", i, rule.ID)
		default:
			return fmt.Errorf("boundaries[%d] %q: unknown severity %q (allowed: info, warning, high, critical)", i, rule.ID, rule.Severity)
		}
		if rule.Action == ActionRequireIntermediate && strings.TrimSpace(rule.Intermediate) == "" {
			return fmt.Errorf("boundaries[%d] %q: require_intermediate requires `intermediate`", i, rule.ID)
		}
		for _, lang := range rule.AppliesToLanguages {
			switch lang {
			case "javascript", "typescript", "go":
				// ok
			default:
				return fmt.Errorf("boundaries[%d] %q: unknown language %q (allowed: javascript, typescript, go)", i, rule.ID, lang)
			}
		}
	}
	return nil
}

// fileEntry pairs a candidate path with the base it was discovered
// under, so the rule's `from` glob is matched against a path relative
// to the base. The CLI uses this indirection to support both
// directory walks (where base is the input dir) and single-file
// queries (where base is the resolved repo root).
type fileEntry struct {
	absPath string
	base    string
}

// Check runs the loaded policy against the candidate paths and returns
// a deterministic Result. The function is pure (no source mutation) and
// has no external dependencies beyond the local file system.
//
// When policy is nil (no policy file), Check returns a Result with
// OK=true and a single warning so the command can default to no-op
// without hard-failing verify gates.
func Check(policy *Policy, policyPath string, paths []string) (*Result, error) {
	result := &Result{
		OK:            true,
		Policy:        policyPath,
		PolicyLoaded:  policy != nil,
		Violations:    []Violation{},
		Warnings:      []string{},
		SchemaVersion: SchemaVersion,
	}
	if policy == nil {
		result.Warnings = append(result.Warnings, "no boundary policy loaded; `xh boundary check` is a no-op (opt-in feature)")
		return result, nil
	}
	result.RulesChecked = len(policy.Boundaries)

	entries, err := collectEntries(paths)
	if err != nil {
		return nil, err
	}
	return runRules(policy, policyPath, entries, result)
}

// CheckFile runs the policy against a single file with an explicit
// base directory. The base is used to compute the path the rule's
// `from` glob is matched against, so callers can anchor a single
// file to the resolved repo root.
func CheckFile(policy *Policy, policyPath, base, file string) (*Result, error) {
	result := &Result{
		OK:            true,
		Policy:        policyPath,
		PolicyLoaded:  policy != nil,
		Violations:    []Violation{},
		Warnings:      []string{},
		SchemaVersion: SchemaVersion,
	}
	if policy == nil {
		result.Warnings = append(result.Warnings, "no boundary policy loaded; `xh boundary check` is a no-op (opt-in feature)")
		return result, nil
	}
	result.RulesChecked = len(policy.Boundaries)
	entries := []fileEntry{{absPath: file, base: base}}
	return runRules(policy, policyPath, entries, result)
}

// CheckFiles runs the policy against a pre-collected list of file
// paths, anchoring every entry to the same base directory. Unlike
// Check, no directory walking is performed: callers are expected to
// have already selected the candidate files.
//
// This is the entry point used by `xh boundary check --changed`,
// where the file list comes from `git diff --name-only` and every
// path must be resolved relative to the resolved repo root so the
// rule's `from` glob (e.g. `src/ui/**`) matches a slash-normalised,
// repo-relative path. Using each file's parent directory as the base
// (as Check does for single-file inputs) would make the glob match
// against just the basename, silently dropping every rule whose
// `from` contains a `/`.
func CheckFiles(policy *Policy, policyPath, base string, files []string) (*Result, error) {
	result := &Result{
		OK:            true,
		Policy:        policyPath,
		PolicyLoaded:  policy != nil,
		Violations:    []Violation{},
		Warnings:      []string{},
		SchemaVersion: SchemaVersion,
	}
	if policy == nil {
		result.Warnings = append(result.Warnings, "no boundary policy loaded; `xh boundary check` is a no-op (opt-in feature)")
		return result, nil
	}
	result.RulesChecked = len(policy.Boundaries)

	entries := make([]fileEntry, 0, len(files))
	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve %s: %w", f, err)
		}
		entries = append(entries, fileEntry{absPath: abs, base: base})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].absPath < entries[j].absPath
	})
	return runRules(policy, policyPath, entries, result)
}

// collectEntries walks each path, expanding directories to candidate
// files. For single-file inputs the base is the file's parent dir.
// For directory inputs the base is the directory itself. The output
// is sorted by absPath so the result is deterministic.
func collectEntries(paths []string) ([]fileEntry, error) {
	var entries []fileEntry
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve %s: %w", p, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("cannot stat %s: %w", p, err)
		}
		if info.IsDir() {
			collected, err := collectCandidateFiles(abs)
			if err != nil {
				return nil, fmt.Errorf("collect files from %s: %w", abs, err)
			}
			for _, f := range collected {
				entries = append(entries, fileEntry{absPath: f, base: abs})
			}
		} else {
			entries = append(entries, fileEntry{absPath: abs, base: filepath.Dir(abs)})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].absPath < entries[j].absPath
	})
	return entries, nil
}

// runRules compiles the policy's rules once, applies them to each
// candidate file, and writes violations back into result. It is shared
// between Check and CheckFile so both code paths stay in sync.
func runRules(policy *Policy, policyPath string, entries []fileEntry, result *Result) (*Result, error) {
	_ = policyPath

	// Pre-compile matchers for each rule.
	type compiled struct {
		rule   Rule
		from   *globMatcher
		to     *globMatcher
		interm *globMatcher
		allow  []*globMatcher
	}
	compiledRules := make([]compiled, 0, len(policy.Boundaries))
	for _, r := range policy.Boundaries {
		from, err := newGlobMatcher(r.From)
		if err != nil {
			return nil, fmt.Errorf("rule %q: invalid from glob: %w", r.ID, err)
		}
		to, err := newGlobMatcher(r.ToImport)
		if err != nil {
			return nil, fmt.Errorf("rule %q: invalid to_import glob: %w", r.ID, err)
		}
		var interm *globMatcher
		if r.Intermediate != "" {
			interm, err = newGlobMatcher(r.Intermediate)
			if err != nil {
				return nil, fmt.Errorf("rule %q: invalid intermediate glob: %w", r.ID, err)
			}
		}
		var allowList []*globMatcher
		for _, a := range r.Allow {
			m, err := newGlobMatcher(a)
			if err != nil {
				return nil, fmt.Errorf("rule %q: invalid allow entry %q: %w", r.ID, a, err)
			}
			allowList = append(allowList, m)
		}
		compiledRules = append(compiledRules, compiled{rule: r, from: from, to: to, interm: interm, allow: allowList})
	}

	for _, entry := range entries {
		file := entry.absPath
		// Always count the file as scanned; the user passed it in.
		result.FilesScanned++

		// Read import lines once per file.
		imports, err := scanImportLines(file)
		if err != nil {
			// We don't fail the whole run on a single unreadable file;
			// surface it as a warning and keep going.
			result.Warnings = append(result.Warnings, fmt.Sprintf("cannot read imports for %s: %v", file, err))
			continue
		}
		if len(imports) == 0 {
			continue
		}

		// Compute the path relative to the input base and normalise
		// separators. This is what the rule's `from` glob is matched
		// against so users can write `src/ui/**` regardless of whether
		// they invoked the command from the repo root or a temp dir.
		relRaw, relErr := filepath.Rel(entry.base, file)
		rel := file
		if relErr == nil {
			rel = relRaw
		}
		rel = normalisePath(rel)
		language := detectLanguage(file)

		for _, cr := range compiledRules {
			if !cr.from.Match(rel) {
				continue
			}
			if !ruleAppliesToLanguage(cr.rule.AppliesToLanguages, language) {
				continue
			}
			result.FilesChecked++

			for _, imp := range imports {
				impPath := normaliseImportPath(imp.Target)
				if impPath == "" {
					continue
				}
				if !cr.to.Match(impPath) {
					continue
				}
				if cr.interm != nil && cr.interm.Match(impPath) {
					// require_intermediate satisfied; no violation.
					continue
				}
				if matchAny(cr.allow, impPath) || matchAny(cr.allow, rel) {
					// allow list exempts this import.
					continue
				}

				v := Violation{
					RuleID:    cr.rule.ID,
					Severity:  cr.rule.Severity,
					Action:    cr.rule.Action,
					File:      rel,
					Line:      imp.Line,
					Import:    impPath,
					Snippet:   truncate(imp.Raw, 160),
					AppliesTo: cr.rule.AppliesToLanguages,
				}
				v.Message = buildMessage(cr.rule, impPath)
				result.Violations = append(result.Violations, v)
				result.OK = false
			}
		}
	}

	sort.SliceStable(result.Violations, func(i, j int) bool {
		if result.Violations[i].RuleID != result.Violations[j].RuleID {
			return result.Violations[i].RuleID < result.Violations[j].RuleID
		}
		if result.Violations[i].File != result.Violations[j].File {
			return result.Violations[i].File < result.Violations[j].File
		}
		return result.Violations[i].Line < result.Violations[j].Line
	})

	return result, nil
}

// buildMessage returns a stable, human-readable message for a violation.
func buildMessage(r Rule, importPath string) string {
	switch r.Action {
	case ActionRequireIntermediate:
		return fmt.Sprintf("%s: import %q must go through %q", r.ID, importPath, r.Intermediate)
	case ActionWarn:
		return fmt.Sprintf("%s: import %q is discouraged", r.ID, importPath)
	default:
		return fmt.Sprintf("%s: forbidden import %q", r.ID, importPath)
	}
}

// matchAny returns true if any of the matchers matches the value.
func matchAny(matchers []*globMatcher, value string) bool {
	for _, m := range matchers {
		if m.Match(value) {
			return true
		}
	}
	return false
}

// truncate shortens s to max characters, appending an ellipsis when it
// would otherwise be cut. Used for the snippet field which is purely
// informational.
func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// normalisePath strips a leading "./" and converts backslashes to
// forward slashes so the matchers behave consistently on Windows-style
// paths and on the literal paths produced by filepath.Walk.
func normalisePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	return p
}

// normaliseImportPath strips surrounding quotes and surrounding angle
// brackets (used by Go) from an import target. The resulting string is
// what the rule's `to_import` glob is matched against.
func normaliseImportPath(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	s = strings.TrimPrefix(s, "'")
	s = strings.TrimSuffix(s, "'")
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")
	return s
}
