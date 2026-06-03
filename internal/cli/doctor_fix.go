package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// DoctorFix is the deterministic repair plan and (optionally) the
// application result. It is attached to the doctor JSON output when
// the --fix flag is set. The CLI never mutates the workspace unless
// --fix is paired with --confirm.
//
// V1 scope is intentionally narrow: restore MISSING files that are
// tracked by the manifest and have a known source in the asset root.
// Files that exist but differ from the manifest hash are NOT
// auto-fixed; users should run `xh repair --apply` to overwrite
// drift. This keeps the fix deterministic and prevents `xh doctor
// --fix` from clobbering user edits to managed files.
type DoctorFix struct {
	ManifestFound bool     `json:"manifest_found"`
	Profile       string   `json:"profile,omitempty"`
	AssetRoot     string   `json:"asset_root,omitempty"`
	DryRun        bool     `json:"dry_run"`
	Confirmed     bool     `json:"confirmed"`
	Applied       []string `json:"applied,omitempty"`
	Skipped       []string `json:"skipped,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

// doctorFixItem is an internal representation of one deterministic
// fix candidate. Action is "restore" for missing files, "skip" for
// anything else (modified, unsafe path, missing source).
type doctorFixItem struct {
	rel    string // manifest-relative path
	dest   string // absolute destination on disk
	src    string // absolute source in asset root
	action string // "restore" or "skip"
	reason string
}

// buildDoctorFixPlan enumerates deterministic fixes. See DoctorFix
// godoc for the V1 scope contract.
func buildDoctorFixPlan(root, assetRoot string, manifest *Manifest) []doctorFixItem {
	var items []doctorFixItem
	if manifest == nil {
		return items
	}
	for _, e := range manifest.Entries {
		if err := validateManifestPath(e.Path); err != nil {
			items = append(items, doctorFixItem{
				rel: e.Path, action: "skip", reason: "unsafe path: " + err.Error(),
			})
			continue
		}
		destPath, err := resolveManifestEntryPath(root, e)
		if err != nil {
			items = append(items, doctorFixItem{
				rel: e.Path, action: "skip", reason: err.Error(),
			})
			continue
		}
		if _, err := os.Stat(destPath); err == nil {
			// File exists. V1 never overwrites; recommend `xh repair
			// --apply` for drift that the user explicitly wants to
			// resolve.
			items = append(items, doctorFixItem{
				rel: e.Path, dest: destPath, action: "skip",
				reason: "file present; use `xh repair --apply` to overwrite drift",
			})
			continue
		}
		// File is missing; try to locate a source in the asset root.
		srcPath := filepath.Join(assetRoot, filepath.FromSlash(e.Path))
		if _, err := os.Stat(srcPath); err != nil {
			for _, base := range []string{"schemas", "policies", "templates", "docs", "examples", "adapters"} {
				alt := filepath.Join(assetRoot, base, e.Path)
				if _, err := os.Stat(alt); err == nil {
					srcPath = alt
					break
				}
			}
			if _, err := os.Stat(srcPath); err != nil {
				items = append(items, doctorFixItem{
					rel: e.Path, action: "skip",
					reason: "no source found in asset root",
				})
				continue
			}
		}
		items = append(items, doctorFixItem{
			rel: e.Path, dest: destPath, src: srcPath, action: "restore",
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].rel < items[j].rel
	})
	return items
}

// runDoctorFix plans (and optionally applies) deterministic repairs.
// The function never mutates the workspace unless confirm is true.
// The exit code is intentionally decided by the caller (it depends
// on both the fix outcome and the post-fix doctor report).
func runDoctorFix(root, assetRoot string, manifest *Manifest, confirm bool, _ io.Writer) *DoctorFix {
	fix := &DoctorFix{
		ManifestFound: manifest != nil,
		AssetRoot:     assetRoot,
		DryRun:        !confirm,
		Confirmed:     confirm,
	}

	if manifest == nil {
		fix.Notes = []string{
			"no manifest at .x-harness/manifest.yaml; run `xh init` first to enable deterministic repair",
		}
		return fix
	}
	fix.Profile = manifest.Profile

	items := buildDoctorFixPlan(root, assetRoot, manifest)

	if !confirm {
		// Dry-run: enumerate plan only, never touch the filesystem.
		hasRestore := false
		for _, it := range items {
			switch it.action {
			case "restore":
				hasRestore = true
				fix.Applied = append(fix.Applied, "would restore: "+it.rel)
			case "skip":
				if isActionableSkipReason(it.reason) {
					fix.Skipped = append(fix.Skipped, it.rel+": "+it.reason)
				}
			}
		}
		if !hasRestore && len(fix.Skipped) == 0 {
			fix.Notes = append(fix.Notes, "no managed files require repair")
		}
		return fix
	}

	// Apply: only "restore" actions are attempted. Skips are reported
	// only when they carry an actionable reason (e.g., missing
	// source, unsafe path). The common "file present" skip on a
	// healthy workspace is intentionally suppressed so the simple
	// "no fixes needed" message is not drowned in noise.
	hadRestore := false
	for _, it := range items {
		if it.action != "restore" {
			continue
		}
		hadRestore = true
		// Defensive backup in case a race introduced the file between
		// the plan and the apply. In normal flow, the destination is
		// already known to be missing.
		if _, err := os.Stat(it.dest); err == nil {
			backupPath := fmt.Sprintf("%s.bak.%d", it.dest, time.Now().UnixMilli())
			if err := os.Rename(it.dest, backupPath); err != nil {
				fix.Skipped = append(fix.Skipped, it.rel+": backup failed: "+err.Error())
				continue
			}
			fix.Notes = append(fix.Notes, "backup: "+it.rel+" -> "+backupPath)
		}
		if err := os.MkdirAll(filepath.Dir(it.dest), 0o755); err != nil {
			fix.Skipped = append(fix.Skipped, it.rel+": mkdir failed: "+err.Error())
			continue
		}
		if err := copyFile(it.src, it.dest); err != nil {
			fix.Skipped = append(fix.Skipped, it.rel+": copy failed: "+err.Error())
			continue
		}
		fix.Applied = append(fix.Applied, "fixed: "+it.rel)
	}
	if !hadRestore {
		fix.Notes = append(fix.Notes, "no managed files require repair")
		return fix
	}
	// We had at least one restorable item; surface actionable skips
	// (e.g., source missing, unsafe path) so the user sees why some
	// drift was left untouched.
	for _, it := range items {
		if it.action == "skip" && isActionableSkipReason(it.reason) {
			fix.Skipped = append(fix.Skipped, it.rel+": "+it.reason)
		}
	}
	return fix
}

// isActionableSkipReason reports whether a "skip" reason is worth
// surfacing to the user. The common "file present" reason on a
// healthy workspace is intentionally treated as non-actionable so
// the fix output stays concise.
func isActionableSkipReason(reason string) bool {
	if reason == "" {
		return false
	}
	if len(reason) >= 13 && reason[:13] == "file present;" {
		return false
	}
	return true
}

// runDoctorFixWithAssetRoot resolves the asset root, reads the
// manifest (if any), and delegates to runDoctorFix. Asset-root
// resolution failures are non-fatal: the function reports them as
// notes so the caller can surface a clear message to the user.
// The exit code is intentionally decided by the caller (it depends
// on both the fix outcome and the post-fix doctor report).
func runDoctorFixWithAssetRoot(root string, confirm bool, stderr io.Writer) *DoctorFix {
	assetRoot, err := resolveInitAssetRoot("")
	if err != nil {
		// Without an asset root we cannot restore anything. Surface a
		// friendly note instead of leaking the resolver error verbatim.
		fix := &DoctorFix{
			ManifestFound: false,
			DryRun:        !confirm,
			Confirmed:     confirm,
			Notes: []string{
				"could not resolve asset root: " + err.Error(),
				"set --asset-root or X_HARNESS_ASSET_ROOT to enable deterministic repair",
			},
		}
		// Try the manifest anyway so the user sees what was tracked.
		if m, mErr := readManifest(root); mErr == nil {
			fix.ManifestFound = true
			fix.Profile = m.Profile
			fix.Notes = append(fix.Notes,
				"manifest is present but no asset root to restore from; run from a clone of x-harness or set --asset-root")
		}
		return fix
	}
	manifest, mErr := readManifest(root)
	if mErr != nil {
		// Manifest absent or unreadable: runDoctorFix already handles
		// the "no manifest" branch.
		return runDoctorFix(root, assetRoot, nil, confirm, stderr)
	}
	return runDoctorFix(root, assetRoot, manifest, confirm, stderr)
}

// renderDoctorFixText prints the fix plan/result in plain text. It is
// a stable, grep-friendly summary intended for terminal output.
func renderDoctorFixText(w io.Writer, fix *DoctorFix) {
	if fix == nil {
		return
	}
	WriteLine(w, "# xh doctor --fix")
	WriteLine(w, "manifest_found: %v", fix.ManifestFound)
	if fix.Profile != "" {
		WriteLine(w, "profile: %s", fix.Profile)
	}
	if fix.AssetRoot != "" {
		WriteLine(w, "asset_root: %s", fix.AssetRoot)
	}
	WriteLine(w, "dry_run: %v", fix.DryRun)
	WriteLine(w, "confirmed: %v", fix.Confirmed)
	if len(fix.Applied) > 0 {
		WriteLine(w, "applied:")
		for _, a := range fix.Applied {
			WriteLine(w, "  - %s", a)
		}
	}
	if len(fix.Skipped) > 0 {
		WriteLine(w, "skipped:")
		for _, s := range fix.Skipped {
			WriteLine(w, "  - %s", s)
		}
	}
	if len(fix.Notes) > 0 {
		WriteLine(w, "notes:")
		for _, n := range fix.Notes {
			WriteLine(w, "  - %s", n)
		}
	}
}
