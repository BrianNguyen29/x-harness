package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/prediction"
	"gopkg.in/yaml.v3"
)

func handlePrediction(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "prediction requires a subcommand: check, verify, report")
		return ExitUsage
	}

	switch args[0] {
	case "check":
		return handlePredictionCheck(args[1:], stdout, stderr)
	case "verify":
		return handlePredictionVerify(args[1:], stdout, stderr)
	case "report":
		return handlePredictionReport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown prediction subcommand: %s\n", args[0])
		return ExitUsage
	}
}

var defaultCardPaths = []string{
	"completion-card.yaml",
	"completion-card.yml",
	".x-harness/completion-card.yaml",
}

func resolveCardPath(cwd, explicit string) string {
	if explicit != "" {
		p := explicit
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
		return ""
	}
	for _, rel := range defaultCardPaths {
		p := filepath.Join(cwd, rel)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func loadCard(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var card map[string]interface{}
	if err := yaml.Unmarshal(data, &card); err != nil {
		// Try JSON fallback
		if err := json.Unmarshal(data, &card); err != nil {
			return nil, err
		}
	}
	return card, nil
}

func handlePredictionCheck(args []string, stdout, stderr io.Writer) int {
	cardPath := ""
	jsonMode := false
	verbose := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --card requires a value")
				return ExitUsage
			}
			cardPath = args[i+1]
			i++
		case "--json":
			jsonMode = true
		case "--verbose":
			verbose = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	cwd, _ := os.Getwd()
	resolved := resolveCardPath(cwd, cardPath)
	if resolved == "" {
		fmt.Fprintf(stderr, "Error: No completion card found. Searched: %s\n", strings.Join(defaultCardPaths, ", "))
		fmt.Fprintln(stderr, "Provide --card <path> to specify a card.")
		return ExitError
	}

	card, err := loadCard(resolved)
	if err != nil {
		fmt.Fprintf(stderr, "Error loading card: %v\n", err)
		return ExitError
	}

	tier := ""
	if v, ok := card["tier"].(string); ok {
		tier = v
	}

	rawPred, ok := card["prediction"].(map[string]interface{})
	if !ok {
		if jsonMode {
			out := map[string]interface{}{
				"ok":    false,
				"error": "No prediction found in completion card",
				"tier":  tier,
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintln(stderr, "Error: No prediction found in completion card.")
			if tier == "standard" || tier == "deep" {
				fmt.Fprintf(stderr, "Tier %q requires a prediction.\n", tier)
			}
		}
		return ExitError
	}

	pred := &prediction.Prediction{
		Claim:               stringFromInterface(rawPred["claim"]),
		ExpectedEffect:      stringFromInterface(rawPred["expected_effect"]),
		FalsificationMethod: stringFromInterface(rawPred["falsification_method"]),
		Horizon:             stringFromInterface(rawPred["horizon"]),
		MeasurableSignal:    stringFromInterface(rawPred["measurable_signal"]),
		Confidence:          stringFromInterface(rawPred["confidence"]),
	}

	result := prediction.ValidatePrediction(pred)
	result.Tier = tier

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else if verbose {
		if result.OK {
			fmt.Fprintln(stdout, "✓ Prediction is valid")
		} else {
			fmt.Fprintln(stdout, "✗ Prediction has errors:")
		}
		for _, e := range result.Errors {
			fmt.Fprintf(stdout, "  - %s\n", e)
		}
		if len(result.Warnings) > 0 {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "Warnings:")
			for _, w := range result.Warnings {
				fmt.Fprintf(stdout, "  - %s\n", w)
			}
		}
		if result.OK && len(result.Warnings) == 0 {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "Prediction structure:")
			fmt.Fprintf(stdout, "  claim: %s\n", pred.Claim)
			fmt.Fprintf(stdout, "  expected_effect: %s\n", pred.ExpectedEffect)
			fmt.Fprintf(stdout, "  falsification_method: %s\n", pred.FalsificationMethod)
			fmt.Fprintf(stdout, "  horizon: %s\n", pred.Horizon)
			if pred.MeasurableSignal != "" {
				fmt.Fprintf(stdout, "  measurable_signal: %s\n", pred.MeasurableSignal)
			}
			if pred.Confidence != "" {
				fmt.Fprintf(stdout, "  confidence: %s\n", pred.Confidence)
			}
		}
	} else {
		if result.OK {
			fmt.Fprintln(stdout, "Prediction is valid.")
		} else {
			fmt.Fprintln(stderr, "Prediction validation failed:")
			for _, e := range result.Errors {
				fmt.Fprintf(stderr, "  - %s\n", e)
			}
		}
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func stringFromInterface(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func handlePredictionVerify(args []string, stdout, stderr io.Writer) int {
	episodePath := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--episode":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --episode requires a value")
				return ExitUsage
			}
			episodePath = args[i+1]
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

	if episodePath == "" {
		fmt.Fprintln(stderr, "Error: prediction verify requires --episode")
		return ExitUsage
	}

	result, err := prediction.VerifyPredictionFromEpisode(episodePath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "# x-harness Prediction Verify")
		fmt.Fprintf(stdout, "- status: %s\n", result.Status)
		fmt.Fprintf(stdout, "- reason: %s\n", result.Reason)
		episodeID := result.EpisodeID
		if episodeID == "" {
			episodeID = "unknown"
		}
		taskID := result.TaskID
		if taskID == "" {
			taskID = "unknown"
		}
		horizon := result.Horizon
		if horizon == "" {
			horizon = "unknown"
		}
		fmt.Fprintf(stdout, "- episode_id: %s\n", episodeID)
		fmt.Fprintf(stdout, "- task_id: %s\n", taskID)
		fmt.Fprintf(stdout, "- horizon: %s\n", horizon)
		admissionOutcome := "unknown"
		acceptanceStatus := "unknown"
		if result.Verdict != nil {
			if result.Verdict.AdmissionOutcome != "" {
				admissionOutcome = result.Verdict.AdmissionOutcome
			}
			if result.Verdict.AcceptanceStatus != "" {
				acceptanceStatus = result.Verdict.AcceptanceStatus
			}
		}
		fmt.Fprintf(stdout, "- verdict: %s / %s\n", admissionOutcome, acceptanceStatus)
	}

	if result.Status == "falsified" {
		return ExitError
	}
	return ExitOK
}

func handlePredictionReport(args []string, stdout, stderr io.Writer) int {
	since := ""
	episodesDir := ".x-harness/episodes"
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--since":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --since requires a value")
				return ExitUsage
			}
			since = args[i+1]
			i++
		case "--episodes-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --episodes-dir requires a value")
				return ExitUsage
			}
			episodesDir = args[i+1]
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

	report, err := prediction.GenerateReport(episodesDir, since)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "# x-harness Prediction Report")
		fmt.Fprintf(stdout, "- episodes_analyzed: %d\n", report.EpisodesAnalyzed)
		fmt.Fprintf(stdout, "- confirmed: %d\n", report.Confirmed)
		fmt.Fprintf(stdout, "- falsified: %d\n", report.Falsified)
		fmt.Fprintf(stdout, "- inconclusive: %d\n", report.Inconclusive)
		if len(report.Results) > 0 {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "## Episodes")
			for _, r := range report.Results {
				episodeID := r.EpisodeID
				if episodeID == "" {
					episodeID = "unknown"
				}
				fmt.Fprintf(stdout, "- %s: %s (%s)\n", episodeID, r.Status, r.Reason)
			}
		}
	}

	return ExitOK
}
