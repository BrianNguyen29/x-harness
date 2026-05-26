package schema

import (
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator holds a compiled JSON Schema and can validate documents against it.
type Validator struct {
	schema *jsonschema.Schema
}

// Compile loads and compiles a JSON Schema from the given file path.
func Compile(path string) (*Validator, error) {
	compiler := jsonschema.NewCompiler()
	s, err := compiler.Compile(path)
	if err != nil {
		return nil, err
	}
	return &Validator{schema: s}, nil
}

// Validate checks whether doc conforms to the compiled schema.
func (v *Validator) Validate(doc any) error {
	return v.schema.Validate(doc)
}
