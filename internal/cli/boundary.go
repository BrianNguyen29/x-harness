package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/boundary"
	"github.com/BrianNguyen29/x-harness/internal/loader"
)

func handleBoundary(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: xh boundary <lint|check|explain> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "lint":
		return handleBoundaryLint(args[1:], stdout, stderr)
	case "check":
		return handleBoundaryCheck(args[1:], stdout, stderr)
	case "explain":
		return handleBoundaryExplain(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: xh boundary <lint|check|explain> [options]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown boundary subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: xh boundary <lint|check|explain> [options]")
		return ExitUsage
	}
}

// boundaryOptions is the shared flag/state for boundary subcommands.
type boundaryOptions struct {
	policyPath string
	format     string
	root       string
}

// parseBoundaryCommonFlags walks args and returns the shared options
// plus the remaining positional arguments. The boolean returned is true
// when the caller should emit a usage error.
func parseBoundaryCommonFlags(args []string, stderr io.Writer, allowPositional bool) (boundaryOptions, []string, bool) {
	opts := boundaryOptions{format: "text", root: "."}
	positional := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--policy":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --policy requires a value")
				return opts, nil, true
			}
			opts.policyPath = args[i+1]
			i++
		case "--format":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --format requires a value")
				return opts, nil, true
			}
			opts.format = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return opts, nil, true
			}
			opts.root = args[i+1]
			i++
		case "-h", "--help":
			return opts, nil, true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return opts, nil, true
			}
			positional = append(positional, args[i])
		}
	}
	if !allowPositional && len(positional) > 0 {
		fmt.Fprintf(stderr, "unexpected argument: %s\n", positional[0])
		return opts, nil, true
	}
	return opts, positional, false
}

// resolveBoundaryPolicy returns the policy path to load, taking the
// explicit --policy flag, the bundled asset, or the safe default
// (policies/boundaries.yaml under the resolved root).
func resolveBoundaryPolicy(opts boundaryOptions) (string, error) {
	if opts.policyPath != "" {
		return opts.policyPath, nil
	}
	root := opts.root
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(abs, "policies", "boundaries.yaml")
	return candidate, nil
}

func handleBoundaryLint(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, _, fail := parseBoundaryCommonFlags(args, stderr, false)
	if fail {
		fmt.Fprintln(stderr, "usage: xh boundary lint [--policy <path>] [--root <dir>] [--format text|json]")
		return ExitUsage
	}
	if !isValidBoundaryFormat(opts.format) {
		fmt.Fprintf(stderr, "unknown format: %s\n", opts.format)
		return ExitUsage
	}

	policyPath, err := resolveBoundaryPolicy(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	// Missing policy file is a no-op (lint passes with a warning), per
	// the V1 spec.
	if _, statErr := os.Stat(policyPath); statErr != nil {
		report := boundaryLintReport{
			OK:            true,
			Policy:        policyPath,
			PolicyLoaded:  false,
			RulesChecked:  0,
			SchemaVersion: boundary.SchemaVersion,
			Warnings:      []string{"no boundary policy loaded; `xh boundary lint` is a no-op (opt-in feature)"},
		}
		return renderBoundaryLintReport(&report, opts.format, stdout)
	}

	p, err := boundary.Load(policyPath)
	if err != nil {
		report := boundaryLintReport{
			OK:            false,
			Policy:        policyPath,
			PolicyLoaded:  true,
			RulesChecked:  0,
			SchemaVersion: boundary.SchemaVersion,
			Errors:        []string{err.Error()},
		}
		if rerr := renderBoundaryLintReport(&report, opts.format, stdout); rerr != 0 {
			return rerr
		}
		return ExitError
	}

	report := boundaryLintReport{
		OK:            true,
		Policy:        policyPath,
		PolicyLoaded:  true,
		RulesChecked:  len(p.Boundaries),
		SchemaVersion: boundary.SchemaVersion,
	}
	return renderBoundaryLintReport(&report, opts.format, stdout)
}

// boundaryLintReport is the JSON output of `xh boundary lint`. We
// keep it as a separate struct (rather than reusing boundary.Result)
// because lint's report focuses on policy validity, not file scanning.
type boundaryLintReport struct {
	OK            bool     `json:"ok"`
	Policy        string   `json:"policy"`
	PolicyLoaded  bool     `json:"policy_loaded"`
	RulesChecked  int      `json:"rules_checked"`
	SchemaVersion string   `json:"schema_version"`
	Warnings      []string `json:"warnings,omitempty"`
	Errors        []string `json:"errors,omitempty"`
}

func renderBoundaryLintReport(report *boundaryLintReport, format string, stdout io.Writer) int {
	switch format {
	case "json":
		if err := WriteJSON(stdout, report); err != nil {
			return ExitError
		}
		return ExitOK
	default:
		WriteLine(stdout, "# x-harness Boundary Lint")
		WriteLine(stdout, "")
		WriteLine(stdout, "policy: %s", report.Policy)
		WriteLine(stdout, "policy_loaded: %t", report.PolicyLoaded)
		WriteLine(stdout, "rules_checked: %d", report.RulesChecked)
		WriteLine(stdout, "ok: %t", report.OK)
		if len(report.Warnings) > 0 {
			WriteLine(stdout, "")
			WriteLine(stdout, "warnings:")
			for _, w := range report.Warnings {
				WriteLine(stdout, "  - %s", w)
			}
		}
		if len(report.Errors) > 0 {
			WriteLine(stdout, "")
			WriteLine(stdout, "errors:")
			for _, e := range report.Errors {
				WriteLine(stdout, "  - %s", e)
			}
		}
		if !report.OK {
			return ExitError
		}
		return ExitOK
	}
}

func handleBoundaryCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	scope := "all"
	// Walk the args once, looking for scope flags. We deliberately
	// do not pull these into parseBoundaryCommonFlags because they
	// are mutually exclusive.
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			scope = "all"
		case "--changed":
			scope = "changed"
		default:
			remaining = append(remaining, args[i])
		}
	}

	opts, extra, fail := parseBoundaryCommonFlags(remaining, stderr, true)
	if fail {
		fmt.Fprintln(stderr, "usage: xh boundary check --all|--changed [--policy <path>] [--root <dir>] [--format text|json] [paths...]")
		return ExitUsage
	}
	if !isValidBoundaryFormat(opts.format) {
		fmt.Fprintf(stderr, "unknown format: %s\n", opts.format)
		return ExitUsage
	}

	policyPath, err := resolveBoundaryPolicy(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	// Load policy (nil is OK; boundary.Check treats it as no-op).
	var pol *boundary.Policy
	if _, statErr := os.Stat(policyPath); statErr == nil {
		pol, err = boundary.Load(policyPath)
		if err != nil {
			fmt.Fprintf(stderr, "error: cannot load policy %s: %v\n", policyPath, err)
			return ExitError
		}
	} else if scope == "all" && opts.policyPath != "" {
		// The user explicitly asked for a policy that does not
		// exist. Surface the error rather than silently no-oping.
		fmt.Fprintf(stderr, "error: policy not found: %s\n", policyPath)
		return ExitError
	}

	// Determine the file set.
	targetPaths := extra
	if len(targetPaths) == 0 {
		// Default to the resolved root directory.
		absRoot, err := filepath.Abs(opts.root)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
		targetPaths = []string{absRoot}
	}

	if scope == "changed" {
		// For --changed we always use the resolved root (the repo
		// root) and the git diff list. Positional paths are ignored
		// with a warning to keep semantics predictable.
		absRoot, err := filepath.Abs(opts.root)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
		if len(extra) > 0 {
			fmt.Fprintln(stderr, "warning: --changed ignores positional path arguments; using git diff in", absRoot)
		}
		files, err := boundary.ListChangedFiles(absRoot)
		if err != nil {
			// Git unavailable or not a repo: emit a warning and
			// continue with an empty file set so the rest of the
			// command still works.
			fmt.Fprintf(stderr, "warning: cannot read git diff: %v\n", err)
			files = nil
		}
		// Filter to files that actually exist on disk (the diff may
		// list deleted files).
		var existing []string
		for _, f := range files {
			abs := f
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(absRoot, f)
			}
			if info, statErr := os.Stat(abs); statErr == nil && !info.IsDir() {
				existing = append(existing, abs)
			}
		}
		// Run check with the explicit list of files. To keep the rule
		// `from` glob path-relative, we pass each file's directory
		// as the base; the policy.go path resolution will treat them
		// as standalone files.
		result, err := runBoundaryCheck(pol, policyPath, existing, absRoot, opts.root)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
		return renderBoundaryCheckResult(result, opts.format, stdout, stderr)
	}

	result, err := runBoundaryCheck(pol, policyPath, targetPaths, "", opts.root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	return renderBoundaryCheckResult(result, opts.format, stdout, stderr)
}

// runBoundaryCheck wraps the boundary package's check entry points so
// the CLI can vary the resolved root and the working directory used
// for git integration without duplicating the call.
//
// When base is empty, the legacy directory-walk entry point
// (boundary.Check) is used. When base is non-empty, the caller has
// already collected a file list (e.g. from `git diff --name-only`)
// and wants every entry anchored to that common base, so the
// batch-friendly boundary.CheckFiles is used. This distinction is
// what makes `xh boundary check --changed` honour `from` globs that
// contain path separators.
func runBoundaryCheck(pol *boundary.Policy, policyPath string, paths []string, base, rootDisplay string) (*boundary.Result, error) {
	_ = rootDisplay
	if base != "" {
		return boundary.CheckFiles(pol, policyPath, base, paths)
	}
	return boundary.Check(pol, policyPath, paths)
}

func renderBoundaryCheckResult(result *boundary.Result, format string, stdout, _ io.Writer) int {
	switch format {
	case "json":
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	default:
		WriteLine(stdout, "# x-harness Boundary Check")
		WriteLine(stdout, "")
		WriteLine(stdout, "policy: %s", result.Policy)
		WriteLine(stdout, "policy_loaded: %t", result.PolicyLoaded)
		WriteLine(stdout, "schema_version: %s", result.SchemaVersion)
		WriteLine(stdout, "ok: %t", result.OK)
		WriteLine(stdout, "files_scanned: %d", result.FilesScanned)
		WriteLine(stdout, "files_checked: %d", result.FilesChecked)
		WriteLine(stdout, "rules_checked: %d", result.RulesChecked)
		WriteLine(stdout, "violations: %d", len(result.Violations))
		if len(result.Warnings) > 0 {
			WriteLine(stdout, "")
			WriteLine(stdout, "warnings:")
			for _, w := range result.Warnings {
				WriteLine(stdout, "  - %s", w)
			}
		}
		if len(result.Violations) > 0 {
			WriteLine(stdout, "")
			WriteLine(stdout, "| Rule | Severity | Action | File | Line | Import | Message |")
			WriteLine(stdout, "| :-- | :-- | :-- | :-- | :-- | :-- | :-- |")
			for _, v := range result.Violations {
				snippet := v.Import
				if len(snippet) > 60 {
					snippet = snippet[:60] + "..."
				}
				WriteLine(stdout, "| %s | %s | %s | %s | %d | `%s` | %s |",
					v.RuleID, v.Severity, v.Action, v.File, v.Line, snippet, v.Message)
			}
		}
	}
	if result.OK {
		return ExitOK
	}
	return ExitError
}

func handleBoundaryExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: xh boundary explain <file> [--policy <path>] [--root <dir>] [--format text|json]")
		return ExitUsage
	}
	// The first positional arg is the file; the rest are flags or
	// further positionals.
	targetFile := args[0]
	rest := args[1:]

	opts, _, fail := parseBoundaryCommonFlags(rest, stderr, false)
	if fail {
		fmt.Fprintln(stderr, "usage: xh boundary explain <file> [--policy <path>] [--root <dir>] [--format text|json]")
		return ExitUsage
	}
	if !isValidBoundaryFormat(opts.format) {
		fmt.Fprintf(stderr, "unknown format: %s\n", opts.format)
		return ExitUsage
	}

	policyPath, err := resolveBoundaryPolicy(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if _, statErr := os.Stat(policyPath); statErr != nil {
		report := boundaryExplainReport{
			OK:            true,
			Policy:        policyPath,
			PolicyLoaded:  false,
			File:          targetFile,
			Rules:         []boundaryExplainRuleHit{},
			SchemaVersion: boundary.SchemaVersion,
			Warnings:      []string{"no boundary policy loaded; nothing to explain"},
		}
		return renderBoundaryExplainReport(&report, opts.format, stdout)
	}

	pol, err := boundary.Load(policyPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot load policy: %v\n", err)
		return ExitError
	}

	// Normalise the file path: relative paths are resolved against
	// the resolved root. We then match the file against each rule
	// in isolation to produce a per-rule "hit" list, without doing
	// a full repo scan.
	absRoot, err := filepath.Abs(opts.root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	targetAbs := targetFile
	if !filepath.IsAbs(targetAbs) {
		// Try the literal path first, then the path under the root.
		if _, statErr := os.Stat(targetAbs); statErr != nil {
			targetAbs = filepath.Join(absRoot, targetFile)
		}
	}
	info, statErr := os.Stat(targetAbs)
	if statErr != nil {
		fmt.Fprintf(stderr, "error: cannot stat %s: %v\n", targetFile, statErr)
		return ExitError
	}
	if info.IsDir() {
		fmt.Fprintf(stderr, "error: %s is a directory; expected a file\n", targetFile)
		return ExitUsage
	}

	rel, relErr := filepath.Rel(absRoot, targetAbs)
	display := targetFile
	if relErr == nil {
		display = rel
	}
	display = strings.ReplaceAll(display, "\\", "/")

	// Compute the rule-by-rule explanation by running a tiny ad-hoc
	// scan: re-use the policy's rules against the single file with
	// absRoot as the base so the rule's `from` glob matches against
	// the repo-relative path.
	result, err := boundary.CheckFile(pol, policyPath, absRoot, targetAbs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	// Group violations by rule for the explain view.
	hitsByRule := map[string][]boundary.Violation{}
	for _, v := range result.Violations {
		hitsByRule[v.RuleID] = append(hitsByRule[v.RuleID], v)
	}

	// Build the rule list: include every rule, marking whether it
	// applies and whether it raised violations.
	var ruleHits []boundaryExplainRuleHit
	ruleIDs := make([]string, 0, len(pol.Boundaries))
	for _, r := range pol.Boundaries {
		ruleIDs = append(ruleIDs, r.ID)
	}
	sort.Strings(ruleIDs)
	for _, id := range ruleIDs {
		var rule boundary.Rule
		for _, r := range pol.Boundaries {
			if r.ID == id {
				rule = r
				break
			}
		}
		violations := hitsByRule[id]
		ruleHits = append(ruleHits, boundaryExplainRuleHit{
			RuleID:           rule.ID,
			Description:      rule.Description,
			From:             rule.From,
			ToImport:         rule.ToImport,
			Action:           string(rule.Action),
			Severity:         string(rule.Severity),
			Intermediate:     rule.Intermediate,
			Allow:            rule.Allow,
			AppliesTo:        rule.AppliesToLanguages,
			AppliesToFile:    ruleAppliesTo(rule, display),
			ViolationCount:   len(violations),
			ViolationImports: importStrings(violations),
		})
	}

	report := boundaryExplainReport{
		OK:            result.OK,
		Policy:        policyPath,
		PolicyLoaded:  true,
		File:          display,
		Rules:         ruleHits,
		SchemaVersion: boundary.SchemaVersion,
	}
	return renderBoundaryExplainReport(&report, opts.format, stdout)
}

type boundaryExplainReport struct {
	OK            bool                     `json:"ok"`
	Policy        string                   `json:"policy"`
	PolicyLoaded  bool                     `json:"policy_loaded"`
	File          string                   `json:"file"`
	Rules         []boundaryExplainRuleHit `json:"rules"`
	SchemaVersion string                   `json:"schema_version"`
	Warnings      []string                 `json:"warnings,omitempty"`
}

type boundaryExplainRuleHit struct {
	RuleID           string   `json:"rule_id"`
	Description      string   `json:"description,omitempty"`
	From             string   `json:"from"`
	ToImport         string   `json:"to_import"`
	Action           string   `json:"action"`
	Severity         string   `json:"severity"`
	Intermediate     string   `json:"intermediate,omitempty"`
	Allow            []string `json:"allow,omitempty"`
	AppliesTo        []string `json:"applies_to_languages,omitempty"`
	AppliesToFile    bool     `json:"applies_to_file"`
	ViolationCount   int      `json:"violation_count"`
	ViolationImports []string `json:"violation_imports,omitempty"`
}

// ruleAppliesTo reports whether a rule's `from` glob matches the file
// path. We re-use the same glob matcher the checker uses so the
// explain output agrees with the checker's behaviour.
func ruleAppliesTo(rule boundary.Rule, file string) bool {
	from, err := boundary.NewGlobMatcherPublic(rule.From)
	if err != nil {
		return false
	}
	return from.Match(file)
}

// importStrings returns the import paths from a slice of violations.
func importStrings(vs []boundary.Violation) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		if v.Import != "" {
			out = append(out, v.Import)
		}
	}
	return out
}

func renderBoundaryExplainReport(report *boundaryExplainReport, format string, stdout io.Writer) int {
	switch format {
	case "json":
		if err := WriteJSON(stdout, report); err != nil {
			return ExitError
		}
	default:
		WriteLine(stdout, "# x-harness Boundary Explain")
		WriteLine(stdout, "")
		WriteLine(stdout, "policy: %s", report.Policy)
		WriteLine(stdout, "policy_loaded: %t", report.PolicyLoaded)
		WriteLine(stdout, "file: %s", report.File)
		WriteLine(stdout, "ok: %t", report.OK)
		WriteLine(stdout, "")
		if len(report.Rules) == 0 {
			WriteLine(stdout, "No rules defined in policy.")
			return ExitOK
		}
		WriteLine(stdout, "rules:")
		for _, r := range report.Rules {
			applies := "no"
			if r.AppliesToFile {
				applies = "yes"
			}
			WriteLine(stdout, "  - id: %s", r.RuleID)
			WriteLine(stdout, "    applies_to_file: %s", applies)
			WriteLine(stdout, "    action: %s", r.Action)
			WriteLine(stdout, "    severity: %s", r.Severity)
			WriteLine(stdout, "    from: %s", r.From)
			WriteLine(stdout, "    to_import: %s", r.ToImport)
			if r.Intermediate != "" {
				WriteLine(stdout, "    intermediate: %s", r.Intermediate)
			}
			if len(r.Allow) > 0 {
				WriteLine(stdout, "    allow: %s", strings.Join(r.Allow, ", "))
			}
			if len(r.AppliesTo) > 0 {
				WriteLine(stdout, "    applies_to_languages: %s", strings.Join(r.AppliesTo, ", "))
			}
			WriteLine(stdout, "    violation_count: %d", r.ViolationCount)
			if len(r.ViolationImports) > 0 {
				WriteLine(stdout, "    violation_imports:")
				for _, imp := range r.ViolationImports {
					WriteLine(stdout, "      - %s", imp)
				}
			}
		}
	}
	if report.OK {
		return ExitOK
	}
	return ExitError
}

// isValidBoundaryFormat reports whether the value is one of the
// supported render formats.
func isValidBoundaryFormat(format string) bool {
	return format == "text" || format == "json"
}

// policyLoadYAML is a tiny helper used by boundary tests to load the
// canonical policy file from the repo root. It is here (rather than
// in the policy package) because the boundary tests do not have
// direct access to the policy package's root-aware loader.
func policyLoadYAML(path string, v any) error {
	return loader.LoadYAML(path, v)
}
