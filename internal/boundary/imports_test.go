package boundary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobMatcherLiteral(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		cand     string
		expected bool
	}{
		{name: "exact match", pattern: "src/ui/login.ts", cand: "src/ui/login.ts", expected: true},
		{name: "literal mismatch", pattern: "src/ui/login.ts", cand: "src/api/login.ts", expected: false},
		{name: "single star", pattern: "src/*/login.ts", cand: "src/ui/login.ts", expected: true},
		{name: "single star mismatch deeper", pattern: "src/*/login.ts", cand: "src/ui/x/login.ts", expected: false},
		{name: "double star mid", pattern: "src/**/login.ts", cand: "src/ui/x/y/login.ts", expected: true},
		{name: "double star tail", pattern: "src/**", cand: "src/ui/x/y/login.ts", expected: true},
		// `**` matches zero or more path segments (doublestar convention).
		{name: "double star empty tail matches src itself", pattern: "src/**", cand: "src", expected: true},
		{name: "double star empty tail matches src and child", pattern: "src/**", cand: "src/ui", expected: true},
		{name: "double star leading", pattern: "**/*.go", cand: "internal/cli/root.go", expected: true},
		{name: "double star leading root", pattern: "**/*.go", cand: "root.go", expected: true},
		{name: "no match on extension", pattern: "**/*.go", cand: "root.ts", expected: false},
		{name: "normalise backslashes", pattern: "src/ui/**", cand: "src\\ui\\login.ts", expected: true},
		{name: "normalise leading dot", pattern: "src/ui/**", cand: "./src/ui/login.ts", expected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newGlobMatcher(tt.pattern)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			got := m.Match(tt.cand)
			if got != tt.expected {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.cand, got, tt.expected)
			}
		})
	}
}

func TestGlobMatcherInvalid(t *testing.T) {
	for _, p := range []string{"", "/"} {
		if _, err := newGlobMatcher(p); err == nil {
			t.Errorf("expected error for %q", p)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := map[string]string{
		"foo.go":         "go",
		"foo.ts":         "typescript",
		"foo.tsx":        "typescript",
		"foo.js":         "javascript",
		"foo.mjs":        "javascript",
		"foo.cjs":        "javascript",
		"foo.jsx":        "javascript",
		"foo.py":         "",
		"foo.md":         "",
		"foo":            "",
		"path/to/foo.go": "go",
	}
	for cand, want := range tests {
		t.Run(cand, func(t *testing.T) {
			if got := detectLanguage(cand); got != want {
				t.Errorf("detectLanguage(%q) = %q, want %q", cand, got, want)
			}
		})
	}
}

func TestRuleAppliesToLanguage(t *testing.T) {
	if !ruleAppliesToLanguage(nil, "go") {
		t.Error("nil langs should match")
	}
	if ruleAppliesToLanguage(nil, "") {
		t.Error("nil langs should not match unknown language")
	}
	if !ruleAppliesToLanguage([]string{"go"}, "go") {
		t.Error("explicit match should pass")
	}
	if ruleAppliesToLanguage([]string{"go"}, "typescript") {
		t.Error("mismatched lang should not pass")
	}
}

func TestNormalisePathAndImport(t *testing.T) {
	if got := normalisePath("./src/ui/login.ts"); got != "src/ui/login.ts" {
		t.Errorf("normalisePath = %q", got)
	}
	if got := normalisePath(`src\ui\login.ts`); got != "src/ui/login.ts" {
		t.Errorf("normalisePath backslash = %q", got)
	}
	if got := normaliseImportPath(`"foo/bar"`); got != "foo/bar" {
		t.Errorf("double quote = %q", got)
	}
	if got := normaliseImportPath(`'foo/bar'`); got != "foo/bar" {
		t.Errorf("single quote = %q", got)
	}
	if got := normaliseImportPath("<foo/bar>"); got != "foo/bar" {
		t.Errorf("angle bracket = %q", got)
	}
}

func TestScanImportLinesJSTS(t *testing.T) {
	dir := t.TempDir()
	ts := filepath.Join(dir, "login.ts")
	if err := os.WriteFile(ts, []byte(`import foo from "internal/db";
import * as ns from "node:fs";
import { x } from "./local";
const c = require("lodash");
import "side-effect";
`), 0644); err != nil {
		t.Fatal(err)
	}
	lines, err := scanImportLines(ts)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	got := importTargets(lines)
	want := map[string]bool{
		"internal/db": true,
		"node:fs":     true,
		"./local":     true,
		"lodash":      true,
		"side-effect": true,
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d imports, got %d: %v", len(want), len(got), got)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected import %q", p)
		}
	}
}

func TestScanImportLinesGo(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, []byte(`package main

import "fmt"

import (
	"context"
	x "errors"
	_ "embed"
	. "strings"
)
`), 0644); err != nil {
		t.Fatal(err)
	}
	lines, err := scanImportLines(goFile)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	got := importTargets(lines)
	want := map[string]bool{
		"fmt":     true,
		"context": true,
		"errors":  true,
		"embed":   true,
		"strings": true,
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d imports, got %d: %v", len(want), len(got), got)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected import %q", p)
		}
	}
}

func TestScanImportLinesUnsupported(t *testing.T) {
	dir := t.TempDir()
	py := filepath.Join(dir, "main.py")
	if err := os.WriteFile(py, []byte("import os\n"), 0644); err != nil {
		t.Fatal(err)
	}
	lines, err := scanImportLines(py)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("expected no imports for .py, got %v", importTargets(lines))
	}
}

// importTargets returns a sorted set of import target strings for
// compact assertions.
func importTargets(lines []importLine) []string {
	seen := map[string]bool{}
	for _, l := range lines {
		seen[l.Target] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}
