package cli

import (
	"fmt"
	"io"
)

// LearnSection represents a single section of the concept tour.
type LearnSection struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// LearnResult is the JSON output shape for the learn command.
type LearnResult struct {
	Sections  []LearnSection `json:"sections"`
	NextSteps []string       `json:"next_steps"`
}

func handleLearn(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode := false
	for _, a := range args {
		switch a {
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh learn [--json]")
			return ExitUsage
		case "--json":
			jsonMode = true
		}
	}

	sections := []LearnSection{
		{
			Title: "Overview",
			Body:  "x-harness is a lightweight verify-gated harness for AI-agent workflows. It enforces that completion is admitted, not claimed, via a read-only verifier.",
		},
		{
			Title: "Core concepts",
			Body: `Completion is admitted, not claimed — only the verify gate can accept work.
Verifier is read-only — it inspects evidence but never edits source files.
Success is the only accepted outcome — all non-success results are withheld.
Canonical tiers are light, standard, and deep — each with increasing evidence requirements.
PGV (pre-gate validation) is advisory-only — it never overrides the verify gate.`,
		},
		{
			Title: "Tiers and evidence",
			Body: `light: files_changed plus command evidence or manual rationale.
standard: adds done_checklist and prediction.
deep: adds evidence scope declaration, untested regions, remaining risks, execution controls, rollback policy, read/write sets, and verification artifacts.`,
		},
	}

	nextSteps := []string{
		"Run xh start for guided onboarding",
		"Run xh check --card <card> to verify a completion card",
		"Run xh actions to see beginner-friendly commands",
		"Read docs/GETTING_STARTED.md",
	}

	if jsonMode {
		result := LearnResult{
			Sections:  sections,
			NextSteps: nextSteps,
		}
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		WriteLine(stdout, "# xh learn - Concept tour")
		WriteLine(stdout, "")
		for _, sec := range sections {
			WriteLine(stdout, "## %s", sec.Title)
			WriteLine(stdout, "")
			WriteLine(stdout, "%s", sec.Body)
			WriteLine(stdout, "")
		}
		WriteLine(stdout, "Next steps:")
		for _, s := range nextSteps {
			WriteLine(stdout, "  - %s", s)
		}
	}

	return ExitOK
}
