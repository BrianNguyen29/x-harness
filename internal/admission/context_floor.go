package admission

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// contextFloorResult holds the errors and notes from context floor evaluation.
type contextFloorResult struct {
	errors []string
	notes  []string
}

// evaluateContextFloor validates context_alignment presence and structure
// when the --context-floor flag is enabled. It enforces policy for standard/deep tiers.
func evaluateContextFloor(doc map[string]any, tier string) contextFloorResult {
	result := contextFloorResult{
		errors: make([]string, 0),
		notes:  make([]string, 0),
	}

	// Context floor is only enforced for standard and deep tiers
	if tier != "standard" && tier != "deep" {
		result.notes = append(result.notes, "context floor advisory only for light tier")
		return result
	}

	// Check 1: context_alignment must be present
	ctxAlign := mapValue(doc, "context_alignment")
	if ctxAlign == nil {
		result.errors = append(result.errors, "context_alignment is required for standard/deep tier when --context-floor is enabled")
		return result
	}

	// Check 2: stale_ground_checked must be true
	staleGroundChecked := boolInMap(ctxAlign, "stale_ground_checked")
	if !staleGroundChecked {
		result.errors = append(result.errors, "context_alignment.stale_ground_checked must be true for standard/deep tier")
		return result
	}

	// Check 3: At least one non-empty ref array among product_contract_refs, architecture_refs, decision_refs, test_matrix_refs
	refArrays := []string{"product_contract_refs", "architecture_refs", "decision_refs", "test_matrix_refs"}
	hasOneRef := false
	for _, refKey := range refArrays {
		refs := sliceInMap(ctxAlign, refKey)
		if len(refs) > 0 {
			hasOneRef = true
			break
		}
	}
	if !hasOneRef {
		result.errors = append(result.errors, "context_alignment must have at least one non-empty ref array (product_contract_refs, architecture_refs, decision_refs, or test_matrix_refs)")
		return result
	}

	// Deep tier additional checks
	if tier == "deep" {
		// Check 4: context_pack_id must be present and non-empty for deep
		contextPackID := stringInMap(ctxAlign, "context_pack_id")
		if strings.TrimSpace(contextPackID) == "" {
			result.errors = append(result.errors, "context_alignment.context_pack_id is required for deep tier")
			return result
		}

		// Check 5: unresolved_context_questions must be empty for deep
		unresolvedQuestions := sliceInMap(ctxAlign, "unresolved_context_questions")
		if len(unresolvedQuestions) > 0 {
			result.errors = append(result.errors, "context_alignment.unresolved_context_questions must be empty for deep tier")
			return result
		}
	}

	// Check 6: All referenced files must exist
	cardDir := ""
	if cardPath, ok := doc["_cardPath"].(string); ok && cardPath != "" {
		cardDir = filepath.Dir(cardPath)
	}

	fileErrors := validateContextFiles(ctxAlign, cardDir)
	result.errors = append(result.errors, fileErrors...)

	return result
}

// validateContextFiles checks that all file references in context_alignment exist.
// File resolution: relative to cardDir if provided, otherwise relative to current working directory.
func validateContextFiles(ctxAlign map[string]any, cardDir string) []string {
	errors := make([]string, 0)

	// Collect all file refs from ref arrays
	refArrays := []string{"product_contract_refs", "architecture_refs", "decision_refs", "test_matrix_refs"}
	for _, refKey := range refArrays {
		refs := sliceInMap(ctxAlign, refKey)
		for _, ref := range refs {
			if refStr, ok := ref.(string); ok {
				// Strip anchor if present (e.g., "docs/product.md#section")
				refPath := StripAnchor(refStr)
				if !FileExists(refPath, cardDir) {
					errors = append(errors, "referenced file does not exist: "+refPath)
				}
			}
		}
	}

	// Check context_evidence refs
	contextEvidence := sliceInMap(ctxAlign, "context_evidence")
	for _, evidence := range contextEvidence {
		if evMap, ok := evidence.(map[string]any); ok {
			ref := stringInMap(evMap, "ref")
			if ref != "" {
				// Strip anchor if present
				refPath := StripAnchor(ref)
				if !FileExists(refPath, cardDir) {
					errors = append(errors, "context_evidence ref file does not exist: "+refPath)
				}
			}
		}
	}

	return errors
}

// StripAnchor removes #anchor suffix from a path reference.
func StripAnchor(path string) string {
	if idx := strings.Index(path, "#"); idx >= 0 {
		return path[:idx]
	}
	return path
}

// ExtractAnchor returns the anchor portion of a path (after #), or empty string if no anchor.
func ExtractAnchor(path string) string {
	if idx := strings.Index(path, "#"); idx >= 0 && idx+1 < len(path) {
		return path[idx+1:]
	}
	return ""
}

// AnchorExists checks if an anchor can be found in a file.
// It checks for literal "#<anchor>" match or a Markdown heading slug match.
// Slug conversion: lowercase, spaces to hyphens, drop basic punctuation.
func AnchorExists(filePath, anchor string) bool {
	if anchor == "" {
		return true
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	content := string(data)

	// Literal anchor match
	if strings.Contains(content, "#"+anchor) {
		return true
	}

	// Markdown heading slug match
	slug := toSlug(anchor)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			// Extract heading text after #
			heading := strings.TrimLeft(line, "#")
			heading = strings.TrimSpace(heading)
			if toSlug(heading) == slug {
				return true
			}
		}
	}
	return false
}

// toSlug converts a heading text to a GitHub-compatible slug.
func toSlug(text string) string {
	slug := strings.ToLower(text)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Drop basic punctuation
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// FileExists checks if a file exists. For relative paths it tries, in order:
//  1. relative to the current working directory (project-root paths like
//     "examples/..." resolve correctly when the CLI is invoked from the repo root),
//  2. relative to cardDir (covers references authored next to the card itself).
//
// Absolute paths skip both lookups and use the literal path.
func FileExists(path, cardDir string) bool {
	if filepath.IsAbs(path) {
		_, err := os.Stat(path)
		return err == nil
	}
	for _, base := range []string{"", cardDir} {
		candidate := path
		if base != "" {
			candidate = filepath.Join(base, path)
		} else {
			absPath, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			candidate = absPath
		}
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}
