package cli

import (
	"fmt"
	"io"

	"github.com/BrianNguyen29/x-harness/internal/doctor"
)

func handleDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	root := ""
	format := "json"

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
		}
	}

	if root == "" {
		fmt.Fprintln(stderr, "usage: x-harness doctor --root <path> [--format json|text] [--json]")
		return ExitUsage
	}

	report := doctor.Run(root)

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
