package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/federation"
)

func handleFederation(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "federation requires a subcommand: export-patterns, import-patterns, validate")
		return ExitUsage
	}

	switch args[0] {
	case "export-patterns":
		return handleFederationExport(args[1:], stdout, stderr)
	case "import-patterns":
		return handleFederationImport(args[1:], stdout, stderr)
	case "validate":
		return handleFederationValidate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown federation subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleFederationExport(args []string, stdout, stderr io.Writer) int {
	root := "."
	indexPath := "evidence/index.jsonl"
	out := ""
	tenant := ""
	source := "local"
	optIn := false
	redacted := false
	benchmarkReport := ""
	policy := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--index":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --index requires a value")
				return ExitUsage
			}
			indexPath = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			out = args[i+1]
			i++
		case "--tenant":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --tenant requires a value")
				return ExitUsage
			}
			tenant = args[i+1]
			i++
		case "--source":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --source requires a value")
				return ExitUsage
			}
			source = args[i+1]
			i++
		case "--opt-in":
			optIn = true
		case "--redacted":
			redacted = true
		case "--benchmark-report":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --benchmark-report requires a value")
				return ExitUsage
			}
			benchmarkReport = args[i+1]
			i++
		case "--policy":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --policy requires a value")
				return ExitUsage
			}
			policy = args[i+1]
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

	if out == "" {
		fmt.Fprintln(stderr, "error: --out <path> is required")
		return ExitUsage
	}
	if tenant == "" {
		fmt.Fprintln(stderr, "error: --tenant <id> is required")
		return ExitUsage
	}

	result, err := federation.ExportFederationPatterns(root, indexPath, out, tenant, source, optIn, redacted, benchmarkReport, policy)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "requires explicit --opt-in") || strings.Contains(msg, "requires --redacted") || strings.Contains(msg, "requires a non-empty --tenant") {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitUsage
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "federation patterns written: %s\n", result.OutPath)
		fmt.Fprintf(stdout, "records: %d\n", result.RecordCount)
	}
	return ExitOK
}

func handleFederationImport(args []string, stdout, stderr io.Writer) int {
	root := "."
	target := ".x-harness/federation/imported-patterns.jsonl"
	dryRun := true
	merge := false
	force := false
	jsonMode := false
	patternsPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--target":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --target requires a value")
				return ExitUsage
			}
			target = args[i+1]
			i++
		case "--dry-run":
			dryRun = true
		case "--no-dry-run":
			dryRun = false
		case "--merge":
			merge = true
		case "--force":
			force = true
		case "--json":
			jsonMode = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			if patternsPath == "" {
				patternsPath = args[i]
			} else {
				fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
				return ExitUsage
			}
		}
	}

	if patternsPath == "" {
		fmt.Fprintln(stderr, "error: patterns path is required")
		return ExitUsage
	}

	if merge || force {
		dryRun = false
	}

	result, err := federation.ImportFederationPatterns(root, patternsPath, target, dryRun, merge, force)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		if result.OK {
			if dryRun {
				fmt.Fprintf(stdout, "federation import dry-run: %d record(s)\n", result.PlannedCount)
			} else {
				fmt.Fprintf(stdout, "federation import wrote %d record(s)\n", result.WrittenCount)
			}
		} else {
			fmt.Fprintln(stdout, "federation import failed:")
			for _, e := range result.Errors {
				fmt.Fprintf(stdout, "- %s\n", e)
			}
		}
	}

	if !result.OK {
		fmt.Fprintln(stderr, "federation import failed")
		return ExitError
	}
	return ExitOK
}

func handleFederationValidate(args []string, stdout, stderr io.Writer) int {
	jsonMode := false
	patternsPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			if patternsPath == "" {
				patternsPath = args[i]
			} else {
				fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
				return ExitUsage
			}
		}
	}

	if patternsPath == "" {
		fmt.Fprintln(stderr, "error: patterns path is required")
		return ExitUsage
	}

	result, err := federation.ValidateFederationPatternFile(patternsPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	output := map[string]any{
		"ok":                   result.OK,
		"record_count":         len(result.Patterns),
		"errors":               result.Errors,
		"admission_authority": false,
	}

	if jsonMode {
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		if result.OK {
			fmt.Fprintf(stdout, "federation patterns valid: %d\n", len(result.Patterns))
		} else {
			fmt.Fprintln(stdout, "federation patterns invalid:")
			for _, e := range result.Errors {
				fmt.Fprintf(stdout, "- %s\n", e)
			}
		}
	}

	if !result.OK {
		fmt.Fprintln(stderr, "federation validation failed")
		return ExitError
	}
	return ExitOK
}
