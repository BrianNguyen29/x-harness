package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/contextcheck"
)

// ContractFact is a single canonical contract fact.
type ContractFact struct {
	Rule        string `json:"rule"`
	Description string `json:"description"`
}

// Contract holds the canonical x-harness contract facts.
type Contract struct {
	Facts []ContractFact `json:"facts"`
}

// CoreContract returns the canonical contract derived from repository assets.
func CoreContract() Contract {
	return Contract{
		Facts: []ContractFact{
			{
				Rule:        "completion_admitted_not_claimed",
				Description: "Completion is admitted, not claimed. Agents may propose completion but cannot self-admit.",
			},
			{
				Rule:        "verifier_read_only",
				Description: "The verifier is read-only. It must not edit source files or repair the work product while verifying.",
			},
			{
				Rule:        "success_only_accepted",
				Description: "Success is the only accepted outcome. admission.outcome: success and acceptance_status: accepted are required.",
			},
			{
				Rule:        "canonical_tiers",
				Description: "Canonical tiers are light, standard, and deep. Do not use small, medium, or large in active runtime handoffs.",
			},
			{
				Rule:        "pgv_advisory_only",
				Description: "PGV is advisory-only. It never overrides verify and never grants admission authority by default.",
			},
		},
	}
}

func generateManagedBlock() string {
	ctx := contextcheck.CanonicalContext()
	hash := contextcheck.ContextHash(ctx)
	return strings.Join([]string{
		contextcheck.ManagedBegin,
		"<!-- generated-by: x-harness -->",
		fmt.Sprintf("<!-- context-hash: %s -->", hash),
		"",
		ctx,
		"",
		contextcheck.ManagedEnd,
	}, "\n")
}

func injectManagedBlock(content, block string) string {
	beginIndex := strings.Index(content, contextcheck.ManagedBegin)
	endIndex := strings.Index(content, contextcheck.ManagedEnd)
	if beginIndex != -1 && endIndex != -1 && endIndex > beginIndex {
		before := content[:beginIndex]
		after := content[endIndex+len(contextcheck.ManagedEnd):]
		return before + block + after
	}
	separator := "\n\n"
	if strings.HasSuffix(content, "\n") {
		separator = ""
	}
	return content + separator + block + "\n"
}

func runContext(args []string, stdout io.Writer, _ io.Writer) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
		}
	}

	contract := CoreContract()

	if jsonMode {
		if err := WriteJSON(stdout, contract); err != nil {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "x-harness Canonical Contract")
	WriteLine(stdout, "")
	for _, fact := range contract.Facts {
		WriteLine(stdout, "- %s", strings.ReplaceAll(fact.Description, "\n", "\n  "))
	}
	return ExitOK
}

func runContextSync(args []string, stdout io.Writer, stderr io.Writer) int {
	checkMode := false
	writeMode := false
	jsonMode := false
	root := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--check":
			checkMode = true
		case "--write":
			writeMode = true
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		}
	}

	if !checkMode && !writeMode {
		fmt.Fprintln(stderr, "usage: x-harness context sync --check|--write [--root <path>] [--json]")
		return ExitUsage
	}

	if root == "" {
		root = "."
	}
	agentsPath := filepath.Join(root, "AGENTS.md")

	agentsContentBytes, err := os.ReadFile(agentsPath)
	if err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{
				"valid": false,
				"note":  fmt.Sprintf("AGENTS.md not found at %s", agentsPath),
			})
		} else {
			fmt.Fprintf(stderr, "Error: AGENTS.md not found at %s\n", agentsPath)
		}
		return ExitUsage
	}
	agentsContent := string(agentsContentBytes)

	if checkMode {
		valid, note := contextcheck.ValidateManagedBlock(agentsContent)
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{
				"valid": valid,
				"note":  note,
			})
		} else {
			if valid {
				fmt.Fprintln(stdout, "✓ AGENTS.md managed context block is valid")
			} else {
				fmt.Fprintf(stderr, "✗ %s\n", note)
			}
		}
		if valid {
			return ExitOK
		}
		return ExitError
	}

	if writeMode {
		block := generateManagedBlock()
		updated := injectManagedBlock(agentsContent, block)
		if err := os.WriteFile(agentsPath, []byte(updated), 0644); err != nil {
			fmt.Fprintf(stderr, "Error: failed to write AGENTS.md: %v\n", err)
			return ExitError
		}
		hashMatch := strings.Index(block, "<!-- context-hash: ")
		var hash string
		if hashMatch != -1 {
			hashStart := hashMatch + len("<!-- context-hash: ")
			hashEnd := strings.Index(block[hashStart:], " -->")
			if hashEnd != -1 {
				hash = block[hashStart : hashStart+hashEnd]
			}
		}
		if hash == "" {
			hash = "unknown"
		}
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{
				"updated":      true,
				"context_hash": hash,
			})
		} else {
			fmt.Fprintf(stdout, "AGENTS.md refreshed (context-hash: %s)\n", hash)
		}
		return ExitOK
	}

	return ExitUsage
}

func runContextGC(args []string, stdout io.Writer, stderr io.Writer) int {
	checkMode := false
	writeMode := false
	jsonMode := false
	root := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--check":
			checkMode = true
		case "--write":
			writeMode = true
		case "--json":
			jsonMode = true
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		}
	}

	if !checkMode && !writeMode {
		fmt.Fprintln(stderr, "usage: x-harness context gc --check|--write [--root <path>] [--json]")
		return ExitUsage
	}

	if root == "" {
		root = "."
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	agentsContentBytes, err := os.ReadFile(agentsPath)
	if err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{
				"ok":   false,
				"note": fmt.Sprintf("AGENTS.md not found at %s", agentsPath),
			})
		} else {
			fmt.Fprintf(stderr, "Error: AGENTS.md not found at %s\n", agentsPath)
		}
		return ExitUsage
	}
	agentsContent := string(agentsContentBytes)

	valid, note := contextcheck.ValidateManagedBlock(agentsContent)

	if checkMode {
		if jsonMode {
			output := map[string]any{
				"ok":       valid,
				"findings": []string{},
			}
			if !valid {
				output["findings"] = []string{note}
			}
			if err := WriteJSON(stdout, output); err != nil {
				return ExitError
			}
		} else {
			if valid {
				fmt.Fprintln(stdout, "✓ Context GC check passed")
			} else {
				fmt.Fprintln(stderr, "✗ Context GC check failed")
				fmt.Fprintf(stderr, "  - %s\n", note)
			}
		}
		if valid {
			return ExitOK
		}
		return ExitError
	}

	if writeMode {
		if valid {
			if jsonMode {
				_ = WriteJSON(stdout, map[string]any{
					"ok":      true,
					"changed": false,
					"note":    "AGENTS.md is already up-to-date",
				})
			} else {
				fmt.Fprintln(stdout, "✓ AGENTS.md is already up-to-date")
			}
			return ExitOK
		}

		block := generateManagedBlock()
		updated := injectManagedBlock(agentsContent, block)
		if err := os.WriteFile(agentsPath, []byte(updated), 0644); err != nil {
			fmt.Fprintf(stderr, "Error: failed to write AGENTS.md: %v\n", err)
			return ExitError
		}

		hashMatch := strings.Index(block, "<!-- context-hash: ")
		var hash string
		if hashMatch != -1 {
			hashStart := hashMatch + len("<!-- context-hash: ")
			hashEnd := strings.Index(block[hashStart:], " -->")
			if hashEnd != -1 {
				hash = block[hashStart : hashStart+hashEnd]
			}
		}
		if hash == "" {
			hash = "unknown"
		}

		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{
				"ok":           true,
				"changed":      true,
				"context_hash": hash,
				"findings":     []string{note},
			})
		} else {
			fmt.Fprintf(stdout, "AGENTS.md refreshed (context-hash: %s)\n", hash)
		}
		return ExitOK
	}

	return ExitUsage
}

func handleContext(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness context --contract [--json] | context sync --check|--write [--root <path>] [--json] | context gc --check|--write [--root <path>] [--json]")
		return ExitUsage
	}

	switch args[0] {
	case "--contract":
		return runContext(args[1:], stdout, stderr)
	case "sync":
		return runContextSync(args[1:], stdout, stderr)
	case "gc":
		return runContextGC(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown context subcommand: %s\n", args[0])
		return ExitUsage
	}
}
