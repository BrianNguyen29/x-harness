package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoadDocument(f *testing.F) {
	f.Add([]byte(`{"key":"value"}`))
	f.Add([]byte("name: test\n"))
	f.Add([]byte(`{`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmp := t.TempDir()

		// Test with .json extension
		pathJSON := filepath.Join(tmp, "doc.json")
		if err := os.WriteFile(pathJSON, data, 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		var v any
		_ = LoadDocument(pathJSON, &v)

		// Test with .yaml extension
		pathYAML := filepath.Join(tmp, "doc.yaml")
		if err := os.WriteFile(pathYAML, data, 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		_ = LoadDocument(pathYAML, &v)

		// Test extensionless (unknown format path)
		pathExtless := filepath.Join(tmp, "doc")
		if err := os.WriteFile(pathExtless, data, 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		_ = LoadDocument(pathExtless, &v)
	})
}
