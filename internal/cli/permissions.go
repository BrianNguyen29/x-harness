package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/permissions"
)

func handlePermissions(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "permissions requires a subcommand: check, explain, test-fixtures")
		return ExitUsage
	}

	switch args[0] {
	case "check":
		return handlePermissionsCheck(args[1:], stdout, stderr)
	case "explain":
		return handlePermissionsExplain(args[1:], stdout, stderr)
	case "test-fixtures":
		return handlePermissionsTestFixtures(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown permissions subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func parsePermissionFlags(args []string, requireRole bool, stderr io.Writer) (role, tier, command, capability, intervention, root string, jsonMode bool, exitCode int) {
	tier = "standard"
	root = "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--role":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --role requires a value")
				return "", "", "", "", "", "", false, ExitUsage
			}
			role = args[i+1]
			i++
		case "--tier":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --tier requires a value")
				return "", "", "", "", "", "", false, ExitUsage
			}
			tier = args[i+1]
			i++
		case "--command":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --command requires a value")
				return "", "", "", "", "", "", false, ExitUsage
			}
			command = args[i+1]
			i++
		case "--capability":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --capability requires a value")
				return "", "", "", "", "", "", false, ExitUsage
			}
			capability = args[i+1]
			i++
		case "--intervention":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --intervention requires a value")
				return "", "", "", "", "", "", false, ExitUsage
			}
			intervention = args[i+1]
			i++
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return "", "", "", "", "", "", false, ExitUsage
			}
			root = args[i+1]
			i++
		case "--json":
			jsonMode = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return "", "", "", "", "", "", false, ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return "", "", "", "", "", "", false, ExitUsage
		}
	}

	if requireRole && role == "" {
		fmt.Fprintln(stderr, "Error: --role is required")
		return "", "", "", "", "", "", false, ExitUsage
	}

	if requireRole && command != "" && capability != "" {
		fmt.Fprintln(stderr, "Error: provide only one of --command or --capability")
		return "", "", "", "", "", "", false, ExitUsage
	}

	if requireRole && command == "" && capability == "" {
		fmt.Fprintln(stderr, "Error: --command or --capability is required")
		return "", "", "", "", "", "", false, ExitUsage
	}

	root, _ = filepath.Abs(root)
	return role, tier, command, capability, intervention, root, jsonMode, -1
}

func renderDecisionText(stdout io.Writer, decision permissions.PermissionDecision) {
	fmt.Fprintf(stdout, "# x-harness Permission %s\n", decision.Status)
	fmt.Fprintf(stdout, "- ok: %v\n", decision.OK)
	fmt.Fprintf(stdout, "- role: %s\n", decision.Role)
	fmt.Fprintf(stdout, "- tier: %s\n", decision.Tier)
	if decision.Command != nil {
		fmt.Fprintf(stdout, "- command: %s\n", *decision.Command)
	}
	if decision.Capability != nil {
		fmt.Fprintf(stdout, "- capability: %s\n", *decision.Capability)
	}
	fmt.Fprintf(stdout, "- reason: %s\n", decision.Reason)
	if decision.Matched.CommandSet != nil {
		fmt.Fprintf(stdout, "- command_set: %s\n", *decision.Matched.CommandSet)
	}
	if decision.Matched.Rule != nil {
		fmt.Fprintf(stdout, "- rule: %s\n", *decision.Matched.Rule)
	}
	if decision.Intervention.Provided {
		fmt.Fprintf(stdout, "- intervention_valid: %v\n", decision.Intervention.Valid)
		if decision.Intervention.Reason != nil {
			fmt.Fprintf(stdout, "- intervention_reason: %s\n", *decision.Intervention.Reason)
		}
	}
}

func handlePermissionsCheck(args []string, stdout, stderr io.Writer) int {
	role, tier, command, capability, intervention, root, jsonMode, exitCode := parsePermissionFlags(args, true, stderr)
	if exitCode >= 0 {
		return exitCode
	}

	policy, err := permissions.LoadPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	decision, err := permissions.CheckPermission(policy, root, role, tier, command, capability, intervention)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(decision, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		renderDecisionText(stdout, decision)
	}

	if decision.OK {
		return ExitOK
	}
	return ExitError
}

func handlePermissionsExplain(args []string, stdout, stderr io.Writer) int {
	role, tier, command, capability, intervention, root, jsonMode, exitCode := parsePermissionFlags(args, true, stderr)
	if exitCode >= 0 {
		return exitCode
	}

	policy, err := permissions.LoadPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	decision, err := permissions.CheckPermission(policy, root, role, tier, command, capability, intervention)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(decision, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		renderDecisionText(stdout, decision)
	}

	return ExitOK
}

func handlePermissionsTestFixtures(args []string, stdout, stderr io.Writer) int {
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
	policy, err := permissions.LoadPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	result, err := permissions.RunFixtures(policy, root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "# x-harness Permission Fixtures")
		for _, fixture := range result.Fixtures {
			status := "fail"
			if fixture.OK {
				status = "pass"
			}
			fmt.Fprintf(stdout, "- %s %s: expected %s, got %s\n", status, fixture.Name, fixture.ExpectedStatus, fixture.ActualStatus)
		}
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}
