package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CommandRecord is a deterministic, file-first record of one executed
// command. It mirrors the shape of a command_evidence entry inside a
// completion card so the record can be referenced from `evidence.records[]`
// later. All fields are sorted/stabilized for byte-identical JSON output.
type CommandRecord struct {
	SchemaVersion   string           `json:"schema_version"`
	EvidenceID      string           `json:"evidence_id"`
	Command         string           `json:"command"`
	Args            []string         `json:"args"`
	Cwd             string           `json:"cwd,omitempty"`
	ExitCode        int              `json:"exit_code"`
	StartedAt       string           `json:"started_at"`
	EndedAt         string           `json:"ended_at"`
	DurationMillis  int64            `json:"duration_millis"`
	StdoutBytes     int              `json:"stdout_bytes"`
	StderrBytes     int              `json:"stderr_bytes"`
	StdoutSHA256    string           `json:"stdout_sha256"`
	StderrSHA256    string           `json:"stderr_sha256"`
	StdoutCaptured  string           `json:"stdout_captured,omitempty"`
	StderrCaptured  string           `json:"stderr_captured,omitempty"`
	StdoutTruncated bool             `json:"stdout_truncated,omitempty"`
	StderrTruncated bool             `json:"stderr_truncated,omitempty"`
	Git             *CommandGitState `json:"git,omitempty"`
}

// CommandGitState captures the working tree state at the time the command
// ran. Each field is optional so the record stays usable in non-git
// directories. DirtyBaselineHash is a sha256 of the working-tree diff
// (computed by internal/worktree) so the record can detect later drift.
type CommandGitState struct {
	Root              string `json:"root,omitempty"`
	Commit            string `json:"commit,omitempty"`
	Branch            string `json:"branch,omitempty"`
	Dirty             bool   `json:"dirty,omitempty"`
	DirtyBaselineHash string `json:"dirty_baseline_hash,omitempty"`
}

// CommandRunOptions configures a single command execution.
type CommandRunOptions struct {
	// Command and its Args. The joined form is stored in CommandRecord.Command.
	Command string
	Args    []string
	// Dir is the working directory. Defaults to the current dir.
	Dir string
	// MaxCaptureBytes caps the inline stdout/stderr capture. When 0
	// the runner keeps all bytes (capped at MaxCaptureAbsolute to keep
	// the evidence record bounded).
	MaxCaptureBytes int
	// Env is appended to os.Environ for the child process.
	Env []string
}

// MaxCaptureAbsolute is the hard upper bound for inline stdout/stderr
// capture. Anything larger is summarized by size and sha256 only.
const MaxCaptureAbsolute = 64 * 1024

// CommandRunResult is the runner's return value.
type CommandRunResult struct {
	OK        bool          `json:"ok"`
	SchemaVer string        `json:"schema_version"`
	Record    CommandRecord `json:"record"`
	OutPath   string        `json:"out_path,omitempty"`
	OutDir    string        `json:"out_dir,omitempty"`
	Errors    []string      `json:"errors,omitempty"`
}

// CommandRunSchemaVersion is the schema version of CommandRecord.
const CommandRunSchemaVersion = "x-harness.command-record.v1"

// RunCommand executes a single shell command and writes a deterministic
// JSON record to OutPath. When OutPath is empty the record is written to
// <OutDir>/<evidence_id>.json where OutDir defaults to
// .x-harness/evidence/.
func RunCommand(opts CommandRunOptions, outDir, outPath string) (*CommandRunResult, error) {
	if strings.TrimSpace(opts.Command) == "" {
		return nil, fmt.Errorf("command is required")
	}

	if opts.Dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine working directory: %w", err)
		}
		opts.Dir = wd
	}

	maxCapture := opts.MaxCaptureBytes
	if maxCapture <= 0 || maxCapture > MaxCaptureAbsolute {
		maxCapture = MaxCaptureAbsolute
	}

	startedAt := time.Now().UTC()
	cmd := exec.Command(opts.Command, opts.Args...)
	cmd.Dir = opts.Dir
	cmd.Env = append(os.Environ(), opts.Env...)

	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot create stdout pipe: %w", err)
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot start command: %w", err)
	}

	// Drain output concurrently. Each pipe is bounded by maxCapture; the
	// remaining bytes are counted but not retained.
	stdoutBytes, stdoutCaptured, stdoutTruncated := drainWithCap(stdoutReader, maxCapture)
	stderrBytes, stderrCaptured, stderrTruncated := drainWithCap(stderrReader, maxCapture)

	waitErr := cmd.Wait()
	endedAt := time.Now().UTC()
	exitCode := 0
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}

	evidenceID := commandRecordID(opts.Command, opts.Args, startedAt, exitCode, stdoutBytes, stderrBytes)

	rec := CommandRecord{
		SchemaVersion:   CommandRunSchemaVersion,
		EvidenceID:      evidenceID,
		Command:         opts.Command,
		Args:            append([]string{}, opts.Args...),
		Cwd:             opts.Dir,
		ExitCode:        exitCode,
		StartedAt:       startedAt.UTC().Format(time.RFC3339Nano),
		EndedAt:         endedAt.UTC().Format(time.RFC3339Nano),
		DurationMillis:  endedAt.Sub(startedAt).Milliseconds(),
		StdoutBytes:     stdoutBytes,
		StderrBytes:     stderrBytes,
		StdoutSHA256:    sha256Hex(stdoutCaptured),
		StderrSHA256:    sha256Hex(stderrCaptured),
		StdoutCaptured:  stdoutCaptured,
		StderrCaptured:  stderrCaptured,
		StdoutTruncated: stdoutTruncated,
		StderrTruncated: stderrTruncated,
	}

	// Capture git state at the same moment so the record can later
	// detect a dirty baseline drift.
	rec.Git = captureGitState(opts.Dir)

	// Determine output path.
	finalOutPath := outPath
	finalOutDir := outDir
	if finalOutPath == "" {
		if finalOutDir == "" {
			finalOutDir = filepath.Join(opts.Dir, ".x-harness", "evidence")
		}
		if err := os.MkdirAll(finalOutDir, 0o755); err != nil {
			return nil, fmt.Errorf("cannot create evidence dir: %w", err)
		}
		finalOutPath = filepath.Join(finalOutDir, evidenceID+".json")
	} else {
		if dir := filepath.Dir(finalOutPath); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("cannot create evidence parent dir: %w", err)
			}
		}
	}

	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal command record: %w", err)
	}
	// Append trailing newline so the file is friendly to cat/diff/git.
	data = append(data, '\n')
	if err := os.WriteFile(finalOutPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("cannot write command record: %w", err)
	}

	res := &CommandRunResult{
		OK:        exitCode == 0 && waitErr == nil,
		SchemaVer: CommandRunSchemaVersion,
		Record:    rec,
		OutPath:   finalOutPath,
		OutDir:    finalOutDir,
	}
	if waitErr != nil {
		res.Errors = []string{waitErr.Error()}
	}
	return res, nil
}

// drainWithCap reads from r up to maxBytes. Returns the total bytes read,
// the captured prefix (up to maxBytes), and whether more bytes remained.
func drainWithCap(r interface {
	Read(p []byte) (n int, err error)
}, maxBytes int) (int, string, bool) {
	buf := make([]byte, 8192)
	var captured strings.Builder
	total := 0
	truncated := false
	for {
		n, err := r.Read(buf)
		if n > 0 {
			total += n
			remaining := maxBytes - captured.Len()
			if remaining > 0 {
				if n <= remaining {
					captured.Write(buf[:n])
				} else {
					captured.Write(buf[:remaining])
					truncated = true
				}
			} else {
				truncated = true
			}
		}
		if err != nil {
			break
		}
	}
	return total, captured.String(), truncated
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// commandRecordID builds a stable evidence id from the command, args,
// start time, exit code, and output sizes. sha256, truncated to 16 hex
// chars to match the existing evidenceID convention.
func commandRecordID(command string, args []string, startedAt time.Time, exitCode, stdoutBytes, stderrBytes int) string {
	parts := []string{
		command,
		strings.Join(args, "\x00"),
		startedAt.UTC().Format(time.RFC3339Nano),
		fmt.Sprintf("exit=%d", exitCode),
		fmt.Sprintf("stdout=%d", stdoutBytes),
		fmt.Sprintf("stderr=%d", stderrBytes),
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])[:16]
}

// captureGitState reads git state at the given dir. Returns nil when git
// is unavailable or dir is not inside a git repo. The function intentionally
// mirrors worktree.CollectInfo for the subset of fields we need so the
// record stays self-contained.
func captureGitState(dir string) *CommandGitState {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	if out, err := gitOutput(dir, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(out) != "true" {
		return nil
	}
	state := &CommandGitState{}
	if out, err := gitOutput(dir, "rev-parse", "--show-toplevel"); err == nil {
		state.Root = strings.TrimSpace(out)
	}
	if out, err := gitOutput(dir, "rev-parse", "HEAD"); err == nil {
		state.Commit = strings.TrimSpace(out)
	}
	if out, err := gitOutput(dir, "branch", "--show-current"); err == nil {
		state.Branch = strings.TrimSpace(out)
	}
	porcelain, err := gitOutput(dir, "status", "--porcelain")
	if err == nil {
		state.Dirty = strings.TrimSpace(porcelain) != ""
	}
	if state.Dirty {
		// Match worktree.computeDirtyHash so downstream consumers can
		// compare hashes deterministically.
		commit := state.Commit
		if commit == "" {
			commit = "unknown"
		}
		diff, err := gitOutput(dir, "diff", "HEAD")
		if err != nil {
			state.DirtyBaselineHash = sha256Hex(commit + ":dirty:unknown")
		} else {
			state.DirtyBaselineHash = sha256Hex(commit + ":dirty:" + diff)
		}
	} else if state.Commit != "" {
		state.DirtyBaselineHash = sha256Hex(state.Commit + ":clean")
	}
	return state
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// SortForDeterminism is exposed for tests that need a stable ordering of
// multiple records. The current schema is a single record; the helper is
// here for forward-compat with `xh evidence run --batch`.
func SortForDeterminism(records []CommandRecord) []CommandRecord {
	out := append([]CommandRecord{}, records...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].EvidenceID < out[j].EvidenceID
	})
	return out
}
