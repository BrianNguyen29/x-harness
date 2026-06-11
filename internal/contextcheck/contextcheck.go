package contextcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
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
The verifier may inspect files, tasks, stories, templates, returns, evidence, diffs, command output, and trace events. It must not edit source files or repair the work product while verifying.

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
	content = strings.ReplaceAll(content, "\r\n", "\n")
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

// CheckDeadLinks scans repo-wide user-facing markdown for repo-local markdown
// links and returns a slice of "file: line: target" strings for any link
// target that does not resolve to an existing file within the repository.
func CheckDeadLinks(root string) []string {
	includePaths := []string{
		".",
		"docs",
		"examples",
		"tests",
		"adapters",
		"templates",
		"packages/cli",
	}

	excludeDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"dist":         true,
		"coverage":     true,
		"build":        true,
		"vendor":       true,
		".x-harness":   true,
	}

	var files []string
	var dead []string

	for _, rel := range includePaths {
		full := filepath.Join(root, rel)
		info, err := os.Stat(full)
		if err != nil {
			// Path does not exist; skip
			continue
		}
		if !info.IsDir() {
			if filepath.Ext(full) == ".md" {
				files = append(files, full)
			}
			continue
		}

		if rel == "." {
			entries, err := os.ReadDir(full)
			if err != nil {
				dead = append(dead, fmt.Sprintf("%s: %v", rel, err))
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
					continue
				}
				files = append(files, filepath.Join(full, entry.Name()))
			}
		} else {
			err := filepath.WalkDir(full, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					if excludeDirs[d.Name()] {
						return filepath.SkipDir
					}
					return nil
				}
				if filepath.Ext(path) == ".md" {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				dead = append(dead, fmt.Sprintf("%s: %v", rel, err))
			}
		}
	}

	sort.Strings(files)

	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			relPath, _ := filepath.Rel(root, path)
			dead = append(dead, fmt.Sprintf("%s: %v", relPath, err))
			continue
		}
		fileDir := filepath.Dir(path)
		relPath, _ := filepath.Rel(root, path)
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
					dead = append(dead, fmt.Sprintf("%s:%d: %s", relPath, lineNum+1, target))
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
		strings.HasPrefix(target, "file://") ||
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

// ValidateManagedBlockGeneric validates a managed block with configurable markers.
// It extracts the block between beginMarker and endMarker, finds the hash after hashPrefix,
// and verifies that the hash of the block body (excluding markers and HTML comments) matches.
func ValidateManagedBlockGeneric(content, beginMarker, endMarker, hashPrefix string) (bool, string) {
	actualHash, actualBody, note, ok := extractManagedBlockBody(content, beginMarker, endMarker, hashPrefix)
	if !ok {
		return false, note
	}
	expectedHash := ContextHash(actualBody)

	if actualHash != expectedHash {
		return false, fmt.Sprintf("hash stale: expected %s, found %s", expectedHash, actualHash)
	}

	return true, "managed block is fresh"
}

func ValidateManagedContractBlock(content, beginMarker, endMarker, hashPrefix string) (bool, string) {
	valid, note := ValidateManagedBlockGeneric(content, beginMarker, endMarker, hashPrefix)
	if !valid {
		return false, note
	}
	_, actualBody, note, ok := extractManagedBlockBody(content, beginMarker, endMarker, hashPrefix)
	if !ok {
		return false, note
	}
	for _, required := range canonicalContractSnippets() {
		if strings.Contains(actualBody, required) {
			return true, "managed contract block is fresh"
		}
	}
	return false, "managed contract body missing canonical snippets"
}

func canonicalContractSnippets() []string {
	return []string{
		"Completion is admitted, not claimed.",
		"Verifier is read-only.",
		"Success is the only accepted outcome.",
		"Canonical tiers: light, standard, deep.",
		"PGV is advisory-only.",
		"Completion cards use claim.fix_status as the canonical fix-status field.",
		"Accepted completion requires `admission.outcome: success` and `acceptance_status: accepted`.",
	}
}

// ValidateManagedBlockExpected validates both the hash and body against an
// authoritative expected body. Context blocks use this path so a stale block
// cannot pass merely because its hash matches its own outdated content.
func ValidateManagedBlockExpected(content, beginMarker, endMarker, hashPrefix, expectedBody string) (bool, string) {
	actualHash, actualBody, note, ok := extractManagedBlockBody(content, beginMarker, endMarker, hashPrefix)
	if !ok {
		return false, note
	}
	expectedBody = strings.TrimSpace(strings.ReplaceAll(expectedBody, "\r\n", "\n"))
	expectedHash := ContextHash(expectedBody)
	if actualHash != expectedHash {
		return false, fmt.Sprintf("hash stale: expected %s, found %s", expectedHash, actualHash)
	}
	if actualBody != expectedBody {
		return false, "managed block body differs from canonical context"
	}
	return true, "managed block is fresh"
}

func extractManagedBlockBody(content, beginMarker, endMarker, hashPrefix string) (string, string, string, bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	beginIndex := strings.Index(content, beginMarker)
	endIndex := strings.Index(content, endMarker)
	if beginIndex == -1 || endIndex == -1 || endIndex < beginIndex {
		return "", "", "missing managed block", false
	}
	block := content[beginIndex : endIndex+len(endMarker)]

	idx := strings.Index(block, hashPrefix)
	if idx == -1 {
		return "", "", "managed block missing hash", false
	}
	hashStart := idx + len(hashPrefix)
	hashEnd := strings.Index(block[hashStart:], " -->")
	if hashEnd == -1 {
		return "", "", "managed block missing hash", false
	}
	actualHash := block[hashStart : hashStart+hashEnd]

	lines := strings.Split(block, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == beginMarker || trimmed == endMarker || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		filtered = append(filtered, line)
	}
	actualBody := strings.TrimSpace(strings.Join(filtered, "\n"))
	return actualHash, actualBody, "", true
}

// RegistryEntry describes a single managed block in the registry.
type RegistryEntry struct {
	Path        string `yaml:"path"`
	Type        string `yaml:"type"`
	BeginMarker string `yaml:"begin_marker"`
	EndMarker   string `yaml:"end_marker"`
	HashPrefix  string `yaml:"hash_prefix"`
}

// Registry describes the managed-blocks registry.
type Registry struct {
	Version string          `yaml:"version"`
	Blocks  []RegistryEntry `yaml:"blocks"`
}

// ResolveRegistryEntryPath resolves an entry path and rejects paths outside
// the workspace root.
func ResolveRegistryEntryPath(root, entryPath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(entryPath)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace root")
	}
	return targetAbs, nil
}

// ReadRegistry loads the managed-block registry from a workspace.
func ReadRegistry(root string) (*Registry, error) {
	registryPath := filepath.Join(root, ".x-harness", "managed-blocks.yaml")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("managed-blocks registry not found: %w", err)
	}

	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("invalid managed-blocks registry: %w", err)
	}
	return &reg, nil
}

// ManagedContextBlock renders the canonical context using registry markers.
func ManagedContextBlock(entry RegistryEntry) string {
	body := CanonicalContext()
	return strings.Join([]string{
		entry.BeginMarker,
		"<!-- generated-by: x-harness -->",
		fmt.Sprintf("%s%s -->", entry.HashPrefix, ContextHash(body)),
		"",
		body,
		"",
		entry.EndMarker,
	}, "\n")
}

// InjectManagedBlock replaces an existing managed block or appends it.
func InjectManagedBlock(content string, entry RegistryEntry, block string) string {
	beginIndex := strings.Index(content, entry.BeginMarker)
	endIndex := strings.Index(content, entry.EndMarker)
	if beginIndex != -1 && endIndex != -1 && endIndex > beginIndex {
		before := content[:beginIndex]
		after := content[endIndex+len(entry.EndMarker):]
		return before + block + after
	}
	separator := "\n\n"
	if strings.HasSuffix(content, "\n") {
		separator = ""
	}
	return content + separator + block + "\n"
}

// ValidateRegistry reads .x-harness/managed-blocks.yaml and validates all registered blocks.
// It returns a slice of error strings (empty if all valid) and an error if the registry itself is unreadable.
func ValidateRegistry(root string) ([]string, error) {
	reg, err := ReadRegistry(root)
	if err != nil {
		return nil, err
	}

	var failures []string
	for _, entry := range reg.Blocks {
		entryPath, err := ResolveRegistryEntryPath(root, entry.Path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", entry.Path, err))
			continue
		}
		b, err := os.ReadFile(entryPath)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: unreadable: %v", entry.Path, err))
			continue
		}
		var valid bool
		var note string
		if entry.Type == "context" {
			valid, note = ValidateManagedBlockExpected(
				string(b),
				entry.BeginMarker,
				entry.EndMarker,
				entry.HashPrefix,
				CanonicalContext(),
			)
		} else {
			valid, note = ValidateManagedContractBlock(string(b), entry.BeginMarker, entry.EndMarker, entry.HashPrefix)
		}
		if !valid {
			failures = append(failures, fmt.Sprintf("%s: %s", entry.Path, note))
		}
	}

	return failures, nil
}
