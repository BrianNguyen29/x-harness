package assets

import (
	"path/filepath"
)

// Locator resolves canonical asset paths relative to a repository root.
type Locator struct {
	Root string
}

// NewLocator creates a Locator rooted at repoRoot.
func NewLocator(repoRoot string) *Locator {
	return &Locator{Root: repoRoot}
}

// Policy returns the path to a policy file by name.
func (l *Locator) Policy(name string) string {
	return filepath.Join(l.Root, "policies", name)
}

// Schema returns the path to a schema file by name.
func (l *Locator) Schema(name string) string {
	return filepath.Join(l.Root, "schemas", name)
}

// Template returns the path to a template file by name.
func (l *Locator) Template(name string) string {
	return filepath.Join(l.Root, "templates", name)
}

// Example returns the path to an example directory or file by name.
func (l *Locator) Example(name string) string {
	return filepath.Join(l.Root, "examples", name)
}

// Adapter returns the path to an adapter directory or file by name.
func (l *Locator) Adapter(name string) string {
	return filepath.Join(l.Root, "adapters", name)
}
