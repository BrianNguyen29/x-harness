package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractCheckNoViolations(t *testing.T) {
	tmpDir := t.TempDir()
	cleanFile := filepath.Join(tmpDir, "clean.go")
	if err := os.WriteFile(cleanFile, []byte("package main\n\nfunc main() {\n}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-rule
    description: "TODO rule"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", policy, tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No violations") {
		t.Fatalf("expected 'No violations' in output, got: %s", stdout.String())
	}
}

func TestContractCheckWithViolations(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(todoFile, []byte("package main\n// TODO: fix this\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-rule
    description: "TODO rule"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", policy, tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "violations: 1") {
		t.Fatalf("expected 'violations: 1' in output, got: %s", stdout.String())
	}
}

func TestContractCheckJSON(t *testing.T) {
	tmpDir := t.TempDir()
	todoFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(todoFile, []byte("package main\n// TODO: fix this\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
grep_rules:
  - id: todo-rule
    description: "TODO rule"
    file_pattern: "*.go"
    pattern: '(?i)//\s*TODO:'
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", policy, "--json", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}

	var result struct {
		OK           bool `json:"ok"`
		FilesScanned int  `json:"files_scanned"`
		Violations   []struct {
			RuleID string `json:"rule_id"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatalf("expected ok=false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].RuleID != "todo-rule" {
		t.Fatalf("expected rule id 'todo-rule', got %s", result.Violations[0].RuleID)
	}
}

func TestContractCheckDefaultPolicy(t *testing.T) {
	// The default policy is at policies/contract-oracle.yaml relative to repo root
	// In test context, we need to use an explicit policy
	tmpDir := t.TempDir()
	cleanFile := filepath.Join(tmpDir, "clean.go")
	if err := os.WriteFile(cleanFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Use the repo's actual default policy if it exists, otherwise create a simple one
	repoPolicy := filepath.Join("..", "..", "policies", "contract-oracle.yaml")
	policyPath := repoPolicy
	if _, err := os.Stat(policyPath); err != nil {
		// Fall back to creating a simple policy
		policy := filepath.Join(tmpDir, "policy.yaml")
		policyContent := `version: 1
grep_rules:
  - id: noop
    description: "noop rule"
    file_pattern: "*.go"
    pattern: '^$'
`
		if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
			t.Fatalf("failed to write policy: %v", err)
		}
		policyPath = policy
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", policyPath, "--json", tmpDir}, &stdout, &stderr)
	// Should succeed with the policy
	if code != ExitOK && code != ExitError {
		t.Fatalf("expected exit code %d or %d, got %d. stderr: %s", ExitOK, ExitError, code, stderr.String())
	}
}

func TestContractCheckPolicyNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", "/nonexistent/policy.yaml", "."}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "policy not found") {
		t.Fatalf("expected 'policy not found' error, got: %s", stderr.String())
	}
}

func TestContractUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "unknown"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown contract subcommand") {
		t.Fatalf("expected 'unknown contract subcommand' error, got: %s", stderr.String())
	}
}

func TestContractMissingPolicyArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "--policy requires a value") {
		t.Fatalf("expected '--policy requires a value' error, got: %s", stderr.String())
	}
}

func TestContractHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "help"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
}

func TestContractCheckNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage message, got: %s", stderr.String())
	}
}

func TestContractUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--unknown-flag"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("expected exit code %d, got %d", ExitUsage, code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("expected 'unknown flag' error, got: %s", stderr.String())
	}
}

func TestContractCheckDependencyRuleViolation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a Go file with a forbidden import
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "github.com/forbidden/package"
func main() {}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", policy, "--json", tmpDir}, &stdout, &stderr)
	if code != ExitError {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitError, code, stderr.String())
	}

	var result struct {
		OK         bool `json:"ok"`
		Violations []struct {
			RuleID string `json:"rule_id"`
		} `json:"violations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON: %v\noutput: %s", err, stdout.String())
	}
	if result.OK {
		t.Fatalf("expected ok=false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].RuleID != "no-forbidden" {
		t.Fatalf("expected rule id 'no-forbidden', got %s", result.Violations[0].RuleID)
	}
}

func TestContractCheckDependencyRuleNoViolation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a clean Go file
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "fmt"
func main() { fmt.Println("hello") }`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden
    description: "No forbidden packages"
    file_pattern: "*.go"
    forbidden_imports:
      - "github.com/forbidden"
`
	if err := os.WriteFile(policy, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", policy, tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d, got %d. stderr: %s", ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No violations") {
		t.Fatalf("expected 'No violations' in output, got: %s", stdout.String())
	}
}

func TestContractCheckDependencyRuleAllowedImport(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a Go file where forbidden is also in allowed
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main
import "github.com/forbidden/package"
func main() {}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	policy := filepath.Join(tmpDir, "policy.yaml")
	policyContent := `version: 1
dependency_rules:
  - id: no-forbidden
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

	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "check", "--policy", policy, tmpDir}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("expected exit code %d (allowed import suppresses), got %d. stderr: %s", ExitOK, code, stderr.String())
	}
}
