// Real-world example fixture: formatSlug utility for a Go CLI.
//
// This file is a tiny placeholder used only as a referenced path in
// the example completion card. The example is meant to demonstrate
// the x-harness flow, not to ship a production utility.
package utils

import (
	"strings"
	"unicode"
)

// formatSlug converts a free-form string into a URL-friendly slug.
// Lower-case, ASCII alphanumerics and dashes only; non-alphanumeric
// characters collapse into single dashes; leading/trailing dashes are
// trimmed; empty input returns "".
func formatSlug(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		case !prevDash && b.Len() > 0:
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
