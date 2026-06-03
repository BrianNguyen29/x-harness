package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/loader"
)

// ExplainExplanation is the top-level JSON output of `xh explain --card`.
// It surfaces the existing verify/admission fields plus a concise
// human-readable explanation block so callers do not need to map blocking
// predicates to recovery hints by hand.
type ExplainExplanation struct {
	SchemaVersion      string                `json:"schema_version"`
	TaskID             string                `json:"task_id,omitempty"`
	Tier               string                `json:"tier,omitempty"`
	Profile            string                `json:"profile,omitempty"`
	OK                 bool                  `json:"ok"`
	AdmissionOutcome   string                `json:"admission_outcome"`
	AcceptanceStatus   string                `json:"acceptance_status"`
	Summary            string                `json:"summary"`
	BlockingPredicates []string              `json:"blocking_predicates,omitempty"`
	AdmissionErrors    []string              `json:"admission_errors,omitempty"`
	WithheldReason     *withheldReason       `json:"withheld_reason,omitempty"`
	Recovery           []ExplainRecoveryHint `json:"recovery_hints,omitempty"`
	Rules              []ExplainRuleHit      `json:"rules,omitempty"`
}

// ExplainRecoveryHint is a single recovery routing entry derived from
// policies/recovery.yaml for a given blocking predicate.
type ExplainRecoveryHint struct {
	BlockingPredicate string `json:"blocking_predicate"`
	NextAction        string `json:"next_action"`
	Owner             string `json:"owner"`
	Source            string `json:"source"`
}

// ExplainRuleHit surfaces a policy rule from the matrix that matches the
// card's tier and profiles. Helps reviewers understand which rules the
// verify pipeline is consulting for this card.
type ExplainRuleHit struct {
	RuleID             string `json:"rule_id"`
	Status             string `json:"status"`
	Description        string `json:"description,omitempty"`
	AdmissionAuthority bool   `json:"admission_authority,omitempty"`
}

// recoveryRouting mirrors the relevant subset of policies/recovery.yaml.
// It is kept inline (no separate policy loader) so `xh explain` works
// without depending on a new package.
type recoveryRouting struct {
	Version int                             `yaml:"version"`
	Routing map[string]recoveryRoutingEntry `yaml:"recovery_routing"`
}

type recoveryRoutingEntry struct {
	NextAction string `yaml:"next_action"`
	Owner      string `yaml:"owner"`
}

func handleExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := ""
	fromReport := ""
	jsonMode := false
	profileName := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --card requires a value")
				return ExitUsage
			}
			cardPath = args[i+1]
			i++
		case "--from-report":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --from-report requires a value")
				return ExitUsage
			}
			fromReport = args[i+1]
			i++
		case "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: --profile requires a value")
				return ExitUsage
			}
			profileName = args[i+1]
			i++
		case "--json":
			jsonMode = true
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh explain --card <path> [--from-report <report>] [--profile <name>] [--json]")
			return ExitUsage
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	if cardPath == "" && fromReport == "" {
		fmt.Fprintln(stderr, "usage: xh explain --card <path> [--from-report <report>] [--profile <name>] [--json]")
		return ExitUsage
	}

	if fromReport != "" {
		return explainFromReport(fromReport, profileName, jsonMode, stdout, stderr)
	}

	return explainFromCard(cardPath, profileName, jsonMode, stdout, stderr)
}

// explainFromCard runs a lightweight in-process verify on the card and
// then renders an explanation derived from the verify output. We reuse
// handleVerify so the explanation stays consistent with the gate.
func explainFromCard(cardPath, profileName string, jsonMode bool, stdout, stderr io.Writer) int {
	// We don't have a clean way to capture the verify JSON output here
	// without re-running it. Build the verify argv and run the
	// existing pipeline through Run, capturing JSON output.
	verifyArgs := []string{"verify", "--card", cardPath, "--json"}
	if profileName != "" {
		verifyArgs = append(verifyArgs, "--profile", profileName)
	}
	var verifyStdout, verifyStderr strings.Builder
	verifyCode := Run(verifyArgs, &verifyStdout, &verifyStderr)
	if verifyCode != ExitOK {
		// Even on failure, the verify pipeline writes a JSON object.
		// Fall through and try to parse it.
	}

	var result VerifyResult
	if err := json.Unmarshal([]byte(verifyStdout.String()), &result); err != nil {
		fmt.Fprintf(stderr, "error: cannot parse verify output: %v\nstderr: %s\n", err, verifyStderr.String())
		return ExitError
	}

	explanation := buildExplanationFromVerify(&result, profileName, cardPath)
	if jsonMode {
		_ = WriteJSON(stdout, explanation)
	} else {
		renderExplainText(explanation, stdout)
	}
	// We do not propagate the verify exit code: explain is informational.
	return ExitOK
}

// explainFromReport renders an explanation from a previously-generated
// verify report (JSON file). This is the offline path that does NOT
// rerun the gate.
func explainFromReport(reportPath, profileName string, jsonMode bool, stdout, stderr io.Writer) int {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read report: %v\n", err)
		return ExitError
	}
	var result VerifyResult
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Fprintf(stderr, "error: cannot parse report: %v\n", err)
		return ExitError
	}
	explanation := buildExplanationFromVerify(&result, profileName, reportPath)
	if jsonMode {
		_ = WriteJSON(stdout, explanation)
	} else {
		renderExplainText(explanation, stdout)
	}
	return ExitOK
}

func buildExplanationFromVerify(result *VerifyResult, profileName, sourcePath string) *ExplainExplanation {
	explanation := &ExplainExplanation{
		SchemaVersion:    "x-harness.explain.v1",
		TaskID:           result.TaskID,
		Tier:             result.Tier,
		Profile:          profileOrDefault(result.Profile, profileName),
		OK:               result.OK,
		AdmissionOutcome: result.AdmissionOutcome,
		AcceptanceStatus: result.AcceptanceStatus,
		AdmissionErrors:  result.AdmissionErrors,
		WithheldReason:   result.WithheldReason,
	}

	var predicates []string
	if result.WithheldReason != nil && result.WithheldReason.BlockingPredicate != "" {
		predicates = append(predicates, result.WithheldReason.BlockingPredicate)
	}
	for _, e := range result.AdmissionErrors {
		if !strings.Contains(e, "blocking_predicate") {
			continue
		}
		// Each error may contain "blocking_predicate=foo" or similar.
		for _, p := range extractBlockingPredicates(e) {
			if p != "" {
				predicates = append(predicates, p)
			}
		}
	}
	explanation.BlockingPredicates = dedupSorted(predicates)

	// Summary line: a one-sentence verdict for humans and CI logs.
	explanation.Summary = summarizeExplain(explanation)

	// Recovery hints: try to load policies/recovery.yaml and surface
	// the routing entries for any blocking predicate.
	explanation.Recovery = loadRecoveryHints(explanation.BlockingPredicates)

	// Rules: pick the matrix rules whose profiles include the active
	// profile (or all rules when no profile is set). This gives
	// reviewers a quick view of which rules the gate is consulting.
	explanation.Rules = matchingMatrixRules(explanation.Profile, explanation.Tier)

	if sourcePath != "" {
		// Surface the source file under task_id when the verify result
		// did not populate it (common for non-card inputs).
		if explanation.TaskID == "" {
			explanation.TaskID = sourcePath
		}
	}

	return explanation
}

// profileOrDefault returns the result's profile when set, otherwise the
// caller-supplied profile name, otherwise "".
func profileOrDefault(resultProfile, fallback string) string {
	if resultProfile != "" {
		return resultProfile
	}
	return fallback
}

// summarizeExplain produces a one-sentence summary line.
func summarizeExplain(e *ExplainExplanation) string {
	if e.OK {
		return fmt.Sprintf("card %q is admitted (tier=%s, profile=%s)", e.TaskID, e.Tier, e.Profile)
	}
	if e.WithheldReason != nil {
		return fmt.Sprintf("card %q is withheld by %s (predicate=%s). Next: %s",
			e.TaskID,
			e.WithheldReason.FailureClass,
			e.WithheldReason.BlockingPredicate,
			e.WithheldReason.NextAction)
	}
	return fmt.Sprintf("card %q is withheld (%s/%s). Errors: %s",
		e.TaskID, e.AdmissionOutcome, e.AcceptanceStatus, strings.Join(e.AdmissionErrors, "; "))
}

// extractBlockingPredicates pulls blocking_predicate=... fragments out
// of an error message. Kept lenient so callers can grep the result.
func extractBlockingPredicates(s string) []string {
	var out []string
	for _, sep := range []string{"blocking_predicate=", "predicate="} {
		idx := strings.Index(s, sep)
		if idx < 0 {
			continue
		}
		rest := s[idx+len(sep):]
		// Take until the next whitespace, comma, or end.
		end := len(rest)
		for i, r := range rest {
			if r == ' ' || r == ',' || r == ';' || r == ')' {
				end = i
				break
			}
		}
		out = append(out, rest[:end])
	}
	return out
}

func dedupSorted(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// loadRecoveryHints reads policies/recovery.yaml and returns matching
// hints for the given blocking predicates. Returns an empty slice when
// the file is missing or unparseable; explain is read-only and must
// never fail just because recovery routing is unavailable.
func loadRecoveryHints(predicates []string) []ExplainRecoveryHint {
	if len(predicates) == 0 {
		return nil
	}
	if _, err := os.Stat("policies/recovery.yaml"); err != nil {
		return nil
	}
	var routing recoveryRouting
	if err := loader.LoadYAML("policies/recovery.yaml", &routing); err != nil {
		return nil
	}
	if routing.Routing == nil {
		return nil
	}
	var hints []ExplainRecoveryHint
	for _, p := range predicates {
		entry, ok := routing.Routing[p]
		if !ok {
			continue
		}
		hints = append(hints, ExplainRecoveryHint{
			BlockingPredicate: p,
			NextAction:        entry.NextAction,
			Owner:             entry.Owner,
			Source:            "policies/recovery.yaml",
		})
	}
	return hints
}

// matchingMatrixRules returns the matrix rules whose profiles include
// the active profile. When no profile is set, the function returns
// runtime_blocking and advisory rules (i.e. everything that is on by
// default) so the explanation stays useful.
func matchingMatrixRules(profile, tier string) []ExplainRuleHit {
	matrix := buildMatrix()
	var hits []ExplainRuleHit
	for _, rule := range matrix.Rules {
		if profile != "" {
			if !containsString(rule.Profiles, profile) {
				continue
			}
		} else {
			// No profile: keep only the rules a default card would hit.
			if rule.Status != matrixStatusRuntimeBlocking && rule.Status != matrixStatusAdvisory {
				continue
			}
		}
		auth := false
		if rule.AdmissionAuthority != nil {
			auth = *rule.AdmissionAuthority
		}
		hits = append(hits, ExplainRuleHit{
			RuleID:             rule.ID,
			Status:             rule.Status,
			Description:        rule.Description,
			AdmissionAuthority: auth,
		})
	}
	return hits
}

func renderExplainText(e *ExplainExplanation, w io.Writer) {
	WriteLine(w, "# x-harness Explain")
	WriteLine(w, "")
	WriteLine(w, "summary: %s", e.Summary)
	if e.Tier != "" {
		WriteLine(w, "tier: %s", e.Tier)
	}
	if e.Profile != "" {
		WriteLine(w, "profile: %s", e.Profile)
	}
	WriteLine(w, "admission_outcome: %s", e.AdmissionOutcome)
	WriteLine(w, "acceptance_status: %s", e.AcceptanceStatus)
	if len(e.BlockingPredicates) > 0 {
		WriteLine(w, "blocking_predicates:")
		for _, p := range e.BlockingPredicates {
			WriteLine(w, "  - %s", p)
		}
	}
	if e.WithheldReason != nil {
		WriteLine(w, "withheld_reason:")
		WriteLine(w, "  class: %s", e.WithheldReason.Class)
		WriteLine(w, "  stage: %s", e.WithheldReason.Stage)
		WriteLine(w, "  owner: %s", e.WithheldReason.Owner)
		WriteLine(w, "  next_action: %s", e.WithheldReason.NextAction)
	}
	if len(e.Recovery) > 0 {
		WriteLine(w, "recovery_hints:")
		for _, h := range e.Recovery {
			WriteLine(w, "  - predicate=%s owner=%s next=%s", h.BlockingPredicate, h.Owner, h.NextAction)
		}
	}
	if len(e.Rules) > 0 {
		WriteLine(w, "rules_consulted:")
		for _, r := range e.Rules {
			if r.Description != "" {
				WriteLine(w, "  - %s [%s] %s", r.RuleID, r.Status, r.Description)
			} else {
				WriteLine(w, "  - %s [%s]", r.RuleID, r.Status)
			}
		}
	}
}
