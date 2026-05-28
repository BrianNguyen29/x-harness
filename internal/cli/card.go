package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	"gopkg.in/yaml.v3"
)

type admissionCard struct {
	SchemaVersion string               `json:"schema_version" yaml:"schema_version"`
	GeneratedAt   string               `json:"generated_at" yaml:"generated_at"`
	XHarnessCard  admissionCardHarness `json:"x_harness_card" yaml:"x_harness_card"`
}

type admissionCardHarness struct {
	SourceRefs []admissionCardRef  `json:"source_refs" yaml:"source_refs"`
	Status     admissionCardStatus `json:"status" yaml:"status"`
}

type admissionCardRef struct {
	Path   string `json:"path" yaml:"path"`
	Exists bool   `json:"exists" yaml:"exists"`
}

type admissionCardStatus struct {
	OK   bool   `json:"ok" yaml:"ok"`
	Note string `json:"note,omitempty" yaml:"note,omitempty"`
}

type cardVerifyResult struct {
	OK          bool     `json:"ok"`
	SchemaError string   `json:"schema_error,omitempty"`
	MissingRefs []string `json:"missing_refs,omitempty"`
}

func handleCard(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness card <generate|verify> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "generate":
		return handleCardGenerate(args[1:], stdout, stderr)
	case "verify":
		return handleCardVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown card subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness card <generate|verify> [options]")
		return ExitUsage
	}
}

func handleCardGenerate(args []string, stdout io.Writer, stderr io.Writer) int {
	outPath := ".x-harness/admission-card.yaml"
	format := "yaml"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 < len(args) {
				outPath = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	if format != "yaml" && format != "json" {
		fmt.Fprintf(stderr, "unknown format: %s (expected yaml or json)\n", format)
		return ExitUsage
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	refs := []admissionCardRef{
		{Path: "AGENTS.md", Exists: fileExists(filepath.Join(root, "AGENTS.md"))},
		{Path: "X_HARNESS.md", Exists: fileExists(filepath.Join(root, "X_HARNESS.md"))},
		{Path: "policies/admission.yaml", Exists: fileExists(filepath.Join(root, "policies", "admission.yaml"))},
		{Path: "schemas/completion-card.schema.json", Exists: fileExists(filepath.Join(root, "schemas", "completion-card.schema.json"))},
	}

	card := admissionCard{
		SchemaVersion: "1.0",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		XHarnessCard: admissionCardHarness{
			SourceRefs: refs,
			Status: admissionCardStatus{
				OK:   true,
				Note: "generated",
			},
		},
	}

	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(root, outPath)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Fprintf(stderr, "error: cannot create parent directory: %v\n", err)
		return ExitError
	}

	var data []byte
	if format == "json" {
		data, err = jsonMarshal(card)
	} else {
		data, err = yaml.Marshal(card)
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot marshal card: %v\n", err)
		return ExitError
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		fmt.Fprintf(stderr, "error: cannot write card: %v\n", err)
		return ExitError
	}

	WriteLine(stdout, "admission card written to %s", outPath)
	return ExitOK
}

func handleCardVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := ".x-harness/admission-card.yaml"
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 < len(args) {
				cardPath = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		}
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	schemaPath := assets.NewLocator(root).Schema("admission-card.schema.json")
	v, err := schema.Compile(schemaPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot compile schema: %v\n", err)
		return ExitError
	}

	var card admissionCard
	if err := loader.LoadDocument(cardPath, &card); err != nil {
		fmt.Fprintf(stderr, "error: cannot load card: %v\n", err)
		return ExitError
	}

	var doc any
	if err := loader.LoadDocument(cardPath, &doc); err != nil {
		fmt.Fprintf(stderr, "error: cannot load card for validation: %v\n", err)
		return ExitError
	}

	schemaErr := v.Validate(doc)

	result := cardVerifyResult{OK: true}
	if schemaErr != nil {
		result.OK = false
		result.SchemaError = schemaErr.Error()
	}

	for _, ref := range card.XHarnessCard.SourceRefs {
		var resolved string
		if filepath.IsAbs(ref.Path) {
			resolved = ref.Path
		} else {
			resolved = filepath.Join(root, ref.Path)
		}
		if !fileExists(resolved) {
			result.OK = false
			result.MissingRefs = append(result.MissingRefs, ref.Path)
		}
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		if result.OK {
			WriteLine(stdout, "card: valid")
		} else {
			WriteLine(stdout, "card: invalid")
		}
		if result.SchemaError != "" {
			WriteLine(stdout, "schema_error: %s", result.SchemaError)
		}
		for _, ref := range result.MissingRefs {
			WriteLine(stdout, "missing_ref: %s", ref)
		}
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func jsonMarshal(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
