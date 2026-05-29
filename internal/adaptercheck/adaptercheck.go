package adaptercheck

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ManagedBlock struct {
	ID        string
	BodyLines []string
}

type Check struct {
	Name   string
	Status string
	Note   string
}

type Result struct {
	Path   string
	OK     bool
	Checks []Check
}

func RunDoctor(root string) ([]Result, bool) {
	adaptersDir := filepath.Join(root, "adapters")
	var results []Result
	overallOK := true

	_ = filepath.Walk(adaptersDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		content, err := os.ReadFile(path)
		if err != nil {
			results = append(results, Result{
				Path: rel,
				OK:   false,
				Checks: []Check{
					{Name: "readable", Status: "failed", Note: err.Error()},
				},
			})
			overallOK = false
			return nil
		}

		blocks := FindManagedBlocks(string(content))
		if len(blocks) == 0 {
			return nil
		}

		result := Result{
			Path:   rel,
			OK:     true,
			Checks: []Check{},
		}

		for _, block := range blocks {
			check := ValidateManagedBlock(block)
			result.Checks = append(result.Checks, check)
			if check.Status != "passed" {
				result.OK = false
				overallOK = false
			}
		}

		results = append(results, result)
		return nil
	})

	return results, overallOK
}

func FindManagedBlocks(content string) []ManagedBlock {
	var blocks []ManagedBlock
	var current *ManagedBlock
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- BEGIN X-HARNESS MANAGED CONTRACT:") {
			id := ExtractMarkerID(trimmed, "<!-- BEGIN X-HARNESS MANAGED CONTRACT:", "-->")
			current = &ManagedBlock{
				ID:        id,
				BodyLines: []string{},
			}
		} else if strings.HasPrefix(trimmed, "<!-- END X-HARNESS MANAGED CONTRACT:") && current != nil {
			id := ExtractMarkerID(trimmed, "<!-- END X-HARNESS MANAGED CONTRACT:", "-->")
			if id == current.ID {
				blocks = append(blocks, *current)
				current = nil
			}
		} else if current != nil {
			current.BodyLines = append(current.BodyLines, line)
		}
	}

	return blocks
}

func ExtractMarkerID(trimmed, prefix, suffix string) string {
	s := strings.TrimPrefix(trimmed, prefix)
	s = strings.TrimSuffix(s, suffix)
	return strings.TrimSpace(s)
}

func ValidateManagedBlock(block ManagedBlock) Check {
	hash := ""
	for _, line := range block.BodyLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- contract-hash:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				hash = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "-->"))
			}
		}
	}

	if hash == "" {
		return Check{
			Name:   "managed_block_" + block.ID,
			Status: "failed",
			Note:   "missing contract-hash",
		}
	}

	body := ExtractBodyForHash(block)
	expectedHash := ComputeContractHash(body)

	if hash != expectedHash {
		return Check{
			Name:   "managed_block_" + block.ID,
			Status: "failed",
			Note:   fmt.Sprintf("hash mismatch: expected %s, found %s", expectedHash, hash),
		}
	}

	return Check{
		Name:   "managed_block_" + block.ID,
		Status: "passed",
	}
}

func ExtractBodyForHash(block ManagedBlock) string {
	var filtered []string
	for _, line := range block.BodyLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func ComputeContractHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)[:16]
}
