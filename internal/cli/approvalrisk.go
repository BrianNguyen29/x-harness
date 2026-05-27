package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/approvalrisk"
)

func handleApprovalRisk(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "approval-risk requires a subcommand: evaluate, check")
		return ExitUsage
	}

	switch args[0] {
	case "evaluate":
		return handleApprovalRiskEvaluate(args[1:], stdout, stderr)
	case "check":
		return handleApprovalRiskCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown approval-risk subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func parseApprovalRiskFlags(args []string, stderr io.Writer) (card, root string, jsonMode bool, exitCode int) {
	root = "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --card requires a value")
				return "", "", false, ExitUsage
			}
			card = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return "", "", false, ExitUsage
			}
			root = args[i+1]
			i++
		case "--json":
			jsonMode = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return "", "", false, ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return "", "", false, ExitUsage
		}
	}

	if card == "" {
		fmt.Fprintln(stderr, "Error: --card is required")
		return "", "", false, ExitUsage
	}

	root, _ = filepath.Abs(root)
	if !filepath.IsAbs(card) {
		card = filepath.Join(root, card)
	}
	card, _ = filepath.Abs(card)
	return card, root, jsonMode, -1
}

func handleApprovalRiskEvaluate(args []string, stdout, stderr io.Writer) int {
	card, root, jsonMode, exitCode := parseApprovalRiskFlags(args, stderr)
	if exitCode >= 0 {
		return exitCode
	}

	report, err := approvalrisk.EvaluateApprovalRisk(card, root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		// Advisory-only: always return ExitOK even on evaluation errors
		return ExitOK
	}

	if jsonMode {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "# x-harness Approval Risk")
		fmt.Fprintf(stdout, "- task_id: %s\n", report.TaskID)
		fmt.Fprintf(stdout, "- risk_class: %s\n", report.RiskClass)
		fmt.Fprintf(stdout, "- score: %d\n", report.Score)
		fmt.Fprintf(stdout, "- required_approvals: %d\n", report.RequiredApprovals)
		fmt.Fprintf(stdout, "- admission_authority: %v\n", report.AdmissionAuthority)
	}

	return ExitOK
}

func handleApprovalRiskCheck(args []string, stdout, stderr io.Writer) int {
	card, root, jsonMode, exitCode := parseApprovalRiskFlags(args, stderr)
	if exitCode >= 0 {
		return exitCode
	}

	report, err := approvalrisk.EvaluateApprovalRisk(card, root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		// Advisory-only: always return ExitOK even on evaluation errors
		return ExitOK
	}

	if jsonMode {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "approval risk: %s\n", report.RiskClass)
	}

	return ExitOK
}
