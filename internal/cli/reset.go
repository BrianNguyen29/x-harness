package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func handleReset(args []string, stdout io.Writer, stderr io.Writer) int {
	confirmed := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--confirm":
			confirmed = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
		}
	}

	if !confirmed {
		WriteLine(stdout, "x-harness reset requires --confirm for safety.")
		WriteLine(stdout, "")
		WriteLine(stdout, "To reset harness state:")
		WriteLine(stdout, "  x-harness reset --confirm")
		WriteLine(stdout, "")
		WriteLine(stdout, "This will delete:")
		WriteLine(stdout, "  - .x-harness/tmp/")
		WriteLine(stdout, "  - .x-harness/cache/")
		return ExitError
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot get current directory: %v\n", err)
		return ExitError
	}

	WriteLine(stdout, "# x-harness clean --tmp --force")
	for _, dir := range []string{".x-harness/tmp", ".x-harness/cache"} {
		fullPath := filepath.Join(cwd, dir)
		if _, err := os.Stat(fullPath); err == nil {
			if err := os.RemoveAll(fullPath); err != nil {
				fmt.Fprintf(stderr, "error: cannot remove %s: %v\n", dir, err)
				return ExitError
			}
			WriteLine(stdout, "deleted: %s/", dir)
		} else {
			WriteLine(stdout, "not found (skipping): %s/", dir)
		}
	}
	WriteLine(stdout, "")
	WriteLine(stdout, "reset complete.")
	return ExitOK
}
