package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"gopkg.in/yaml.v3"
)

// P2-S3 — `xh recover --patch-card`.
//
// This file implements the deterministic, conservative completion-card
// patch flow for `xh recover --patch-card <path>`. The contract is:
//
//   - Default mode is preview/dry-run. The card file is never mutated.
//   - `--confirm` is required to actually mutate the card file.
//   - The patcher NEVER touches source files (read-only on disk
//     outside the target card).
//   - The patcher NEVER overwrites a user-provided scalar field. If
//     a field is present (non-empty), it is left alone and reported
//     as skipped.
//   - Patch categories are intentionally narrow in V1:
//     1. `handoff.next_action` and `handoff.owner` are filled when
//        missing and the card is withheld/blocked, using the
//        deterministic recovery route from `defaultRoutes`.
//     2. `--evidence <id-or-path>` appends the value to
//        `claim.evidence` and `evidence.files_changed` only when the
//        target field is empty (schema-safe list append).
//
// Comments in the source YAML are NOT preserved: we re-marshal via
// the typed `completionCard` struct. Schema correctness/safety is
// more important than comment round-tripping in V1; the deferred
// risks section in the proposal already calls this out.

// patchOpAction identifies what the patcher did or would do with
// one field. Values:
//
//   - "would_set" — preview only (dry-run)
//   - "set"       — actually mutated the field
//   - "skipped"   — field was already populated; nothing to do
type patchOp struct {
	Field  string `json:"field"`
	Action string `json:"action"`
	Value  any    `json:"value,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// recoverPatchOutput is the JSON shape of `xh recover --patch-card
// [--json]`. The text renderer uses the same fields in a more
// human-friendly layout.
type recoverPatchOutput struct {
	SchemaVersion  string    `json:"schema_version"`
	Card           string    `json:"card"`
	OutPath        string    `json:"out_path,omitempty"`
	DryRun         bool      `json:"dry_run"`
	Confirmed      bool      `json:"confirmed"`
	Backup         string    `json:"backup,omitempty"`
	Admitted       bool      `json:"admitted"`
	RoutePredicate string    `json:"route_predicate,omitempty"`
	Ops            []patchOp `json:"ops"`
	Notes          []string  `json:"notes,omitempty"`
}

// patchFlags holds parsed flags for the patch subcommand. They are
// kept separate from the global recover flag struct so future
// `recover` subcommands can opt into the same fields without
// affecting the existing suggest flow.
type patchFlags struct {
	cardPath string
	outPath  string
	confirm  bool
	jsonMode bool
	evidence []string
}

// parsePatchFlags extracts the patch-specific flags. Unknown flags
// are surfaced as errors so the caller can return ExitUsage. Boolean
// flags (e.g. `--confirm`, `--json`) are idempotent. Repeated
// `--evidence` values are appended in the order seen.
func parsePatchFlags(args []string, stderr io.Writer) (patchFlags, error) {
	pf := patchFlags{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--patch-card":
			if i+1 >= len(args) {
				return pf, fmt.Errorf("--patch-card requires a value")
			}
			pf.cardPath = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				return pf, fmt.Errorf("--out requires a value")
			}
			pf.outPath = args[i+1]
			i++
		case "--confirm":
			pf.confirm = true
		case "--json":
			pf.jsonMode = true
		case "--evidence":
			if i+1 >= len(args) {
				return pf, fmt.Errorf("--evidence requires a value")
			}
			pf.evidence = append(pf.evidence, args[i+1])
			i++
		default:
			return pf, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	if pf.cardPath == "" {
		return pf, fmt.Errorf("--patch-card <path> is required")
	}
	return pf, nil
}

// handleRecoverPatch implements the patch flow. It is invoked from
// handleRecover when the user passes `--patch-card`.
//
// Exit codes:
//   - ExitOK: preview succeeded, or confirm succeeded.
//   - ExitError: load/parse failure, or a flagged error.
//   - ExitUsage: missing/invalid flags.
func handleRecoverPatch(args []string, stdout io.Writer, stderr io.Writer) int {
	pf, err := parsePatchFlags(args, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "usage: xh recover --patch-card <path> [--confirm] [--out <path>] [--evidence <id-or-path>] [--json]")
		fmt.Fprintf(stderr, "error: %s\n", err)
		return ExitUsage
	}

	cardBytes, err := os.ReadFile(pf.cardPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read card %s: %v\n", pf.cardPath, err)
		return ExitError
	}

	var card completionCard
	if err := loader.LoadYAML(pf.cardPath, &card); err != nil {
		// loader.LoadYAML already tries yaml.Unmarshal; surface a clear
		// message so the user can tell parse failure from missing file.
		fmt.Fprintf(stderr, "error: cannot parse card %s: %v\n", pf.cardPath, err)
		return ExitError
	}

	predicate, route := pickPatchRoute(&card)
	ops := buildPatchOps(&card, route, pf.evidence, predicate)

	out := recoverPatchOutput{
		SchemaVersion:  "x-harness.recover.patch.v1",
		Card:           pf.cardPath,
		OutPath:        pf.outPath,
		DryRun:         !pf.confirm,
		Confirmed:      pf.confirm,
		Admitted:       isAdmittedCard(&card),
		RoutePredicate: predicate,
		Ops:            ops,
	}

	// If we have nothing to do, skip the write step and report.
	hasMutations := false
	for _, op := range ops {
		if op.Action == "would_set" || op.Action == "set" {
			hasMutations = true
			break
		}
	}

	if pf.confirm && hasMutations {
		// Backup first so a partial write cannot destroy the only
		// copy. The backup is a sibling file with a millisecond
		// timestamp suffix to keep multiple runs idempotent.
		backupPath := fmt.Sprintf("%s.bak.%d", pf.cardPath, time.Now().UnixMilli())
		if err := os.WriteFile(backupPath, cardBytes, 0o644); err != nil {
			fmt.Fprintf(stderr, "error: cannot write backup %s: %v\n", backupPath, err)
			return ExitError
		}
		out.Backup = backupPath

		// Re-marshal the (now mutated) struct back to YAML. The typed
		// struct guarantees we only emit schema-known fields.
		newBytes, err := yaml.Marshal(card)
		if err != nil {
			fmt.Fprintf(stderr, "error: cannot marshal patched card: %v\n", err)
			return ExitError
		}

		// Prefer --out for the written file when provided; this is
		// the "dry-run review" use case from the proposal. Without
		// --out, write back to the original path.
		dest := pf.cardPath
		if pf.outPath != "" {
			dest = pf.outPath
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			fmt.Fprintf(stderr, "error: cannot create parent dir for %s: %v\n", dest, err)
			return ExitError
		}
		if err := os.WriteFile(dest, newBytes, 0o644); err != nil {
			fmt.Fprintf(stderr, "error: cannot write patched card %s: %v\n", dest, err)
			return ExitError
		}

		// Mark would_set entries as set now that the file is on disk.
		for i := range out.Ops {
			if out.Ops[i].Action == "would_set" {
				out.Ops[i].Action = "set"
			}
		}
	} else if !pf.confirm && pf.outPath != "" {
		// Dry-run with --out: render the would-be patched file so
		// reviewers can inspect the proposed change in isolation.
		newBytes, err := yaml.Marshal(card)
		if err != nil {
			fmt.Fprintf(stderr, "error: cannot marshal dry-run card: %v\n", err)
			return ExitError
		}
		if err := os.MkdirAll(filepath.Dir(pf.outPath), 0o755); err != nil {
			fmt.Fprintf(stderr, "error: cannot create parent dir for %s: %v\n", pf.outPath, err)
			return ExitError
		}
		if err := os.WriteFile(pf.outPath, newBytes, 0o644); err != nil {
			fmt.Fprintf(stderr, "error: cannot write dry-run patch %s: %v\n", pf.outPath, err)
			return ExitError
		}
		out.Notes = append(out.Notes, "dry-run patch written to "+pf.outPath)
	}

	if !hasMutations {
		out.Notes = append(out.Notes, "no deterministic patches applicable; card is already consistent or has no safe field to fill")
	}

	if pf.jsonMode {
		if err := WriteJSON(stdout, out); err != nil {
			return ExitError
		}
	} else {
		renderPatchText(stdout, &out)
	}
	return ExitOK
}

// isAdmittedCard reports whether the card is in a "clean" state
// (success+accepted). Patchers still run on admitted cards in V1
// because an admitted card may have missing handoff metadata; the
// predicate just becomes a no-op for the route.
func isAdmittedCard(card *completionCard) bool {
	return card.Admission.Outcome == "success" && card.AcceptanceStatus == "accepted"
}

// pickPatchRoute chooses the recovery route to apply to the card.
// V1 uses the in-process defaultRoutes map keyed by a narrow set of
// card-state predicates. The mapping is intentionally conservative:
// when the card is admitted we still return a route (the caller
// can decide to skip), but for withheld/blocked cards we surface
// the most specific deterministic route we can derive from the
// card state itself.
func pickPatchRoute(card *completionCard) (string, recoveryRoute) {
	outcome := strings.ToLower(strings.TrimSpace(card.Admission.Outcome))
	accepted := strings.EqualFold(strings.TrimSpace(card.AcceptanceStatus), "accepted")

	switch outcome {
	case "success":
		if accepted {
			// Admitted card: the route is irrelevant for handoff
			// patches (we only fill missing fields), but report a
			// stable predicate for the JSON output.
			return "admitted", defaultRoutes["admission_failed"]
		}
		return "admission_failed", defaultRoutes["admission_failed"]
	case "blocked", "failed", "error", "timeout", "skipped":
		return outcome, defaultRoutes["admission_failed"]
	}
	// No outcome set: behave like a fresh draft; the safe default
	// route is the admission_failed fallback.
	return "admission_failed", defaultRoutes["admission_failed"]
}

// buildPatchOps constructs the planned patch operations WITHOUT
// mutating the card. The caller decides whether to apply them. The
// returned ops are in the order they would be applied; each op is
// either "would_set" (preview) or "skipped".
//
// The function mutates `card` only when building the actual values
// it would set, so a confirm run can re-marshal the struct without
// recomputing the route. In a pure preview the caller never
// re-marshals (out_path is unset), so the in-memory mutation has
// no observable effect.
func buildPatchOps(card *completionCard, route recoveryRoute, evidence []string, predicate string) []patchOp {
	var ops []patchOp

	// handoff.next_action: only fill when missing and the card has a
	// withheld/blocked outcome. Admitted cards keep whatever the user
	// wrote; we never second-guess an accepted handoff.
	if strings.TrimSpace(card.Handoff.NextAction) == "" && !isAdmittedCard(card) {
		ops = append(ops, patchOp{
			Field:  "handoff.next_action",
			Action: "would_set",
			Value:  route.NextAction,
			Reason: "missing handoff.next_action for " + predicate + " card",
		})
		card.Handoff.NextAction = route.NextAction
	} else if strings.TrimSpace(card.Handoff.NextAction) == "" && isAdmittedCard(card) {
		ops = append(ops, patchOp{
			Field:  "handoff.next_action",
			Action: "skipped",
			Reason: "card is admitted; leaving handoff.next_action alone",
		})
	} else {
		ops = append(ops, patchOp{
			Field:  "handoff.next_action",
			Action: "skipped",
			Reason: "user-provided value preserved",
			Value:  card.Handoff.NextAction,
		})
	}

	// handoff.owner: only fill when missing. Always safe to fill
	// (admitted or withheld) because the schema requires the field;
	// but we still skip when the user already set it.
	if strings.TrimSpace(card.Handoff.Owner) == "" {
		ops = append(ops, patchOp{
			Field:  "handoff.owner",
			Action: "would_set",
			Value:  route.Owner,
			Reason: "missing handoff.owner",
		})
		card.Handoff.Owner = route.Owner
	} else {
		ops = append(ops, patchOp{
			Field:  "handoff.owner",
			Action: "skipped",
			Reason: "user-provided value preserved",
			Value:  card.Handoff.Owner,
		})
	}

	// --evidence: append the supplied id/path to claim.evidence
	// (list of strings) and to evidence.files_changed (list of
	// strings) only when those fields are absent. We never overwrite
	// or merge into a non-empty list in V1 — appending into a
	// user-populated list would be silently destructive. This is
	// the schema-safe list append mentioned in the proposal.
	for _, ev := range evidence {
		if ev == "" {
			continue
		}
		// claim.evidence is required (minItems 1) and accepts any
		// scalar. Empty list is invalid; we append into it.
		if len(card.Claim.Evidence) == 0 {
			ops = append(ops, patchOp{
				Field:  "claim.evidence",
				Action: "would_set",
				Value:  []string{ev},
				Reason: "claim.evidence is empty; appending --evidence value",
			})
			card.Claim.Evidence = []any{ev}
		} else {
			ops = append(ops, patchOp{
				Field:  "claim.evidence",
				Action: "skipped",
				Reason: "claim.evidence is non-empty; not overwriting user values",
			})
		}

		// evidence is optional in the schema. We only create the
		// block when the user has no evidence.files_changed. We do
		// NOT touch an existing files_changed list.
		if card.Evidence == nil {
			ops = append(ops, patchOp{
				Field:  "evidence.files_changed",
				Action: "would_set",
				Value:  []string{ev},
				Reason: "evidence block absent; adding files_changed",
			})
			card.Evidence = &completionCardEvidence{FilesChanged: []string{ev}}
		} else if len(card.Evidence.FilesChanged) == 0 {
			ops = append(ops, patchOp{
				Field:  "evidence.files_changed",
				Action: "would_set",
				Value:  []string{ev},
				Reason: "evidence.files_changed is empty; appending --evidence value",
			})
			card.Evidence.FilesChanged = []string{ev}
		} else {
			ops = append(ops, patchOp{
				Field:  "evidence.files_changed",
				Action: "skipped",
				Reason: "evidence.files_changed is non-empty; not overwriting user values",
			})
		}
	}

	return ops
}

// renderPatchText prints the patch plan/result in plain text. It
// mirrors the doctor --fix text layout: a header, key/values, then
// the per-op list. The output is grep-friendly for CI logs.
func renderPatchText(w io.Writer, out *recoverPatchOutput) {
	WriteLine(w, "# xh recover --patch-card")
	WriteLine(w, "card: %s", out.Card)
	if out.OutPath != "" {
		WriteLine(w, "out: %s", out.OutPath)
	}
	WriteLine(w, "dry_run: %v", out.DryRun)
	WriteLine(w, "confirmed: %v", out.Confirmed)
	if out.Backup != "" {
		WriteLine(w, "backup: %s", out.Backup)
	}
	WriteLine(w, "admitted: %v", out.Admitted)
	if out.RoutePredicate != "" {
		WriteLine(w, "route_predicate: %s", out.RoutePredicate)
	}
	if len(out.Ops) > 0 {
		WriteLine(w, "ops:")
		for _, op := range out.Ops {
			if op.Value != nil && op.Value != "" {
				WriteLine(w, "  - %s: %s [%s] value=%v reason=%q", op.Field, op.Action, op.Reason, op.Value, op.Reason)
			} else {
				WriteLine(w, "  - %s: %s [%s]", op.Field, op.Action, op.Reason)
			}
		}
	}
	if len(out.Notes) > 0 {
		WriteLine(w, "notes:")
		for _, n := range out.Notes {
			WriteLine(w, "  - %s", n)
		}
	}
}
