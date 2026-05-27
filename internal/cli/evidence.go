package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/evidence"
)

func handleEvidence(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness evidence validate --index <path> [--json]")
		return ExitUsage
	}

	subcommand := args[0]
	switch subcommand {
	case "validate":
		return handleEvidenceValidate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown evidence subcommand: %s\n", subcommand)
		fmt.Fprintln(stderr, "usage: x-harness evidence validate --index <path> [--json]")
		return ExitUsage
	}
}

func handleEvidenceValidate(args []string, stdout io.Writer, stderr io.Writer) int {
	indexPath := "evidence/index.jsonl"
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--index":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --index requires a value")
				return ExitUsage
			}
			indexPath = args[i+1]
			i++
		case "--json":
			jsonMode = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	ok, errs, count, err := evidence.ValidateIndexFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "error: evidence index not found: %s\n", indexPath)
			return ExitUsage
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		result := map[string]any{
			"ok":          ok,
			"errors":      errs,
			"entry_count": count,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
		if !ok {
			return ExitError
		}
		return ExitOK
	}

	if ok {
		fmt.Fprintf(stdout, "Evidence index valid (%d entries).\n", count)
		return ExitOK
	}

	fmt.Fprintln(stderr, "Evidence index invalid:")
	for _, e := range errs {
		fmt.Fprintf(stderr, "- %s\n", e)
	}
	return ExitError
}
