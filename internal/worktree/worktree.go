package worktree

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info holds worktree metadata collected from git.
type Info struct {
	Root             string `json:"root"`
	GitCommonDir     string `json:"git_common_dir"`
	Branch           string `json:"branch"`
	Commit           string `json:"commit"`
	DirtyBaselineHash string `json:"dirty_baseline_hash"`
}

// CollectInfo gathers worktree metadata from git at the given root.
// Returns nil if git is unavailable or the directory is not inside a git repository.
func CollectInfo(root string) *Info {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}

	// Verify we're inside a git repo
	if out, err := gitOutput(root, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(out) != "true" {
		return nil
	}

	info := &Info{}

	if out, err := gitOutput(root, "rev-parse", "--show-toplevel"); err == nil {
		info.Root = strings.TrimSpace(out)
	}

	if out, err := gitOutput(root, "rev-parse", "--git-common-dir"); err == nil {
		info.GitCommonDir = strings.TrimSpace(out)
		if !filepath.IsAbs(info.GitCommonDir) && info.Root != "" {
			info.GitCommonDir = filepath.Join(info.Root, info.GitCommonDir)
		}
	}

	if out, err := gitOutput(root, "branch", "--show-current"); err == nil {
		info.Branch = strings.TrimSpace(out)
	}

	if out, err := gitOutput(root, "rev-parse", "HEAD"); err == nil {
		info.Commit = strings.TrimSpace(out)
	}

	info.DirtyBaselineHash = computeDirtyHash(root, info.Commit)

	return info
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func computeDirtyHash(root, commit string) string {
	if commit == "" {
		commit = "unknown"
	}

	// Check if working tree is dirty
	out, err := gitOutput(root, "status", "--porcelain")
	if err != nil {
		return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(commit+":unknown")))
	}

	if strings.TrimSpace(out) == "" {
		// Clean working tree
		return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(commit+":clean")))
	}

	// Dirty working tree: hash the diff for a reproducible dirty baseline
	diffOut, err := gitOutput(root, "diff", "HEAD")
	if err != nil {
		return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(commit+":dirty:unknown")))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(commit+":dirty:"+diffOut)))
}
