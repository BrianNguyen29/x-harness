package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

func handleIntervention(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "intervention requires a subcommand: validate")
		return ExitUsage
	}

	switch args[0] {
	case "validate":
		return handleInterventionValidate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown intervention subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func convertYAMLTimes(v any) any {
	switch val := v.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case map[string]any:
		for k, vv := range val {
			val[k] = convertYAMLTimes(vv)
		}
		return val
	case []any:
		for i, vv := range val {
			val[i] = convertYAMLTimes(vv)
		}
		return val
	default:
		return v
	}
}

func convertYAMLTimesMap(m map[string]any) map[string]any {
	converted := convertYAMLTimes(m)
	if result, ok := converted.(map[string]any); ok {
		return result
	}
	return m
}

func handleInterventionValidate(args []string, stdout, stderr io.Writer) int {
	interventionPath := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--intervention":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --intervention requires a value")
				return ExitUsage
			}
			interventionPath = args[i+1]
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

	if interventionPath == "" {
		fmt.Fprintln(stderr, "error: --intervention <path> is required")
		return ExitUsage
	}

	absPath, err := filepath.Abs(interventionPath)
	if err != nil {
		absPath = interventionPath
	}

	if _, err := os.Stat(absPath); err != nil {
		fmt.Fprintf(stderr, "error: Intervention file not found: %s\n", absPath)
		return ExitUsage
	}

	var artifact map[string]any
	if err := loader.LoadDocument(absPath, &artifact); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitError
	}

	artifact = convertYAMLTimesMap(artifact)

	valid := true
	var validationErrors []string

	// Schema validation
	schemaPath := filepath.Join(".", "schemas", "intervention.schema.json")
	if _, err := os.Stat(schemaPath); err == nil {
		validator, err := schema.Compile(schemaPath)
		if err == nil {
			if err := validator.Validate(artifact); err != nil {
				valid = false
				validationErrors = append(validationErrors, err.Error())
			}
		}
	}

	if valid {
		decision, _ := artifact["decision"].(string)
		if decision != "allow" && decision != "override" {
			valid = false
			validationErrors = append(validationErrors, "intervention decision must be allow or override")
		}
	}

	if jsonMode {
		result := map[string]any{
			"valid":  valid,
			"errors": validationErrors,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		if valid {
			fmt.Fprintln(stdout, "Intervention is valid.")
		} else {
			fmt.Fprintln(stdout, "Intervention validation failed:")
			for _, e := range validationErrors {
				fmt.Fprintf(stdout, "  - %s\n", e)
			}
		}
	}

	if !valid {
		return ExitError
	}
	return ExitOK
}
