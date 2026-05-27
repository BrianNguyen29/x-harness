package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/intake"
	"github.com/BrianNguyen29/x-harness/internal/loader"
)

func handleIntake(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "intake requires a subcommand: classify, explain")
		return ExitUsage
	}

	switch args[0] {
	case "classify":
		return handleIntakeClassify(args[1:], stdout, stderr)
	case "explain":
		return handleIntakeExplain(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown intake subcommand: %s\n", args[0])
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
