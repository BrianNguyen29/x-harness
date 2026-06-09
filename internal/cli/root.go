package cli

import (
	"fmt"
	"io"
	"strings"
)

var Version = "0.99.0-rc1"

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
	{Name: "decision", Description: "Record or list decision memory records (ADR-lite)", Maturity: MaturityExperimental},
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
	{Name: "start", Description: "Guided onboarding: doctor, examples verify, init wizard, next steps", Primary: true, Maturity: MaturityBeta},
	{Name: "learn", Description: "Read-only concept tour for beginners", Primary: true, Maturity: MaturityBeta},
	{Name: "quick", Description: "Read-only next-action recommender for newcomers", Primary: true, Maturity: MaturityBeta},
	{Name: "run", Description: "Run a built-in workflow recipe", Primary: true, Maturity: MaturityBeta},
	{Name: "ci", Description: "Run the built-in CI workflow", Primary: true, Maturity: MaturityBeta},
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
	{Name: "boundary", Description: "Lint/check/explain boundary policy against repo source files", Maturity: MaturityBeta},
}

func isBeginnerCommand(name string) bool {
	switch name {
	case "check", "prepare", "recover", "doctor", "actions", "status", "reset", "init", "add", "start", "learn", "quick", "run", "ci":
		return true
	}
	return false
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
	WriteLine(w, "  %-18s %s", "start", beginnerCommandDesc("start", lang))
	WriteLine(w, "  %-18s %s", "learn", beginnerCommandDesc("learn", lang))
	WriteLine(w, "  %-18s %s", "quick", beginnerCommandDesc("quick", lang))
	WriteLine(w, "  %-18s %s", "init", beginnerCommandDesc("init", lang))
	WriteLine(w, "")
	WriteLine(w, "%s", categoryDailyTasks(lang))
	WriteLine(w, "  %-18s %s", "check (verify)", beginnerCommandDesc("check", lang))
	WriteLine(w, "  %-18s %s", "actions", beginnerCommandDesc("actions", lang))
	WriteLine(w, "  %-18s %s", "status", beginnerCommandDesc("status", lang))
	WriteLine(w, "  %-18s %s", "add", beginnerCommandDesc("add", lang))
	WriteLine(w, "")
	WriteLine(w, "%s", categoryHealthRecovery(lang))
	WriteLine(w, "  %-18s %s", "doctor", beginnerCommandDesc("doctor", lang))
	WriteLine(w, "  %-18s %s", "recover", beginnerCommandDesc("recover", lang))
	WriteLine(w, "  %-18s %s", "reset", beginnerCommandDesc("reset", lang))
	WriteLine(w, "")
	WriteLine(w, "%s", categoryAutomation(lang))
	WriteLine(w, "  %-18s %s", "run", beginnerCommandDesc("run", lang))
	WriteLine(w, "  %-18s %s", "ci", beginnerCommandDesc("ci", lang))
	WriteLine(w, "  %-18s %s", "prepare", beginnerCommandDesc("prepare", lang))
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
	WriteLine(w, "  %-18s %s", "start", beginnerCommandDesc("start", lang))
	WriteLine(w, "  %-18s %s", "learn", beginnerCommandDesc("learn", lang))
	WriteLine(w, "  %-18s %s", "quick", beginnerCommandDesc("quick", lang))
	WriteLine(w, "  %-18s %s", "init", beginnerCommandDesc("init", lang))
	WriteLine(w, "")
	WriteLine(w, "%s", categoryDailyTasks(lang))
	WriteLine(w, "  %-18s %s", "check (verify)", beginnerCommandDesc("check", lang))
	WriteLine(w, "  %-18s %s", "actions", beginnerCommandDesc("actions", lang))
	WriteLine(w, "  %-18s %s", "status", beginnerCommandDesc("status", lang))
	WriteLine(w, "  %-18s %s", "add", beginnerCommandDesc("add", lang))
	WriteLine(w, "")
	WriteLine(w, "%s", categoryHealthRecovery(lang))
	WriteLine(w, "  %-18s %s", "doctor", beginnerCommandDesc("doctor", lang))
	WriteLine(w, "  %-18s %s", "recover", beginnerCommandDesc("recover", lang))
	WriteLine(w, "  %-18s %s", "reset", beginnerCommandDesc("reset", lang))
	WriteLine(w, "")
	WriteLine(w, "%s", categoryAutomation(lang))
	WriteLine(w, "  %-18s %s", "run", beginnerCommandDesc("run", lang))
	WriteLine(w, "  %-18s %s", "ci", beginnerCommandDesc("ci", lang))
	WriteLine(w, "  %-18s %s", "prepare", beginnerCommandDesc("prepare", lang))
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
	WriteLine(w, "| **start** | %s |", beginnerCommandDesc("start", lang))
	WriteLine(w, "| **learn** | %s |", beginnerCommandDesc("learn", lang))
	WriteLine(w, "| **quick** | %s |", beginnerCommandDesc("quick", lang))
	WriteLine(w, "| **init** | %s |", beginnerCommandDesc("init", lang))
	WriteLine(w, "")
	WriteLine(w, "## %s", categoryDailyTasks(lang))
	WriteLine(w, "| %s | %s |", actionHeader(lang), descriptionHeader(lang))
	WriteLine(w, "| :-- | :-- |")
	WriteLine(w, "| **check** | %s |", beginnerCommandDesc("check", lang))
	WriteLine(w, "| **actions** | %s |", beginnerCommandDesc("actions", lang))
	WriteLine(w, "| **status** | %s |", beginnerCommandDesc("status", lang))
	WriteLine(w, "| **add** | %s |", beginnerCommandDesc("add", lang))
	WriteLine(w, "")
	WriteLine(w, "## %s", categoryHealthRecovery(lang))
	WriteLine(w, "| %s | %s |", actionHeader(lang), descriptionHeader(lang))
	WriteLine(w, "| :-- | :-- |")
	WriteLine(w, "| **doctor** | %s |", beginnerCommandDesc("doctor", lang))
	WriteLine(w, "| **recover** | %s |", beginnerCommandDesc("recover", lang))
	WriteLine(w, "| **reset** | %s |", beginnerCommandDesc("reset", lang))
	WriteLine(w, "")
	WriteLine(w, "## %s", categoryAutomation(lang))
	WriteLine(w, "| %s | %s |", actionHeader(lang), descriptionHeader(lang))
	WriteLine(w, "| :-- | :-- |")
	WriteLine(w, "| **run** | %s |", beginnerCommandDesc("run", lang))
	WriteLine(w, "| **ci** | %s |", beginnerCommandDesc("ci", lang))
	WriteLine(w, "| **prepare** | %s |", beginnerCommandDesc("prepare", lang))
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
