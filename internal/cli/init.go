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

	// Wizard-only flags (P2-S1). The wizard is a thin, deterministic
	// wrapper around the existing init plan/copy logic: it prints a
	// 3-step plan (profile, actions, apply decision), reuses
	// buildInitPlan / copyRecursive, and optionally scaffolds a
	// first completion card via the existing `xh add completion-card`
	// helper. It never blocks on stdin so it is safe to run in CI.
	wizard := false
	wizardProfile := ""
	wizardDryRun := false
	wizardWithCard := ""

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
		case "--wizard":
			wizard = true
		case "--wizard-profile":
			if i+1 >= len(args) {
				WriteLine(stderr, "usage: init [target] [options]")
				WriteLine(stderr, "missing value for --wizard-profile")
				return ExitUsage
			}
			wizardProfile = args[i+1]
			i++
		case "--wizard-dry-run":
			wizardDryRun = true
		case "--wizard-with-card":
			if i+1 >= len(args) {
				WriteLine(stderr, "usage: init [target] [options]")
				WriteLine(stderr, "missing value for --wizard-with-card")
				return ExitUsage
			}
			wizardWithCard = args[i+1]
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

	// Wizard mode setup. The wizard is intentionally restricted to
	// the `--wizard-*` flag family so that it can never accidentally
	// toggle legacy `--minimal/--standard/--full` or the explicit
	// `--profile` path. The existing `if profile != ""` block below
	// validates the resolved profile name and converts it to mode.
	if wizard {
		if legacyModeSet {
			WriteLine(stderr, "usage: init [target] [options]")
			WriteLine(stderr, "cannot use --wizard with --minimal, --standard, or --full")
			return ExitUsage
		}
		if profile != "" {
			WriteLine(stderr, "usage: init [target] [options]")
			WriteLine(stderr, "cannot use --wizard with --profile; use --wizard-profile")
			return ExitUsage
		}
		if wizardProfile != "" {
			profile = wizardProfile
		} else {
			profile = "minimal"
		}
		if wizardDryRun {
			dryRun = true
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
		if wizard {
			printWizardPreview(stdout, displayMode, targetDir, len(plan), wizardWithCard)
		}
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

	// Wizard apply path: print a clear summary, then optionally
	// scaffold a first completion card via the existing safe helper.
	// Card scaffold is intentionally NOT performed on dry-run so the
	// preview remains a pure read-only inspection.
	if wizard {
		WriteLine(stdout, "# xh init --wizard complete")
		WriteLine(stdout, "  profile: %s", displayMode)
		WriteLine(stdout, "  target:  %s", targetDir)
		if wizardWithCard != "" {
			cardPath := filepath.Join(targetDir, "completion-card.yaml")
			WriteLine(stdout, "  scaffold: completion-card -> %s", cardPath)
			addCode := handleAdd([]string{
				"completion-card",
				"task_id=" + wizardWithCard + ",tier=light",
				"--out", cardPath,
			}, stdout, stderr)
			if addCode != ExitOK {
				return addCode
			}
		}
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
				"AGENTS.md",
				"X_HARNESS.md",
				".github/workflows/x-harness-verify.yml",
				".x-harness/managed-blocks.yaml",
				"docs",
				"components",
				"tools",
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
			if strings.HasPrefix(asset, ".") {
				dest = filepath.Join(targetDir, filepath.FromSlash(asset))
			}
			plan = append(plan, initPlanItem{src: src, dest: dest})
		}

		// Include adapter guidance in standard mode
		if mode == "standard" {
			adaptersDocSrc := filepath.Join(assetRoot, "docs/ADAPTERS.md")
			if pathExistsBool(adaptersDocSrc) {
				plan = append(plan, initPlanItem{
					src:  adaptersDocSrc,
					dest: filepath.Join(targetDir, "docs/ADAPTERS.md"),
				})
			} else {
				fmt.Fprintf(stderr, "warning: source not found, skipping: %s\n", adaptersDocSrc)
			}
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

		// Include adapter guidance when adapters are requested
		adaptersDocSrc := filepath.Join(assetRoot, "docs/ADAPTERS.md")
		adaptersDocDest := filepath.Join(targetDir, "docs/ADAPTERS.md")
		if pathExistsBool(adaptersDocSrc) {
			alreadyPlanned := false
			for _, p := range plan {
				if p.dest == adaptersDocDest {
					alreadyPlanned = true
					break
				}
			}
			if !alreadyPlanned {
				plan = append(plan, initPlanItem{
					src:  adaptersDocSrc,
					dest: adaptersDocDest,
				})
			}
		} else {
			fmt.Fprintf(stderr, "warning: source not found, skipping: %s\n", adaptersDocSrc)
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

// printWizardPreview emits the wizard 3-step plan to stdout. It is
// only called from the dry-run path; the apply path uses inline
// WriteLine calls so the wizard summary can be interleaved with
// the existing init copy/manifest output.
//
// The format is intentionally plain and grep-friendly so it is
// testable without depending on terminal width, ANSI codes, or TTY
// detection.
func printWizardPreview(stdout io.Writer, displayMode, targetDir string, planCount int, taskID string) {
	WriteLine(stdout, "# xh init --wizard")
	WriteLine(stdout, "step 1/3: profile")
	WriteLine(stdout, "  -> %s", displayMode)
	WriteLine(stdout, "step 2/3: planned actions")
	WriteLine(stdout, "  -> %d file(s) would be copied into %s", planCount, targetDir)
	WriteLine(stdout, "step 3/3: apply decision")
	if taskID != "" {
		WriteLine(stdout, "  -> preview only (would scaffold completion-card task_id=%s on apply)", taskID)
	} else {
		WriteLine(stdout, "  -> preview only")
	}
}
