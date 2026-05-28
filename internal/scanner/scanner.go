package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Severity represents the finding severity level.
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Finding represents a single scanner finding.
type Finding struct {
	Severity  string `json:"severity"`
	Category  string `json:"category"`
	RuleID    string `json:"rule_id"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Snippet   string `json:"snippet"`
	Waivable  bool   `json:"waivable"`
}

// Summary represents the aggregate scan summary.
type Summary struct {
	Low       int    `json:"low"`
	Medium    int    `json:"medium"`
	High      int    `json:"high"`
	Total     int    `json:"total"`
	Risk      string `json:"risk"`
	FilesScanned int `json:"files_scanned"`
}

// Result represents the full scan result.
type Result struct {
	FilesScanned int       `json:"files_scanned"`
	Findings     []Finding `json:"findings"`
	Summary      Summary   `json:"summary"`
}

// Rule defines a single scanner heuristic rule.
type Rule struct {
	ID       string
	Category string
	Severity Severity
	Pattern  *regexp.Regexp
	Waivable bool
	Exclude  []string // matched line must NOT contain any of these substrings
}

// DefaultRules returns the built-in deterministic scanner rules.
func DefaultRules() []Rule {
	return []Rule{
		{
			ID:       "remote-pipe-shell",
			Category: "remote_code_execution",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)(curl|wget)[^|]*\|\s*(bash|sh|zsh|fish)`),
			Waivable: false,
		},
		{
			ID:       "rm-rf-root",
			Category: "destructive_command",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*f[a-zA-Z]*\s+(/|\.+/|~/)`),
			Waivable: false,
		},
		{
			ID:       "rm-rf-dot",
			Category: "destructive_command",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*r[a-zA-Z]*\s+\.`),
			Waivable: false,
		},
		{
			ID:       "chmod-executable-remote",
			Category: "permission_change",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)chmod\s+\+x\s+(/tmp/|/var/tmp/|~/|\.*/)`),
			Waivable: true,
		},
		{
			ID:       "ssh-secret-access",
			Category: "secret_access",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)cat\s+\S*(/\.ssh/|/\.aws/|/\.gcp/|/\.azure/)`),
			Waivable: false,
		},
		{
			ID:       "env-exfiltration",
			Category: "secret_access",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)(env\s*|printenv\s*|cat\s+.*\.env).*\|\s*(curl|wget|nc\s|netcat)`),
			Waivable: false,
		},
		{
			ID:       "api-key-token-exfil",
			Category: "secret_access",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)(API_KEY|TOKEN|SECRET|PASSWORD|PRIVATE_KEY).*\|\s*(curl|wget|nc\s|netcat)`),
			Waivable: false,
		},
		{
			ID:       "mcp-auto-enable",
			Category: "mcp_untrusted",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)mcp.*auto[-_]?enable|auto[-_]?enable.*mcp|enable\s+mcp\s+server`),
			Waivable: true,
		},
		{
			ID:       "hook-outside-namespace",
			Category: "hook_misconfiguration",
			Severity: SeverityMedium,
			Pattern:  regexp.MustCompile(`(?i)hook.*(install|run|exec)`),
			Waivable: true,
			Exclude:  []string{"x-harness", "xharness"},
		},
		{
			ID:       "sudo-dangerous",
			Category: "privilege_escalation",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)sudo\s+(rm|chmod|chown|dd|mkfs|fdisk)`),
			Waivable: false,
		},
		{
			ID:       "chown-dangerous",
			Category: "permission_change",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)chown\s+-R\s+root|chown\s+.*(/etc/|/usr/|/bin/|/sbin/)`),
			Waivable: false,
		},
		{
			ID:       "chmod-dangerous",
			Category: "permission_change",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)chmod\s+777|chmod\s+.*(/etc/|/usr/|/bin/|/sbin/)`),
			Waivable: false,
		},
		{
			ID:       "path-traversal",
			Category: "path_traversal",
			Severity: SeverityMedium,
			Pattern:  regexp.MustCompile(`(?i)(\.\./){2,}|/\.\./\.\./|\.{3,}/`),
			Waivable: true,
		},
		{
			ID:       "network-fetch",
			Category: "network_outbound",
			Severity: SeverityMedium,
			Pattern:  regexp.MustCompile(`(?i)^(curl|wget|fetch|download)\s+[^|]*\b(https?://|ftp://)[^|]*$`),
			Waivable: true,
		},
		{
			ID:       "browser-profile-access",
			Category: "secret_access",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)cat\s+\S*(/\.config/|/Library/Application\s+Support/|/AppData/).*(chrome|firefox|safari|edge|opera)`),
			Waivable: false,
		},
		{
			ID:       "eval-or-exec-string",
			Category: "remote_code_execution",
			Severity: SeverityHigh,
			Pattern:  regexp.MustCompile(`(?i)\beval\s*\(|\bexec\s*\(|\bsystem\s*\(`),
			Waivable: false,
		},
		{
			ID:       "broad-write-outside-allowlist",
			Category: "filesystem_write",
			Severity: SeverityMedium,
			Pattern:  regexp.MustCompile(`(?i)(>|>>|tee)\s+(/etc/|/usr/|/bin/|/sbin/|/sys/|/proc/)`),
			Waivable: true,
		},
	}
}

// ScanResult holds the result of scanning a single file.
type fileScanResult struct {
	path     string
	findings []Finding
}

// Scan scans the given paths (files or directories) and returns a Result.
func Scan(rules []Rule, paths []string) (*Result, error) {
	var allFindings []Finding
	filesScanned := 0

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("cannot stat %s: %w", p, err)
		}

		if info.IsDir() {
			res, err := scanDir(rules, p)
			if err != nil {
				return nil, err
			}
			allFindings = append(allFindings, res.Findings...)
			filesScanned += res.FilesScanned
		} else {
			res, err := scanFile(rules, p)
			if err != nil {
				return nil, err
			}
			allFindings = append(allFindings, res.findings...)
			filesScanned++
		}
	}

	summary := buildSummary(allFindings, filesScanned)
	if allFindings == nil {
		allFindings = []Finding{}
	}
	return &Result{
		FilesScanned: filesScanned,
		Findings:     allFindings,
		Summary:      summary,
	}, nil
}

func scanDir(rules []Rule, dir string) (*Result, error) {
	var allFindings []Finding
	filesScanned := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// Skip binary files by extension
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".ico" || ext == ".woff" || ext == ".woff2" || ext == ".ttf" || ext == ".eot" {
			return nil
		}
		// Skip files larger than 1MB
		if info.Size() > 1024*1024 {
			return nil
		}
		res, err := scanFile(rules, path)
		if err != nil {
			return err
		}
		allFindings = append(allFindings, res.findings...)
		filesScanned++
		return nil
	})
	if err != nil {
		return nil, err
	}

	summary := buildSummary(allFindings, filesScanned)
	if allFindings == nil {
		allFindings = []Finding{}
	}
	return &Result{
		FilesScanned: filesScanned,
		Findings:     allFindings,
		Summary:      summary,
	}, nil
}

func scanFile(rules []Rule, path string) (*fileScanResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var findings []Finding
	scanner := bufio.NewScanner(f)
	const maxCapacity = 1024 * 1024 // 1MB line buffer
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, rule := range rules {
			if rule.Pattern.MatchString(line) {
				excluded := false
				for _, ex := range rule.Exclude {
					if strings.Contains(strings.ToLower(line), strings.ToLower(ex)) {
						excluded = true
						break
					}
				}
				if excluded {
					continue
				}
				snippet := strings.TrimSpace(line)
				if len(snippet) > 120 {
					snippet = snippet[:120] + "..."
				}
				findings = append(findings, Finding{
					Severity: string(rule.Severity),
					Category: rule.Category,
					RuleID:   rule.ID,
					File:     path,
					Line:     lineNum,
					Snippet:  snippet,
					Waivable: rule.Waivable,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &fileScanResult{path: path, findings: findings}, nil
}

func buildSummary(findings []Finding, filesScanned int) Summary {
	low, medium, high := 0, 0, 0
	for _, f := range findings {
		switch f.Severity {
		case "low":
			low++
		case "medium":
			medium++
		case "high":
			high++
		}
	}

	risk := "none"
	if high > 0 {
		risk = "high"
	} else if medium > 0 {
		risk = "medium"
	} else if low > 0 {
		risk = "low"
	}

	return Summary{
		Low:          low,
		Medium:       medium,
		High:         high,
		Total:        len(findings),
		Risk:         risk,
		FilesScanned: filesScanned,
	}
}
