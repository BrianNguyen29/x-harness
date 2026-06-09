package cli

import (
	"fmt"
	"io"

	"github.com/BrianNguyen29/x-harness/internal/conformance"
	"github.com/BrianNguyen29/x-harness/internal/repo"
)

func handleConformance(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness conformance <run> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "run":
		return handleConformanceRun(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: x-harness conformance <run> [options]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown conformance subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness conformance <run> [options]")
		return ExitUsage
	}
}

func handleConformanceRun(args []string, stdout io.Writer, stderr io.Writer) int {
	profile := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 < len(args) {
				profile = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		}
	}

	if profile == "" {
		fmt.Fprintln(stderr, "usage: x-harness conformance run --profile <profile> [--json]")
		return ExitUsage
	}

	if profile != "minimal" && profile != "strict" {
		fmt.Fprintf(stderr, "unknown profile: %s\n", profile)
		return ExitUsage
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	var report *conformance.Report
	if profile == "minimal" {
		report = conformance.RunMinimal(root)
	} else {
		report = conformance.RunStrict(root)
	}

	if jsonMode {
		if err := WriteJSON(stdout, report); err != nil {
			return ExitError
		}
	} else {
		WriteLine(stdout, "profile: %s", report.Profile)
		WriteLine(stdout, "ok: %v", report.OK)
		for _, c := range report.Checks {
			if c.Note != "" {
				WriteLine(stdout, "  [%s] %s (%s)", c.Status, c.Name, c.Note)
			} else {
				WriteLine(stdout, "  [%s] %s", c.Status, c.Name)
			}
		}
	}

	if report.OK {
		return ExitOK
	}
	return ExitError
}
