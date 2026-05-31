package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func handleUninstall(args []string, stdout io.Writer, stderr io.Writer) int {
	preview := true
	apply := false
	force := false
	target := "."

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--preview":
			preview = true
		case "--apply":
			apply = true
		case "--force":
			force = true
		default:
			if strings.HasPrefix(args[i], "-") {
				WriteLine(stderr, "unknown flag: %s", args[i])
				return ExitUsage
			}
			if target == "." {
				target = args[i]
			} else {
				WriteLine(stderr, "usage: uninstall [target] [--preview|--apply] [--force]")
				return ExitUsage
			}
		}
	}

	if apply {
		preview = false
	}

	targetDir, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(stderr, "error resolving target: %v\n", err)
		return ExitError
	}

	manifest, err := readManifest(targetDir)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	type uninstallItem struct {
		entry    ManifestEntry
		status   string // present, absent, modified
		destPath string
	}

	var items []uninstallItem
	for _, entry := range manifest.Entries {
		destPath, err := resolveManifestEntryPath(targetDir, entry)
		if err != nil {
			fmt.Fprintf(stderr, "warning: skipping unsafe path %s: %v\n", entry.Path, err)
			continue
		}

		status := "present"
		info, err := os.Stat(destPath)
		if err != nil {
			status = "absent"
		} else if info.IsDir() {
			// Directories should not appear in manifest, but if they do, treat as present
			status = "present"
		} else {
			hash, err := fileHash(destPath)
			if err != nil || hash != entry.Hash {
				status = "modified"
			}
		}

		items = append(items, uninstallItem{entry: entry, status: status, destPath: destPath})
	}

	if preview {
		WriteLine(stdout, "# x-harness uninstall (preview)")
		WriteLine(stdout, "manifest: %d entries", len(manifest.Entries))
		if len(items) == 0 {
			WriteLine(stdout, "no managed entries to remove")
			return ExitOK
		}
		for _, it := range items {
			switch it.status {
			case "absent":
				WriteLine(stdout, "absent: %s (already removed)", it.entry.Path)
			case "modified":
				WriteLine(stdout, "modified: %s (would remove; content differs from manifest)", it.entry.Path)
			default:
				WriteLine(stdout, "would remove: %s", it.entry.Path)
			}
		}
		WriteLine(stdout, "")
		WriteLine(stdout, "To apply, run: xh uninstall --apply --force")
		return ExitOK
	}

	// Apply
	if !force {
		WriteLine(stderr, "error: uninstall --apply requires --force")
		WriteLine(stderr, "Run with --preview first, then use --apply --force")
		return ExitError
	}

	WriteLine(stdout, "# x-harness uninstall (apply)")

	// Backup existing managed content first
	var toBackup []uninstallItem
	for _, it := range items {
		if it.status == "present" || it.status == "modified" {
			toBackup = append(toBackup, it)
		}
	}

	if len(toBackup) > 0 {
		timestamp := time.Now().UTC().Format("20060102T150405Z")
		backupDir := filepath.Join(targetDir, ".x-harness", "backup", timestamp)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			fmt.Fprintf(stderr, "error: failed to create backup directory: %v\n", err)
			return ExitError
		}
		for _, it := range toBackup {
			backupPath := filepath.Join(backupDir, filepath.FromSlash(it.entry.Path))
			if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
				fmt.Fprintf(stderr, "warning: failed to create backup subdir for %s: %v\n", it.entry.Path, err)
				continue
			}
			if err := copyFile(it.destPath, backupPath); err != nil {
				fmt.Fprintf(stderr, "warning: failed to backup %s: %v\n", it.entry.Path, err)
				continue
			}
			WriteLine(stdout, "backup: %s -> %s", it.entry.Path, backupPath)
		}
	}

	removed := 0
	for _, it := range items {
		if it.status == "absent" {
			WriteLine(stdout, "already absent: %s", it.entry.Path)
			continue
		}
		if err := os.Remove(it.destPath); err != nil {
			fmt.Fprintf(stderr, "warning: failed to remove %s: %v\n", it.entry.Path, err)
			continue
		}
		WriteLine(stdout, "removed: %s", it.entry.Path)
		removed++
	}

	// Remove manifest last
	manifestFile := filepath.Join(targetDir, manifestPath)
	if err := os.Remove(manifestFile); err != nil {
		fmt.Fprintf(stderr, "warning: failed to remove manifest: %v\n", err)
	} else {
		WriteLine(stdout, "removed: %s", manifestPath)
	}

	WriteLine(stdout, "uninstall complete: %d files removed", removed)
	return ExitOK
}
