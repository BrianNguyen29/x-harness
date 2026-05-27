package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/agentprofile"
)

func handleAgentProfile(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "agent-profile requires a subcommand: update, report")
		return ExitUsage
	}

	switch args[0] {
	case "update":
		return handleAgentProfileUpdate(args[1:], stdout, stderr)
	case "report":
		return handleAgentProfileReport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown agent-profile subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleAgentProfileUpdate(args []string, stdout, stderr io.Writer) int {
	var agentID, benchmarkPath, outPath, root string
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --agent requires a value")
				return ExitUsage
			}
			agentID = args[i+1]
			i++
		case "--from-benchmark":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --from-benchmark requires a value")
				return ExitUsage
			}
			benchmarkPath = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --out requires a value")
				return ExitUsage
			}
			outPath = args[i+1]
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

	if agentID == "" {
		fmt.Fprintln(stderr, "Error: --agent <id> is required")
		return ExitUsage
	}

	if root == "" {
		root = "."
	}
	root, _ = filepath.Abs(root)

	profile, err := agentprofile.BuildAgentProfile(agentID, benchmarkPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if err := agentprofile.ValidateAgentProfile(profile, root); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if outPath == "" {
		outPath = agentprofile.DefaultAgentProfilePath(root, agentID)
	}

	if err := agentprofile.WriteAgentProfile(profile, outPath); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		output := map[string]any{
			"ok":      true,
			"path":    outPath,
			"profile": profile,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "# x-harness Agent Profile: %s\n", profile.AgentID)
		fmt.Fprintf(stdout, "- observed_failure_modes: %d\n", len(profile.ObservedFailureModes))
		fmt.Fprintf(stdout, "- required_extra_checks: %s\n", strings.Join(profile.RequiredExtraChecks, ", "))
		fmt.Fprintf(stdout, "- path: %s\n", outPath)
	}

	return ExitOK
}

func handleAgentProfileReport(args []string, stdout, stderr io.Writer) int {
	var profilePath, agentID, root string
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --profile requires a value")
				return ExitUsage
			}
			profilePath = args[i+1]
			i++
		case "--agent":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --agent requires a value")
				return ExitUsage
			}
			agentID = args[i+1]
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

	if root == "" {
		root = "."
	}
	root, _ = filepath.Abs(root)

	if profilePath == "" {
		if agentID == "" {
			fmt.Fprintln(stderr, "Error: agent-profile report requires --profile or --agent")
			return ExitUsage
		}
		profilePath = agentprofile.DefaultAgentProfilePath(root, agentID)
	}

	profile, err := agentprofile.ReadAgentProfile(profilePath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if err := agentprofile.ValidateAgentProfile(profile, root); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(profile, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "# x-harness Agent Profile: %s\n", profile.AgentID)
		fmt.Fprintf(stdout, "- advisory_only: %v\n", profile.AdvisoryOnly)
		fmt.Fprintf(stdout, "- admission_authority: %v\n", profile.AdmissionAuthority)
	}

	return ExitOK
}
