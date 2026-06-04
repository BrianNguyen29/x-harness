package boundary

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// importLine is a single import statement extracted from a file. The
// raw line is preserved for the snippet field; Target is the quoted
// import path (or module path for Go) with surrounding quotes stripped.
type importLine struct {
	Line   int
	Raw    string
	Target string
}

// jsImportRe matches JS/TS import statements. We deliberately avoid
// parsing: the pattern extracts the import target. The `m` flag lets `^`
// match at the start of each scanned line.
//
// Covers: import "side-effect", import X from "Y", import {x} from "Y",
// import * as X from "Y", export {x} from "Y", export * from "Y",
// const x = require("Y"), var x = require("Y").
var jsImportRe = regexp.MustCompile(
	`(?m)^\s*(?:import\s+(?:.*?\s+from\s+)?|export\s+.*?\s+from\s+|(?:const|var)\s+\w+\s*=\s*require\s*\()\s*["']([^"']+)["']`,
)

// goSingleImportRe matches a Go single-line import. Group 1 is the import path.
//
//	import "fmt"
//	import alias "fmt"
var goSingleImportRe = regexp.MustCompile(`^\s*import\s+(?:\w+\s+)?"([^"]+)"`)

// goImportBlockStartRe detects the start of an `import ( ... )` block.
var goImportBlockStartRe = regexp.MustCompile(`^\s*import\s*\(\s*$`)

// goImportPathRe matches a single import path inside a Go import block.
var goImportPathRe = regexp.MustCompile(`"([^"]+)"`)

// scanImportLines extracts import targets from a file. The function is
// intentionally line-oriented so the V1 surface is predictable: if a
// line is not detected as an import, it is silently skipped. We do not
// fail on ambiguous lines — false negatives are preferred to false
// positives in V1.
func scanImportLines(path string) ([]importLine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return scanGoImports(string(data)), nil
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx":
		return scanJSImports(string(data)), nil
	default:
		// Unsupported file type: no imports to scan.
		return nil, nil
	}
}

// scanJSImports walks the file content line by line and extracts import
// targets using jsImportRe. The `m` flag in the regex is what makes
// `^` match per-line.
func scanJSImports(content string) []importLine {
	matches := jsImportRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []importLine
	lines := strings.Split(content, "\n")
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		// Find the line number by counting newlines up to m[0].
		lineNum := 1 + strings.Count(content[:m[0]], "\n")
		target := content[m[2]:m[3]]
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		raw := ""
		if lineNum-1 < len(lines) {
			raw = lines[lineNum-1]
		}
		out = append(out, importLine{
			Line:   lineNum,
			Raw:    raw,
			Target: target,
		})
	}
	return out
}

// scanGoImports handles both single-line imports and the
// `import ( ... )` block. We track block depth so we can attribute the
// block to the line of the opening `import (` and aggregate the inner
// paths.
func scanGoImports(content string) []importLine {
	lines := strings.Split(content, "\n")
	var out []importLine
	seen := map[string]bool{}
	var blockOpenLine int
	var blockPaths []string
	var blockRawLines []string
	inBlock := false

	flush := func() {
		if !inBlock {
			return
		}
		raw := strings.Join(blockRawLines, "\n")
		for _, p := range blockPaths {
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, importLine{
				Line:   blockOpenLine,
				Raw:    raw,
				Target: p,
			})
		}
		inBlock = false
		blockPaths = nil
		blockRawLines = nil
	}

	for i, line := range lines {
		lineNum := i + 1
		if inBlock {
			blockRawLines = append(blockRawLines, line)
			if strings.Contains(line, ")") {
				flush()
				continue
			}
			for _, m := range goImportPathRe.FindAllStringSubmatch(line, -1) {
				if len(m) >= 2 {
					blockPaths = append(blockPaths, m[1])
				}
			}
			continue
		}
		if goImportBlockStartRe.MatchString(line) {
			inBlock = true
			blockOpenLine = lineNum
			blockRawLines = []string{line}
			if strings.Contains(line, ")") {
				flush()
			}
			continue
		}
		if m := goSingleImportRe.FindStringSubmatch(line); len(m) >= 2 {
			target := m[1]
			if target == "" || seen[target] {
				continue
			}
			seen[target] = true
			out = append(out, importLine{
				Line:   lineNum,
				Raw:    line,
				Target: target,
			})
		}
	}
	// Unterminated block: still flush what we have.
	flush()
	return out
}

// detectLanguage returns the boundary language identifier for a file
// path, or "" when the extension is not in V1 scope.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".go":
		return "go"
	}
	return ""
}

// ruleAppliesToLanguage returns true when langs is empty (rule applies
// to all supported languages) or when it contains the candidate's
// detected language.
func ruleAppliesToLanguage(langs []string, language string) bool {
	if len(langs) == 0 {
		return language != "" // only apply to known language files
	}
	for _, l := range langs {
		if l == language {
			return true
		}
	}
	return false
}

// bufio is imported for symmetry; the current implementation uses
// strings + regex instead of bufio to keep Go blocks easy to reason
// about. The blank assignment keeps the import live if a future change
// reintroduces line-buffer scanning.
var _ = bufio.NewScanner
