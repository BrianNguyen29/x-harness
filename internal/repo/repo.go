package repo

import (
	"errors"
	"os"
	"path/filepath"
)

var rootMarkers = []string{
	".git",
	"go.mod",
	"X_HARNESS.md",
	"AGENTS.md",
}

// ErrNotFound indicates the repository root could not be located.
var ErrNotFound = errors.New("repository root not found")

// FindRoot walks upward from startDir looking for a directory that contains
// at least one root marker. If startDir is empty, the current working directory
// is used.
func FindRoot(startDir string) (string, error) {
	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		startDir = wd
	}

	for {
		for _, marker := range rootMarkers {
			path := filepath.Join(startDir, marker)
			if _, err := os.Stat(path); err == nil {
				return startDir, nil
			}
		}

		parent := filepath.Dir(startDir)
		if parent == startDir {
			break
		}
		startDir = parent
	}

	return "", ErrNotFound
}
