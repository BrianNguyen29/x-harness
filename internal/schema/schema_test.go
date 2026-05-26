package schema

import (
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
