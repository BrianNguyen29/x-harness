package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/frozen"
)

func handleFrozen(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "frozen requires a subcommand: export, import, verify")
		return ExitUsage
	}

	switch args[0] {
	case "export":
		return handleFrozenExport(args[1:], stdout, stderr)
	case "import":
		return handleFrozenImport(args[1:], stdout, stderr)
	case "verify":
		return handleFrozenVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown frozen subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleFrozenExport(args []string, stdout, stderr io.Writer) int {
	root := "."
	out := ""
	frozenFlag := false
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--frozen":
			frozenFlag = true
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			out = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
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

	if !frozenFlag {
		fmt.Fprintln(stderr, "export currently requires --frozen")
		return ExitUsage
	}
	if out == "" {
		fmt.Fprintln(stderr, "error: --out <path> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	result, err := frozen.ExportFrozenBundle(root, out)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "frozen bundle written: %s\n", result.Out)
		fmt.Fprintf(stdout, "files: %d\n", result.FileCount)
	}
	return ExitOK
}

func handleFrozenImport(args []string, stdout, stderr io.Writer) int {
	bundle := ""
	target := ""
	frozenFlag := false
	dryRun := true
	merge := false
	force := false
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--frozen":
			frozenFlag = true
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
			if bundle == "" {
				bundle = args[i]
			} else {
				fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
				return ExitUsage
			}
		}
	}

	if !frozenFlag {
		fmt.Fprintln(stderr, "import currently requires --frozen")
		return ExitUsage
	}
	if bundle == "" {
		fmt.Fprintln(stderr, "error: bundle path is required")
		return ExitUsage
	}
	if target == "" {
		fmt.Fprintln(stderr, "error: --target <path> is required")
		return ExitUsage
	}

	if merge || force {
		dryRun = false
	}

	result, err := frozen.ImportFrozenBundle(bundle, target, dryRun, merge, force)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		if dryRun {
			fmt.Fprintf(stdout, "frozen import dry-run: %d file(s)\n", len(result.Planned))
		} else {
			fmt.Fprintf(stdout, "frozen import wrote %d file(s)\n", len(result.Written))
		}
		for _, conflict := range result.Conflicts {
			fmt.Fprintf(stdout, "conflict: %s\n", conflict)
		}
	}

	if !result.OK {
		fmt.Fprintln(stderr, "frozen import failed")
		return ExitError
	}
	return ExitOK
}

func handleFrozenVerify(args []string, stdout, stderr io.Writer) int {
	bundle := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			if bundle == "" {
				bundle = args[i]
			} else {
				fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
				return ExitUsage
			}
		}
	}

	if bundle == "" {
		fmt.Fprintln(stderr, "error: bundle path is required")
		return ExitUsage
	}

	result, err := frozen.VerifyFrozenBundle(bundle)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else if result.OK {
		fmt.Fprintf(stdout, "frozen bundle valid: %s\n", result.BundlePath)
		fmt.Fprintf(stdout, "files: %d\n", result.FileCount)
	} else {
		fmt.Fprintln(stdout, "frozen bundle invalid:")
		for _, e := range result.Errors {
			fmt.Fprintf(stdout, "- %s\n", e)
		}
	}

	if !result.OK {
		fmt.Fprintln(stderr, "frozen verify failed")
		return ExitError
	}
	return ExitOK
}
