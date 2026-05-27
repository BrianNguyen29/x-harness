package components

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"github.com/BrianNguyen29/x-harness/internal/schema"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// ComponentsRegistry represents the registry structure
type ComponentsRegistry struct {
	Version    int              `yaml:"version" json:"version"`
	Components []ComponentEntry `yaml:"components" json:"components"`
}

// ComponentEntry represents a single component
type ComponentEntry struct {
	ID          string   `yaml:"id" json:"id"`
	Kind        string   `yaml:"kind" json:"kind"`
	Paths       []string `yaml:"paths" json:"paths"`
	Owner       string   `yaml:"owner" json:"owner"`
	Stability   string   `yaml:"stability" json:"stability"`
	AgentEdit   string   `yaml:"agent_edit" json:"agent_edit"`
	Tests       []string `yaml:"tests" json:"tests"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
}

// ValidationResult holds the result of registry validation
type ValidationResult struct {
	OK                    bool     `json:"ok"`
	Errors                []string `json:"errors"`
	Warnings              []string `json:"warnings"`
	ComponentCount        int      `json:"component_count"`
	ProtectedPathsChecked int      `json:"protected_paths_checked"`
	ProtectedPathsCovered int      `json:"protected_paths_covered"`
}

// ComponentMatch maps a component to matched files
type ComponentMatch struct {
	Component ComponentEntry `json:"component"`
	Files     []string       `json:"files"`
}

// AuthorityPolicy represents the authority.yaml structure
type AuthorityPolicy struct {
	Version          int `yaml:"version"`
	AuthorityClasses map[string]struct {
		Description string   `yaml:"description"`
		Examples    []string `yaml:"examples"`
	} `yaml:"authority_classes"`
	ProtectedPaths []struct {
		Path      string `yaml:"path"`
		Authority string `yaml:"authority"`
		Rationale string `yaml:"rationale"`
	} `yaml:"protected_paths"`
	ReportOnly bool `yaml:"report_only"`
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

func componentPathMatches(pattern, filePath string) bool {
	return globToRegex(pattern).MatchString(normalizePath(filePath))
}

func componentPathCoversPattern(componentPattern, protectedPattern string) bool {
	component := normalizePath(componentPattern)
	protectedPath := normalizePath(protectedPattern)
	if component == protectedPath {
		return true
	}
	if strings.HasSuffix(component, "/**") {
		prefix := component[:len(component)-3]
		return protectedPath == prefix || strings.HasPrefix(protectedPath, prefix+"/")
	}
	if !strings.Contains(protectedPath, "*") {
		return componentPathMatches(component, protectedPath)
	}
	return false
}

// LoadRegistry loads and parses components/registry.yaml
func LoadRegistry(root string) (*ComponentsRegistry, error) {
	path := filepath.Join(root, "components", "registry.yaml")
	var reg ComponentsRegistry
	if err := loader.LoadYAML(path, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// ValidateRegistry validates registry against schema and checks coverage
func ValidateRegistry(root string) (*ValidationResult, error) {
	errors := make([]string, 0)
	warnings := make([]string, 0)

	reg, err := LoadRegistry(root)
	if err != nil {
		return &ValidationResult{
			OK:                    false,
			Errors:                []string{fmt.Sprintf("components registry load error: %v", err)},
			Warnings:              warnings,
			ComponentCount:        0,
			ProtectedPathsChecked: 0,
			ProtectedPathsCovered: 0,
		}, nil
	}

	// Schema validation
	schemaPath := filepath.Join(root, "schemas", "components-registry.schema.json")
	if _, statErr := os.Stat(schemaPath); statErr == nil {
		v, err := schema.Compile(schemaPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("components registry schema error: %v", err))
		} else {
			// Load registry as map for schema validation
			var doc map[string]interface{}
			if yamlErr := loader.LoadYAML(filepath.Join(root, "components", "registry.yaml"), &doc); yamlErr == nil {
				if valErr := v.Validate(doc); valErr != nil {
					if verr, ok := valErr.(*jsonschema.ValidationError); ok {
						var msgs []string
						flattenSchemaError(verr, &msgs)
						if len(msgs) == 0 {
							msgs = append(msgs, "validation failed")
						}
						errors = append(errors, fmt.Sprintf("components registry schema validation failed: %s", strings.Join(msgs, "; ")))
					} else {
						errors = append(errors, fmt.Sprintf("components registry schema validation failed: %v", valErr))
					}
				}
			}
		}
	}

	// Check duplicate IDs
	seen := make(map[string]bool)
	for _, c := range reg.Components {
		if seen[c.ID] {
			errors = append(errors, fmt.Sprintf("duplicate component id: %s", c.ID))
		}
		seen[c.ID] = true
	}

	// Protected path coverage
	protectedPathsChecked := 0
	protectedPathsCovered := 0
	policyPath := filepath.Join(root, "policies", "authority.yaml")
	if _, statErr := os.Stat(policyPath); statErr == nil {
		var policy AuthorityPolicy
		if loadErr := loader.LoadYAML(policyPath, &policy); loadErr != nil {
			errors = append(errors, fmt.Sprintf("authority policy coverage check failed: %v", loadErr))
		} else {
			for _, pp := range policy.ProtectedPaths {
				protectedPathsChecked++
				covered := false
				for _, c := range reg.Components {
					for _, cp := range c.Paths {
						if componentPathCoversPattern(cp, pp.Path) {
							covered = true
							break
						}
					}
					if covered {
						break
					}
				}
				if covered {
					protectedPathsCovered++
				} else {
					errors = append(errors, fmt.Sprintf("protected path is not registered to any component: %s", pp.Path))
				}
			}
		}
	}

	return &ValidationResult{
		OK:                    len(errors) == 0,
		Errors:                errors,
		Warnings:              warnings,
		ComponentCount:        len(reg.Components),
		ProtectedPathsChecked: protectedPathsChecked,
		ProtectedPathsCovered: protectedPathsCovered,
	}, nil
}

func flattenSchemaError(err error, out *[]string) {
	if err == nil {
		return
	}
	// jsonschema/v6 validation errors implement Error() but may have nested details
	// Use string representation for simplicity
	msg := err.Error()
	if msg != "" {
		*out = append(*out, msg)
	}
}

// FindComponent finds a component by ID
func FindComponent(reg *ComponentsRegistry, id string) *ComponentEntry {
	for i := range reg.Components {
		if reg.Components[i].ID == id {
			return &reg.Components[i]
		}
	}
	return nil
}

func findComponentsForFile(registry *ComponentsRegistry, filePath string) []ComponentEntry {
	normalized := normalizePath(filePath)
	var results []ComponentEntry
	for _, component := range registry.Components {
		for _, componentPath := range component.Paths {
			if componentPathMatches(componentPath, normalized) {
				results = append(results, component)
				break
			}
		}
	}
	return results
}

// ClassifyFiles maps file paths to components
func ClassifyFiles(reg *ComponentsRegistry, files []string) ([]ComponentMatch, []string) {
	componentFiles := make(map[string]*ComponentMatch)
	var unregistered []string

	for _, file := range files {
		normalized := normalizePath(file)
		components := findComponentsForFile(reg, normalized)
		if len(components) == 0 {
			unregistered = append(unregistered, normalized)
			continue
		}
		for _, component := range components {
			match, ok := componentFiles[component.ID]
			if !ok {
				match = &ComponentMatch{Component: component}
				componentFiles[component.ID] = match
			}
			found := false
			for _, f := range match.Files {
				if f == normalized {
					found = true
					break
				}
			}
			if !found {
				match.Files = append(match.Files, normalized)
			}
		}
	}

	var results []ComponentMatch
	for _, match := range componentFiles {
		sort.Strings(match.Files)
		results = append(results, *match)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Component.ID < results[j].Component.ID
	})
	sort.Strings(unregistered)
	return results, unregistered
}

// ListChangedFilesFromGit returns changed file paths from git diff
func ListChangedFilesFromGit(root, base string) ([]string, error) {
	out, err := execGit([]string{"diff", "--name-only", base + "...HEAD"}, root)
	if err != nil {
		out, err = execGit([]string{"diff", "--name-only", base, "HEAD"}, root)
		if err != nil {
			return nil, err
		}
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func execGit(args []string, cwd string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return stdout.String(), nil
}
