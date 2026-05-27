package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/cost"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"gopkg.in/yaml.v3"
)

func handleCost(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "cost requires a subcommand: check, report")
		return ExitUsage
	}

	switch args[0] {
	case "check":
		return handleCostCheck(args[1:], stdout, stderr)
	case "report":
		return handleCostReport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown cost subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleCostCheck(args []string, stdout, stderr io.Writer) int {
	var actualUSDStr, inputTokensStr, outputTokensStr, root string
	var enforce, jsonMode bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--actual-usd":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --actual-usd requires a value")
				return ExitUsage
			}
			actualUSDStr = args[i+1]
			i++
		case "--input-tokens":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --input-tokens requires a value")
				return ExitUsage
			}
			inputTokensStr = args[i+1]
			i++
		case "--output-tokens":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --output-tokens requires a value")
				return ExitUsage
			}
			outputTokensStr = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--enforce":
			enforce = true
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

	if actualUSDStr == "" {
		fmt.Fprintln(stderr, "error: --actual-usd is required")
		return ExitUsage
	}
	if inputTokensStr == "" {
		fmt.Fprintln(stderr, "error: --input-tokens is required")
		return ExitUsage
	}
	if outputTokensStr == "" {
		fmt.Fprintln(stderr, "error: --output-tokens is required")
		return ExitUsage
	}

	actualUSD, err := strconv.ParseFloat(actualUSDStr, 64)
	if err != nil || actualUSD < 0 {
		fmt.Fprintln(stderr, "error: --actual-usd must be a non-negative number")
		return ExitUsage
	}

	inputTokens, err := strconv.ParseInt(inputTokensStr, 10, 64)
	if err != nil || inputTokens < 0 {
		fmt.Fprintln(stderr, "error: --input-tokens must be a non-negative integer")
		return ExitUsage
	}

	outputTokens, err := strconv.ParseInt(outputTokensStr, 10, 64)
	if err != nil || outputTokens < 0 {
		fmt.Fprintln(stderr, "error: --output-tokens must be a non-negative integer")
		return ExitUsage
	}

	if root == "" {
		root = "."
	}
	root, _ = filepath.Abs(root)

	policy, err := cost.LoadPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	report := cost.EvaluateCostBudget(policy, actualUSD, inputTokens, outputTokens, enforce)

	if jsonMode {
		recovery := "none"
		if report.OverBudget {
			recovery = policy.OverBudgetRecovery
		}
		output := map[string]interface{}{
			"schema_version":      "1",
			"max_usd":             report.MaxUSD,
			"actual_usd":          report.ActualUSD,
			"token_usage":         map[string]int64{"input": report.InputTokens, "output": report.OutputTokens},
			"over_budget":         report.OverBudget,
			"status":              report.Status,
			"recovery":            recovery,
			"policy_enabled":      policy.Enabled,
			"enforcement_enabled": report.EnforcementEnabled,
			"admission_authority": policy.AffectsAdmission,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "# x-harness Cost Budget")
		fmt.Fprintf(stdout, "- status: %s\n", report.Status)
		fmt.Fprintf(stdout, "- over_budget: %v\n", report.OverBudget)
		fmt.Fprintf(stdout, "- enforcement_enabled: %v\n", report.EnforcementEnabled)
	}

	if report.OverBudget && report.EnforcementEnabled {
		return ExitError
	}
	return ExitOK
}

func handleCostReport(args []string, stdout, stderr io.Writer) int {
	var from string
	var jsonMode bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --from requires a value")
				return ExitUsage
			}
			from = args[i+1]
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

	if from == "" {
		fmt.Fprintln(stderr, "error: --from is required")
		return ExitUsage
	}

	from, _ = filepath.Abs(from)

	// Read raw file for validation
	data, err := os.ReadFile(from)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	// Try to validate against schema if available
	roots := []string{filepath.Dir(from), "."}
	for _, root := range roots {
		schemaPath := filepath.Join(root, "schemas", "cost-budget.schema.json")
		if stat, err := os.Stat(schemaPath); err == nil && !stat.IsDir() {
			validator, err := schema.Compile(schemaPath)
			if err == nil {
				var doc map[string]interface{}
				if err := json.Unmarshal(data, &doc); err != nil {
					_ = yaml.Unmarshal(data, &doc)
				}
				if doc != nil {
					if err := validator.Validate(doc); err != nil {
						fmt.Fprintf(stderr, "error: cost budget report validation failed: %v\n", err)
						return ExitError
					}
				}
			}
			break
		}
	}

	report, err := cost.ReadCostBudgetReport(from)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		// Preserve original JSON structure if possible
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err == nil {
			out, _ := json.MarshalIndent(raw, "", "  ")
			fmt.Fprintln(stdout, string(out))
		} else {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Fprintln(stdout, string(data))
		}
	} else {
		fmt.Fprintf(stdout, "cost budget: %s\n", report.Status)
	}

	return ExitOK
}
