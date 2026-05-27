package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/evolve"
)

func handleEvolve(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "evolve requires a subcommand: evaluate, analyze, propose, constitution-check, compare, promote, rollback")
		return ExitUsage
	}

	switch args[0] {
	case "evaluate":
		return handleEvolveEvaluate(args[1:], stdout, stderr)
	case "analyze":
		return handleEvolveAnalyze(args[1:], stdout, stderr)
	case "propose":
		return handleEvolvePropose(args[1:], stdout, stderr)
	case "constitution-check":
		return handleEvolveConstitutionCheck(args[1:], stdout, stderr)
	case "compare":
		return handleEvolveCompare(args[1:], stdout, stderr)
	case "promote":
		return handleEvolvePromote(args[1:], stdout, stderr)
	case "rollback":
		return handleEvolveRollback(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown evolve subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleEvolveEvaluate(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
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

	root, _ = filepath.Abs(root)
	budget, err := evolve.LoadBudget(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	result := evolve.EvaluateBudget(budget)
	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, result.Message)
	}
	return ExitOK
}

func handleEvolveAnalyze(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	runID := ""
	outPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--run":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --run requires a value")
				return ExitUsage
			}
			runID = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			outPath = args[i+1]
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

	if runID == "" {
		fmt.Fprintln(stderr, "Error: --run <run-id> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	content := evolve.RenderChangeRequest("analysis", fmt.Sprintf("Analyze evolution run %s", runID), "", "", nil)
	var out string
	var err error
	if outPath != "" {
		out, err = evolve.WriteChangeRequest(root, content, outPath)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
	}

	result := map[string]interface{}{
		"ok":                  true,
		"status":              "written",
		"path":                out,
		"run_id":              runID,
		"admission_authority": false,
	}
	if out == "" {
		result["status"] = "proposed"
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		if out != "" {
			fmt.Fprintf(stdout, "analysis request written: %s\n", out)
		} else {
			fmt.Fprint(stdout, content)
		}
	}
	return ExitOK
}

func handleEvolvePropose(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	component := ""
	outPath := ""
	writeFlag := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--component":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --component requires a value")
				return ExitUsage
			}
			component = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			outPath = args[i+1]
			i++
		case "--write":
			writeFlag = true
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

	if component == "" {
		fmt.Fprintln(stderr, "Error: --component <id> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	content := evolve.RenderChangeRequest("proposal", fmt.Sprintf("Propose a candidate for %s", component), component, "", nil)
	var out string
	var err error
	if writeFlag || outPath != "" {
		out, err = evolve.WriteChangeRequest(root, content, outPath)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
	}

	result := map[string]interface{}{
		"ok":                  true,
		"status":              "written",
		"path":                out,
		"component":           component,
		"admission_authority": false,
	}
	if out == "" {
		result["status"] = "proposed"
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		if out != "" {
			fmt.Fprintf(stdout, "change request written: %s\n", out)
		} else {
			fmt.Fprint(stdout, content)
		}
	}
	return ExitOK
}

func handleEvolveConstitutionCheck(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	candidate := ""
	constitutionPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--candidate":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --candidate requires a value")
				return ExitUsage
			}
			candidate = args[i+1]
			i++
		case "--constitution":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --constitution requires a value")
				return ExitUsage
			}
			constitutionPath = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
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

	if candidate == "" {
		fmt.Fprintln(stderr, "Error: --candidate <path-or-id> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	constitution, cpath, err := evolve.LoadConstitution(root, constitutionPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	cand, candPath, err := evolve.LoadCandidate(root, candidate)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	result := evolve.CheckConstitution(constitution, cpath, cand, candPath)

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else if result.OK {
		fmt.Fprintf(stdout, "constitution passed: %s\n", result.CandidateID)
	} else {
		fmt.Fprintf(stdout, "constitution failed: %s\n", result.CandidateID)
		for _, v := range result.Violations {
			fmt.Fprintf(stdout, "- %s\n", v)
		}
	}

	if !result.OK {
		return ExitError
	}
	return ExitOK
}

func handleEvolveCompare(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	candidate := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--candidate":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --candidate requires a value")
				return ExitUsage
			}
			candidate = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
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

	if candidate == "" {
		fmt.Fprintln(stderr, "Error: --candidate <path-or-id> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	constitution, cpath, err := evolve.LoadConstitution(root, "")
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	cand, candPath, err := evolve.LoadCandidate(root, candidate)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	result := evolve.CheckConstitution(constitution, cpath, cand, candPath)
	falseAcceptRegression := false
	for _, v := range result.Violations {
		if strings.Contains(v, "false_accept") {
			falseAcceptRegression = true
			break
		}
	}

	output := map[string]interface{}{
		"ok":                      result.OK,
		"candidate_id":            result.CandidateID,
		"constitution_status":     result.Status,
		"false_accept_regression": falseAcceptRegression,
		"admission_authority":     false,
	}

	if jsonMode {
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(stdout, string(data))
	}

	if !result.OK {
		return ExitError
	}
	return ExitOK
}

func handleEvolvePromote(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	candidate := ""
	outPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--candidate":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --candidate requires a value")
				return ExitUsage
			}
			candidate = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			outPath = args[i+1]
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

	if candidate == "" {
		fmt.Fprintln(stderr, "Error: --candidate <path-or-id> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	constitution, cpath, err := evolve.LoadConstitution(root, "")
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	cand, candPath, err := evolve.LoadCandidate(root, candidate)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	checkResult := evolve.CheckConstitution(constitution, cpath, cand, candPath)
	if !checkResult.OK {
		if jsonMode {
			data, _ := json.MarshalIndent(checkResult, "", "  ")
			fmt.Fprintln(stdout, string(data))
		}
		fmt.Fprintf(stderr, "promotion blocked by constitution\n")
		return ExitError
	}

	content := evolve.RenderChangeRequest("promotion", "Promotion requires human review and explicit merge outside x-harness.", "", checkResult.CandidateID, checkResult)
	out, err := evolve.WriteChangeRequest(root, content, outPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	result := map[string]interface{}{
		"ok":                  true,
		"status":              "written",
		"path":                out,
		"candidate_id":        checkResult.CandidateID,
		"admission_authority": false,
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "promotion request written: %s\n", out)
	}
	return ExitOK
}

func handleEvolveRollback(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	candidate := ""
	outPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--candidate":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --candidate requires a value")
				return ExitUsage
			}
			candidate = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			outPath = args[i+1]
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

	if candidate == "" {
		fmt.Fprintln(stderr, "Error: --candidate <path-or-id> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	constitution, cpath, err := evolve.LoadConstitution(root, "")
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	cand, candPath, err := evolve.LoadCandidate(root, candidate)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	checkResult := evolve.CheckConstitution(constitution, cpath, cand, candPath)
	content := evolve.RenderChangeRequest("rollback", "Rollback requires human review and explicit git operation outside x-harness.", "", checkResult.CandidateID, checkResult)
	out, err := evolve.WriteChangeRequest(root, content, outPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	result := map[string]interface{}{
		"ok":                  true,
		"status":              "written",
		"path":                out,
		"candidate_id":        checkResult.CandidateID,
		"admission_authority": false,
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "rollback request written: %s\n", out)
	}
	return ExitOK
}
