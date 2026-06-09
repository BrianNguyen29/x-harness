package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/worktree"
)

// TraceEvent is a single trace event stored as a flexible map.
type TraceEvent map[string]interface{}

func (e TraceEvent) getString(key string) string {
	if v, ok := e[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sha256String(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func computeEventHash(event TraceEvent, previousHash string) string {
	// Marshal to map so we can exclude previous_hash and event_hash.
	data, _ := json.Marshal(event)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	delete(m, "previous_hash")
	delete(m, "event_hash")
	m["previous_hash"] = previousHash
	canonical, _ := json.Marshal(m)
	return sha256String(string(canonical))
}

// AppendTrace appends a trace event to events.jsonl in traceDir, enriching it with hash chain fields.
func AppendTrace(event TraceEvent, traceDir string) (TraceEvent, error) {
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(traceDir, "events.jsonl")

	events, err := ReadTrace(traceDir)
	if err != nil {
		return nil, err
	}

	var previousHash string
	if len(events) > 0 {
		if h := events[len(events)-1].getString("event_hash"); h != "" {
			previousHash = h
		}
	}

	enriched := make(TraceEvent)
	for k, v := range event {
		enriched[k] = v
	}
	enriched["previous_hash"] = previousHash
	if previousHash == "" {
		enriched["previous_hash"] = nil
	}
	enriched["event_hash"] = computeEventHash(event, previousHash)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	line, err := json.Marshal(enriched)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(line); err != nil {
		return nil, err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return nil, err
	}
	return enriched, nil
}

// ReadTrace reads events from traceDir/events.jsonl.
func ReadTrace(traceDir string) ([]TraceEvent, error) {
	filePath := filepath.Join(traceDir, "events.jsonl")
	return ReadTraceFromFile(filePath)
}

// ReadTraceFromFile reads events from a specific JSONL file.
func ReadTraceFromFile(filePath string) ([]TraceEvent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var events []TraceEvent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event TraceEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// ChainVerificationResult is the result of verifyTraceChain.
type ChainVerificationResult struct {
	Valid              bool   `json:"valid"`
	EventsChecked      int    `json:"events_checked"`
	FirstBrokenIndex   *int   `json:"first_broken_index,omitempty"`
	FirstBrokenEventID string `json:"first_broken_event_id,omitempty"`
	ExpectedHash       string `json:"expected_hash,omitempty"`
	ActualHash         string `json:"actual_hash,omitempty"`
}

// VerifyTraceChain verifies the integrity of the trace hash chain.
func VerifyTraceChain(events []TraceEvent) ChainVerificationResult {
	if len(events) == 0 {
		return ChainVerificationResult{Valid: true, EventsChecked: 0}
	}

	for i := 0; i < len(events); i++ {
		event := events[i]
		var previousHash string
		if i > 0 {
			previousHash = events[i-1].getString("event_hash")
		}

		// Skip legacy events without event_hash.
		if event.getString("event_hash") == "" {
			continue
		}

		expectedHash := computeEventHash(event, previousHash)
		if event.getString("event_hash") != expectedHash {
			idx := i
			return ChainVerificationResult{
				Valid:              false,
				EventsChecked:      i + 1,
				FirstBrokenIndex:   &idx,
				FirstBrokenEventID: event.getString("event_id"),
				ExpectedHash:       expectedHash,
				ActualHash:         event.getString("event_hash"),
			}
		}

		// Verify previous_hash linkage for i > 0.
		if i > 0 {
			expectedPrev := events[i-1].getString("event_hash")
			actualPrev := event.getString("previous_hash")
			if actualPrev != expectedPrev {
				idx := i
				return ChainVerificationResult{
					Valid:              false,
					EventsChecked:      i + 1,
					FirstBrokenIndex:   &idx,
					FirstBrokenEventID: event.getString("event_id"),
					ExpectedHash:       expectedPrev,
					ActualHash:         actualPrev,
				}
			}
		}
	}

	return ChainVerificationResult{
		Valid:         true,
		EventsChecked: len(events),
	}
}

var validTraceOutcomes = []string{"success", "failed", "blocked", "skipped", "timeout", "error"}
var validTraceTiers = []string{"light", "standard", "deep"}
var validAcceptanceStatuses = []string{"accepted", "withheld"}

func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func handleTrace(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness trace <add|verify-chain|timeline|explain|inspect> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "add":
		return handleTraceAdd(args[1:], stdout, stderr)
	case "verify-chain":
		return handleTraceVerifyChain(args[1:], stdout, stderr)
	case "timeline":
		return handleTraceTimeline(args[1:], stdout, stderr)
	case "explain":
		return handleTraceExplain(args[1:], stdout, stderr)
	case "inspect":
		return handleTraceInspect(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown trace subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness trace <add|verify-chain|timeline|explain|inspect> [options]")
		return ExitUsage
	}
}

func handleTraceAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	outcome := "success"
	acceptanceStatus := "accepted"
	taskID := "TASK-UNKNOWN"
	tier := "standard"
	claimID := ""
	evidenceID := ""
	traceDir := ".x-harness/traces"
	worktreeAware := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--outcome":
			if i+1 < len(args) {
				outcome = args[i+1]
				i++
			}
		case "--acceptance-status":
			if i+1 < len(args) {
				acceptanceStatus = args[i+1]
				i++
			}
		case "--task-id":
			if i+1 < len(args) {
				taskID = args[i+1]
				i++
			}
		case "--tier":
			if i+1 < len(args) {
				tier = args[i+1]
				i++
			}
		case "--claim-id":
			if i+1 < len(args) {
				claimID = args[i+1]
				i++
			}
		case "--evidence-id":
			if i+1 < len(args) {
				evidenceID = args[i+1]
				i++
			}
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--worktree-aware":
			worktreeAware = true
		}
	}

	if !containsString(validTraceTiers, tier) {
		fmt.Fprintln(stderr, "invalid tier: must be one of light, standard, deep")
		return ExitUsage
	}
	if !containsString(validTraceOutcomes, outcome) {
		fmt.Fprintln(stderr, "invalid outcome: must be one of success, failed, blocked, skipped, timeout, error")
		return ExitUsage
	}
	if !containsString(validAcceptanceStatuses, acceptanceStatus) {
		fmt.Fprintln(stderr, "invalid acceptance status: must be accepted or withheld")
		return ExitUsage
	}
	if (outcome == "success" && acceptanceStatus != "accepted") || (outcome != "success" && acceptanceStatus != "withheld") {
		fmt.Fprintln(stderr, "invalid admission mapping: success requires accepted; non-success requires withheld")
		return ExitUsage
	}

	event := TraceEvent{
		"event_id":             fmt.Sprintf("VE-%d", time.Now().UnixMilli()),
		"event_type":           "verify_completed",
		"task_id":              taskID,
		"tier":                 tier,
		"verifier":             "x-harness",
		"verifier_mode":        "read_only",
		"outcome":              outcome,
		"acceptance_status":    acceptanceStatus,
		"blocking_predicate":   nil,
		"blocked_reason_class": nil,
		"next_owner":           nil,
		"next_action":          nil,
		"created_at":           time.Now().UTC().Format(time.RFC3339Nano),
	}
	if claimID != "" {
		event["claim_id"] = claimID
	} else {
		event["claim_id"] = nil
	}
	if evidenceID != "" {
		event["evidence_id"] = evidenceID
	} else {
		event["evidence_id"] = nil
	}
	if worktreeAware {
		wt := worktree.CollectInfo("")
		if wt != nil {
			event["worktree"] = wt
		}
	}

	enriched, err := AppendTrace(event, traceDir)
	if err != nil {
		fmt.Fprintf(stderr, "failed to append trace: %v\n", err)
		return ExitError
	}

	fmt.Fprintln(stdout, "trace event appended")
	fmt.Fprintf(stdout, "event_id: %s\n", enriched.getString("event_id"))
	fmt.Fprintf(stdout, "event_hash: %s\n", enriched.getString("event_hash"))
	if enriched.getString("previous_hash") != "" {
		fmt.Fprintf(stdout, "previous_hash: %s\n", enriched.getString("previous_hash"))
	}
	return ExitOK
}

func handleTraceVerifyChain(args []string, stdout io.Writer, stderr io.Writer) int {
	traceDir := ".x-harness/traces"
	fromFile := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--from":
			if i+1 < len(args) {
				fromFile = args[i+1]
				i++
			}
		}
	}

	var events []TraceEvent
	var err error
	if fromFile != "" {
		events, err = ReadTraceFromFile(fromFile)
	} else {
		events, err = ReadTrace(traceDir)
	}
	if err != nil {
		fmt.Fprintf(stderr, "failed to read trace: %v\n", err)
		return ExitError
	}

	result := VerifyTraceChain(events)
	if result.Valid {
		fmt.Fprintf(stdout, "chain valid: %d event(s) checked\n", result.EventsChecked)
		return ExitOK
	}

	fmt.Fprintf(stderr, "chain broken at index %d (event_id: %s)\n", *result.FirstBrokenIndex, result.FirstBrokenEventID)
	fmt.Fprintf(stderr, "expected hash: %s\n", result.ExpectedHash)
	fmt.Fprintf(stderr, "actual hash:   %s\n", result.ActualHash)
	return ExitError
}

func readEvents(args []string, traceDir string, fromFile string) ([]TraceEvent, string, string, error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--from":
			if i+1 < len(args) {
				fromFile = args[i+1]
				i++
			}
		}
	}

	var events []TraceEvent
	var err error
	if fromFile != "" {
		events, err = ReadTraceFromFile(fromFile)
	} else {
		events, err = ReadTrace(traceDir)
	}
	if err != nil {
		return nil, traceDir, fromFile, err
	}
	return events, traceDir, fromFile, nil
}

func handleTraceTimeline(args []string, stdout io.Writer, stderr io.Writer) int {
	taskID := ""
	traceDir := ".x-harness/traces"
	fromFile := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--task", "--task-id":
			if i+1 < len(args) {
				taskID = args[i+1]
				i++
			}
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--from":
			if i+1 < len(args) {
				fromFile = args[i+1]
				i++
			}
		}
	}

	if taskID == "" {
		fmt.Fprintln(stderr, "usage: x-harness trace timeline --task-id <task_id> [--trace-dir <dir>]")
		return ExitUsage
	}

	var events []TraceEvent
	var err error
	if fromFile != "" {
		events, err = ReadTraceFromFile(fromFile)
	} else {
		events, err = ReadTrace(traceDir)
	}
	if err != nil {
		fmt.Fprintf(stderr, "failed to read trace: %v\n", err)
		return ExitError
	}

	var filtered []TraceEvent
	for _, e := range events {
		if e.getString("task_id") == taskID {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		fmt.Fprintf(stdout, "no trace events for task %s\n", taskID)
		return ExitOK
	}

	fmt.Fprintf(stdout, "%s\n", taskID)
	for _, e := range filtered {
		stage := e.getString("event_type")
		if stage == "" {
			stage = "unknown"
		}
		outcome := e.getString("outcome")
		if outcome == "" {
			outcome = "-"
		}
		fmt.Fprintf(stdout, "  %-26s %s\n", stage, outcome)
		if reason := e.getString("blocked_reason_class"); reason != "" {
			fmt.Fprintf(stdout, "    reason: %s\n", reason)
		}
		if pred := e.getString("blocking_predicate"); pred != "" {
			fmt.Fprintf(stdout, "    predicate: %s\n", pred)
		}
		if next := e.getString("next_action"); next != "" {
			fmt.Fprintf(stdout, "    next: %s\n", next)
		}
	}
	return ExitOK
}

func handleTraceExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	taskID := ""
	traceDir := ".x-harness/traces"
	fromFile := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--task", "--task-id":
			if i+1 < len(args) {
				taskID = args[i+1]
				i++
			}
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--from":
			if i+1 < len(args) {
				fromFile = args[i+1]
				i++
			}
		}
	}

	if taskID == "" {
		fmt.Fprintln(stderr, "usage: x-harness trace explain --task-id <task_id> [--trace-dir <dir>]")
		return ExitUsage
	}

	var events []TraceEvent
	var err error
	if fromFile != "" {
		events, err = ReadTraceFromFile(fromFile)
	} else {
		events, err = ReadTrace(traceDir)
	}
	if err != nil {
		fmt.Fprintf(stderr, "failed to read trace: %v\n", err)
		return ExitError
	}

	var filtered []TraceEvent
	for _, e := range events {
		if e.getString("task_id") == taskID {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		fmt.Fprintf(stdout, "no trace events for task %s\n", taskID)
		return ExitOK
	}

	fmt.Fprintf(stdout, "%s explain\n", taskID)
	hasBlocking := false
	for _, e := range filtered {
		outcome := e.getString("outcome")
		if outcome == "success" || outcome == "" {
			continue
		}
		hasBlocking = true
		stage := e.getString("event_type")
		if stage == "" {
			stage = "unknown"
		}
		fmt.Fprintf(stdout, "  %s: %s\n", stage, outcome)
		if pred := e.getString("blocking_predicate"); pred != "" {
			fmt.Fprintf(stdout, "    predicate: %s\n", pred)
		}
		if next := e.getString("next_action"); next != "" {
			fmt.Fprintf(stdout, "    next_action: %s\n", next)
		}
		if reason := e.getString("blocked_reason_class"); reason != "" {
			fmt.Fprintf(stdout, "    reason_class: %s\n", reason)
		}
	}
	if !hasBlocking {
		fmt.Fprintf(stdout, "  no blocking events found\n")
	}
	return ExitOK
}

func handleTraceInspect(args []string, stdout io.Writer, stderr io.Writer) int {
	withheldOnly := false
	traceDir := ".x-harness/traces"
	fromFile := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--withheld":
			withheldOnly = true
		case "--trace-dir":
			if i+1 < len(args) {
				traceDir = args[i+1]
				i++
			}
		case "--from":
			if i+1 < len(args) {
				fromFile = args[i+1]
				i++
			}
		}
	}

	var events []TraceEvent
	var err error
	if fromFile != "" {
		events, err = ReadTraceFromFile(fromFile)
	} else {
		events, err = ReadTrace(traceDir)
	}
	if err != nil {
		fmt.Fprintf(stderr, "failed to read trace: %v\n", err)
		return ExitError
	}

	if len(events) == 0 {
		fmt.Fprintln(stdout, "no trace events found")
		return ExitOK
	}

	groups := make(map[string]map[string]struct{})
	var total int
	for _, e := range events {
		outcome := e.getString("outcome")
		acceptance := e.getString("acceptance_status")
		if withheldOnly {
			if outcome == "success" || acceptance == "accepted" {
				continue
			}
		}
		total++
		class := e.getString("blocked_reason_class")
		if class == "" {
			if acceptance != "" {
				class = acceptance
			} else if outcome != "" {
				class = outcome
			} else {
				class = "unknown"
			}
		}
		if groups[class] == nil {
			groups[class] = make(map[string]struct{})
		}
		groups[class][e.getString("task_id")] = struct{}{}
	}

	if total == 0 {
		fmt.Fprintln(stdout, "no matching trace events found")
		return ExitOK
	}

	fmt.Fprintf(stdout, "withheld summary (%d events, %d classes)\n", total, len(groups))
	for class, tasks := range groups {
		fmt.Fprintf(stdout, "  %s (%d)\n", class, len(tasks))
		for taskID := range tasks {
			fmt.Fprintf(stdout, "    %s\n", taskID)
		}
	}
	return ExitOK
}
