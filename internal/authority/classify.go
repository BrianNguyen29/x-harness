package authority

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/loader"
)

// AuthorityPolicy represents the authority.yaml structure.
type AuthorityPolicy struct {
	Version          int `yaml:"version"`
	AuthorityClasses map[string]struct {
		Description string   `yaml:"description"`
		Examples    []string `yaml:"examples"`
	} `yaml:"authority_classes"`
	ProtectedPaths []PathRule `yaml:"protected_paths"`
	ReportOnly     bool       `yaml:"report_only"`
}

// PathRule represents a single protected path entry.
type PathRule struct {
	Path      string `yaml:"path"`
	Authority string `yaml:"authority"`
	Rationale string `yaml:"rationale"`
}

func normalizePath(filePath string) string {
	s := strings.ReplaceAll(filePath, "\\", "/")
	return strings.TrimPrefix(s, "./")
}

func globToRegex(pattern string) *regexp.Regexp {
	normalized := normalizePath(pattern)
	re := regexp.MustCompile(`[.+^${}()|\[\]\\]`)
	normalized = re.ReplaceAllStringFunc(normalized, func(s string) string {
		return "\\" + s
	})
	normalized = strings.ReplaceAll(normalized, "**", "{{DOUBLE_STAR}}")
	normalized = strings.ReplaceAll(normalized, "*", "[^/]*")
	normalized = strings.ReplaceAll(normalized, "{{DOUBLE_STAR}}", ".*")
	return regexp.MustCompile("^" + normalized + "$")
}

func matchGlob(pattern, filePath string) bool {
	return globToRegex(pattern).MatchString(normalizePath(filePath))
}

func matchPath(pattern, filePath string) bool {
	normalizedPattern := normalizePath(pattern)
	normalizedFilePath := normalizePath(filePath)

	// Direct glob match
	if matchGlob(normalizedPattern, normalizedFilePath) {
		return true
	}

	// Directory prefix match (e.g., "schemas/**" matches "schemas/foo.json")
	if strings.HasSuffix(pattern, "/**") {
		dirPattern := pattern[:len(pattern)-3]
		if strings.HasPrefix(normalizedFilePath, dirPattern) {
			return true
		}
	}

	return false
}

// LoadAuthorityPolicy loads the authority policy from the given repository root.
func LoadAuthorityPolicy(root string) (*AuthorityPolicy, error) {
	path := filepath.Join(root, "policies", "authority.yaml")
	var policy AuthorityPolicy
	if err := loader.LoadYAML(path, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// ClassifyPath returns the authority classification for a file path.
// Possible returns: "agent_editable", "agent_proposable_human_approved", "human_only"
func ClassifyPath(policy *AuthorityPolicy, filePath string) string {
	normalizedPath := normalizePath(filePath)

	for _, protectedPath := range policy.ProtectedPaths {
		if matchPath(protectedPath.Path, normalizedPath) {
			return protectedPath.Authority
		}
	}

	return "agent_editable"
}
