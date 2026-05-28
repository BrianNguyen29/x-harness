package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func handleRepair(args []string, stdout io.Writer, stderr io.Writer) int {
	preview := true
	apply := false
	target := "."
	assetRootFlag := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--preview":
			preview = true
		case "--apply":
			apply = true
		case "--asset-root":
			if i+1 >= len(args) {
				WriteLine(stderr, "usage: repair [target] [--preview|--apply] [--asset-root <path>]")
				WriteLine(stderr, "missing value for --asset-root")
				return ExitUsage
			}
			assetRootFlag = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				WriteLine(stderr, "unknown flag: %s", args[i])
				return ExitUsage
			}
			if target == "." {
				target = args[i]
			} else {
				WriteLine(stderr, "usage: repair [target] [--preview|--apply] [--asset-root <path>]")
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

	assetRoot, err := resolveInitAssetRoot(assetRootFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	type repairItem struct {
		entry    ManifestEntry
		status   string // missing, modified, ok
		destPath string
		srcPath  string
	}

	var items []repairItem
	for _, entry := range manifest.Entries {
		destPath, err := resolveManifestEntryPath(targetDir, entry)
		if err != nil {
			fmt.Fprintf(stderr, "warning: skipping unsafe path %s: %v\n", entry.Path, err)
			continue
		}

		status := "ok"
		if _, err := os.Stat(destPath); err != nil {
			status = "missing"
		} else {
			hash, err := fileHash(destPath)
			if err != nil || hash != entry.Hash {
				status = "modified"
			}
		}

		// Resolve source path from asset root using the relative path in manifest
		srcPath := filepath.Join(assetRoot, entry.Path)
		if _, err := os.Stat(srcPath); err != nil {
			// Fallback: try to find in common asset directories
			for _, base := range []string{"templates", "policies", "schemas", "docs", "examples", "adapters"} {
				alt := filepath.Join(assetRoot, base, entry.Path)
				if _, err := os.Stat(alt); err == nil {
					srcPath = alt
					break
				}
			}
		}

		items = append(items, repairItem{entry: entry, status: status, destPath: destPath, srcPath: srcPath})
	}

	var toRepair []repairItem
	for _, it := range items {
		if it.status != "ok" {
			toRepair = append(toRepair, it)
		}
	}

	if preview {
		WriteLine(stdout, "# x-harness repair (preview)")
		WriteLine(stdout, "manifest: %d entries", len(manifest.Entries))
		if len(toRepair) == 0 {
			WriteLine(stdout, "no drift detected; all managed files match manifest")
			return ExitOK
		}
		for _, it := range toRepair {
			if it.status == "missing" {
				WriteLine(stdout, "missing: %s (would restore from %s)", it.entry.Path, it.srcPath)
			} else {
				WriteLine(stdout, "modified: %s (would overwrite from %s)", it.entry.Path, it.srcPath)
			}
		}
		WriteLine(stdout, "")
		WriteLine(stdout, "To apply, run: x-harness repair --apply")
		return ExitOK
	}

	// Apply
	WriteLine(stdout, "# x-harness repair (apply)")
	if len(toRepair) == 0 {
		WriteLine(stdout, "no drift detected; all managed files match manifest")
		return ExitOK
	}

	restored := 0
	backedUp := 0
	for _, it := range toRepair {
		if _, err := os.Stat(it.srcPath); err != nil {
			fmt.Fprintf(stderr, "error: source not found for %s: %v\n", it.entry.Path, err)
			continue
		}

		if it.status == "modified" {
			backupPath := it.destPath + ".bak." + fmt.Sprintf("%d", time.Now().UnixMilli())
			if err := os.Rename(it.destPath, backupPath); err != nil {
				fmt.Fprintf(stderr, "error: failed to backup %s: %v\n", it.destPath, err)
				continue
			}
			WriteLine(stdout, "backup: %s -> %s", it.entry.Path, backupPath)
			backedUp++
		}

		if err := os.MkdirAll(filepath.Dir(it.destPath), 0755); err != nil {
			fmt.Fprintf(stderr, "error: failed to create directory for %s: %v\n", it.entry.Path, err)
			continue
		}
		if err := copyFile(it.srcPath, it.destPath); err != nil {
			fmt.Fprintf(stderr, "error: failed to restore %s: %v\n", it.entry.Path, err)
			continue
		}
		WriteLine(stdout, "restored: %s", it.entry.Path)
		restored++
	}

	WriteLine(stdout, "repair complete: %d restored, %d backed up", restored, backedUp)
	return ExitOK
}

func copyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, info.Mode())
}
