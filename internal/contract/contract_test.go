package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckNoViolations(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a clean file that should not match any rules
	cleanFile := filepath.Join(tmpDir, "clean.go")
	if err := os.WriteFile(cleanFile, []byte("package main\n\nfunc main() {\n}\n"), 0644); err != nil {
		t.Fatalf("failed to write clean file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-without-owner
    description: "TODO without owner"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK=true, got false")
	}
	if result.FilesScanned == 0 {
		t.Fatalf("expected at least 1 file scanned")
	}
	if len(result.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(result.Violations))
	}
}

func TestCheckWithViolations(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with TODO
	todoFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(todoFile, []byte("package main\n\n// TODO: fix this\nfunc main() {\n}\n"), 0644); err != nil {
		t.Fatalf("failed to write todo file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-without-owner
    description: "TODO without owner"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected OK=false, got true")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].RuleID != "todo-without-owner" {
		t.Fatalf("expected rule id 'todo-without-owner', got %s", result.Violations[0].RuleID)
	}
}

func TestCheckExclude(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file in a path that should be excluded (mimics vendor/ or packages/cli/examples)
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}
	excludeFile := filepath.Join(vendorDir, "main.go")
	if err := os.WriteFile(excludeFile, []byte("package main\n\n// TODO: fix this\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write exclude file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-in-go
    description: "TODO in go file"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
    exclude:
      - "vendor"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// File in vendor/ path should be excluded
	if len(result.Violations) != 0 {
		t.Fatalf("expected 0 violations (file in vendor/ should be excluded), got %d", len(result.Violations))
	}
}

func TestCheckFilePatternMatch(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a .txt file that should NOT match *.go pattern
	txtFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("TODO: fix this later\n"), 0644); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}
	// Create a .go file that SHOULD match
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n// TODO: fix\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-in-go
    description: "TODO in go file"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected violations")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if !strings.HasSuffix(result.Violations[0].File, ".go") {
		t.Fatalf("expected violation in .go file, got %s", result.Violations[0].File)
	}
}

func TestCheckInvalidRegex(t *testing.T) {
	tmpDir := t.TempDir()
	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: bad-regex
    description: "bad regex"
    file_pattern: "*.go"
    pattern: '(?i)[invalid'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	_, err := Check(policy, []string{tmpDir})
	if err == nil {
		t.Fatalf("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "invalid pattern") {
		t.Fatalf("expected 'invalid pattern' error, got: %v", err)
	}
}

func TestCheckInvalidPolicyVersion(t *testing.T) {
	tmpDir := t.TempDir()
	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 99
grep_rules: []
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	_, err := Check(policy, []string{tmpDir})
	if err == nil {
		t.Fatalf("expected error for invalid version")
	}
	if !strings.Contains(err.Error(), "unsupported policy version") {
		t.Fatalf("expected 'unsupported policy version' error, got: %v", err)
	}
}

func TestCheckPolicyNotFound(t *testing.T) {
	_, err := Check("/nonexistent/policy.yaml", []string{"."})
	if err == nil {
		t.Fatalf("expected error for nonexistent policy")
	}
}

func TestCheckEmptyGrepRules(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with content that would match if rules existed
	todoFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(todoFile, []byte("package main\n// TODO: fix\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules: []
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK=true with empty rules")
	}
}

func TestResultToJSON(t *testing.T) {
	result := &Result{
		OK:           true,
		Policy:       "test-policy.yaml",
		FilesScanned: 5,
		Violations:   []Violation{},
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["ok"] != true {
		t.Fatalf("expected ok=true in JSON")
	}
	if parsed["files_scanned"].(float64) != 5 {
		t.Fatalf("expected files_scanned=5 in JSON")
	}
}

func TestCheckGlobPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	// Create files with different extensions
	for _, ext := range []string{".go", ".ts", ".js", ".md"} {
		fname := filepath.Join(tmpDir, "file"+ext)
		content := "// TODO: fix\n"
		if err := os.WriteFile(fname, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", ext, err)
		}
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	// filepath.Match supports: *.go, *.ts, * (any file)
	// Test multiple rules for different patterns
	policyContent := `version: 1
grep_rules:
  - id: todo-in-go
    description: "TODO in Go"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
  - id: todo-in-ts
    description: "TODO in TypeScript"
    file_pattern: "*.ts"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should match .go and .ts files
	if len(result.Violations) != 2 {
		t.Fatalf("expected 2 violations (.go and .ts), got %d", len(result.Violations))
	}
}

func TestCheckMessageFallback(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(todoFile, []byte("package main\n// TODO: fix this\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	// Policy without explicit message - should fall back to description
	policyContent := `version: 1
grep_rules:
  - id: todo-no-message
    description: "TODO without owner"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Message != "TODO without owner" {
		t.Fatalf("expected message to fallback to description, got: %s", result.Violations[0].Message)
	}
}

func TestCheckMessageOverride(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(todoFile, []byte("package main\n// TODO: fix this\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-with-message
    description: "TODO without owner"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
    message: "TODO must include an owner or ticket number"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Message != "TODO must include an owner or ticket number" {
		t.Fatalf("expected custom message, got: %s", result.Violations[0].Message)
	}
}

func TestDependencyRuleViolation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a Go file with a forbidden import
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main

import "github.com/forbidden/package"

func main() {
}
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected OK=false, got true")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].RuleID != "no-forbidden-packages" {
		t.Fatalf("expected rule id 'no-forbidden-packages', got %s", result.Violations[0].RuleID)
	}
	if !strings.Contains(result.Violations[0].Snippet, "github.com/forbidden") {
		t.Fatalf("expected snippet to contain forbidden import, got %s", result.Violations[0].Snippet)
	}
}

func TestDependencyRuleNoViolation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a clean Go file
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK=true, got false")
	}
	if len(result.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(result.Violations))
	}
}

func TestDependencyRuleIndentedGoImport(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a Go file with indented import (common in real Go code)
	goFile := filepath.Join(tmpDir, "main.go")
	// Note: "  import" has two leading spaces to test indented import detection
	if err := os.WriteFile(goFile, []byte(`package main

  import "fmt"

func main() {
	fmt.Println("hello")
}
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK=true, got false. violations: %+v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("expected 0 violations for indented import, got %d", len(result.Violations))
	}
}

func TestDependencyRuleIndentedForbiddenGoImport(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a Go file with indented forbidden import
	goFile := filepath.Join(tmpDir, "main.go")
	// Note: "  import" has two leading spaces and the import is forbidden
	if err := os.WriteFile(goFile, []byte(`package main

  import "github.com/forbidden"

func main() {
}
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected OK=false for indented forbidden import, got true")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation for indented forbidden import, got %d", len(result.Violations))
	}
	if result.Violations[0].RuleID != "no-forbidden-packages" {
		t.Fatalf("expected rule id 'no-forbidden-packages', got %s", result.Violations[0].RuleID)
	}
}

func TestDependencyRuleAllowedImport(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a Go file with a forbidden import but allowed sub-path
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main

import "github.com/forbidden/package"

func main() {
}
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
    allowed_imports:
      - "github.com/forbidden/package"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be OK because the allowed_imports exception matches
	if !result.OK {
		t.Fatalf("expected OK=true (allowed import suppresses), got false. violations: %+v", result.Violations)
	}
}

func TestDependencyRuleFilePattern(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a .txt file with import that would match if .go file
	txtFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte(`import "github.com/forbidden/package"`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create a .go file that should match
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "github.com/forbidden/package"
func main() {}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-go
    description: "No forbidden packages in Go files"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected violations")
	}
	// Should only match the .go file, not the .txt file
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if !strings.HasSuffix(result.Violations[0].File, ".go") {
		t.Fatalf("expected violation in .go file, got %s", result.Violations[0].File)
	}
}

func TestDependencyRuleExclude(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file in a path that should be excluded
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}
	excludeFile := filepath.Join(vendorDir, "main.go")
	if err := os.WriteFile(excludeFile, []byte(`package main
import "github.com/forbidden/package"
func main() {}`), 0644); err != nil {
		t.Fatalf("failed to write exclude file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
    exclude:
      - "vendor"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// File in vendor/ path should be excluded
	if len(result.Violations) != 0 {
		t.Fatalf("expected 0 violations (file in vendor/ should be excluded), got %d", len(result.Violations))
	}
}

func TestDependencyRuleMixedWithGrep(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with both a TODO comment and a forbidden import
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(`package main
// TODO: fix this
import "github.com/forbidden/package"

func main() {
}
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-rule
    description: "TODO without owner"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
dependency_rules:
  - id: no-forbidden-packages
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected violations")
	}
	// Should have both grep and dependency violations
	if len(result.Violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(result.Violations))
	}
	ruleIDs := make(map[string]bool)
	for _, v := range result.Violations {
		ruleIDs[v.RuleID] = true
	}
	if !ruleIDs["todo-rule"] {
		t.Fatalf("expected grep violation for 'todo-rule'")
	}
	if !ruleIDs["no-forbidden-packages"] {
		t.Fatalf("expected dependency violation for 'no-forbidden-packages'")
	}
}

func TestDependencyRuleTypeScriptImport(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a TypeScript file with import
	tsFile := filepath.Join(tmpDir, "main.ts")
	if err := os.WriteFile(tsFile, []byte(`import { something } from 'forbidden-package';
export default something;
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-ts
    description: "No forbidden packages in TypeScript"
    file_pattern: "*.ts"
    forbidden_imports:
      - "forbidden-package"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected violations")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if !strings.Contains(result.Violations[0].Snippet, "forbidden-package") {
		t.Fatalf("expected snippet to contain forbidden import, got %s", result.Violations[0].Snippet)
	}
}

func TestDependencyRuleJavaScriptRequire(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a JavaScript file with require
	jsFile := filepath.Join(tmpDir, "main.js")
	if err := os.WriteFile(jsFile, []byte(`const foo = require('forbidden-lib');
module.exports = foo;
`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden-js
    description: "No forbidden packages in JavaScript"
    file_pattern: "*.js"
    forbidden_imports:
      - "forbidden-lib"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OK {
		t.Fatalf("expected violations")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if !strings.Contains(result.Violations[0].Snippet, "forbidden-lib") {
		t.Fatalf("expected snippet to contain forbidden import, got %s", result.Violations[0].Snippet)
	}
}

func TestDependencyRuleEmptyRules(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with forbidden import
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "github.com/forbidden/package"
func main() {}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules: []
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Check(policy, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected OK=true with empty dependency_rules, got false")
	}
}
