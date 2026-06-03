package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/repo"
)

func handleInit(args []string, stdout io.Writer, stderr io.Writer) int {
	mode := "minimal"
	legacyModeSet := false
	dryRun := false
	merge := false
	force := false
	adapters := ""
	assetRootFlag := ""
	target := "."
	profile := ""
	preview := false
	apply := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--minimal":
			mode = "minimal"
			legacyModeSet = true
		case "--standard":
			mode = "standard"
			legacyModeSet = true
		case "--full":
			mode = "full"
			legacyModeSet = true
		case "--dry-run":
			dryRun = true
		case "--preview":
			preview = true
		case "--apply":
			apply = true
		case "--merge":
			merge = true
		case "--force":
			force = true
		case "--adapters":
			if i+1 >= len(args) {
				WriteLine(stderr, "usage: init [target] [options]")
				WriteLine(stderr, "missing value for --adapters")
				return ExitUsage
			}
			adapters = args[i+1]
			i++
		case "--asset-root":
			if i+1 >= len(args) {
				WriteLine(stderr, "usage: init [target] [options]")
				WriteLine(stderr, "missing value for --asset-root")
				return ExitUsage
			}
			assetRootFlag = args[i+1]
			i++
		case "--profile":
			if i+1 >= len(args) {
				WriteLine(stderr, "usage: init [target] [options]")
				WriteLine(stderr, "missing value for --profile")
				return ExitUsage
			}
			profile = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				WriteLine(stderr, "unknown flag: %s", args[i])
				return ExitUsage
			}
			if target == "." {
				target = args[i]
			} else {
				WriteLine(stderr, "usage: init [target] [options]")
				return ExitUsage
			}
		}
	}

	if profile != "" {
		if legacyModeSet {
			WriteLine(stderr, "usage: init [target] [options]")
			WriteLine(stderr, "cannot use --profile with --minimal, --standard, or --full")
			return ExitUsage
		}
		switch profile {
		case "minimal", "standard":
			mode = profile
		case "deep":
			mode = "full"
		default:
			WriteLine(stderr, "usage: init [target] [options]")
			WriteLine(stderr, "invalid profile: %s", profile)
			return ExitUsage
		}
	}

	displayMode := mode
	if profile == "deep" {
		displayMode = "deep"
	}
	if preview {
		dryRun = true
	}
	_ = apply

	targetDir, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(stderr, "error resolving target: %v\n", err)
		return ExitError
	}

	assetRoot, err := resolveInitAssetRoot(assetRootFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	plan := buildInitPlan(mode, assetRoot, targetDir, adapters, stderr)

	if dryRun {
		WriteLine(stdout, "# x-harness init (%s) - dry run", displayMode)
		for _, p := range plan {
			WriteLine(stdout, "would copy: %s -> %s", p.src, p.dest)
		}
		return ExitOK
	}

	// Idempotency: same profile with unchanged managed files is a no-op (unless force/merge)
	if !force && !merge {
		if existingManifest, err := readManifest(targetDir); err == nil {
			if existingManifest.Profile == displayMode {
				upToDate := true
				for _, p := range plan {
					if !pathExistsBool(p.dest) {
						upToDate = false
						break
					}
				}
				if upToDate {
					for _, entry := range existingManifest.Entries {
						entryPath, err := resolveManifestEntryPath(targetDir, entry)
						if err != nil {
							upToDate = false
							break
						}
						hash, err := fileHash(entryPath)
						if err != nil || hash != entry.Hash {
							upToDate = false
							break
						}
					}
				}
				if upToDate {
					WriteLine(stdout, "x-harness init (%s) already up-to-date: no changes needed", displayMode)
					return ExitOK
				}
			}
		}
	}

	var conflicts []string
	for _, p := range plan {
		if pathExistsBool(p.dest) && !force && !merge {
			conflicts = append(conflicts, p.dest)
		}
	}

	if len(conflicts) > 0 {
		fmt.Fprintf(stderr, "\n# x-harness init (%s) blocked: %d file(s) already exist\n", displayMode, len(conflicts))
		for _, c := range conflicts {
			fmt.Fprintf(stderr, "conflict: %s\n", c)
		}
		fmt.Fprintln(stderr, "\nUse --force to overwrite or --merge to merge with existing files.")
		return ExitError
	}

	var copied []string
	for _, p := range plan {
		if pathExistsBool(p.dest) && force {
			if err := os.RemoveAll(p.dest); err != nil {
				fmt.Fprintf(stderr, "error removing %s: %v\n", p.dest, err)
				return ExitError
			}
		} else if pathExistsBool(p.dest) && merge {
			continue
		}

		if err := copyRecursive(p.src, p.dest); err != nil {
			fmt.Fprintf(stderr, "error copying %s -> %s: %v\n", p.src, p.dest, err)
			return ExitError
		}
		copied = append(copied, p.dest)
		WriteLine(stdout, "copied: %s", p.dest)
	}

	// Write manifest for repair/uninstall lifecycle
	manifestEntries := computeManifestEntries(plan, targetDir)
	if err := writeManifest(targetDir, displayMode, manifestEntries); err != nil {
		fmt.Fprintf(stderr, "warning: failed to write manifest: %v\n", err)
	} else {
		WriteLine(stdout, "manifest: %s", filepath.Join(targetDir, manifestPath))
	}

	WriteLine(stdout, "x-harness init (%s) complete: %d assets copied to %s", displayMode, len(copied), targetDir)
	return ExitOK
}

type initPlanItem struct {
	src  string
	dest string
}

func buildInitPlan(mode, assetRoot, targetDir, adapters string, stderr io.Writer) []initPlanItem {
	var plan []initPlanItem

	if mode == "minimal" {
		minimalFiles := []struct{ src, dest string }{
			{"AGENTS.md", "AGENTS.md"},
			{"X_HARNESS.md", "X_HARNESS.md"},
			{"docs/VERIFY_GATE.md", "docs/VERIFY_GATE.md"},
			{"docs/RUNTIME_CONTRACT.md", "docs/RUNTIME_CONTRACT.md"},
			{"templates/SUBAGENT_TASK_light.md", "templates/SUBAGENT_TASK_light.md"},
			{"templates/SUBAGENT_TASK_standard.md", "templates/SUBAGENT_TASK_standard.md"},
			{"templates/SUBAGENT_TASK_deep.md", "templates/SUBAGENT_TASK_deep.md"},
			{"templates/COMPLETION_CARD.md", "templates/COMPLETION_CARD.md"},
			{"policies/admission.yaml", "policies/admission.yaml"},
		}
		for _, f := range minimalFiles {
			src := filepath.Join(assetRoot, f.src)
			if pathExistsBool(src) {
				plan = append(plan, initPlanItem{src: src, dest: filepath.Join(targetDir, f.dest)})
			} else {
				fmt.Fprintf(stderr, "warning: source not found, skipping: %s\n", src)
			}
		}

		// Include the schemas/ directory so verify/check can compile
		// completion-card.schema.json and other core schemas required by
		// the user journey (init -> add completion-card -> check).
		schemasSrc := filepath.Join(assetRoot, "schemas")
		if pathExistsBool(schemasSrc) {
			plan = append(plan, initPlanItem{
				src:  schemasSrc,
				dest: filepath.Join(targetDir, "schemas"),
			})
		} else {
			fmt.Fprintf(stderr, "warning: source not found, skipping: %s\n", schemasSrc)
		}
	} else {
		modeAssets := map[string][]string{
			"standard": {
				"examples/01-solo-agent",
				"examples/02-assisted-agent",
				"schemas",
				"policies",
			},
			"full": {
				"examples",
				"schemas",
				"policies",
				"templates",
				"adapters",
			},
		}
		assets := modeAssets[mode]
		for _, asset := range assets {
			src := filepath.Join(assetRoot, asset)
			if !pathExistsBool(src) {
				fmt.Fprintf(stderr, "warning: source not found, skipping: %s\n", src)
				continue
			}
			dest := filepath.Join(targetDir, filepath.Base(asset))
			plan = append(plan, initPlanItem{src: src, dest: dest})
		}
	}

	if adapters != "" {
		for _, adapter := range strings.Split(adapters, ",") {
			adapter = strings.TrimSpace(adapter)
			if adapter == "" {
				continue
			}
			src := filepath.Join(assetRoot, "adapters", adapter)
			if pathExistsBool(src) {
				plan = append(plan, initPlanItem{
					src:  src,
					dest: filepath.Join(targetDir, "adapters", adapter),
				})
			} else {
				fmt.Fprintf(stderr, "warning: source not found, skipping: %s\n", src)
			}
		}
	}

	return plan
}

func resolveInitAssetRoot(flagRoot string) (string, error) {
	markers := []string{
		"templates/COMPLETION_CARD.md",
		"policies/admission.yaml",
		"schemas/completion-card.schema.json",
	}

	if flagRoot != "" {
		absPath, err := filepath.Abs(flagRoot)
		if err != nil {
			return "", fmt.Errorf("invalid --asset-root: %v", err)
		}
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(absPath, marker)); err != nil {
				return "", fmt.Errorf("invalid --asset-root: missing marker %s", marker)
			}
		}
		return absPath, nil
	}

	if envRoot := os.Getenv("X_HARNESS_ASSET_ROOT"); envRoot != "" {
		absPath, err := filepath.Abs(envRoot)
		if err == nil {
			valid := true
			for _, marker := range markers {
				if _, err := os.Stat(filepath.Join(absPath, marker)); err != nil {
					valid = false
					break
				}
			}
			if valid {
				return absPath, nil
			}
		}
	}

	if repoRoot, err := repo.FindRoot(""); err == nil {
		valid := true
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(repoRoot, marker)); err != nil {
				valid = false
				break
			}
		}
		if valid {
			return repoRoot, nil
		}
	}

	if wd, err := os.Getwd(); err == nil {
		valid := true
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(wd, marker)); err != nil {
				valid = false
				break
			}
		}
		if valid {
			return wd, nil
		}
	}

	return "", fmt.Errorf("x-harness assets not found; set --asset-root or X_HARNESS_ASSET_ROOT")
}

func pathExistsBool(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyRecursive(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if err := os.MkdirAll(dest, info.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			srcPath := filepath.Join(src, entry.Name())
			destPath := filepath.Join(dest, entry.Name())
			if err := copyRecursive(srcPath, destPath); err != nil {
				return err
			}
		}
		return nil
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	return os.WriteFile(dest, data, info.Mode())
}
