package cli

import (
	"fmt"
	"io"
	"strings"
)

// ContractFact is a single canonical contract fact.
type ContractFact struct {
	Rule        string `json:"rule"`
	Description string `json:"description"`
}

// Contract holds the canonical x-harness contract facts.
type Contract struct {
	Facts []ContractFact `json:"facts"`
}

// CoreContract returns the canonical contract derived from repository assets.
func CoreContract() Contract {
	return Contract{
		Facts: []ContractFact{
			{
				Rule:        "completion_admitted_not_claimed",
				Description: "Completion is admitted, not claimed. Agents may propose completion but cannot self-admit.",
			},
			{
				Rule:        "verifier_read_only",
				Description: "The verifier is read-only. It must not edit source files or repair the work product while verifying.",
			},
			{
				Rule:        "success_only_accepted",
				Description: "Success is the only accepted outcome. admission.outcome: success and acceptance_status: accepted are required.",
			},
			{
				Rule:        "canonical_tiers",
				Description: "Canonical tiers are light, standard, and deep. Do not use small, medium, or large in active runtime handoffs.",
			},
			{
				Rule:        "pgv_advisory_only",
				Description: "PGV is advisory-only. It never overrides verify and never grants admission authority by default.",
			},
		},
	}
}

func runContext(args []string, stdout io.Writer, _ io.Writer) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
		}
	}

	contract := CoreContract()

	if jsonMode {
		if err := WriteJSON(stdout, contract); err != nil {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "x-harness Canonical Contract")
	WriteLine(stdout, "")
	for _, fact := range contract.Facts {
		WriteLine(stdout, "- %s", strings.ReplaceAll(fact.Description, "\n", "\n  "))
	}
	return ExitOK
}

func handleContext(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness context --contract [--json]")
		return ExitUsage
	}

	switch args[0] {
	case "--contract":
		return runContext(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown context subcommand: %s\n", args[0])
		return ExitUsage
	}
}
