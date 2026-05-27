package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/episode"
)

func handleEpisode(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "episode requires a subcommand: inspect")
		return ExitUsage
	}

	switch args[0] {
	case "inspect":
		return handleEpisodeInspect(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown episode subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleEpisodeInspect(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonOutput := false
	var path string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			if path != "" {
				fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
				return ExitUsage
			}
			path = args[i]
		}
	}

	if path == "" {
		fmt.Fprintln(stderr, "Error: episode inspect requires a path argument")
		return ExitUsage
	}

	result, err := episode.InspectEpisode(path)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "Error: failed to serialize result: %v\n", err)
			return ExitError
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "# x-harness Episode Inspect")
		fmt.Fprintf(stdout, "- ok: %v\n", result.OK)
		episodeID := "unknown"
		if result.EpisodeID != nil {
			episodeID = *result.EpisodeID
		}
		taskID := "unknown"
		if result.TaskID != nil {
			taskID = *result.TaskID
		}
		fmt.Fprintf(stdout, "- episode_id: %s\n", episodeID)
		fmt.Fprintf(stdout, "- task_id: %s\n", taskID)
		fmt.Fprintf(stdout, "- file_count: %d\n", result.FileCount)
		if len(result.Errors) > 0 {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "## Errors")
			for _, error := range result.Errors {
				fmt.Fprintf(stdout, "- %s\n", error)
			}
		}
		if len(result.Warnings) > 0 {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "## Warnings")
			for _, warning := range result.Warnings {
				fmt.Fprintf(stdout, "- %s\n", warning)
			}
		}
	}

	if !result.OK {
		return ExitError
	}
	return ExitOK
}
