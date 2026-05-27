package evidence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/repo"
	"github.com/BrianNguyen29/x-harness/internal/schema"
)

// ReadIndex reads an evidence index file in JSONL or JSON envelope format.
// It detects the format by checking if the first non-empty line starts with "{"
// and whether the parsed object contains an "entries" key.
func ReadIndex(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var firstNonEmpty string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			firstNonEmpty = line
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if firstNonEmpty == "" {
		return []map[string]any{}, nil
	}

	// Reset to beginning for full read
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(firstNonEmpty, "{") {
		var envelope map[string]any
		if err := json.Unmarshal(data, &envelope); err == nil {
			if entriesRaw, ok := envelope["entries"]; ok {
				if entriesArr, ok := entriesRaw.([]any); ok {
					entries := make([]map[string]any, 0, len(entriesArr))
					for _, e := range entriesArr {
						if m, ok := e.(map[string]any); ok {
							entries = append(entries, m)
						}
					}
					return entries, nil
				}
			}
		}
		// Fall through to JSONL parsing if envelope detection fails
	}

	entries := []map[string]any{}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(trimmed), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ValidateEntries validates each entry against the evidence-index schema.
func ValidateEntries(entries []map[string]any) (bool, []string) {
	root, err := repo.FindRoot("")
	if err != nil {
		return false, []string{fmt.Sprintf("cannot find repository root: %v", err)}
	}
	schemaPath := assets.NewLocator(root).Schema("evidence-index.schema.json")
	validator, err := schema.Compile(schemaPath)
	if err != nil {
		return false, []string{fmt.Sprintf("cannot compile schema: %v", err)}
	}

	var errors []string
	for i, entry := range entries {
		if err := validator.Validate(entry); err != nil {
			errors = append(errors, fmt.Sprintf("entry %d: %v", i, err))
		}
	}
	return len(errors) == 0, errors
}

// ValidateIndexFile combines ReadIndex and ValidateEntries.
func ValidateIndexFile(path string) (bool, []string, int, error) {
	entries, err := ReadIndex(path)
	if err != nil {
		return false, nil, 0, err
	}
	ok, errs := ValidateEntries(entries)
	return ok, errs, len(entries), nil
}
