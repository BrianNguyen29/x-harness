package cli

import (
	"fmt"
	"io"

	"github.com/BrianNguyen29/x-harness/internal/admission"
	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/mutationguard"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

// VerifyResult is the minimal output of the verify command.
type VerifyResult struct {
	OK               bool                  `json:"ok"`
	TaskID           string                `json:"task_id"`
	Tier             string                `json:"tier"`
	AdmissionOutcome string                `json:"admission_outcome"`
	AcceptanceStatus string                `json:"acceptance_status"`
	SchemaError      string                `json:"schema_error,omitempty"`
	AdmissionErrors  []string              `json:"admission_errors,omitempty"`
	MutationGuard    *mutationguard.Result `json:"mutation_guard,omitempty"`
}

func handleVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := ""
	jsonMode := false
	verbose := false
	useMutationGuard := false
	strict := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 < len(args) {
				cardPath = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		case "--verbose":
			verbose = true
		case "--mutation-guard":
			useMutationGuard = true
		case "--strict":
			strict = true
			useMutationGuard = true
		}
	}

	if cardPath == "" {
		fmt.Fprintln(stderr, "usage: x-harness verify --card <path> [--json] [--verbose] [--mutation-guard] [--strict]")
		return ExitUsage
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	schemaPath := assets.NewLocator(root).Schema("completion-card.schema.json")

	var doc map[string]any
	if err := loader.LoadDocument(cardPath, &doc); err != nil {
		fmt.Fprintf(stderr, "error: cannot load card: %v\n", err)
		return ExitError
	}

	v, err := schema.Compile(schemaPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot compile schema: %v\n", err)
		return ExitError
	}

	var result VerifyResult

	if useMutationGuard {
		result = runWithMutationGuard(root, strict, doc, v, stderr)
	} else {
		schemaErr := v.Validate(doc)
		result = buildVerifyResult(doc, schemaErr, nil)
	}

	renderVerifyResult(result, jsonMode, verbose, stdout, cardPath, schemaPath)

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func runWithMutationGuard(root string, strict bool, doc map[string]any, validator *schema.Validator, stderr io.Writer) VerifyResult {
	if !mutationguard.IsGitAvailable() {
		mg := &mutationguard.Result{Enabled: true, SkippedReason: "git not available", Violated: strict}
		schemaErr := validator.Validate(doc)
		result := buildVerifyResult(doc, schemaErr, mg)
		if strict {
			fmt.Fprintln(stderr, "mutation_guard_error: git not available in strict mode")
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
		}
		return result
	}

	gitRoot, err := mutationguard.FindGitRoot(root)
	if err != nil {
		mg := &mutationguard.Result{Enabled: true, SkippedReason: "cannot find git root", Violated: strict}
		schemaErr := validator.Validate(doc)
		result := buildVerifyResult(doc, schemaErr, mg)
		if strict {
			fmt.Fprintf(stderr, "mutation_guard_error: cannot find git root in strict mode: %v\n", err)
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
		}
		return result
	}

	var schemaErr error
	mgResult, guardErr := mutationguard.Guard(gitRoot, func() error {
		schemaErr = validator.Validate(doc)
		return nil
	})

	if guardErr != nil {
		mg := &mutationguard.Result{Enabled: true, SkippedReason: guardErr.Error(), Violated: strict}
		result := buildVerifyResult(doc, schemaErr, mg)
		if strict {
			fmt.Fprintf(stderr, "mutation_guard_error: guard failed in strict mode: %v\n", guardErr)
			result.OK = false
			result.AdmissionOutcome = "blocked"
			result.AcceptanceStatus = "withheld"
		}
		return result
	}

	result := buildVerifyResult(doc, schemaErr, mgResult)
	if mgResult != nil && mgResult.Violated {
		result.OK = false
		result.AdmissionOutcome = "blocked"
		result.AcceptanceStatus = "withheld"
	}
	return result
}

func buildVerifyResult(doc map[string]any, schemaErr error, mgResult *mutationguard.Result) VerifyResult {
	result := VerifyResult{
		TaskID: stringValue(doc, "task_id"),
		Tier:   stringValue(doc, "tier"),
	}

	if schemaErr != nil {
		result.SchemaError = schemaErr.Error()
		result.AdmissionOutcome = "failed"
		result.AcceptanceStatus = "withheld"
		result.OK = false
		result.MutationGuard = mgResult
		return result
	}

	admResult := admission.Run(doc)
	result.AdmissionOutcome = admResult.Outcome
	result.AcceptanceStatus = admResult.AcceptanceStatus
	result.AdmissionErrors = admResult.Errors
	result.OK = admResult.Outcome == "success" && admResult.AcceptanceStatus == "accepted" && len(admResult.Errors) == 0
	result.MutationGuard = mgResult

	if mgResult != nil && mgResult.Violated {
		result.OK = false
		result.AdmissionOutcome = "blocked"
		result.AcceptanceStatus = "withheld"
	}

	return result
}

func renderVerifyResult(result VerifyResult, jsonMode, verbose bool, stdout io.Writer, cardPath, schemaPath string) {
	if jsonMode {
		WriteJSON(stdout, result)
		return
	}
	if verbose {
		WriteLine(stdout, "card: %s", cardPath)
		WriteLine(stdout, "schema: %s", schemaPath)
	}
	WriteLine(stdout, "task_id: %s", result.TaskID)
	WriteLine(stdout, "tier: %s", result.Tier)
	WriteLine(stdout, "outcome: %s", result.AdmissionOutcome)
	WriteLine(stdout, "acceptance_status: %s", result.AcceptanceStatus)
	if result.SchemaError != "" {
		WriteLine(stdout, "schema_error: %s", result.SchemaError)
	}
	for _, e := range result.AdmissionErrors {
		WriteLine(stdout, "admission_error: %s", e)
	}
	if result.MutationGuard != nil {
		if result.MutationGuard.SkippedReason != "" {
			WriteLine(stdout, "mutation_guard: skipped (%s)", result.MutationGuard.SkippedReason)
		} else if result.MutationGuard.Violated {
			WriteLine(stdout, "mutation_guard: violated")
			for _, d := range result.MutationGuard.UnexpectedDeltas {
				WriteLine(stdout, "  - %s", d.Path)
			}
		} else {
			WriteLine(stdout, "mutation_guard: clean")
		}
	}
}

func stringValue(doc map[string]any, key string) string {
	if v, ok := doc[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func mapValue(doc map[string]any, key string) map[string]any {
	if v, ok := doc[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}
