package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// StartResult is the JSON output shape for the start command.
type StartResult struct {
	OK        bool        `json:"ok"`
	Steps     []StartStep `json:"steps"`
	NextSteps []string    `json:"next_steps"`
}

// StartStep represents a single phase of the start flow.
type StartStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

func handleStart(args []string, stdout io.Writer, stderr io.Writer) int {
	root := "."
	profile := "minimal"
	apply := false
	skipDoctor := false
	skipExamples := false
	jsonMode := false
	wizardWithCard := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Fprintln(stderr, "usage: xh start [--root <path>] [--profile minimal|standard|full|deep] [--apply] [--skip-doctor] [--skip-examples] [--json] [--wizard-with-card <task_id>]")
			return ExitUsage
		case "--root":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		case "--profile":
			if i+1 < len(args) {
				profile = args[i+1]
				i++
			}
		case "--apply":
			apply = true
		case "--skip-doctor":
			skipDoctor = true
		case "--skip-examples":
			skipExamples = true
		case "--json":
			jsonMode = true
		case "--wizard-with-card":
			if i+1 < len(args) {
				wizardWithCard = args[i+1]
				i++
			}
		}
	}

	validProfiles := map[string]bool{"minimal": true, "standard": true, "full": true, "deep": true}
	if !validProfiles[profile] {
		fmt.Fprintf(stderr, "usage: xh start [--root <path>] [--profile minimal|standard|full|deep] [--apply] [--skip-doctor] [--skip-examples] [--json] [--wizard-with-card <task_id>]\n")
		fmt.Fprintf(stderr, "invalid profile: %s\n", profile)
		return ExitUsage
	}

	initProfile := profile
	if profile == "deep" {
		initProfile = "full"
	}

	var steps []StartStep

	// Step 1: Doctor
	if !skipDoctor {
		var doctorStdout, doctorStderr bytes.Buffer
		doctorCode := handleDoctor([]string{"--root", root, "--json"}, &doctorStdout, &doctorStderr)
		status := "passed"
		if doctorCode != ExitOK {
			status = "failed"
		}
		note := "workspace healthy"
		if doctorCode != ExitOK {
			note = "workspace has issues"
		}
		if doctorStderr.Len() > 0 {
			note = strings.TrimSpace(doctorStderr.String())
		}
		steps = append(steps, StartStep{Name: "doctor", Status: status, Note: note})
	}

	// Step 2: Examples verify
	if !skipExamples {
		var exStdout, exStderr bytes.Buffer
		exCode := handleExamplesVerify([]string{"--json"}, &exStdout, &exStderr)
		status := "passed"
		if exCode != ExitOK {
			status = "failed"
		}
		note := "all golden examples passed"
		if exCode != ExitOK {
			note = "some golden examples failed"
			if exStderr.Len() > 0 {
				note = strings.TrimSpace(exStderr.String())
			}
		}
		steps = append(steps, StartStep{Name: "examples_verify", Status: status, Note: note})
	}

	// Step 3: Init wizard
	{
		initArgs := []string{root, "--wizard", "--wizard-profile", initProfile}
		if !apply {
			initArgs = append(initArgs, "--wizard-dry-run")
		}
		if wizardWithCard != "" {
			initArgs = append(initArgs, "--wizard-with-card", wizardWithCard)
		}
		var initStdout, initStderr bytes.Buffer
		initCode := handleInit(initArgs, &initStdout, &initStderr)
		status := "passed"
		if initCode != ExitOK {
			status = "failed"
		}
		note := "init wizard previewed"
		if apply {
			note = "init wizard applied"
		}
		if initCode != ExitOK {
			note = "init wizard failed"
			if initStderr.Len() > 0 {
				note = strings.TrimSpace(initStderr.String())
			}
		}
		steps = append(steps, StartStep{Name: "init_wizard", Status: status, Note: note})
	}

	// Determine overall ok
	ok := true
	for _, s := range steps {
		if s.Status != "passed" && s.Status != "skipped" {
			ok = false
			break
		}
	}

	nextSteps := []string{
		"Run your first verification: xh check --card completion-card.yaml",
		"Read the docs: docs/GETTING_STARTED.md",
	}

	if jsonMode {
		result := StartResult{
			OK:        ok,
			Steps:     steps,
			NextSteps: nextSteps,
		}
		if err := WriteJSON(stdout, result); err != nil {
			return ExitError
		}
	} else {
		WriteLine(stdout, "# xh start - Guided onboarding")
		WriteLine(stdout, "")
		stepNum := 1
		if !skipDoctor {
			WriteLine(stdout, "Step %d/4: doctor", stepNum)
			WriteLine(stdout, "  status: %s", steps[stepNum-1].Status)
			if steps[stepNum-1].Note != "" {
				WriteLine(stdout, "  note: %s", steps[stepNum-1].Note)
			}
			WriteLine(stdout, "")
			stepNum++
		}
		if !skipExamples {
			WriteLine(stdout, "Step %d/4: examples verify", stepNum)
			WriteLine(stdout, "  status: %s", steps[stepNum-1].Status)
			if steps[stepNum-1].Note != "" {
				WriteLine(stdout, "  note: %s", steps[stepNum-1].Note)
			}
			WriteLine(stdout, "")
			stepNum++
		}
		WriteLine(stdout, "Step %d/4: init wizard", stepNum)
		// init wizard is always the last step in steps slice
		last := steps[len(steps)-1]
		WriteLine(stdout, "  status: %s", last.Status)
		if last.Note != "" {
			WriteLine(stdout, "  note: %s", last.Note)
		}
		WriteLine(stdout, "")
		WriteLine(stdout, "Next steps:")
		for _, s := range nextSteps {
			WriteLine(stdout, "  - %s", s)
		}
	}

	if ok {
		return ExitOK
	}
	return ExitError
}
