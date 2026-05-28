package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	managedBegin = "<!-- BEGIN X-HARNESS MANAGED CONTEXT -->"
	managedEnd   = "<!-- END X-HARNESS MANAGED CONTEXT -->"
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

func canonicalContext() string {
	return `# x-harness Canonical Context

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

## Source-of-Truth Reading Order

The managed context block in AGENTS.md is authoritative. Files are read in this order:

1. AGENTS.md (managed block)
1. X_HARNESS.md
1. policies/admission.yaml
1. policies/recovery.yaml
1. policies/intake.yaml
1. schemas/completion-card.schema.json

## Rules

### Completion is admitted, not claimed
Agents may propose completion but cannot self-admit. A completion card with ` + "`" + `claim.fix_status: fixed` + "`" + ` is only a completion candidate. Compatibility subagent returns may use ` + "`" + `result.fix_status` + "`" + `.

### Verifier is read-only
The verifier may inspect files, evidence, diffs, and trace events. It must not edit source files or repair the work product while verifying.

### Success is the only accepted outcome
` + "`" + `admission.outcome: success` + "`" + ` and ` + "`" + `acceptance_status: accepted` + "`" + ` are required for admission. All other outcomes are withheld.

### Canonical tiers
Use only ` + "`" + `light` + "`" + `, ` + "`" + `standard` + "`" + `, and ` + "`" + `deep` + "`" + `. Do not use ` + "`" + `small` + "`" + `, ` + "`" + `medium` + "`" + `, or ` + "`" + `large` + "`" + ` in active runtime handoffs.

### PGV is advisory-only
Pre-gate validation (PGV) advice never overrides the verify gate and never grants admission authority by default.`
}

func contextHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])[:16]
}

func generateManagedBlock() string {
	ctx := canonicalContext()
	hash := contextHash(ctx)
	return strings.Join([]string{
		managedBegin,
		"<!-- generated-by: x-harness -->",
		fmt.Sprintf("<!-- context-hash: %s -->", hash),
		"",
		ctx,
		"",
		managedEnd,
	}, "\n")
}

func extractManagedBlock(content string) (string, bool) {
	beginIndex := strings.Index(content, managedBegin)
	endIndex := strings.Index(content, managedEnd)
	if beginIndex == -1 || endIndex == -1 || endIndex < beginIndex {
		return "", false
	}
	return content[beginIndex : endIndex+len(managedEnd)], true
}

func injectManagedBlock(content, block string) string {
	beginIndex := strings.Index(content, managedBegin)
	endIndex := strings.Index(content, managedEnd)
	if beginIndex != -1 && endIndex != -1 && endIndex > beginIndex {
		before := content[:beginIndex]
		after := content[endIndex+len(managedEnd):]
		return before + block + after
	}
	separator := "\n\n"
	if strings.HasSuffix(content, "\n") {
		separator = ""
	}
	return content + separator + block + "\n"
}

func validateManagedBlock(content string) (bool, string) {
	block, ok := extractManagedBlock(content)
	if !ok {
		return false, "AGENTS.md missing managed context block"
	}

	idx := strings.Index(block, "<!-- context-hash: ")
	if idx == -1 {
		return false, "AGENTS.md managed block missing context-hash"
	}
	hashStart := idx + len("<!-- context-hash: ")
	hashEnd := strings.Index(block[hashStart:], " -->")
	if hashEnd == -1 {
		return false, "AGENTS.md managed block missing context-hash"
	}
	actualHash := block[hashStart : hashStart+hashEnd]

	currentContext := canonicalContext()
	expectedHash := contextHash(currentContext)

	if actualHash != expectedHash {
		return false, fmt.Sprintf("AGENTS.md context hash stale: expected %s, found %s", expectedHash, actualHash)
	}

	lines := strings.Split(block, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == managedBegin || trimmed == managedEnd || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		filtered = append(filtered, line)
	}
	actualContext := strings.TrimSpace(strings.Join(filtered, "\n"))

	if actualContext != strings.TrimSpace(currentContext) {
		return false, "AGENTS.md managed context body differs from canonical context"
	}

	return true, "AGENTS.md managed context block is fresh"
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
		valid, note := validateManagedBlock(agentsContent)
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

	if writeMode {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{
				"ok":   false,
				"note": "context gc --write is planned but not yet implemented",
			})
		} else {
			fmt.Fprintln(stderr, "context gc --write is planned but not yet implemented")
		}
		return ExitUsage
	}

	if !checkMode {
		fmt.Fprintln(stderr, "usage: x-harness context gc --check [--root <path>] [--json]")
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

	valid, note := validateManagedBlock(string(agentsContentBytes))

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

func handleContext(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness context --contract [--json] | context sync --check|--write [--root <path>] [--json] | context gc --check [--root <path>] [--json]")
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
