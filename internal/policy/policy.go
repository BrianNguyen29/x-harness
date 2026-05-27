package policy

import (
	"github.com/BrianNguyen29/x-harness/internal/assets"
	"github.com/BrianNguyen29/x-harness/internal/loader"
)

// Resolve returns the canonical path to a policy file by name.
func Resolve(root, name string) string {
	return assets.NewLocator(root).Policy(name)
}

// LoadYAML resolves and loads a YAML policy file into v.
func LoadYAML(root, name string, v any) error {
	return loader.LoadYAML(Resolve(root, name), v)
}

// LoadDocument resolves and loads a policy file (JSON or YAML) into v.
func LoadDocument(root, name string, v any) error {
	return loader.LoadDocument(Resolve(root, name), v)
}
