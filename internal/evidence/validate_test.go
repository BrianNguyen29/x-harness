package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func makeValidEntry() map[string]any {
	return map[string]any{
		"schema_version":      "1",
		"task_id":             "task-001",
		"evidence_id":         "ev-001",
		"layer":               "raw",
		"kind":                "other",
		"path":                "test.txt",
		"sha256":              "0000000000000000000000000000000000000000000000000000000000000000",
		"size_bytes":          0,
		"redacted":            false,
		"created_at":          "2024-01-01T00:00:00Z",
		"admission_authority": false,
	}
}

func TestReadIndex_JSONL(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "index.jsonl")
	entry1 := makeValidEntry()
	entry1["task_id"] = "task-a"
	entry2 := makeValidEntry()
	entry2["task_id"] = "task-b"
	b1, _ := json.Marshal(entry1)
	b2, _ := json.Marshal(entry2)
	data := string(b1) + "\n" + string(b2) + "\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadIndex(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["task_id"] != "task-a" {
		t.Fatalf("expected task-a, got %v", entries[0]["task_id"])
	}
	if entries[1]["task_id"] != "task-b" {
		t.Fatalf("expected task-b, got %v", entries[1]["task_id"])
	}
}

func TestReadIndex_JSONEnvelope(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "index.json")
	entry := makeValidEntry()
	entry["task_id"] = "task-a"
	envelope := map[string]any{
		"schema_version": "1",
		"task_id":        "task-env",
		"created_at":     "2024-01-01T00:00:00Z",
		"entry_count":    1,
		"index_hash":     "0000000000000000000000000000000000000000000000000000000000000000",
		"entries":        []map[string]any{entry},
	}
	data, _ := json.MarshalIndent(envelope, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadIndex(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["task_id"] != "task-a" {
		t.Fatalf("expected task-a, got %v", entries[0]["task_id"])
	}
}

func TestReadIndex_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadIndex(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadIndex_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.jsonl")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadIndex(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestValidateEntries_ValidEntry(t *testing.T) {
	entries := []map[string]any{makeValidEntry()}
	ok, errs := ValidateEntries(entries)
	if !ok {
		t.Fatalf("expected valid, got errors: %v", errs)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidateEntries_MissingRequiredField(t *testing.T) {
	entry := makeValidEntry()
	delete(entry, "task_id")
	entries := []map[string]any{entry}
	ok, errs := ValidateEntries(entries)
	if ok {
		t.Fatal("expected invalid")
	}
	if len(errs) == 0 {
		t.Fatal("expected errors")
	}
}

func TestValidateEntries_WrongSchemaVersion(t *testing.T) {
	entry := makeValidEntry()
	entry["schema_version"] = "2"
	entries := []map[string]any{entry}
	ok, errs := ValidateEntries(entries)
	if ok {
		t.Fatal("expected invalid")
	}
	if len(errs) == 0 {
		t.Fatal("expected errors")
	}
}
