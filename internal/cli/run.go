package cli

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/repo"
)

// RunStep represents a single step in a workflow run.
type RunStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

// RunResult is the JSON output shape for the run command.
type RunResult struct {
	Recipe string    `json:"recipe"`
	OK     bool      `json:"ok"`
	Steps  []RunStep `json:"steps"`
}

func handleRun(args []string, stdout io.Writer, stderr io.Writer) int {
	listMode := false
	dryRun := false
	jsonMode := false
	recipe := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh run [--list] [<recipe>] [--dry-run] [--json]")
			return ExitUsage
		case "--list":
			listMode = true
		case "--dry-run":
			dryRun = true
		case "--json":
			jsonMode = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				recipe = args[i]
			}
		}
	}

	if listMode {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{"recipes": []string{"builtin:ci"}})
		} else {
			WriteLine(stdout, "Available recipes:")
			WriteLine(stdout, "  builtin:ci")
		}
		return ExitOK
	}

	if recipe == "" {
		fmt.Fprintln(stderr, "usage: xh run [--list] [<recipe>] [--dry-run] [--json]")
		return ExitUsage
	}

	if recipe != "builtin:ci" {
		fmt.Fprintf(stderr, "unknown recipe: %s\n", recipe)
		fmt.Fprintln(stderr, "run `xh run --list` for available recipes")
		return ExitUsage
	}

	steps := builtinCISteps(dryRun, stdout, stderr)

	ok := true
	for _, s := range steps {
		if s.Status != "passed" && s.Status != "skipped" && s.Status != "planned" {
			ok = false
			break
		}
	}

	if jsonMode {
		result := RunResult{
			Recipe: recipe,
			OK:     ok,
			Steps:  steps,
		}
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		if dryRun {
			WriteLine(stdout, "# xh run %s --dry-run", recipe)
		} else {
			WriteLine(stdout, "# xh run %s", recipe)
		}
		WriteLine(stdout, "")
		for _, s := range steps {
			WriteLine(stdout, "step: %s", s.Name)
			WriteLine(stdout, "  status: %s", s.Status)
			if s.Note != "" {
				WriteLine(stdout, "  note: %s", s.Note)
			}
		}
		WriteLine(stdout, "")
		if ok {
			WriteLine(stdout, "Result: ok")
		} else {
			WriteLine(stdout, "Result: failed")
		}
	}

	if ok {
		return ExitOK
	}
	return ExitError
}

func builtinCISteps(dryRun bool, stdout io.Writer, stderr io.Writer) []RunStep {
	var steps []RunStep

	root, err := repo.FindRoot("")
	if err != nil {
		root = "."
	}

	// Step 1: doctor --root <root> --json
	{
		if dryRun {
			steps = append(steps, RunStep{
				Name:   "doctor",
				Status: "planned",
				Note:   fmt.Sprintf("xh doctor --root %s --json", root),
			})
		} else {
			var errBuf bytes.Buffer
			code := handleDoctor([]string{"--root", root, "--json"}, io.Discard, &errBuf)
			status := "passed"
			if code != ExitOK {
				status = "failed"
			}
			note := "workspace healthy"
			if code != ExitOK {
				note = "workspace has issues"
			}
			if errBuf.Len() > 0 {
				note = strings.TrimSpace(errBuf.String())
			}
			steps = append(steps, RunStep{Name: "doctor", Status: status, Note: note})
			if status == "failed" {
				return steps
			}
		}
	}

	// Step 2: doctor --docs-drift --root <root> --json
	{
		if dryRun {
			steps = append(steps, RunStep{
				Name:   "doctor_docs_drift",
				Status: "planned",
				Note:   fmt.Sprintf("xh doctor --docs-drift --root %s --json", root),
			})
		} else {
			var errBuf bytes.Buffer
			code := handleDoctor([]string{"--docs-drift", "--root", root, "--json"}, io.Discard, &errBuf)
			status := "passed"
			if code != ExitOK {
				status = "failed"
			}
			note := "docs drift healthy"
			if code != ExitOK {
				note = "docs drift detected"
			}
			if errBuf.Len() > 0 {
				note = strings.TrimSpace(errBuf.String())
			}
			steps = append(steps, RunStep{Name: "doctor_docs_drift", Status: status, Note: note})
			if status == "failed" {
				return steps
			}
		}
	}

	// Step 3: examples verify --json
	{
		if dryRun {
			steps = append(steps, RunStep{
				Name:   "examples_verify",
				Status: "planned",
				Note:   "xh examples verify --json",
			})
		} else {
			var errBuf bytes.Buffer
			code := handleExamplesVerify([]string{"--json"}, io.Discard, &errBuf)
			status := "passed"
			if code != ExitOK {
				status = "failed"
			}
			note := "all golden examples passed"
			if code != ExitOK {
				note = "some golden examples failed"
				if errBuf.Len() > 0 {
					note = strings.TrimSpace(errBuf.String())
				}
			}
			steps = append(steps, RunStep{Name: "examples_verify", Status: status, Note: note})
			if status == "failed" {
				return steps
			}
		}
	}

	// Step 4: verify --profile ci-standard --card <root>/examples/ci/strict-verify/completion-card.yaml --json
	{
		cardPath := filepath.Join(root, "examples/ci/strict-verify/completion-card.yaml")
		if dryRun {
			steps = append(steps, RunStep{
				Name:   "verify_ci_standard",
				Status: "planned",
				Note:   fmt.Sprintf("xh verify --profile ci-standard --card %s --json", cardPath),
			})
		} else {
			var errBuf bytes.Buffer
			code := handleVerify([]string{"--profile", "ci-standard", "--card", cardPath, "--json"}, io.Discard, &errBuf)
			status := "passed"
			if code != ExitOK {
				status = "failed"
			}
			note := "ci-standard verify passed"
			if code != ExitOK {
				note = "ci-standard verify failed"
				if errBuf.Len() > 0 {
					note = strings.TrimSpace(errBuf.String())
				}
			}
			steps = append(steps, RunStep{Name: "verify_ci_standard", Status: status, Note: note})
			if status == "failed" {
				return steps
			}
		}
	}

	return steps
}
