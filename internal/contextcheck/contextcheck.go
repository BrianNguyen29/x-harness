package contextcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
