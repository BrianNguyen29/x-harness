package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/BrianNguyen29/x-harness/internal/attribution"
)

func handleAttribution(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "attribution requires a subcommand: explain, report")
		return ExitUsage
	}

	subcommand := args[0]
	switch subcommand {
	case "explain":
		return handleAttributionExplain(args[1:], stdout, stderr)
	case "report":
		return handleAttributionReport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown attribution subcommand: %s\n", subcommand)
		fmt.Fprintln(stderr, "usage: x-harness attribution explain --episode <dir> [--json]")
		fmt.Fprintln(stderr, "       x-harness attribution report [--episodes-dir <dir>] [--group-by <field>] [--since <duration>] [--json]")
		return ExitUsage
	}
}

func handleAttributionExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	episodeDir := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
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

func handleAttributionReport(args []string, stdout io.Writer, stderr io.Writer) int {
	episodesDir := ".x-harness/episodes"
	groupBy := "predicate"
	since := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--episodes-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --episodes-dir requires a value")
				return ExitUsage
			}
			episodesDir = args[i+1]
			i++
		case "--group-by":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --group-by requires a value")
				return ExitUsage
			}
			groupBy = args[i+1]
			i++
		case "--since":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --since requires a value")
				return ExitUsage
			}
			since = args[i+1]
			i++
		case "--json":
			jsonMode = true
		default:
			if len(args[i]) > 0 && args[i][0] == '-' {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				fmt.Fprintln(stderr, "usage: x-harness attribution report [--episodes-dir <dir>] [--group-by <field>] [--since <duration>] [--json]")
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	if groupBy != "predicate" && groupBy != "taxonomy" && groupBy != "component" {
		fmt.Fprintln(stderr, "--group-by must be predicate, taxonomy, or component")
		return ExitUsage
	}

	attributions, err := attribution.ListAttributions(episodesDir)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	attributions = attribution.FilterSince(attributions, since)
	report := attribution.BuildAttributionReport(attributions, groupBy)

	if jsonMode {
		if err := WriteJSON(stdout, report); err != nil {
			return ExitError
		}
		return ExitOK
	}

	fmt.Fprintln(stdout, "# x-harness Attribution Report")
	fmt.Fprintf(stdout, "- group_by: %s\n", report.GroupBy)
	fmt.Fprintf(stdout, "- total_episodes: %d\n", report.TotalEpisodes)
	fmt.Fprintf(stdout, "- withheld_episodes: %d\n", report.WithheldEpisodes)
	fmt.Fprintf(stdout, "- unknown_rate: %g\n", report.UnknownRate)
	if report.EntropyWarning != nil {
		fmt.Fprintf(stdout, "- entropy_warning: %s\n", *report.EntropyWarning)
	}
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "## Groups")
	if len(report.Groups) == 0 {
		fmt.Fprintln(stdout, "None.")
	} else {
		for _, group := range report.Groups {
			fmt.Fprintf(stdout, "- %s: %d\n", group.Key, group.Count)
		}
	}

	return ExitOK
}
