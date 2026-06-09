package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/intake"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"gopkg.in/yaml.v3"
)

func handleIntake(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "intake requires a subcommand: classify, explain, contract, handoff")
		return ExitUsage
	}

	switch args[0] {
	case "classify":
		return handleIntakeClassify(args[1:], stdout, stderr)
	case "explain":
		return handleIntakeExplain(args[1:], stdout, stderr)
	case "contract":
		return handleIntakeContract(args[1:], stdout, stderr)
	case "handoff":
		return handleIntakeHandoff(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: x-harness intake <classify|explain|contract|handoff> [options]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown intake subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness intake <classify|explain|contract|handoff> [options]")
		return ExitUsage
	}
}

func handleIntakeClassify(args []string, stdout, stderr io.Writer) int {
	task := "unknown"
	filesCSV := ""
	change := ""
	root := "."
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--task":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --task requires a value")
				return ExitUsage
			}
			task = args[i+1]
			i++
		case "--files":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --files requires a value")
				return ExitUsage
			}
			filesCSV = args[i+1]
			i++
		case "--change":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --change requires a value")
				return ExitUsage
			}
			change = args[i+1]
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

	root, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	policy, err := intake.LoadIntakePolicy(root)
	if err != nil {
		fmt.Fprintln(stderr, "Error: policies/intake.yaml not found")
		return ExitUsage
	}

	var files []string
	if filesCSV != "" {
		for _, f := range strings.Split(filesCSV, ",") {
			files = append(files, strings.TrimSpace(f))
		}
	}

	result := intake.ClassifyTask(task, files, change, policy)

	if jsonMode {
		data, _ := json.MarshalIndent(map[string]any{
			"intake_label":                result.IntakeLabel,
			"runtime_tier":                result.RuntimeTier,
			"task":                        task,
			"files":                       files,
			"change":                      change,
			"reasoning":                   result.Reasoning,
			"signals":                     result.Signals,
			"negative_signals_considered": result.NegativeSignalsConsidered,
			"auto_escalated":              result.AutoEscalated,
			"policy_valid":                true,
		}, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Task: %s\n", task)
		fmt.Fprintf(stdout, "Files: %s\n", strings.Join(files, ", "))
		if change != "" {
			fmt.Fprintf(stdout, "Change type: %s\n", change)
		}
		fmt.Fprintln(stdout)
		fmt.Fprintf(stdout, "Intake label: %s\n", result.IntakeLabel)
		fmt.Fprintf(stdout, "Runtime tier: %s\n", result.RuntimeTier)
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Reasoning:")
		for _, r := range result.Reasoning {
			fmt.Fprintf(stdout, "  - %s\n", r)
		}
	}

	return ExitOK
}

func handleIntakeExplain(args []string, stdout, stderr io.Writer) int {
	cardPath := ""
	root := "."
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --card requires a value")
				return ExitUsage
			}
			cardPath = args[i+1]
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

	if cardPath == "" {
		fmt.Fprintln(stderr, "error: --card <path> is required")
		return ExitUsage
	}

	root, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	policy, err := intake.LoadIntakePolicy(root)
	if err != nil {
		fmt.Fprintln(stderr, "Error: policies/intake.yaml not found")
		return ExitUsage
	}

	absCardPath := cardPath
	if !filepath.IsAbs(cardPath) {
		absCardPath = filepath.Join(root, cardPath)
	}

	var card map[string]any
	if err := loader.LoadDocument(absCardPath, &card); err != nil {
		fmt.Fprintf(stderr, "Error loading card: %v\n", err)
		return ExitError
	}

	explanation := intake.ExplainCardIntake(card, policy)

	if jsonMode {
		data, _ := json.MarshalIndent(explanation, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		rel, _ := filepath.Rel(root, absCardPath)
		if rel == "" {
			rel = absCardPath
		}
		fmt.Fprintf(stdout, "Card: %s\n", rel)
		fmt.Fprintf(stdout, "Source: %s\n", explanation.Source)
		if explanation.DeclaredTier != nil {
			fmt.Fprintf(stdout, "Declared tier: %s\n", *explanation.DeclaredTier)
		} else {
			fmt.Fprintln(stdout, "Declared tier: (missing)")
		}
		fmt.Fprintf(stdout, "Intake label: %s\n", explanation.IntakeLabel)
		fmt.Fprintf(stdout, "Mapped tier: %s\n", explanation.MappedTier)
		fmt.Fprintf(stdout, "Tier downgrade: %s\n", map[bool]string{true: "yes", false: "no"}[explanation.TierDowngrade])
		if explanation.InterventionRequired {
			fmt.Fprintf(stdout, "Intervention approved: %s\n", map[bool]string{true: "yes", false: "no"}[explanation.InterventionApproved])
		}
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Reasoning:")
		for _, r := range explanation.Reasoning {
			fmt.Fprintf(stdout, "  - %s\n", r)
		}
		if len(explanation.Warnings) > 0 {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "Warnings:")
			for _, w := range explanation.Warnings {
				fmt.Fprintf(stdout, "  - %s\n", w)
			}
		}
		if len(explanation.Errors) > 0 {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, "Errors:")
			for _, e := range explanation.Errors {
				fmt.Fprintf(stdout, "  - %s\n", e)
			}
		}
	}

	if !explanation.OK {
		return ExitError
	}
	return ExitOK
}

// productIntentSchemaVersion is the schema_version emitted by
// `xh intake contract` for safe V1 product intent records. The version is
// fixed for the first slice to keep the contract deterministic.
const productIntentSchemaVersion = "1"

// productIntentSpec captures the structured-flag input to
// `xh intake contract` before it is normalized into the product intent
// record. The CLI accepts repeatable and comma-delimited values for the
// list-shaped fields so users can express them either way.
type productIntentSpec struct {
	ID                 string
	ProductGoal        string
	UserVisibleChange  *bool
	NonGoals           []string
	Acceptance         []string
	ProtectedBehavior  []string
	AmbiguityStatus    string
	AmbiguityQuestions []string
	Notes              string
}

func handleIntakeContract(args []string, stdout, stderr io.Writer) int {
	spec := productIntentSpec{
		AmbiguityStatus: "none",
	}
	outputPath := ""
	jsonMode := false
	fromPath := ""
	ambiguitySet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--id":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --id requires a value")
				return ExitUsage
			}
			spec.ID = args[i+1]
			i++
		case "--goal":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --goal requires a value")
				return ExitUsage
			}
			spec.ProductGoal = args[i+1]
			i++
		case "--visible":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --visible requires a value (true|false)")
				return ExitUsage
			}
			val, err := parseBoolStrict(args[i+1])
			if err != nil {
				fmt.Fprintf(stderr, "error: --visible %v\n", err)
				return ExitUsage
			}
			spec.UserVisibleChange = &val
			i++
		case "--non-goal":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --non-goal requires a value")
				return ExitUsage
			}
			spec.NonGoals = appendList(spec.NonGoals, args[i+1])
			i++
		case "--acceptance":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --acceptance requires a value")
				return ExitUsage
			}
			spec.Acceptance = appendList(spec.Acceptance, args[i+1])
			i++
		case "--protected-behavior":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --protected-behavior requires a value")
				return ExitUsage
			}
			spec.ProtectedBehavior = appendList(spec.ProtectedBehavior, args[i+1])
			i++
		case "--ambiguity":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --ambiguity requires a value (none|unresolved|partial)")
				return ExitUsage
			}
			spec.AmbiguityStatus = args[i+1]
			ambiguitySet = true
			i++
		case "--ambiguity-question":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --ambiguity-question requires a value")
				return ExitUsage
			}
			spec.AmbiguityQuestions = appendList(spec.AmbiguityQuestions, args[i+1])
			i++
		case "--note":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --note requires a value")
				return ExitUsage
			}
			spec.Notes = args[i+1]
			i++
		case "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --from requires a value")
				return ExitUsage
			}
			fromPath = args[i+1]
			i++
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --output requires a value")
				return ExitUsage
			}
			outputPath = args[i+1]
			i++
		case "--json":
			jsonMode = true
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh intake contract [--id <id>] [--goal <text>] [--visible true|false] [--non-goal <text> ...] [--acceptance <text> ...] [--protected-behavior <text> ...] [--ambiguity none|unresolved|partial] [--ambiguity-question <text> ...] [--note <text>] [--from <markdown-path>] [--output <path>] [--json]")
			return ExitUsage
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", arg)
			return ExitUsage
		}
	}

	if fromPath != "" {
		if hasContentFlags(spec, ambiguitySet) {
			fmt.Fprintln(stderr, "error: --from is mutually exclusive with --id/--goal/--visible/--non-goal/--acceptance/--protected-behavior/--ambiguity/--ambiguity-question/--note")
			return ExitUsage
		}
		data, err := os.ReadFile(fromPath)
		if err != nil {
			fmt.Fprintf(stderr, "error: --from %v\n", err)
			return ExitError
		}
		mdSpec, err := intake.ParseMarkdown(string(data))
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitUsage
		}
		spec.ID = mdSpec.ID
		spec.ProductGoal = mdSpec.ProductGoal
		spec.UserVisibleChange = mdSpec.UserVisibleChange
		spec.NonGoals = mdSpec.NonGoals
		spec.Acceptance = mdSpec.Acceptance
		spec.ProtectedBehavior = mdSpec.ProtectedBehaviors
		if mdSpec.AmbiguitySet {
			spec.AmbiguityStatus = "partial"
		}
		spec.AmbiguityQuestions = mdSpec.AmbiguityQuestions
		spec.Notes = mdSpec.Notes
	}

	ambiguityStatus, err := normalizeAmbiguityStatus(spec.AmbiguityStatus)
	if err != nil {
		fmt.Fprintf(stderr, "error: --ambiguity %v\n", err)
		return ExitUsage
	}
	spec.AmbiguityStatus = ambiguityStatus

	record, err := buildProductIntentRecord(spec)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsage
	}

	if jsonMode {
		data, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
		output := append(data, '\n')
		if outputPath != "" {
			if err := writeIntentContractOutput(outputPath, output); err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				return ExitError
			}
		} else {
			_, _ = stdout.Write(output)
		}
		return ExitOK
	}

	out, err := yaml.Marshal(record)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}
	if outputPath != "" {
		if err := writeIntentContractOutput(outputPath, out); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
	} else {
		_, _ = stdout.Write(out)
	}
	return ExitOK
}

// buildProductIntentRecord converts a structured spec into a map matching
// schemas/product-intent.schema.json (safe V1). Required fields:
// schema_version, id, product_goal, acceptance_criteria. Optional fields
// that are unset are emitted as null/empty values for stable round-trip.
func buildProductIntentRecord(spec productIntentSpec) (map[string]any, error) {
	if strings.TrimSpace(spec.ID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	if strings.TrimSpace(spec.ProductGoal) == "" {
		return nil, fmt.Errorf("--goal is required")
	}
	if len(spec.Acceptance) == 0 {
		return nil, fmt.Errorf("at least one --acceptance is required (when using --from, include an ## Acceptance section with at least one item)")
	}
	for i, item := range spec.Acceptance {
		if strings.TrimSpace(item) == "" {
			return nil, fmt.Errorf("--acceptance entry %d is blank", i+1)
		}
	}

	acceptance := make([]any, 0, len(spec.Acceptance))
	for i, item := range spec.Acceptance {
		acceptance = append(acceptance, map[string]any{
			"id":         fmt.Sprintf("ac-%d", i+1),
			"statement":  item,
			"source_ref": "",
		})
	}

	ambiguity := map[string]any{
		"status":    spec.AmbiguityStatus,
		"questions": toAnySlice(spec.AmbiguityQuestions),
	}

	record := map[string]any{
		"schema_version": productIntentSchemaVersion,
		"id":             spec.ID,
		"product_goal":   spec.ProductGoal,
	}
	// user_visible_change is optional. The schema accepts an explicit
	// false as a valid non-user-visible declaration. Omit the key when
	// the flag was not provided so callers can distinguish "not set"
	// from "explicit false".
	if spec.UserVisibleChange != nil {
		record["user_visible_change"] = *spec.UserVisibleChange
	} else {
		record["user_visible_change"] = nil
	}
	record["non_goals"] = toAnySlice(spec.NonGoals)
	record["acceptance_criteria"] = acceptance
	record["protected_behaviors"] = toAnySlice(spec.ProtectedBehavior)
	record["ambiguity"] = ambiguity
	record["notes"] = spec.Notes
	return record, nil
}

func toAnySlice(in []string) []any {
	if len(in) == 0 {
		return nil
	}
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

func appendList(dst []string, raw string) []string {
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		dst = append(dst, item)
	}
	return dst
}

func parseBoolStrict(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	}
	return false, fmt.Errorf("expected true or false, got %q", raw)
}

// hasContentFlags reports whether any content flag was provided
// alongside a `--from` markdown path. --ambiguity defaults to "none"
// in the spec, so the caller tracks the explicit-set case via
// ambiguitySet.
func hasContentFlags(spec productIntentSpec, ambiguitySet bool) bool {
	return spec.ID != "" ||
		spec.ProductGoal != "" ||
		spec.UserVisibleChange != nil ||
		len(spec.NonGoals) > 0 ||
		len(spec.Acceptance) > 0 ||
		len(spec.ProtectedBehavior) > 0 ||
		ambiguitySet ||
		len(spec.AmbiguityQuestions) > 0 ||
		spec.Notes != ""
}

func normalizeAmbiguityStatus(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "none":
		return "none", nil
	case "unresolved":
		return "unresolved", nil
	case "partial":
		return "partial", nil
	}
	return "", fmt.Errorf("expected none, unresolved, or partial, got %q", raw)
}

// writeIntentContractOutput writes the rendered product intent record to
// disk. To keep the slice safe V1, the parent directory must already
// exist; we do not auto-create intermediate directories because that
// would hide typos and is unnecessary for the typical use case where the
// user writes into a known workspace path.
func writeIntentContractOutput(path string, data []byte) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	parent := filepath.Dir(abs)
	if _, err := os.Stat(parent); err != nil {
		return fmt.Errorf("parent directory does not exist: %s", parent)
	}
	return os.WriteFile(abs, data, 0644)
}

// handleIntakeHandoff is the entry point for `xh intake handoff`. Safe V1
// supports a single --tier auto subcommand that uses the existing
// intake classifier to pick light/standard/deep and prints a minimal
// handoff suggestion. This avoids duplicating the canonical handoff
// generator internals (see internal/cli/handoff.go) which produce full
// SUBAGENT_TASK prompts for explicit tiers.
func handleIntakeHandoff(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: xh intake handoff --tier auto [--task <text>] [--file <path> ...] [--root <path>] [--json]")
		return ExitUsage
	}

	tierArg := ""
	task := "unknown"
	var files []string
	root := "."
	jsonMode := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--tier":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --tier requires a value (auto|light|standard|deep)")
				return ExitUsage
			}
			tierArg = args[i+1]
			i++
		case "--task":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --task requires a value")
				return ExitUsage
			}
			task = args[i+1]
			i++
		case "--file":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --file requires a value")
				return ExitUsage
			}
			files = appendList(files, args[i+1])
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
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh intake handoff --tier auto [--task <text>] [--file <path> ...] [--root <path>] [--json]")
			return ExitUsage
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", arg)
			return ExitUsage
		}
	}

	// Safe V1 only exposes --tier auto. The handoff generator already
	// supports explicit light/standard/deep via `xh handoff <tier>`, so
	// re-adding them here would duplicate that surface and risk drift.
	if tierArg == "" {
		fmt.Fprintln(stderr, "error: --tier is required (safe V1 supports only --tier auto)")
		return ExitUsage
	}
	if tierArg != "auto" {
		fmt.Fprintf(stderr, "error: --tier %q is not supported in safe V1; use `xh handoff %s` for explicit tiers, or pass --tier auto\n", tierArg, tierArg)
		return ExitUsage
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	policy, err := intake.LoadIntakePolicy(absRoot)
	if err != nil {
		fmt.Fprintln(stderr, "Error: policies/intake.yaml not found")
		return ExitUsage
	}

	classification := intake.ClassifyTask(task, files, "", policy)

	result := map[string]any{
		"selected_tier":      string(classification.RuntimeTier),
		"intake_label":       string(classification.IntakeLabel),
		"task":               task,
		"files":              files,
		"signals":            classification.Signals,
		"reasoning":          classification.Reasoning,
		"auto_escalated":     classification.AutoEscalated,
		"command_suggestion": fmt.Sprintf("xh handoff %s --task %q", classification.RuntimeTier, task),
	}

	if jsonMode {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
		fmt.Fprintln(stdout, string(data))
		return ExitOK
	}

	fmt.Fprintf(stdout, "Task: %s\n", task)
	if len(files) > 0 {
		fmt.Fprintf(stdout, "Files: %s\n", strings.Join(files, ", "))
	}
	fmt.Fprintf(stdout, "Selected tier: %s\n", classification.RuntimeTier)
	fmt.Fprintf(stdout, "Intake label: %s\n", classification.IntakeLabel)
	if classification.AutoEscalated {
		fmt.Fprintln(stdout, "Auto escalated: yes")
	}
	fmt.Fprintln(stdout, "Reasoning:")
	for _, r := range classification.Reasoning {
		fmt.Fprintf(stdout, "  - %s\n", r)
	}
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Suggested next: %s\n", result["command_suggestion"])
	return ExitOK
}
