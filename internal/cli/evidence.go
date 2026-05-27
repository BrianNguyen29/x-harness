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
		fmt.Fprintln(stderr, "evidence requires a subcommand: validate, index")
		return ExitUsage
	}

	subcommand := args[0]
	switch subcommand {
	case "validate":
		return handleEvidenceValidate(args[1:], stdout, stderr)
	case "index":
		return handleEvidenceIndex(args[1:], stdout, stderr)
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

func handleEvidenceIndex(args []string, stdout io.Writer, stderr io.Writer) int {
	var opts evidence.IndexOptions
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--episode":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --episode requires a value")
				return ExitUsage
			}
			opts.Episode = args[i+1]
			i++
		case "--card":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --card requires a value")
				return ExitUsage
			}
			opts.Card = args[i+1]
			i++
		case "--task-id":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --task-id requires a value")
				return ExitUsage
			}
			opts.TaskID = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			opts.Out = args[i+1]
			i++
		case "--redact":
			opts.Redact = true
		case "--redacted-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --redacted-dir requires a value")
				return ExitUsage
			}
			opts.RedactedDir = args[i+1]
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

	if opts.Episode == "" && opts.Card == "" {
		fmt.Fprintln(stderr, "evidence index requires --episode or --card")
		return ExitUsage
	}

	result, err := evidence.BuildIndex(opts)
	if err != nil {
		if result != nil && !result.OK {
			if jsonMode {
				out := map[string]any{
					"ok":     false,
					"errors": result.Errors,
				}
				for k, v := range structToMap(result) {
					if k != "ok" && k != "errors" {
						out[k] = v
					}
				}
				data, _ := json.MarshalIndent(out, "", "  ")
				fmt.Fprintln(stdout, string(data))
			} else {
				fmt.Fprintf(stderr, "evidence index validation failed: %s\n", strings.Join(result.Errors, "; "))
			}
			return ExitError
		}
		if jsonMode {
			out := map[string]any{
				"ok":     false,
				"errors": []string{err.Error()},
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stderr, "error: %v\n", err)
		}
		return ExitError
	}

	if jsonMode {
		out := structToMap(result)
		out["ok"] = true
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "Evidence index written.")
		fmt.Fprintf(stdout, "- task_id: %s\n", result.TaskID)
		fmt.Fprintf(stdout, "- entries: %d\n", result.EntryCount)
		fmt.Fprintf(stdout, "- index_hash: %s\n", result.IndexHash)
		if result.OutPath != "" {
			fmt.Fprintf(stdout, "- out: %s\n", result.OutPath)
		}
		if result.RedactedDir != "" {
			fmt.Fprintf(stdout, "- redacted_dir: %s\n", result.RedactedDir)
		}
		for _, warning := range result.Warnings {
			fmt.Fprintf(stdout, "warning: %s\n", warning)
		}
	}
	return ExitOK
}

func structToMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
