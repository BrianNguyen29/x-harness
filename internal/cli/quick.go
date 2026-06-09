package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BrianNguyen29/x-harness/internal/repo"
)

// QuickResult is the JSON output shape for the quick command.
type QuickResult struct {
	Root            string   `json:"root"`
	Recommendation  string   `json:"recommendation"`
	Reason          string   `json:"reason"`
	NextSteps       []string `json:"next_steps"`
	DetectedSignals []string `json:"detected_signals"`
}

func handleQuick(args []string, stdout io.Writer, stderr io.Writer, lang Lang) int {
	root := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh quick [--root <path>] [--json]")
			return ExitUsage
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		}
	}

	// Resolve root
	resolvedRoot := root
	if resolvedRoot == "" {
		found, err := repo.FindRoot("")
		if err == nil {
			resolvedRoot = found
		} else {
			wd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				return ExitError
			}
			resolvedRoot = wd
		}
	}
	resolvedRoot, _ = filepath.Abs(resolvedRoot)

	signals := detectSignals(resolvedRoot)
	if signals == nil {
		signals = []string{}
	}
	recommendation, reason, nextSteps := buildRecommendation(resolvedRoot, signals)

	result := QuickResult{
		Root:            resolvedRoot,
		Recommendation:  recommendation,
		Reason:          reason,
		NextSteps:       nextSteps,
		DetectedSignals: signals,
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		WriteLine(stdout, "# %s", quickTitle(lang))
		WriteLine(stdout, "")
		WriteLine(stdout, "%s: %s", quickRootLabel(lang), result.Root)
		WriteLine(stdout, "%s: %s", quickRecommendationLabel(lang), result.Recommendation)
		WriteLine(stdout, "%s: %s", quickReasonLabel(lang), result.Reason)
		WriteLine(stdout, "")
		WriteLine(stdout, "%s", quickDetectedSignalsLabel(lang))
		if len(result.DetectedSignals) == 0 {
			WriteLine(stdout, "%s", quickNoneLabel(lang))
		} else {
			for _, s := range result.DetectedSignals {
				WriteLine(stdout, "  - %s", s)
			}
		}
		WriteLine(stdout, "")
		WriteLine(stdout, "%s", quickNextStepsLabel(lang))
		for _, s := range result.NextSteps {
			WriteLine(stdout, "  - %s", s)
		}
	}

	return ExitOK
}

// harnessMarkers are files/directories that indicate a harness is present.
var harnessMarkers = []string{
	"AGENTS.md",
	"X_HARNESS.md",
	".x-harness",
}

// cardNames are common completion card filenames to look for.
var cardNames = []string{
	"completion-card.yaml",
	"completion-card.yml",
	"completion-card.json",
}

func detectSignals(root string) []string {
	var signals []string
	for _, marker := range harnessMarkers {
		path := filepath.Join(root, marker)
		if _, err := os.Stat(path); err == nil {
			signals = append(signals, "harness_marker:"+marker)
		}
	}
	// Look for completion cards recursively under root, limited depth for speed.
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			// Skip deep or irrelevant directories to keep it fast
			if name == "node_modules" || name == ".git" || name == "vendor" || name == "dist" || name == "coverage" {
				return filepath.SkipDir
			}
			// Skip generated harness state directories
			if name == "tmp" || name == "cache" {
				parent := filepath.Dir(path)
				if filepath.Base(parent) == ".x-harness" {
					return filepath.SkipDir
				}
			}
			// Limit depth by skipping very deep walks: if we're more than 4 levels
			// below root, skip further descent. This is a simple heuristic.
			rel, _ := filepath.Rel(root, path)
			if rel != "" && rel != "." {
				depth := 0
				for i := 0; i < len(rel); i++ {
					if rel[i] == filepath.Separator {
						depth++
					}
				}
				if depth >= 4 {
					return filepath.SkipDir
				}
			}
			return nil
		}
		for _, name := range cardNames {
			if d.Name() == name {
				rel, _ := filepath.Rel(root, path)
				signals = append(signals, "completion_card:"+rel)
				return nil
			}
		}
		return nil
	})
	return signals
}

func buildRecommendation(root string, signals []string) (recommendation, reason string, nextSteps []string) {
	hasHarness := false
	var cardPaths []string
	for _, s := range signals {
		if len(s) > 15 && s[:15] == "harness_marker:" {
			hasHarness = true
		}
		if len(s) > 16 && s[:16] == "completion_card:" {
			cardPaths = append(cardPaths, s[16:])
		}
	}

	if !hasHarness {
		recommendation = "xh start"
		reason = "No harness markers found under root. Begin with guided onboarding."
		nextSteps = append(nextSteps, "xh start")
		nextSteps = append(nextSteps, "xh init")
	} else if len(cardPaths) > 0 {
		recommendation = "xh check --card " + cardPaths[0]
		reason = "A completion card was found. Verify it as the next step."
		nextSteps = append(nextSteps, "xh check --card "+cardPaths[0])
	} else {
		recommendation = "xh doctor --root " + root + " --json"
		reason = "Harness is present but no completion card found yet. Check workspace health first."
		nextSteps = append(nextSteps, "xh doctor --root "+root+" --json")
	}

	// Always include these safe, read-only next steps
	nextSteps = append(nextSteps, "xh run builtin:ci --dry-run")
	nextSteps = append(nextSteps, "xh learn")

	return recommendation, reason, nextSteps
}
