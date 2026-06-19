package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

// ExamplesVerifyResult is the JSON output shape for "examples verify".
type ExamplesVerifyResult struct {
	OK      bool                  `json:"ok"`
	Total   int                   `json:"total"`
	Passed  int                   `json:"passed"`
	Failed  int                   `json:"failed"`
	Results []ExampleVerifyRecord `json:"results"`
}

// ExampleVerifyRecord is the per-example result.
type ExampleVerifyRecord struct {
	Name             string   `json:"name"`
	Passed           bool     `json:"passed"`
	Outcome          string   `json:"outcome"`
	AcceptanceStatus string   `json:"acceptance_status"`
	Errors           []string `json:"errors"`
	OutputMismatch   *string  `json:"output_mismatch"`
}

func handleExamples(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		WriteLine(stderr, "usage: x-harness examples <subcommand>")
		WriteLine(stderr, "Subcommands:")
		WriteLine(stderr, "  verify    Verify all golden completion cards")
		return ExitUsage
	}

	switch args[0] {
	case "verify":
		return handleExamplesVerify(args[1:], stdout, stderr)
	default:
		WriteLine(stderr, "unknown examples subcommand: %s", args[0])
		return ExitUsage
	}
}

func handleExamplesVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode := false
	suite := ""
	for i, a := range args {
		if a == "--json" {
			jsonMode = true
		}
		if strings.HasPrefix(a, "--suite=") {
			suite = strings.TrimPrefix(a, "--suite=")
		}
		if a == "--suite" && i+1 < len(args) {
			suite = args[i+1]
		}
	}

	validSuites := map[string]bool{"regression": true, "capability": true, "adversarial": true}
	if suite != "" && !validSuites[suite] {
		fmt.Fprintf(stderr, "error: invalid suite %q. valid suites: regression, capability, adversarial\n", suite)
		return ExitUsage
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	examples, err := discoverGoldenExamples(root, suite)
	if err != nil {
		fmt.Fprintf(stderr, "error: discovering examples: %v\n", err)
		return ExitError
	}

	if len(examples) == 0 {
		msg := "No golden examples found."
		if jsonMode {
			WriteJSON(stdout, map[string]any{"ok": false, "error": msg})
		} else {
			fmt.Fprintln(stderr, msg)
		}
		return ExitError
	}

	schemaPath := assets.NewLocator(root).Schema("completion-card.schema.json")
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot compile schema: %v\n", err)
		return ExitError
	}

	var records []ExampleVerifyRecord
	allPassed := true

	for _, ex := range examples {
		record := verifyExample(ex, validator, root)
		if !record.Passed {
			allPassed = false
		}
		records = append(records, record)
	}

	result := ExamplesVerifyResult{
		OK:      allPassed,
		Total:   len(records),
		Passed:  countPassed(records),
		Failed:  countFailed(records),
		Results: records,
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		renderExamplesVerifyText(stdout, result)
	}

	if allPassed {
		return ExitOK
	}
	return ExitError
}

type goldenExample struct {
	Name     string
	Dir      string
	CardPath string
}

func discoverGoldenExamples(root string, suite string) ([]goldenExample, error) {
	goldenDir := filepath.Join(root, "examples", "golden")
	var examples []goldenExample

	if suite != "" {
		suiteDir := filepath.Join(goldenDir, suite)
		entries, err := os.ReadDir(suiteDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(suiteDir, entry.Name())
			cardPath := filepath.Join(dir, "completion-card.yaml")
			if _, err := os.Stat(cardPath); err == nil {
				examples = append(examples, goldenExample{
					Name:     suite + "/" + entry.Name(),
					Dir:      dir,
					CardPath: cardPath,
				})
			}
		}
	} else {
		// Scan flat dirs for backward compatibility
		entries, err := os.ReadDir(goldenDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(goldenDir, entry.Name())
			cardPath := filepath.Join(dir, "completion-card.yaml")
			if _, err := os.Stat(cardPath); err == nil {
				examples = append(examples, goldenExample{
					Name:     entry.Name(),
					Dir:      dir,
					CardPath: cardPath,
				})
			}
		}

		// Also scan known suite subdirectories
		for _, s := range []string{"regression", "capability", "adversarial"} {
			suiteDir := filepath.Join(goldenDir, s)
			entries, err := os.ReadDir(suiteDir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				dir := filepath.Join(suiteDir, entry.Name())
				cardPath := filepath.Join(dir, "completion-card.yaml")
				if _, err := os.Stat(cardPath); err == nil {
					examples = append(examples, goldenExample{
						Name:     s + "/" + entry.Name(),
						Dir:      dir,
						CardPath: cardPath,
					})
				}
			}
		}
	}

	// Sort by name for stable output
	for i := 0; i < len(examples)-1; i++ {
		for j := i + 1; j < len(examples); j++ {
			if examples[i].Name > examples[j].Name {
				examples[i], examples[j] = examples[j], examples[i]
			}
		}
	}

	return examples, nil
}

func verifyExample(ex goldenExample, validator *schema.Validator, root string) ExampleVerifyRecord {
	errors := []string{}

	var doc map[string]any
	if err := loader.LoadDocument(ex.CardPath, &doc); err != nil {
		errors = append(errors, fmt.Sprintf("unexpected error: %v", err))
		return ExampleVerifyRecord{
			Name:             ex.Name,
			Passed:           false,
			Outcome:          "error",
			AcceptanceStatus: "withheld",
			Errors:           errors,
			OutputMismatch:   nil,
		}
	}

	schemaErr := validator.Validate(doc)
	if schemaErr != nil {
		errors = append(errors, fmt.Sprintf("completion card validation failed: %v", schemaErr))
	}

	admResult := admission.Run(doc, admission.AdmissionOptions{})
	errors = append(errors, admResult.Errors...)

	outcome := admResult.Outcome
	if len(errors) > 0 {
		outcome = "failed"
	}
	acceptance := acceptanceStatus(outcome)

	expectedOutputPath := filepath.Join(ex.Dir, "expected-verify-output.txt")
	var outputMismatch *string
	expectedOutcome, expectedAcceptance, err := readExpectedVerifySummary(expectedOutputPath)
	if err != nil {
		msg := fmt.Sprintf("Missing expected output snapshot: %s", expectedOutputPath)
		outputMismatch = &msg
	} else {
		if outcome != expectedOutcome || acceptance != expectedAcceptance {
			msg := fmt.Sprintf("Output mismatch. Expected outcome=%s acceptance_status=%s, got outcome=%s acceptance_status=%s", expectedOutcome, expectedAcceptance, outcome, acceptance)
			outputMismatch = &msg
		}
	}

	passed := outputMismatch == nil

	return ExampleVerifyRecord{
		Name:             ex.Name,
		Passed:           passed,
		Outcome:          outcome,
		AcceptanceStatus: acceptance,
		Errors:           errors,
		OutputMismatch:   outputMismatch,
	}
}

func readExpectedVerifySummary(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	var outcome string
	var acceptance string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "outcome: "):
			outcome = strings.TrimSpace(strings.TrimPrefix(line, "outcome: "))
		case strings.HasPrefix(line, "acceptance_status: "):
			acceptance = strings.TrimSpace(strings.TrimPrefix(line, "acceptance_status: "))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if outcome == "" || acceptance == "" {
		return "", "", fmt.Errorf("missing outcome or acceptance_status")
	}
	return outcome, acceptance, nil
}

func acceptanceStatus(outcome string) string {
	if outcome == "success" {
		return "accepted"
	}
	return "withheld"
}

func countPassed(records []ExampleVerifyRecord) int {
	count := 0
	for _, r := range records {
		if r.Passed {
			count++
		}
	}
	return count
}

func countFailed(records []ExampleVerifyRecord) int {
	count := 0
	for _, r := range records {
		if !r.Passed {
			count++
		}
	}
	return count
}

func renderExamplesVerifyText(w io.Writer, result ExamplesVerifyResult) {
	WriteLine(w, "Golden examples: %d total", result.Total)
	for _, r := range result.Results {
		icon := "✓"
		if !r.Passed {
			icon = "✗"
		}
		WriteLine(w, "%s %s: %s (%s)", icon, r.Name, r.Outcome, r.AcceptanceStatus)
		for _, e := range r.Errors {
			WriteLine(w, "  - %s", e)
		}
		if r.OutputMismatch != nil {
			WriteLine(w, "  - %s", *r.OutputMismatch)
		}
	}
	WriteLine(w, "")
	if result.OK {
		WriteLine(w, "All golden examples passed.")
	} else {
		WriteLine(w, "Some golden examples failed.")
	}
}
