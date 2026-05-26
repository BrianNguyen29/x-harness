package mutationguard

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Snapshot captures the Git working tree state at a point in time.
type Snapshot struct {
	StatusMap map[string]string
	HashMap   map[string]string
	RepoRoot  string
}

// Delta represents a change detected between two snapshots.
type Delta struct {
	Path         string `json:"path"`
	BeforeStatus string `json:"before_status"`
	AfterStatus  string `json:"after_status"`
	BeforeHash   string `json:"before_hash,omitempty"`
	AfterHash    string `json:"after_hash,omitempty"`
}

// Result is the outcome of a mutation guard evaluation.
type Result struct {
	Enabled          bool    `json:"enabled"`
	SkippedReason    string  `json:"skipped_reason,omitempty"`
	Deltas           []Delta `json:"deltas,omitempty"`
	UnexpectedDeltas []Delta `json:"unexpected_deltas,omitempty"`
	Violated         bool    `json:"violated"`
}

// IsGitAvailable returns true if the git command is available.
func IsGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// FindGitRoot returns the Git repository root for the given directory.
func FindGitRoot(cwd string) (string, error) {
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// TakeSnapshot captures the current Git working tree state.
func TakeSnapshot(repoRoot string) (*Snapshot, error) {
	out, err := exec.Command("git", "-C", repoRoot, "status", "--porcelain=v1", "-z", "--untracked-files=all").Output()
	if err != nil {
		return nil, err
	}

	statusMap := make(map[string]string)
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry == "" {
			continue
		}
		if len(entry) < 4 {
			continue
		}
		status := entry[:2]
		filePath := entry[3:]
		statusMap[filePath] = status
		// Skip rename/copy source path
		if strings.Contains(status, "R") || strings.Contains(status, "C") {
			i++
		}
	}

	hashMap := make(map[string]string)
	for path := range statusMap {
		hash, err := contentHash(repoRoot, path)
		if err == nil {
			hashMap[path] = hash
		}
	}

	return &Snapshot{
		StatusMap: statusMap,
		HashMap:   hashMap,
		RepoRoot:  repoRoot,
	}, nil
}

func contentHash(repoRoot, filePath string) (string, error) {
	resolved := filepath.Join(repoRoot, filePath)
	// Safety: ensure path is within repoRoot
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) && abs != rootAbs {
		return "", fmt.Errorf("path escapes repo root")
	}

	info, err := os.Lstat(abs)
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(abs)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("symlink:"+target))), nil
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data)), nil
}

// Compare returns the deltas between two snapshots.
func Compare(before, after *Snapshot) []Delta {
	allPaths := make(map[string]struct{})
	for p := range before.StatusMap {
		allPaths[p] = struct{}{}
	}
	for p := range after.StatusMap {
		allPaths[p] = struct{}{}
	}

	var deltas []Delta
	for p := range allPaths {
		beforeStatus := before.StatusMap[p]
		afterStatus := after.StatusMap[p]
		beforeHash := before.HashMap[p]
		afterHash := after.HashMap[p]
		if beforeStatus != afterStatus || beforeHash != afterHash {
			deltas = append(deltas, Delta{
				Path:         p,
				BeforeStatus: beforeStatus,
				AfterStatus:  afterStatus,
				BeforeHash:   beforeHash,
				AfterHash:    afterHash,
			})
		}
	}
	return deltas
}

// IsAllowlisted returns true if the path is allowed to change.
func IsAllowlisted(filePath string) bool {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	return normalized == ".x-harness" ||
		strings.HasPrefix(normalized, ".x-harness/") ||
		strings.Contains(normalized, "/.x-harness/") ||
		strings.HasSuffix(normalized, ".x-harness") ||
		strings.HasSuffix(normalized, ".x-harness/")
}

// FilterUnexpected removes allowlisted deltas.
func FilterUnexpected(deltas []Delta) []Delta {
	var result []Delta
	for _, d := range deltas {
		if !IsAllowlisted(d.Path) {
			result = append(result, d)
		}
	}
	return result
}

// Guard runs the given function between two snapshots and returns the mutation guard result.
func Guard(repoRoot string, fn func() error) (*Result, error) {
	before, err := TakeSnapshot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("before snapshot failed: %w", err)
	}

	if err := fn(); err != nil {
		return nil, err
	}

	after, err := TakeSnapshot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("after snapshot failed: %w", err)
	}

	deltas := Compare(before, after)
	unexpected := FilterUnexpected(deltas)

	return &Result{
		Enabled:          true,
		Deltas:           deltas,
		UnexpectedDeltas: unexpected,
		Violated:         len(unexpected) > 0,
	}, nil
}
