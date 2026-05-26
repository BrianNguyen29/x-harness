package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceAppendAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	event1 := TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}
	event2 := TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "failed"}

	_, err := AppendTrace(event1, tmpDir)
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	_, err = AppendTrace(event2, tmpDir)
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}

	events, err := ReadTrace(tmpDir)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].getString("event_id") != "E1" {
		t.Fatalf("expected E1, got %s", events[0].getString("event_id"))
	}
	if events[1].getString("event_id") != "E2" {
		t.Fatalf("expected E2, got %s", events[1].getString("event_id"))
	}
}

func TestTraceReadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	events, err := ReadTrace(filepath.Join(tmpDir, "nonexistent"))
	if err != nil {
		t.Fatalf("expected no error for missing file: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestTraceEnrichesHashFields(t *testing.T) {
	tmpDir := t.TempDir()
	event := TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}

	enriched, err := AppendTrace(event, tmpDir)
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if enriched.getString("event_hash") == "" {
		t.Fatal("expected event_hash to be set")
	}
	if enriched["previous_hash"] != nil && enriched.getString("previous_hash") != "" {
		t.Fatal("expected previous_hash to be nil for first event")
	}

	// Second event should link to first
	event2 := TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "failed"}
	enriched2, err := AppendTrace(event2, tmpDir)
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if enriched2.getString("previous_hash") != enriched.getString("event_hash") {
		t.Fatalf("expected previous_hash to match first event_hash")
	}
}

func TestVerifyTraceChainValid(t *testing.T) {
	tmpDir := t.TempDir()
	_, _ = AppendTrace(TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}, tmpDir)
	_, _ = AppendTrace(TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "failed"}, tmpDir)

	events, _ := ReadTrace(tmpDir)
	result := VerifyTraceChain(events)
	if !result.Valid {
		t.Fatalf("expected valid chain, got broken at index %v", result.FirstBrokenIndex)
	}
	if result.EventsChecked != 2 {
		t.Fatalf("expected 2 events checked, got %d", result.EventsChecked)
	}
}

func TestVerifyTraceChainEmpty(t *testing.T) {
	result := VerifyTraceChain(nil)
	if !result.Valid {
		t.Fatal("expected empty chain to be valid")
	}
	if result.EventsChecked != 0 {
		t.Fatalf("expected 0 events checked, got %d", result.EventsChecked)
	}
}

func TestVerifyTraceChainDetectsTamperedEventHash(t *testing.T) {
	tmpDir := t.TempDir()
	_, _ = AppendTrace(TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}, tmpDir)
	_, _ = AppendTrace(TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "failed"}, tmpDir)

	events, _ := ReadTrace(tmpDir)
	events[1]["event_hash"] = "tampered"
	result := VerifyTraceChain(events)
	if result.Valid {
		t.Fatal("expected invalid chain")
	}
	if result.FirstBrokenIndex == nil || *result.FirstBrokenIndex != 1 {
		t.Fatalf("expected broken at index 1, got %v", result.FirstBrokenIndex)
	}
	if result.FirstBrokenEventID != "E2" {
		t.Fatalf("expected broken event E2, got %s", result.FirstBrokenEventID)
	}
}

func TestVerifyTraceChainDetectsTamperedPreviousHash(t *testing.T) {
	tmpDir := t.TempDir()
	_, _ = AppendTrace(TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}, tmpDir)
	_, _ = AppendTrace(TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "failed"}, tmpDir)

	events, _ := ReadTrace(tmpDir)
	events[1]["previous_hash"] = "tampered"
	result := VerifyTraceChain(events)
	if result.Valid {
		t.Fatal("expected invalid chain")
	}
	if result.FirstBrokenIndex == nil || *result.FirstBrokenIndex != 1 {
		t.Fatalf("expected broken at index 1, got %v", result.FirstBrokenIndex)
	}
}

func TestVerifyTraceChainHandlesLegacyEvent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	legacyLine := `{"event_id":"LEGACY1","event_type":"verify_completed","outcome":"success"}` + "\n"
	if err := os.WriteFile(filePath, []byte(legacyLine), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	events, _ := ReadTrace(tmpDir)
	result := VerifyTraceChain(events)
	if !result.Valid {
		t.Fatalf("expected valid chain for legacy event, got broken at %v", result.FirstBrokenIndex)
	}
	if result.EventsChecked != 1 {
		t.Fatalf("expected 1 event checked, got %d", result.EventsChecked)
	}
}

func TestVerifyTraceChainHandlesMixedLegacyAndHashed(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	legacyLine := `{"event_id":"LEGACY1","event_type":"verify_completed","outcome":"success"}` + "\n"
	if err := os.WriteFile(filePath, []byte(legacyLine), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	_, _ = AppendTrace(TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "failed"}, tmpDir)

	events, _ := ReadTrace(tmpDir)
	result := VerifyTraceChain(events)
	if !result.Valid {
		t.Fatalf("expected valid chain, got broken at %v", result.FirstBrokenIndex)
	}
	if result.EventsChecked != 2 {
		t.Fatalf("expected 2 events checked, got %d", result.EventsChecked)
	}
}

func TestHashIncludesAllFieldsExceptHashFields(t *testing.T) {
	tmpDir := t.TempDir()
	_, _ = AppendTrace(TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success", "tier": "standard", "task_id": "TASK-1"}, tmpDir)

	events, _ := ReadTrace(tmpDir)
	// Tamper with an extra field
	events[0]["tier"] = "deep"
	result := VerifyTraceChain(events)
	if result.Valid {
		t.Fatal("expected invalid chain after tampering extra field")
	}
	if result.FirstBrokenIndex == nil || *result.FirstBrokenIndex != 0 {
		t.Fatalf("expected broken at index 0, got %v", result.FirstBrokenIndex)
	}
}

func TestTraceCLIAddSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "add", "--trace-dir", tmpDir, "--task-id", "TASK-1", "--tier", "standard"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "trace event appended") {
		t.Fatalf("expected appended message, got:\n%s", out)
	}
	if !strings.Contains(out, "event_id: VE-") {
		t.Fatalf("expected event_id, got:\n%s", out)
	}
	if !strings.Contains(out, "event_hash: ") {
		t.Fatalf("expected event_hash, got:\n%s", out)
	}
}

func TestTraceCLIAddInvalidTier(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "add", "--tier", "invalid"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid tier") {
		t.Fatalf("expected invalid tier error, got:\n%s", stderr.String())
	}
}

func TestTraceCLIAddInvalidOutcome(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "add", "--outcome", "invalid"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid outcome") {
		t.Fatalf("expected invalid outcome error, got:\n%s", stderr.String())
	}
}

func TestTraceCLIAddInvalidAcceptanceStatus(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "add", "--acceptance-status", "invalid"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid acceptance status") {
		t.Fatalf("expected invalid acceptance status error, got:\n%s", stderr.String())
	}
}

func TestTraceCLIAddInvalidMapping(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "add", "--outcome", "success", "--acceptance-status", "withheld"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "invalid admission mapping") {
		t.Fatalf("expected mapping error, got:\n%s", stderr.String())
	}
}

func TestTraceCLIVerifyChainSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	_, _ = AppendTrace(TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}, tmpDir)
	_, _ = AppendTrace(TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "success"}, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "verify-chain", "--trace-dir", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "chain valid") {
		t.Fatalf("expected chain valid, got:\n%s", out)
	}
	if !strings.Contains(out, "2 event(s)") {
		t.Fatalf("expected 2 events, got:\n%s", out)
	}
}

func TestTraceCLIVerifyChainFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, _ = AppendTrace(TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}, tmpDir)
	_, _ = AppendTrace(TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "success"}, tmpDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "verify-chain", "--from", filepath.Join(tmpDir, "events.jsonl")}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "chain valid") {
		t.Fatalf("expected chain valid, got:\n%s", out)
	}
}

func TestTraceCLIVerifyChainFailsForTampered(t *testing.T) {
	tmpDir := t.TempDir()
	_, _ = AppendTrace(TraceEvent{"event_id": "E1", "event_type": "verify_completed", "outcome": "success"}, tmpDir)
	_, _ = AppendTrace(TraceEvent{"event_id": "E2", "event_type": "verify_completed", "outcome": "success"}, tmpDir)

	events, _ := ReadTrace(tmpDir)
	events[1]["event_hash"] = "tampered"
	data, _ := json.Marshal(events[0])
	data2, _ := json.Marshal(events[1])
	_ = os.WriteFile(filepath.Join(tmpDir, "events.jsonl"), append(append(data, '\n'), append(data2, '\n')...), 0644)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "verify-chain", "--trace-dir", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d", ExitError, code)
	}
	if !strings.Contains(stderr.String(), "chain broken") {
		t.Fatalf("expected chain broken error, got:\n%s", stderr.String())
	}
}

func TestTraceCLIVerifyChainEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "verify-chain", "--trace-dir", tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d", ExitOK, code)
	}
	if !strings.Contains(stdout.String(), "chain valid: 0 event(s) checked") {
		t.Fatalf("expected 0 events valid, got:\n%s", stdout.String())
	}
}

func TestTraceCLIMissingSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got:\n%s", stderr.String())
	}
}

func TestTraceCLIUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"trace", "invalid"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown trace subcommand") {
		t.Fatalf("expected unknown subcommand error, got:\n%s", stderr.String())
	}
}
