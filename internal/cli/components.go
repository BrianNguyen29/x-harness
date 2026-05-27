package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/components"
)

func handleComponents(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "components requires a subcommand: validate, list, explain, changed")
		return ExitUsage
	}

	switch args[0] {
	case "validate":
		return handleComponentsValidate(args[1:], stdout, stderr)
	case "list":
		return handleComponentsList(args[1:], stdout, stderr)
	case "explain":
		return handleComponentsExplain(args[1:], stdout, stderr)
	case "changed":
		return handleComponentsChanged(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown components subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func handleComponentsValidate(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
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
	result, err := components.ValidateRegistry(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
		if !result.OK {
			return ExitError
		}
		return ExitOK
	}

	fmt.Fprintf(stdout, "ok: %v\n", result.OK)
	fmt.Fprintf(stdout, "components: %d\n", result.ComponentCount)
	fmt.Fprintf(stdout, "protected_paths: %d/%d covered\n", result.ProtectedPathsCovered, result.ProtectedPathsChecked)
	if len(result.Errors) > 0 {
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Errors:")
		for _, e := range result.Errors {
			fmt.Fprintf(stdout, "  - %s\n", e)
		}
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Warnings:")
		for _, w := range result.Warnings {
			fmt.Fprintf(stdout, "  - %s\n", w)
		}
	}
	if !result.OK {
		return ExitError
	}
	return ExitOK
}

func handleComponentsList(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
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
	reg, err := components.LoadRegistry(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(reg, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return ExitOK
	}

	fmt.Fprintln(stdout, "# x-harness Components")
	fmt.Fprintln(stdout, "")
	for _, c := range reg.Components {
		fmt.Fprintf(stdout, "- %s (%s, %s)\n", c.ID, c.Kind, c.Stability)
		fmt.Fprintf(stdout, "  owner: %s\n", c.Owner)
		fmt.Fprintf(stdout, "  agent_edit: %s\n", c.AgentEdit)
		fmt.Fprintf(stdout, "  paths: %s\n", strings.Join(c.Paths, ", "))
	}
	return ExitOK
}

func handleComponentsExplain(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	id := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--id":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --id requires a value")
				return ExitUsage
			}
			id = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	if id == "" {
		fmt.Fprintln(stderr, "Error: --id <component-id> is required")
		return ExitUsage
	}

	root, _ = filepath.Abs(root)
	reg, err := components.LoadRegistry(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	component := components.FindComponent(reg, id)
	if component == nil {
		fmt.Fprintf(stderr, "Error: component not found: %s\n", id)
		return ExitError
	}

	if jsonMode {
		data, _ := json.MarshalIndent(component, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return ExitOK
	}

	fmt.Fprintf(stdout, "Component: %s\n", component.ID)
	fmt.Fprintf(stdout, "Kind: %s\n", component.Kind)
	fmt.Fprintf(stdout, "Owner: %s\n", component.Owner)
	fmt.Fprintf(stdout, "Stability: %s\n", component.Stability)
	fmt.Fprintf(stdout, "Agent edit: %s\n", component.AgentEdit)
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Paths:")
	for _, p := range component.Paths {
		fmt.Fprintf(stdout, "  - %s\n", p)
	}
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Tests:")
	for _, t := range component.Tests {
		fmt.Fprintf(stdout, "  - %s\n", t)
	}
	return ExitOK
}

func handleComponentsChanged(args []string, stdout, stderr io.Writer) int {
	root := "."
	jsonMode := false
	base := "main"
	filesArg := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --root requires a value")
				return ExitUsage
			}
			root = args[i+1]
			i++
		case "--base":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --base requires a value")
				return ExitUsage
			}
			base = args[i+1]
			i++
		case "--files":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --files requires a value")
				return ExitUsage
			}
			filesArg = args[i+1]
			i++
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
	reg, err := components.LoadRegistry(root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	var files []string
	if filesArg != "" {
		for _, f := range strings.Split(filesArg, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				files = append(files, f)
			}
		}
	} else {
		files, err = components.ListChangedFilesFromGit(root, base)
		if err != nil {
			fmt.Fprintf(stderr, "Error reading changed files: %v\n", err)
			return ExitUsage
		}
	}

	matches, unregistered := components.ClassifyFiles(reg, files)

	if jsonMode {
		type changedComponent struct {
			ID        string   `json:"id"`
			Kind      string   `json:"kind"`
			Owner     string   `json:"owner"`
			Stability string   `json:"stability"`
			AgentEdit string   `json:"agent_edit"`
			Files     []string `json:"files"`
			Tests     []string `json:"tests"`
		}
		var comps []changedComponent
		for _, m := range matches {
			comps = append(comps, changedComponent{
				ID:        m.Component.ID,
				Kind:      m.Component.Kind,
				Owner:     m.Component.Owner,
				Stability: m.Component.Stability,
				AgentEdit: m.Component.AgentEdit,
				Files:     m.Files,
				Tests:     m.Component.Tests,
			})
		}
		result := map[string]interface{}{
			"base":               base,
			"files":              files,
			"components":         comps,
			"unregistered_files": unregistered,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return ExitOK
	}

	fmt.Fprintf(stdout, "Changed files: %d\n", len(files))
	fmt.Fprintf(stdout, "Components touched: %d\n", len(matches))
	for _, m := range matches {
		fmt.Fprintln(stdout, "")
		fmt.Fprintf(stdout, "- %s\n", m.Component.ID)
		for _, f := range m.Files {
			fmt.Fprintf(stdout, "  - %s\n", f)
		}
	}
	if len(unregistered) > 0 {
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Unregistered files:")
		for _, f := range unregistered {
			fmt.Fprintf(stdout, "  - %s\n", f)
		}
	}
	return ExitOK
}
