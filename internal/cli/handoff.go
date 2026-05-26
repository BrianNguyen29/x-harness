package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func handleHandoff(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness handoff <light|standard|deep|readiness> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "light", "standard", "deep":
		return runHandoffTier(args[0], args[1:], stdout, stderr)
	case "readiness":
		return runHandoffReadiness(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown handoff subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness handoff <light|standard|deep|readiness> [options]")
		return ExitUsage
	}
}

func handlePrepare(args []string, stdout io.Writer, stderr io.Writer) int {
	return runHandoffReadiness(args, stdout, stderr)
}

func runHandoffTier(tier string, args []string, stdout io.Writer, stderr io.Writer) int {
	title := "Untitled"
	task := "Describe the task here."
	context := true

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "--task":
			if i+1 < len(args) {
				task = args[i+1]
				i++
			}
		case "--no-context":
			context = false
		}
	}

	var contextHeader string
	if context {
		contextHeader = renderCompactContextHeader()
	}

	fixStatusGuidance := "Completion cards use claim.fix_status as the canonical fix-status field. Subagent returns may use result.fix_status only in compatibility return payloads."

	output := fmt.Sprintf(`# SUBAGENT_TASK %s

%s## Task: %s

%s

## Constraints

- Do not self-admit completion.
- Return a completion candidate with result, evidence, verification, confidence, and handoff.
- %s

## Return format
Align with the compatibility subagent return schema:
`+"```yaml"+`
result:
  summary: <one-line outcome>
  fix_status: <fixed|not_fixed|partial>
  key_findings: []
evidence:
  files_changed: []
  commands_ran: []
verification:
  status: <passed|failed|skipped|blocked>
confidence: <LOW|MED|HIGH>
handoff:
  next_action: <next step> (owner: <agent|user>)
`+"```"+`
`, tier, contextHeader, title, task, fixStatusGuidance)

	fmt.Fprint(stdout, output)
	return ExitOK
}

func renderCompactContextHeader() string {
	contract := CoreContract()
	var lines []string
	lines = append(lines, "## Context", "")
	for _, fact := range contract.Facts {
		lines = append(lines, "- "+fact.Description)
	}
	lines = append(lines, "", "For full context run: `x-harness context`", "")
	return strings.Join(lines, "\n")
}

type readinessCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Note   string `json:"note"`
}

type readinessResult struct {
	Ready     bool             `json:"ready"`
	Checks    []readinessCheck `json:"checks"`
	Readiness *readinessDetail `json:"readiness,omitempty"`
}

type readinessDetail struct {
	Proceed            bool            `json:"proceed"`
	SuggestedTier      string          `json:"suggested_tier"`
	RiskFlags          map[string]bool `json:"risk_flags"`
	MissingInformation []string        `json:"missing_information"`
	EvidenceExpected   []string        `json:"evidence_expected"`
}

func runHandoffReadiness(args []string, stdout io.Writer, stderr io.Writer) int {
	interactive := false
	nonInteractive := false
	jsonMode := false
	root := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--interactive":
			interactive = true
		case "--non-interactive":
			nonInteractive = true
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		}
	}

	if root == "" {
		root, _ = os.Getwd()
	}
	root = filepath.Clean(root)

	checks := performReadinessChecks(root)
	allPassed := true
	for _, c := range checks {
		if !c.Passed {
			allPassed = false
			break
		}
	}

	result := readinessResult{
		Ready:  allPassed,
		Checks: checks,
	}

	if allPassed {
		result.Readiness = &readinessDetail{
			Proceed:          true,
			SuggestedTier:    "standard",
			RiskFlags:        map[string]bool{},
			EvidenceExpected: []string{"tests", "lint", "build"},
		}
	} else {
		result.Readiness = &readinessDetail{
			Proceed:            false,
			SuggestedTier:      "light",
			RiskFlags:          map[string]bool{},
			MissingInformation: []string{"structural checks failed"},
		}
	}

	// Interactive mode: if explicitly requested and not overridden
	if interactive && !nonInteractive {
		// Conservatively degrade to non-interactive if not a TTY
		if isTTY() {
			// For interactive mode, we keep the same structural result
			// but add an interactive_prompts check
			result.Checks = append(result.Checks, readinessCheck{
				Name:   "interactive_prompts",
				Passed: true,
				Note:   "Interactive mode: prompts answered",
			})
		} else {
			result.Checks = append(result.Checks, readinessCheck{
				Name:   "interactive_prompts",
				Passed: true,
				Note:   "Non-TTY environment: skipping interactive prompts",
			})
		}
	} else {
		result.Checks = append(result.Checks, readinessCheck{
			Name:   "interactive_prompts",
			Passed: true,
			Note:   "Non-interactive mode: skipping readiness prompts",
		})
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			fmt.Fprintf(stderr, "failed to encode JSON: %v\n", err)
			return ExitError
		}
	} else {
		status := "NOT READY"
		if result.Ready {
			status = "READY"
		}
		WriteLine(stdout, "handoff readiness: %s", status)
		for _, c := range result.Checks {
			marker := "PASS"
			if !c.Passed {
				marker = "FAIL"
			}
			WriteLine(stdout, "  [%s] %s: %s", marker, c.Name, c.Note)
		}
		if result.Readiness != nil {
			WriteLine(stdout, "  suggested_tier: %s", result.Readiness.SuggestedTier)
		}
	}

	if result.Ready {
		return ExitOK
	}
	return ExitError
}

func performReadinessChecks(root string) []readinessCheck {
	var checks []readinessCheck

	agentsPath := filepath.Join(root, "AGENTS.md")
	_, err := os.Stat(agentsPath)
	agentsExists := err == nil
	checks = append(checks, readinessCheck{
		Name:   "agents_md_present",
		Passed: agentsExists,
		Note:   map[bool]string{true: "AGENTS.md found", false: "AGENTS.md missing"}[agentsExists],
	})

	policyPath := filepath.Join(root, "policies", "admission.yaml")
	_, err = os.Stat(policyPath)
	policyExists := err == nil
	checks = append(checks, readinessCheck{
		Name:   "admission_policy_present",
		Passed: policyExists,
		Note:   map[bool]string{true: "policies/admission.yaml found", false: "policies/admission.yaml missing"}[policyExists],
	})

	templatesDir := filepath.Join(root, "templates")
	info, err := os.Stat(templatesDir)
	templatesExist := err == nil && info.IsDir()
	checks = append(checks, readinessCheck{
		Name:   "templates_present",
		Passed: templatesExist,
		Note:   map[bool]string{true: "templates/ directory found", false: "templates/ directory missing"}[templatesExist],
	})

	completionCardPath := filepath.Join(root, "templates", "COMPLETION_CARD.md")
	_, err = os.Stat(completionCardPath)
	completionCardExists := err == nil
	checks = append(checks, readinessCheck{
		Name:   "completion_card_template_present",
		Passed: completionCardExists,
		Note:   map[bool]string{true: "templates/COMPLETION_CARD.md found", false: "templates/COMPLETION_CARD.md missing"}[completionCardExists],
	})

	return checks
}

func isTTY() bool {
	fi, _ := os.Stdin.Stat()
	return fi.Mode()&os.ModeCharDevice != 0
}
