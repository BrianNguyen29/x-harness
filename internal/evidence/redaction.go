package evidence

import (
	"regexp"
	"strings"
)

type redactionPattern struct {
	id      string
	regex   *regexp.Regexp
	replace func(match string, groups []string) string
}

var redactionPatterns = []redactionPattern{
	{
		id: "private_key",
		regex: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`),
		replace: func(string, []string) string { return "[REDACTED:private_key]" },
	},
	{
		id: "github_token",
		regex: regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{20,}\b`),
		replace: func(string, []string) string { return "[REDACTED:github_token]" },
	},
	{
		id: "npm_token",
		regex: regexp.MustCompile(`\bnpm_[A-Za-z0-9]{20,}\b`),
		replace: func(string, []string) string { return "[REDACTED:npm_token]" },
	},
	{
		id: "bearer_token",
		regex: regexp.MustCompile(`\bBearer\s+([A-Za-z0-9._~+/=-]{10,})\b`),
		replace: func(match string, groups []string) string {
			return "Bearer [REDACTED:bearer_token]"
		},
	},
	{
		id: "jwt",
		regex: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
		replace: func(string, []string) string { return "[REDACTED:jwt]" },
	},
	{
		id: "connection_string",
		regex: regexp.MustCompile(`(?i)\b((?:postgres(?:ql)?|mysql|mongodb|redis):\/\/)[^\s"'<>]+`),
		replace: func(match string, groups []string) string {
			if len(groups) > 0 {
				return groups[0] + "[REDACTED:connection_string]"
			}
			return "[REDACTED:connection_string]"
		},
	},
	{
		id: "api_key",
		regex: regexp.MustCompile(`(?i)\b(api[_-]?key|apikey|access[_-]?key|secret[_-]?key)\s*[:=]\s*["']?([A-Za-z0-9._~+/=-]{12,})["']?`),
		replace: func(match string, groups []string) string {
			if len(groups) > 0 {
				return groups[0] + "=[REDACTED:api_key]"
			}
			return "[REDACTED:api_key]"
		},
	},
	{
		id: "password_assignment",
		regex: regexp.MustCompile("(?i)\\b(password|passwd|pwd)\\s*[:=]\\s*[\"']?([^\\s\"'`]{6,})[\"']?"),
		replace: func(match string, groups []string) string {
			if len(groups) > 0 {
				return groups[0] + "=[REDACTED:password_assignment]"
			}
			return "[REDACTED:password_assignment]"
		},
	},
}

// RedactText applies redaction patterns to text content.
// It returns the redacted content, matched pattern IDs, and total replacements.
func RedactText(content string) (redacted string, patterns []string, replacements int) {
	text := content
	var matchedPatterns []string
	totalReplacements := 0

	for _, pattern := range redactionPatterns {
		count := 0
		text = pattern.regex.ReplaceAllStringFunc(text, func(match string) string {
			groups := pattern.regex.FindStringSubmatch(match)
			var subgroups []string
			if len(groups) > 1 {
				subgroups = groups[1:]
			}
			count++
			return pattern.replace(match, subgroups)
		})
		if count > 0 {
			matchedPatterns = append(matchedPatterns, pattern.id)
			totalReplacements += count
		}
	}

	return text, matchedPatterns, totalReplacements
}

// IsTextFile returns true if the file extension indicates a text file,
// or if the content contains no null bytes.
func IsTextFile(path string, content []byte) bool {
	ext := strings.ToLower(path)
	for _, suffix := range textExtensions {
		if strings.HasSuffix(ext, suffix) {
			return true
		}
	}
	return !bytesContainsNull(content)
}

func bytesContainsNull(b []byte) bool {
	for _, v := range b {
		if v == 0 {
			return true
		}
	}
	return false
}

var textExtensions = []string{
	".txt", ".md", ".json", ".jsonl", ".yaml", ".yml",
	".log", ".out", ".err", ".stdout", ".stderr", ".env",
	".ts", ".js", ".tsx", ".jsx", ".go", ".sh", ".mod", ".sum",
}
