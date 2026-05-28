package cli

import (
	"fmt"
	"io"
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
		fmt.Fprintln(stderr, "usage: x-harness adapters <matrix>")
		return ExitUsage
	}

	switch args[0] {
	case "matrix":
		return handleAdaptersMatrix(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown adapters subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness adapters <matrix>")
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
