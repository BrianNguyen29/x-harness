package trace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTraceReadFromFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	data := []byte(`{"event_id":"E1","event_type":"verify_completed","outcome":"success"}` + "\n")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	events, err := ReadTraceFromFile(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].getString("event_id") != "E1" {
		t.Fatalf("expected E1, got %s", events[0].getString("event_id"))
	}
}

func TestTraceReadFromFile_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	events, err := ReadTraceFromFile(filepath.Join(tmpDir, "events.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if events != nil {
		t.Fatalf("expected nil for missing file, got %v", events)
	}
}

func TestTraceVerifyTraceChain_Empty(t *testing.T) {
	result := VerifyTraceChain(nil)
	if !result.Valid {
		t.Fatal("expected empty chain to be valid")
	}
	if result.EventsChecked != 0 {
		t.Fatalf("expected 0 events checked, got %d", result.EventsChecked)
	}
}

func TestTraceVerifyTraceChain_Valid(t *testing.T) {
	events := []TraceEvent{
		{"event_id": "E1", "event_type": "verify_completed", "outcome": "success", "event_hash": "", "previous_hash": nil},
	}
	result := VerifyTraceChain(events)
	if !result.Valid {
		t.Fatalf("expected valid chain for legacy event, got broken at %v", result.FirstBrokenIndex)
	}
}

func TestTraceVerifyTraceChain_Tampered(t *testing.T) {
	events := []TraceEvent{
		{"event_id": "E1", "event_type": "verify_completed", "outcome": "success", "event_hash": "tampered"},
	}
	result := VerifyTraceChain(events)
	if result.Valid {
		t.Fatal("expected invalid chain")
	}
	if result.FirstBrokenIndex == nil || *result.FirstBrokenIndex != 0 {
		t.Fatalf("expected broken at index 0, got %v", result.FirstBrokenIndex)
	}
}
