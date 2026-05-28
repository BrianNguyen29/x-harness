package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type profileRecommendation struct {
	RecommendedProfile string   `json:"recommended_profile"`
	Reason             string   `json:"reason"`
	RequiredCommands   []string `json:"required_commands"`
	RecommendedChecks  []string `json:"recommended_checks"`
	NotNeeded          []string `json:"not_needed"`
}

func recommendProfile(goal string) profileRecommendation {
	goalLower := strings.ToLower(goal)

	deepKeywords := []string{"release", "security", "deep", "governance", "approval"}
	standardKeywords := []string{"pr", "ci", "team", "verification"}
	minimalKeywords := []string{"local", "basic", "quick", "single-agent"}

	for _, kw := range deepKeywords {
		if strings.Contains(goalLower, kw) {
			return profileRecommendation{
				RecommendedProfile: "deep",
				Reason: fmt.Sprintf(
					"Goal %q involves release, security, governance, or approval concerns; deep profile provides full evidence floor, rollback policy, and release readiness.",
					goal,
				),
				RequiredCommands: []string{
					"x-harness verify --strict",
					"x-harness report --format json",
					"x-harness conformance run --profile minimal",
				},
				RecommendedChecks: []string{
					"mutation_guard",
					"evidence_provenance",
					"denominator_contract",
					"approval_receipt",
					"packet_chain",
				},
				NotNeeded: []string{},
			}
		}
	}

	for _, kw := range standardKeywords {
		if strings.Contains(goalLower, kw) {
			return profileRecommendation{
				RecommendedProfile: "standard",
				Reason: fmt.Sprintf(
					"Goal %q involves PR/CI/team verification; standard profile provides mutation guard, trace, and report config.",
					goal,
				),
				RequiredCommands: []string{
					"x-harness verify",
					"x-harness report --format json",
				},
				RecommendedChecks: []string{
					"mutation_guard",
					"evidence_provenance",
				},
				NotNeeded: []string{
					"packet_chain",
					"release_evidence_bundle",
					"approval_receipt",
				},
			}
		}
	}

	for _, kw := range minimalKeywords {
		if strings.Contains(goalLower, kw) {
			return profileRecommendation{
				RecommendedProfile: "minimal",
				Reason: fmt.Sprintf(
					"Goal %q is local/basic/quick; minimal profile provides core verify contract and templates.",
					goal,
				),
				RequiredCommands: []string{
					"x-harness verify",
				},
				RecommendedChecks: []string{
					"standard_verify_gate",
				},
				NotNeeded: []string{
					"mutation_guard",
					"packet_chain",
					"release_evidence_bundle",
					"approval_receipt",
				},
			}
		}
	}

	// Default to standard for unknown goals
	return profileRecommendation{
		RecommendedProfile: "standard",
		Reason: fmt.Sprintf(
			"Goal %q does not match a specific pattern; defaulting to standard profile for general verification.",
			goal,
		),
		RequiredCommands: []string{
			"x-harness verify",
			"x-harness report --format json",
		},
		RecommendedChecks: []string{
			"mutation_guard",
			"evidence_provenance",
		},
		NotNeeded: []string{
			"packet_chain",
			"release_evidence_bundle",
			"approval_receipt",
		},
	}
}

func handleProfile(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "profile requires a subcommand: recommend")
		return ExitUsage
	}

	switch args[0] {
	case "recommend":
		return handleProfileRecommend(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown profile subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleProfileRecommend(args []string, stdout, stderr io.Writer) int {
	goal := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--goal":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --goal requires a value")
				return ExitUsage
			}
			goal = args[i+1]
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

	if goal == "" {
		fmt.Fprintln(stderr, "usage: x-harness profile recommend --goal <goal> [--json]")
		return ExitUsage
	}

	rec := recommendProfile(goal)

	if jsonMode {
		data, _ := json.MarshalIndent(rec, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Recommended profile: %s\n", rec.RecommendedProfile)
		fmt.Fprintf(stdout, "Reason: %s\n", rec.Reason)
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Required commands:")
		for _, cmd := range rec.RequiredCommands {
			fmt.Fprintf(stdout, "  - %s\n", cmd)
		}
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Recommended checks:")
		for _, check := range rec.RecommendedChecks {
			fmt.Fprintf(stdout, "  - %s\n", check)
		}
		if len(rec.NotNeeded) > 0 {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "Not needed:")
			for _, item := range rec.NotNeeded {
				fmt.Fprintf(stdout, "  - %s\n", item)
			}
		}
	}

	return ExitOK
}
