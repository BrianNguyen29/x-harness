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
	worktreeFlag := false

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
		case "--worktree":
			worktreeFlag = true
		}
	}

	if root == "" {
		fmt.Fprintln(stderr, "usage: x-harness doctor --root <path> [--format json|text] [--json] [--staleness] [--worktree]")
		return ExitUsage
	}

	report := doctor.RunWithOptions(root, doctor.Options{Staleness: staleness})

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

	switch format {
	case "json":
		if err := WriteJSON(stdout, report); err != nil {
			return ExitError
		}
	case "text":
		renderDoctorText(stdout, report)
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
