package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/contextcheck"
	"github.com/BrianNguyen29/x-harness/internal/contextmanifest"
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

func runtimeContractMarkdown() string {
	return strings.Join([]string{
		"# x-harness Generated Runtime Contract",
		"",
		"Generated from file-first source artifacts and the renderer mirror:",
		"",
		"- policies/admission.yaml",
		"- schemas/completion-card.schema.json",
		"- packages/cli/src/core/contract.ts",
		"",
		"## Canonical Rules",
		"",
		"- Completion is admitted, not claimed.",
		"- Verifier is read-only.",
		"- Success is the only accepted outcome.",
		"- Canonical tiers: light, standard, deep.",
		"- PGV is advisory-only.",
		"",
		"## Fix Status Fields",
		"",
		"Completion cards use claim.fix_status as the canonical fix-status field. Subagent returns may use result.fix_status only in compatibility return payloads.",
		"",
		"## Completion Candidate",
		"",
		"```yaml",
		"claim:",
		"  fix_status: fixed",
		"verification:",
		"  status: passed",
		"```",
		"",
		"## Accepted Completion",
		"",
		"```yaml",
		"admission:",
		"  outcome: success",
		"acceptance_status: accepted",
		"```",
		"",
		"## Evidence Floor",
		"",
		"- **light**: files_changed + (command_evidence or manual_rationale).",
		"- **standard**: files_changed + command_evidence + done_checklist + prediction.",
		"- **deep**: files_changed + command_evidence + evidence_scope_declared + untested_regions_declared + remaining_risks_declared + execution_controls_present + rollback_policy_present + done_checklist + prediction. Runtime-enforced: verification_artifacts, state.read_set, state.write_set.",
		"",
		"## Strict Evidence Provenance",
		"",
		"- verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.",
		"- verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.",
	}, "\n")
}

type evidenceFloorTier struct {
	Required        []string `json:"required"`
	OneOf           []string `json:"oneOf,omitempty"`
	Recommended     []string `json:"recommended,omitempty"`
	RuntimeEnforced []string `json:"runtimeEnforced,omitempty"`
}

type contractJSONOutput struct {
	Facts     []ContractFact `json:"facts"`
	Rules     []string       `json:"rules"`
	FixStatus struct {
		CompletionCard string `json:"completionCard"`
		SubagentReturn string `json:"subagentReturn"`
	} `json:"fixStatus"`
	CompletionCandidate struct {
		Claim        map[string]string `json:"claim"`
		Verification map[string]string `json:"verification"`
	} `json:"completionCandidate"`
	AcceptedCompletion struct {
		Admission        map[string]string `json:"admission"`
		AcceptanceStatus string            `json:"acceptanceStatus"`
	} `json:"acceptedCompletion"`
	EvidenceFloor struct {
		Light    evidenceFloorTier `json:"light"`
		Standard evidenceFloorTier `json:"standard"`
		Deep     evidenceFloorTier `json:"deep"`
	} `json:"evidenceFloor"`
	StrictProvenance []string `json:"strictProvenance"`
	Hash             string   `json:"hash"`
	Markdown         string   `json:"markdown"`
}

func generateManagedBlock() string {
	return contextcheck.ManagedContextBlock(rootContextRegistryEntry())
}

func injectManagedBlock(content, block string) string {
	return contextcheck.InjectManagedBlock(content, rootContextRegistryEntry(), block)
}

func rootContextRegistryEntry() contextcheck.RegistryEntry {
	return contextcheck.RegistryEntry{
		Path:        "AGENTS.md",
		Type:        "context",
		BeginMarker: contextcheck.ManagedBegin,
		EndMarker:   contextcheck.ManagedEnd,
		HashPrefix:  "<!-- context-hash: ",
	}
}

func runContextRegistrySync(root string, checkMode, jsonMode bool, stdout io.Writer, stderr io.Writer) int {
	registry, err := contextcheck.ReadRegistry(root)
	if err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{"valid": false, "registry": true, "error": err.Error()})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return ExitError
	}

	checked := 0
	stale := []string{}
	updated := []string{}
	for _, entry := range registry.Blocks {
		if entry.Type != "context" {
			continue
		}
		checked++
		targetPath, resolveErr := contextcheck.ResolveRegistryEntryPath(root, entry.Path)
		if resolveErr != nil {
			stale = append(stale, fmt.Sprintf("%s: %v", entry.Path, resolveErr))
			continue
		}
		content, readErr := os.ReadFile(targetPath)
		if readErr != nil {
			stale = append(stale, fmt.Sprintf("%s: unreadable: %v", entry.Path, readErr))
			continue
		}
		valid, note := contextcheck.ValidateManagedBlockExpected(
			string(content),
			entry.BeginMarker,
			entry.EndMarker,
			entry.HashPrefix,
			contextcheck.CanonicalContext(),
		)
		if valid {
			continue
		}
		if checkMode {
			stale = append(stale, fmt.Sprintf("%s: %s", entry.Path, note))
			continue
		}

		block := contextcheck.ManagedContextBlock(entry)
		next := contextcheck.InjectManagedBlock(string(content), entry, block)
		if err := os.WriteFile(targetPath, []byte(next), 0644); err != nil {
			stale = append(stale, fmt.Sprintf("%s: write failed: %v", entry.Path, err))
			continue
		}
		updated = append(updated, entry.Path)
	}

	if checked == 0 {
		stale = append(stale, "registry contains no context entries")
	}
	valid := len(stale) == 0
	if jsonMode {
		_ = WriteJSON(stdout, map[string]any{
			"valid":    valid,
			"registry": true,
			"checked":  checked,
			"stale":    stale,
			"updated":  updated,
		})
	} else if checkMode {
		if valid {
			fmt.Fprintf(stdout, "managed context registry is fresh (%d file(s) checked)\n", checked)
		} else {
			fmt.Fprintf(stderr, "managed context registry is stale:\n- %s\n", strings.Join(stale, "\n- "))
		}
	} else {
		if valid {
			fmt.Fprintf(stdout, "refreshed %d managed context file(s)\n", len(updated))
		} else {
			fmt.Fprintf(stderr, "managed context registry refresh failed:\n- %s\n", strings.Join(stale, "\n- "))
		}
	}

	if valid {
		return ExitOK
	}
	return ExitError
}

func runContext(args []string, stdout io.Writer, _ io.Writer) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
		}
	}

	markdown := runtimeContractMarkdown()
	hash := contextcheck.ContextHash(markdown)

	if jsonMode {
		output := contractJSONOutput{
			Facts: CoreContract().Facts,
			Rules: []string{
				"Completion is admitted, not claimed.",
				"Verifier is read-only.",
				"Success is the only accepted outcome.",
				"Canonical tiers: light, standard, deep.",
				"PGV is advisory-only.",
			},
			FixStatus: struct {
				CompletionCard string `json:"completionCard"`
				SubagentReturn string `json:"subagentReturn"`
			}{
				CompletionCard: "Completion cards use claim.fix_status as the canonical fix-status field.",
				SubagentReturn: "Subagent returns may use result.fix_status only in compatibility return payloads.",
			},
			CompletionCandidate: struct {
				Claim        map[string]string `json:"claim"`
				Verification map[string]string `json:"verification"`
			}{
				Claim:        map[string]string{"fix_status": "fixed"},
				Verification: map[string]string{"status": "passed"},
			},
			AcceptedCompletion: struct {
				Admission        map[string]string `json:"admission"`
				AcceptanceStatus string            `json:"acceptanceStatus"`
			}{
				Admission:        map[string]string{"outcome": "success"},
				AcceptanceStatus: "accepted",
			},
			EvidenceFloor: struct {
				Light    evidenceFloorTier `json:"light"`
				Standard evidenceFloorTier `json:"standard"`
				Deep     evidenceFloorTier `json:"deep"`
			}{
				Light: evidenceFloorTier{
					Required: []string{"files_changed"},
					OneOf:    []string{"command_evidence", "manual_rationale"},
				},
				Standard: evidenceFloorTier{
					Required: []string{"files_changed", "command_evidence", "done_checklist", "prediction"},
				},
				Deep: evidenceFloorTier{
					Required:        []string{"files_changed", "command_evidence", "evidence_scope_declared", "untested_regions_declared", "remaining_risks_declared", "execution_controls_present", "rollback_policy_present", "done_checklist", "prediction"},
					RuntimeEnforced: []string{"verification_artifacts", "state.read_set", "state.write_set"},
				},
			},
			StrictProvenance: []string{
				"verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.",
				"verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.",
			},
			Hash:     hash,
			Markdown: markdown,
		}
		if err := WriteJSON(stdout, output); err != nil {
			return ExitError
		}
		return ExitOK
	}

	WriteLine(stdout, "%s", markdown)
	WriteLine(stdout, "")
	WriteLine(stdout, "contract-hash: %s", hash)
	return ExitOK
}

func runContextSync(args []string, stdout io.Writer, stderr io.Writer) int {
	checkMode := false
	writeMode := false
	jsonMode := false
	registryMode := false
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
		case "--registry":
			registryMode = true
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		}
	}

	if !checkMode && !writeMode {
		fmt.Fprintln(stderr, "usage: x-harness context sync --check|--write [--registry] [--root <path>] [--json]")
		return ExitUsage
	}
	if checkMode && writeMode {
		fmt.Fprintln(stderr, "error: context sync accepts only one of --check or --write")
		return ExitUsage
	}

	if root == "" {
		root = "."
	}
	if registryMode {
		return runContextRegistrySync(root, checkMode, jsonMode, stdout, stderr)
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

func runContextManifest(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness context manifest write --files <paths> [--out <path>] [--json] | context manifest check --manifest <path> [--json]")
		return ExitUsage
	}

	switch args[0] {
	case "write":
		return runContextManifestWrite(args[1:], stdout, stderr)
	case "check":
		return runContextManifestCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown manifest subcommand: %s\n", args[0])
		return ExitUsage
	}
}

func runContextManifestWrite(args []string, stdout io.Writer, stderr io.Writer) int {
	var files []string
	out := ".x-harness/context-manifest.yaml"
	jsonMode := false
	reason := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--files":
			if i+1 < len(args) {
				for _, f := range strings.Split(args[i+1], ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						files = append(files, f)
					}
				}
				i++
			}
		case "--out":
			if i+1 < len(args) {
				out = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		case "--reason":
			if i+1 < len(args) {
				reason = args[i+1]
				i++
			}
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness context manifest write --files <comma-separated-paths> [--out <path>] [--json] [--reason <reason>]")
		return ExitUsage
	}

	manifest, err := contextmanifest.Generate(files, ".", reason)
	if err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{"ok": false, "error": err.Error()})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return ExitError
	}

	if err := contextmanifest.Write(manifest, out); err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{"ok": false, "error": err.Error()})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return ExitError
	}

	if jsonMode {
		entries := make([]map[string]string, len(manifest.Entries))
		for i, e := range manifest.Entries {
			entries[i] = map[string]string{
				"path":   e.Path,
				"sha256": e.SHA256,
			}
		}
		_ = WriteJSON(stdout, map[string]any{
			"ok":      true,
			"out":     out,
			"entries": entries,
		})
	} else {
		fmt.Fprintf(stdout, "wrote manifest (%d entries) to %s\n", len(manifest.Entries), out)
	}
	return ExitOK
}

func runContextManifestCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	manifestPath := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--manifest":
			if i+1 < len(args) {
				manifestPath = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		}
	}

	if manifestPath == "" {
		fmt.Fprintln(stderr, "usage: x-harness context manifest check --manifest <path> [--json]")
		return ExitUsage
	}

	manifest, err := contextmanifest.Read(manifestPath)
	if err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{"ok": false, "error": err.Error()})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return ExitError
	}

	if err := contextmanifest.Validate(manifest); err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{"ok": false, "error": err.Error()})
		} else {
			fmt.Fprintf(stderr, "Error: invalid manifest: %v\n", err)
		}
		return ExitError
	}

	stale, err := contextmanifest.Check(manifest, ".")
	if err != nil {
		if jsonMode {
			_ = WriteJSON(stdout, map[string]any{"ok": false, "error": err.Error()})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return ExitError
	}

	if jsonMode {
		result := map[string]any{
			"ok":    len(stale) == 0,
			"stale": stale,
		}
		data, _ := json.Marshal(result)
		fmt.Fprintln(stdout, string(data))
	} else {
		if len(stale) == 0 {
			fmt.Fprintln(stdout, "manifest check passed: all entries fresh")
		} else {
			fmt.Fprintf(stdout, "manifest check failed: stale entries: %s\n", strings.Join(stale, ", "))
		}
	}
	if len(stale) > 0 {
		return ExitError
	}
	return ExitOK
}

func handleContext(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness context --contract [--json] | context sync --check|--write [--registry] [--root <path>] [--json] | context gc --check|--write [--root <path>] [--json] | context manifest write --files <paths> [--out <path>] [--json] | context manifest check --manifest <path> [--json]")
		return ExitUsage
	}

	switch args[0] {
	case "--contract":
		return runContext(args[1:], stdout, stderr)
	case "sync":
		return runContextSync(args[1:], stdout, stderr)
	case "gc":
		return runContextGC(args[1:], stdout, stderr)
	case "manifest":
		return runContextManifest(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown context subcommand: %s\n", args[0])
		return ExitUsage
	}
}
