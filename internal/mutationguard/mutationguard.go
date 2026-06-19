package mutationguard

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Snapshot captures the Git working tree state at a point in time.
type Snapshot struct {
	StatusMap       map[string]string
	HashMap         map[string]string
	RepoRoot        string
	HeadSHA         string
	DirtyTreeHash   string
	SubmoduleStatus string
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

	headSHA, _ := gitHeadSHA(repoRoot)
	dirtyHash, _ := gitDirtyHash(repoRoot)
	subStatus, _ := gitSubmoduleStatus(repoRoot)

	return &Snapshot{
		StatusMap:       statusMap,
		HashMap:         hashMap,
		RepoRoot:        repoRoot,
		HeadSHA:         headSHA,
		DirtyTreeHash:   dirtyHash,
		SubmoduleStatus: subStatus,
	}, nil
}

func gitHeadSHA(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitDirtyHash(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "diff", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(out)), nil
}

func gitSubmoduleStatus(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "submodule", "status").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
	if before.HeadSHA != after.HeadSHA {
		deltas = append(deltas, Delta{
			Path:         "@@HEAD",
			BeforeStatus: before.HeadSHA,
			AfterStatus:  after.HeadSHA,
		})
	}
	if before.SubmoduleStatus != after.SubmoduleStatus {
		deltas = append(deltas, Delta{
			Path:         "@@SUBMODULE",
			BeforeStatus: before.SubmoduleStatus,
			AfterStatus:  after.SubmoduleStatus,
		})
	}
	if before.DirtyTreeHash != after.DirtyTreeHash {
		deltas = append(deltas, Delta{
			Path:         "@@DIRTYTREE",
			BeforeStatus: before.DirtyTreeHash,
			AfterStatus:  after.DirtyTreeHash,
		})
	}
	return deltas
}

// IsAllowlisted returns true if the path is allowed to change.
func IsAllowlisted(filePath string) bool {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	return normalized == ".x-harness" ||
		strings.HasPrefix(normalized, ".x-harness/")
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

func hashConcurrency() int {
	v := os.Getenv("X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY")
	if v == "" {
		return 16
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 16
	}
	if n > 64 {
		return 64
	}
	return n
}

type fallbackIgnoreConfig struct {
	FallbackIgnore struct {
		Dirs     []string `yaml:"dirs"`
		Paths    []string `yaml:"paths"`
		Patterns []string `yaml:"patterns"`
	} `yaml:"fallback_ignore"`
}

func loadIgnorePatterns(root string) ([]string, error) {
	var ignores []string
	ignores = append(ignores, ".git/", "node_modules/", ".x-harness/")

	gitignore, err := loadGitignore(root)
	if err == nil {
		ignores = append(ignores, gitignore...)
	}

	fallback, err := loadFallbackIgnore(root)
	if err == nil {
		ignores = append(ignores, fallback...)
	}

	return ignores, nil
}

func loadGitignore(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, nil
}

func loadFallbackIgnore(root string) ([]string, error) {
	path := filepath.Join(root, "policies", "mutation-guard.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg fallbackIgnoreConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	var ignores []string
	for _, d := range cfg.FallbackIgnore.Dirs {
		ignores = append(ignores, d+"/")
	}
	ignores = append(ignores, cfg.FallbackIgnore.Paths...)
	ignores = append(ignores, cfg.FallbackIgnore.Patterns...)
	return ignores, nil
}

func shouldIgnore(rel string, ignores []string) bool {
	for _, pat := range ignores {
		if pat == "" {
			continue
		}
		pat = strings.TrimPrefix(pat, "/")
		isDir := strings.HasSuffix(pat, "/")
		pat = strings.TrimSuffix(pat, "/")

		for _, part := range strings.Split(rel, "/") {
			if matched, _ := filepath.Match(pat, part); matched {
				return true
			}
			if isDir && part == pat {
				return true
			}
		}
		if matched, _ := filepath.Match(pat, rel); matched {
			return true
		}
		if pat == rel {
			return true
		}
	}
	return false
}

// TakeFallbackSnapshot captures the current directory tree state without Git.
func TakeFallbackSnapshot(root string) (*Snapshot, error) {
	root = filepath.Clean(root)

	ignores, _ := loadIgnorePatterns(root)

	type entry struct {
		rel string
	}
	var entries []entry
	statusMap := make(map[string]string)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if shouldIgnore(relSlash, ignores) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			statusMap[relSlash] = "F"
			entries = append(entries, entry{rel: relSlash})
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		statusMap[relSlash] = "F"
		entries = append(entries, entry{rel: relSlash})
		return nil
	})
	if err != nil {
		return nil, err
	}

	hashMap := make(map[string]string)
	concurrency := hashConcurrency()
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, e := range entries {
		wg.Add(1)
		go func(e entry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			h, err := contentHash(root, e.rel)
			if err != nil {
				return
			}
			mu.Lock()
			hashMap[e.rel] = h
			mu.Unlock()
		}(e)
	}
	wg.Wait()

	return &Snapshot{
		StatusMap:       statusMap,
		HashMap:         hashMap,
		RepoRoot:        root,
		HeadSHA:         "",
		DirtyTreeHash:   "",
		SubmoduleStatus: "",
	}, nil
}

// GuardFallback runs the given function between two fallback snapshots.
func GuardFallback(root string, fn func() error) (*Result, error) {
	before, err := TakeFallbackSnapshot(root)
	if err != nil {
		return nil, fmt.Errorf("before fallback snapshot failed: %w", err)
	}

	if err := fn(); err != nil {
		return nil, err
	}

	after, err := TakeFallbackSnapshot(root)
	if err != nil {
		return nil, fmt.Errorf("after fallback snapshot failed: %w", err)
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
