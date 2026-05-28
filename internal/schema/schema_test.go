package schema

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/loader"
)

func TestCompileCompletionCardSchema(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	_, err := Compile(path)
	if err != nil {
		t.Fatalf("expected schema to compile, got error: %v", err)
	}
}

func TestCompileAdmissionCardSchema(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "admission-card.schema.json")
	_, err := Compile(path)
	if err != nil {
		t.Fatalf("expected schema to compile, got error: %v", err)
	}
}

func TestCompileAdapterMatrixSchema(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "adapter-matrix.schema.json")
	_, err := Compile(path)
	if err != nil {
		t.Fatalf("expected schema to compile, got error: %v", err)
	}
}

func TestCompileReleaseEvidenceSchema(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "release-evidence.schema.json")
	_, err := Compile(path)
	if err != nil {
		t.Fatalf("expected schema to compile, got error: %v", err)
	}
}

func TestValidGoldenCardPasses(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	v, err := Compile(schemaPath)
	if err != nil {
		t.Fatalf("expected schema to compile, got error: %v", err)
	}

	cardPath := filepath.Join("..", "..", "examples", "golden", "success-light", "completion-card.yaml")
	var doc any
	if err := loader.LoadDocument(cardPath, &doc); err != nil {
		t.Fatalf("expected to load card, got error: %v", err)
	}

	if err := v.Validate(doc); err != nil {
		t.Fatalf("expected valid card to pass validation, got error: %v", err)
	}
}

func TestInvalidGoldenCardFails(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	v, err := Compile(schemaPath)
	if err != nil {
		t.Fatalf("expected schema to compile, got error: %v", err)
	}

	cardPath := filepath.Join("..", "..", "examples", "golden", "failed-invalid-status", "completion-card.yaml")
	var doc any
	if err := loader.LoadDocument(cardPath, &doc); err != nil {
		t.Fatalf("expected to load card, got error: %v", err)
	}

	if err := v.Validate(doc); err == nil {
		t.Fatal("expected invalid card to fail validation, but it passed")
	}
}

func FuzzValidate(f *testing.F) {
	schemaPath := filepath.Join("..", "..", "schemas", "completion-card.schema.json")
	v, err := Compile(schemaPath)
	if err != nil {
		f.Fatalf("expected schema to compile, got error: %v", err)
	}

	f.Add([]byte(`{"schema_version":"1.0","task_id":"t","tier":"light","owner":"o","accountable":"a","claim":{"fix_status":"fixed"},"verification":{"status":"passed"},"admission":{"outcome":"success"},"acceptance_status":"accepted","handoff":{"next_action":"n","owner":"u"}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"schema_version":"1.0"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Skip()
		}
		_ = v.Validate(doc)
	})
}
