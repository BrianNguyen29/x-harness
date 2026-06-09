package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/conformance"
	"github.com/BrianNguyen29/x-harness/internal/doctor"
	"github.com/BrianNguyen29/x-harness/internal/repo"
)

type readinessLevelResult struct {
	ReadinessLevel   string `json:"readiness_level"`
	OK               bool   `json:"ok"`
	AdmissionOutcome string `json:"admission_outcome,omitempty"`
	AcceptanceStatus string `json:"acceptance_status,omitempty"`
	Note             string `json:"note,omitempty"`
}

func handleReadiness(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness readiness <task|pr|release> [options]")
		return ExitUsage
	}

	switch args[0] {
	case "task":
		return handleReadinessTask(args[1:], stdout, stderr)
	case "pr":
		return handleReadinessPR(args[1:], stdout, stderr)
	case "release":
		return handleReadinessRelease(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stderr, "usage: x-harness readiness <task|pr|release> [options]")
		return ExitUsage
	default:
		fmt.Fprintf(stderr, "unknown readiness subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: x-harness readiness <task|pr|release> [options]")
		return ExitUsage
	}
}

func handleReadinessTask(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 < len(args) {
				cardPath = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		}
	}

	if cardPath == "" || strings.HasPrefix(cardPath, "-") {
		fmt.Fprintln(stderr, "usage: x-harness readiness task --card <path> [--json]")
		return ExitUsage
	}

	result := readinessLevelResult{
		ReadinessLevel: "task",
	}

	// Compose verify logic: reuse handleVerify but capture output
	if jsonMode {
		// We need the verify result; run verify in JSON mode and parse
		// Simpler: run handleVerify directly with a custom stdout to capture JSON
		// But handleVerify writes to stdout directly. Let's use a buffer.
		var buf bytes.Buffer
		code := handleVerify([]string{"--card", cardPath, "--json"}, &buf, stderr)
		result.OK = code == ExitOK
		var vr VerifyResult
		if err := json.Unmarshal(buf.Bytes(), &vr); err == nil {
			result.AdmissionOutcome = vr.AdmissionOutcome
			result.AcceptanceStatus = vr.AcceptanceStatus
		}
	} else {
		code := handleVerify([]string{"--card", cardPath}, stdout, stderr)
		result.OK = code == ExitOK
	}

	if !result.OK && result.AcceptanceStatus == "" {
		result.AcceptanceStatus = "withheld"
	}
	if !result.OK && result.AdmissionOutcome == "" {
		result.AdmissionOutcome = "failed"
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func handleReadinessPR(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := ""
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 < len(args) {
				cardPath = args[i+1]
				i++
			}
		case "--json":
			jsonMode = true
		}
	}

	if cardPath == "" {
		fmt.Fprintln(stderr, "usage: x-harness readiness pr --card <path> [--json]")
		return ExitUsage
	}

	result := readinessLevelResult{
		ReadinessLevel: "pr",
	}

	// Strict verify
	var verifyBuf bytes.Buffer
	code := handleVerify([]string{"--card", cardPath, "--strict", "--json"}, &verifyBuf, stderr)
	verifyOK := code == ExitOK
	var vr VerifyResult
	if err := json.Unmarshal(verifyBuf.Bytes(), &vr); err == nil {
		result.AdmissionOutcome = vr.AdmissionOutcome
		result.AcceptanceStatus = vr.AcceptanceStatus
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	docReport := doctor.Run(root)
	doctorOK := docReport.Healthy

	result.OK = verifyOK && doctorOK
	if !result.OK {
		if result.AcceptanceStatus == "" {
			result.AcceptanceStatus = "withheld"
		}
		if result.AdmissionOutcome == "" {
			result.AdmissionOutcome = "failed"
		}
		notes := []string{}
		if !verifyOK {
			notes = append(notes, "strict verify failed")
		}
		if !doctorOK {
			notes = append(notes, "doctor unhealthy")
		}
		result.Note = "; " + strings.Join(notes, "; ")
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		WriteLine(stdout, "readiness_level: %s", result.ReadinessLevel)
		WriteLine(stdout, "ok: %v", result.OK)
		WriteLine(stdout, "admission_outcome: %s", result.AdmissionOutcome)
		WriteLine(stdout, "acceptance_status: %s", result.AcceptanceStatus)
		if result.Note != "" {
			WriteLine(stdout, "note:%s", result.Note)
		}
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}

func handleReadinessRelease(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonMode := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		}
	}

	root, err := repo.FindRoot("")
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot find repository root: %v\n", err)
		return ExitError
	}

	confReport := conformance.RunMinimal(root)

	result := readinessLevelResult{
		ReadinessLevel:   "release",
		OK:               confReport.OK,
		AdmissionOutcome: "success",
		AcceptanceStatus: "accepted",
		Note:             "local evidence generation/verification available; publish/tag readiness not claimed",
	}

	if !result.OK {
		result.AdmissionOutcome = "failed"
		result.AcceptanceStatus = "withheld"
	}

	if jsonMode {
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		WriteLine(stdout, "readiness_level: %s", result.ReadinessLevel)
		WriteLine(stdout, "ok: %v", result.OK)
		WriteLine(stdout, "admission_outcome: %s", result.AdmissionOutcome)
		WriteLine(stdout, "acceptance_status: %s", result.AcceptanceStatus)
		WriteLine(stdout, "note: %s", result.Note)
	}

	if result.OK {
		return ExitOK
	}
	return ExitError
}
