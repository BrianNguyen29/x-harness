package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/scanner"
)

func handleScan(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness scan <adapter|skill|managed> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "adapter":
		return handleScanAdapter(args[1:], stdout, stderr)
	case "skill":
		return handleScanSkill(args[1:], stdout, stderr)
	case "managed":
		return handleScanManaged(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: x-harness scan <adapter|skill|managed> [options]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown scan subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness scan <adapter|skill|managed> [options]")
		return ExitUsage
	}
}

func parseScanFlags(args []string, stderr io.Writer) (jsonMode bool, root string, remaining []string, exitCode int) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return false, "", nil, ExitUsage
			}
			remaining = append(remaining, args[i])
		}
	}
	return jsonMode, root, remaining, -1
}

func handleScanAdapter(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode, root, _, exitCode := parseScanFlags(args, stderr)
	if exitCode >= 0 {
		return exitCode
	}

	if root == "" {
		root = findAdaptersRepoRoot()
		if root == "" {
			root = "."
		}
	}
	adaptersDir := filepath.Join(root, "adapters")

	info, err := os.Stat(adaptersDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "adapters directory not found: %s\n", adaptersDir)
		return ExitError
	}

	rules := scanner.DefaultRules()
	result, err := scanner.Scan(rules, []string{adaptersDir})
	if err != nil {
		fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return ExitError
	}

	return renderScanResult(result, jsonMode, stdout, stderr)
}

func handleScanSkill(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode, _, remaining, exitCode := parseScanFlags(args, stderr)
	if exitCode >= 0 {
		return exitCode
	}
	if len(remaining) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness scan skill <path> [--json]")
		return ExitUsage
	}

	path := remaining[0]
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(stderr, "path not found: %s\n", path)
		return ExitError
	}

	rules := scanner.DefaultRules()
	result, err := scanner.Scan(rules, []string{path})
	if err != nil {
		fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return ExitError
	}

	return renderScanResult(result, jsonMode, stdout, stderr)
}

func handleScanManaged(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode, root, _, exitCode := parseScanFlags(args, stderr)
	if exitCode >= 0 {
		return exitCode
	}

	if root == "" {
		root = findAdaptersRepoRoot()
		if root == "" {
			root = "."
		}
	}

	// Collect all files that contain managed block markers
	var managedFiles []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// Skip common large/binary directories
		rel, _ := filepath.Rel(root, path)
		lower := strings.ToLower(rel)
		if strings.Contains(lower, ".git") || strings.Contains(lower, "node_modules") || strings.Contains(lower, "vendor") {
			return nil
		}
		// Skip files larger than 1MB
		if info.Size() > 1024*1024 {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(content), "BEGIN X-HARNESS MANAGED CONTRACT") {
			managedFiles = append(managedFiles, path)
		}
		return nil
	})

	if len(managedFiles) == 0 {
		result := &scanner.Result{
			FilesScanned: 0,
			Findings:     []scanner.Finding{},
			Summary: scanner.Summary{
				Low:          0,
				Medium:       0,
				High:         0,
				Total:        0,
				Risk:         "none",
				FilesScanned: 0,
			},
		}
		return renderScanResult(result, jsonMode, stdout, stderr)
	}

	rules := scanner.DefaultRules()
	result, err := scanner.Scan(rules, managedFiles)
	if err != nil {
		fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return ExitError
	}

	return renderScanResult(result, jsonMode, stdout, stderr)
}

func renderScanResult(result *scanner.Result, jsonMode bool, stdout io.Writer, _ io.Writer) int {
	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "# x-harness Static Scan")
	WriteLine(stdout, "")
	WriteLine(stdout, "files_scanned: %d", result.FilesScanned)
	WriteLine(stdout, "findings: %d", len(result.Findings))
	WriteLine(stdout, "")

	if len(result.Findings) == 0 {
		WriteLine(stdout, "No findings.")
	} else {
		WriteLine(stdout, "| Severity | Category | Rule | File | Line | Snippet |")
		WriteLine(stdout, "| :-- | :-- | :-- | :-- | :-- | :-- |")
		for _, f := range result.Findings {
			snippet := f.Snippet
			if len(snippet) > 60 {
				snippet = snippet[:60] + "..."
			}
			waivable := "no"
			if f.Waivable {
				waivable = "yes"
			}
			WriteLine(stdout, "| %s | %s | %s | %s | %d | %s |", f.Severity, f.Category, f.RuleID, f.File, f.Line, snippet)
			_ = waivable
		}
	}

	WriteLine(stdout, "")
	WriteLine(stdout, "Summary:")
	WriteLine(stdout, "  low:    %d", result.Summary.Low)
	WriteLine(stdout, "  medium: %d", result.Summary.Medium)
	WriteLine(stdout, "  high:   %d", result.Summary.High)
	WriteLine(stdout, "  total:  %d", result.Summary.Total)
	WriteLine(stdout, "  risk:   %s", result.Summary.Risk)

	return ExitOK
}
