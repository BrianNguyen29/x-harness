package cli

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const manifestPath = ".x-harness/manifest.yaml"

// ManifestEntry represents a single managed file in the manifest.
type ManifestEntry struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
}

// Manifest is the x-harness installation manifest.
type Manifest struct {
	Version     string          `yaml:"version"`
	GeneratedAt string          `yaml:"generated_at"`
	Profile     string          `yaml:"profile"`
	Entries     []ManifestEntry `yaml:"entries"`
}

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data)), nil
}

func computeManifestEntries(plan []initPlanItem, targetDir string) []ManifestEntry {
	var entries []ManifestEntry
	for _, p := range plan {
		info, err := os.Stat(p.dest)
		if err != nil {
			continue
		}
		if info.IsDir() {
			_ = filepath.WalkDir(p.dest, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(targetDir, path)
				if err != nil {
					return nil
				}
				hash, err := fileHash(path)
				if err != nil {
					return nil
				}
				entries = append(entries, ManifestEntry{Path: filepath.ToSlash(rel), Hash: hash})
				return nil
			})
		} else {
			rel, err := filepath.Rel(targetDir, p.dest)
			if err != nil {
				continue
			}
			hash, err := fileHash(p.dest)
			if err != nil {
				continue
			}
			entries = append(entries, ManifestEntry{Path: filepath.ToSlash(rel), Hash: hash})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}

func writeManifest(targetDir, profile string, entries []ManifestEntry) error {
	m := Manifest{
		Version:     "1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Profile:     profile,
		Entries:     entries,
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	manifestFile := filepath.Join(targetDir, manifestPath)
	if err := os.MkdirAll(filepath.Dir(manifestFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(manifestFile, data, 0644)
}

func readManifest(targetDir string) (*Manifest, error) {
	manifestFile := filepath.Join(targetDir, manifestPath)
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("no manifest found at %s; run init first", manifestFile)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	return &m, nil
}

func validateManifestPath(entryPath string) error {
	if filepath.IsAbs(entryPath) {
		return fmt.Errorf("absolute path not allowed: %s", entryPath)
	}
	if strings.Contains(entryPath, "..") {
		return fmt.Errorf("path contains ..: %s", entryPath)
	}
	return nil
}

func resolveManifestEntryPath(targetDir string, entry ManifestEntry) (string, error) {
	if err := validateManifestPath(entry.Path); err != nil {
		return "", err
	}
	return filepath.Join(targetDir, filepath.FromSlash(entry.Path)), nil
}
