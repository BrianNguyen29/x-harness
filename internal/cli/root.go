package cli

import (
	"fmt"
	"io"
	"strings"
)

var Version = "0.1.0"

func SetVersion(v string) {
	if v != "" {
		Version = v
	}
}

type Maturity string

const (
	MaturityStable       Maturity = "stable"
	MaturityBeta         Maturity = "beta"
	MaturityExperimental Maturity = "experimental"
	MaturitySkeletal     Maturity = "skeletal"
)

type CommandInfo struct {
	Name        string
	Description string
	Primary     bool
	Maturity    Maturity
}

var commands = []CommandInfo{
	{Name: "verify", Description: "Run read-only verification against a completion card", Primary: true, Maturity: MaturityStable},
	{Name: "check", Description: "Alias for verify", Primary: true, Maturity: MaturityStable},
	{Name: "doctor", Description: "Validate workspace health and configuration", Primary: true, Maturity: MaturityStable},
	{Name: "examples", Description: "Verify bundled examples", Primary: true, Maturity: MaturityStable},
	{Name: "context", Description: "Show canonical context and runtime contract", Primary: true, Maturity: MaturityStable},
	{Name: "benchmark", Description: "Measure latency and verification benchmark behavior", Primary: true, Maturity: MaturityStable},
	{Name: "handoff", Description: "Generate structured handoff prompts", Primary: true, Maturity: MaturityStable},
	{Name: "prepare", Description: "Alias for handoff readiness", Primary: true, Maturity: MaturityStable},
	{Name: "report", Description: "Show trace summary or metrics report", Primary: true, Maturity: MaturityStable},
	{Name: "status", Description: "Alias for report", Primary: true, Maturity: MaturityStable},
	{Name: "trace", Description: "Append or verify trace events", Primary: true, Maturity: MaturityStable},
	{Name: "clean", Description: "Clean generated harness state", Primary: true, Maturity: MaturityStable},
	{Name: "reset", Description: "Alias for safe generated-state cleanup", Primary: true, Maturity: MaturityStable},
	{Name: "init", Description: "Install harness assets into a workspace", Maturity: MaturityStable},
	{Name: "add", Description: "Add claim, evidence, or completion card helpers", Maturity: MaturityStable},
	{Name: "recovery", Description: "Generate recovery suggestions", Maturity: MaturityStable},
	{Name: "recover", Description: "Alias for recovery suggest", Maturity: MaturityStable},
	{Name: "packet", Description: "Work with claim/evidence packets", Maturity: MaturityBeta},
	{Name: "intake", Description: "Evaluate task intake tiering", Maturity: MaturityExperimental},
	{Name: "governance", Description: "Evaluate governance rules", Maturity: MaturityExperimental},
	{Name: "intervention", Description: "Record governance interventions", Maturity: MaturityExperimental},
	{Name: "prediction", Description: "Evaluate prediction/checklist claims", Maturity: MaturityExperimental},
	{Name: "components", Description: "Inspect component registry coverage", Maturity: MaturityExperimental},
	{Name: "evidence", Description: "Manage evidence corpus entries", Maturity: MaturityExperimental},
	{Name: "episode", Description: "Create episode packages", Maturity: MaturityExperimental},
	{Name: "attribution", Description: "Evaluate attribution metadata", Maturity: MaturityExperimental},
	{Name: "permissions", Description: "Evaluate permission rules", Maturity: MaturityExperimental},
	{Name: "evolve", Description: "Evaluate evolution candidates", Maturity: MaturityExperimental},
	{Name: "export", Description: "Export frozen artifacts", Maturity: MaturityExperimental},
	{Name: "import", Description: "Import frozen artifacts", Maturity: MaturityExperimental},
	{Name: "frozen", Description: "Inspect frozen manifests", Maturity: MaturityExperimental},
	{Name: "federation", Description: "Evaluate federation patterns", Maturity: MaturityExperimental},
	{Name: "approval-risk", Description: "Evaluate approval risk", Maturity: MaturityExperimental},
	{Name: "agent-profile", Description: "Inspect agent profiles", Maturity: MaturityExperimental},
	{Name: "cost", Description: "Evaluate cost budget data", Maturity: MaturityExperimental},
	{Name: "profile", Description: "Recommend installation profiles", Maturity: MaturityBeta},
	{Name: "repair", Description: "Repair managed files from manifest", Maturity: MaturityBeta},
	{Name: "uninstall", Description: "Uninstall managed files using manifest", Maturity: MaturityBeta},
	{Name: "actions", Description: "List beginner-friendly actions", Maturity: MaturityBeta},
	{Name: "card", Description: "Generate or verify admission cards", Maturity: MaturityBeta},
	{Name: "conformance", Description: "Run conformance checks", Maturity: MaturityBeta},
	{Name: "readiness", Description: "Evaluate readiness levels", Maturity: MaturityBeta},
	{Name: "release", Description: "Generate or verify release evidence", Maturity: MaturityBeta},
	{Name: "adapters", Description: "Inspect adapter matrix", Maturity: MaturityBeta},
	{Name: "scan", Description: "Run static security scan on adapter or skill files", Maturity: MaturityBeta},
	{Name: "contract", Description: "Run contract oracle checks", Maturity: MaturityExperimental},
	{Name: "policy", Description: "Show policy enforcement matrix and rule explainers", Maturity: MaturityBeta},
	{Name: "explain", Description: "Explain a completion card's admission/withheld state", Maturity: MaturityBeta},
}

func isBeginnerCommand(name string) bool {
	switch name {
	case "check", "prepare", "recover", "doctor", "actions", "status", "reset", "init", "add":
		return true
	}
	return false
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printStartHere(stdout)
		return ExitOK
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
		return ExitOK
	case "--help-all":
		printHelpAll(stdout)
		return ExitOK
	case "--help-maturity":
		printHelpMaturity(stdout)
		return ExitOK
	case "-v", "--version", "version":
		WriteLine(stdout, "xh %s", Version)
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
	case "add":
		return handleAdd(args[1:], stdout, stderr)
	case "init":
		return handleInit(args[1:], stdout, stderr)
	case "recovery":
		return handleRecovery(args[1:], stdout, stderr)
	case "packet":
		return handlePacket(args[1:], stdout, stderr)
	case "attribution":
		return handleAttribution(args[1:], stdout, stderr)
	case "evidence":
		return handleEvidence(args[1:], stdout, stderr)
	case "episode":
		return handleEpisode(args[1:], stdout, stderr)
	case "components":
		return handleComponents(args[1:], stdout, stderr)
	case "permissions":
		return handlePermissions(args[1:], stdout, stderr)
	case "prediction":
		return handlePrediction(args[1:], stdout, stderr)
	case "approval-risk":
		return handleApprovalRisk(args[1:], stdout, stderr)
	case "agent-profile":
		return handleAgentProfile(args[1:], stdout, stderr)
	case "cost":
		return handleCost(args[1:], stdout, stderr)
	case "evolve":
		return handleEvolve(args[1:], stdout, stderr)
	case "frozen":
		return handleFrozen(args[1:], stdout, stderr)
	case "federation":
		return handleFederation(args[1:], stdout, stderr)
	case "governance":
		return handleGovernance(args[1:], stdout, stderr)
	case "clean":
		return handleClean(args[1:], stdout, stderr)
	case "intervention":
		return handleIntervention(args[1:], stdout, stderr)
	case "intake":
		return handleIntake(args[1:], stdout, stderr)
	case "export":
		return handleFrozenExport(append([]string{"--frozen"}, args[1:]...), stdout, stderr)
	case "import":
		return handleFrozenImport(append([]string{"--frozen"}, args[1:]...), stdout, stderr)
	case "card":
		return handleCard(args[1:], stdout, stderr)
	case "conformance":
		return handleConformance(args[1:], stdout, stderr)
	case "readiness":
		return handleReadiness(args[1:], stdout, stderr)
	case "release":
		return handleRelease(args[1:], stdout, stderr)
	case "adapters":
		return handleAdapters(args[1:], stdout, stderr)
	case "scan":
		return handleScan(args[1:], stdout, stderr)
	case "contract":
		return handleContract(args[1:], stdout, stderr)
	case "policy":
		return handlePolicy(args[1:], stdout, stderr)
	case "explain":
		return handleExplain(args[1:], stdout, stderr)
	case "profile":
		return handleProfile(args[1:], stdout, stderr)
	case "repair":
		return handleRepair(args[1:], stdout, stderr)
	case "uninstall":
		return handleUninstall(args[1:], stdout, stderr)
	default:
		return handleStub(args, stdout, stderr)
	}
}

func handleStub(args []string, _ io.Writer, stderr io.Writer) int {
	name := args[0]
	if !knownCommand(name) {
		WriteLine(stderr, "unknown command: %s", name)
		WriteLine(stderr, "run `xh --help` for usage")
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

func printStartHere(w io.Writer) {
	WriteLine(w, "xh %s", Version)
	WriteLine(w, "")
	WriteLine(w, "Start here — a few commands to get you going:")
	WriteLine(w, "")
	WriteLine(w, "  xh check          Run read-only verification against a completion card")
	WriteLine(w, "  xh prepare        Check if workspace is ready for agent task handoff")
	WriteLine(w, "  xh recover        Get recovery playbook suggestions from errors or trace")
	WriteLine(w, "  xh doctor         Validate workspace health and configuration")
	WriteLine(w, "  xh actions        Show this list of actions")
	WriteLine(w, "  xh status         Show trace summary")
	WriteLine(w, "  xh reset          Clean generated harness state")
	WriteLine(w, "")
	WriteLine(w, "For the full command list:")
	WriteLine(w, "  xh --help-all")
	WriteLine(w, "")
	WriteLine(w, "For commands grouped by maturity:")
	WriteLine(w, "  xh --help-maturity")
}

func printHelp(w io.Writer) {
	WriteLine(w, "xh %s", Version)
	WriteLine(w, "")
	WriteLine(w, "A lightweight verify-gated harness for AI-agent workflows.")
	WriteLine(w, "")
	WriteLine(w, "Usage:")
	WriteLine(w, "  xh <command> [options]")
	WriteLine(w, "")
	WriteLine(w, "Commands:")
	for _, command := range commands {
		if isBeginnerCommand(command.Name) {
			WriteLine(w, "  %-12s %s", command.Name, command.Description)
		}
	}
	WriteLine(w, "")
	WriteLine(w, "Advanced:")
	WriteLine(w, "  xh --help-all          Show all commands")
	WriteLine(w, "  xh --help-maturity     Show commands grouped by maturity")
	WriteLine(w, "")
	WriteLine(w, "Global options:")
	WriteLine(w, "  -h, --help          Show help")
	WriteLine(w, "  --help-all          Show all commands")
	WriteLine(w, "  --help-maturity     Show help with maturity labels for all commands")
	WriteLine(w, "  -v, --version       Show version")
}

func printHelpAll(w io.Writer) {
	WriteLine(w, "xh %s", Version)
	WriteLine(w, "")
	WriteLine(w, "A lightweight verify-gated harness for AI-agent workflows.")
	WriteLine(w, "")
	WriteLine(w, "Usage:")
	WriteLine(w, "  xh <command> [options]")
	WriteLine(w, "")
	WriteLine(w, "All commands:")
	for _, command := range commands {
		WriteLine(w, "  %-12s [%s] %s", command.Name, command.Maturity, command.Description)
	}
	WriteLine(w, "")
	WriteLine(w, "Global options:")
	WriteLine(w, "  -h, --help          Show help")
	WriteLine(w, "  --help-all          Show all commands")
	WriteLine(w, "  --help-maturity     Show help with maturity labels for all commands")
	WriteLine(w, "  -v, --version       Show version")
}

func printHelpMaturity(w io.Writer) {
	WriteLine(w, "xh %s", Version)
	WriteLine(w, "")
	WriteLine(w, "A lightweight verify-gated harness for AI-agent workflows.")
	WriteLine(w, "")
	WriteLine(w, "Maturity labels:")
	WriteLine(w, "  stable       Core command; tested and relied on in CI")
	WriteLine(w, "  beta         Functional but may change; feedback welcome")
	WriteLine(w, "  experimental New or advanced; semantics may shift")
	WriteLine(w, "  skeletal     Declared but not yet implemented")
	WriteLine(w, "")
	WriteLine(w, "Usage:")
	WriteLine(w, "  xh <command> [options]")
	WriteLine(w, "")

	maturityGroups := map[Maturity][]CommandInfo{
		MaturityStable:       {},
		MaturityBeta:         {},
		MaturityExperimental: {},
		MaturitySkeletal:     {},
	}

	for _, command := range commands {
		maturityGroups[command.Maturity] = append(maturityGroups[command.Maturity], command)
	}

	for _, mat := range []Maturity{MaturityStable, MaturityBeta, MaturityExperimental, MaturitySkeletal} {
		group := maturityGroups[mat]
		if len(group) > 0 {
			WriteLine(w, "%s:", mat)
			for _, command := range group {
				WriteLine(w, "  %-12s %s", command.Name, command.Description)
			}
			WriteLine(w, "")
		}
	}

	WriteLine(w, "Global options:")
	WriteLine(w, "  -h, --help          Show help")
	WriteLine(w, "  --help-all          Show all commands")
	WriteLine(w, "  --help-maturity     Show help with maturity labels for all commands")
	WriteLine(w, "  -v, --version       Show version")
}

func printActions(w io.Writer) {
	WriteLine(w, "# xh Beginner Actions")
	WriteLine(w, "")
	WriteLine(w, "| Action | Maturity | Description |")
	WriteLine(w, "| :-- | :-- | :-- |")
	for _, command := range commands {
		if isBeginnerCommand(command.Name) {
			WriteLine(w, "| %s | %s | %s |", command.Name, command.Maturity, command.Description)
		}
	}
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
	return fmt.Sprintf("xh %s\n", Version)
}
