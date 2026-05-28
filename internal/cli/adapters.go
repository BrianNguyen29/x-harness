package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type adapterInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	Formats      []string `json:"formats"`
}

var adapters = []adapterInfo{
	{
		Name:         "opencode",
		Description:  "OpenCode agent platform integration",
		Capabilities: []string{"prepare", "check", "recover", "doctor", "actions", "status", "reset"},
		Formats:      []string{"verify-agent", "orchestrator", "example-json"},
	},
	{
		Name:         "claude-code",
		Description:  "Claude Code agent platform integration",
		Capabilities: []string{"prepare", "check", "recover", "doctor", "actions", "status", "reset"},
		Formats:      []string{"CLAUDE.md", "agents", "skills"},
	},
	{
		Name:         "cursor",
		Description:  "Cursor IDE agent platform integration",
		Capabilities: []string{"prepare", "check", "recover", "doctor", "actions", "status", "reset"},
		Formats:      []string{"rules"},
	},
	{
		Name:         "generic",
		Description:  "System-agnostic x-harness adapter",
		Capabilities: []string{"prepare", "check", "recover", "doctor", "actions", "status", "reset"},
		Formats:      []string{"AGENTS.md"},
	},
	{
		Name:         "antigravity",
		Description:  "Antigravity agent platform integration",
		Capabilities: []string{"prepare", "check", "recover", "doctor", "actions", "status", "reset"},
		Formats:      []string{"rules", "workflows"},
	},
}

func handleAdapters(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness adapters <matrix|eval>")
		return ExitUsage
	}

	switch args[0] {
	case "matrix":
		return handleAdaptersMatrix(args[1:], stdout, stderr)
	case "eval":
		return handleAdaptersEval(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown adapters subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness adapters <matrix|eval>")
		return ExitUsage
	}
}

func handleAdaptersMatrix(args []string, stdout io.Writer, _ io.Writer) int {
	jsonMode := false

	for i := 0; i < len(args); i++ {
		if args[i] == "--json" {
			jsonMode = true
		}
	}

	if jsonMode {
		result := map[string]any{"adapters": adapters}
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "# x-harness Adapter Matrix")
	WriteLine(stdout, "")
	WriteLine(stdout, "adapters: %d", len(adapters))
	WriteLine(stdout, "")
	WriteLine(stdout, "| Adapter | Description | Capabilities | Formats |")
	WriteLine(stdout, "| :-- | :-- | :-- | :-- |")
	for _, a := range adapters {
		caps := strings.Join(a.Capabilities, ", ")
		fmts := strings.Join(a.Formats, ", ")
		WriteLine(stdout, "| %s | %s | %s | %s |", a.Name, a.Description, caps, fmts)
	}
	return ExitOK
}

type adapterEvalResult struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	HasReadme  bool   `json:"has_readme"`
	HasCaps    bool   `json:"has_capabilities"`
	HasFormats bool   `json:"has_formats"`
}

func findAdaptersRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		for _, marker := range []string{".git", "go.mod", "X_HARNESS.md", "AGENTS.md"} {
			path := filepath.Join(wd, marker)
			if _, err := os.Stat(path); err == nil {
				return wd
			}
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return ""
}

func evalAdapter(a adapterInfo, root string) adapterEvalResult {
	readmePath := fmt.Sprintf("adapters/%s/README.md", a.Name)
	if root != "" {
		readmePath = filepath.Join(root, readmePath)
	}
	_, err := os.Stat(readmePath)
	hasReadme := err == nil
	return adapterEvalResult{
		Name:       a.Name,
		OK:         hasReadme && len(a.Capabilities) > 0 && len(a.Formats) > 0,
		HasReadme:  hasReadme,
		HasCaps:    len(a.Capabilities) > 0,
		HasFormats: len(a.Formats) > 0,
	}
}

func handleAdaptersEval(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--json" {
			jsonMode = true
		}
	}

	results := make([]adapterEvalResult, 0, len(adapters))
	passCount := 0
	root := findAdaptersRepoRoot()
	for _, a := range adapters {
		r := evalAdapter(a, root)
		results = append(results, r)
		if r.OK {
			passCount++
		}
	}

	if jsonMode {
		output := map[string]any{
			"adapters":   results,
			"pass_count": passCount,
			"total":      len(adapters),
		}
		if err := WriteJSON(stdout, output); err != nil {
			fmt.Fprintf(stderr, "failed to write JSON: %v\n", err)
			return ExitError
		}
		if passCount < len(adapters) {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "# Adapter Evaluation")
	WriteLine(stdout, "")
	for _, r := range results {
		status := "ok"
		if !r.OK {
			status = "fail"
		}
		WriteLine(stdout, "%s: %s", r.Name, status)
		if !r.HasReadme {
			WriteLine(stdout, "  missing README.md")
		}
		if !r.HasCaps {
			WriteLine(stdout, "  missing capabilities")
		}
		if !r.HasFormats {
			WriteLine(stdout, "  missing formats")
		}
	}
	WriteLine(stdout, "")
	WriteLine(stdout, "pass: %d/%d", passCount, len(adapters))

	if passCount < len(adapters) {
		return ExitError
	}
	return ExitOK
}
