package cli

import (
	"crypto/sha256"
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
		fmt.Fprintln(stderr, "usage: x-harness adapters <matrix|eval|doctor>")
		return ExitUsage
	}

	switch args[0] {
	case "matrix":
		return handleAdaptersMatrix(args[1:], stdout, stderr)
	case "eval":
		return handleAdaptersEval(args[1:], stdout, stderr)
	case "doctor":
		return handleAdaptersDoctor(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown adapters subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness adapters <matrix|eval|doctor>")
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

type adapterDoctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

type adapterDoctorResult struct {
	Path   string              `json:"path"`
	OK     bool                `json:"ok"`
	Checks []adapterDoctorCheck `json:"checks"`
}

type managedBlock struct {
	ID        string
	BodyLines []string
}

func handleAdaptersDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--json" {
			jsonMode = true
		}
	}

	root := findAdaptersRepoRoot()
	if root == "" {
		fmt.Fprintln(stderr, "could not find repository root")
		return ExitError
	}

	results, ok := runAdaptersDoctor(root)

	passCount := 0
	for _, r := range results {
		if r.OK {
			passCount++
		}
	}

	if jsonMode {
		output := map[string]any{
			"adapters":    results,
			"pass_count":  passCount,
			"total_files": len(results),
		}
		if err := WriteJSON(stdout, output); err != nil {
			fmt.Fprintf(stderr, "failed to write JSON: %v\n", err)
			return ExitError
		}
		if !ok {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "# Adapter Doctor")
	WriteLine(stdout, "")
	for _, r := range results {
		status := "ok"
		if !r.OK {
			status = "fail"
		}
		WriteLine(stdout, "%s: %s", r.Path, status)
		for _, c := range r.Checks {
			if c.Note != "" {
				WriteLine(stdout, "  %s: %s (%s)", c.Name, c.Status, c.Note)
			} else {
				WriteLine(stdout, "  %s: %s", c.Name, c.Status)
			}
		}
	}
	WriteLine(stdout, "")
	WriteLine(stdout, "pass: %d/%d", passCount, len(results))

	if !ok {
		return ExitError
	}
	return ExitOK
}

func runAdaptersDoctor(root string) ([]adapterDoctorResult, bool) {
	adaptersDir := filepath.Join(root, "adapters")
	var results []adapterDoctorResult
	overallOK := true

	_ = filepath.Walk(adaptersDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		content, err := os.ReadFile(path)
		if err != nil {
			results = append(results, adapterDoctorResult{
				Path: rel,
				OK:   false,
				Checks: []adapterDoctorCheck{
					{Name: "readable", Status: "failed", Note: err.Error()},
				},
			})
			overallOK = false
			return nil
		}

		blocks := findManagedBlocks(string(content))
		if len(blocks) == 0 {
			return nil
		}

		result := adapterDoctorResult{
			Path:   rel,
			OK:     true,
			Checks: []adapterDoctorCheck{},
		}

		for _, block := range blocks {
			check := validateAdapterManagedBlock(block)
			result.Checks = append(result.Checks, check)
			if check.Status != "passed" {
				result.OK = false
				overallOK = false
			}
		}

		results = append(results, result)
		return nil
	})

	return results, overallOK
}

func findManagedBlocks(content string) []managedBlock {
	var blocks []managedBlock
	var current *managedBlock
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- BEGIN X-HARNESS MANAGED CONTRACT:") {
			id := extractMarkerID(trimmed, "<!-- BEGIN X-HARNESS MANAGED CONTRACT:", "-->")
			current = &managedBlock{
				ID:        id,
				BodyLines: []string{},
			}
		} else if strings.HasPrefix(trimmed, "<!-- END X-HARNESS MANAGED CONTRACT:") && current != nil {
			id := extractMarkerID(trimmed, "<!-- END X-HARNESS MANAGED CONTRACT:", "-->")
			if id == current.ID {
				blocks = append(blocks, *current)
				current = nil
			}
		} else if current != nil {
			current.BodyLines = append(current.BodyLines, line)
		}
	}

	return blocks
}

func extractMarkerID(trimmed, prefix, suffix string) string {
	s := strings.TrimPrefix(trimmed, prefix)
	s = strings.TrimSuffix(s, suffix)
	return strings.TrimSpace(s)
}

func validateAdapterManagedBlock(block managedBlock) adapterDoctorCheck {
	hash := ""
	for _, line := range block.BodyLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- contract-hash:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				hash = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "-->"))
			}
		}
	}

	if hash == "" {
		return adapterDoctorCheck{
			Name:   "managed_block_" + block.ID,
			Status: "failed",
			Note:   "missing contract-hash",
		}
	}

	body := extractBodyForHash(block)
	expectedHash := computeContractHash(body)

	if hash != expectedHash {
		return adapterDoctorCheck{
			Name:   "managed_block_" + block.ID,
			Status: "failed",
			Note:   fmt.Sprintf("hash mismatch: expected %s, found %s", expectedHash, hash),
		}
	}

	return adapterDoctorCheck{
		Name:   "managed_block_" + block.ID,
		Status: "passed",
	}
}

func extractBodyForHash(block managedBlock) string {
	var filtered []string
	for _, line := range block.BodyLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func computeContractHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)[:16]
}
