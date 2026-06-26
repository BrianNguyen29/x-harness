package cli

import (
	"fmt"
	"io"
	"strings"
)

var Version = "dev"

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
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Primary         bool     `json:"primary,omitempty"`
	Onboarding      bool     `json:"onboarding,omitempty"`
	OnboardingOrder int      `json:"onboarding_order,omitempty"`
	Maturity        Maturity `json:"maturity"`
}

func isBeginnerCommand(name string) bool {
	command, ok := commandByName(name)
	return ok && command.Onboarding
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	lang, cleanArgs := parseLang(args)

	if len(cleanArgs) == 0 {
		printStartHere(stdout, lang)
		return ExitOK
	}

	switch cleanArgs[0] {
	case "-h", "--help", "help":
		printHelp(stdout, lang)
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
		printActions(stdout, lang)
		return ExitOK
	case "start":
		return handleStart(cleanArgs[1:], stdout, stderr, lang)
	case "learn":
		return handleLearn(cleanArgs[1:], stdout, stderr, lang)
	case "quick":
		return handleQuick(cleanArgs[1:], stdout, stderr, lang)
	case "run":
		return handleRun(cleanArgs[1:], stdout, stderr)
	case "ci":
		return handleRun(append([]string{"builtin:ci"}, cleanArgs[1:]...), stdout, stderr)
	case "context":
		return handleContext(cleanArgs[1:], stdout, stderr)
	case "verify", "check":
		return handleVerify(cleanArgs[1:], stdout, stderr)
	case "doctor":
		return handleDoctor(cleanArgs[1:], stdout, stderr)
	case "benchmark":
		return handleBenchmark(cleanArgs[1:], stdout, stderr)
	case "examples":
		return handleExamples(cleanArgs[1:], stdout, stderr)
	case "handoff":
		return handleHandoff(cleanArgs[1:], stdout, stderr)
	case "prepare":
		return handlePrepare(cleanArgs[1:], stdout, stderr)
	case "trace":
		return handleTrace(cleanArgs[1:], stdout, stderr)
	case "report", "status":
		return handleReport(cleanArgs[1:], stdout, stderr)
	case "recover":
		return handleRecover(cleanArgs[1:], stdout, stderr)
	case "reset":
		return handleReset(cleanArgs[1:], stdout, stderr)
	case "add":
		return handleAdd(cleanArgs[1:], stdout, stderr)
	case "init":
		return handleInit(cleanArgs[1:], stdout, stderr)
	case "recovery":
		return handleRecovery(cleanArgs[1:], stdout, stderr)
	case "packet":
		return handlePacket(cleanArgs[1:], stdout, stderr)
	case "attribution":
		return handleAttribution(cleanArgs[1:], stdout, stderr)
	case "evidence":
		return handleEvidence(cleanArgs[1:], stdout, stderr)
	case "episode":
		return handleEpisode(cleanArgs[1:], stdout, stderr)
	case "components":
		return handleComponents(cleanArgs[1:], stdout, stderr)
	case "permissions":
		return handlePermissions(cleanArgs[1:], stdout, stderr)
	case "prediction":
		return handlePrediction(cleanArgs[1:], stdout, stderr)
	case "approval-risk":
		return handleApprovalRisk(cleanArgs[1:], stdout, stderr)
	case "agent-profile":
		return handleAgentProfile(cleanArgs[1:], stdout, stderr)
	case "cost":
		return handleCost(cleanArgs[1:], stdout, stderr)
	case "evolve":
		return handleEvolve(cleanArgs[1:], stdout, stderr)
	case "frozen":
		return handleFrozen(cleanArgs[1:], stdout, stderr)
	case "federation":
		return handleFederation(cleanArgs[1:], stdout, stderr)
	case "governance":
		return handleGovernance(cleanArgs[1:], stdout, stderr)
	case "clean":
		return handleClean(cleanArgs[1:], stdout, stderr)
	case "intervention":
		return handleIntervention(cleanArgs[1:], stdout, stderr)
	case "intake":
		return handleIntake(cleanArgs[1:], stdout, stderr)
	case "decision":
		return handleDecision(cleanArgs[1:], stdout, stderr)
	case "export":
		return handleFrozenExport(append([]string{"--frozen"}, cleanArgs[1:]...), stdout, stderr)
	case "import":
		return handleFrozenImport(append([]string{"--frozen"}, cleanArgs[1:]...), stdout, stderr)
	case "card":
		return handleCard(cleanArgs[1:], stdout, stderr)
	case "conformance":
		return handleConformance(cleanArgs[1:], stdout, stderr)
	case "readiness":
		return handleReadiness(cleanArgs[1:], stdout, stderr)
	case "release":
		return handleRelease(cleanArgs[1:], stdout, stderr)
	case "adapters":
		return handleAdapters(cleanArgs[1:], stdout, stderr)
	case "scan":
		return handleScan(cleanArgs[1:], stdout, stderr)
	case "contract":
		return handleContract(cleanArgs[1:], stdout, stderr)
	case "policy":
		return handlePolicy(cleanArgs[1:], stdout, stderr)
	case "explain":
		return handleExplain(cleanArgs[1:], stdout, stderr)
	case "boundary":
		return handleBoundary(cleanArgs[1:], stdout, stderr)
	case "profile":
		return handleProfile(cleanArgs[1:], stdout, stderr)
	case "repair":
		return handleRepair(cleanArgs[1:], stdout, stderr)
	case "uninstall":
		return handleUninstall(cleanArgs[1:], stdout, stderr)
	default:
		return handleStub(cleanArgs, stdout, stderr)
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

func printStartHere(w io.Writer, lang Lang) {
	WriteLine(w, "xh %s", Version)
	WriteLine(w, "")
	WriteLine(w, "A lightweight verify-gated harness for AI-agent workflows.")
	WriteLine(w, "")
	WriteLine(w, "%s", startHereTitle(lang))
	WriteLine(w, "")
	WriteLine(w, "%s", categoryGettingStarted(lang))
	for _, command := range onboardingCommands() {
		WriteLine(w, "  %-18s %s", command.Name, beginnerCommandDesc(command.Name, lang))
	}
	WriteLine(w, "")
	WriteLine(w, "%s", discoverMore(lang))
	WriteLine(w, "  xh --help            %s", discoverHelpDesc(lang))
	WriteLine(w, "  xh --help-all        %s", discoverHelpAllDesc(lang))
	WriteLine(w, "  xh --help-maturity   %s", discoverHelpMaturityDesc(lang))
	WriteLine(w, "")
	WriteLine(w, "%s", newToXHarness(lang))
}

func printHelp(w io.Writer, lang Lang) {
	WriteLine(w, "xh %s", Version)
	WriteLine(w, "")
	WriteLine(w, "A lightweight verify-gated harness for AI-agent workflows.")
	WriteLine(w, "")
	WriteLine(w, "%s", usageLabel(lang))
	WriteLine(w, "  xh <command> [options]")
	WriteLine(w, "")
	WriteLine(w, "%s", categoryGettingStarted(lang))
	for _, command := range onboardingCommands() {
		WriteLine(w, "  %-18s %s", command.Name, beginnerCommandDesc(command.Name, lang))
	}
	WriteLine(w, "")
	WriteLine(w, "%s", forCommandSpecificHelp(lang))
	WriteLine(w, "  xh <command> --help")
	WriteLine(w, "")
	WriteLine(w, "%s", advancedLabel(lang))
	WriteLine(w, "  xh --help-all          %s", showAllCommandsText(lang))
	WriteLine(w, "  xh --help-maturity     %s", showMaturityLabelsText(lang))
	WriteLine(w, "")
	WriteLine(w, "%s", globalOptionsLabel(lang))
	WriteLine(w, "  -h, --help          %s", showHelpText(lang))
	WriteLine(w, "  --help-all          %s", showAllCommandsText(lang))
	WriteLine(w, "  --help-maturity     %s", showMaturityLabelsText(lang))
	WriteLine(w, "  -v, --version       %s", showVersionText(lang))
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

func printActions(w io.Writer, lang Lang) {
	WriteLine(w, "%s", beginnerActionsTitle(lang))
	WriteLine(w, "")
	WriteLine(w, "%s", invokeUsingEither(lang))
	WriteLine(w, "  - %s  xh <action>", installedCLIText(lang))
	WriteLine(w, "  - %s  go run ./cmd/x-harness <action>", localSourceText(lang))
	WriteLine(w, "")
	WriteLine(w, "## %s", categoryGettingStarted(lang))
	WriteLine(w, "| %s | %s |", actionHeader(lang), descriptionHeader(lang))
	WriteLine(w, "| :-- | :-- |")
	for _, command := range onboardingCommands() {
		WriteLine(w, "| **%s** | %s |", command.Name, beginnerCommandDesc(command.Name, lang))
	}
	WriteLine(w, "")
	WriteLine(w, "%s", forMoreInfoText(lang))
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
	printHelp(&builder, LangEN)
	return builder.String()
}

func VersionText() string {
	return fmt.Sprintf("xh %s\n", Version)
}
