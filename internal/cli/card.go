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

// completion-card structs
type completionCard struct {
	SchemaVersion    string                      `json:"schema_version" yaml:"schema_version"`
	TaskID           string                      `json:"task_id" yaml:"task_id"`
	Tier             string                      `json:"tier" yaml:"tier"`
	Owner            string                      `json:"owner" yaml:"owner"`
	Accountable      string                      `json:"accountable" yaml:"accountable"`
	Claim            completionCardClaim         `json:"claim" yaml:"claim"`
	Evidence         *completionCardEvidence     `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Verification     completionCardVerification  `json:"verification" yaml:"verification"`
	Admission        completionCardAdmission     `json:"admission" yaml:"admission"`
	AcceptanceStatus string                      `json:"acceptance_status" yaml:"acceptance_status"`
	Handoff          completionCardHandoff       `json:"handoff" yaml:"handoff"`
	DoneChecklist    *completionCardDoneChecklist `json:"done_checklist,omitempty" yaml:"done_checklist,omitempty"`
	Prediction       *completionCardPrediction    `json:"prediction,omitempty" yaml:"prediction,omitempty"`
}

type completionCardClaim struct {
	FixStatus string   `json:"fix_status" yaml:"fix_status"`
	Summary   string   `json:"summary" yaml:"summary"`
	Evidence  []any    `json:"evidence" yaml:"evidence"`
}

type completionCardEvidence struct {
	FilesChanged     []string                         `json:"files_changed,omitempty" yaml:"files_changed,omitempty"`
	CommandEvidence  []completionCardCommandEvidence  `json:"command_evidence,omitempty" yaml:"command_evidence,omitempty"`
	ManualRationale  string                           `json:"manual_rationale,omitempty" yaml:"manual_rationale,omitempty"`
}

type completionCardCommandEvidence struct {
	Command string `json:"command" yaml:"command"`
}

type completionCardVerification struct {
	Status string `json:"status" yaml:"status"`
	Checks []any  `json:"checks" yaml:"checks"`
}

type completionCardAdmission struct {
	Outcome string `json:"outcome" yaml:"outcome"`
}

type completionCardHandoff struct {
	NextAction string `json:"next_action" yaml:"next_action"`
	Owner      string `json:"owner" yaml:"owner"`
}

type completionCardDoneChecklist struct {
	SourceOfTruthRead    bool     `json:"source_of_truth_read" yaml:"source_of_truth_read"`
	ScopeExplained       bool     `json:"scope_explained" yaml:"scope_explained"`
	ReadWriteSetsDeclared bool    `json:"read_write_sets_declared" yaml:"read_write_sets_declared"`
	EvidenceAttached     bool     `json:"evidence_attached" yaml:"evidence_attached"`
	CoverageGapDeclared  bool     `json:"coverage_gap_declared" yaml:"coverage_gap_declared"`
	RiskAndRollbackDeclared bool  `json:"risk_and_rollback_declared" yaml:"risk_and_rollback_declared"`
	PredictionDeclared   bool     `json:"prediction_declared" yaml:"prediction_declared"`
	Notes                []string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type completionCardPrediction struct {
	Claim               string                         `json:"claim" yaml:"claim"`
	ExpectedEffect      string                         `json:"expected_effect" yaml:"expected_effect"`
	MeasurableSignal    string                         `json:"measurable_signal,omitempty" yaml:"measurable_signal,omitempty"`
	FalsificationMethod string                         `json:"falsification_method" yaml:"falsification_method"`
	Horizon             string                         `json:"horizon" yaml:"horizon"`
	Confidence          string                         `json:"confidence,omitempty" yaml:"confidence,omitempty"`
	Verdict             *completionCardPredictionVerdict `json:"verdict,omitempty" yaml:"verdict,omitempty"`
}

type completionCardPredictionVerdict struct {
	Status string `json:"status" yaml:"status"`
}

func handleCard(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness card <generate|verify|init> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "generate":
		return handleCardGenerate(args[1:], stdout, stderr)
	case "verify":
		return handleCardVerify(args[1:], stdout, stderr)
	case "init":
		return handleCardInit(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown card subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness card <generate|verify|init> [options]")
		return ExitUsage
	}
}

func handleCardInit(args []string, stdout io.Writer, stderr io.Writer) int {
	var (
		tier             string
		taskID           string
		owner            string
		accountable      string
		summary          string
		fixStatus        = "fixed"
		verificationStatus = "passed"
		admissionOutcome = "success"
		acceptanceStatus = "accepted"
		nextAction       = "none"
		handoffOwner     string
		outPath          string
		manualRationale  string
		predictionClaim  = "change produces intended effect"
		predictionEffect = "issue is resolved"
		predictionFalsification = "re-run verification checks"
		predictionHorizon = "same_verify"
		predictionConfidence = "medium"
	)

	var files []string
	var commands []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tier":
			if i+1 < len(args) {
				tier = args[i+1]
				i++
			}
		case "--task-id":
			if i+1 < len(args) {
				taskID = args[i+1]
				i++
			}
		case "--owner":
			if i+1 < len(args) {
				owner = args[i+1]
				i++
			}
		case "--accountable":
			if i+1 < len(args) {
				accountable = args[i+1]
				i++
			}
		case "--summary":
			if i+1 < len(args) {
				summary = args[i+1]
				i++
			}
		case "--fix-status":
			if i+1 < len(args) {
				fixStatus = args[i+1]
				i++
			}
		case "--verification-status":
			if i+1 < len(args) {
				verificationStatus = args[i+1]
				i++
			}
		case "--admission-outcome":
			if i+1 < len(args) {
				admissionOutcome = args[i+1]
				i++
			}
		case "--acceptance-status":
			if i+1 < len(args) {
				acceptanceStatus = args[i+1]
				i++
			}
		case "--next-action":
			if i+1 < len(args) {
				nextAction = args[i+1]
				i++
			}
		case "--handoff-owner":
			if i+1 < len(args) {
				handoffOwner = args[i+1]
				i++
			}
		case "--file":
			if i+1 < len(args) {
				files = append(files, args[i+1])
				i++
			}
		case "--command":
			if i+1 < len(args) {
				commands = append(commands, args[i+1])
				i++
			}
		case "--manual-rationale":
			if i+1 < len(args) {
				manualRationale = args[i+1]
				i++
			}
		case "--out":
			if i+1 < len(args) {
				outPath = args[i+1]
				i++
			}
		case "--prediction-claim":
			if i+1 < len(args) {
				predictionClaim = args[i+1]
				i++
			}
		case "--prediction-effect":
			if i+1 < len(args) {
				predictionEffect = args[i+1]
				i++
			}
		case "--prediction-falsification":
			if i+1 < len(args) {
				predictionFalsification = args[i+1]
				i++
			}
		case "--prediction-horizon":
			if i+1 < len(args) {
				predictionHorizon = args[i+1]
				i++
			}
		case "--prediction-confidence":
			if i+1 < len(args) {
				predictionConfidence = args[i+1]
				i++
			}
		}
	}

	// Validate required fields
	missing := []string{}
	if tier == "" {
		missing = append(missing, "--tier")
	}
	if taskID == "" {
		missing = append(missing, "--task-id")
	}
	if owner == "" {
		missing = append(missing, "--owner")
	}
	if accountable == "" {
		missing = append(missing, "--accountable")
	}
	if summary == "" {
		missing = append(missing, "--summary")
	}
	if len(files) == 0 && len(commands) == 0 && manualRationale == "" {
		missing = append(missing, "--file, --command, or --manual-rationale")
	}

	if tier != "" && tier != "light" && tier != "standard" && tier != "deep" {
		fmt.Fprintf(stderr, "error: invalid tier %q (expected light, standard, or deep)\n", tier)
		return ExitUsage
	}

	if len(missing) > 0 {
		fmt.Fprintln(stderr, "error: missing required fields:")
		for _, m := range missing {
			fmt.Fprintf(stderr, "  - %s\n", m)
		}
		return ExitUsage
	}

	if handoffOwner == "" {
		handoffOwner = owner
	}

	claimEvidence := []any{}
	for _, f := range files {
		claimEvidence = append(claimEvidence, f)
	}
	if manualRationale != "" {
		claimEvidence = append(claimEvidence, manualRationale)
	}
	for _, c := range commands {
		claimEvidence = append(claimEvidence, c)
	}

	evidence := &completionCardEvidence{
		FilesChanged: files,
	}
	if manualRationale != "" {
		evidence.ManualRationale = manualRationale
	}
	for _, c := range commands {
		evidence.CommandEvidence = append(evidence.CommandEvidence, completionCardCommandEvidence{Command: c})
	}

	card := completionCard{
		SchemaVersion:    "1.0",
		TaskID:           taskID,
		Tier:             tier,
		Owner:            owner,
		Accountable:      accountable,
		Claim: completionCardClaim{
			FixStatus: fixStatus,
			Summary:   summary,
			Evidence:  claimEvidence,
		},
		Evidence: evidence,
		Verification: completionCardVerification{
			Status: verificationStatus,
			Checks: []any{},
		},
		Admission: completionCardAdmission{
			Outcome: admissionOutcome,
		},
		AcceptanceStatus: acceptanceStatus,
		Handoff: completionCardHandoff{
			NextAction: nextAction,
			Owner:      handoffOwner,
		},
	}

	if tier == "standard" || tier == "deep" {
		card.DoneChecklist = &completionCardDoneChecklist{
			SourceOfTruthRead:       false,
			ScopeExplained:          false,
			ReadWriteSetsDeclared:   false,
			EvidenceAttached:        false,
			CoverageGapDeclared:     false,
			RiskAndRollbackDeclared: false,
			PredictionDeclared:      false,
		}
		card.Prediction = &completionCardPrediction{
			Claim:               predictionClaim,
			ExpectedEffect:      predictionEffect,
			FalsificationMethod: predictionFalsification,
			Horizon:             predictionHorizon,
			Confidence:          predictionConfidence,
			Verdict: &completionCardPredictionVerdict{
				Status: "pending",
			},
		}
	}

	data, err := yaml.Marshal(card)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot marshal card: %v\n", err)
		return ExitError
	}

	if outPath != "" {
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			fmt.Fprintf(stderr, "error: cannot write card: %v\n", err)
			return ExitError
		}
		WriteLine(stdout, "completion card written to %s", outPath)
	} else {
		if _, err := stdout.Write(data); err != nil {
			fmt.Fprintf(stderr, "error: cannot write to stdout: %v\n", err)
			return ExitError
		}
	}

	return ExitOK
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
