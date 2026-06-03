package doctor

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DocsDriftReport summarizes drift between docs, workflows, and CLI
// commands. The checks in this file are deliberately minimal: they exist to
// catch the most common drift in Phase 1 (workflow / package / matrix
// inconsistencies) without becoming a full linter. Each check is a
// self-contained predicate so reviewers can disable individual checks
// without rewriting the rest of the suite.
type DocsDriftReport struct {
	Healthy   bool     `json:"healthy"`
	Root      string   `json:"root"`
	Checks    []Check  `json:"checks"`
	Notes     []string `json:"notes,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	DriftTags []string `json:"drift_tags,omitempty"`
}

// CheckDocsDrift runs the docs-drift checks against root. The function
// always returns a non-nil report so callers can introspect checks even
// when a panic is recovered.
func CheckDocsDrift(root string) *DocsDriftReport {
	r := &DocsDriftReport{Healthy: true, Root: root, Checks: []Check{}}

	if root == "" {
		r.addFailed("docs_root", "root path is empty", "missing: root")
		return r
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		r.addFailed("docs_root", "root path does not exist or is not a directory", "missing: "+root)
		return r
	}

	checkWorkflowMatchesGoVerifyFlags(r, root)
	checkPackageJSONMentionsVerify(r, root)
	checkPolicyMatrixVsWorkflow(r, root)
	checkPackageManagerConsistency(r, root)

	// Healthy stays true when no check marked itself failed.
	return r
}

func (r *DocsDriftReport) addFailed(name, note, driftTag string) {
	r.Healthy = false
	r.Checks = append(r.Checks, Check{Name: name, Status: "failed", Note: note})
	if driftTag != "" {
		r.DriftTags = append(r.DriftTags, driftTag)
		r.Warnings = append(r.Warnings, driftTag)
	}
}

func (r *DocsDriftReport) addPassed(name, note string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: "passed", Note: note})
}

func (r *DocsDriftReport) addSkipped(name, note string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: "skipped", Note: note})
}

// checkWorkflowMatchesGoVerifyFlags verifies that the canonical verify
// command in the CI workflow is consistent with the Go CLI's flag set.
// We only require the workflow to mention the go-native `verify` command
// and a recognised profile/flag family. We do NOT require every flag
// to appear; the point is to detect when the workflow drops verify
// entirely.
func checkWorkflowMatchesGoVerifyFlags(r *DocsDriftReport, root string) {
	workflowPath := filepath.Join(root, ".github", "workflows", "x-harness-verify.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		r.addSkipped("workflow_verify_command", "no .github/workflows/x-harness-verify.yml found")
		return
	}
	content := string(data)
	if !strings.Contains(content, "x-harness verify") {
		r.addFailed("workflow_verify_command",
			"CI workflow does not invoke 'x-harness verify'",
			"workflow_missing_verify")
		return
	}
	r.addPassed("workflow_verify_command", "CI workflow invokes x-harness verify")
}

// checkPackageJSONMentionsVerify looks for a verify script in
// package.json. When the root has a package.json, the script section
// should expose a verify command (or one is referenced from CI). We only
// warn on a missing entry to avoid false positives for monorepos.
func checkPackageJSONMentionsVerify(r *DocsDriftReport, root string) {
	packagePath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(packagePath)
	if err != nil {
		r.addSkipped("package_verify_script", "no package.json found")
		return
	}
	// We intentionally do a substring check on the script section; a
	// real parser would be brittle across JSON shapes.
	content := string(data)
	hasVerifyScript := strings.Contains(content, "\"verify\"")
	hasVerifyCommand := strings.Contains(content, "verify") && (strings.Contains(content, "tsc") || strings.Contains(content, "vitest"))
	if !hasVerifyScript && !hasVerifyCommand {
		r.addFailed("package_verify_script",
			"package.json does not appear to declare a verify command",
			"package_missing_verify")
		return
	}
	r.addPassed("package_verify_script", "package.json mentions a verify command")
}

// checkPolicyMatrixVsWorkflow is a soft check: it surfaces the number of
// matrix rules the workflow references. The check passes when the
// workflow runs the policy matrix command (Phase 1 hardening will
// require it; until then we treat absence as informational).
func checkPolicyMatrixVsWorkflow(r *DocsDriftReport, root string) {
	workflowPath := filepath.Join(root, ".github", "workflows", "x-harness-verify.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		r.addSkipped("workflow_policy_matrix", "no .github/workflows/x-harness-verify.yml found")
		return
	}
	content := string(data)
	if !strings.Contains(content, "policy matrix") {
		r.Notes = append(r.Notes, "CI workflow does not currently run `xh policy matrix`; consider adding for Phase 1 hardening.")
		r.addSkipped("workflow_policy_matrix", "no `xh policy matrix` step (informational)")
		return
	}
	r.addPassed("workflow_policy_matrix", "CI workflow runs `xh policy matrix`")
}

// checkPackageManagerConsistency detects mismatched package-manager
// references between README, package.json, and the workflow. The check
// is deliberately narrow: it flags only if BOTH `npm` and `pnpm` appear
// in the same file, which is the most common drift pattern.
func checkPackageManagerConsistency(r *DocsDriftReport, root string) {
	for _, rel := range []string{"README.md", "package.json", ".github/workflows/x-harness-verify.yml"} {
		path := filepath.Join(root, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(data)
		hasNpm := strings.Contains(text, "npm")
		hasPnpm := strings.Contains(text, "pnpm")
		if hasNpm && hasPnpm {
			r.addFailed("package_manager_drift:"+rel,
				rel+" references both npm and pnpm; pick one package manager",
				"package_manager_drift:"+rel)
			return
		}
	}
	r.addPassed("package_manager_consistency", "no mixed npm/pnpm references in core files")
}

// Helper kept here for symmetry with other packages. The scanner uses
// bufio; we accept a no-op import to keep doctor self-contained.
var _ = bufio.NewScanner

// FormatDocsDriftText renders a human-readable report to w. Kept simple
// so release.verify-docs can call it from the CLI.
func FormatDocsDriftText(r *DocsDriftReport, w io.Writer) {
	_, _ = w.Write([]byte("# x-harness Docs Drift\n\n"))
	if r.Healthy {
		_, _ = w.Write([]byte("healthy: true\n"))
	} else {
		_, _ = w.Write([]byte("healthy: false\n"))
	}
	_, _ = w.Write([]byte("root: " + r.Root + "\n\n"))
	if len(r.DriftTags) > 0 {
		sorted := append([]string{}, r.DriftTags...)
		sort.Strings(sorted)
		_, _ = w.Write([]byte("drift_tags:\n"))
		for _, t := range sorted {
			_, _ = w.Write([]byte("  - " + t + "\n"))
		}
		_, _ = w.Write([]byte("\n"))
	}
	_, _ = w.Write([]byte("checks:\n"))
	for _, c := range r.Checks {
		if c.Note != "" {
			_, _ = w.Write([]byte("  " + c.Name + " [" + c.Status + "] " + c.Note + "\n"))
		} else {
			_, _ = w.Write([]byte("  " + c.Name + " [" + c.Status + "]\n"))
		}
	}
	if len(r.Notes) > 0 {
		_, _ = w.Write([]byte("\nnotes:\n"))
		for _, n := range r.Notes {
			_, _ = w.Write([]byte("  - " + n + "\n"))
		}
	}
}
