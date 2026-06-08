package contextmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Manifest is a machine-readable snapshot of context file hashes.
type Manifest struct {
	Version string  `yaml:"version" json:"version"`
	Entries []Entry `yaml:"entries" json:"entries"`
}

// Entry is a single context file entry.
type Entry struct {
	Path   string `yaml:"path" json:"path"`
	SHA256 string `yaml:"sha256" json:"sha256"`
	ReadAt string `yaml:"read_at,omitempty" json:"read_at,omitempty"`
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// Generate creates a manifest from a list of file paths using raw-content SHA256.
// Paths are normalized relative to baseDir. If baseDir is empty, the current
// working directory is used.
func Generate(filePaths []string, baseDir string, reason string) (*Manifest, error) {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot get working directory: %w", err)
		}
	}
	baseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve baseDir: %w", err)
	}

	entries := make([]Entry, 0, len(filePaths))
	now := time.Now().UTC().Format(time.RFC3339)

	for _, fp := range filePaths {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		absPath, err := filepath.Abs(fp)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve path %q: %w", fp, err)
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %q: %w", fp, err)
		}

		hash := sha256.Sum256(data)
		relPath, err := filepath.Rel(baseDir, absPath)
		if err != nil {
			relPath = fp
		}
		// Normalize to forward slashes for cross-platform stability
		relPath = filepath.ToSlash(relPath)

		entries = append(entries, Entry{
			Path:   relPath,
			SHA256: hex.EncodeToString(hash[:]),
			ReadAt: now,
			Reason: reason,
		})
	}

	return &Manifest{
		Version: "1",
		Entries: entries,
	}, nil
}

// Check compares the manifest entries against the current filesystem.
// It returns a list of stale paths (modified or deleted). Extra untracked
// files are ignored. Paths are resolved relative to baseDir.
func Check(manifest *Manifest, baseDir string) ([]string, error) {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot get working directory: %w", err)
		}
	}
	baseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve baseDir: %w", err)
	}

	stale := make([]string, 0)
	for _, entry := range manifest.Entries {
		// Normalize path separator for the current OS
		entryPath := filepath.FromSlash(entry.Path)
		resolved := filepath.Join(baseDir, entryPath)

		data, err := os.ReadFile(resolved)
		if err != nil {
			stale = append(stale, entry.Path)
			continue
		}

		hash := sha256.Sum256(data)
		currentHash := hex.EncodeToString(hash[:])
		if currentHash != entry.SHA256 {
			stale = append(stale, entry.Path)
		}
	}
	return stale, nil
}

// Write writes the manifest to path as YAML.
func Write(manifest *Manifest, path string) error {
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("cannot marshal manifest: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cannot create parent directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("cannot write manifest: %w", err)
	}
	return nil
}

// Read reads a manifest from path.
func Read(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read manifest: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("cannot parse manifest: %w", err)
	}
	return &manifest, nil
}

// Validate checks structural validity of the manifest.
func Validate(manifest *Manifest) error {
	if manifest.Version == "" {
		return fmt.Errorf("manifest version is required")
	}
	if manifest.Version != "1" {
		return fmt.Errorf("unsupported manifest version: %s", manifest.Version)
	}
	seen := make(map[string]struct{})
	for i, entry := range manifest.Entries {
		if strings.TrimSpace(entry.Path) == "" {
			return fmt.Errorf("entry[%d]: path is required", i)
		}
		if strings.TrimSpace(entry.SHA256) == "" {
			return fmt.Errorf("entry[%d]: sha256 is required", i)
		}
		if _, ok := seen[entry.Path]; ok {
			return fmt.Errorf("duplicate path in manifest: %s", entry.Path)
		}
		seen[entry.Path] = struct{}{}
	}
	return nil
}
