package boundary

import (
	"bytes"
	"errors"
	"os/exec"
	"sort"
	"strings"
)

// ListChangedFiles returns the list of files changed in the working
// tree, suitable for `xh boundary check --changed`. The function
// shells out to `git diff --name-only HEAD` (working tree vs HEAD) and
// falls back to `git diff --name-only` (staged + unstaged vs index) so
// the user always gets a useful list even in a partial commit state.
//
// When the directory is not a git repository, or git is unavailable,
// the function returns a non-nil error and a nil slice. The caller is
// expected to surface this as a warning and continue with an empty
// file set.
func ListChangedFiles(root string) ([]string, error) {
	if root == "" {
		return nil, errors.New("git root is empty")
	}
	candidates := [][]string{
		{"diff", "--name-only", "HEAD"},
		{"diff", "--name-only"},
		{"status", "--porcelain"},
	}
	var files []string
	var lastErr error
	for _, args := range candidates {
		out, err := runGit(root, args)
		if err != nil {
			lastErr = err
			continue
		}
		files = parseGitDiff(out, args[0] == "status")
		break
	}
	if len(files) == 0 {
		return files, lastErr
	}
	sort.Strings(files)
	return files, nil
}

// parseGitDiff extracts file paths from the output of `git diff
// --name-only` or `git status --porcelain`. The status format prefixes
// each line with a two-character status; we strip it.
func parseGitDiff(output string, isStatus bool) []string {
	var files []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if isStatus {
			// Format: "XY filename" where XY is two characters
			// (e.g. " M", "M ", "A ", "??", etc.). Use the
			// rightmost segment to be safe with renamed entries
			// ("R  old -> new"); for V1 we take the last token.
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			files = append(files, fields[len(fields)-1])
			continue
		}
		files = append(files, strings.TrimSpace(line))
	}
	return files
}

// runGit runs git with the given args inside root and returns stdout.
// A non-zero exit produces an error wrapping stderr.
func runGit(root string, args []string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return stdout.String(), nil
}
