package cli

import (
	"fmt"
	"io"
	"strings"
)

const Version = "0.1.0"

type CommandInfo struct {
	Name        string
	Description string
	Primary     bool
}

var commands = []CommandInfo{
	{Name: "verify", Description: "Run read-only verification against a completion card", Primary: true},
	{Name: "check", Description: "Alias for verify", Primary: true},
	{Name: "doctor", Description: "Validate workspace health and configuration", Primary: true},
	{Name: "examples", Description: "Verify bundled examples", Primary: true},
	{Name: "context", Description: "Show canonical context and runtime contract", Primary: true},
	{Name: "benchmark", Description: "Measure latency and verification benchmark behavior", Primary: true},
	{Name: "handoff", Description: "Generate structured handoff prompts", Primary: true},
	{Name: "prepare", Description: "Alias for handoff readiness", Primary: true},
	{Name: "report", Description: "Show trace summary or metrics report", Primary: true},
	{Name: "status", Description: "Alias for report", Primary: true},
	{Name: "trace", Description: "Append or verify trace events", Primary: true},
	{Name: "clean", Description: "Clean generated harness state", Primary: true},
	{Name: "reset", Description: "Alias for safe generated-state cleanup", Primary: true},
	{Name: "init", Description: "Install harness assets into a workspace"},
	{Name: "add", Description: "Add claim, evidence, or completion card helpers"},
	{Name: "recovery", Description: "Generate recovery suggestions"},
	{Name: "recover", Description: "Alias for recovery suggest"},
	{Name: "packet", Description: "Work with claim/evidence packets"},
	{Name: "intake", Description: "Evaluate task intake tiering"},
	{Name: "governance", Description: "Evaluate governance rules"},
	{Name: "intervention", Description: "Record governance interventions"},
	{Name: "prediction", Description: "Evaluate prediction/checklist claims"},
	{Name: "components", Description: "Inspect component registry coverage"},
	{Name: "evidence", Description: "Manage evidence corpus entries"},
	{Name: "episode", Description: "Create episode packages"},
	{Name: "attribution", Description: "Evaluate attribution metadata"},
	{Name: "permissions", Description: "Evaluate permission rules"},
	{Name: "evolve", Description: "Evaluate evolution candidates"},
	{Name: "export", Description: "Export frozen artifacts"},
	{Name: "import", Description: "Import frozen artifacts"},
	{Name: "frozen", Description: "Inspect frozen manifests"},
	{Name: "federation", Description: "Evaluate federation patterns"},
	{Name: "approval-risk", Description: "Evaluate approval risk"},
	{Name: "agent-profile", Description: "Inspect agent profiles"},
	{Name: "cost", Description: "Evaluate cost budget data"},
	{Name: "actions", Description: "List beginner-friendly actions"},
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return ExitOK
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
		return ExitOK
	case "-v", "--version", "version":
		WriteLine(stdout, "x-harness %s", Version)
		return ExitOK
	case "actions":
		printActions(stdout)
		return ExitOK
	case "context":
		return handleContext(args[1:], stdout, stderr)
	case "verify", "check":
		return handleVerify(args[1:], stdout, stderr)
	case "doctor":
		return handleDoctor(args[1:], stdout, stderr)
	case "benchmark":
		return handleBenchmark(args[1:], stdout, stderr)
	case "examples":
		return handleExamples(args[1:], stdout, stderr)
	case "handoff":
		return handleHandoff(args[1:], stdout, stderr)
	case "prepare":
		return handlePrepare(args[1:], stdout, stderr)
	case "trace":
		return handleTrace(args[1:], stdout, stderr)
	case "report", "status":
		return handleReport(args[1:], stdout, stderr)
	case "recover":
		return handleRecover(args[1:], stdout, stderr)
	case "reset":
		return handleReset(args[1:], stdout, stderr)
	default:
		return handleStub(args, stdout, stderr)
	}
}

func handleStub(args []string, _ io.Writer, stderr io.Writer) int {
	name := args[0]
	if !knownCommand(name) {
		WriteLine(stderr, "unknown command: %s", name)
		WriteLine(stderr, "run `x-harness --help` for usage")
		return ExitUsage
	}

	WriteLine(
		stderr,
		"command %q is declared in the Go CLI skeleton but not implemented yet",
		name,
	)
	return ExitUsage
}

func knownCommand(name string) bool {
	for _, command := range commands {
		if command.Name == name {
			return true
		}
	}
	return false
}

func printHelp(w io.Writer) {
	WriteLine(w, "x-harness %s", Version)
	WriteLine(w, "")
	WriteLine(w, "A lightweight verify-gated harness for AI-agent workflows.")
	WriteLine(w, "")
	WriteLine(w, "Usage:")
	WriteLine(w, "  x-harness <command> [options]")
	WriteLine(w, "")
	WriteLine(w, "Primary commands:")
	for _, command := range commands {
		if command.Primary {
			WriteLine(w, "  %-12s %s", command.Name, command.Description)
		}
	}
	WriteLine(w, "")
	WriteLine(w, "Global options:")
	WriteLine(w, "  -h, --help       Show help")
	WriteLine(w, "  -v, --version    Show version")
}

func printActions(w io.Writer) {
	WriteLine(w, "# x-harness Beginner Actions")
	WriteLine(w, "")
	WriteLine(w, "| Action | Description |")
	WriteLine(w, "| :-- | :-- |")
	WriteLine(w, "| prepare | Check if workspace is ready for agent task handoff |")
	WriteLine(w, "| check | Run read-only verification against a completion card |")
	WriteLine(w, "| recover | Get recovery playbook suggestions from errors or trace |")
	WriteLine(w, "| doctor | Validate workspace health and configuration |")
	WriteLine(w, "| actions | Show this list of actions |")
	WriteLine(w, "| status | Show trace summary or card metrics |")
	WriteLine(w, "| reset | Clean generated harness state |")
}

func PrimaryCommandNames() []string {
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		if command.Primary {
			names = append(names, command.Name)
		}
	}
	return names
}

func HelpText() string {
	var builder strings.Builder
	printHelp(&builder)
	return builder.String()
}

func VersionText() string {
	return fmt.Sprintf("x-harness %s\n", Version)
}
