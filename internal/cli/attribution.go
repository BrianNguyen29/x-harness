package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/BrianNguyen29/x-harness/internal/attribution"
)

func handleAttribution(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness attribution explain --episode <dir> [--json]")
		return ExitUsage
	}

	subcommand := args[0]
	if subcommand != "explain" {
		fmt.Fprintf(stderr, "unknown attribution subcommand: %s\n", subcommand)
		fmt.Fprintln(stderr, "usage: x-harness attribution explain --episode <dir> [--json]")
		return ExitUsage
	}

	episodeDir := ""
	jsonMode := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--episode":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --episode requires a value")
				return ExitUsage
			}
			episodeDir = args[i+1]
			i++
		case "--json":
			jsonMode = true
		default:
			if len(args[i]) > 0 && args[i][0] == '-' {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				fmt.Fprintln(stderr, "usage: x-harness attribution explain --episode <dir> [--json]")
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	if episodeDir == "" {
		fmt.Fprintln(stderr, "usage: x-harness attribution explain --episode <dir> [--json]")
		return ExitUsage
	}

	info, err := os.Stat(episodeDir)
	if os.IsNotExist(err) {
		fmt.Fprintf(stderr, "error: episode directory does not exist: %s\n", episodeDir)
		return ExitUsage
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot access episode directory: %v\n", err)
		return ExitError
	}
	if !info.IsDir() {
		fmt.Fprintf(stderr, "error: episode path is not a directory: %s\n", episodeDir)
		return ExitUsage
	}

	result, err := attribution.LoadOrCreateAttribution(episodeDir)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
		return ExitOK
	}

	fmt.Fprintln(stdout, "# x-harness Failure Attribution")
	fmt.Fprintf(stdout, "- episode_id: %s\n", result.EpisodeID)
	fmt.Fprintf(stdout, "- task_id: %s\n", result.TaskID)
	fmt.Fprintf(stdout, "- acceptance_status: %s\n", result.Verdict.AcceptanceStatus)
	fmt.Fprintf(stdout, "- admission_outcome: %s\n", result.Verdict.AdmissionOutcome)
	if result.Primary != nil {
		fmt.Fprintf(stdout, "- taxonomy: %s\n", result.Primary.Taxonomy)
		fmt.Fprintf(stdout, "- predicate: %s\n", result.Primary.Predicate)
		fmt.Fprintf(stdout, "- component_id: %s\n", result.Primary.ComponentID)
		fmt.Fprintf(stdout, "- confidence: %s\n", result.Primary.Confidence)
		fmt.Fprintf(stdout, "- rationale: %s\n", result.Primary.Rationale)
	} else {
		fmt.Fprintln(stdout, "- taxonomy: none")
		fmt.Fprintln(stdout, "- predicate: none")
	}

	return ExitOK
}
