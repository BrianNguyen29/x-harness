package boundary

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// globMatcher matches a single forward-slash glob pattern against a
// candidate string. Patterns support `*` (matches any sequence of
// non-separator characters within one segment) and `**` (matches any
// sequence including separators and any number of path segments).
// Patterns are anchored at both ends: a match means the whole candidate
// is consumed.
//
// The matcher is intentionally simple: it splits the pattern on `/` and
// walks the segments, expanding `**` greedily. Within a single segment
// (no `/`), `*` is a wildcard for any chars except `/`. This is enough
// to express the V1 use cases (`from`, `to_import`, `intermediate`,
// `allow`) and avoids the surprise behaviour of filepath.Match (e.g.
// character classes) and the strict literal-only interpretation of
// `path.Match`.
type globMatcher struct {
	segments []segment
}

type segmentKind int

const (
	segLiteral    segmentKind = iota // fully literal, no wildcards
	segSingleStar                    // segment is exactly "*" (any name)
	segDoubleStar                    // segment is exactly "**" (any depth)
	segGlob                          // segment has * wildcards inside literal text
)

type segment struct {
	kind   segmentKind
	text   string         // literal / source for segGlob
	regexp *regexp.Regexp // compiled for segGlob
}

// NewGlobMatcherPublic is the public, exported constructor used by
// callers (e.g. the CLI) that need to evaluate a pattern outside the
// Check pipeline. Most users do not need this; it exists so the
// explain command can answer "does rule R apply to file F?" without
// duplicating the glob semantics.
func NewGlobMatcherPublic(pattern string) (GlobMatcher, error) {
	m, err := newGlobMatcher(pattern)
	if err != nil {
		return GlobMatcher{}, err
	}
	return GlobMatcher{segments: m.segments}, nil
}

// GlobMatcher is the public handle around a compiled pattern. The
// underlying segments are unexported; callers may only invoke Match.
type GlobMatcher struct {
	segments []segment
}

// Match reports whether candidate matches the compiled pattern.
func (g GlobMatcher) Match(candidate string) bool {
	if g.segments == nil {
		return false
	}
	candidate = strings.TrimPrefix(strings.ReplaceAll(candidate, "\\", "/"), "./")
	if candidate == "" {
		return matchEmpty(g.segments)
	}
	parts := strings.Split(candidate, "/")
	return matchSegments(g.segments, parts)
}

// newGlobMatcher compiles pattern into a matcher. The pattern uses
// forward slashes regardless of platform; callers are expected to
// normalise the candidate to forward slashes as well.
func newGlobMatcher(pattern string) (*globMatcher, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty glob pattern")
	}
	pattern = strings.TrimPrefix(strings.ReplaceAll(pattern, "\\", "/"), "./")
	if pattern == "" {
		return nil, fmt.Errorf("glob pattern resolves to empty after normalisation")
	}

	raw := strings.Split(pattern, "/")
	segs := make([]segment, 0, len(raw))
	for _, s := range raw {
		switch s {
		case "":
			// collapse "//" and trailing "/" silently
			continue
		case "**":
			segs = append(segs, segment{kind: segDoubleStar})
		case "*":
			segs = append(segs, segment{kind: segSingleStar})
		default:
			if strings.Contains(s, "*") {
				re, err := compileSegmentGlob(s)
				if err != nil {
					return nil, fmt.Errorf("invalid glob in segment %q: %w", s, err)
				}
				segs = append(segs, segment{kind: segGlob, text: s, regexp: re})
				continue
			}
			segs = append(segs, segment{kind: segLiteral, text: s})
		}
	}
	if len(segs) == 0 {
		return nil, fmt.Errorf("glob pattern %q has no segments", pattern)
	}
	return &globMatcher{segments: segs}, nil
}

// compileSegmentGlob converts a single-segment pattern like "*.go" or
// "foo*.ts" into a regular expression that matches the segment. Only
// `*` is special: it matches any sequence of characters except `/`.
func compileSegmentGlob(s string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for _, r := range s {
		switch r {
		case '*':
			b.WriteString("[^/]*")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// Match reports whether candidate matches the pattern. The candidate is
// normalised to forward slashes before matching.
func (g *globMatcher) Match(candidate string) bool {
	candidate = strings.TrimPrefix(strings.ReplaceAll(candidate, "\\", "/"), "./")
	if candidate == "" {
		return matchEmpty(g.segments)
	}
	parts := strings.Split(candidate, "/")
	return matchSegments(g.segments, parts)
}

// matchEmpty handles the edge case where the candidate is empty.
func matchEmpty(segs []segment) bool {
	for _, s := range segs {
		if s.kind != segDoubleStar {
			return false
		}
	}
	return true
}

// matchSegments walks segs and parts. `**` consumes zero or more
// segments greedily (with backtracking when needed). `*` and segGlob
// consume exactly one segment. Literal segments require an exact
// string match.
func matchSegments(segs []segment, parts []string) bool {
	var i, j int
	for i < len(segs) && j < len(parts) {
		s := segs[i]
		switch s.kind {
		case segLiteral:
			if parts[j] != s.text {
				return false
			}
			i++
			j++
		case segSingleStar:
			// `*` matches exactly one segment of any name.
			i++
			j++
		case segGlob:
			if !s.regexp.MatchString(parts[j]) {
				return false
			}
			i++
			j++
		case segDoubleStar:
			// Skip consecutive `**` to avoid duplicate work.
			for i+1 < len(segs) && segs[i+1].kind == segDoubleStar {
				i++
			}
			// If `**` is the last segment, the rest of the candidate
			// (or nothing) is allowed.
			if i == len(segs)-1 {
				return true
			}
			// Try every possible position for the next concrete segment.
			for k := j; k <= len(parts); k++ {
				if matchSegments(segs[i+1:], parts[k:]) {
					return true
				}
			}
			return false
		}
	}
	// Consume trailing `**` segments.
	for i < len(segs) && segs[i].kind == segDoubleStar {
		i++
	}
	return i == len(segs) && j == len(parts)
}

// path is imported for completeness; the current implementation only
// uses strings + regex but keeping the import means future helpers can
// reuse path.Clean without an extra import.
var _ = path.Clean
