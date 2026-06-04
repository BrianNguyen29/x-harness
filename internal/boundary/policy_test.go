package boundary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	policy := `version: 1
description: test
boundaries:
  - id: r1
    description: "rule one"
    from: "src/**"
    to_import: "internal/db/**"
    action: deny
    severity: warning
    applies_to_languages: [typescript]
`
	path := filepath.Join(dir, "boundaries.yaml")
	writeFile(t, path, policy)
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.Version != 1 {
		t.Errorf("version = %d", p.Version)
	}
	if len(p.Boundaries) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(p.Boundaries))
	}
	if p.Boundaries[0].ID != "r1" {
		t.Errorf("id = %q", p.Boundaries[0].ID)
	}
}

func TestLoadErrors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name   string
		policy string
	}{
		{
			name:   "missing version",
			policy: "boundaries: []",
		},
		{
			name: "wrong version",
			policy: `version: 2
boundaries: []`,
		},
		{
			name: "empty id",
			policy: `version: 1
boundaries:
  - id: ""
    from: x
    to_import: y
    action: deny
    severity: warning`,
		},
		{
			name: "duplicate id",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    to_import: y
    action: deny
    severity: warning
  - id: r1
    from: a
    to_import: b
    action: deny
    severity: warning`,
		},
		{
			name: "unknown action",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    to_import: y
    action: explode
    severity: warning`,
		},
		{
			name: "unknown severity",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    to_import: y
    action: deny
    severity: catastrophic`,
		},
		{
			name: "require_intermediate without intermediate",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    to_import: y
    action: require_intermediate
    severity: warning`,
		},
		{
			name: "unknown language",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    to_import: y
    action: deny
    severity: warning
    applies_to_languages: [python]`,
		},
		{
			name: "missing from",
			policy: `version: 1
boundaries:
  - id: r1
    to_import: y
    action: deny
    severity: warning`,
		},
		{
			name: "missing to_import",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    action: deny
    severity: warning`,
		},
		{
			name: "missing action",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    to_import: y
    severity: warning`,
		},
		{
			name: "missing severity",
			policy: `version: 1
boundaries:
  - id: r1
    from: x
    to_import: y
    action: deny`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, "p.yaml")
			writeFile(t, path, tt.policy)
			if _, err := Load(path); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestCheckNoPolicyIsNoop(t *testing.T) {
	res, err := Check(nil, "", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Errorf("expected OK=true when no policy, got false")
	}
	if res.PolicyLoaded {
		t.Errorf("expected PolicyLoaded=false")
	}
	if len(res.Warnings) == 0 {
		t.Errorf("expected warning when no policy loaded")
	}
}

func TestCheckSafeDefaultRule(t *testing.T) {
	dir := t.TempDir()
	// A rule whose `from` glob matches nothing in the dir.
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "ui-cannot-access-db",
				From:               "src/ui/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityWarning,
				AppliesToLanguages: []string{"typescript", "javascript"},
			},
		},
	}
	// Put a TS file that imports from internal/db but lives outside src/ui.
	writeFile(t, filepath.Join(dir, "src/api/login.ts"),
		`import { getUser } from "internal/db/users";
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Errorf("expected OK=true, got %v", res.Violations)
	}
	if res.FilesScanned == 0 {
		t.Errorf("expected files scanned > 0")
	}
}

func TestCheckDenyRuleFires(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "ui-cannot-access-db",
				From:               "src/ui/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript", "javascript"},
			},
		},
	}
	writeFile(t, filepath.Join(dir, "src/ui/login.ts"),
		`import { getUser } from "internal/db/users";
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Errorf("expected OK=false, got true")
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(res.Violations), res.Violations)
	}
	v := res.Violations[0]
	if v.RuleID != "ui-cannot-access-db" {
		t.Errorf("RuleID = %q", v.RuleID)
	}
	if v.Severity != SeverityHigh {
		t.Errorf("Severity = %q", v.Severity)
	}
	if v.File != "src/ui/login.ts" {
		t.Errorf("File = %q", v.File)
	}
	if v.Line != 1 {
		t.Errorf("Line = %d", v.Line)
	}
	if v.Import != "internal/db/users" {
		t.Errorf("Import = %q", v.Import)
	}
}

func TestCheckAllowListSuppresses(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "ui-cannot-access-db",
				From:               "src/ui/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript", "javascript"},
				Allow:              []string{"internal/db/public/**"},
			},
		},
	}
	writeFile(t, filepath.Join(dir, "src/ui/login.ts"),
		`import { getUser } from "internal/db/public/users";
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Errorf("expected OK=true (allow suppresses), got false: %+v", res.Violations)
	}
}

func TestCheckRequireIntermediateSatisfied(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:           "app-via-service",
				From:         "apps/**",
				ToImport:     "internal/data/**",
				Action:       ActionRequireIntermediate,
				Intermediate: "internal/services/**",
				Severity:     SeverityWarning,
			},
		},
	}
	writeFile(t, filepath.Join(dir, "apps/web/main.go"),
		`package main
import "internal/services/users"
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Errorf("expected OK=true when intermediate satisfied, got: %+v", res.Violations)
	}
}

func TestCheckRequireIntermediateUnsatisfied(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:           "app-via-service",
				From:         "apps/**",
				ToImport:     "internal/data/**",
				Action:       ActionRequireIntermediate,
				Intermediate: "internal/services/**",
				Severity:     SeverityHigh,
			},
		},
	}
	writeFile(t, filepath.Join(dir, "apps/web/main.go"),
		`package main
import "internal/data/users"
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Errorf("expected OK=false when intermediate not satisfied")
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(res.Violations))
	}
}

func TestCheckLanguageFilter(t *testing.T) {
	dir := t.TempDir()
	// Rule applies to typescript only; Go file should be ignored.
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "ui-cannot-access-db",
				From:               "src/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript"},
			},
		},
	}
	writeFile(t, filepath.Join(dir, "src/api/main.go"),
		`package main
import "internal/db/users"
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Errorf("expected OK=true for language filter (Go file should be ignored), got: %+v", res.Violations)
	}
}

func TestCheckDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "z-rule",
				From:               "src/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript", "javascript", "go"},
			},
			{
				ID:                 "a-rule",
				From:               "src/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript", "javascript", "go"},
			},
		},
	}
	writeFile(t, filepath.Join(dir, "src/b.ts"),
		`import "internal/db/users";
`)
	writeFile(t, filepath.Join(dir, "src/a.ts"),
		`import "internal/db/users";
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("expected OK=false")
	}
	if len(res.Violations) < 2 {
		t.Fatalf("expected at least 2 violations, got %d", len(res.Violations))
	}
	for i := 1; i < len(res.Violations); i++ {
		prev := res.Violations[i-1]
		cur := res.Violations[i]
		if prev.RuleID > cur.RuleID {
			t.Errorf("rule_id not sorted: %q > %q at %d", prev.RuleID, cur.RuleID, i)
		} else if prev.RuleID == cur.RuleID && prev.File > cur.File {
			t.Errorf("file not sorted within rule: %q > %q at %d", prev.File, cur.File, i)
		}
	}
}

func TestResultJSONShape(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "ui-cannot-access-db",
				From:               "src/ui/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript", "javascript"},
			},
		},
	}
	writeFile(t, filepath.Join(dir, "src/ui/login.ts"),
		`import "internal/db/users";
`)
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	// Stable, expected keys.
	for _, want := range []string{
		`"schema_version":"x-harness.boundary.v1"`,
		`"ok":false`,
		`"rule_id":"ui-cannot-access-db"`,
		`"file":"src/ui/login.ts"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in JSON: %s", want, s)
		}
	}
}

func TestCheckSkipsUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	// Create a directory where a file is expected — scanFile returns
	// an error, which Check should surface as a warning, not a fatal.
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "r",
				From:               "src/**",
				ToImport:           "x/**",
				Action:             ActionDeny,
				Severity:           SeverityWarning,
				AppliesToLanguages: []string{"typescript"},
			},
		},
	}
	// A path that exists but is a directory; scanImportLines will
	// fail to open it. Check should warn and continue.
	if err := os.MkdirAll(filepath.Join(dir, "src/a.ts"), 0755); err != nil {
		t.Fatal(err)
	}
	res, err := Check(policy, "", []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Errorf("expected OK=true with no violations; got %+v", res.Violations)
	}
	// No warning expected because collectCandidateFiles filters out
	// directories. The result is just an empty scan.
}

func TestViolationIDsUniqueSorted(t *testing.T) {
	// Sort stability: same input twice must produce the same order.
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "r",
				From:               "src/**",
				ToImport:           "x/**",
				Action:             ActionDeny,
				Severity:           SeverityWarning,
				AppliesToLanguages: []string{"typescript"},
			},
		},
	}
	writeFile(t, filepath.Join(dir, "src/z.ts"), `import "x/y";`)
	writeFile(t, filepath.Join(dir, "src/a.ts"), `import "x/y";`)
	res1, _ := Check(policy, "", []string{dir})
	res2, _ := Check(policy, "", []string{dir})
	if len(res1.Violations) != len(res2.Violations) {
		t.Fatalf("non-deterministic count: %d vs %d", len(res1.Violations), len(res2.Violations))
	}
	for i := range res1.Violations {
		if res1.Violations[i].File != res2.Violations[i].File {
			t.Errorf("non-deterministic order at %d: %q vs %q", i, res1.Violations[i].File, res2.Violations[i].File)
		}
	}
	// files should be in sorted order
	gotFiles := make([]string, len(res1.Violations))
	for i, v := range res1.Violations {
		gotFiles[i] = v.File
	}
	if !sort.StringsAreSorted(gotFiles) {
		t.Errorf("violations not sorted by file: %v", gotFiles)
	}
}

func TestListChangedFilesUnknownRepo(t *testing.T) {
	// Use a fresh temp dir that is NOT a git repo.
	dir := t.TempDir()
	files, err := ListChangedFiles(dir)
	if err == nil {
		t.Errorf("expected error when not a git repo, got files=%v", files)
	}
}

// TestCheckFilesAnchoredToBase exercises the regression for
// `xh boundary check --changed`: when a file list is passed with an
// explicit base (the repo root), the rule's `from` glob must match
// against the path relative to that base, not against the basename
// that collectEntries would otherwise produce.
func TestCheckFilesAnchoredToBase(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "ui-cannot-access-db",
				From:               "src/ui/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript"},
			},
		},
	}
	login := filepath.Join(dir, "src", "ui", "login.ts")
	writeFile(t, login, `import "internal/db/users";`)

	res, err := CheckFiles(policy, "", dir, []string{login})
	if err != nil {
		t.Fatalf("CheckFiles: %v", err)
	}
	if res.OK {
		t.Fatalf("expected OK=false, got true")
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(res.Violations), res.Violations)
	}
	v := res.Violations[0]
	if v.File != "src/ui/login.ts" {
		t.Errorf("File = %q, want %q", v.File, "src/ui/login.ts")
	}
	if v.Import != "internal/db/users" {
		t.Errorf("Import = %q", v.Import)
	}
}

// TestCheckFilesSkipsFilesOutsideBase confirms that files which are
// not children of the base directory still get a repo-relative path
// (via filepath.Rel) but rules that don't match the rel path are
// simply not flagged.
func TestCheckFilesSkipsFilesOutsideBase(t *testing.T) {
	dir := t.TempDir()
	policy := &Policy{
		Version: 1,
		Boundaries: []Rule{
			{
				ID:                 "ui-cannot-access-db",
				From:               "src/ui/**",
				ToImport:           "internal/db/**",
				Action:             ActionDeny,
				Severity:           SeverityHigh,
				AppliesToLanguages: []string{"typescript"},
			},
		},
	}
	outside := filepath.Join(dir, "src", "api", "login.ts")
	writeFile(t, outside, `import "internal/db/users";`)

	res, err := CheckFiles(policy, "", dir, []string{outside})
	if err != nil {
		t.Fatalf("CheckFiles: %v", err)
	}
	if !res.OK {
		t.Errorf("expected OK=true (file outside `from` glob), got %+v", res.Violations)
	}
	if len(res.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(res.Violations))
	}
}

// TestCheckFilesNoPolicyIsNoop mirrors the Check behaviour: when the
// policy is nil, CheckFiles should return OK=true with the same
// warning text.
func TestCheckFilesNoPolicyIsNoop(t *testing.T) {
	res, err := CheckFiles(nil, "", "/tmp", []string{"/tmp/x.ts"})
	if err != nil {
		t.Fatalf("CheckFiles: %v", err)
	}
	if !res.OK {
		t.Errorf("expected OK=true when no policy")
	}
	if res.PolicyLoaded {
		t.Errorf("expected PolicyLoaded=false")
	}
	if len(res.Warnings) == 0 {
		t.Errorf("expected warning when no policy loaded")
	}
}
