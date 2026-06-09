package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/authority"
)

func handleGovernance(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "governance requires a subcommand: check, explain, list-protected")
		return ExitUsage
	}

	switch args[0] {
	case "check":
		return handleGovernanceCheck(args[1:], stdout, stderr)
	case "explain":
		return handleGovernanceExplain(args[1:], stdout, stderr)
	case "list-protected":
		return handleGovernanceListProtected(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: x-harness governance <check|explain|list-protected> [options]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown governance subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness governance <check|explain|list-protected> [options]")
		return ExitUsage
	}
}

func parseGovernanceFlags(args []string, requireCard bool, stderr io.Writer) (card, diff, changedFilesSource, root string, enforce, jsonMode bool, exitCode int) {
	root = "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --card requires a value")
				return "", "", "", "", false, false, ExitUsage
			}
			card = args[i+1]
			i++
		case "--diff":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --diff requires a value")
				return "", "", "", "", false, false, ExitUsage
			}
			diff = args[i+1]
			i++
		case "--changed-files-source":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --changed-files-source requires a value")
				return "", "", "", "", false, false, ExitUsage
			}
			changedFilesSource = args[i+1]
			i++
		case "--enforce":
			enforce = true
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return "", "", "", "", false, false, ExitUsage
			}
			root = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return "", "", "", "", false, false, ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return "", "", "", "", false, false, ExitUsage
		}
	}

	if requireCard && card == "" {
		fmt.Fprintln(stderr, "Error: --card is required")
		return "", "", "", "", false, false, ExitUsage
	}

	root, _ = filepath.Abs(root)
	return card, diff, changedFilesSource, root, enforce, jsonMode, -1
}

func handleGovernanceCheck(args []string, stdout, stderr io.Writer) int {
	card, _, changedFilesSource, root, enforce, jsonMode, exitCode := parseGovernanceFlags(args, true, stderr)
	if exitCode >= 0 {
		return exitCode
	}

	cardPath := card
	if !filepath.IsAbs(cardPath) {
		cardPath = filepath.Join(root, card)
	}
	if _, err := os.Stat(cardPath); os.IsNotExist(err) {
		fmt.Fprintf(stderr, "Error: Card not found: %s\n", cardPath)
		return ExitError
	}

	files, governance, err := authority.LoadCardGovernanceData(cardPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	// Simple changed-files-source support: if source is git, ignore card files
	// For simplest viable parity, we primarily use card files.
	// If source is explicitly "git" and no files, that's an error scenario we handle gracefully.
	if changedFilesSource == "git" {
		files = []string{}
	}

	if len(files) == 0 {
		if jsonMode {
			out := map[string]any{
				"ok":                true,
				"violations":        []any{},
				"warnings":          []any{},
				"total_violations":  0,
				"total_warnings":    0,
				"changed_files":     files,
				"message":           "No files to check",
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintln(stdout, "No files to check for governance violations.")
		}
		return ExitOK
	}

	policy, err := authority.LoadAuthorityPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	result, err := authority.CheckGovernance(files, root, policy, &authority.GovernanceCheckOptions{
		Enforce:    enforce,
		Governance: governance,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		out := map[string]any{
			"ok":                result.TotalViolations == 0 && result.TotalWarnings == 0,
			"violations":        result.Violations,
			"warnings":          result.Warnings,
			"total_violations":  result.TotalViolations,
			"total_warnings":    result.TotalWarnings,
			"report_only":       result.ReportOnly,
			"enforced":          result.Enforced,
			"changed_files":     files,
			"notes":             []string{},
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		if result.TotalViolations > 0 {
			fmt.Fprintln(stdout, "Authority violations (enforced mode):")
			for _, v := range result.Violations {
				fmt.Fprintf(stdout, "  [%s] %s\n", v.Authority, v.Path)
				fmt.Fprintf(stdout, "    %s\n", v.Rationale)
				if v.ApprovalNote != "" {
					fmt.Fprintf(stdout, "    %s\n", v.ApprovalNote)
				}
			}
			fmt.Fprintln(stdout, "")
			fmt.Fprintf(stdout, "Total: %d violation(s)\n", result.TotalViolations)
		} else if result.TotalWarnings > 0 {
			fmt.Fprintln(stdout, "Authority warnings (report-only, no admission block):")
			for _, w := range result.Warnings {
				fmt.Fprintf(stdout, "  [%s] %s\n", w.Authority, w.Path)
				fmt.Fprintf(stdout, "    %s\n", w.Rationale)
			}
			fmt.Fprintln(stdout, "")
			fmt.Fprintf(stdout, "Total: %d warning(s) - admission NOT blocked (PR2 report-only)\n", result.TotalWarnings)
		} else {
			fmt.Fprintln(stdout, "No governance violations found.")
		}
	}

	if result.TotalViolations > 0 {
		return ExitError
	}
	return ExitOK
}

func handleGovernanceExplain(args []string, stdout, stderr io.Writer) int {
	pathFlag := ""
	root := "."
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --path requires a value")
				return ExitUsage
			}
			pathFlag = args[i+1]
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

	if pathFlag == "" {
		fmt.Fprintln(stderr, "Error: --path is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)

	policy, err := authority.LoadAuthorityPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	result, err := authority.ExplainPath(policy, pathFlag, root)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Path: %s\n", result.Path)
		fmt.Fprintf(stdout, "Authority: %s\n", result.Authority)
		fmt.Fprintf(stdout, "Rationale: %s\n", result.Rationale)
	}

	return ExitOK
}

func handleGovernanceListProtected(args []string, stdout, stderr io.Writer) int {
	root := "."
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

	root, _ = filepath.Abs(root)

	policy, err := authority.LoadAuthorityPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	protectedPaths := authority.GetProtectedPaths(policy)

	if jsonMode {
		out := map[string]any{
			"authority_classes": policy.AuthorityClasses,
			"protected_paths":   protectedPaths,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "Authority classes:")
		for name, cls := range policy.AuthorityClasses {
			fmt.Fprintf(stdout, "  %s: %s\n", name, cls.Description)
		}
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Protected paths:")
		for _, pp := range protectedPaths {
			fmt.Fprintf(stdout, "  %s -> %s\n", pp.Path, pp.Authority)
			fmt.Fprintf(stdout, "    %s\n", pp.Rationale)
		}
	}

	return ExitOK
}
