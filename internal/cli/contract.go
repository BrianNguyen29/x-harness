package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/contract"
)

func handleContract(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness contract <check|help> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "check":
		return handleContractCheck(args[1:], stdout, stderr)
	case "help":
		fmt.Fprintln(stderr, "usage: x-harness contract check [--policy <path>] [--json] [paths...]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown contract subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness contract check [--policy <path>] [--json] [paths...]")
		return ExitUsage
	}
}

func handleContractCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	var policyPath string
	var jsonMode bool
	var paths []string

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "--policy":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --policy requires a value")
				return ExitUsage
			}
			policyPath = args[i+1]
			i += 2
		case "--json":
			jsonMode = true
			i++
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: x-harness contract check [--policy <path>] [--json] [paths...]")
			return ExitUsage
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
				return ExitUsage
			}
			paths = append(paths, arg)
			i++
		}
	}

	// Default policy
	if policyPath == "" {
		policyPath = "policies/contract-oracle.yaml"
	}

	// Default paths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Verify policy exists
	if _, err := os.Stat(policyPath); err != nil {
		fmt.Fprintf(stderr, "error: policy not found: %s\n", policyPath)
		return ExitUsage
	}

	result, err := contract.Check(policyPath, paths)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, err := result.ToJSON()
		if err != nil {
			fmt.Fprintf(stderr, "error: failed to format JSON: %v\n", err)
			return ExitError
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		renderContractResult(result, stdout)
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func renderContractResult(result *contract.Result, stdout io.Writer) {
	WriteLine(stdout, "# x-harness Contract Check")
	WriteLine(stdout, "")
	WriteLine(stdout, "policy: %s", result.Policy)
	WriteLine(stdout, "files_scanned: %d", result.FilesScanned)
	WriteLine(stdout, "violations: %d", len(result.Violations))
	WriteLine(stdout, "")

	if len(result.Violations) == 0 {
		WriteLine(stdout, "No violations.")
	} else {
		WriteLine(stdout, "| Rule ID | File | Line | Message |")
		WriteLine(stdout, "| :-- | :-- | :-- | :-- |")
		for _, v := range result.Violations {
			snippet := v.Snippet
			if len(snippet) > 60 {
				snippet = snippet[:60] + "..."
			}
			WriteLine(stdout, "| %s | %s | %d | %s |", v.RuleID, v.File, v.Line, v.Message)
			WriteLine(stdout, "| | | | `%s` |", snippet)
		}
	}
}
