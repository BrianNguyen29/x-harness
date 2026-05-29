package contextcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	ManagedBegin = "<!-- BEGIN X-HARNESS MANAGED CONTEXT -->"
	ManagedEnd   = "<!-- END X-HARNESS MANAGED CONTEXT -->"
)

func CanonicalContext() string {
	return `# x-harness Canonical Context

- Completion is admitted, not claimed.
- Verifier is read-only.
- Success is the only accepted outcome.
- Canonical tiers: light, standard, deep.
- PGV is advisory-only.

## Source-of-Truth Reading Order

The managed context block in AGENTS.md is authoritative. Files are read in this order:

1. AGENTS.md (managed block)
1. X_HARNESS.md
1. policies/admission.yaml
1. policies/recovery.yaml
1. policies/intake.yaml
1. schemas/completion-card.schema.json

## Rules

### Completion is admitted, not claimed
Agents may propose completion but cannot self-admit. A completion card with ` + "`" + `claim.fix_status: fixed` + "`" + ` is only a completion candidate. Compatibility subagent returns may use ` + "`" + `result.fix_status` + "`" + `.

### Verifier is read-only
The verifier may inspect files, evidence, diffs, and trace events. It must not edit source files or repair the work product while verifying.

### Success is the only accepted outcome
` + "`" + `admission.outcome: success` + "`" + ` and ` + "`" + `acceptance_status: accepted` + "`" + ` are required for admission. All other outcomes are withheld.

### Canonical tiers
Use only ` + "`" + `light` + "`" + `, ` + "`" + `standard` + "`" + `, and ` + "`" + `deep` + "`" + `. Do not use ` + "`" + `small` + "`" + `, ` + "`" + `medium` + "`" + `, or ` + "`" + `large` + "`" + ` in active runtime handoffs.

### PGV is advisory-only
Pre-gate validation (PGV) advice never overrides the verify gate and never grants admission authority by default.`
}

func ContextHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])[:16]
}

func ExtractManagedBlock(content string) (string, bool) {
	beginIndex := strings.Index(content, ManagedBegin)
	endIndex := strings.Index(content, ManagedEnd)
	if beginIndex == -1 || endIndex == -1 || endIndex < beginIndex {
		return "", false
	}
	return content[beginIndex : endIndex+len(ManagedEnd)], true
}

func ValidateManagedBlock(content string) (bool, string) {
	block, ok := ExtractManagedBlock(content)
	if !ok {
		return false, "AGENTS.md missing managed context block"
	}

	idx := strings.Index(block, "<!-- context-hash: ")
	if idx == -1 {
		return false, "AGENTS.md managed block missing context-hash"
	}
	hashStart := idx + len("<!-- context-hash: ")
	hashEnd := strings.Index(block[hashStart:], " -->")
	if hashEnd == -1 {
		return false, "AGENTS.md managed block missing context-hash"
	}
	actualHash := block[hashStart : hashStart+hashEnd]

	currentContext := CanonicalContext()
	expectedHash := ContextHash(currentContext)

	if actualHash != expectedHash {
		return false, fmt.Sprintf("AGENTS.md context hash stale: expected %s, found %s", expectedHash, actualHash)
	}

	lines := strings.Split(block, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ManagedBegin || trimmed == ManagedEnd || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		filtered = append(filtered, line)
	}
	actualContext := strings.TrimSpace(strings.Join(filtered, "\n"))

	if actualContext != strings.TrimSpace(currentContext) {
		return false, "AGENTS.md managed context body differs from canonical context"
	}

	return true, "AGENTS.md managed context block is fresh"
}

var markdownLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

// CheckDeadLinks scans docs/*.md for repo-local markdown links and returns
// a slice of "file: line: target" strings for any link target that does not
// resolve to an existing file within the repository.
func CheckDeadLinks(root string) []string {
	docsDir := filepath.Join(root, "docs")
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return []string{fmt.Sprintf("docs: %v", err)}
	}

	var dead []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(docsDir, entry.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			dead = append(dead, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		fileDir := filepath.Dir(path)
		lines := strings.Split(string(b), "\n")
		for lineNum, line := range lines {
			matches := markdownLinkRe.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) < 3 {
					continue
				}
				target := strings.TrimSpace(m[2])
				if isExternalLink(target) {
					continue
				}
				resolved := resolveLinkTarget(root, fileDir, target)
				if _, err := os.Stat(resolved); err != nil {
					dead = append(dead, fmt.Sprintf("%s:%d: %s", entry.Name(), lineNum+1, target))
				}
			}
		}
	}
	return dead
}

func isExternalLink(target string) bool {
	return strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "mailto:") ||
		strings.HasPrefix(target, "#")
}

func resolveLinkTarget(root, fileDir, target string) string {
	// Remove leading ./ if present
	target = strings.TrimPrefix(target, "./")
	if filepath.IsAbs(target) {
		return filepath.Join(root, target)
	}
	// Resolve relative to the source file's directory
	return filepath.Join(fileDir, target)
}
