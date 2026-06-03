package cli

import (
	"fmt"
	"io"

	"github.com/BrianNguyen29/x-harness/internal/doctor"
	"github.com/BrianNguyen29/x-harness/internal/worktree"
)

func handleDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	root := ""
	format := "json"
	staleness := false
	overclaim := false
	context := false
	worktreeFlag := false
	docsDrift := false

	// P2-S2: --fix enables a deterministic repair flow. Dry-run is
	// the default; --confirm is required to actually mutate files.
	fixFlag := false
	confirm := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--json":
			format = "json"
		case "--staleness":
			staleness = true
		case "--overclaim":
			overclaim = true
		case "--context":
			context = true
		case "--worktree":
			worktreeFlag = true
		case "--docs-drift":
			docsDrift = true
		case "--fix":
			fixFlag = true
		case "--confirm":
			confirm = true
		}
	}

	if root == "" {
		fmt.Fprintln(stderr, "usage: x-harness doctor --root <path> [--format json|text] [--json] [--staleness] [--overclaim] [--context] [--worktree] [--docs-drift] [--fix] [--confirm]")
		return ExitUsage
	}

	// --docs-drift is a separate mode: it runs the lightweight
	// docs-drift checks and returns immediately. We deliberately do
	// not blend it with the full doctor report so the two failure
	// domains stay separate.
	if docsDrift {
		return runDocsDrift(root, format, stdout, stderr)
	}

	report := doctor.RunWithOptions(root, doctor.Options{Staleness: staleness, Overclaim: overclaim, Context: context})

	// P2-S2: when --fix is requested, build (and optionally apply) the
	// deterministic repair plan. The plan is attached to the JSON
	// output and printed alongside the text report. The fix flow
	// never mutates the workspace unless --confirm is also passed.
	var fixResult *DoctorFix
	if fixFlag {
		fixResult = runDoctorFixWithAssetRoot(root, confirm, stderr)
	}

	if worktreeFlag {
		wt := worktree.CollectInfo(root)
		if wt != nil {
			report.Checks = append(report.Checks, doctor.Check{
				Name:   "worktree_info",
				Status: "passed",
				Note:   fmt.Sprintf("branch=%s commit=%s root=%s", wt.Branch, wt.Commit, wt.Root),
			})
		} else {
			report.Checks = append(report.Checks, doctor.Check{
				Name:   "worktree_info",
				Status: "skipped",
				Note:   "not a git repository or git unavailable",
			})
		}
	}

	// P2-S2: when --confirm is set we re-evaluate the doctor report
	// so the rendered health summary and the exit code reflect the
	// post-fix state. The original report is still emitted, but the
	// re-evaluation ensures that issues we just fixed are no longer
	// counted, while non-deterministic issues (overclaim, tier labels,
	// etc.) keep the workspace unhealthy and the exit code at
	// ExitError.
	reportForOutput := report
	if fixFlag && confirm && fixResult != nil && fixResult.ManifestFound {
		reportForOutput = doctor.RunWithOptions(root, doctor.Options{Staleness: staleness, Overclaim: overclaim, Context: context})
		if worktreeFlag {
			wt := worktree.CollectInfo(root)
			if wt != nil {
				reportForOutput.Checks = append(reportForOutput.Checks, doctor.Check{
					Name:   "worktree_info",
					Status: "passed",
					Note:   fmt.Sprintf("branch=%s commit=%s root=%s", wt.Branch, wt.Commit, wt.Root),
				})
			} else {
				reportForOutput.Checks = append(reportForOutput.Checks, doctor.Check{
					Name:   "worktree_info",
					Status: "skipped",
					Note:   "not a git repository or git unavailable",
				})
			}
		}
	}

	switch format {
	case "json":
		if fixFlag {
			// Embed the report and add a top-level "fix" block. The
			// embedded struct is JSON-flattened, so non-fix consumers
			// see the same shape as before.
			out := struct {
				*doctor.Report
				Fix *DoctorFix `json:"fix,omitempty"`
			}{Report: reportForOutput, Fix: fixResult}
			if err := WriteJSON(stdout, out); err != nil {
				return ExitError
			}
		} else {
			if err := WriteJSON(stdout, report); err != nil {
				return ExitError
			}
		}
	case "text":
		renderDoctorText(stdout, reportForOutput)
		if fixFlag {
			renderDoctorFixText(stdout, fixResult)
		}
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", format)
		return ExitUsage
	}

	// P2-S2: in --fix mode the exit code is driven by the fix
	// outcome, not the doctor health summary, so dry-run previews do
	// not fail the command and confirm runs only succeed when the
	// planned fixes were actually applied.
	if fixFlag && fixResult != nil {
		// Dry-run: always ExitOK (it's a preview).
		if !confirm {
			return ExitOK
		}
		// Confirm without manifest: ExitError (nothing to fix).
		if !fixResult.ManifestFound {
			return ExitError
		}
		// Confirm with manifest: ExitOK only if the post-fix
		// doctor report is healthy.
		if reportForOutput.Healthy {
			return ExitOK
		}
		return ExitError
	}

	if report.Healthy {
		return ExitOK
	}
	return ExitError
}

// runDocsDrift runs the docs-drift checks and renders either JSON or
// text output. Returns ExitOK on healthy, ExitError otherwise. The
// function is intentionally read-only: it never mutates files.
func runDocsDrift(root, format string, stdout, stderr io.Writer) int {
	report := doctor.CheckDocsDrift(root)
	switch format {
	case "json":
		if err := WriteJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
	case "text":
		doctor.FormatDocsDriftText(report, stdout)
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", format)
		return ExitUsage
	}
	if report.Healthy {
		return ExitOK
	}
	return ExitError
}

func renderDoctorText(w io.Writer, report *doctor.Report) {
	WriteLine(w, "healthy: %v", report.Healthy)
	WriteLine(w, "present: %d", report.PresentCount)
	WriteLine(w, "missing: %d", report.MissingCount)
	if len(report.Missing) > 0 {
		WriteLine(w, "missing_items:")
		for _, item := range report.Missing {
			WriteLine(w, "  - %s", item)
		}
	}
	WriteLine(w, "checks:")
	for _, check := range report.Checks {
		status := check.Status
		if check.Note != "" {
			WriteLine(w, "  %-30s %s (%s)", check.Name, status, check.Note)
		} else {
			WriteLine(w, "  %-30s %s", check.Name, status)
		}
	}
	if len(report.Notes) > 0 {
		WriteLine(w, "notes:")
		for _, note := range report.Notes {
			WriteLine(w, "  - %s", note)
		}
	}
}
