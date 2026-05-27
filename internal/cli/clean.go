package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var cleanProtectedPaths = []string{
	"AGENTS.md",
	"X_HARNESS.md",
	"README.md",
	"templates",
	"schemas",
	"policies",
	"docs",
	"adapters",
	"examples",
	"packages",
	".git",
}

func isCleanProtected(targetPath string, cwd string) bool {
	rel, err := filepath.Rel(cwd, targetPath)
	if err != nil {
		return false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		for _, protected := range cleanProtectedPaths {
			if part == protected {
				return true
			}
		}
	}
	return false
}

type cleanAction struct {
	Type string
	Path string
	Note string
}

func handleClean(args []string, stdout, stderr io.Writer) int {
	dryRunFlag := false
	tmpFlag := false
	resetCardFlag := false
	archiveSuccessFlag := false
	forceFlag := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRunFlag = true
		case "--tmp":
			tmpFlag = true
		case "--reset-card":
			resetCardFlag = true
		case "--archive-success":
			archiveSuccessFlag = true
		case "--force":
			forceFlag = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot get current directory: %v\n", err)
		return ExitError
	}

	wouldMutate := tmpFlag || resetCardFlag || archiveSuccessFlag
	// Default dry-run when no force or when --dry-run explicitly set
	// TS behavior: dryRun = opts.dryRun !== false && (!opts.force || !wouldMutate)
	// In Go: if --dry-run is passed, dryRun is true.
	// If no flags that would mutate are set, it's effectively dry-run/no-op.
	// Mutating clean requires --force.
	dryRun := dryRunFlag || !forceFlag || !wouldMutate

	var actions []cleanAction

	if tmpFlag {
		for _, dir := range []string{".x-harness/tmp", ".x-harness/cache"} {
			fullPath := filepath.Join(cwd, dir)
			if _, err := os.Stat(fullPath); err == nil {
				actions = append(actions, cleanAction{Type: "delete", Path: fullPath, Note: fmt.Sprintf("remove %s", dir)})
			}
		}
	}

	if resetCardFlag {
		cardPath := filepath.Join(cwd, "completion-card.yaml")
		if _, err := os.Stat(cardPath); err == nil {
			backupPath := filepath.Join(cwd, fmt.Sprintf("completion-card.yaml.bak.%d", time.Now().UnixMilli()))
			actions = append(actions, cleanAction{Type: "rename", Path: fmt.Sprintf("%s -> %s", cardPath, backupPath), Note: "reset completion card"})
		} else {
			fmt.Fprintln(stdout, "No completion-card.yaml found to reset.")
		}
	}

	if archiveSuccessFlag {
		cardPath := filepath.Join(cwd, "completion-card.yaml")
		if _, err := os.Stat(cardPath); err == nil {
			actions = append(actions, cleanAction{Type: "archive-accept", Path: cardPath, Note: "archive accepted card"})
		} else {
			fmt.Fprintln(stdout, "No completion-card.yaml found to archive.")
		}
	}

	// Safety: filter out protected paths
	var safeActions []cleanAction
	for _, a := range actions {
		target := strings.Split(a.Path, " -> ")[0]
		if isCleanProtected(target, cwd) {
			fmt.Fprintf(stdout, "SKIPPED (protected): %s\n", a.Path)
			continue
		}
		safeActions = append(safeActions, a)
	}

	if len(safeActions) == 0 {
		fmt.Fprintln(stdout, "Nothing to clean.")
		fmt.Fprintln(stdout, "Use --tmp, --reset-card, or --archive-success to specify what to clean.")
		return ExitOK
	}

	if dryRun {
		fmt.Fprintln(stdout, "# x-harness clean (dry-run)")
		for _, a := range safeActions {
			fmt.Fprintf(stdout, "would %s: %s (%s)\n", a.Type, a.Path, a.Note)
		}
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "To apply, run again with --force")
		return ExitOK
	}

	// Execute mutations
	fmt.Fprintln(stdout, "# x-harness clean (applying)")
	for _, a := range safeActions {
		switch a.Type {
		case "delete":
			if err := os.RemoveAll(a.Path); err != nil {
				fmt.Fprintf(stderr, "failed: %s - %v\n", a.Path, err)
			} else {
				fmt.Fprintf(stdout, "deleted: %s\n", a.Path)
			}
		case "rename":
			parts := strings.SplitN(a.Path, " -> ", 2)
			if len(parts) == 2 {
				src := strings.TrimSpace(parts[0])
				dest := strings.TrimSpace(parts[1])
				if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
					fmt.Fprintf(stderr, "failed: %s - %v\n", a.Path, err)
					continue
				}
				if err := os.Rename(src, dest); err != nil {
					fmt.Fprintf(stderr, "failed: %s - %v\n", a.Path, err)
				} else {
					fmt.Fprintf(stdout, "renamed: %s\n", a.Path)
				}
			}
		case "archive-accept":
			// Re-verify content at move time to avoid TOCTOU window
			content, err := os.ReadFile(a.Path)
			if err != nil {
				fmt.Fprintf(stderr, "failed: %s - %v\n", a.Path, err)
				continue
			}
			// Simple YAML parsing for acceptance_status and admission.outcome
			// We parse as string and look for patterns
			data := string(content)
			acceptanceStatus := ""
			admissionOutcome := ""
			for _, line := range strings.Split(data, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "acceptance_status:") {
					acceptanceStatus = strings.TrimSpace(strings.TrimPrefix(line, "acceptance_status:"))
				}
				if strings.HasPrefix(line, "outcome:") {
					admissionOutcome = strings.TrimSpace(strings.TrimPrefix(line, "outcome:"))
				}
			}
			if acceptanceStatus != "accepted" || admissionOutcome != "success" {
				fmt.Fprintln(stdout, "Current completion card is not accepted; skipping archive.")
				continue
			}
			archiveDir := filepath.Join(cwd, ".x-harness", "archive")
			archiveName := fmt.Sprintf("completion-card-%d.yaml", time.Now().UnixMilli())
			archivePath := filepath.Join(archiveDir, archiveName)
			if err := os.MkdirAll(archiveDir, 0755); err != nil {
				fmt.Fprintf(stderr, "failed: %s - %v\n", a.Path, err)
				continue
			}
			if err := os.Rename(a.Path, archivePath); err != nil {
				fmt.Fprintf(stderr, "failed: %s - %v\n", a.Path, err)
			} else {
				fmt.Fprintf(stdout, "archived: %s -> %s\n", a.Path, archivePath)
			}
		}
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "clean complete.")
	return ExitOK
}
