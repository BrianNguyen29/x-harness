package classify

import (
	"regexp"
	"strings"
)

// CommandClassification is the structured result of ClassifyCommand.
type CommandClassification struct {
	Command string   `json:"command"`
	Intents []string `json:"intents"`
	Risk    string   `json:"risk"`
	Unknown bool     `json:"unknown"`
}

var riskOrder = map[string]int{
	"low":        1,
	"med" + "ium": 2,
	"high":       3,
}

type intentRule struct {
	pattern *regexp.Regexp
	intents []string
	risk    string
}

// Ordered rules: earlier rules are evaluated first.
// A command may match multiple rules; intents accumulate and the highest risk wins.
var intentRules = []intentRule{
	// Destructive / dangerous (high risk)
	{pattern: regexp.MustCompile(`^rm\s+(-[rf]+|\s+)`), intents: []string{"delete_files", "shell_exec"}, risk: "high"},
	{pattern: regexp.MustCompile(`^curl\s+`), intents: []string{"network_outbound", "shell_exec"}, risk: "high"},
	{pattern: regexp.MustCompile(`^wget\s+`), intents: []string{"network_outbound", "shell_exec"}, risk: "high"},

	// Package publish (high risk)
	{pattern: regexp.MustCompile(`^(npm|pnpm|yarn)\s+publish`), intents: []string{"package_publish", "network_outbound", "shell_exec"}, risk: "high"},
	{pattern: regexp.MustCompile(`^cargo\s+publish`), intents: []string{"package_publish", "network_outbound", "shell_exec"}, risk: "high"},

	// Cloud / secret access (high risk)
	{pattern: regexp.MustCompile(`^aws\s+`), intents: []string{"secret_access", "permission_change", "shell_exec"}, risk: "high"},
	{pattern: regexp.MustCompile(`^gcloud\s+`), intents: []string{"secret_access", "permission_change", "shell_exec"}, risk: "high"},
	{pattern: regexp.MustCompile(`^az\s+`), intents: []string{"secret_access", "permission_change", "shell_exec"}, risk: "high"},

	// Deploy / publish (high risk)
	{pattern: regexp.MustCompile(`^kubectl\s+apply`), intents: []string{"deploy_or_publish", "network_outbound", "shell_exec"}, risk: "high"},
	{pattern: regexp.MustCompile(`^terraform\s+apply`), intents: []string{"deploy_or_publish", "network_outbound", "shell_exec"}, risk: "high"},
	{pattern: regexp.MustCompile(`^serverless\s+deploy`), intents: []string{"deploy_or_publish", "network_outbound", "shell_exec"}, risk: "high"},

	// Database mutation (high risk)
	{pattern: regexp.MustCompile(`^(psql|mysql|sqlite3)\s+`), intents: []string{"database_mutation", "shell_exec"}, risk: "high"},

	// Git mutation (high risk)
	{pattern: regexp.MustCompile(`^git\s+(push|commit|merge|rebase|reset|checkout\s+-b|branch\s+-D)`), intents: []string{"git_mutation", "shell_exec"}, risk: "high"},

	// Permission change (high risk)
	{pattern: regexp.MustCompile(`^(chmod|chown|sudo|su\s+)`), intents: []string{"permission_change", "shell_exec"}, risk: "high"},

	// Package install (risk: medium)
	{pattern: regexp.MustCompile(`^(npm|pnpm|yarn)\s+install`), intents: []string{"package_install", "network_outbound", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^go\s+(get|mod\s+tidy)`), intents: []string{"package_install", "network_outbound", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^cargo\s+(add|install)`), intents: []string{"package_install", "network_outbound", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^pip\s+(install|uninstall)`), intents: []string{"package_install", "network_outbound", "shell_exec"}, risk: "medium"},

	// Build (risk: medium)
	{pattern: regexp.MustCompile(`^(go|npm|pnpm|yarn)\s+build`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^(npm|pnpm|yarn)\s+run\s+build`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^cargo\s+build`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^tsc\s+`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^make\s+`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},

	// File write (risk: medium)
	{pattern: regexp.MustCompile(`^sed\s+-i`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^echo\s+.*>`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},
	{pattern: regexp.MustCompile(`^tee\s+`), intents: []string{"write_files", "shell_exec"}, risk: "medium"},

	// Tests (low risk)
	{pattern: regexp.MustCompile(`^go\s+test`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^(npm|pnpm|yarn)\s+test`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^(npm|pnpm|yarn)\s+run\s+test`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^pytest`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^cargo\s+test`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^vitest`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^jest`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^npm\s+run\s+typecheck`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
	{pattern: regexp.MustCompile(`^tsc\s+--noEmit`), intents: []string{"read_files", "shell_exec"}, risk: "low"},

	// Git read (low risk)
	{pattern: regexp.MustCompile(`^git\s+(status|diff|log|show|ls-files|blame)`), intents: []string{"read_files", "shell_exec"}, risk: "low"},

	// General read (low risk)
	{pattern: regexp.MustCompile(`^(cat|ls|find|head|tail|grep|awk|sed\s+[^-i])`), intents: []string{"read_files", "shell_exec"}, risk: "low"},
}

// ClassifyCommand deterministically classifies a shell command string into
// intents, risk level, and whether the command is unknown.
func ClassifyCommand(command string) CommandClassification {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return CommandClassification{
			Command: cmd,
			Intents: []string{"unknown"},
			Risk:    "high",
			Unknown: true,
		}
	}

	intents := make(map[string]struct{})
	var highestRisk string
	matched := false

	for _, rule := range intentRules {
		if rule.pattern.MatchString(cmd) {
			matched = true
			for _, intent := range rule.intents {
				intents[intent] = struct{}{}
			}
			if riskOrder[rule.risk] > riskOrder[highestRisk] {
				highestRisk = rule.risk
			}
		}
	}

	if !matched {
		return CommandClassification{
			Command: cmd,
			Intents: []string{"unknown"},
			Risk:    "high",
			Unknown: true,
		}
	}

	// Ensure shell_exec is always present when other intents exist.
	if len(intents) > 0 {
		intents["shell_exec"] = struct{}{}
	}

	resultIntents := make([]string, 0, len(intents))
	for intent := range intents {
		resultIntents = append(resultIntents, intent)
	}

	if highestRisk == "" {
		highestRisk = "low"
	}

	return CommandClassification{
		Command: cmd,
		Intents: resultIntents,
		Risk:    highestRisk,
		Unknown: false,
	}
}
