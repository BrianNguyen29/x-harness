package boundary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// TestBundledPolicyMatchesSchema verifies that the example policy
// shipped at policies/boundaries.yaml conforms to the JSON Schema
// declared at schemas/boundary-policy.schema.json. The check is
// pinned to the schema compiler used elsewhere in the repo so the
// "shape" guarantee stays consistent with the rest of the harness.
func TestBundledPolicyMatchesSchema(t *testing.T) {
	repoRoot := findRepoRoot(t)
	schemaPath := filepath.Join(repoRoot, "schemas", "boundary-policy.schema.json")
	policyPath := filepath.Join(repoRoot, "policies", "boundaries.yaml")

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	var doc map[string]interface{}
	if err := loader.LoadYAML(policyPath, &doc); err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if err := schema.Validate(doc); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

// findRepoRoot walks up from the test's working directory to find the
// repository root (the directory containing go.mod). Tests are skipped
// when the marker file is not found so the suite does not fail in
// unsupported environments.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := filepath.Abs(".")
	if err != nil {
		t.Skip("cannot resolve cwd")
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Skip("repo root not found")
		}
		wd = parent
	}
}
