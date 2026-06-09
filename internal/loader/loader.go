package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format represents the detected format of a document.
type Format string

const (
	FormatJSON    Format = "json"
	FormatYAML    Format = "yaml"
	FormatUnknown Format = "unknown"
)

// DetectFormat identifies the document format from its file extension.
func DetectFormat(path string) Format {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return FormatJSON
	case ".yaml", ".yml":
		return FormatYAML
	default:
		return FormatUnknown
	}
}

// LoadJSON reads a JSON file and unmarshals it into v.
func LoadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// LoadYAML reads a YAML file and unmarshals it into v.
func LoadYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, v)
}

// LoadDocument detects the file format and loads it into v.
// JSON and YAML files are unmarshaled directly. For files without a recognized
// extension, it attempts JSON first (stricter) and falls back to YAML.
func LoadDocument(path string, v any) error {
	format := DetectFormat(path)
	switch format {
	case FormatJSON:
		return LoadJSON(path, v)
	case FormatYAML:
		return LoadYAML(path, v)
	default:
		jsonErr := LoadJSON(path, v)
		if jsonErr == nil {
			return nil
		}
		yamlErr := LoadYAML(path, v)
		if yamlErr == nil {
			return nil
		}
		return fmt.Errorf("unsupported file format for %q: %s (json: %v, yaml: %v)", path, format, jsonErr, yamlErr)
	}
}
