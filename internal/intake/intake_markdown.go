package intake

import (
	"regexp"
	"strings"
)

// MarkdownSpec is the result of parsing a markdown file for
// `xh intake contract --from <path>`. Field names mirror the
// product-intent schema. Unknown sections are skipped. Section
// validation is delegated to the canonical record builder.
type MarkdownSpec struct {
	Title              string
	ID                 string
	ProductGoal        string
	UserVisibleChange  *bool
	NonGoals           []string
	Acceptance         []string
	ProtectedBehaviors []string
	AmbiguityQuestions []string
	Notes              string
	AmbiguitySet       bool
}

// ParseMarkdown parses heading-based product intent content. Headings
// are matched case-insensitively; only `##` (level 2) sections are
// recognized for known section names. `###` headings are treated as
// content within the current section. The top-level `#` title provides
// default id and product_goal values unless overridden by sections.
func ParseMarkdown(md string) (MarkdownSpec, error) {
	var spec MarkdownSpec
	currentSection := ""
	var sectionLines []string

	flush := func() error {
		if currentSection == "" {
			return nil
		}
		return applySection(&spec, currentSection, sectionLines)
	}

	for _, raw := range strings.Split(md, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if spec.Title == "" {
				spec.Title = title
			}
			continue
		}
		if strings.HasPrefix(line, "## ") {
			if err := flush(); err != nil {
				return MarkdownSpec{}, err
			}
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			sectionLines = nil
			continue
		}
		sectionLines = append(sectionLines, line)
	}
	if err := flush(); err != nil {
		return MarkdownSpec{}, err
	}

	// Title-derived defaults: id always defaults to slugify(title);
	// product_goal defaults to title only when no Goal section set it.
	if spec.ID == "" && spec.Title != "" {
		spec.ID = slugify(spec.Title)
	}
	if spec.ProductGoal == "" && spec.Title != "" {
		spec.ProductGoal = spec.Title
	}
	return spec, nil
}

// applySection dispatches a known section header to the right field
// on the spec. Unknown sections are skipped. The heading text is
// normalized (lowercased, hyphenated whitespace) before lookup.
func applySection(spec *MarkdownSpec, heading string, lines []string) error {
	canonical, ok := canonicalSection(heading)
	if !ok {
		return nil
	}
	switch canonical {
	case "goal":
		// First non-empty paragraph/text (list markers stripped).
		spec.ProductGoal = firstNonEmptyText(lines)
	case "acceptance":
		spec.Acceptance = extractListItems(lines)
	case "non-goals":
		spec.NonGoals = extractListItems(lines)
	case "protected-behavior":
		spec.ProtectedBehaviors = extractListItems(lines)
	case "ambiguity":
		items := extractListItems(lines)
		if len(items) > 0 {
			spec.AmbiguitySet = true
			spec.AmbiguityQuestions = items
		}
	case "user-visible-change":
		text := firstNonEmptyText(lines)
		if text == "" {
			return nil
		}
		val, err := parseMarkdownBool(text)
		if err != nil {
			return err
		}
		spec.UserVisibleChange = &val
	case "notes":
		// Paragraph text joined. List markers are stripped so
		// bullets render as plain text in the notes field.
		spec.Notes = joinNonEmptyText(lines)
	}
	return nil
}

// canonicalSection normalizes a `##` heading and returns its canonical
// key for switch dispatch. Unknown headings return ("", false) and
// are silently skipped by the caller.
func canonicalSection(heading string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(heading))
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	switch normalized {
	case "goal", "product goal":
		return "goal", true
	case "acceptance", "acceptance criteria", "criteria":
		return "acceptance", true
	case "non goals", "out of scope":
		return "non-goals", true
	case "protected behavior", "protected behaviors":
		return "protected-behavior", true
	case "ambiguity", "open questions", "unresolved questions":
		return "ambiguity", true
	case "user visible change":
		return "user-visible-change", true
	case "notes", "context", "problem":
		return "notes", true
	}
	return "", false
}

// listItemRe matches common markdown list markers so list items can be
// extracted uniformly. Accepts "- item", "* item", "1. item", and
// checkbox variants "- [ ] item" / "- [x] item".
var listItemRe = regexp.MustCompile(`^\s*(?:[-*]|\d+\.)\s+(?:\[[ xX]\]\s+)?(.*)$`)

func extractListItems(lines []string) []string {
	var items []string
	for _, l := range lines {
		m := listItemRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[1])
		if text == "" {
			continue
		}
		items = append(items, text)
	}
	return items
}

func firstNonEmptyText(lines []string) string {
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		if m := listItemRe.FindStringSubmatch(l); m != nil {
			text := strings.TrimSpace(m[1])
			if text != "" {
				return text
			}
			continue
		}
		return trimmed
	}
	return ""
}

func joinNonEmptyText(lines []string) string {
	var parts []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		if m := listItemRe.FindStringSubmatch(l); m != nil {
			text := strings.TrimSpace(m[1])
			if text != "" {
				parts = append(parts, text)
			}
			continue
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, "\n")
}

// slugRe collapses runs of non-alphanumeric characters into a single
// hyphen. Used to derive a stable id from the markdown `#` title.
var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// parseMarkdownBool accepts the same yes/no/true/false/1/0 variants
// the structured `--visible` flag uses. Anything else is a parse
// error so the CLI can surface a usage error to the user.
func parseMarkdownBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	}
	return false, &markdownSectionError{
		Section: "user-visible change",
		Message: "expected true or false, got " + strconvQuote(raw),
	}
}

// markdownSectionError reports a section-level parse error so the
// caller can surface a usage error.
type markdownSectionError struct {
	Section string
	Message string
}

func (e *markdownSectionError) Error() string {
	return "--from " + e.Section + ": " + e.Message
}

// strconvQuote wraps a value in double quotes for error messages.
// Kept tiny to avoid pulling strconv just for one call site.
func strconvQuote(s string) string {
	return `"` + s + `"`
}
