package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/BrianNguyen29/x-harness/internal/conformance"
	"github.com/BrianNguyen29/x-harness/internal/contextcheck"
	"github.com/BrianNguyen29/x-harness/internal/doctor"
	"github.com/BrianNguyen29/x-harness/internal/release"
	"github.com/BrianNguyen29/x-harness/internal/repo"
)

func handleRelease(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness release <evidence|verify-evidence|report|verify-docs> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "evidence":
		return handleReleaseEvidence(args[1:], stdout, stderr)
	case "verify-evidence":
		return handleReleaseVerifyEvidence(args[1:], stdout, stderr)
	case "report":
		return handleReleaseReport(args[1:], stdout, stderr)
	case "verify-docs":
		return handleReleaseVerifyDocs(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown release subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness release <evidence|verify-evidence|report|verify-docs> [options]")
		return ExitUsage
	}
}

// handleReleaseVerifyDocs runs the docs-drift check and reports the
// result. It is a thin wrapper around `xh doctor --docs-drift` so
// release pipelines can call a single command while keeping the
// underlying checks in the doctor package.
func handleReleaseVerifyDocs(args []string, stdout io.Writer, stderr io.Writer) int {
	root := "."
	format := "json"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: x-harness release verify-docs [--root <path>] [--format json|text]")
			return ExitUsage
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	report := doctor.CheckDocsDrift(root)
	switch format {
	case "json":
		if err := WriteJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitError
		}
	case "text":
		doctor.FormatDocsDriftText(report, stdout)
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", format)
		return ExitUsage
	}

	if report.Healthy {
		return ExitOK
	}
	return ExitError
}

func handleReleaseEvidence(args []string, stdout io.Writer, stderr io.Writer) int {
	outPath := ""
	var artifacts []string
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 < len(args) {
				outPath = args[i+1]
				i++
			}
		case "--artifact":
			if i+1 < len(args) {
				artifacts = append(artifacts, args[i+1])
				i++
			}
		case "--json":
			jsonMode = true
		}
	}

	if outPath == "" {
		fmt.Fprintln(stderr, "usage: x-harness release evidence --out <path> [--artifact <path>...] [--json]")
		return ExitUsage
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	if len(artifacts) == 0 {
		if _, err := os.Stat("./x-harness"); err == nil {
			artifacts = []string{"./x-harness"}
		}
	}

	var evidenceArtifacts []release.Artifact
	for _, path := range artifacts {
		hash, size, err := release.ComputeArtifactHash(path)
		if err != nil {
			fmt.Fprintf(stderr, "error: cannot hash artifact %s: %v\n", path, err)
			return ExitError
		}
		evidenceArtifacts = append(evidenceArtifacts, release.Artifact{
			Path:   path,
			SHA256: hash,
			Size:   size,
		})
	}

	confReport := conformance.RunMinimal(root)
	docReport := doctor.Run(root)

	agentsPath := "AGENTS.md"
	agentsContentBytes, err := os.ReadFile(agentsPath)
	contextSyncStatus := "drift"
	if err == nil {
		valid, _ := contextcheck.ValidateManagedBlock(string(agentsContentBytes))
		if valid {
			contextSyncStatus = "no_drift"
		}
	}

	commit := ""
	if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		commit = strings.TrimSpace(string(out))
	}

	ev := release.Evidence{
		SchemaVersion: "x-harness.release-evidence.v1",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Version:       Version,
		Commit:        commit,
		GoVersion:     runtime.Version(),
		Artifacts:     evidenceArtifacts,
		Conformance:   release.ConformanceStatus{Minimal: boolToStatus(confReport.OK)},
		Doctor:        &release.DoctorStatus{Status: boolToHealth(docReport.Healthy)},
		ContextSync:   &release.ContextSyncStatus{Status: contextSyncStatus},
	}

	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot marshal evidence: %v\n", err)
		return ExitError
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		fmt.Fprintf(stderr, "error: cannot write evidence file: %v\n", err)
		return ExitError
	}

	if jsonMode {
		_ = WriteJSON(stdout, map[string]any{
			"ok":        true,
			"out":       outPath,
			"artifacts": len(evidenceArtifacts),
		})
	} else {
		fmt.Fprintf(stdout, "Release evidence written to %s\n", outPath)
		fmt.Fprintf(stdout, "artifacts: %d\n", len(evidenceArtifacts))
		fmt.Fprintf(stdout, "conformance: %s\n", ev.Conformance.Minimal)
		fmt.Fprintf(stdout, "doctor: %s\n", ev.Doctor.Status)
		fmt.Fprintf(stdout, "context_sync: %s\n", ev.ContextSync.Status)
	}

	return ExitOK
}

func handleReleaseVerifyEvidence(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness release verify-evidence <path> [--json]")
		return ExitUsage
	}

	path := args[0]
	jsonMode := false
	if len(args) > 1 && args[1] == "--json" {
		jsonMode = true
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read evidence file: %v\n", err)
		return ExitError
	}

	var ev release.Evidence
	if err := json.Unmarshal(data, &ev); err != nil {
		fmt.Fprintf(stderr, "error: cannot parse evidence file: %v\n", err)
		return ExitError
	}

	verifyErr := release.VerifyEvidence(&ev)

	if jsonMode {
		result := map[string]any{
			"ok": verifyErr == nil,
		}
		if verifyErr != nil {
			result["error"] = verifyErr.Error()
		}
		_ = WriteJSON(stdout, result)
	} else {
		if verifyErr == nil {
			fmt.Fprintln(stdout, "Release evidence verified successfully.")
		} else {
			fmt.Fprintf(stderr, "Release evidence verification failed: %v\n", verifyErr)
		}
	}

	if verifyErr != nil {
		return ExitError
	}
	return ExitOK
}

func handleReleaseReport(args []string, stdout io.Writer, stderr io.Writer) int {
	evidencePath := ""
	format := "markdown"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--evidence":
			if i+1 < len(args) {
				evidencePath = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	if evidencePath == "" {
		fmt.Fprintln(stderr, "usage: x-harness release report --evidence <path> [--format markdown|json]")
		return ExitUsage
	}

	data, err := os.ReadFile(evidencePath)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot read evidence file: %v\n", err)
		return ExitError
	}

	var ev release.Evidence
	if err := json.Unmarshal(data, &ev); err != nil {
		fmt.Fprintf(stderr, "error: cannot parse evidence file: %v\n", err)
		return ExitError
	}

	if verifyErr := release.VerifyEvidence(&ev); verifyErr != nil {
		fmt.Fprintf(stderr, "error: invalid or incomplete evidence: %v\n", verifyErr)
		return ExitError
	}

	switch format {
	case "json":
		report := buildReleaseReport(&ev)
		if err := WriteJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "error: cannot marshal report: %v\n", err)
			return ExitError
		}
	case "markdown":
		renderMarkdownReleaseReport(stdout, &ev)
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", format)
		fmt.Fprintln(stderr, "usage: x-harness release report --evidence <path> [--format markdown|json]")
		return ExitUsage
	}

	return ExitOK
}

type releaseReport struct {
	SchemaVersion  string             `json:"schema_version"`
	Version        string             `json:"version"`
	Commit         string             `json:"commit,omitempty"`
	GoVersion      string             `json:"go_version,omitempty"`
	GeneratedAt    string             `json:"generated_at"`
	Artifacts      []release.Artifact `json:"artifacts"`
	Conformance    map[string]string  `json:"conformance"`
	Doctor         string             `json:"doctor,omitempty"`
	ContextSync    string             `json:"context_sync,omitempty"`
	PlatformMatrix string             `json:"platform_matrix,omitempty"`
	SBOM           string             `json:"sbom,omitempty"`
	Provenance     string             `json:"provenance,omitempty"`
}

func buildReleaseReport(ev *release.Evidence) releaseReport {
	r := releaseReport{
		SchemaVersion: ev.SchemaVersion,
		Version:       ev.Version,
		Commit:        ev.Commit,
		GoVersion:     ev.GoVersion,
		GeneratedAt:   ev.GeneratedAt,
		Artifacts:     ev.Artifacts,
		Conformance:   map[string]string{"minimal": ev.Conformance.Minimal},
	}
	if ev.Doctor != nil {
		r.Doctor = ev.Doctor.Status
	}
	if ev.ContextSync != nil {
		r.ContextSync = ev.ContextSync.Status
	}
	r.PlatformMatrix = "not declared in minimal evidence"
	r.SBOM = "not declared in minimal evidence"
	r.Provenance = "not declared in minimal evidence"
	return r
}

func renderMarkdownReleaseReport(w io.Writer, ev *release.Evidence) {
	fmt.Fprintf(w, "# Release Report\n\n")
	fmt.Fprintf(w, "- **Schema Version**: %s\n", ev.SchemaVersion)
	fmt.Fprintf(w, "- **Version**: %s\n", ev.Version)
	if ev.Commit != "" {
		fmt.Fprintf(w, "- **Commit**: %s\n", ev.Commit)
	}
	if ev.GoVersion != "" {
		fmt.Fprintf(w, "- **Go Version**: %s\n", ev.GoVersion)
	}
	fmt.Fprintf(w, "- **Generated At**: %s\n", ev.GeneratedAt)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Artifacts")
	for _, art := range ev.Artifacts {
		fmt.Fprintf(w, "- `%s` (%d bytes)\n", art.Path, art.Size)
		fmt.Fprintf(w, "  - SHA-256: `%s`\n", art.SHA256)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Conformance")
	fmt.Fprintf(w, "- **Minimal**: %s\n", ev.Conformance.Minimal)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Health & Context")
	if ev.Doctor != nil {
		fmt.Fprintf(w, "- **Doctor**: %s\n", ev.Doctor.Status)
	} else {
		fmt.Fprintln(w, "- **Doctor**: not recorded")
	}
	if ev.ContextSync != nil {
		fmt.Fprintf(w, "- **Context Sync**: %s\n", ev.ContextSync.Status)
	} else {
		fmt.Fprintln(w, "- **Context Sync**: not recorded")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Platform / SBOM / Provenance")
	fmt.Fprintln(w, "- **Platform matrix**: not declared in minimal evidence")
	fmt.Fprintln(w, "- **SBOM**: not declared in minimal evidence")
	fmt.Fprintln(w, "- **Provenance**: not declared in minimal evidence")
}

func boolToStatus(ok bool) string {
	if ok {
		return "passed"
	}
	return "failed"
}

func boolToHealth(ok bool) string {
	if ok {
		return "healthy"
	}
	return "unhealthy"
}
